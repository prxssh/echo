package peer

import (
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
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     20 * time.Second,
		HandshakeTimeout: 30 * time.Second,
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

	dialWorkersWg sync.WaitGroup
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
		candidatesBuf: make(chan *tracker.Peer, 1000),
		peers:         make(map[string]*Peer),
	}
	if cfg == nil {
		m.cfg = defaultConfig()
	} else {
		m.cfg = *cfg
	}

	return m, nil
}

func (m *Manager) Start() {
	for w := 0; w < m.cfg.DialWorkers; w++ {
		m.dialWorkersWg.Go(func() { m.dialPeers() })
	}
}

func (m *Manager) Stop() {
	select {
	case <-m.done:
	default:
		close(m.done)
	}

	m.dialWorkersWg.Wait()

	m.peerMut.RLock()
	for _, peer := range m.peers {
		peer.Stop()
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

func (m *Manager) dialPeers() {
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
			if !m.admitPeer(trackerPeer.Addr(), peer) {
				peer.Stop()
				continue
			}

			go func(addr string, peer *Peer) {
				peer.Start(m.done)
				m.removePeer(addr)
			}(trackerPeer.Addr(), peer)
		}
	}
}

func (m *Manager) admitPeer(addr string, peer *Peer) bool {
	m.peerMut.Lock()
	defer m.peerMut.Unlock()

	if _, exists := m.peers[addr]; exists {
		return false
	}
	m.peers[addr] = peer

	return true
}

func (m *Manager) removePeer(addr string) {
	m.peerMut.Lock()
	defer m.peerMut.Unlock()

	peer, ok := m.peers[addr]
	if !ok {
		return
	}
	peer.Stop()
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
