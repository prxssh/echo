package ui

import (
	"crypto/sha1"

	"github.com/prxssh/echo/internal/torrent"
)

// UI holds app UI state and exposes methods to the frontend via Wails bindings.
type UI struct {
	torrents map[[sha1.Size]byte]*torrent.Torrent
}

func New() *UI {
	return &UI{torrents: make(map[[sha1.Size]byte]*torrent.Torrent)}
}

func (ui *UI) ParseTorrent(data []byte) (*torrent.Torrent, error) {
	return torrent.ParseTorrent(data)
}

func (ui *UI) RemoveTorrent(infoHash [sha1.Size]byte) {
	delete(ui.torrents, infoHash)
}
