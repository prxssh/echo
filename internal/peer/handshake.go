package peer

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"io"
)

// Handshake represents the BitTorrent protocol handshake exchanged between
// peers before any wire messages. It contains the protocol string, the
// torrent infohash, and the peer ID.
type Handshake struct {
	Pstr     string
	InfoHash [sha1.Size]byte
	PeerID   [sha1.Size]byte
}

// szReservedBytes defines the reserved field length in the handshake.
// These bytes are not used here but must be present for compatibility.
const szReservedBytes = 8

// NewHandshake returns a Handshake with the standard protocol string and
// the provided infohash and peer ID.
func NewHandshake(infoHash, peerID [sha1.Size]byte) *Handshake {
	return &Handshake{
		Pstr:     "BitTorrent protocol",
		InfoHash: infoHash,
		PeerID:   peerID,
	}
}

// Serialize encodes the handshake into the wire format:
// <pstrlen><pstr><reserved:8><info_hash:20><peer_id:20>.
func (h *Handshake) Serialize() []byte {
	buf := make([]byte, len(h.Pstr)+49)

	buf[0] = byte(len(h.Pstr))
	offset := 1
	offset += copy(buf[offset:], []byte(h.Pstr))
	offset += copy(buf[offset:], make([]byte, szReservedBytes))
	offset += copy(buf[offset:], h.InfoHash[:])
	offset += copy(buf[offset:], h.PeerID[:])

	return buf
}

// Perform writes this handshake to w and reads the remote peer's handshake
// back, verifying that the infohash and peer ID match.
func (h *Handshake) Perform(w io.ReadWriter) error {
	_, err := w.Write(h.Serialize())
	if err != nil {
		return err
	}

	res, err := readHanshake(w)
	if err != nil {
		return err
	}

	if !bytes.Equal(h.InfoHash[:], res.InfoHash[:]) {
		return errors.New("handshake: info hash mismatch")
	}
	if !bytes.Equal(h.PeerID[:], res.PeerID[:]) {
		return errors.New("handshake: peer id mismatch")
	}

	return nil
}

// readHanshake reads a remote handshake from r and returns its parsed form.
// Expected format: <pstrlen><pstr><reserved:8><info_hash:20><peer_id:20>.
func readHanshake(r io.Reader) (*Handshake, error) {
	sizeBuf := make([]byte, 1)
	_, err := io.ReadFull(r, sizeBuf)
	if err != nil {
		return nil, err
	}

	pstrlen := sizeBuf[0]
	if pstrlen == 0 {
		return nil, errors.New("pstrlen can't be 0")
	}

	handshakeBuf := make([]byte, 48+pstrlen)
	if _, err := io.ReadFull(r, handshakeBuf); err != nil {
		return nil, err
	}

	var infoHash, peerID [sha1.Size]byte

	copy(
		infoHash[:],
		handshakeBuf[pstrlen+szReservedBytes:pstrlen+szReservedBytes+sha1.Size],
	)
	copy(peerID[:], handshakeBuf[pstrlen+szReservedBytes+sha1.Size:])

	return &Handshake{
		Pstr:     string(handshakeBuf[0:pstrlen]),
		InfoHash: infoHash,
		PeerID:   peerID,
	}, nil
}
