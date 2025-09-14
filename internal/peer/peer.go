package peer

import (
	"encoding/binary"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/prxssh/echo/internal/bitfield"
	"github.com/prxssh/echo/internal/tracker"
)

type Peer struct {
	m *Manager

	conn net.Conn

	amChoking      bool
	amInterested   bool
	peerChoking    bool
	peerInterested bool

	mailbox  chan *Message
	stopped  chan struct{}
	stopOnce sync.Once

	pieceBF bitfield.Bitfield
}

func NewPeer(trackerPeer *tracker.Peer, m *Manager) (*Peer, error) {
	conn, err := net.DialTimeout(
		"tcp",
		trackerPeer.Addr(),
		m.cfg.HandshakeTimeout,
	)
	if err != nil {
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(m.cfg.HandshakeTimeout))
	handshake := NewHandshake(m.infoHash, m.peerID)
	if err := handshake.Perform(conn); err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetReadDeadline(time.Time{})

	return &Peer{
		m:              m,
		conn:           conn,
		amChoking:      true,
		amInterested:   false,
		peerChoking:    true,
		peerInterested: false,
		pieceBF:        bitfield.New(m.pieces),
		mailbox:        make(chan *Message, 128),
	}, nil
}

func (p *Peer) Start(globalDone <-chan struct{}) {
	var wg sync.WaitGroup
	wg.Go(func() { p.readMessages(globalDone) })
	wg.Go(func() { p.writeMessages(globalDone) })

	wg.Wait()
}

func (p *Peer) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopped)
		_ = p.conn.Close()
		close(p.mailbox)
	})
}

func (p *Peer) readMessages(globalDone <-chan struct{}) {
	defer p.Stop()

	for {
		select {
		case <-globalDone:
			return
		case <-p.stopped:
			return
		default:
		}

		message, err := p.readMessage()
		if err != nil {
			slog.Error(
				"peer read error",
				slog.String("error", err.Error()),
				slog.String(
					"addr",
					p.conn.RemoteAddr().String(),
				),
			)
			return
		}
		if message == nil { // keep-alive
			continue
		}

		switch message.ID {
		case MsgChoke:
			p.peerChoking = true
		case MsgUnchoke:
			p.peerChoking = false
		case MsgInterested:
			p.peerInterested = true
		case MsgNotInterested:
			p.peerInterested = false
		case MsgBitfield:
			p.pieceBF = bitfield.FromBytes(message.Payload)
		case MsgHave:
			if len(message.Payload) < 4 {
				slog.Debug("have: short payload")
				break
			}

			idx := binary.BigEndian.Uint32(message.Payload)
			p.pieceBF.Set(int(idx))
		case MsgPiece:
			if len(message.Payload) < 8 {
				slog.Debug("piece: short payload")
				break
			}

			index := binary.BigEndian.Uint32(message.Payload[0:4])
			begin := binary.BigEndian.Uint32(message.Payload[4:8])
			block := message.Payload[8:]

			// TODO: hand off to piece downloader
			_ = index
			_ = begin
			_ = block
		default:
			slog.Warn(
				"unknown message",
				slog.Int("id", int(message.ID)),
				slog.Any("payload", message.Payload),
			)
		}
	}
}

func (p *Peer) writeMessages(globalDone <-chan struct{}) {
	defer p.Stop()

	for {
		select {
		case <-globalDone:
			return
		case <-p.stopped:
			return
		case message, ok := <-p.mailbox:
			if !ok {
				return
			}
			if message == nil {
				continue
			}

			if err := p.writeMessage(message); err != nil {
				slog.Debug(
					"peer write error",
					slog.String("error", err.Error()),
				)
				return
			}
		}
	}
}

func (p *Peer) writeMessage(message *Message) error {
	_ = p.conn.SetWriteDeadline(time.Now().Add(p.m.cfg.WriteTimeout))
	defer p.conn.SetWriteDeadline(time.Time{})

	return WriteMessage(p.conn, message)
}

func (p *Peer) readMessage() (*Message, error) {
	_ = p.conn.SetReadDeadline(time.Now().Add(p.m.cfg.ReadTimeout))
	defer p.conn.SetReadDeadline(time.Time{})

	return ReadMessage(p.conn)
}
