package peer

import (
	"encoding/binary"
	"io"
)

type Message struct {
	ID      MessageID
	Payload []byte
}

type MessageID uint8

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

func (mid MessageID) String() string {
	switch mid {
	case MsgChoke:
		return "Choke"
	case MsgUnchoke:
		return "Unchoke"
	case MsgInterested:
		return "Interested"
	case MsgNotInterested:
		return "Not Interested"
	case MsgHave:
		return "Have"
	case MsgBitfield:
		return "Bitfield"
	case MsgRequest:
		return "Request"
	case MsgPiece:
		return "Piece"
	case MsgCancel:
		return "Cancel"
	default:
		return "-"
	}
}

func (m *Message) Serialize() []byte {
	if m == nil { // keep-alive message
		return make([]byte, 4)
	}

	// <length prefix><message ID><payload>
	length := uint32(len(m.Payload) + 1) // +1 for ID
	buf := make([]byte, 4+length)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)

	return buf
}

func ReadMessage(r io.Reader) (*Message, error) {
	var length uint32
	if err := binary.Read(r, binary.BigEndian, &length); err != nil {
		return nil, err
	}
	if length == 0 { // keep-alive
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

func MessageChoke() *Message {
	return &Message{ID: MsgChoke}
}

func MessageUnchoke() *Message {
	return &Message{ID: MsgUnchoke}
}

func MessageInterested() *Message {
	return &Message{ID: MsgInterested}
}

func MessageNotInterested() *Message {
	return &Message{ID: MsgNotInterested}
}

func MessageHave(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))

	return &Message{ID: MsgHave, Payload: payload}
}

func MessageRequest(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &Message{ID: MsgRequest, Payload: payload}
}

func MessagePiece(index, begin int, block []byte) *Message {
	payload := make([]byte, 8+len(block))
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	copy(payload[8:], block)

	return &Message{ID: MsgPiece, Payload: payload}
}

func MessageCancel(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[0:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:12], uint32(length))

	return &Message{ID: MsgCancel, Payload: payload}
}
