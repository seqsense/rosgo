package main

//go:generate gengo -vendor=false msg std_msgs/String
import (
	"fmt"
	"github.com/akio/rosgo/ros"
	"github.com/seqsense/rosgo/test/test_talker_without_vendoring/rosmsgs/std_msgs"
	"os"
	"time"
)

func main() {
	node, err := ros.NewNode("/talker", os.Args)
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}
	defer node.Shutdown()
	node.Logger().SetSeverity(ros.LogLevelDebug)
	pub := node.NewPublisher("/chatter", std_msgs.MsgString)

	for node.OK() {
		node.SpinOnce()
		var msg std_msgs.String
		msg.Data = fmt.Sprintf("hello %s", time.Now())
		fmt.Println(msg.Data)
		pub.Publish(&msg)
		time.Sleep(time.Second)
	}
}
