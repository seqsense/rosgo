// Automatically generated from the message definition "std_msgs/Int32.msg"

package std_msgs

import (
	"bytes"
	"encoding/binary"
	"github.com/seqsense/rosgo/ros"
)

type _MsgInt32 struct {
	text   string
	name   string
	md5sum string
}

func (t *_MsgInt32) Text() string {
	return t.text
}

func (t *_MsgInt32) Name() string {
	return t.name
}

func (t *_MsgInt32) MD5Sum() string {
	return t.md5sum
}

func (t *_MsgInt32) NewMessage() ros.Message {
	m := new(Int32)
	m.Data = 0
	return m
}

var (
	MsgInt32 = &_MsgInt32{
		`int32 data`,
		"std_msgs/Int32",
		"da5909fbe378aeaf85e547e830cc1bb7",
	}
)

type Int32 struct {
	Data int32 `rosmsg:"data:int32"`
}

func (m *Int32) Type() ros.MessageType {
	return MsgInt32
}

func (m *Int32) Serialize(buf *bytes.Buffer) error {
	var err error = nil
	binary.Write(buf, binary.LittleEndian, m.Data)
	return err
}

func (m *Int32) Deserialize(buf *bytes.Reader) error {
	var err error = nil
	if err = binary.Read(buf, binary.LittleEndian, &m.Data); err != nil {
		return err
	}
	return err
}
