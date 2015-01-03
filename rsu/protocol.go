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
	ReadPacketError     = errors.New("read packet error")
	MessageUnknownError = errors.New("unknown message type")
	RsuNotFoundError    = errors.New("RSU not found")
	SetParameterError   = errors.New("set parameter error")
)

// Message types
const (
	HeartbeatRequest  uint16 = 0xD468
	HeartbeatResponse uint16 = 0xC468

	GetTxPowerRequest  uint16 = 0xD371
	GetTxPowerResponse uint16 = 0xC371

	SetTxPowerRequest  uint16 = 0xD06F
	SetTxPowerResponse uint16 = 0xC06F
)

type RsuMessage struct {
	msgId   uint8
	msgType uint16
	data    []byte
}

func NewRsuMessage(msgId uint8, msgType uint16, data []byte) *RsuMessage {
	return &RsuMessage{
		msgId:   msgId,
		msgType: msgType,
		data:    data,
	}
}

func (m *RsuMessage) TxPower() uint8 {
	return m.data[0]
}

func (m *RsuMessage) RsuStatus() uint8 {
	return m.data[0]
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
	b = append(b, 0xFF)
	b = append(b, 0xFF)
	b = append(b, 0x80|m.msgId)
	b = append(b, uint8((m.msgType&0xFF00)>>8))
	b = append(b, uint8(m.msgType&0x00FF))
	b = append(b, m.data...)
	b = append(b, GetBCC(b[:]))
	b = append(b, 0xFF)
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
		hdr       []byte = make([]byte, 5)
		end       []byte = make([]byte, 2)
		dataLen   int32
		frameType int32
	)

	_, err := io.ReadFull(r, hdr)
	if err != nil {
		fmt.Println("Read message header error")
		return -1, nil, ReadPacketError
	}

	m.msgId = hdr[2]
	m.msgType = uint16(hdr[3])<<8 | uint16(hdr[4])

	switch m.msgType {
	case HeartbeatResponse:
		dataLen = 1
		frameType = FrameTypeMessage
		fmt.Printf("Heartbeat <- ")
	case GetTxPowerResponse:
		dataLen = 1
		frameType = FrameTypeResponse
		fmt.Printf("Get TxPower <- ")
	case SetTxPowerResponse:
		dataLen = 1
		frameType = FrameTypeResponse
		fmt.Printf("Set TxPower <- ")
	default:
		fmt.Printf("Unknown message type(%x)\n", hdr[3:])
		return -1, nil, MessageUnknownError
	}

	fmt.Printf("%x", hdr)

	if dataLen > 0 {
		data := make([]byte, dataLen)
		_, err = io.ReadFull(r, data)
		if err != nil {
			fmt.Println("Read message payload error")
			return -1, nil, ReadPacketError
		}
		m.data = data[:]
		fmt.Printf("%x", data)
	}

	_, err = io.ReadFull(r, end)
	if err != nil {
		fmt.Println("Read message trailer error")
		return -1, nil, ReadPacketError
	}
	fmt.Printf("%x\n", end)

	return frameType, &m, nil
}

func (p *RsuProtoInst) HandleMessage(msg Message) Message {
	m := msg.(*RsuMessage)

	switch m.msgType {
	case HeartbeatResponse:
		// do nothing
	}

	return nil
}

func (p *RsuProtoInst) WriteMessage(w io.Writer, msg Message) error {
	m := msg.(*RsuMessage)

	switch m.msgType {
	case HeartbeatRequest:
		fmt.Printf("Heartbeat -> ")
	case GetTxPowerRequest:
		fmt.Printf("Get TxPower -> ")
	case SetTxPowerRequest:
		fmt.Printf("Set TxPower -> ")
	default:
		fmt.Println("Error sending unknown message type")
		return MessageUnknownError
	}

	buf := m.Bytes()
	fmt.Printf("%x\n", buf)

	_, err := w.Write(buf)
	if err != nil {
		fmt.Println("Failed to send message")
		return err
	}

	return nil
}

func (p *RsuProtoInst) NewGetTxPowerMsg() Message {
	return NewRsuMessage(p.genMsgId(), GetTxPowerRequest, nil)
}

func (p *RsuProtoInst) NewSetTxPowerMsg(txPower uint8) Message {
	data := make([]byte, 1)
	data[0] = txPower
	return NewRsuMessage(p.genMsgId(), SetTxPowerRequest, data)
}

func (p *RsuProtoInst) NewHeartbeatMsg() Message {
	return NewRsuMessage(p.genMsgId(), HeartbeatRequest, nil)
}

func (p *RsuProtoInst) HeartbeatInterval() time.Duration {
	return 5 * time.Second
}
