package tracker

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"net/url"
	"time"
)

type UDPTrackerClient struct {
	conn            *net.UDPConn
	key             uint32
	connectionID    uint64
	connectionIDTTL time.Time
	isIPV6          bool
	announceURL     string
}

const (
	protocolID      = 0x41727101980
	baseBackoff     = 15 * time.Second
	connectionIDTTL = 60 * time.Second
	maxRetries      = 8
	maxUDPPacket    = 2048
)

const (
	actionConnect uint32 = iota
	actionAnnounce
	actionScrape
	actionError
)

var (
	errActionMismatch        = errors.New("action mismatch")
	errTransactionIDMismatch = errors.New("transaction id mismatch")
)

func NewUDPTrackerClient(u *url.URL) (*UDPTrackerClient, error) {
	addr, err := net.ResolveUDPAddr("udp", u.Host)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return nil, err
	}

	key, err := randU32()
	if err != nil {
		return nil, err
	}
	if key == 0 {
		key = 1
	}

	return &UDPTrackerClient{
		conn:        conn,
		key:         key,
		isIPV6:      addr.IP.To4() == nil,
		announceURL: u.String(),
	}, nil
}

func (c *UDPTrackerClient) Close() error {
	c.conn.Close()
	return nil
}

func (c *UDPTrackerClient) URL() string {
	return c.announceURL
}

func (c *UDPTrackerClient) SupportsScrape() bool {
	return false
}

func (c *UDPTrackerClient) Announce(
	ctx context.Context,
	params *AnnounceParams,
) (*AnnounceResponse, error) {
	deadline, hasDeadline := ctx.Deadline()

	for n := 0; n <= maxRetries; n++ {
		timeout := backoffWindow(deadline, hasDeadline, n)
		if timeout <= 0 {
			return nil, context.DeadlineExceeded
		}
		_ = c.conn.SetDeadline(time.Now().Add(timeout))

		// Refresh connection id if expired.
		if time.Now().After(c.connectionIDTTL) {
			transactionID, err := randU32()
			if err != nil {
				continue
			}
			if err := c.sendConnectPacket(transactionID); err != nil {
				continue
			}
			connectionID, err := c.readConnectPacket(transactionID)
			if err != nil {
				continue
			}
			c.connectionID = connectionID
			c.connectionIDTTL = time.Now().Add(connectionIDTTL)
		}

		transactionID, err := randU32()
		if err != nil {
			continue
		}
		if err := c.sendAnnouncePacket(
			transactionID,
			c.connectionID,
			params,
		); err != nil {
			continue
		}
		resp, err := c.readAnnouncePacket(transactionID)
		if err != nil {
			// On mismatch, force re-connect next attempt.
			if errors.Is(err, errActionMismatch) ||
				errors.Is(err, errTransactionIDMismatch) {
				c.connectionIDTTL = time.Time{}
			}
			continue
		}
		return resp, nil
	}

	return nil, errors.New("announce failed, exhausted all attempts")
}

func (c *UDPTrackerClient) Scrape(
	ctx context.Context,
	params *ScrapeParams,
) (*ScrapeResponse, error) {
	return nil, errors.ErrUnsupported
}

func (c *UDPTrackerClient) sendConnectPacket(transactionID uint32) error {
	var packet [16]byte
	binary.BigEndian.PutUint64(packet[0:8], protocolID)
	binary.BigEndian.PutUint32(packet[8:12], actionConnect)
	binary.BigEndian.PutUint32(packet[12:16], transactionID)

	if _, err := c.conn.Write(packet[:]); err != nil {
		return err
	}

	return nil
}

func (c *UDPTrackerClient) readConnectPacket(
	transactionID uint32,
) (uint64, error) {
	var packet [16]byte

	nread, err := c.conn.Read(packet[:])
	if err != nil {
		return 0, err
	}
	if nread < 16 {
		return 0, errors.New("small packet size")
	}

	action := binary.BigEndian.Uint32(packet[0:4])
	if action == actionError {
		return 0, errors.New(string(packet[8:nread]))
	}
	if action != actionConnect {
		return 0, errActionMismatch
	}
	receivedTransactionID := binary.BigEndian.Uint32(packet[4:8])
	if receivedTransactionID != transactionID {
		return 0, errTransactionIDMismatch
	}

	return binary.BigEndian.Uint64(packet[8:16]), nil
}

func (c *UDPTrackerClient) sendAnnouncePacket(
	transactionID uint32,
	connectionID uint64,
	params *AnnounceParams,
) error {
	var packet [98]byte

	binary.BigEndian.PutUint64(packet[0:8], connectionID)
	binary.BigEndian.PutUint32(packet[8:12], actionAnnounce)
	binary.BigEndian.PutUint32(packet[12:16], transactionID)
	copy(packet[16:36], params.InfoHash[:])
	copy(packet[36:56], params.PeerID[:])
	binary.BigEndian.PutUint64(packet[56:64], params.Downloaded)
	binary.BigEndian.PutUint64(packet[64:72], params.Left)
	binary.BigEndian.PutUint64(packet[72:80], params.Uploaded)
	binary.BigEndian.PutUint32(packet[80:84], uint32(params.Event))
	binary.BigEndian.PutUint32(packet[84:88], 0)
	binary.BigEndian.PutUint32(packet[88:92], c.key)
	binary.BigEndian.PutUint32(packet[92:96], params.NumWant)
	binary.BigEndian.PutUint16(packet[96:98], params.Port)

	if _, err := c.conn.Write(packet[:]); err != nil {
		return err
	}
	return nil
}

func (c *UDPTrackerClient) readAnnouncePacket(
	transactionID uint32,
) (*AnnounceResponse, error) {
	packet := make([]byte, maxUDPPacket)
	nread, err := c.conn.Read(packet)
	if err != nil {
		return nil, err
	}
	if nread < 20 {
		return nil, errors.New("announce resp too short")
	}

	action := binary.BigEndian.Uint32(packet[0:4])
	if action == actionError {
		return nil, errors.New(string(packet[8:nread]))
	}
	if action != actionAnnounce {
		return nil, errActionMismatch
	}
	receivedTransactionID := binary.BigEndian.Uint32(packet[4:8])
	if receivedTransactionID != transactionID {
		return nil, errTransactionIDMismatch
	}
	interval := binary.BigEndian.Uint32(packet[8:12])
	leechers := binary.BigEndian.Uint32(packet[12:16])
	seeders := binary.BigEndian.Uint32(packet[16:20])

	body := packet[20:nread]
	stride := strideIPV4
	if c.isIPV6 {
		stride = strideIPV6
	}
	body = body[:len(body)/stride*stride]

	peers := make([]*Peer, 0, len(body)/stride)
	for i := 0; i+stride <= len(body); i += stride {
		var peer Peer

		if c.isIPV6 {
			peer.IP = net.IP(body[i : i+16])
			peer.Port = binary.BigEndian.Uint16(body[i+16 : i+18])
		} else {
			peer.IP = net.IPv4(body[i], body[i+1], body[i+2], body[i+3])
			peer.Port = binary.BigEndian.Uint16(body[i+4 : i+6])
		}

		peers = append(peers, &peer)
	}

	return &AnnounceResponse{
		// UDP interval is specified in seconds
		Interval: time.Duration(interval) * time.Second,
		Leechers: leechers,
		Seeders:  seeders,
		Peers:    peers,
	}, nil
}

func randU32() (uint32, error) {
	var b [4]byte

	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint32(b[:]), nil
}

func backoffWindow(deadline time.Time, hasDeadline bool, n int) time.Duration {
	d := baseBackoff << n
	if !hasDeadline {
		return d
	}

	remain := time.Until(deadline)
	if remain <= 0 {
		return 0
	}
	if remain < d {
		return remain
	}
	return d
}
