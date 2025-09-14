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

type Tracker interface {
	URL() string

	SupportsScrape() bool

	Announce(
		ctx context.Context,
		params *AnnounceParams,
	) (*AnnounceResponse, error)

	Scrape(
		ctx context.Context,
		params *ScrapeParams,
	) (*ScrapeResponse, error)
}

type AnnounceParams struct {
	InfoHash   [sha1.Size]byte
	PeerID     [sha1.Size]byte
	Port       uint16
	Uploaded   uint64
	Downloaded uint64
	Left       uint64
	Event      Event
	NumWant    uint32
	Key        uint32
	TrackerID  string
}

type AnnounceResponse struct {
	TrackerID   string        `json:"-"`
	Interval    time.Duration `json:"interval"`
	MinInterval time.Duration `json:"minInterval"`
	Leechers    uint32        `json:"leechers"`
	Seeders     uint32        `json:"seeders"`
	Peers       []*Peer       `json:"peers"`
}

type Peer struct {
	IP   net.IP `json:"ip"`
	Port uint16 `json:"port"`
}

func (p *Peer) Addr() string {
	return net.JoinHostPort(p.IP.String(), strconv.Itoa(int(p.Port)))
}

type ScrapeParams struct {
	AnnounceURLs []string
	InfoHashes   [][sha1.Size]byte
}

type ScrapeResponse struct {
	Stats map[[sha1.Size]byte]ScrapeStats
}

type ScrapeStats struct {
	Seeders   uint32
	Leechers  uint32
	Completed uint32
	Name      string
}

type Event uint32

const (
	EventNone Event = iota
	EventStarted
	EventStopped
	EventCompleted
)

const (
	strideIPV4 = 6
	strideIPV6 = 18
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
		return NewHTTPTrackerClient(url)
	case "udp":
		return NewUDPTrackerClient(url)
	default:
		return nil, fmt.Errorf(
			"tracker: unsupported schema %q",
			url.Scheme,
		)
	}
}
