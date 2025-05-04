package torrent

type Session struct {
	ID    string
	Meta  *Metainfo
	Peers []*PeerConn
}

func NewSession(clientID string, meta *Metainfo) *Session {
	return &Session{ID: clientID, Meta: meta}
}
