package tracker

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// ITrackerProtocol defines the standard Tracker operations
type ITrackerProtocol interface {
	// Announce sends the client's state to the tracker and returns the
	// tracker's response (interval, peer list, etc...).
	Announce(
		ctx context.Context,
		params AnnounceParams,
	) (*AnnounceResponse, error)
}

// Event is the announce "event" parameter
type Event string

const (
	EventStarted   Event = "started"
	EventCompleted Event = "completed"
	EventStopped   Event = "stopped"
)

// AnnounceParams holds all the fields the tracker needs
type AnnounceParams struct {
	InfoHash   [20]byte
	PeerID     [20]byte
	Port       uint16
	Uploaded   int64
	Downloaded int64
	Left       int64
	Event      Event
}

// AnnounceResponse is what tracker returns on announce
type AnnounceResponse struct {
	Interval uint32 // seconds until next announce
	Leechers uint32
	Seeders  uint32
	Peers    []*Peer // list of peers to connect to
}

// Peer is one peer endpoint from the tracker
type Peer struct {
	ID   string
	IP   net.IP
	Port uint16
}

func NewTrackerClient(announce string) (ITrackerProtocol, error) {
	u, err := url.Parse(announce)
	if err != nil {
		return nil, fmt.Errorf(
			"tracker: invalid announce %q: %w",
			announce,
			err,
		)
	}

	switch u.Scheme {
	case "http", "https":
		return NewHTTPTrackerClient(announce)
	case "udp":
		return NewUDPTrackerClient(u.Host)
	default:
		return nil, fmt.Errorf(
			"tracker: unsupported tracker protocol %q",
			u.Scheme,
		)
	}
}
