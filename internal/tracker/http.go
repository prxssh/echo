package tracker

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/prxssh/echo/internal/bencode"
	utils "github.com/prxssh/echo/pkg"
)

// HTTPTrackerClient is an HTTP-based implementation of ITrackerProtocol
type HTTPTrackerClient struct {
	AnnounceURL string
	client      *http.Client
}

func NewHTTPTrackerClient(announce string) (*HTTPTrackerClient, error) {
	return &HTTPTrackerClient{
		AnnounceURL: announce,
		client:      &http.Client{},
	}, nil
}

func (t *HTTPTrackerClient) Announce(
	ctx context.Context,
	params AnnounceParams,
) (*AnnounceResponse, error) {
	reqParams, err := buildTrackerRequestParams(t.AnnounceURL, &params)
	if err != nil {
		return nil, fmt.Errorf("tracker: build params: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqParams.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("tracker: create request: %w", err)
	}
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("tracker: request failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker: announce returned status %d", resp.StatusCode)
	}

	return parseTrackerResponse(resp.Body)
}

func buildTrackerRequestParams(announceURL string, params *AnnounceParams) (*url.URL, error) {
	u, err := url.Parse(announceURL)
	if err != nil {
		return nil, err
	}

	q := u.Query()
	q.Set("info_hash", string(params.InfoHash[:]))
	q.Set("peer_id", string(params.PeerID[:]))
	q.Set("port", strconv.Itoa(int(params.Port)))
	q.Set("uploaded", strconv.FormatInt(params.Uploaded, 10))
	q.Set("downloaded", strconv.FormatInt(params.Downloaded, 10))
	q.Set("left", strconv.FormatInt(params.Left, 10))
	q.Set("compact", "1")

	if params.Event != "" {
		q.Set("event", string(params.Event))
	}
	u.RawQuery = q.Encode()

	return u, nil
}

func parseTrackerResponse(body io.ReadCloser) (*AnnounceResponse, error) {
	raw, err := bencode.NewDecoder(body).Decode()
	if err != nil {
		return nil, fmt.Errorf("tracker: decode response: %w", err)
	}
	m, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("tracker: unexpected response type %T", raw)
	}

	if fail, ok := m["failure reason"].(string); ok {
		return nil, fmt.Errorf("tracker: failure %s", fail)
	}

	if warn, ok := m["warning message"].(string); ok {
		slog.Warn("tracker warning", slog.String("message", warn))
	}

	interval, err := utils.ParseInt(m, "interval", true)
	if err != nil {
		return nil, fmt.Errorf("tracker: interval error %w", err)
	}

	minInterval, _ := utils.ParseInt(m, "min interval", false)
	trackerID, _ := utils.ParseString(m, "tracker id", false)
	complete, _ := utils.ParseInt(m, "complete", false)
	incomplete, _ := utils.ParseInt(m, "incomplete", false)

	peers, err := parsePeers(m)
	if err != nil {
		return nil, err
	}

	return &AnnounceResponse{
		Peers:       peers,
		TrackerID:   trackerID,
		Interval:    uint32(interval),
		Seeders:     uint32(complete),
		Leechers:    uint32(incomplete),
		MinInterval: uint32(minInterval),
	}, nil
}

func parsePeers(m map[string]any) ([]*Peer, error) {
	switch p := m["peers"].(type) {
	case string:
		return parseCompactPeers([]byte(p))
	case []any:
		return parseDictPeers(p)
	default:
		return nil, fmt.Errorf("tracker: unsupported peers type %T", m["peers"])
	}
}

func parseCompactPeers(data []byte) ([]*Peer, error) {
	const entryLen = 6
	if len(data)%entryLen != 0 {
		return nil, fmt.Errorf("tracker: invalid peers blog length %d", len(data))
	}

	var out []*Peer
	for i := 0; i < len(data); i += entryLen {
		ip := net.IPv4(data[i], data[i+1], data[i+2], data[i+3])
		port := binary.BigEndian.Uint16(data[i+4 : i+6])
		out = append(out, &Peer{IP: ip, Port: port})
	}

	return out, nil
}

func parseDictPeers(list []any) ([]*Peer, error) {
	var out []*Peer
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("tracker: peer entry is %T, expected dict", item)
		}

		ipStr, err := utils.ParseString(m, "ip", true)
		if err != nil {
			return nil, err
		}

		portI, err := utils.ParseInt(m, "port", true)
		if err != nil {
			return nil, err
		}

		peerID, _ := utils.ParseString(m, "peer id", false)

		var ip net.IP
		if strings.Contains(ipStr, ":") || strings.Contains(ipStr, ".") {
			ip = net.ParseIP(ipStr)
		}
		if ip == nil {
			ip = net.IP([]byte(ipStr))
		}

		out = append(out, &Peer{ID: peerID, IP: ip, Port: uint16(portI)})
	}

	return out, nil
}
