package peer

import (
	"crypto/sha1"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/prxssh/echo/internal/bitfield"
	"github.com/prxssh/echo/internal/tracker"
)

// Peer represents a remote BitTorrent peer connected over TCP.
type Peer struct {
	// Addr is the "host:port" address of the peer.
	Addr string
	// conn is the underlying TCP connection.
	conn net.Conn
	// state holds the local and remote choke/interest flags.
	state *PeerState
	// Bitfield keeps track of the pieces a peer has.
	pieceBF bitfield.Bitfield
}

// PeerState tracks local and remote choke/interest flags for a connection.
type PeerState struct {
	AmChoking      bool
	AmInterested   bool
	PeerChoking    bool
	PeerInterested bool
}

// PeerOpts bundles parameters required to talk to a remote peer.
type PeerOpts struct {
	// InfoHash is the torrent's 20-byte SHA1 info hash.
	InfoHash [sha1.Size]byte
	// PeerID is our 20-byte peer identifier.
	PeerID [sha1.Size]byte
	// Pieces is the number of pieces in the torrent.
	Pieces int
}

// defaultTimeout is the connection and handshake timeout.
const defaultTimeout time.Duration = 5 * time.Second

// ConnectRemotePeers dials all tracker-provided peers concurrently,
// performs the BitTorrent handshake, and returns the successfully
// connected peers. Failed connections are logged and skipped.
func ConnectRemotePeers(
	trackerPeers []*tracker.Peer,
	opts *PeerOpts,
) ([]*Peer, error) {
	var wg sync.WaitGroup
	peerCh := make(chan *Peer, len(trackerPeers))

	for _, trackerPeer := range trackerPeers {
		wg.Go(func() {
			addr := fmt.Sprintf(
				"%s:%d",
				trackerPeer.IP,
				trackerPeer.Port,
			)
			conn, err := net.DialTimeout(
				"tcp",
				addr,
				defaultTimeout,
			)
			if err != nil {
				slog.Error(
					"failed to connect remote peer",
					slog.String("address", addr),
					slog.Any("error", err),
					slog.Any("peer", trackerPeer),
				)
				return
			}

			peer := &Peer{
				Addr:    addr,
				conn:    conn,
				pieceBF: bitfield.New(opts.Pieces),
				state:   initPeerState(),
			}
			if err := peer.performHandshake(opts); err != nil {
				slog.Error(
					"failed to perform handshake",
					slog.Any("error", err),
					slog.Any("peer", trackerPeer),
				)
				return
			}

			go peer.Start()
			peerCh <- peer
		})
	}
	wg.Wait()
	close(peerCh)

	peers := make([]*Peer, len(peerCh))
	for peer := range peerCh {
		peers = append(peers, peer)
	}

	return peers, nil
}

// initPeerState returns the initial state for a new peer connection.
func initPeerState() *PeerState {
	return &PeerState{
		AmChoking:      true,
		AmInterested:   false,
		PeerChoking:    true,
		PeerInterested: false,
	}
}

// Start begins processing messages from the remote peer until the
// connection is closed or an error occurs.
func (p *Peer) Start() {
	slog.Info(
		"remote peer connected, starting process",
		slog.Any("peer", p),
	)

	defer p.conn.Close()
	p.readMessages()
}

// Read pulls the next wire message from the peer connection.
func (p *Peer) Read() (*Message, error) {
	return ReadMessage(p.conn)
}

// performHandshake exchanges the BitTorrent handshake with the remote peer.
func (p *Peer) performHandshake(opts *PeerOpts) error {
	p.conn.SetDeadline(time.Now().Add(defaultTimeout))
	defer p.conn.SetDeadline(time.Time{})

	h := NewHandshake(opts.InfoHash, opts.PeerID)
	return h.Perform(p.conn)
}

// readMessages continuously reads and handles messages from the peer.
func (p *Peer) readMessages() {
	for {
		p.conn.SetReadDeadline(time.Now().Add(2 * time.Minute))

		msg, err := p.Read()
		if err != nil {
			return
		}
		if msg == nil { // keep-alive
			continue
		}

		switch msg.ID {
		case MsgBitfield:
			p.pieceBF = bitfield.FromBytes(msg.Payload)

		case MsgChoke:
			p.state.PeerChoking = true

		case MsgUnchoke:
			p.state.PeerChoking = false

		case MsgInterested:
			p.state.PeerInterested = true

		case MsgNotInterested:
			p.state.PeerInterested = false

		case MsgHave:
			return

		case MsgPiece:
			return

		default:
			slog.Warn(
				"received unknown message from peer",
				slog.Any("message", msg),
				slog.String(
					"peer",
					p.conn.RemoteAddr().String(),
				),
			)
		}
	}
}
