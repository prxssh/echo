package ui

import (
	"context"
	"crypto/sha1"

	"github.com/prxssh/echo/internal/torrent"
)

type UI struct {
	ctx      context.Context
	torrents map[[sha1.Size]byte]*torrent.Torrent
}

func New() *UI {
	return &UI{torrents: make(map[[sha1.Size]byte]*torrent.Torrent)}
}

func (ui *UI) Startup(ctx context.Context) {
	ui.ctx = ctx
}

func (ui *UI) AddTorrent(data []byte) (*torrent.Torrent, error) {
	torrent, err := torrent.ParseTorrent(data)
	if err != nil {
		return nil, err
	}
	torrent.Start(ui.ctx)

	return torrent, nil
}

func (ui *UI) RemoveTorrent(infoHash [sha1.Size]byte) {
	delete(ui.torrents, infoHash)
}
