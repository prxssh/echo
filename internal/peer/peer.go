package peer

import (
	"context"
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

	requestsQueue chan *Message
	stopped       chan struct{}
	stopOnce      sync.Once

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
		requestsQueue:  make(chan *Message, 128),
		stopped:        make(chan struct{}),
	}, nil
}

func (p *Peer) Start(ctx context.Context, globalDone <-chan struct{}) {
	p.emitStarted(ctx)

	var wg sync.WaitGroup
	wg.Go(func() { p.readMessages(ctx, globalDone) })
	wg.Go(func() { p.writeMessages(ctx, globalDone) })

	wg.Wait()
}

func (p *Peer) Addr() string {
	return p.conn.RemoteAddr().String()
}

func (p *Peer) Stop(ctx context.Context) {
	p.stopOnce.Do(func() {
		close(p.stopped)
		_ = p.conn.Close()
		close(p.requestsQueue)

		p.emitStopped(ctx)
	})
}

func (p *Peer) readMessages(ctx context.Context, globalDone <-chan struct{}) {
	defer p.Stop(ctx)

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
			if ne, ok := err.(net.Error); ok &&
				ne.Timeout() { // peer is just idle
				slog.Debug(
					"peer idle timeout",
					slog.String("addr", p.Addr()),
				)
				continue
			}

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
			p.emitMessage(ctx, "Keep Alive")
			continue
		}

		p.emitMessage(ctx, message.ID.String())

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
			index, ok := message.ParseHave()
			if !ok {
				continue
			}
			p.pieceBF.Set(int(index))
		case MsgPiece:
			continue
		case MsgRequest:
			continue
		default:
			slog.Warn(
				"unknown message",
				slog.Int("id", int(message.ID)),
				slog.Any("payload", message.Payload),
			)
		}
	}
}

func (p *Peer) writeMessages(ctx context.Context, globalDone <-chan struct{}) {
	defer p.Stop(ctx)

	lastKeepAliveSend := time.Now()
	keepAliveTicker := time.NewTicker(p.m.cfg.KeepAlive)
	defer keepAliveTicker.Stop()

	for {
		select {
		case <-globalDone:
			return
		case <-p.stopped:
			return
		case <-keepAliveTicker.C:
			if time.Since(lastKeepAliveSend) < p.m.cfg.KeepAlive {
				continue
			}

			if err := p.writeMessage(nil); err != nil {
				slog.Debug(
					"keep-alive write error",
					slog.String("addr", p.Addr()),
					slog.String("error", err.Error()),
				)
				return
			}
			lastKeepAliveSend = time.Now()

		case message, ok := <-p.requestsQueue:
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
