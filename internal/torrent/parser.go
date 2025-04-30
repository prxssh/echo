package torrent

import (
	"fmt"
	"io"
	"time"

	"github.com/prxssh/echo/internal/bencode"
)

// File represents a file in a multi-file torrent
type File struct {
	Length  int64    // File size in bytes
	MD5Hash string   // Optional MD5 checksum
	Path    []string // Path components (directories + filename)
}

type Piece [20]byte

// Info holds the "info" dictionary of the torrent, merged for single and multi
// file mode
type Info struct {
	Name     string  // Filename (single-file) or directory name (mult-file)
	Length   int64   // File length (single-file) or zero for multi-file
	Files    []File  // List of files (multi-file) or nil for single-file
	PieceLen int64   // Number of bytes in each piece
	Pieces   []Piece // Concatenated 20-byte SHA1 hashes
	Private  bool    // Whether the torrent is private
}

// Metainfo represents the top-level of .torrent file structure
type Metainfo struct {
	Announce     string     // Tracker URL
	AnnounceList [][]string // Optional list of tracker URLs
	CreationDate time.Time  // Optional creation date (zero value if absent)
	Comment      string     // Optional comment
	CreatedBy    string     // Optional creator identifier
	Encoding     string     // Optional text encoding
	Info         *Info      // Info dictionary
}

// Decode reads bencoded data from `r` and parses it into a `Metainfo` struct
func Decode(r io.Reader) (*Metainfo, error) {
	raw, err := bencode.NewDecoder(r).Decode()
	if err != nil {
		return nil, fmt.Errorf("torrent decode: %w", err)
	}
	dict, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("torrent decode: top-level is not a dict")
	}

	return readMetainfoIntoStruct(dict)
}

func readMetainfoIntoStruct(dict map[string]any) (*Metainfo, error) {
	var (
		meta Metainfo
		err  error
	)

	meta.Announce, err = getString(dict, "announce", true)
	if err != nil {
		return nil, err
	}
	meta.AnnounceList = getAnnounceList(dict)
	meta.CreationDate = getTime(dict, "creation date")
	meta.Comment, _ = getString(dict, "comment", false)
	meta.CreatedBy, _ = getString(dict, "created by", false)
	meta.Encoding, _ = getString(dict, "encoding", false)

	infoRaw, ok := dict["info"]
	if !ok {
		return nil, fmt.Errorf("torrent: missing 'info'")
	}
	infoDict, ok := infoRaw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("torrent: 'info' is not a dict")
	}
	info, err := parseInfo(infoDict)
	if err != nil {
		return nil, fmt.Errorf("torrent: parse info: %w", err)
	}
	meta.Info = info

	return &meta, nil
}

func parseInfo(d map[string]any) (*Info, error) {
	var (
		info Info
		err  error
	)

	if info.Name, err := getString(d, "name"); err != nil {
		return nil, err
	}

	return info, nil
}

// parsePieces converts raw "pieces" into []Piece.
func parsePieces(v any) ([]Piece, error) {
	rs, ok := v.(string)
	if !ok {
		return nil, fmt.Errorf("torrent: pieces not string")
	}
	b := []byte(rs)
	if len(b)%20 != 0 {
		return nil, fmt.Errorf("torrent: pieces length %d not multiple of 20", len(b))
	}
	n := len(b) / 20
	out := make([]Piece, n)
	for i := 0; i < n; i++ {
		copy(out[i][:], b[i*20:(i+1)*20])
	}
	return out, nil
}

// getString extracts a string; required if req=true.
func getString(m map[string]any, key string, req bool) (string, error) {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s, nil
		}
		return "", fmt.Errorf("torrent: '%s' not string", key)
	}
	if req {
		return "", fmt.Errorf("torrent: missing '%s'", key)
	}
	return "", nil
}

// getInt extracts an int64; required if req=true.
func getInt(m map[string]any, key string, req bool) (int64, error) {
	if v, ok := m[key]; ok {
		return toInt(v), nil
	}
	if req {
		return 0, fmt.Errorf("torrent: missing '%s'", key)
	}
	return 0, nil
}

// getTime extracts a time.Time from a unix timestamp; zero if missing.
func getTime(m map[string]any, key string) time.Time {
	if v, ok := m[key]; ok {
		if sec, ok := v.(int64); ok {
			return time.Unix(sec, 0)
		}
	}
	return time.Time{}
}

// getAnnounceList handles optional 'announce-list'.
func getAnnounceList(m map[string]any) [][]string {
	var list [][]string
	if v, ok := m["announce-list"]; ok {
		if outer, ok := v.([]any); ok {
			for _, inner := range outer {
				if arr, ok := inner.([]any); ok {
					var tier []string
					for _, u := range arr {
						if s, ok := u.(string); ok {
							tier = append(tier, s)
						}
					}
					list = append(list, tier)
				}
			}
		}
	}
	return list
}

// toInt converts any numeric Bencode type to int64.
func toInt(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	}
	return 0
}

// toString is shorthand for getString without error.
func toString(m map[string]any, key string) string {
	s, _ := getString(m, key, false)
	return s
}
