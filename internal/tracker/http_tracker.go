package tracker

import (
	"context"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/prxssh/echo/internal/bencode"
)

type HTTPTrackerClient struct {
	announceURL *url.URL
	client      *http.Client
}

const (
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
)

const (
	keyFailureReason = "failure reason"
	keyWarningMsg    = "warning message"
	keyInterval      = "interval"
	keyMinInterval   = "min interval"
	keyTrackerID     = "tracker id"
	keyComplete      = "complete"
	keyIncomplete    = "incomplete"
	keyPeers         = "peers"
	keyPeers6        = "peers6"
	keyPeerID        = "peer id"
	keyPeerIP        = "ip"
	keyPeerPort      = "port"
)

func newHTTPTrackerClient(u *url.URL) (*HTTPTrackerClient, error) {
	transport := &http.Transport{
		MaxIdleConns:        100,
		IdleConnTimeout:     30 * time.Second,
		DisableCompression:  false,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	return &HTTPTrackerClient{
		announceURL: u,
		client: &http.Client{
			Transport: transport,
			Timeout:   20 * time.Second,
		},
	}, nil
}

func (c *HTTPTrackerClient) URL() string {
	return c.announceURL.String()
}

func (c *HTTPTrackerClient) SupportsScrape() bool {
	seg := path.Base(c.announceURL.Path)
	return strings.Contains(seg, "announce")
}

func (c *HTTPTrackerClient) Announce(
	ctx context.Context,
	params *AnnounceParams,
) (*AnnounceResponse, error) {
	announceURL := c.buildAnnounceURL(params)
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		announceURL,
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
			"tracker announce returned non-ok status %d: %s",
			resp.StatusCode,
			string(bodyBytes),
		)
	}
	return parseAnnounceResponse(resp.Body)
}

func (c *HTTPTrackerClient) Scrape(
	ctx context.Context,
	params *ScrapeParams,
) (*ScrapeResponse, error) {
	if params == nil || len(params.InfoHashes) == 0 {
		return &ScrapeResponse{
			Stats: map[[sha1.Size]byte]ScrapeStats{},
		}, nil
	}
	if !c.SupportsScrape() {
		return nil, fmt.Errorf(
			"scrape unsupported for %q",
			c.announceURL,
		)
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

func (c *HTTPTrackerClient) buildAnnounceURL(
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

func parseAnnounceResponse(r io.Reader) (*AnnounceResponse, error) {
	raw, err := bencode.NewDecoder(r).Decode()
	if err != nil {
		return nil, fmt.Errorf(
			"failed to unmarshal tracker response: %w",
			err,
		)
	}
	announceDict, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf(
			"unexpected response type, expected dictionary, got %T",
			raw,
		)
	}

	if failure, ok := announceDict[keyFailureReason].(string); ok {
		return nil, fmt.Errorf("tracker error: %s", failure)
	}
	if warning, ok := announceDict[keyWarningMsg].(string); ok {
		slog.Warn("tracker warning", "message", warning)
	}

	interval, ok := asInt64(announceDict[keyInterval])
	if !ok {
		return nil, fmt.Errorf("announce: missing 'interval'")
	}
	minInterval, _ := asInt64(announceDict[keyMinInterval])
	complete, _ := asInt64(announceDict[keyComplete])
	incomplete, _ := asInt64(announceDict[keyIncomplete])
	trackerID, _ := announceDict[keyTrackerID].(string)
	peers, err := parsePeersField(announceDict)
	if err != nil {
		return nil, err
	}

	return &AnnounceResponse{
		Peers:       peers,
		TrackerID:   trackerID,
		Seeders:     uint32(complete),
		Leechers:    uint32(incomplete),
		Interval:    time.Duration(interval) * time.Second,
		MinInterval: time.Duration(minInterval) * time.Second,
	}, nil
}

func parsePeersField(d map[string]any) ([]*Peer, error) {
	var out []*Peer

	if v, ok := d[keyPeers]; ok {
		ps, err := parsePeersAny(v, false)
		if err != nil {
			return nil, fmt.Errorf("parse peers: %w", err)
		}
		out = append(out, ps...)
	}
	if v6, ok := d[keyPeers6]; ok {
		ps, err := parsePeersAny(v6, true)
		if err != nil {
			return nil, fmt.Errorf("parse peers6: %w", err)
		}
		out = append(out, ps...)
	}

	return out, nil
}

func parsePeersAny(v any, ipv6 bool) ([]*Peer, error) {
	switch t := v.(type) {
	case string:
		return parseCompactPeers([]byte(t), ipv6)
	case []byte:
		return parseCompactPeers(t, ipv6)
	case []any:
		return parseDictPeers(t)
	default:
		return nil, fmt.Errorf("invalid peers type %T", v)
	}
}

func parseCompactPeers(b []byte, ipv6 bool) ([]*Peer, error) {
	stride := strideIPV4
	if ipv6 {
		stride = strideIPV6
	}
	if len(b)%stride != 0 {
		b = b[:len(b)/stride*stride]
	}

	n := len(b) / stride
	peers := make([]*Peer, 0, n)
	for i := 0; i < n; i++ {
		var peer *Peer
		offset := i * stride

		if ipv6 {
			peer.IP = net.IP(b[offset : offset+16])
			peer.Port = binary.BigEndian.Uint16(
				b[offset+16 : offset+18],
			)
		} else {
			peer.IP = net.IPv4(b[offset], b[offset+1], b[offset+2], b[offset+3])
			peer.Port = binary.BigEndian.Uint16(b[offset+4 : offset+6])

		}
		peers = append(peers, peer)
	}

	return peers, nil
}

func parseDictPeers(list []any) ([]*Peer, error) {
	peers := make([]*Peer, 0, len(list))
	for i, it := range list {
		m, ok := it.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("peer[%d]: not dict (%T)", i, it)
		}

		var ip net.IP
		if s, ok := asString(m[keyPeerIP]); ok {
			ip = net.ParseIP(s)
		} else if bs, ok := m[keyPeerIP].([]byte); ok {
			ip = net.IP(bs)
		}
		if ip == nil {
			return nil, fmt.Errorf("peer[%d]: invalid ip", i)
		}

		port64, ok := asInt64(m[keyPeerPort])
		if !ok || port64 < 1 || port64 > 65535 {
			return nil, fmt.Errorf("peer[%d]: invalid port", i)
		}

		peers = append(peers, &Peer{IP: ip, Port: uint16(port64)})
	}

	return peers, nil
}

func (c *HTTPTrackerClient) buildScrapeURL(
	params *ScrapeParams,
) (string, error) {
	u := *c.announceURL
	base := path.Base(u.Path)
	dir := path.Dir(u.Path)
	u.Path = path.Join(dir, strings.Replace(base, "announce", "scrape", 1))

	var sb strings.Builder
	for i, h := range params.InfoHashes {
		if i > 0 {
			sb.WriteByte('&')
		}

		sb.WriteString(paramInfoHash)
		sb.WriteByte('=')
		sb.WriteString(string(h[:]))
	}
	u.RawQuery = sb.String()

	return u.String(), nil
}

func parseScrapeResponse(r io.Reader) (*ScrapeResponse, error) {
	decoded, err := bencode.NewDecoder(r).Decode()
	if err != nil {
		return nil, fmt.Errorf("decode scrape: %w", err)
	}
	root, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("scrape: not a dict (%T)", decoded)
	}
	if failure, ok := asString(root[keyFailureReason]); ok &&
		failure != "" {
		return nil, fmt.Errorf("tracker error: %s", failure)
	}

	files, _ := root["files"].(map[string]any)
	out := make(map[[sha1.Size]byte]ScrapeStats, len(files))
	for k, v := range files {
		fdict, ok := v.(map[string]any)
		if !ok {
			continue
		}

		var ih [sha1.Size]byte
		kb := []byte(k)
		if len(kb) != sha1.Size {
			continue
		}
		copy(ih[:], kb)

		var s ScrapeStats
		if n, ok := asInt64(fdict["complete"]); ok && n >= 0 {
			s.Seeders = uint32(n)
		}
		if n, ok := asInt64(fdict["incomplete"]); ok && n >= 0 {
			s.Leechers = uint32(n)
		}
		if n, ok := asInt64(fdict["downloaded"]); ok && n >= 0 {
			s.Completed = uint32(n)
		}
		if name, ok := asString(fdict["name"]); ok {
			s.Name = name
		}
		out[ih] = s
	}

	return &ScrapeResponse{Stats: out}, nil
}

func asString(v any) (string, bool) {
	switch t := v.(type) {
	case string:
		return t, true
	case []byte:
		return string(t), true
	default:
		return "", false
	}
}

func asInt64(v any) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	}
	return 0, false
}
