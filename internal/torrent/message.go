package torrent

import (
	"encoding/binary"
	"io"
)

type messageID uint8

const (
	MsgChoke        messageID = 0
	MsgUnchoke      messageID = 1
	MsgInterested   messageID = 2
	MsgUninterested messageID = 3
	MsgHave         messageID = 4
	MsgBitfield     messageID = 5
	MsgRequest      messageID = 6
	MsgPiece        messageID = 7
	MsgCancel       messageID = 8
	MsgPort         messageID = 9
)

// Message stores ID and payload of a message
type Message struct {
	ID      messageID
	Payload []byte
}

// Marshal serializes a message into a buffer of the form:
// <len prefix><message ID><payload>
// Interprets `nil` as a keep-alive message
func (m *Message) Marshal() []byte {
	if m == nil {
		return make([]byte, 4)
	}

	length := uint32(len(m.Payload) + 1) // + 1 for id
	buf := make([]byte, length+4)
	binary.BigEndian.PutUint32(buf[0:4], length)
	buf[4] = byte(m.ID)
	copy(buf[5:], m.Payload)

	return buf
}

func ReadMessage(r io.Reader) (*Message, error) {
	lenBuf := make([]byte, 4)
	_, err := io.ReadFull(r, lenBuf)
	if err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf)

	// keep-alive message
	if length == 0 {
		return nil, nil
	}

	messageBuf := make([]byte, length)
	if _, err := io.ReadFull(r, messageBuf); err != nil {
		return nil, err
	}

	return &Message{ID: messageID(messageBuf[0]), Payload: messageBuf[1:]}, nil
}
