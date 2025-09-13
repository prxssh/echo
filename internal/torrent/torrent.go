package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"log/slog"

	"github.com/prxssh/echo/internal/tracker"
)

type Torrent struct {
	PeerID         [sha1.Size]byte  `json:"-"`
	Metainfo       *Metainfo        `json:"metainfo"`
	TrackerManager *tracker.Manager `json:"-"`
	Uploaded       uint64           `json:"uploaded"`
	Downloaded     uint64           `json:"downloaded"`
	Left           uint64           `json:"left"`
}

func ParseTorrent(data []byte) (*Torrent, error) {
	peerID, err := generatePeerID()
	if err != nil {
		return nil, err
	}

	metainfo, err := ParseMetainfo(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	trackerManager := tracker.NewManager(
		metainfo.AnnounceURLs,
		tracker.Opts{
			InfoHash: metainfo.Info.Hash,
			PeerID:   peerID,
			Port:     6969,
			Left:     metainfo.Size,
		},
	)

	torrent := &Torrent{
		PeerID:         peerID,
		Metainfo:       metainfo,
		TrackerManager: trackerManager,
		Left:           metainfo.Size,
	}
	trackerManager.SetOnPeers(torrent.connectRemotePeers)

	return torrent, nil
}

func (t *Torrent) Start() {
}

func (t *Torrent) connectRemotePeers(from string, peers []*tracker.Peer) {
	slog.Debug(
		"received peers from trackers",
		slog.String("url", from),
		slog.Int("lenPeers", len(peers)),
	)
}

func generatePeerID() ([sha1.Size]byte, error) {
	var peerID [sha1.Size]byte

	prefix := []byte("-EC0001-")
	copy(peerID[:], prefix)

	if _, err := rand.Read(peerID[len(prefix):]); err != nil {
		return [sha1.Size]byte{}, err
	}

	return peerID, nil
}
