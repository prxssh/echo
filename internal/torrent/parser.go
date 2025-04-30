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
	Files    []File  // List of files (multi-file) or nil for single-file
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
	announceURL, err := parseString(meta, "announce")
	if err != nil {
		return nil, err
	}

	announceList, _ := meta["announce-list"].([][]string)
	creationDate, _ := parseInt(meta, "creation date")
	createdBy, _ := parseString(meta, "created by")
	encoding, _ := parseString(meta, "encoding")
	comment, _ := parseString(meta, "comment")

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
	pieceLen, err := parseInt(d, "piece length")
	if err != nil {
		return nil, err
	}

	name, err := parseString(d, "name")
	if err != nil {
		return nil, err
	}

	pstr, err := parseString(d, "pieces")
	if err != nil {
		return nil, err
	}
	pbytes := []byte(pstr)
	plen := len(pbytes)
	if plen%20 != 0 {
		return nil, fmt.Errorf("torrent decode: invalid pieces length %d; must be multiple of 20", plen)
	}
	var pieces []Piece
	for i := 0; i < plen; i += 20 {
		var p Piece

		copy(p[:], pbytes[i:i+20])
		pieces = append(pieces, p)
	}

	length, _ := parseInt(d, "length")
	files, _ := parseFiles(d)

	return &Info{
		Name:     name,
		Pieces:   pieces,
		Length:   length,
		Files:    files,
		PieceLen: pieceLen,
	}, nil
}

func parseString(m map[string]any, key string) (string, error) {
	raw, ok := m[key]
	if !ok {
		return "", fmt.Errorf("missing required key %q", key)
	}

	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("key %q is not a string", key)
	}

	return s, nil
}

func parseInt(m map[string]any, key string) (int64, error) {
	raw, ok := m[key]
	if !ok {
		return 0, fmt.Errorf("missing required key %q", key)
	}

	s, ok := raw.(int64)
	if !ok {
		return 0, fmt.Errorf("key %q is not a string", key)
	}

	return s, nil
}

func parseFiles(m map[string]any) ([]File, error) {
	rawFiles, ok := m["files"]
	if !ok {
		return nil, fmt.Errorf("'files' is not present")
	}

	list, ok := rawFiles.([]any)
	if !ok {
		return nil, fmt.Errorf("'files' is not a list")
	}

	var files []File
	for _, entry := range list {
		m, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("file entry is not a dict")
		}

		length, err := parseInt(m, "length")
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
		files = append(files, File{Length: length, Path: path})
	}

	return files, nil
}
