package main

import (
	"os"
	"torrent_client/torrentfile"
)

func main() {
	inPath := os.Args[1]
	outPath := os.Args[2]

	torrentFile, err := torrentfile.Open(inPath)
	if err != nil {
		panic(err)
	}

	err = torrentFile.DownloadToFile(outPath)
	if err != nil {
		panic(err)
	}
}
