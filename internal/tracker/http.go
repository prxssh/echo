package tracker

import (
	"context"
	"net/http"
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
	return nil, nil
}
