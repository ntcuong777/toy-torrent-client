package handshake

import (
	"fmt"
	"io"
)

type Handshake struct {
	Pstr     string
	InfoHash [20]byte
	Peer     [20]byte
}

func New(peerId, infoHash [20]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		Peer:     peerId,
	}
}

// Serialize serializes the handshake into a buffer
func (h *Handshake) Serialize() []byte {
	buff := make([]byte, len(h.Pstr)+49)
	buff[0] = byte(len(h.Pstr)) // the len is always 19, or 0x13
	curr := 1
	curr += copy(buff[curr:], h.Pstr)
	curr += copy(buff[curr:], make([]byte, 8)) // 8 reserved bytes for extension
	curr += copy(buff[curr:], h.InfoHash[:])
	curr += copy(buff[curr:], h.Peer[:])
	return buff
}

// Read reads the handshake response from a reader, the structure is the same as the initial handshake request
func Read(r io.Reader) (*Handshake, error) {
	lengthBuf := make([]byte, 1)
	_, err := io.ReadFull(r, lengthBuf)
	if err != nil {
		return nil, err
	}

	pstrlen := int(lengthBuf[0])
	if pstrlen == 0 {
		return nil, fmt.Errorf("pstrlen cannot be 0")
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	_, err = io.ReadFull(r, handshakeBuf)
	if err != nil {
		return nil, err
	}

	var infoHash, peer [20]byte
	pstr := string(handshakeBuf[:pstrlen])
	// omit the 8-bit reserved byte
	copy(infoHash[:], handshakeBuf[pstrlen+8:pstrlen+8+20])
	copy(peer[:], handshakeBuf[pstrlen+8+20:])
	h := Handshake{
		Pstr:     pstr,
		InfoHash: infoHash,
		Peer:     peer,
	}
	return &h, nil
}
