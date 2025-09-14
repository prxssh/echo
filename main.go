package main

import (
	"context"
	"embed"
	"log/slog"
	"os"
	"time"

	"github.com/prxssh/echo/internal/ui"
	"github.com/prxssh/echo/pkg/logging"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	setupLogger()
	app := ui.New()

	err := wails.Run(&options.App{
		Title:      "Echo - BitTorrent Client & Search Engine",
		Fullscreen: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			app.Startup(ctx)
		},
		Bind:             []any{app},
		BackgroundColour: &options.RGBA{R: 27, G: 38, B: 54, A: 1},
	})
	if err != nil {
		slog.Error(
			"failed to start wails",
			slog.String("error", err.Error()),
		)
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
