package rsu

import (
	//"encoding/binary"
	"errors"
	"fmt"
	. "github.com/aiyi/agent/agent"
	"io"
	"time"
)

// Error types
var (
	ConnClosedError     = errors.New("connection was closed")
	WriteBlockedError   = errors.New("write blocking")
	ReadBlockedError    = errors.New("read blocking")
	ReadPacketError     = errors.New("read packet error")
	PacketTooLargeError = errors.New("packet too large")
	MessageUnknownError = errors.New("unknown message type")
)

// Message types
const (
	HeartbeatRequest  uint16 = 0xD468
	HeartbeatResponse uint16 = 0xC468
)

// Packet: pacLen + pacType + pacData
// Big endian: uint32 + uint32 + []byte
type RsuMessage struct {
	msgId   uint8
	msgType uint16
	data    []byte
}

func NewRsuMessage(msgId uint8, msgType uint16) *RsuMessage {
	return &RsuMessage{
		msgId:   msgId,
		msgType: msgType,
	}
}

func GetBCC(buf []byte) uint8 {
	n := len(buf)
	b := uint8(0)
	for i := 0; i < n; i++ {
		b ^= buf[i]
	}
	return b
}

//Bytes operates on a Message pointer and returns a slice of bytes
//representing the Message ready for transmission over the network
func (m *RsuMessage) Bytes() []byte {
	var b []byte
	b = append(b, 0x80|m.msgId)
	b = append(b, uint8((m.msgType&0xFF00)>>8))
	b = append(b, uint8(m.msgType&0x00FF))
	b = append(b, m.data...)
	b = append(b, GetBCC(b[:]))
	return b
}

type RsuProtocol struct {
}

func (this *RsuProtocol) NewProtoInstance() ProtoInstance {
	return &RsuProtoInst{
		seq: 0,
	}
}

type RsuProtoInst struct {
	seq uint8
}

func (p *RsuProtoInst) genMsgId() uint8 {
	id := p.seq
	p.seq++
	if p.seq == 8 {
		p.seq = 9
	} else if p.seq > 9 {
		p.seq = 0
	}
	return id
}

func (p *RsuProtoInst) DecodeMessage(r io.Reader) (int32, Message, error) {
	var (
		m         RsuMessage
		hdr       []byte = make([]byte, 3)
		bcc       []byte = make([]byte, 1)
		dataLen   int32
		frameType int32
	)

	_, err := io.ReadFull(r, hdr)
	if err != nil {
		return -1, nil, ReadPacketError
	}

	m.msgId = hdr[0]
	m.msgType = uint16(hdr[1])<<8 | uint16(hdr[2])

	switch m.msgType {
	case HeartbeatResponse:
		dataLen = 1
		frameType = FrameTypeMessage
	default:
		return -1, nil, MessageUnknownError
	}

	buf := make([]byte, dataLen)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return -1, nil, ReadPacketError
	}

	m.data = buf[:]

	_, err = io.ReadFull(r, bcc)
	if err != nil {
		return -1, nil, ReadPacketError
	}

	return frameType, &m, nil
}

func (p *RsuProtoInst) HandleMessage(msg Message) (resp Message) {
	m := msg.(*RsuMessage)

	switch m.msgType {
	case HeartbeatResponse:
		fmt.Println("received heartbeat response")
	}

	return nil
}

func (p *RsuProtoInst) WriteMessage(w io.Writer, msg Message) error {
	m := msg.(*RsuMessage)

	switch m.msgType {
	case HeartbeatRequest:
	default:
		return MessageUnknownError
	}

	_, err := w.Write(m.Bytes())
	if err != nil {
		return err
	}

	return nil
}

func (p *RsuProtoInst) NewHeartbeatMsg() (hb Message) {
	return NewRsuMessage(p.genMsgId(), HeartbeatRequest)
}

func (p *RsuProtoInst) HeartbeatInterval() time.Duration {
	return 5 * time.Second
}
