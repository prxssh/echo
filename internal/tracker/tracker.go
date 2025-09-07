package tracker

import (
	"context"
	"crypto/sha1"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"
)

// Tracker abstracts a BitTorrent tracker. Implementations may target
// HTTP/HTTPS, UDP, or other tracker transports. Calls are expected to be
// idempotent and respect the provided context for cancellation/timeouts.
type Tracker interface {
	// URL returns the underlying url associated with this tracker
	URL() string

	// Announce notifies the tracker of the client's state for a single
	// torrent and retrieves peer candidates and tracker directives.
	Announce(
		ctx context.Context,
		params *AnnounceParams,
	) (*AnnounceResponse, error)

	// SupportsScrape returns a boolean for whether the tracker
	// supports scraping or not.
	SupportsScrape() bool

	// Scrape retrieves aggregate swarm statistics for one or more
	// infohashes.
	Scrape(
		ctx context.Context,
		params *ScrapeParams,
	) (*ScrapeResponse, error)
}

// AnnounceParams carries the parameters for an announce request (BEP 3).
type AnnounceParams struct {
	// InfoHash is the 20-byte SHA‑1 (v1) infohash of the torrent.
	InfoHash [sha1.Size]byte

	// PeerID is the client's 20-byte peer identifier.
	PeerID [sha1.Size]byte

	// Port is the TCP port on which the client is listening for incoming
	// peers.
	Port uint16

	// Uploaded is the total uploaded bytes for this torrent.
	Uploaded uint64

	// Downloaded is the total downloaded bytes for this torrent.
	Downloaded uint64

	// Left is the number of bytes left to download.
	Left uint64

	// Event indicates lifecycle transitions as defined by BEP 3.
	Event Event

	// NumWant is the desired number of peers the client would like to
	// receive. Trackers may ignore this or cap the value.
	NumWant int32

	// Key is an optional client-generated key to allow the tracker to match
	// future announces from the same client.
	Key uint32

	// TrackerID is an opaque identifier previously returned by the tracker.
	// If present, it allows the tracker to associate state with the client.
	TrackerID string
}

// AnnounceResponse contains the result of an announce request.
type AnnounceResponse struct {
	// TrackerID is an opaque identifier to be echoed back in future
	// announces.
	TrackerID string

	// Interval is the recommended delay before the next regular announce.
	// Trackers specify this value in seconds; it is represented here as a
	// time.Duration for convenience.
	Interval time.Duration

	// MinInterval, if non-zero, is the minimum allowed delay between
	// announces. Also provided in seconds by trackers; represented as a
	// time.Duration.
	MinInterval time.Duration

	// Leechers is the number of non-seeding peers known to the tracker.
	Leechers uint32

	// Seeders is the number of seeding peers known to the tracker.
	Seeders uint32

	// Peers is the set of peer candidates returned by the tracker.
	// Implementations may populate from compact or non-compact responses.
	Peers []*Peer
}

// Peer describes a peer candidate returned by a tracker.
type Peer struct {
	// IP is the IPv4 or IPv6 address of the peer.
	IP net.IP

	// Port is the TCP port on which the peer accepts incoming connections.
	Port uint16
}

func (p *Peer) String() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

// ScrapeParams carries infohashes to query in a scrape request.
type ScrapeParams struct {
	// AnnounceURLs lists tracker announce endpoints to scrape.
	// Implementations may derive the scrape endpoint from the announce URL
	// (e.g., /announce → /scrape).
	AnnounceURLs []string

	// InfoHashes lists the infohashes to query.
	InfoHashes [][sha1.Size]byte
}

// ScrapeResponse contains swarm statistics keyed by infohash.
type ScrapeResponse struct {
	// Stats maps each infohash to aggregate swarm statistics.
	Stats map[[sha1.Size]byte]ScrapeStats
}

// ScrapeStats describes aggregate swarm counts for a torrent.
type ScrapeStats struct {
	// Seeders is the number of complete peers.
	Seeders uint32

	// Leechers is the number of incomplete peers.
	Leechers uint32

	// Completed is the total number of times the torrent has been
	// completed.
	Completed uint32

	// Name is an optional display name of the torrent, if provided by the
	// tracker.
	Name string
}

type Event uint8

const (
	EventNone Event = iota
	EventStarted
	EventStopped
	EventCompleted
)

func (e Event) String() string {
	switch e {
	case EventNone:
		return ""
	case EventStarted:
		return "started"
	case EventStopped:
		return "stopped"
	default:
		return "completed"
	}
}

func NewTracker(announceURL string) (Tracker, error) {
	url, err := url.Parse(announceURL)
	if err != nil {
		return nil, fmt.Errorf(
			"tracker: invalid announce url %q:%w",
			announceURL,
			err,
		)
	}

	switch url.Scheme {
	case "http", "https":
		return newHTTPTrackerClient(url)
	default:
		return nil, fmt.Errorf(
			"tracker: unsupported schema %q",
			url.Scheme,
		)
	}
}
