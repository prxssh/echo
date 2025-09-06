package torrent

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/prxssh/echo/internal/tracker"
	"golang.org/x/sync/errgroup"
)

type Torrent struct {
	PeerID     [sha1.Size]byte   `json:"-"`
	Metainfo   *Metainfo         `json:"metainfo"`
	Trackers   []tracker.Tracker `json:"-"`
	Uploaded   uint64            `json:"uploaded"`
	Downloaded uint64            `json:"downloaded"`
	Left       uint64            `json:"left"`
}

func ParseTorrent(data []byte) (*Torrent, error) {
	peerID, err := generatePeerID()
	if err != nil {
		return nil, err
	}

	metainfo, err := parseMetainfo(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}

	trackers, err := initializeTrackers(metainfo.AnnounceURLs)
	if err != nil {
		return nil, err
	}

	return &Torrent{
		PeerID:   peerID,
		Metainfo: metainfo,
		Trackers: trackers,
		Left:     metainfo.Size,
	}, nil
}

func initializeTrackers(announceURLs []string) ([]tracker.Tracker, error) {
	var (
		grp      errgroup.Group
		mut      sync.Mutex
		trackers []tracker.Tracker
	)

	for _, url := range announceURLs {
		u := url
		grp.Go(func() error {
			t, err := tracker.New(url)
			if err != nil {
				slog.Warn(
					"failed to initialize new tracker",
					slog.String("announceURL", u),
					slog.String("error", err.Error()),
				)

				return nil
			}

			mut.Lock()
			trackers = append(trackers, t)
			mut.Unlock()

			return nil
		})

	}

	if err := grp.Wait(); err != nil {
		return nil, err
	}

	if len(trackers) < 1 {
		return nil, errors.New("no trackers available")
	}
	return trackers, nil
}

func generatePeerID() ([sha1.Size]byte, error) {
	var peerID [sha1.Size]byte

	prefix := []byte("-EC0001-")
	copy(peerID[:], prefix)

	if _, err := rand.Read(peerID[len(prefix):]); err != nil {
		return [sha1.Size]byte{}, fmt.Errorf("rand.Read: %w", err)
	}

	return peerID, nil
}
