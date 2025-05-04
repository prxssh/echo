package ui

import (
	"io"
	"log/slog"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
)

func (e *Echo) ButtonUploadTorrent() *widget.Button {
	return widget.NewButton("Upload .torrent", func() {
		d := dialog.NewFileOpen(func(r fyne.URIReadCloser, err error) {
			if err != nil || r == nil {
				dialog.ShowError(err, e.window)
				return
			}
			defer r.Close()

			content, err := io.ReadAll(r)
			if err != nil {
				dialog.ShowError(err, e.window)
				return
			}

			meta, err := e.torrentClient.UnmarshalMetainfo(content)
			if err != nil {
				dialog.ShowError(err, e.window)
				return
			}
			session, err := e.torrentClient.Add(meta)
			if err != nil {
				dialog.ShowError(err, e.window)
				return

			}
			slog.Info("Uploaded .torrent", slog.String("Announce", session.Meta.Announce))
		}, e.window)
		d.SetFilter(storage.NewExtensionFileFilter([]string{".torrent"}))
		d.Show()
	})
}
