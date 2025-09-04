package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/prxssh/echo/internal/torrent"
	"github.com/prxssh/echo/pkg/logging"
)

func main() {
	setupLogger()

	torrent, err := torrent.ReadFile("./data/ubuntu.torrent")
	if err != nil {
		slog.Error(
			"failed to read torrent from path",
			slog.Any("error", err),
		)
		os.Exit(1)
	}

	slog.Info("torrent info", slog.Any("torrent", torrent))
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
