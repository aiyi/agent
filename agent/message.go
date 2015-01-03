package agent

import (
	"io"
	"time"
)

const (
	FrameTypeResponse int32 = 0
	FrameTypeError    int32 = 1
	FrameTypeMessage  int32 = 2
)

type Message interface {
}

type Protocol interface {
	NewProtoInstance() ProtoInstance
}

type ProtoInstance interface {
	DecodeMessage(r io.Reader) (int32, Message, error)
	HandleMessage(msg Message) Message
	WriteMessage(w io.Writer, msg Message) error
	NewHeartbeatMsg() Message
	HeartbeatInterval() time.Duration
}
