package torrent

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sort"
	"sync"
)

type Client struct {
	ID       string
	mu       sync.Mutex
	sessions map[string]*Session
}

const clientIDPrefix string = "-ECHO-"

func NewClient() *Client {
	return &Client{ID: generateClientID(), sessions: make(map[string]*Session)}
}

func (c *Client) UnmarshalMetainfo(data []byte) (*Metainfo, error) {
	return UnmarshalMetainfo(bytes.NewReader(data))
}

func (c *Client) Add(meta *Metainfo) (*Session, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := string(meta.InfoHash[:])
	if _, ok := c.sessions[key]; ok {
		return nil, fmt.Errorf("duplicate torrent %s", meta.Info.Name)
	}
	session := NewSession(c.ID, meta)
	c.sessions[key] = session

	return session, nil
}

func (c *Client) Sessions() []*Session {
	c.mu.Lock()
	defer c.mu.Unlock()

	out := make([]*Session, 0, len(c.sessions))
	for _, s := range c.sessions {
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Meta.Info.Name < out[j].Meta.Info.Name
	})

	return out
}

func generateClientID() string {
	clientID := make([]byte, 20)

	copy(clientID, []byte(clientIDPrefix))
	if _, err := rand.Read(clientID[len(clientIDPrefix):]); err != nil {
		panic("unable to generate client id: " + err.Error())
	}

	return string(clientID)
}
