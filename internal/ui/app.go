package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"github.com/prxssh/echo/internal/torrent"
)

type Echo struct {
	app    fyne.App
	window fyne.Window

	torrentClient *torrent.Client
}

func NewEchoApp() *Echo {
	a := app.New()
	w := a.NewWindow("Echo - BitTorrent Client")

	return &Echo{app: a, window: w, torrentClient: torrent.NewClient()}
}

func (e *Echo) Run() {
	toolbar := container.NewHBox(e.ButtonUploadTorrent())
	list := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {},
	)

	e.window.SetContent(container.NewBorder(toolbar, nil, nil, nil, list))
	e.window.Resize(fyne.NewSize(600, 400))
	e.window.ShowAndRun()
}
