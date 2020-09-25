// Automatically generated from the message definition "std_msgs/Int16.msg"

package std_msgs

import (
	"bytes"
	"encoding/binary"
	"github.com/seqsense/rosgo/ros"
)

type _MsgInt16 struct {
	text   string
	name   string
	md5sum string
}

func (t *_MsgInt16) Text() string {
	return t.text
}

func (t *_MsgInt16) Name() string {
	return t.name
}

func (t *_MsgInt16) MD5Sum() string {
	return t.md5sum
}

func (t *_MsgInt16) NewMessage() ros.Message {
	m := new(Int16)
	m.Data = 0
	return m
}

var (
	MsgInt16 = &_MsgInt16{
		`int16 data
`,
		"std_msgs/Int16",
		"8524586e34fbd7cb1c08c5f5f1ca0e57",
	}
)

type Int16 struct {
	Data int16 `rosmsg:"data:int16"`
}

func (m *Int16) Type() ros.MessageType {
	return MsgInt16
}

func (m *Int16) Serialize(buf *bytes.Buffer) error {
	var err error = nil
	binary.Write(buf, binary.LittleEndian, m.Data)
	return err
}

func (m *Int16) Deserialize(buf *bytes.Reader) error {
	var err error = nil
	if err = binary.Read(buf, binary.LittleEndian, &m.Data); err != nil {
		return err
	}
	return err
}
