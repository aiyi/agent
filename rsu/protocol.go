package rsu

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	kafka "github.com/Shopify/sarama"
	. "github.com/aiyi/agent/agent"
	"io"
	"strings"
	"time"
)

// Error types
var (
	ReadPacketError     = errors.New("read packet error")
	InvalidPacketError  = errors.New("invalid packet")
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

	ObuEventReport uint16 = 0xC465
)

type RsuMessage struct {
	msgId   uint8
	msgType uint16
	data    []byte
}

func (m *RsuMessage) readNBytes(offset int, n int) ([]byte, int) {
	buf := m.data[offset : offset+n]
	numFE := 0

	for i := 0; i < n; i++ {
		if buf[i] == 0xFE {
			numFE++
		}
	}

	if numFE == 0 {
		return buf, offset + n
	}

	b := make([]byte, n)
	n += numFE
	buf = m.data[offset : offset+n]
	i := 0
	for k := 0; k < n; k++ {
		if buf[k] != 0xFE {
			b[i] = buf[k]
		} else {
			b[i] = buf[k] + buf[k+1]
			k++
		}
		i++
	}
	return b, offset + n
}

func (m *RsuMessage) GetTxPower() uint8 {
	return m.data[0]
}

func (m *RsuMessage) GetRsuStatus() uint8 {
	return m.data[0]
}

type ObuEvent struct {
	//RsuTransactionMode uint8
	VehicleNumber string
	VehicleType   uint8
	UserType      uint8
	//ContractSN    [8]byte
	ObuMAC string
	//ObuStatus     [2]byte
	Battery   uint8
	Timestamp int64
	//PSAMID        [6]byte
	//TrSN          [4]byte
	Station uint16
	Roadway uint8
}

func (m *RsuMessage) GetObuEvent() *ObuEvent {
	e := &ObuEvent{}
	var b []byte

	_, offset := m.readNBytes(0, 1)
	b, offset = m.readNBytes(offset, 12)
	e.VehicleNumber, _ = Iconv.ConvertString(string(b))
	e.VehicleNumber = strings.TrimRight(e.VehicleNumber, "\u0000")
	b, offset = m.readNBytes(offset, 1)
	e.VehicleType = b[0]
	b, offset = m.readNBytes(offset, 1)
	e.UserType = b[0]
	_, offset = m.readNBytes(offset, 8)
	b, offset = m.readNBytes(offset, 4)
	e.ObuMAC = fmt.Sprintf("%02x:%02x:%02x:%02x", b[0], b[1], b[2], b[3])
	b, offset = m.readNBytes(offset, 3)
	e.Battery = b[2]
	b, offset = m.readNBytes(offset, 4)
	e.Timestamp = int64(binary.BigEndian.Uint32(b[:]))
	_, offset = m.readNBytes(offset, 6)
	_, offset = m.readNBytes(offset, 4)
	b, offset = m.readNBytes(offset, 2)
	e.Station = binary.BigEndian.Uint16(b[:])
	b, offset = m.readNBytes(offset, 1)
	e.Roadway = b[0]
	return e
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
	bcc := GetBCC(b[:])
	if bcc > 0xFD {
		b = append(b, 0xFE)
		b = append(b, bcc-0xFE)
	} else {
		b = append(b, bcc)
	}
	b = append(b, 0xFF)
	return b
}

type RsuProtocol struct {
}

func (this *RsuProtocol) NewProtoInstance(a *AgentD) ProtoInstance {
	inst := &RsuProtoInst{
		agentd:  a,
		seqChan: make(chan uint8, 8),
	}
	for i := 0; i < 8; i++ {
		inst.seqChan <- uint8(i)
	}

	return inst
}

type RsuProtoInst struct {
	agentd  *AgentD
	seqChan chan uint8
	hdr     [5]byte
	bcc     [1]byte
	end     [1]byte
}

func (p *RsuProtoInst) msgId() uint8 {
	id := <-p.seqChan
	p.seqChan <- id
	return id
}

func (p *RsuProtoInst) NewRsuMessage(msgType uint16, data []byte) *RsuMessage {
	return &RsuMessage{
		msgId:   p.msgId(),
		msgType: msgType,
		data:    data,
	}
}

func (p *RsuProtoInst) DecodeMessage(r io.Reader) (int32, Message, error) {
	var (
		m         RsuMessage
		dataLen   int
		numFE     int
		frameType int32
	)

	_, err := io.ReadFull(r, p.hdr[:])
	if err != nil {
		fmt.Println("Read header error")
		return -1, nil, ReadPacketError
	}
	if p.hdr[0] != 0xFF || p.hdr[1] != 0xFF {
		fmt.Println("Invalid STX")
		return -1, nil, InvalidPacketError
	}

	m.msgId = p.hdr[2]
	m.msgType = uint16(p.hdr[3])<<8 | uint16(p.hdr[4])

	switch m.msgType {
	case HeartbeatResponse:
		dataLen = 1
		frameType = FrameTypeMessage
		fmt.Printf("Heartbeat <- ")
	case ObuEventReport:
		dataLen = 65
		frameType = FrameTypeMessage
		fmt.Printf("OBU Event <- ")
	case GetTxPowerResponse:
		dataLen = 1
		frameType = FrameTypeResponse
		fmt.Printf("Get TxPower <- ")
	case SetTxPowerResponse:
		dataLen = 1
		frameType = FrameTypeResponse
		fmt.Printf("Set TxPower <- ")
	default:
		fmt.Printf("Unknown message type(%x)\n", p.hdr[3:])
		return -1, nil, MessageUnknownError
	}

	fmt.Printf("%x", p.hdr)

	if dataLen > 0 {
		data := make([]byte, dataLen)
		_, err = io.ReadFull(r, data)
		if err != nil {
			fmt.Println("Read payload error")
			return -1, nil, ReadPacketError
		}
		m.data = data[:]

	moredata:
		numFE = 0
		for i := 0; i < dataLen; i++ {
			if data[i] == 0xFE {
				numFE++
			}
		}
		if numFE > 0 {
			dataLen = numFE
			data = make([]byte, dataLen)
			_, err = io.ReadFull(r, data)
			if err != nil {
				fmt.Println("Read payload error")
				return -1, nil, ReadPacketError
			}
			m.data = append(m.data, data...)
			goto moredata
		}

		fmt.Printf("%x", m.data)
	}

	_, err = io.ReadFull(r, p.bcc[:])
	if err != nil {
		fmt.Println("Read BCC error")
		return -1, nil, ReadPacketError
	}
	fmt.Printf("%x", p.bcc)
	if p.bcc[0] == 0xFE {
		_, err = io.ReadFull(r, p.bcc[:])
		if err != nil {
			fmt.Println("Read BCC error")
			return -1, nil, ReadPacketError
		}
		fmt.Printf("%x", p.bcc)
	}

	_, err = io.ReadFull(r, p.end[:])
	if err != nil {
		fmt.Println("Read ETX error")
		return -1, nil, ReadPacketError
	}
	fmt.Printf("%x\n", p.end)
	if p.end[0] != 0xFF {
		fmt.Println("Invalid ETX")
		return -1, nil, InvalidPacketError
	}

	return frameType, &m, nil
}

func (p *RsuProtoInst) HandleMessage(msg Message) Message {
	m := msg.(*RsuMessage)

	switch m.msgType {
	case HeartbeatResponse:
		// do nothing
	case ObuEventReport:
		event := m.GetObuEvent()
		buf, _ := json.Marshal(event)
		fmt.Println(time.Unix(event.Timestamp, 0))
		fmt.Println(string(buf))
		err := p.agentd.KafkaProducer.SendMessage(nil, kafka.StringEncoder(buf))
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("> message sent to broker")
		}
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
	return p.NewRsuMessage(GetTxPowerRequest, nil)
}

func (p *RsuProtoInst) NewSetTxPowerMsg(txPower uint8) Message {
	data := make([]byte, 1)
	data[0] = txPower
	return p.NewRsuMessage(SetTxPowerRequest, data)
}

func (p *RsuProtoInst) NewHeartbeatMsg() Message {
	return p.NewRsuMessage(HeartbeatRequest, nil)
}

func (p *RsuProtoInst) HeartbeatInterval() time.Duration {
	return 5 * time.Second
}
