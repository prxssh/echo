package peer

import (
	"context"
	"crypto/sha1"
	"sync"
	"time"

	"github.com/prxssh/echo/internal/tracker"
)

type Config struct {
	MaxPeers         uint32
	DialWorkers      int
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	HandshakeTimeout time.Duration
	KeepAlive        time.Duration
}

func defaultConfig() Config {
	return Config{
		MaxPeers:         100,
		DialWorkers:      50,
		ReadTimeout:      2 * time.Minute,
		WriteTimeout:     30 * time.Second,
		HandshakeTimeout: 1 * time.Second,
		KeepAlive:        30 * time.Second,
	}
}

type Manager struct {
	infoHash [sha1.Size]byte
	peerID   [sha1.Size]byte
	pieces   int
	cfg      Config

	candidatesBuf chan *tracker.Peer
	done          chan struct{}

	peerMut sync.RWMutex
	peers   map[string]*Peer

	dialWorkers sync.WaitGroup
}

func NewManager(
	infoHash, peerID [sha1.Size]byte,
	pieces int,
	cfg *Config,
) (*Manager, error) {
	m := &Manager{
		infoHash:      infoHash,
		peerID:        peerID,
		pieces:        pieces,
		done:          make(chan struct{}),
		candidatesBuf: make(chan *tracker.Peer, 1001),
		peers:         make(map[string]*Peer),
	}
	if cfg == nil {
		m.cfg = defaultConfig()
	} else {
		m.cfg = *cfg
	}

	return m, nil
}

func (m *Manager) Start(ctx context.Context) {
	for w := 0; w < m.cfg.DialWorkers; w++ {
		m.dialWorkers.Go(func() { m.dialPeers(ctx) })
	}
}

func (m *Manager) Stop(ctx context.Context) {
	select {
	case <-m.done:
	default:
		close(m.done)
	}
	m.dialWorkers.Wait()

	m.peerMut.RLock()
	for _, peer := range m.peers {
		peer.Stop(ctx)
	}
	m.peerMut.RUnlock()
}

func (m *Manager) Enqueue(trackerPeers []*tracker.Peer) {
	for _, trackerPeer := range trackerPeers {
		if m.hasPeer(trackerPeer.Addr()) {
			continue
		}

		select {
		case <-m.done:
			return
		case m.candidatesBuf <- trackerPeer:
		default: // queue full, drop
		}
	}
}

func (m *Manager) dialPeers(ctx context.Context) {
	for {
		select {
		case <-m.done:
			return
		case trackerPeer, ok := <-m.candidatesBuf:
			if !ok {
				continue
			}
			if m.countPeers() >= int(m.cfg.MaxPeers) {
				continue
			}

			peer, err := NewPeer(trackerPeer, m)
			if err != nil {
				continue
			}
			if !m.admitPeer(peer) {
				peer.Stop(ctx)
				continue
			}

			go func(ctx context.Context, peer *Peer) {
				peer.Start(ctx, m.done)
				m.removePeer(ctx, peer.Addr())
			}(ctx, peer)
		}
	}
}

func (m *Manager) admitPeer(peer *Peer) bool {
	m.peerMut.Lock()
	defer m.peerMut.Unlock()

	addr := peer.Addr()
	if _, exists := m.peers[addr]; exists {
		return false
	}
	m.peers[addr] = peer

	return true
}

func (m *Manager) removePeer(ctx context.Context, addr string) {
	m.peerMut.Lock()
	defer m.peerMut.Unlock()

	peer, ok := m.peers[addr]
	if !ok {
		return
	}
	peer.Stop(ctx)
}

func (m *Manager) hasPeer(addr string) bool {
	m.peerMut.RLock()
	_, ok := m.peers[addr]
	m.peerMut.RUnlock()

	return ok
}

func (m *Manager) countPeers() int {
	m.peerMut.RLock()
	n := len(m.peers)
	m.peerMut.RUnlock()

	return n
}
