package main

import (
	"log/slog"

	"github.com/prxssh/echo/internal/ui"
	"github.com/prxssh/echo/pkg/log"
)

func main() {
	setupLogger()

	ui.NewEchoApp().Run()
}

func setupLogger() {
	prettyHandler := log.NewHandler(&slog.HandlerOptions{
		Level:       slog.LevelInfo,
		AddSource:   false,
		ReplaceAttr: nil,
	})
	l := slog.New(prettyHandler)
	slog.SetDefault(l)
}
