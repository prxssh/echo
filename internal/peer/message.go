package peer

import (
	"encoding/binary"
	"io"
)

// Message is a BitTorrent wire message. A nil message represents a keep-alive
// (length prefix 0). For non-keepalive messages, ID is set and Payload contains
// the message-specific data.
type Message struct {
	ID      MessageID
	Payload []byte
}

// MessageID identifies the BitTorrent wire message type.
type MessageID uint8

// Standard BitTorrent message IDs (BEP 3).
const (
	MsgChoke         MessageID = 0
	MsgUnchoke       MessageID = 1
	MsgInterested    MessageID = 2
	MsgNotInterested MessageID = 3
	MsgHave          MessageID = 4
	MsgBitfield      MessageID = 5
	MsgRequest       MessageID = 6
	MsgPiece         MessageID = 7
	MsgCancel        MessageID = 8
)

// Serialize encodes the message to the wire format:
// <length prefix><message ID><payload>. A nil message returns the
// 4-byte zero keep-alive frame.
func (m *Message) Serialize() []byte {
	if m == nil { // keep-alive message
		return make([]byte, 4)
	}

	length := uint32(len(m.Payload) + 1) // +1 for ID
	buf := make([]byte, 4+length)

	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)

	return buf
}

// ReadMessage reads a single message from r. It returns nil for a
// keep-alive frame, or a populated Message for other types.
func ReadMessage(r io.Reader) (*Message, error) {
	var length uint32

	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}

	return &Message{ID: MessageID(buf[0]), Payload: buf[1:]}, nil
}

func WriteMessage(w io.Writer, message *Message) error {
	packet := message.Serialize()
	_, err := w.Write(packet)
	return err
}

// MessageChoke creates a choke message.
func MessageChoke() *Message {
	return &Message{ID: MsgChoke}
}

// MessageUnchoke creates an unchoke message (unexported).
func MessageUnchoke() *Message {
	return &Message{ID: MsgUnchoke}
}

// MessageInterested creates an interested message.
func MessageInterested() *Message {
	return &Message{ID: MsgInterested}
}

// MessageNotInterested creates a not interested message.
func MessageNotInterested() *Message {
	return &Message{ID: MsgNotInterested}
}

// MessageHave creates a have message for a given piece index.
func MessageHave(index int) *Message {
	payload := make([]byte, 4)

	binary.BigEndian.PutUint32(payload, uint32(index))

	return &Message{ID: MsgHave, Payload: payload}
}

// MessageRequest creates a request message for a block defined by
// piece index, begin offset, and length.
func MessageRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &Message{ID: MsgRequest, Payload: payload}
}

// MessagePiece creates a piece message carrying a data block for the
// given piece index and begin offset.
func MessagePiece(index, begin int, block []byte) *Message {
	payload := make([]byte, 8+len(block))

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	copy(payload[8:], block)

	return &Message{ID: MsgPiece, Payload: payload}
}

// MessageCancel creates a cancel message for a previously requested block.
func MessageCancel(index, begin, length int) *Message {
	payload := make([]byte, 12)

	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &Message{ID: MsgCancel, Payload: payload}
}
