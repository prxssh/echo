package torrent

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/prxssh/echo/internal/tracker"
)

// peerState Keeps track of whether or not client is interested in remote peer,
// and if it has the remote peer choked or unchoked. The initial state starts
// off with:
// - AmChoking: true
// - AmInterested: false
// - PeerChoking: true
// - PeerInterested: false
// FIXME (@prxssh): We could have nicely packed this information inside an
// integer using bit manipulation
type peerState struct {
	amChoking      bool
	amInterested   bool
	peerChoking    bool
	peerInterested bool
}

// PeerConn encapsulates the complete information associated with a remote peer
// and the connection object.
type PeerConn struct {
	ID    string
	conn  net.Conn
	state *peerState
}

type ConnectRemotePeerOpts struct {
	InfoHash [20]byte
	ClientID [20]byte
}

const (
	handshakeTimeout = 5 * time.Second
	protocolName     = "BitTorrent Protocol"
)

func (pc *PeerConn) Handshake(opts *ConnectRemotePeerOpts) error {
	pc.conn.SetDeadline(time.Now().Add(handshakeTimeout))
	defer pc.conn.SetDeadline(time.Time{})

	if err := pc.sendHandshake(opts.InfoHash, opts.ClientID); err != nil {
		return fmt.Errorf("send handshake: %w", err)
	}

	if err := pc.verifyHandshake(opts.InfoHash); err != nil {
	}

	return nil
}

// sendHandshake writes the Bittorrent Handshake to the remote peer.
// <pstrlen><pstr><reserved><info_hash><peer_id>
func (pc *PeerConn) sendHandshake(infoHash [20]byte, clientPeerID [20]byte) error {
	buf := bytes.NewBuffer(nil)

	buf.WriteByte(byte(len(protocolName)))
	buf.WriteString(protocolName)
	buf.Write(make([]byte, 8))
	buf.Write(infoHash[:])
	buf.Write(clientPeerID[:])

	_, err := pc.conn.Write(buf.Bytes())
	return err
}

// verifyHandshake reads back the handshake and checks the info_hash and peer_id
func (pc *PeerConn) verifyHandshake(expectedInfoHash [20]byte) error {
	var pstrlen [1]byte
	if _, err := io.ReadFull(pc.conn, pstrlen[:]); err != nil {
		return err
	}
	if int(pstrlen[0]) != len(protocolName) {
		return errors.New("unexpected protocol name length")
	}

	total := int(pstrlen[0]) + 8 + 20 + 20
	buf := make([]byte, total)
	if _, err := io.ReadFull(pc.conn, buf); err != nil {
		return err
	}

	pstr := string(buf[:pstrlen[0]])
	if pstr != protocolName {
		return fmt.Errorf("protocol mismatch: got %q", pstr)
	}

	infoOffset := pstrlen[0] + 8
	if !bytes.Equal(buf[infoOffset:infoOffset+20], expectedInfoHash[:]) {
		return errors.New("handshake info_hash mismatch")
	}

	peerOffset := infoOffset + 20
	if !bytes.Equal(buf[peerOffset:peerOffset+20], []byte(pc.ID)) {
		return errors.New("peer_id mismatch")
	}

	return nil
}

// ConnectRemotePeer dials the peer, performs the handshake and returns the
// initialized PeerConn.
func ConnectRemotePeer(p *tracker.Peer, opts *ConnectRemotePeerOpts) (*PeerConn, error) {
	addr := net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
	conn, err := net.DialTimeout("tcp", addr, handshakeTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect remote peers: dial %s: %w", addr, err)
	}

	conn.SetDeadline(time.Now().Add(handshakeTimeout))
	pc := &PeerConn{
		ID:   p.ID,
		conn: conn,
		state: &peerState{
			amChoking:      true,
			amInterested:   false,
			peerChoking:    true,
			peerInterested: false,
		},
	}

	if err := pc.Handshake(opts); err != nil {
		pc.conn.Close()
		return nil, err
	}

	return pc, nil
}
