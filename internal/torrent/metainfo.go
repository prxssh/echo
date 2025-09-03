package torrent

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/prxssh/echo/internal/bencode"
)

// Metainfo describes the contents of a .torrent file (BEP 3).
//
// It mirrors the top-level fields found in the bencoded metainfo, with
// types chosen for convenient use by the rest of the codebase.
type Metainfo struct {
	// Info: the "info" dictionary that describes the payload. Its exact
	// shape differs between single-file and multi-file torrents and is used
	// to compute the infohash.
	Info *Info

	// AnnounceURLs contains all tracker announce URLs discovered from
	// "announce" and/or "announce-list" fields, in order of tiers if
	// applicable.
	AnnounceURLs []string

	// CreationDate is the optional creation timestamp of the torrent, if
	// present in the metainfo. If absent, it is the zero time.
	CreationDate time.Time

	// Comment is an optional, free-form human-readable note set by the
	// creator.
	Comment string

	// Encoding is the optional character encoding for string fields when
	// not UTF-8 (rare; most modern torrents are UTF-8). It corresponds to
	// the top-level "encoding" key in the metainfo.
	Encoding string

	// Mode indicates whether the torrent is in single-file or multi-file
	// mode. It is typically derived from the presence of the Info.Files
	// field.
	Mode FileMode

	// Size is the total size of the payload in bytes. For multi-file
	// torrents this is the sum of all file lengths; for single-file
	// torrents it is the length of the single file.
	Size uint64
}

// Info is the bencoded "info" dictionary that describes the file(s) and
// piece layout of the torrent.
type Info struct {
	// Hash is the 20-byte SHA-1 of the raw bencoded "info" dictionary (the
	// BitTorrent v1 infohash). This is the identifier used to locate peers
	// for the torrent.
	Hash [sha1.Size]byte

	// Name is the suggested display name. In multi-file mode this is the
	// name of the top-level directory; in single-file mode it is the
	// filename.
	Name string

	// Files lists the files in multi-file mode. It is nil in single-file
	// mode. Each entry contains a relative path as a sequence of path
	// elements.
	Files *[]File

	// PieceLength is the number of bytes per piece. All pieces except the
	// last are this size; the last may be shorter.
	PieceLength uint64

	// Pieces holds the 20-byte SHA-1 hash of each piece, in order. In the
	// .torrent file this data is stored as a single string formed by
	// concatenating the 20-byte hashes.
	Pieces [][sha1.Size]byte

	// Private indicates the BEP 27 "private" flag. When true, clients MUST
	// restrict peer discovery to the trackers in the metainfo (no DHT, PEX,
	// or Local Service Discovery).
	Private bool
}

// File represents a single file entry within a multi-file torrent.
type File struct {
	// Length is the exact size of this file in bytes.
	Length uint64

	// Path is the relative path of the file expressed as path elements. For
	// example, a file "dir1/dir2/file.ext" is represented as
	// []string{"dir1", "dir2", "file.ext"}.
	Path []string
}

// FileMode identifies whether the "info" dictionary is single- or multi-file.
type FileMode string

const (
	// FileModeSingle indicates a single-file torrent.
	FileModeSingle FileMode = "single"
	// FileModeMultiple indicates a multi-file torrent.
	FileModeMultiple FileMode = "multiple"
)

func New(r io.Reader) (*Metainfo, error) {
	p, err := newParser(r)
	if err != nil {
		return nil, err
	}

	return p.parse()
}

type parser struct {
	data map[string]any
}

func newParser(r io.Reader) (*parser, error) {
	decoded, err := bencode.NewDecoder(r).Decode()
	if err != nil {
		return nil, err
	}

	data, ok := decoded.(map[string]any)
	if !ok {
		return nil, errors.New(
			"metainfo: top-level is not a bencoded dictionary",
		)
	}

	return &parser{data: data}, nil
}

func (p *parser) parse() (*Metainfo, error) {
	info, totalSize, err := p.parseInfoDict()
	if err != nil {
		return nil, fmt.Errorf(
			"metainfo: failed to parse info dict: %w",
			err,
		)
	}

	announceURLs, err := p.parseAnnounceURLs()
	if err != nil {
		return nil, err
	}

	creation := p.getInt("creation date")
	comment := p.getString("comment")
	encoding := p.getString("encoding")

	mode := FileModeSingle
	if info.Files != nil {
		mode = FileModeMultiple
	}

	return &Metainfo{
		Info:         info,
		AnnounceURLs: announceURLs,
		CreationDate: time.Unix(creation, 0),
		Comment:      comment,
		Encoding:     encoding,
		Mode:         mode,
		Size:         totalSize,
	}, nil
}

func (p *parser) parseInfoDict() (*Info, uint64, error) {
	raw, ok := p.data["info"].(map[string]any)
	if !ok {
		return nil, 0, errors.New(
			"metainfo: missing or invalid 'info' dictionary",
		)
	}

	hash, err := computeInfoHash(raw)
	if err != nil {
		return nil, 0, err
	}

	pieceLength, err := parsePieceLength(raw)
	if err != nil {
		return nil, 0, err
	}

	pieces, err := parsePieces(raw)
	if err != nil {
		return nil, 0, err
	}

	files, totalSize, err := parseFilesSection(raw)
	if err != nil {
		return nil, 0, err
	}

	name, _ := stringFrom(raw, "name")
	priv := parsePrivateFlag(raw)

	info := &Info{
		Hash:        hash,
		Name:        name,
		Files:       files,
		PieceLength: pieceLength,
		Pieces:      pieces,
		Private:     priv,
	}
	return info, totalSize, nil
}

func (p *parser) parseAnnounceURLs() ([]string, error) {
	urls := make([]string, 0)
	seen := make(map[string]struct{})

	if al, ok := p.data["announce-list"].([]any); ok {
		for _, tier := range al {
			lst, ok := tier.([]any)
			if !ok {
				continue // skip invalid tier shapes
			}

			for _, u := range lst {
				s, ok := u.(string)
				if !ok || s == "" {
					continue
				}

				if _, dup := seen[s]; dup {
					continue
				}

				seen[s] = struct{}{}
				urls = append(urls, s)
			}
		}
	}

	if len(urls) == 0 {
		if a, ok := p.data["announce"].(string); ok && a != "" {
			urls = append(urls, a)
		}
	}

	return urls, nil
}

func computeInfoHash(raw map[string]any) ([sha1.Size]byte, error) {
	var buf bytes.Buffer

	if err := bencode.NewEncoder(&buf).Encode(raw); err != nil {
		return [sha1.Size]byte{}, fmt.Errorf(
			"metainfo: failed to re-encode info for hash: %w",
			err,
		)
	}

	sum := sha1.Sum(buf.Bytes())
	return sum, nil
}

func parsePieceLength(raw map[string]any) (uint64, error) {
	pl, ok := intFrom(raw, "piece length")
	if !ok {
		return 0, errors.New("metainfo: missing 'piece length'")
	}
	if pl <= 0 {
		return 0, errors.New("metainfo: invalid 'piece length'")
	}

	return uint64(pl), nil
}

func parsePieces(raw map[string]any) ([][sha1.Size]byte, error) {
	s, ok := raw["pieces"].(string)
	if !ok {
		return nil, errors.New("metainfo: missing or invalid 'pieces'")
	}

	b := []byte(s)
	if len(b)%sha1.Size != 0 {
		return nil, errors.New(
			"metainfo: 'pieces' length is not a multiple of 20 bytes",
		)
	}

	n := len(b) / sha1.Size
	out := make([][sha1.Size]byte, n)
	for i := 0; i < n; i++ {
		copy(out[i][:], b[i*sha1.Size:(i+1)*sha1.Size])
	}

	return out, nil
}

func parsePrivateFlag(raw map[string]any) bool {
	if v, ok := intFrom(raw, "private"); ok {
		return v == 1
	}
	return false
}

func parseFilesSection(raw map[string]any) (*[]File, uint64, error) {
	if filesAny, ok := raw["files"].([]any); ok {
		return parseMultiFiles(filesAny)
	}

	// Single-file mode
	l, ok := intFrom(raw, "length")
	if !ok || l < 0 {
		return nil, 0, errors.New(
			"metainfo: missing or invalid 'length' for single-file torrent",
		)
	}

	return nil, uint64(l), nil
}

func parseMultiFiles(filesAny []any) (*[]File, uint64, error) {
	flist := make([]File, 0, len(filesAny))
	var total uint64

	for i, fe := range filesAny {
		fdict, ok := fe.(map[string]any)
		if !ok {
			return nil, 0, fmt.Errorf(
				"metainfo: file entry %d is not a dictionary",
				i,
			)
		}

		length, ok := intFrom(fdict, "length")
		if !ok || length < 0 {
			return nil, 0, fmt.Errorf(
				"metainfo: invalid or missing file length at index %d",
				i,
			)
		}

		pathAny, ok := fdict["path"].([]any)
		if !ok || len(pathAny) == 0 {
			return nil, 0, fmt.Errorf(
				"metainfo: invalid or missing file path at index %d",
				i,
			)
		}

		path := make([]string, 0, len(pathAny))
		for j, pe := range pathAny {
			ps, ok := pe.(string)
			if !ok {
				return nil, 0, fmt.Errorf(
					"metainfo: non-string path element at file %d index %d",
					i,
					j,
				)
			}
			path = append(path, ps)
		}

		flist = append(flist, File{Length: uint64(length), Path: path})
		total += uint64(length)
	}
	return &flist, total, nil
}

func stringFrom(m map[string]any, key string) (string, bool) {
	v, ok := m[key].(string)
	return v, ok
}

func intFrom(m map[string]any, key string) (int64, bool) {
	v, ok := m[key].(int64)
	return v, ok
}

func (p *parser) getString(key string) string {
	if val, ok := p.data[key].(string); ok {
		return val
	}
	return ""
}

func (p *parser) getInt(key string) int64 {
	if val, ok := p.data[key].(int64); ok {
		return val
	}
	return 0
}
