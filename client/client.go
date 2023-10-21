package client

import (
	"bytes"
	"fmt"
	"net"
	"time"
	"torrent_client/bitfield"
	"torrent_client/handshake"
	"torrent_client/message"
	"torrent_client/peers"
)

// A Client is a TCP connection with a peer
type Client struct {
	Conn     net.Conn
	Choked   bool
	Bitfield bitfield.Bitfield
	Peer     peers.Peer
	InfoHash [20]byte
	PeerId   [20]byte
}

func receiveBitField(conn net.Conn) (bitfield.Bitfield, error) {
	conn.SetDeadline(time.Now().Add(5 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	msg, err := message.Read(conn)
	if err != nil {
		return nil, err
	}
	if msg == nil {
		return nil, fmt.Errorf("expected bitfield, got nil")
	}
	if msg.ID != message.MsgBitfield {
		return nil, fmt.Errorf("expected bitfield, got ID %d", msg.ID)
	}
	return msg.Payload, nil
}

func completeHandshake(conn net.Conn, peerId, infoHash [20]byte) (*handshake.Handshake, error) {
	_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	req := handshake.New(peerId, infoHash)
	_, err := conn.Write(req.Serialize())
	if err != nil {
		return nil, err
	}

	res, err := handshake.Read(conn)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(res.InfoHash[:], infoHash[:]) {
		return nil, fmt.Errorf("expected infohash %x, got %x", infoHash, res.InfoHash)
	}
	return res, nil
}

func New(peer peers.Peer, peerId [20]byte, infoHash [20]byte) (*Client, error) {
	conn, err := net.DialTimeout("tcp", peer.String(), 3*time.Second)
	if err != nil {
		return nil, err
	}

	_, err = completeHandshake(conn, peerId, infoHash)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	bf, err := receiveBitField(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	client := Client{
		Conn:     conn,
		Choked:   true,
		Bitfield: bf,
		Peer:     peer,
		InfoHash: infoHash,
		PeerId:   peerId,
	}
	return &client, nil
}
