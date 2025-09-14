package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"time"

	"github.com/prxssh/echo/internal/tracker"
	"github.com/prxssh/echo/pkg/logging"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	setupLogger()

	err := wails.Run(&options.App{
		Title:  "Echo - BitTorrent Client & Search Engine",
		Width:  1024,
		Height: 768,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			tracker.SetWailsContext(ctx)
		},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
	})
	if err != nil {
		slog.Error("failed to start wails", slog.Any("error", err))
		os.Exit(1)
	}
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
