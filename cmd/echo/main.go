package main

import (
	"fmt"
	"os"

	"github.com/prxssh/echo/internal/torrent"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s <path-to-torrent-file>\n", os.Args[0])
		os.Exit(1)
	}

	path := os.Args[1]
	f, err := os.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening %q: %v\n", path, err)
		os.Exit(1)
	}
	defer f.Close()

	meta, err := torrent.Decode(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to decode torrent: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("announce: %s\n announceList: %v\n name: %s\n files: %v\n", meta.Announce, meta.AnnounceList, meta.Info.Name, meta.Info.Files)
}
