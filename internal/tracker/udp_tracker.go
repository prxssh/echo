package tracker

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"time"
)

// UDPTrackerClient is a Tracker implementation that speaks the BitTorrent
// UDP tracker protocol (BEP 15 / UDP-tracker extension by Arvid Norberg).
type UDPTrackerClient struct {
	addr *net.UDPAddr
}

const (
	protoMagic int64 = 0x41727101980

	actionConnect  uint32 = 0
	actionAnnounce uint32 = 1
	actionScrape   uint32 = 2
	actionError    uint32 = 3
)

func newUDPTrackerClient(u *url.URL) (*UDPTrackerClient, error) {
	addr, err := net.ResolveUDPAddr("udp", u.Host)
	if err != nil {
		return nil, err
	}

	return &UDPTrackerClient{addr: addr}, nil
}

func (c *UDPTrackerClient) URL() string { return c.addr.String() }

func (c *UDPTrackerClient) SupportsScrape() bool { return false }

func (c *UDPTrackerClient) Scrape(
	ctx context.Context,
	params *ScrapeParams,
) (*ScrapeResponse, error) {
	return nil, errors.ErrUnsupported
}

func (c *UDPTrackerClient) Announce(
	ctx context.Context,
	params *AnnounceParams,
) (*AnnounceResponse, error) {
	conn, err := net.DialUDP("udp", nil, c.addr)
	if err != nil {
		return nil, fmt.Errorf("tracker: dial udp: %w", err)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
	}

	txn, err := randUint32()
	if err != nil {
		return nil, err
	}
	if err := writeConnect(conn, txn); err != nil {
		return nil, err
	}
	connID, err := readConnectResp(conn, txn)
	if err != nil {
		return nil, err
	}

	if err := writeAnnounce(
		conn,
		connID,
		txn,
		params,
	); err != nil {
		return nil, err
	}
	resp, err := readAnnounceResp(conn, txn)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func writeConnect(w io.Writer, txn uint32) error {
	var buf [16]byte
	binary.BigEndian.PutUint64(buf[0:8], uint64(protoMagic))
	binary.BigEndian.PutUint32(buf[8:12], actionConnect)
	binary.BigEndian.PutUint32(buf[12:16], txn)

	_, err := w.Write(buf[:])
	if err != nil {
		return fmt.Errorf("tracker: udp connect write: %w", err)
	}
	return nil
}

func readConnectResp(r io.Reader, wantTxn uint32) (int64, error) {
	var buf [16]byte
	if _, err := io.ReadFull(r, buf[:]); err != nil {
		return 0, fmt.Errorf("tracker: udp connect read: %w", err)
	}

	action := binary.BigEndian.Uint32(buf[0:4])
	txn := binary.BigEndian.Uint32(buf[4:8])
	if action == actionError {
		return 0, errors.New("tracker: udp error on connect")
	}
	if action != actionConnect || txn != wantTxn {
		return 0, fmt.Errorf(
			"tracker: udp connect mismatch action=%d txn=%d",
			action,
			txn,
		)
	}

	return int64(binary.BigEndian.Uint64(buf[8:16])), nil
}

func writeAnnounce(
	w io.Writer,
	connID int64,
	txn uint32,
	p *AnnounceParams,
) error {
	// Base packet is 98 bytes
	var buf [98]byte
	// header
	binary.BigEndian.PutUint64(buf[0:8], uint64(connID))
	binary.BigEndian.PutUint32(buf[8:12], actionAnnounce)
	binary.BigEndian.PutUint32(buf[12:16], txn)
	// payload
	copy(buf[16:36], p.InfoHash[:])
	copy(buf[36:56], p.PeerID[:])
	binary.BigEndian.PutUint64(buf[56:64], p.Downloaded)
	binary.BigEndian.PutUint64(buf[64:72], p.Left)
	binary.BigEndian.PutUint64(buf[72:80], p.Uploaded)
	binary.BigEndian.PutUint32(buf[80:84], uint32(p.Event))
	// ip = 0 (let tracker infer)
	binary.BigEndian.PutUint32(buf[84:88], 0)

	// key = random
	key, err := randUint32()
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(buf[88:92], key)
	binary.BigEndian.PutUint32(buf[92:96], uint32(p.NumWant))
	binary.BigEndian.PutUint16(buf[96:98], uint16(p.Port))

	_, err = w.Write(buf[:])
	if err != nil {
		return fmt.Errorf("tracker: udp announce write: %w", err)
	}
	return nil
}

func readAnnounceResp(r io.Reader, wantTxn uint32) (*AnnounceResponse, error) {
	// First 20 bytes are fixed header: action, txn, interval, leechers,
	// seeders
	var hdr [20]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf(
			"tracker: udp announce read header: %w",
			err,
		)
	}

	action := binary.BigEndian.Uint32(hdr[0:4])
	txn := binary.BigEndian.Uint32(hdr[4:8])
	if action == actionError {
		return nil, errors.New("tracker: udp error on announce")
	}
	if action != actionAnnounce || txn != wantTxn {
		return nil, fmt.Errorf(
			"tracker: udp announce mismatch action=%d txn=%d",
			action,
			txn,
		)
	}

	interval := time.Duration(
		binary.BigEndian.Uint32(hdr[8:12]),
	) * time.Second
	leechers := binary.BigEndian.Uint32(hdr[12:16])
	seeders := binary.BigEndian.Uint32(hdr[16:20])

	peers := make([]*Peer, 0, 64)
	for {
		var pbuf [6]byte
		if _, err := io.ReadFull(r, pbuf[:]); err != nil {
			if errors.Is(err, io.EOF) ||
				errors.Is(err, io.ErrUnexpectedEOF) {
				break
			}
			// If the socket times out after reading all data,
			// that's fine.
			if nerr, ok := err.(net.Error); ok && nerr.Timeout() {
				break
			}
			return nil, fmt.Errorf(
				"tracker: udp announce read peer: %w",
				err,
			)
		}
		ip := net.IPv4(pbuf[0], pbuf[1], pbuf[2], pbuf[3])
		port := binary.BigEndian.Uint16(pbuf[4:6])
		peers = append(peers, &Peer{IP: ip, Port: port})
	}

	return &AnnounceResponse{
		Interval: interval,
		Leechers: leechers,
		Seeders:  seeders,
		Peers:    peers,
	}, nil
}

func randUint32() (uint32, error) {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, fmt.Errorf("tracker: rand.Read: %w", err)
	}
	return binary.BigEndian.Uint32(b[:]), nil
}
