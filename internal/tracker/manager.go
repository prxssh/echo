package tracker

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"log/slog"
	"math"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"
)

// TODO: add metrics and telemetry

// Config tunes how the tracker Manager announces and scrapes.
type Config struct {
	// NumWant is how many peers we ask a tracker for in each announce.
	// Typical values are 50-200. Too high can flood your peers.
	NumWant int32

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
func DefaultConfig() Config {
	return Config{
		NumWant:            100,
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
	OnPeers func(from string, peers []*Peer)
}

type Identity struct {
	InfoHash   [sha1.Size]byte
	PeerID     [sha1.Size]byte
	Port       uint16
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
}

func NewManager(
	announceURLs []string,
	id Identity,
	cfg *Config,
) *Manager {
	m := &Manager{
		cfg:      DefaultConfig(),
		port:     id.Port,
		infoHash: id.InfoHash,
		peerID:   id.PeerID,
		trackers: make([]Tracker, 0, len(announceURLs)),
	}
	if cfg != nil {
		m.cfg = *cfg
	}

	m.UpdateStats(id.Uploaded, id.Downloaded, id.Left)

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

	return m
}

func (m *Manager) SetOnPeers(cb func(from string, peers []*Peer)) {
	m.OnPeers = cb
}

// UpdateStats atomically updates uploaded, downloaded, and left counters,
// which are included in subsequent announce requests.
func (m *Manager) UpdateStats(uploaded, downloaded, left uint64) {
	m.uploaded.Store(uploaded)
	m.downloaded.Store(downloaded)
	m.left.Store(left)
}

// Start launches per-tracker announce (and optional scrape) loops and blocks
// until the context is canceled or a fatal error occurs. If no trackers are
// available, it returns an error.
func (m *Manager) Start(ctx context.Context) error {
	if len(m.trackers) == 0 {
		slog.Warn(
			"no trackers to start",
			slog.String(
				"infoHash",
				hex.EncodeToString(m.infoHash[:]),
			),
		)
		return errors.New("no tracker to start")
	}

	grp, ctx := errgroup.WithContext(ctx)

	for _, tracker := range m.trackers {
		tracker := tracker

		slog.Debug(
			"announce loop starting",
			slog.String("url", tracker.URL()),
		)
		grp.Go(func() error { return m.runAnnounceLoop(ctx, tracker) })

		if m.cfg.ScrapeEvery > 0 && tracker.SupportsScrape() {
			slog.Debug(
				"scrape loop starting",
				slog.String("url", tracker.URL()),
			)

			grp.Go(
				func() error { return m.runScrapeLoop(ctx, tracker) },
			)
		}
	}

	err := grp.Wait()
	m.closed.Store(true)

	if err != nil && !errors.Is(err, context.Canceled) {
		slog.Error(
			"tracker manager exited with error",
			slog.String("error", err.Error()),
		)
	} else {
		slog.Debug("tracker manager stopped")
	}

	return err
}

func (m *Manager) Stop(ctx context.Context) {
	if m.closed.Load() {
		return
	}

	var wg sync.WaitGroup
	for _, tracker := range m.trackers {
		wg.Go(func() {
			if err := m.sendStopped(ctx, tracker); err != nil {
				slog.Error(
					"failed to send stopped event to tracker",
					slog.String("url", tracker.URL()),
					slog.String("error", err.Error()),
				)
				return
			}

			slog.Debug(
				"stopped event sent",
				slog.String("url", tracker.URL()),
			)
		})
	}
	wg.Wait()
	m.closed.Store(true)
}

func (m *Manager) runAnnounceLoop(ctx context.Context, tracker Tracker) error {
	event := EventStarted
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
			Event:      event,
		}
		if req.Left == 0 && event != EventCompleted {
			event = EventCompleted
			req.Event = EventCompleted
		}

		slog.Debug(
			"announce attempt",
			slog.String("url", tracker.URL()),
			slog.String("event", string(req.Event)),
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
				slog.String("erorr", err.Error()),
			)

			backoff = time.Duration(
				math.Min(
					float64(backoff*2),
					float64(m.cfg.MaxBackoff),
				),
			)
			event = EventNone
			continue
		}

		slog.Debug(
			"announce success",
			slog.String("url", tracker.URL()),
			slog.Any("interval", resp.MinInterval),
			slog.Int("peers", len(resp.Peers)),
		)

		m.emitPeers(tracker.URL(), resp.Peers)

		if resp.Interval > 0 {
			interval = resp.Interval
		}
		next := interval
		if m.cfg.RespectMinInterval && resp.MinInterval > 0 &&
			interval < resp.MinInterval {
			next = resp.MinInterval
		}

		if err := sleepCtx(ctx, next); err != nil {
			_ = m.sendStopped(ctx, tracker)
			return err
		}

		event = EventNone
	}
}

func (m *Manager) runScrapeLoop(ctx context.Context, tracker Tracker) error {
	return errors.New("function not implemented")
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
	if m.OnPeers == nil || len(peers) == 0 {
		return
	}

	snapshot := make([]*Peer, len(peers))
	copy(snapshot, peers)

	go func(cb func(string, []*Peer), from string, ps []*Peer) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error(
					"OnPeers panic recovered",
					slog.Any("recover", r),
				)
			}
		}()

		cb(from, ps)
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
