package p2p

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"log"
	"time"
	"torrent_client/client"
	"torrent_client/message"
	"torrent_client/peers"
)

const MaxBlockSize = 16384     // max size of each piece block in bytes
const MaxParallelDownloads = 5 // max number of block downloads to make in parallel

type pieceWork struct {
	index  int
	hash   [20]byte
	length int
}

type pieceResult struct {
	index int
	buf   []byte
}

type pieceProgress struct {
	index                int
	client               *client.Client
	buff                 []byte
	numParallelDownloads int
	downloaded           int
	requested            int
}

// Torrent holds data required to download a torrent from a list of peers
type Torrent struct {
	Peers         []peers.Peer
	MachinePeerID [20]byte
	InfoHash      [20]byte
	PieceHashes   [][20]byte
	PiecesLength  int
	Length        int
	Name          string
}

func (p *pieceProgress) readMessage() error {
	msg, err := p.client.Read()
	if err != nil {
		return err
	}

	if msg == nil {
		return nil // keepalive
	}

	switch msg.ID {
	case message.MsgUnchoke:
		p.client.Choked = false
	case message.MsgChoke:
		p.client.Choked = true
	case message.MsgHave:
		index, err := message.ParseHave(msg)
		if err != nil {
			return err
		}
		p.client.Bitfield.SetPiece(index)
	case message.MsgPiece:
		n, err := message.ParsePiece(p.index, p.buff, msg)
		if err != nil {
			return err
		}
		p.downloaded += n
		p.numParallelDownloads--
	}
	return nil
}

func attemptDownloadPiece(c *client.Client, work *pieceWork) ([]byte, error) {
	progress := pieceProgress{
		index:                work.index,
		client:               c,
		buff:                 make([]byte, work.length),
		numParallelDownloads: 0,
		downloaded:           0,
		requested:            0,
	}

	// Setting a deadline helps get unresponsive peers unstuck.
	// 30 seconds is more than enough time to download a 262 KB piece
	conn := c.Conn
	conn.SetDeadline(time.Now().Add(30 * time.Second))
	defer conn.SetDeadline(time.Time{}) // Disable the deadline

	for progress.downloaded < work.length {
		if !progress.client.Choked {
			for progress.numParallelDownloads < MaxParallelDownloads && progress.requested < work.length {
				blockSize := MaxBlockSize
				if work.length-progress.requested < blockSize {
					blockSize = work.length - progress.requested
				}

				err := c.SendRequest(work.index, progress.requested, blockSize)
				if err != nil {
					return nil, err
				}
				progress.numParallelDownloads++
				progress.requested += blockSize
			}
		}
		err := progress.readMessage()
		if err != nil {
			return nil, err
		}
	}
	return progress.buff, nil
}

func checkIntegrity(work *pieceWork, buf []byte) error {
	bufHash := sha1.Sum(buf)
	if !bytes.Equal(bufHash[:], work.hash[:]) {
		return fmt.Errorf("piece %d failed integrity check", work.index)
	}
	return nil
}

func (t *Torrent) startDownloadWorker(peer peers.Peer, workQueue chan *pieceWork, results chan *pieceResult) {
	c, err := client.NewClient(peer, t.MachinePeerID, t.InfoHash)
	if err != nil {
		log.Println("Error connecting to peer ", peer.IP, ":", peer.Port)
		return
	}
	defer c.Conn.Close()
	log.Println("Connected to peer ", peer.IP, ":", peer.Port)

	c.SendUnchoke()
	c.SendInterested()

	for work := range workQueue {
		if !c.Bitfield.HasPiece(work.index) {
			workQueue <- work // put back for downloading from other peers
			continue
		}

		buf, err := attemptDownloadPiece(c, work)
		if err != nil {
			log.Println("Error downloading piece ", work.index, " from ", peer.IP, ":", peer.Port, ". Exiting...")
			workQueue <- work // put back for downloading from other peers
			return
		}

		err = checkIntegrity(work, buf)
		if err != nil {
			log.Println("Piece ", work.index, " failed integrity check from ", peer.IP, ":", peer.Port)
			workQueue <- work // put back for retry
			continue
		}

		c.SendHave(work.index)
		results <- &pieceResult{work.index, buf}
	}
}

func (t *Torrent) getPieceRange(index int) (begin int, end int) {
	begin = index * t.PiecesLength
	end = begin + t.PiecesLength
	if end > t.Length {
		end = t.Length
	}
	return begin, end
}

func (t *Torrent) getPieceSize(index int) int {
	begin, end := t.getPieceRange(index)
	return end - begin
}

// Download downloads a torrent from a list of peers
func (t *Torrent) Download() ([]byte, error) {
	log.Println("Starting download for torrent ", t.Name)
	// queues to retrieve pieces and save downloaded pieces
	workQueue := make(chan *pieceWork, len(t.PieceHashes))
	results := make(chan *pieceResult)
	// create a goroutine for each peer
	for index, hash := range t.PieceHashes {
		length := t.getPieceSize(index)
		workQueue <- &pieceWork{index, hash, length}
	}

	// Start workers
	for _, peer := range t.Peers {
		go t.startDownloadWorker(peer, workQueue, results)
	}

	// Collect results into a buffer in correct order
	buf := make([]byte, t.Length)
	donePieces := 0
	for donePieces < len(t.PieceHashes) {
		res := <-results
		begin, end := t.getPieceRange(res.index)
		copy(buf[begin:end], res.buf)
		donePieces++

		percent := float64(donePieces) / float64(len(t.PieceHashes)) * 100
		numWorkers := len(t.Peers) - 1 // don't count the main thread
		log.Printf("(%0.2f%%) Downloaded piece #%d from %d peers\n", percent, res.index, numWorkers)
	}
	close(workQueue)

	return buf, nil
}
