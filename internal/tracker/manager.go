package tracker

import (
	"context"
	"crypto/sha1"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"
	"golang.org/x/sync/errgroup"
)

// Config tunes how the tracker Manager announces and scrapes.
type Config struct {
	// NumWant is how many peers we ask a tracker for in each announce.
	// Typical values are 50-200. Too high can flood your peers.
	NumWant uint32

	// ScrapeEvery controls how often we perform a scrape request (if the
	// tracker supports it). 0 disables scrape.
	ScrapeEvery time.Duration

	// AnnounceTimeout is the per-request timeout for announces.
	AnnounceTimeout time.Duration

	// MaxBackoff caps the exponential backoff after repeated announce
	// failures. Ensures we don't backoff forever.
	MaxBackoff time.Duration

	// InitialBackoff is the starting delay after the first error. Backoff
	// doubles on each failure until MaxBackoff.
	InitialBackoff time.Duration

	// FallbackInterval is used if the tracker response omits an interval.
	// Common default is 30 minutes.
	FallbackInterval time.Duration

	// MinInterval is the minimum allowed announce interval. Some trackers
	// require you not to announce more often than this. If >0, we honor the
	// larger of Interval vs MinInterval.
	MinInterval time.Duration

	// JitterFraction adds randomness to all sleeps so we don't sync up with
	// thousands of other clients.
	JitterFraction float64

	// RespectMinInterval, if true, enforces the tracker's min interval
	// field. If false, we may announce sooner (not recommended).
	RespectMinInterval bool

	// StoppedTimeout is the timeout for sending a "stopped" event when
	// shutting down. Should be short (a few seconds).
	StoppedTimeout time.Duration
}

// DefaultConfig returns a conservative set of defaults for tracker
// announcements and scrapes, including timeouts, backoff, and jitter.
func defaultConfig() Config {
	return Config{
		NumWant:            50,
		ScrapeEvery:        0,
		AnnounceTimeout:    12 * time.Second,
		MaxBackoff:         15 * time.Minute,
		InitialBackoff:     10 * time.Second,
		FallbackInterval:   30 * time.Minute,
		MinInterval:        0,
		JitterFraction:     0.10,
		RespectMinInterval: true,
		StoppedTimeout:     5 * time.Second,
	}
}

// Represents the function which is called when peers are received from the
// tracker.
type OnPeersFunc func(peers []*Peer)

// Manager coordinates all trackers for a torrent.
// It runs announce/scrape loops, merges peers, and tracks session stats.
type Manager struct {
	// cfg holds all announce/scrape tuning knobs (timeouts, backoff, etc.).
	cfg Config

	// trackers is the set of tracker clients (HTTP/UDP) this manager
	// drives. Typically populated from the .torrent announce-list tiers.
	trackers []Tracker

	// port is the TCP/UDP listen port we advertise to trackers for incoming
	// peers.
	port uint16

	// infoHash uniquely identifies the torrent (SHA-1 of the info dict).
	infoHash [sha1.Size]byte

	// peerID is this client's unique identifier in the swarm (20 bytes).
	// Sent on every announce/handshake.
	peerID [sha1.Size]byte

	// uploaded/downloaded/left track aggregate stats for this torrent.
	// Values are updated atomically and passed in announces.
	uploaded   atomic.Uint64
	downloaded atomic.Uint64
	left       atomic.Uint64

	// closed signals whether the manager has been stopped.
	// Once true, further announces are suppressed.
	closed atomic.Bool

	// OnPeers is the function that is called when announce returns peers.
	OnPeers OnPeersFunc
}

type Opts struct {
	InfoHash   [sha1.Size]byte
	PeerID     [sha1.Size]byte
	Port       uint16
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Cfg        *Config
	OnPeers    OnPeersFunc
}

func NewManager(announceURLs []string, opts Opts) (*Manager, error) {
	m := &Manager{
		cfg:      defaultConfig(),
		port:     opts.Port,
		infoHash: opts.InfoHash,
		peerID:   opts.PeerID,
		trackers: make([]Tracker, 0, len(announceURLs)),
	}
	if opts.OnPeers == nil {
		return nil, errors.New(
			"expected OnPeers to be a function, but got nil",
		)
	} else {
		m.OnPeers = opts.OnPeers
	}
	if opts.Cfg != nil {
		m.cfg = *opts.Cfg
	}

	m.UpdateStats(opts.Uploaded, opts.Downloaded, opts.Left)

	for _, url := range announceURLs {
		tracker, err := NewTracker(url)
		if err != nil {
			slog.Warn(
				"tracker init failed",
				slog.String("url", url),
				slog.String("error", err.Error()),
			)
			continue
		}

		m.trackers = append(m.trackers, tracker)
		slog.Debug("tracker added", slog.String("url", url))
	}

	return m, nil
}

func (m *Manager) UpdateStats(uploaded, downloaded, left uint64) {
	m.uploaded.Store(uploaded)
	m.downloaded.Store(downloaded)
	m.left.Store(left)
}

func (m *Manager) Start(ctx context.Context) error {
	if len(m.trackers) == 0 {
		return errors.New("no tracker to start")
	}

	grp, ctx := errgroup.WithContext(ctx)

	for _, tracker := range m.trackers {
		tracker := tracker

		grp.Go(func() error { return m.runAnnounceLoop(ctx, tracker) })

		if m.cfg.ScrapeEvery > 0 && tracker.SupportsScrape() {
			grp.Go(
				func() error { return m.runScrapeLoop(ctx, tracker) },
			)
		}
	}
	err := grp.Wait()
	if err != nil {
		slog.Error(
			"tracker manager exited with error",
			slog.String("error", err.Error()),
		)
	}
	m.closed.Store(true)

	return err
}

func (m *Manager) Stop(ctx context.Context) {
	if m.closed.Load() {
		return
	}

	var wg sync.WaitGroup
	for _, tracker := range m.trackers {
		tr := tracker
		wg.Go(func() {
			_ = m.sendStopped(context.Background(), tr)
		})
	}
	wg.Wait()
	m.closed.Store(true)
}

func (m *Manager) runAnnounceLoop(ctx context.Context, tracker Tracker) error {
	startedSent, completedSent := false, false
	interval := m.cfg.FallbackInterval
	backoff := m.cfg.InitialBackoff

	for {
		req := &AnnounceParams{
			InfoHash:   m.infoHash,
			PeerID:     m.peerID,
			Port:       m.port,
			Uploaded:   m.uploaded.Load(),
			Downloaded: m.downloaded.Load(),
			Left:       m.left.Load(),
			NumWant:    m.cfg.NumWant,
		}
		switch {
		case !startedSent:
			req.Event = EventStarted
		case req.Left == 0 && !completedSent:
			req.Event = EventCompleted
		default:
			req.Event = EventNone
		}

		slog.Debug(
			"tracker announce",
			slog.String("url", tracker.URL()),
			slog.String("event", req.Event.String()),
			slog.Int64("numwant", int64(req.NumWant)),
		)

		callCtx, cancel := context.WithTimeout(
			ctx,
			m.cfg.AnnounceTimeout,
		)
		resp, err := tracker.Announce(callCtx, req)
		cancel()
		if err != nil {
			slog.Warn(
				"announce failed",
				slog.String("url", tracker.URL()),
				slog.String("error", err.Error()),
			)

			backoff = time.Duration(
				math.Min(
					float64(backoff*2),
					float64(m.cfg.MaxBackoff),
				),
			)
			if err := sleepCtx(ctx, jitter(m.cfg, backoff)); err != nil {
				_ = m.sendStopped(context.Background(), tracker)
				return err
			}
			continue
		}

		slog.Debug(
			"announce successful",
			slog.String("url", tracker.URL()),
			slog.Any("interval", resp.Interval),
			slog.Any("peers", len(resp.Peers)),
		)

		if req.Event == EventStarted {
			startedSent = true
		}
		if req.Event == EventCompleted {
			completedSent = true
		}

		runtime.EventsEmit(ctx, "tracker:announce", map[string]any{
			"tracker":     tracker.URL(),
			"seeders":     resp.Seeders,
			"leechers":    resp.Leechers,
			"interval":    resp.Interval,
			"minInterval": resp.MinInterval,
			"peersCount":  len(resp.Peers),
		})

		m.emitPeers(tracker.URL(), resp.Peers)
		backoff = m.cfg.InitialBackoff

		if resp.Interval > 0 {
			interval = resp.Interval
		}
		next := interval
		if m.cfg.RespectMinInterval && resp.MinInterval > 0 &&
			next < resp.MinInterval {
			next = resp.MinInterval
		}
		if err := sleepCtx(ctx, jitter(m.cfg, next)); err != nil {
			_ = m.sendStopped(context.Background(), tracker)
			return err
		}
	}
}

func (m *Manager) runScrapeLoop(ctx context.Context, tracker Tracker) error {
	t := time.NewTicker(m.cfg.ScrapeEvery)
	defer t.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.C:
			// TODO: implement scrape
		}
	}
}

func (m *Manager) sendStopped(ctx context.Context, tracker Tracker) error {
	callCtx, cancel := context.WithTimeout(ctx, m.cfg.StoppedTimeout)
	defer cancel()

	_, err := tracker.Announce(callCtx, &AnnounceParams{
		InfoHash:   m.infoHash,
		PeerID:     m.peerID,
		Port:       m.port,
		Uploaded:   m.uploaded.Load(),
		Downloaded: m.downloaded.Load(),
		Left:       m.left.Load(),
		NumWant:    0,
		Event:      EventStopped,
	})
	if err != nil {
		slog.Warn(
			"stopped announce failed",
			slog.String("url", tracker.URL()),
			slog.String("error", err.Error()),
		)
	}

	return nil
}

func (m *Manager) emitPeers(from string, peers []*Peer) {
	if m.OnPeers == nil {
		slog.Warn(
			"emit peers callback undefined",
			slog.String("tracker", from),
		)
		return
	}
	if len(peers) == 0 {
		return
	}

	snapshot := make([]*Peer, len(peers))
	copy(snapshot, peers)

	go func(callback OnPeersFunc, src string, ps []*Peer) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error(
					"OnPeers panic recovered",
					slog.Any("recover", r),
				)
			}
		}()

		callback(ps)
	}(m.OnPeers, from, snapshot)
}

func jitter(cfg Config, d time.Duration) time.Duration {
	if d <= 0 {
		d = cfg.FallbackInterval
	}
	f := cfg.JitterFraction
	lo, hi := float64(d)*(1.0-f), float64(d)*(1.0+f)

	return time.Duration(lo + rand.Float64()*(hi-lo))
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
