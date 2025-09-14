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

type Metainfo struct {
	Info         *Info     `json:"info"`
	AnnounceURLs []string  `json:"announceUrls"`
	CreationDate time.Time `json:"creationDate"`
	Comment      string    `json:"comment"`
	Encoding     string    `json:"encoding"`
	Mode         FileMode  `json:"-"`
	Size         uint64    `json:"size"`
}

type Info struct {
	Hash        [sha1.Size]byte   `json:"infoHash"`
	Name        string            `json:"name"`
	Files       *[]File           `json:"files"`
	PieceLength uint64            `json:"pieceLength"`
	Pieces      [][sha1.Size]byte `json:"pieces"`
	Private     bool              `json:"private"`
}

type File struct {
	Length uint64   `json:"length"`
	Path   []string `json:"path"`
}

type FileMode string

const (
	FileModeSingle   FileMode = "single"
	FileModeMultiple FileMode = "multiple"
)

func ParseMetainfo(r io.Reader) (*Metainfo, error) {
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
