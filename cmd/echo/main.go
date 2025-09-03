package main

import (
	"bytes"
	"log/slog"
	"os"
	"time"

	"github.com/prxssh/echo/internal/torrent"
	"github.com/prxssh/echo/pkg/logging"
)

func main() {
	setupLogger()

	data, err := os.ReadFile("./data/fedora.torrent")
	if err != nil {
		os.Exit(1)
	}

	buf := bytes.NewBuffer(data)
	metainfo, err := torrent.New(buf)
	if err != nil {
		os.Exit(1)
	}

	slog.Info("read torrent file from path", slog.Any("metainfo", metainfo))
}

func setupLogger() {
	opts := &logging.PrettyHandlerOptions{
		SlogOpts: slog.HandlerOptions{
			Level:     slog.LevelDebug,
			AddSource: true,
		},
		UseColor:          true,
		ShowSource:        true,
		FullSource:        false,
		CompactJSON:       false,
		TimeFormat:        time.RFC3339,
		LevelWidth:        7,
		FieldSeparator:    " | ",
		DisableHTMLEscape: true,
	}
	handler := logging.NewPrettyHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)
}
