package ros

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"reflect"
	"sync"
	"time"
)

type messageEvent struct {
	bytes []byte
	event MessageEvent
}

// The subscription object runs in own goroutine (startSubscription).
// Do not access any properties from other goroutine.
type defaultSubscriber struct {
	topic            string
	msgType          MessageType
	pubList          []string
	pubListChan      chan []string
	msgChan          chan messageEvent
	callbacks        []interface{}
	addCallbackChan  chan interface{}
	shutdownChan     chan struct{}
	connections      map[string]chan struct{}
	disconnectedChan chan string
}

func newDefaultSubscriber(topic string, msgType MessageType, callback interface{}) *defaultSubscriber {
	sub := new(defaultSubscriber)
	sub.topic = topic
	sub.msgType = msgType
	sub.msgChan = make(chan messageEvent, 10)
	sub.pubListChan = make(chan []string, 10)
	sub.addCallbackChan = make(chan interface{}, 10)
	sub.shutdownChan = make(chan struct{}, 10)
	sub.disconnectedChan = make(chan string, 10)
	sub.connections = make(map[string]chan struct{})
	sub.callbacks = []interface{}{callback}
	return sub
}

func (sub *defaultSubscriber) start(wg *sync.WaitGroup, nodeId, nodeApiUri, masterUri string, jobChan chan func(), logger Logger) {
	logger.Debugf("Subscriber goroutine for %s started.", sub.topic)
	wg.Add(1)
	defer wg.Done()
	defer func() {
		logger.Debug("defaultSubscriber.start exit")
	}()
	for {
		logger.Debug("Loop")
		select {
		case list := <-sub.pubListChan:
			logger.Debug("Receive pubListChan")
			deadPubs := setDifference(sub.pubList, list)
			newPubs := setDifference(list, sub.pubList)
			sub.pubList = list

			for _, pub := range deadPubs {
				quitChan, ok := sub.connections[pub]
				if !ok {
					continue
				}
				quitChan <- struct{}{}
				close(quitChan)
				delete(sub.connections, pub)
			}
			for _, pub := range newPubs {
				protocols := []interface{}{[]interface{}{"TCPROS"}}
				result, err := callRosApi(pub, "requestTopic", nodeId, sub.topic, protocols)
				if err != nil {
					logger.Errorf("Failed to request topic: %v", err)
					continue
				}
				protocolParams := result.([]interface{})
				for _, x := range protocolParams {
					logger.Debug(x)
				}
				name := protocolParams[0].(string)
				if name == "TCPROS" {
					addr := protocolParams[1].(string)
					port := protocolParams[2].(int32)
					uri := fmt.Sprintf("%s:%d", addr, port)
					quitChan := make(chan struct{}, 10)
					sub.connections[pub] = quitChan
					go startRemotePublisherConn(logger,
						uri, sub.topic,
						sub.msgType.MD5Sum(),
						sub.msgType.Name(), nodeId,
						sub.msgChan,
						quitChan,
						sub.disconnectedChan)
				} else {
					logger.Warnf("Rosgo does not support protocol '%s'", name)
				}
			}
		case callback := <-sub.addCallbackChan:
			logger.Debug("Receive addCallbackChan")
			sub.callbacks = append(sub.callbacks, callback)
		case msgEvent := <-sub.msgChan:
			// Pop received message then bind callbacks and enqueue to the job channle.
			logger.Debug("Receive msgChan")
			callbacks := make([]interface{}, len(sub.callbacks))
			copy(callbacks, sub.callbacks)
			jobChan <- func() {
				m := sub.msgType.NewMessage()
				reader := bytes.NewReader(msgEvent.bytes)
				if err := m.Deserialize(reader); err != nil {
					logger.Error(err)
				}
				args := []reflect.Value{reflect.ValueOf(m), reflect.ValueOf(msgEvent.event)}
				for _, callback := range callbacks {
					fun := reflect.ValueOf(callback)
					num_args_needed := fun.Type().NumIn()
					if num_args_needed <= 2 {
						fun.Call(args[0:num_args_needed])
					}
				}
			}
			logger.Debug("Callback job enqueued.")
		case pubUri := <-sub.disconnectedChan:
			logger.Debugf("Connection to %s was disconnected.", pubUri)
			if quitChan, ok := sub.connections[pubUri]; ok {
				close(quitChan)
				delete(sub.connections, pubUri)
			}
		case <-sub.shutdownChan:
			// Shutdown subscription goroutine
			logger.Debug("Receive shutdownChan")
			for _, quitChan := range sub.connections {
				quitChan <- struct{}{}
				close(quitChan)
			}
			_, err := callRosApi(masterUri, "unregisterSubscriber", nodeId, sub.topic, nodeApiUri)
			if err != nil {
				logger.Warn(err)
			}
			return
		}
	}
}

func startRemotePublisherConn(logger Logger,
	pubUri string, topic string, md5sum string,
	msgType string, nodeId string,
	msgChan chan messageEvent,
	quitChan chan struct{},
	disconnectedChan chan string) {
	logger.Debug("startRemotePublisherConn()")

	defer func() {
		logger.Debug("startRemotePublisherConn() exit")
	}()

	conn, err := net.Dial("tcp", pubUri)
	if err != nil {
		logger.Errorf("Failed to connect %s!", pubUri)
		disconnectedChan <- pubUri
		return
	}
	defer conn.Close()

	// 1. Write connection header
	var headers []header
	headers = append(headers, header{"topic", topic})
	headers = append(headers, header{"md5sum", md5sum})
	headers = append(headers, header{"type", msgType})
	headers = append(headers, header{"callerid", nodeId})
	logger.Debug("TCPROS Connection Header")
	for _, h := range headers {
		logger.Debugf("  `%s` = `%s`", h.key, h.value)
	}
	err = writeConnectionHeader(headers, conn)
	if err != nil {
		logger.Error("Failed to write connection header.")
		disconnectedChan <- pubUri
		return
	}

	// 2. Read reponse header
	var resHeaders []header
	resHeaders, err = readConnectionHeader(conn)
	if err != nil {
		logger.Error("Failed to read reasponse header.")
		disconnectedChan <- pubUri
		return
	}
	logger.Debug("TCPROS Response Header:")
	resHeaderMap := make(map[string]string)
	for _, h := range resHeaders {
		resHeaderMap[h.key] = h.value
		logger.Debugf("  `%s` = `%s`", h.key, h.value)
	}
	if resHeaderMap["type"] != msgType || resHeaderMap["md5sum"] != md5sum {
		logger.Error("Incompatible message type!")
		disconnectedChan <- pubUri
		return
	}
	logger.Debug("Start receiving messages...")
	event := MessageEvent{ // Event struct to be sent with each message.
		PublisherName:    resHeaderMap["callerid"],
		ConnectionHeader: resHeaderMap,
	}

	// 3. Start reading messages
	var quit bool
	go func() {
		<-quitChan
		// set idle timeout
		conn.SetDeadline(time.Now().Add(time.Second))
		quit = true
	}()

	var msgSize uint32
	var buffer []byte

	for {
		if quit {
			return
		}
		//logger.Debug("Reading message size...")
		if err := binary.Read(conn, binary.LittleEndian, &msgSize); err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				return
			}
			if err == io.EOF {
				return
			}
			logger.Errorf("Failed to read a message size. err: %+v", err)
			disconnectedChan <- pubUri
			return
		}
		logger.Debugf("msgSize=%d", msgSize)
		buffer = make([]byte, int(msgSize))
		//logger.Debug("Reading message body...")
		_, err := io.ReadFull(conn, buffer)
		if err != nil {
			if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
				return
			}
			if err == io.EOF {
				return
			}
			logger.Errorf("Failed to read a message body. err: %+v", err)
			disconnectedChan <- pubUri
			return
		}
		event.ReceiptTime = time.Now()
		msgChan <- messageEvent{bytes: buffer, event: event}
	}
}

func (sub *defaultSubscriber) Shutdown() {
	sub.shutdownChan <- struct{}{}
}

func (sub *defaultSubscriber) GetNumPublishers() int {
	return len(sub.pubList)
}
