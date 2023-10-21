package torrentfile

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"github.com/jackpal/bencode-go"
	"io"
)

type TorrentFile struct {
	Announce     string
	InfoHash     [20]byte
	PieceHashes  [][20]byte
	PiecesLength int
	Length       int
	Name         string
}

type bencodeInfo struct {
	Pieces      string `bencode:"pieces"`
	PieceLength int    `bencode:"piece length"`
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
}

type bencodeTorrent struct {
	Announce string      `bencode:"announce"`
	Info     bencodeInfo `bencode:"info"`
}

// Open opens a torrentfile file and returns a bencodeTorrent struct
func Open(r io.Reader) (*bencodeTorrent, error) {
	bto := bencodeTorrent{}
	err := bencode.Unmarshal(r, &bto)
	if err != nil {
		return nil, err
	}
	return &bto, nil
}

func (i *bencodeInfo) hash() ([20]byte, error) {
	var buff bytes.Buffer
	err := bencode.Marshal(&buff, i)
	if err != nil {
		return [20]byte{}, err
	}
	h := sha1.Sum(buff.Bytes())
	return h, nil
}

func (i *bencodeInfo) splitPieceHashes() ([][20]byte, error) {
	hashLen := 20 // SHA1 length
	buff := []byte(i.Pieces)
	if len(buff)%hashLen != 0 {
		return nil, fmt.Errorf("received malformed pieces of length %d", len(buff))
	}
	numHashes := len(buff) / hashLen
	hashes := make([][20]byte, numHashes)
	for i := 0; i < numHashes; i++ {
		copy(hashes[i][:], buff[i*hashLen:(i+1)*hashLen])
	}
	return hashes, nil
}

func (bto *bencodeTorrent) toTorrentFile() (TorrentFile, error) {
	infoHash, err := bto.Info.hash()
	if err != nil {
		return TorrentFile{}, err
	}
	pieceHashes, err := bto.Info.splitPieceHashes()
	if err != nil {
		return TorrentFile{}, err
	}
	t := TorrentFile{
		Announce:     bto.Announce,
		InfoHash:     infoHash,
		PieceHashes:  pieceHashes,
		PiecesLength: bto.Info.PieceLength,
		Length:       bto.Info.Length,
		Name:         bto.Info.Name,
	}
	return t, nil
}
