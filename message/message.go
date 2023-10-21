package message

import (
	"encoding/binary"
	"io"
)

type messageID uint8

const (
	// MsgChoke chokes the receiver
	MsgChoke messageID = 0
	// MsgUnchoke unchokes the receiver
	MsgUnchoke messageID = 1
	// MsgInterested expresses interest in receiving data
	MsgInterested messageID = 2
	// MsgNotInterested expresses disinterest in receiving data
	MsgNotInterested messageID = 3
	// MsgHave alerts the receiver that the sender has downloaded a piece
	MsgHave messageID = 4
	// MsgBitfield encodes which pieces that the sender has downloaded
	MsgBitfield messageID = 5
	// MsgRequest requests a block of data from the receiver
	MsgRequest messageID = 6
	// MsgPiece delivers a block of data to fulfill a request
	MsgPiece messageID = 7
	// MsgCancel cancels a request
	MsgCancel messageID = 8
)

type Message struct {
	ID      messageID
	Payload []byte
}

func Read(r io.Reader) (*Message, error) {
	lengthBuff := make([]byte, 4)
	_, err := io.ReadFull(r, lengthBuff)
	if err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(lengthBuff)
	// Keepalive message
	if length == 0 {
		return nil, nil
	}

	msgBuff := make([]byte, length)
	_, err = io.ReadFull(r, msgBuff)
	if err != nil {
		return nil, err
	}

	m := Message{
		ID:      messageID(msgBuff[0]),
		Payload: msgBuff[1:],
	}
	return &m, nil
}

// Serialize serializes the message into a buffer
func Serialize(m *Message) []byte {
	if m == nil {
		return make([]byte, 4) // keepalive message
	}
	length := uint32(len(m.Payload) + 1) // +1 for ID
	buff := make([]byte, length+4)
	binary.BigEndian.PutUint32(buff[:4], length)
	buff[4] = byte(m.ID)
	copy(buff[5:], m.Payload)
	return buff
}
