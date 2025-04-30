package torrent

import (
	"fmt"
	"io"

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
	Files    []*File // List of files (multi-file) or nil for single-file
	PieceLen int64   // Number of bytes in each piece
	Pieces   []Piece // Concatenated 20-byte SHA1 hashes
	Private  bool    // Whether the torrent is private
}

// Metainfo represents the top-level of .torrent file structure
type Metainfo struct {
	Announce     string     // Tracker URL
	AnnounceList [][]string // Optional list of tracker URLs
	CreationDate int64      // Optional creation date (zero value if absent)
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

func readMetainfoIntoStruct(meta map[string]any) (*Metainfo, error) {
	announceURL, err := parseString(meta, "announce", true)
	if err != nil {
		return nil, err
	}

	announceList, err := parseAnnounceList(meta)
	if err != nil {
		return nil, err
	}

	creationDate, err := parseInt(meta, "creation date", false)
	if err != nil {
		return nil, err
	}

	createdBy, err := parseString(meta, "created by", false)
	if err != nil {
		return nil, err
	}

	encoding, err := parseString(meta, "encoding", false)
	if err != nil {
		return nil, err
	}

	comment, err := parseString(meta, "comment", false)
	if err != nil {
		return nil, err
	}

	infoRaw, ok := meta["info"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("readMetainfoIntoStruct: info absent")
	}

	info, err := parseInfoDict(infoRaw)
	if err != nil {
		return nil, err
	}

	return &Metainfo{
		Announce:     announceURL,
		AnnounceList: announceList,
		CreationDate: creationDate,
		CreatedBy:    createdBy,
		Encoding:     encoding,
		Comment:      comment,
		Info:         info,
	}, nil
}

func parseInfoDict(d map[string]any) (*Info, error) {
	pieceLen, err := parseInt(d, "piece length", true)
	if err != nil {
		return nil, err
	}

	name, err := parseString(d, "name", true)
	if err != nil {
		return nil, err
	}

	pstr, err := parseString(d, "pieces", true)
	if err != nil {
		return nil, err
	}
	pbytes := []byte(pstr)
	plen := len(pbytes)
	if plen%20 != 0 {
		return nil, fmt.Errorf(
			"torrent decode: invalid pieces length %d; must be multiple of 20",
			plen,
		)
	}
	var pieces []Piece
	for i := 0; i < plen; i += 20 {
		var p Piece

		copy(p[:], pbytes[i:i+20])
		pieces = append(pieces, p)
	}

	length, err := parseInt(d, "length", false)
	if err != nil {
		return nil, err
	}

	files, err := parseFiles(d)
	if err != nil {
		return nil, err
	}

	return &Info{
		Name:     name,
		Pieces:   pieces,
		Length:   length,
		Files:    files,
		PieceLen: pieceLen,
	}, nil
}

func parseAnnounceList(meta map[string]any) ([][]string, error) {
	raw, ok := meta["announce-list"]
	if !ok {
		return nil, nil // optional
	}

	announceListRaw, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("announce-list is not a list")
	}
	var out [][]string
	for _, group := range announceListRaw {
		inner, ok := group.([]any)
		if !ok {
			return nil, fmt.Errorf("announce-list group is not a list")
		}

		var urls []string
		for _, u := range inner {
			s, ok := u.(string)
			if !ok {
				return nil, fmt.Errorf("announce-list URL is not a string")
			}
			urls = append(urls, s)
		}

		out = append(out, urls)
	}

	return out, nil
}

func parseString(m map[string]any, key string, required bool) (string, error) {
	raw, ok := m[key]
	if !ok {
		if required {
			return "", fmt.Errorf("missing required key %q", key)
		}
		return "", nil
	}

	s, ok := raw.(string)
	if !ok {
		if required {
			return "", fmt.Errorf("key %q is not a string", key)
		}
		return "", nil
	}

	return s, nil
}

func parseInt(m map[string]any, key string, required bool) (int64, error) {
	raw, ok := m[key]
	if !ok {
		if required {
			return 0, fmt.Errorf("missing required key %q", key)
		}
		return 0, nil
	}

	s, ok := raw.(int64)
	if !ok {
		if required {
			return 0, fmt.Errorf("key %q is not a string", key)
		}
		return 0, nil
	}

	return s, nil
}

func parseFiles(m map[string]any) ([]*File, error) {
	rawFiles, ok := m["files"]
	if !ok {
		return nil, nil // optional
	}

	list, ok := rawFiles.([]any)
	if !ok {
		return nil, fmt.Errorf("'files' is not a list")
	}

	var files []*File
	for _, entry := range list {
		m, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("file entry is not a dict")
		}

		length, err := parseInt(m, "length", true)
		if err != nil {
			return nil, err
		}

		md5sum, err := parseString(m, "md5sum", false)
		if err != nil {
			return nil, err
		}

		rawPath, ok := m["path"].([]any)
		if !ok {
			return nil, fmt.Errorf("'path' is not alist")
		}
		path := make([]string, len(rawPath))
		for i, p := range rawPath {
			s, ok := p.(string)
			if !ok {
				return nil, fmt.Errorf("path element is not a string")
			}
			path[i] = s
		}

		files = append(
			files,
			&File{Length: length, Path: path, MD5Hash: md5sum},
		)
	}

	return files, nil
}
