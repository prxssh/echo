package tracker

import (
	"context"
	"net"
	"time"
)

type UDPTrackerClient struct {
	Addr        *net.UDPAddr
	Timeout     time.Duration
	transaction uint32
}

func NewUDPTrackerClient(addr string) (*UDPTrackerClient, error) {
	a, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	return &UDPTrackerClient{Addr: a, Timeout: 5 * time.Second}, nil
}

func (t *UDPTrackerClient) Announce(
	ctx context.Context,
	params AnnounceParams,
) (*AnnounceResponse, error) {
	return nil, nil
}
