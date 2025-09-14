package torrent

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1"

	"github.com/prxssh/echo/internal/peer"
	"github.com/prxssh/echo/internal/tracker"
)

type Torrent struct {
	PeerID         [sha1.Size]byte  `json:"-"`
	Metainfo       *Metainfo        `json:"metainfo"`
	TrackerManager *tracker.Manager `json:"-"`
	Uploaded       uint64           `json:"uploaded"`
	Downloaded     uint64           `json:"downloaded"`
	Left           uint64           `json:"left"`
	PeerManager    *peer.Manager    `json:"-"`
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

	peerManager, err := peer.NewManager(
		metainfo.Info.Hash,
		peerID,
		len(metainfo.Info.Pieces),
		nil,
	)
	if err != nil {
		return nil, err
	}

	trackerManager, err := tracker.NewManager(
		metainfo.AnnounceURLs,
		tracker.Opts{
			InfoHash: metainfo.Info.Hash,
			PeerID:   peerID,
			Port:     6969,
			Left:     metainfo.Size,
			OnPeers:  peerManager.Enqueue,
		},
	)
	if err != nil {
		return nil, err
	}

	torrent := &Torrent{
		PeerID:         peerID,
		Metainfo:       metainfo,
		TrackerManager: trackerManager,
		Left:           metainfo.Size,
		PeerManager:    peerManager,
	}

	return torrent, nil
}

func (t *Torrent) Start(ctx context.Context) {
	go t.TrackerManager.Start(ctx)
}

func (t *Torrent) Close() {
}

func connectRemotePeers(
	from string,
	peers []*tracker.Peer,
	peerManager *peer.Manager,
) {
	peerManager.Enqueue(peers)
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
