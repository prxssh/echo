package main

import (
	"log/slog"
	"os"
	"time"

	"github.com/prxssh/echo/pkg/logging"
)

func main() {
	setupLogger()
	slog.Info("echo is up and running...")
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
