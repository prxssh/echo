package torrent

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"strconv"
	"time"

	"github.com/prxssh/echo/internal/bitfield"
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
	ID       string
	conn     net.Conn
	state    *peerState
	piecesBF bitfield.Bitfield
}

const (
	handshakeTimeout  = 5 * time.Second
	keepAliveInterval = 1 * time.Minute
	keepAliveTimeout  = 2 * time.Minute
	protocolName      = "BitTorrent Protocol"
)

type ConnectRemotePeerOpts struct {
	InfoHash  [20]byte
	ClientID  [20]byte
	NumPieces int64
}

// ConnectRemotePeer dials the peer, performs the handshake and returns the
// initialized PeerConn.
func ConnectRemotePeer(p *tracker.Peer, opts *ConnectRemotePeerOpts) (*PeerConn, error) {
	addr := net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
	conn, err := net.DialTimeout("tcp", addr, handshakeTimeout)
	if err != nil {
		return nil, fmt.Errorf("connect remote peers: dial %s: %w", addr, err)
	}

	pc := &PeerConn{
		ID:   p.ID,
		conn: conn,
		state: &peerState{
			amChoking:      true,
			amInterested:   false,
			peerChoking:    true,
			peerInterested: false,
		},
		piecesBF: bitfield.New(opts.NumPieces),
	}

	if err := pc.Handshake(opts); err != nil {
		pc.conn.Close()
		return nil, err
	}

	return pc, nil
}

func (pc *PeerConn) Handshake(opts *ConnectRemotePeerOpts) error {
	pc.conn.SetDeadline(time.Now().Add(handshakeTimeout))
	defer pc.conn.SetDeadline(time.Time{})

	if err := pc.sendHandshake(opts.InfoHash, opts.ClientID); err != nil {
		return fmt.Errorf("send handshake: %w", err)
	}

	if err := pc.verifyHandshake(opts.InfoHash); err != nil {
		return fmt.Errorf("verify handshake: %w", err)
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

// --------- Peer Messages ---------- //

type messageID uint8

const (
	msgKeepAlive    messageID = 255
	msgChoke        messageID = 0
	msgUnchoke      messageID = 1
	msgInterested   messageID = 2
	msgUninterested messageID = 3
	msgHave         messageID = 4
	msgBitfield     messageID = 5
	msgRequest      messageID = 6
	msgPiece        messageID = 7
	msgCancel       messageID = 8
	msgPort         messageID = 9
)

// Message stores ID and payload of a message
type message struct {
	id      messageID
	payload []byte
}

// HandleIncoming runs in a loop, dispatching messages by ID
func (pc *PeerConn) HandleIncoming() error {
	for {
		msg, err := pc.ReadMessage()
		if err != nil {
			return err
		}

		switch msg.id {
		case msgKeepAlive:
			// do nothing

		case msgChoke:
			pc.state.peerChoking = true

		case msgUnchoke:
			pc.state.peerChoking = false

		case msgInterested:
			pc.state.peerInterested = true

		case msgUninterested:
			pc.state.peerInterested = false

		case msgHave:
			// do something

		case msgBitfield:
			// do something

		case msgRequest:
			// do something

		case msgCancel:
			// do something

		case msgPort:
			// port := binary.BigEndian.Uint16(msg.payload)
			// update port somethwere

		default:
			return fmt.Errorf("unknown message id: %d", msg.id)
		}
	}
}

func (pc *PeerConn) ReadMessage() (*message, error) {
	var lenBuf [4]byte
	if _, err := io.ReadFull(pc.conn, lenBuf[:]); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(lenBuf[:])
	if length == 0 {
		return &message{id: msgKeepAlive}, nil
	}

	buf := make([]byte, length)
	if _, err := io.ReadFull(pc.conn, buf); err != nil {
		return nil, err
	}
	return &message{id: messageID(buf[0]), payload: buf[1:]}, nil
}

func (pc *PeerConn) sendRaw(id messageID, payload []byte) error {
	length := uint32(1 + len(payload)) // +1 for id
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], length)

	buf := bytes.NewBuffer(header[:])
	buf.WriteByte(byte(id))
	if len(payload) > 0 {
		buf.Write(payload)
	}

	_, err := pc.conn.Write(buf.Bytes())
	return err
}

func (pc *PeerConn) SendKeepAlive() error {
	_, err := pc.conn.Write([]byte{0, 0, 0, 0})
	return err
}

func (pc *PeerConn) SendChoke() error        { return pc.sendRaw(msgChoke, nil) }
func (pc *PeerConn) SendUnchoke() error      { return pc.sendRaw(msgUnchoke, nil) }
func (pc *PeerConn) SendInterested() error   { return pc.sendRaw(msgInterested, nil) }
func (pc *PeerConn) SendUnInterested() error { return pc.sendRaw(msgUninterested, nil) }

func (pc *PeerConn) SendHave(idx uint32) error {
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, idx)

	return pc.sendRaw(msgHave, buf)
}

func (pc *PeerConn) SendBitfield(bf []byte) error {
	return pc.sendRaw(msgBitfield, bf)
}

func (pc *PeerConn) SendRequest(index, begin, length uint32) error {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], index)
	binary.BigEndian.PutUint32(buf[4:8], begin)
	binary.BigEndian.PutUint32(buf[8:12], length)

	return pc.sendRaw(msgRequest, buf)
}

func (pc *PeerConn) SendPiece(index, begin uint32, block []byte) error {
	buf := bytes.NewBuffer(nil)
	binary.Write(buf, binary.BigEndian, index)
	binary.Write(buf, binary.BigEndian, begin)
	buf.Write(block)

	return pc.sendRaw(msgPiece, buf.Bytes())
}

func (pc *PeerConn) SendCancel(index, begin, length uint32) error {
	buf := make([]byte, 12)
	binary.BigEndian.PutUint32(buf[0:4], index)
	binary.BigEndian.PutUint32(buf[4:8], begin)
	binary.BigEndian.PutUint32(buf[8:12], length)

	return pc.sendRaw(msgCancel, buf)
}

func (pc *PeerConn) SendPort(listenPort uint16) error {
	buf := make([]byte, 2)
	binary.BigEndian.PutUint16(buf, listenPort)

	return pc.sendRaw(msgPort, buf)
}
