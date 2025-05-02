package ui

import (
	"fmt"
	"io"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

type Echo struct {
	app    fyne.App
	window fyne.Window
}

func NewEchoApp() *Echo {
	a := app.New()
	w := a.NewWindow("Echo - BitTorrent Client")
	return &Echo{app: a, window: w}
}

func (e *Echo) Run() {
	addBtn := widget.NewButton("Upload .torrent", func() {
		d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				return
			}
			defer r.Close()

			content, _ := io.ReadAll(r)
			fmt.Printf("Loaded torrent (%d bytes)\n", len(content))
		}, e.window)
		d.SetFilter(storage.NewExtensionFileFilter([]string{".torrent"}))
		d.Show()
	})

	toolbar := container.NewHBox(addBtn)
	list := widget.NewList(
		func() int { return 0 },
		func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {},
	)

	e.window.SetContent(container.NewBorder(toolbar, nil, nil, nil, list))
	e.window.Resize(fyne.NewSize(600, 400))
	e.window.ShowAndRun()
}
