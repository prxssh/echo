package torrent

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"sync"

	"github.com/prxssh/echo/internal/tracker"
	"golang.org/x/sync/errgroup"
)

type Torrent struct {
	Metainfo   *Metainfo
	Trackers   []tracker.Tracker
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
}

func ReadFile(path string) (*Torrent, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	metainfo, err := New(bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}

	trackers, err := initializeTrackers(metainfo.AnnounceURLs)
	if err != nil {
		return nil, err
	}

	return &Torrent{
		Metainfo: metainfo,
		Trackers: trackers,
		Left:     metainfo.Size,
	}, nil
}

// Start a goroutine for announce that
// Start a goroutine for scrape (if supported)
func (t *Torrent) Start() {
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
					slog.Any("error", err),
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
