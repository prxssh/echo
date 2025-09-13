package tracker

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prxssh/echo/internal/bencode"
)

// HTTPTrackerClient is a Tracker implementation that speaks the HTTP(S)
// tracker protocol defined in BEP 3 (and commonly used scrape endpoint).
type HTTPTrackerClient struct {
	announceURL *url.URL
	client      *http.Client
}

const (
	// Query parameters
	paramInfoHash   = "info_hash"
	paramPeerID     = "peer_id"
	paramPort       = "port"
	paramUploaded   = "uploaded"
	paramDownloaded = "downloaded"
	paramLeft       = "left"
	paramCompact    = "compact"
	paramNumWant    = "numwant"
	paramKey        = "key"
	paramTrackerID  = "trackerid"
	paramEvent      = "event"

	// Bencode dictionary keys
	keyFailureReason = "failure reason"
	keyWarningMsg    = "warning message"
	keyInterval      = "interval"
	keyMinInterval   = "min interval"
	keyTrackerID     = "tracker id"
	keyComplete      = "complete"
	keyIncomplete    = "incomplete"
	keyPeers         = "peers"
	keyPeerID        = "peer id"
	keyPeerIP        = "ip"
	keyPeerPort      = "port"
)

// newHTTPTrackerClient creates a new HTTP tracker client for the given announce
// URL.
func newHTTPTrackerClient(u *url.URL) (*HTTPTrackerClient, error) {
	return &HTTPTrackerClient{announceURL: u, client: &http.Client{}}, nil
}

func (c *HTTPTrackerClient) URL() string { return c.announceURL.String() }

// Announce sends an announce request and parses the response.
func (c *HTTPTrackerClient) Announce(
	ctx context.Context,
	params *AnnounceParams,
) (*AnnounceResponse, error) {
	reqURL := c.buildAnnounceRequest(params)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf(
			"tracker announce returned non-ok status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	return parseAnnounceResponse(resp.Body)
}

func (c *HTTPTrackerClient) SupportsScrape() bool {
	path := c.announceURL.Path
	lastSlash := strings.LastIndex(path, "/")
	if lastSlash == -1 {
		return false
	}

	return strings.HasPrefix(path[lastSlash+1:], "announce")
}

// Scrape queries the tracker's scrape endpoint for aggregate swarm statistics.
func (c *HTTPTrackerClient) Scrape(
	ctx context.Context,
	params *ScrapeParams,
) (*ScrapeResponse, error) {
	if !c.SupportsScrape() {
		return nil, errors.New("scrape unsupported")
	}

	scrapeURL, err := c.buildScrapeURL(params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		scrapeURL,
		nil,
	)
	if err != nil {
		return nil, err
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, fmt.Errorf(
			"tracker scrape returned status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}

	return parseScrapeResponse(resp.Body)
}

// buildAnnounceRequest creates the full announce URL with query parameters.
func (c *HTTPTrackerClient) buildAnnounceRequest(
	params *AnnounceParams,
) string {
	reqURL := *c.announceURL
	q := reqURL.Query()

	q.Set(paramInfoHash, string(params.InfoHash[:]))
	q.Set(paramPeerID, string(params.PeerID[:]))

	q.Set(paramPort, strconv.Itoa(int(params.Port)))
	q.Set(paramUploaded, strconv.FormatUint(params.Uploaded, 10))
	q.Set(paramDownloaded, strconv.FormatUint(params.Downloaded, 10))
	q.Set(paramLeft, strconv.FormatUint(params.Left, 10))
	q.Set(paramCompact, "1")

	if params.NumWant > 0 {
		q.Set(paramNumWant, strconv.Itoa(int(params.NumWant)))
	}
	if params.Key != 0 {
		q.Set(paramKey, strconv.FormatUint(uint64(params.Key), 10))
	}
	if params.TrackerID != "" {
		q.Set(paramTrackerID, params.TrackerID)
	}
	if params.Event != EventNone {
		q.Set(paramEvent, params.Event.String())
	}

	reqURL.RawQuery = q.Encode()
	return reqURL.String()
}

// parseAnnounceResponse converts a bencoded tracker response into
// AnnounceResponse.
func parseAnnounceResponse(r io.Reader) (*AnnounceResponse, error) {
	raw, err := bencode.NewDecoder(r).Decode()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal tracker response: %w",
			err,
		)
	}
	data, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"unexpected response type, expected dictionary, got %T",
			raw,
		)
	}

	if failure, ok := data[keyFailureReason].(string); ok {
		return nil, fmt.Errorf("tracker error: %s", failure)
	}

	if warning, ok := data[keyWarningMsg].(string); ok {
		slog.Warn("Tracker warning", "message", warning)
	}

	getInt64 := func(key string) (int64, bool) {
		val, ok := data[key]
		if !ok {
			return 0, false
		}
		num, ok := val.(int64)
		return num, ok
	}

	interval, ok := getInt64(keyInterval)
	if !ok {
		return nil, fmt.Errorf(
			"tracker response missing or invalid 'interval'",
		)
	}

	// Parse optional fields.
	minInterval, _ := getInt64(keyMinInterval)
	complete, _ := getInt64(keyComplete)
	incomplete, _ := getInt64(keyIncomplete)
	trackerID, _ := data[keyTrackerID].(string)

	peers, err := parsePeers(data)
	if err != nil {
		return nil, err
	}

	return &AnnounceResponse{
		Peers:       peers,
		TrackerID:   trackerID,
		Interval:    time.Duration(interval) * time.Second,
		Seeders:     uint32(complete),
		Leechers:    uint32(incomplete),
		MinInterval: time.Duration(minInterval) * time.Second,
	}, nil
}

// parsePeers decodes either the compact or non-compact peers formats.
func parsePeers(data map[string]any) ([]*Peer, error) {
	peersData, ok := data[keyPeers]
	if !ok {
		// It's common for trackers to omit the 'peers' key if there are
		// none.
		// Return an empty slice instead of an error.
		return []*Peer{}, nil
	}

	switch peers := peersData.(type) {
	case string:
		return parseCompactPeers([]byte(peers))
	case []byte:
		return parseCompactPeers(peers)
	case []any:
		return parseDictPeers(peers)
	default:
		return nil, fmt.Errorf("invalid 'peers' format: expected string or list, got %T", peersData)
	}
}

// parseCompactPeers parses the compact peer list format (6 bytes per peer).
func parseCompactPeers(peerData []byte) ([]*Peer, error) {
	const peerSize = 6 // 4 bytes for IP, 2 for port.
	if len(peerData)%peerSize != 0 {
		return nil, fmt.Errorf(
			"invalid compact peer len=%d",
			len(peerData),
		)
	}
	numPeers := len(peerData) / peerSize
	peers := make([]*Peer, 0, numPeers)

	for i := 0; i < numPeers; i++ {
		offset := i * peerSize
		peers = append(peers, &Peer{
			IP: net.IP(peerData[offset : offset+4]),
			Port: binary.BigEndian.Uint16(
				peerData[offset+4 : offset+6],
			),
		})
	}

	return peers, nil
}

// parseDictPeers parses the non-compact (dictionary) peer list format.
func parseDictPeers(peerList []any) ([]*Peer, error) {
	peers := make([]*Peer, 0, len(peerList))

	for i, item := range peerList {
		peerDict, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf(
				"invalid peer dictionary entry at index %d: got %T",
				i,
				item,
			)
		}

		ipStr, ok := peerDict[keyPeerIP].(string)
		if !ok {
			return nil, fmt.Errorf(
				"missing or invalid 'ip' in peer entry at index %d",
				i,
			)
		}

		portVal, ok := peerDict[keyPeerPort].(int64)
		if !ok {
			return nil, fmt.Errorf(
				"missing or invalid 'port' in peer entry at index %d",
				i,
			)
		}

		ip := net.ParseIP(ipStr)
		if ip == nil {
			return nil, fmt.Errorf(
				"invalid IP address string '%s' in peer entry at index %d",
				ipStr,
				i,
			)
		}

		peers = append(peers, &Peer{IP: ip, Port: uint16(portVal)})
	}
	return peers, nil
}

// buildScrapeURL returns the scrape URL with repeated info_hash parameters.
// Only trackers whose announce URL ends with a segment containing "announce"
// are considered to support scrape. Otherwise, ErrScrapeNotSupported is
// returned.
func (c *HTTPTrackerClient) buildScrapeURL(
	params *ScrapeParams,
) (string, error) {
	u := *c.announceURL
	path := u.Path

	// idx will never be -1 here
	idx := strings.LastIndex(path, "/")
	u.Path = path[:idx] + strings.Replace(
		path[idx+1:],
		"announce",
		"scrape",
		1,
	)

	q := u.Query()
	for _, h := range params.InfoHashes {
		q.Add(paramInfoHash, string(h[:]))
	}
	u.RawQuery = q.Encode()

	return u.String(), nil
}

// parseScrapeResponse parses the HTTP scrape response into ScrapeResponse.
func parseScrapeResponse(r io.Reader) (*ScrapeResponse, error) {
	decoded, err := bencode.NewDecoder(r).Decode()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to parse scrape response: %w",
			err,
		)
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"unexpected scrape response type: %T",
			decoded,
		)
	}

	files, ok := root["files"].(map[string]any)
	if !ok {
		// Some trackers may return empty stats; treat as empty map.
		return &ScrapeResponse{
			Stats: map[[sha1.Size]byte]ScrapeStats{},
		}, nil
	}

	out := make(map[[sha1.Size]byte]ScrapeStats, len(files))
	for k, v := range files {
		fdict, ok := v.(map[string]any)
		if !ok {
			continue
		}
		var ih [sha1.Size]byte
		kb := []byte(k)
		if len(kb) != sha1.Size {
			// Skip invalid keys
			continue
		}
		copy(ih[:], kb)

		var s ScrapeStats
		if n, ok := fdict["complete"].(int64); ok && n >= 0 {
			s.Seeders = uint32(n)
		}
		if n, ok := fdict["incomplete"].(int64); ok && n >= 0 {
			s.Leechers = uint32(n)
		}
		if n, ok := fdict["downloaded"].(int64); ok && n >= 0 {
			s.Completed = uint32(n)
		}
		if name, ok := fdict["name"].(string); ok {
			s.Name = name
		}
		out[ih] = s
	}
	return &ScrapeResponse{Stats: out}, nil
}
