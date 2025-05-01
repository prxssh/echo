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
	Announce(ctx context.Context, params AnnounceParams) (*AnnounceResponse, error)
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
	InfoHash   [20]byte // SHA1 hash of the info key
	PeerID     [20]byte // Echo client PeerID
	Port       uint16   // Port on which we're listening for connections
	Uploaded   int64    // data that has been seeded so far
	Downloaded int64    // data that has been downloaded so far
	Left       int64    // data left to download
	Event      Event    // current event (started/completed/stopped)
}

// AnnounceResponse is what tracker returns on announce
type AnnounceResponse struct {
	TrackerID   string  // unique identifier for the tracker
	Interval    uint32  // seconds until next announce
	Leechers    uint32  // clients downloading this torrent
	Seeders     uint32  // clients uploading this torrent
	Peers       []*Peer // active peers
	MinInterval uint32  // interval after which we should call the tracker
}

// Peer is one peer endpoint from the tracker
type Peer struct {
	ID   string // identifier for this peer (absent in compact mode)
	IP   net.IP // ip-address of this peer
	Port uint16 // port on which this peer is listenting to connections
}

func NewTrackerClient(announce string) (ITrackerProtocol, error) {
	u, err := url.Parse(announce)
	if err != nil {
		return nil, fmt.Errorf("tracker: invalid announce %q: %w", announce, err)
	}

	switch u.Scheme {
	case "http", "https":
		return NewHTTPTrackerClient(announce)
	case "udp":
		return NewUDPTrackerClient(u.Host)
	default:
		return nil, fmt.Errorf("tracker: unsupported tracker protocol %q", u.Scheme)
	}
}
