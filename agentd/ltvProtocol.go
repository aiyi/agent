package agentd

import (
	"encoding/binary"
	"errors"
	"fmt"
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
)

// BigEndian: uint32 --> []byte
func Uint32ToBytes(v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return b
}

// BigEndian: []byte --> uint32
func BytesToUint32(b []byte) uint32 {
	return binary.BigEndian.Uint32(b)
}

// Packet: pacLen + pacType + pacData
// Big endian: uint32 + uint32 + []byte
type LtvMessage struct {
	pacLen  uint32
	pacType uint32
	pacData []byte
}

func (p *LtvMessage) Serialize() []byte {
	buf := make([]byte, 8+len(p.pacData))
	copy(buf[0:4], Uint32ToBytes(p.pacLen))
	copy(buf[4:8], Uint32ToBytes(p.pacType))
	copy(buf[8:], p.pacData)
	return buf
}

func (p *LtvMessage) GetLen() uint32 {
	return p.pacLen
}

func (p *LtvMessage) GetTypeInt() uint32 {
	return p.pacType
}

func (p *LtvMessage) GetTypeString() string {
	return ""
}

func (p *LtvMessage) GetData() []byte {
	return p.pacData
}

func NewLtvMessage(pacType uint32, pacData []byte) *LtvMessage {
	return &LtvMessage{
		pacLen:  uint32(8) + uint32(len(pacData)),
		pacType: pacType,
		pacData: pacData,
	}
}

type LtvProtocol struct {
}

func (this *LtvProtocol) DecodeMessage(r io.Reader) (int32, Message, error) {
	var (
		pacBLen  []byte = make([]byte, 4)
		pacBType []byte = make([]byte, 4)
		pacLen   uint32
	)

	// read pacLen
	if n, err := io.ReadFull(r, pacBLen); err != nil && n != 4 {
		return -1, nil, ReadPacketError
	}
	if pacLen = BytesToUint32(pacBLen); pacLen > 1025 {
		return -1, nil, PacketTooLargeError
	}

	// read pacType
	if n, err := io.ReadFull(r, pacBType); err != nil && n != 4 {
		return -1, nil, ReadPacketError
	}

	// read pacData
	pacData := make([]byte, pacLen-8)
	if n, err := io.ReadFull(r, pacData); err != nil && n != int(pacLen) {
		return -1, nil, ReadPacketError
	}

	return FrameTypeMessage, NewLtvMessage(BytesToUint32(pacBType), pacData), nil
}

func (this *LtvProtocol) HandleMessage(msg Message) (resp Message) {
	return NewLtvMessage(200, []byte("reply ok"))
}

func (this *LtvProtocol) WriteMessage(w io.Writer, msg Message) error {
	ltvmsg := msg.(*LtvMessage)
	if n, err := w.Write(ltvmsg.Serialize()); err != nil || n != int(ltvmsg.GetLen()) {
		return errors.New(fmt.Sprintf("Write error: [%v], n[%v], msg len[%v]", err, n, ltvmsg.GetLen()))
	}

	return nil
}

func (this *LtvProtocol) NewHeartbeatMsg() (hb Message) {
	return NewLtvMessage(200, []byte("PING"))
}

func (this *LtvProtocol) HeartbeatInterval() time.Duration {
	return 5 * time.Second
}
