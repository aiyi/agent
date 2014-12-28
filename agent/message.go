package agent

import (
	"io"
	"time"
)

// frame types
const (
	FrameTypeResponse int32 = 0
	FrameTypeError    int32 = 1
	FrameTypeMessage  int32 = 2
)

// The number of bytes for a Message.ID
const MsgIDLength = 16

// MessageID is the ASCII encoded hexadecimal message ID
type MessageID [MsgIDLength]byte

/*
// Message is the fundamental data type containing
// the id, body, and metadata
type Message struct {
	ID        MessageID
	Body      []byte
	Timestamp int64
	Attempts  uint16

	autoResponseDisabled int32
	responded            int32
}

// NewMessage creates a Message, initializes some metadata,
// and returns a pointer

func NewMessage(id MessageID, body []byte) *Message {
	return &Message{
		ID:        id,
		Body:      body,
		Timestamp: time.Now().UnixNano(),
	}
}
*/
// WriteTo implements the WriterTo interface and serializes
// the message into the supplied producer.
//
// It is suggested that the target Writer is buffered to
// avoid performing many system calls.
/*
func (m *Message) WriteTo(w io.Writer) (int64, error) {
	var buf [10]byte
	var total int64

	binary.BigEndian.PutUint64(buf[:8], uint64(m.Timestamp))
	binary.BigEndian.PutUint16(buf[8:10], uint16(m.Attempts))

	n, err := w.Write(buf[:])
	total += int64(n)
	if err != nil {
		return total, err
	}

	n, err = w.Write(m.ID[:])
	total += int64(n)
	if err != nil {
		return total, err
	}

	n, err = w.Write(m.Body)
	total += int64(n)
	if err != nil {
		return total, err
	}

	return total, nil
}
*/
// DecodeMessage deseralizes data (as []byte) and creates a new Message
/*
func DecodeMessage(b []byte) (*Message, error) {
	var msg Message

	msg.Timestamp = int64(binary.BigEndian.Uint64(b[:8]))
	msg.Attempts = binary.BigEndian.Uint16(b[8:10])

	buf := bytes.NewBuffer(b[10:])

	_, err := io.ReadFull(buf, msg.ID[:])
	if err != nil {
		return nil, err
	}

	msg.Body, err = ioutil.ReadAll(buf)
	if err != nil {
		return nil, err
	}

	return &msg, nil
}
*/

type Message interface {
}

type Protocol interface {
	NewProtoInstance() ProtoInstance
}

type ProtoInstance interface {
	DecodeMessage(r io.Reader) (int32, Message, error)
	HandleMessage(msg Message) (resp Message)
	WriteMessage(w io.Writer, msg Message) error
	NewHeartbeatMsg() Message
	HeartbeatInterval() time.Duration
}

/*
// ReadResponse is a client-side utility function to read from the supplied Reader
// according to the NSQ protocol spec:
//
//    [x][x][x][x][x][x][x][x]...
//    |  (int32) || (binary)
//    |  4-byte  || N-byte
//    ------------------------...
//        size       data
func ReadResponse(r io.Reader) ([]byte, error) {
	var msgSize int32

	// message size
	err := binary.Read(r, binary.BigEndian, &msgSize)
	if err != nil {
		return nil, err
	}

	// message binary data
	buf := make([]byte, msgSize)
	_, err = io.ReadFull(r, buf)
	if err != nil {
		return nil, err
	}

	return buf, nil
}

// UnpackResponse is a client-side utility function that unpacks serialized data
// according to NSQ protocol spec:
//
//    [x][x][x][x][x][x][x][x]...
//    |  (int32) || (binary)
//    |  4-byte  || N-byte
//    ------------------------...
//      frame ID     data
//
// Returns a triplicate of: frame type, data ([]byte), error
func UnpackResponse(response []byte) (int32, []byte, error) {
	//if len(response) < 4 {
	//	return -1, nil, errors.New("length of response is too small")
	//}

	return int32(binary.BigEndian.Uint32(response)), response[4:], nil
}

// ReadUnpackedResponse reads and parses data from the underlying
// TCP connection according to the NSQ TCP protocol spec and
// returns the frameType, data or error
func DecodeMessage(c *Conn) (int32, Message, error) {
	//resp, err := ReadResponse(r)
	//if err != nil {
	//	return -1, nil, err
	//}
	//return UnpackResponse(resp)
	packet, err := c.protocol.ReadPacket(c.r, 1024)
	if err != nil {
		return -1, nil, err
	}
	return FrameTypeMessage, packet, nil
}
*/
