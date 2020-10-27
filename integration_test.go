// +build integration

package main

import (
	"os"
	"sync"
	"testing"
	"time"

	"github.com/seqsense/rosgo/internal/std_msgs"
	"github.com/seqsense/rosgo/ros"
)

func TestIntegration_PubSub(t *testing.T) {
	testCases := map[string]struct {
		delayAdvertise, delaySubscribe, delayPublish time.Duration
	}{
		"AdvertiseThenSubscribe": {
			delayAdvertise: 0,
			delaySubscribe: 100 * time.Millisecond,
			delayPublish:   200 * time.Millisecond,
		},
		"SubscribeThenAdvertise": {
			delayAdvertise: 100 * time.Millisecond,
			delaySubscribe: 0,
			delayPublish:   100 * time.Millisecond,
		},
	}

	for name, tt := range testCases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			nodePub, err := ros.NewNode("publisher", os.Args)
			if err != nil {
				t.Fatal(err)
			}
			defer nodePub.Shutdown()
			nodeSub, err := ros.NewNode("subscriber", os.Args)
			if err != nil {
				t.Fatal(err)
			}
			defer nodeSub.Shutdown()

			chMsg := make(chan std_msgs.Int32, 1)

			var wg sync.WaitGroup
			wg.Add(2)
			done := make(chan struct{})
			go func() {
				wg.Wait()
				close(done)
			}()

			const expectedValue = 123

			go func() {
				time.Sleep(tt.delayAdvertise)
				pub, err := nodePub.NewPublisher("/test_topic", std_msgs.MsgInt32)
				if err != nil {
					t.Fatal(err)
				}

				time.Sleep(tt.delayPublish)
				pub.Publish(&std_msgs.Int32{Data: expectedValue})

				nodePub.Spin()
				wg.Done()
			}()
			go func() {
				time.Sleep(tt.delaySubscribe)
				_, err := nodeSub.NewSubscriber("/test_topic", std_msgs.MsgInt32, func(msg *std_msgs.Int32) {
					select {
					case chMsg <- *msg:
					default:
						t.Error("Received too many number of messages")
					}
				})
				if err != nil {
					t.Error(err)
				}

				nodeSub.Spin()
				wg.Done()
			}()

			select {
			case msg := <-chMsg:
				if msg.Data != expectedValue {
					t.Errorf("Expected: %d, received: %d", expectedValue, msg.Data)
				}
			case <-time.After(time.Second):
				t.Error("Message timeout")
			}

			nodePub.Shutdown()
			nodeSub.Shutdown()

			select {
			case <-done:
			case <-time.After(time.Second):
				t.Error("Shutdown timeout")
			}
		})
	}
}
