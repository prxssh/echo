package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/prxssh/echo/internal/torrent"
	"github.com/prxssh/echo/pkg/log"
)

func main() {
	setupLogger()

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
	slog.Info(
		"Parsed torrent file",
		slog.String("path", path),
		slog.String("name", meta.Info.Name),
		slog.String("announce", meta.Announce),
		slog.Any("announceList", meta.AnnounceList),
		slog.Any("files", meta.Info.Files),
	)
}

func setupLogger() {
	prettyHandler := log.NewHandler(&slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ReplaceAttr: nil,
	})
	logger := slog.New(prettyHandler)
	slog.SetDefault(logger)
}
