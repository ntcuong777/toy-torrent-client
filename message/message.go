package message

import (
	"encoding/binary"
	"fmt"
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

func CreateRequestMsg(index, begin, length int) *Message {
	payload := make([]byte, 12)
	binary.BigEndian.PutUint32(payload[:4], uint32(index))
	binary.BigEndian.PutUint32(payload[4:8], uint32(begin))
	binary.BigEndian.PutUint32(payload[8:], uint32(length))
	return &Message{
		ID:      MsgRequest,
		Payload: payload,
	}
}

func CreateHaveMsg(index int) *Message {
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(index))
	return &Message{
		ID:      MsgHave,
		Payload: payload,
	}
}

// Read parses a message from a stream. Returns `nil` on keep-alive message
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
func (m *Message) Serialize() []byte {
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

func ParseHave(msg *Message) (int, error) {
	if msg.ID != MsgHave {
		return 0, fmt.Errorf("expected have message, got %d", msg.ID)
	}
	if len(msg.Payload) != 4 {
		return 0, fmt.Errorf("expected payload length 4, got %d", len(msg.Payload))
	}
	index := binary.BigEndian.Uint32(msg.Payload)
	return int(index), nil
}

func ParsePiece(index int, buf []byte, msg *Message) (int, error) {
	if msg.ID != MsgPiece {
		return 0, fmt.Errorf("expected piece message, got %d", msg.ID)
	}
	if len(msg.Payload) < 8 {
		return 0, fmt.Errorf("expected payload length > 8, got %d", len(msg.Payload))
	}
	parsedIndex := int(binary.BigEndian.Uint32(msg.Payload[:4]))
	if index != parsedIndex {
		return 0, fmt.Errorf("expected index %d, got %d", index, parsedIndex)
	}
	begin := int(binary.BigEndian.Uint32(msg.Payload[4:8]))
	if begin >= len(buf) {
		return 0, fmt.Errorf("expected begin offset < %d, got %d", len(buf), begin)
	}
	payload := msg.Payload[8:]
	if begin+len(payload) > len(buf) {
		return 0, fmt.Errorf("data too long, expected begin + payload length < %d, got %d", len(buf), begin+len(payload))
	}
	copy(buf[begin:], payload)
	return len(payload), nil
}
