package torrent

import (
	"bytes"
	"crypto/sha1"
	"reflect"
	"testing"

	"github.com/prxssh/echo/internal/bencode"
)

func buildSingleFileMeta(
	t *testing.T,
	withPrivate bool,
) ([]byte, map[string]any) {
	t.Helper()
	pieces := append(
		bytes.Repeat([]byte{'A'}, 20),
		bytes.Repeat([]byte{'B'}, 20)...)

	info := map[string]any{
		"name":         "file.bin",
		"piece length": int64(16384),
		"pieces":       string(pieces),
		"length":       int64(12345),
	}
	if withPrivate {
		info["private"] = int64(1)
	}

	top := map[string]any{
		"info":          info,
		"announce":      "http://tracker/announce",
		"creation date": int64(1700000000),
		"comment":       "test torrent",
		"encoding":      "UTF-8",
	}

	var buf bytes.Buffer
	if err := bencode.NewEncoder(&buf).Encode(top); err != nil {
		t.Fatalf("failed to encode metainfo: %v", err)
	}
	return buf.Bytes(), info
}

func buildMultiFileMeta(t *testing.T) ([]byte, map[string]any) {
	t.Helper()

	pieces := append(
		append(
			bytes.Repeat([]byte{'X'}, 20),
			bytes.Repeat([]byte{'Y'}, 20)...),
		bytes.Repeat([]byte{'Z'}, 20)...)

	files := []any{
		map[string]any{
			"length": int64(100),
			"path":   []any{"a.txt"},
		},
		map[string]any{
			"length": int64(200),
			"path":   []any{"sub", "b.dat"},
		},
	}

	info := map[string]any{
		"name":         "my-dir",
		"piece length": int64(32768),
		"pieces":       string(pieces),
		"files":        files,
		"private":      int64(1),
	}

	announceList := []any{
		[]any{"http://t1/a", "http://t1/b"},
		[]any{
			"http://t2/a",
			"http://t1/a",
		},
	}

	top := map[string]any{
		"info":          info,
		"announce-list": announceList,
	}

	var buf bytes.Buffer
	if err := bencode.NewEncoder(&buf).Encode(top); err != nil {
		t.Fatalf("failed to encode metainfo: %v", err)
	}
	return buf.Bytes(), info
}

func TestNew_SingleFile(t *testing.T) {
	data, infoDict := buildSingleFileMeta(t, false)
	m, err := ParseMetainfo(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if m == nil || m.Info == nil {
		t.Fatalf("expected non-nil Metainfo and Info")
	}

	// Expect fallback to single announce
	if want := []string{"http://tracker/announce"}; !reflect.DeepEqual(
		m.AnnounceURLs,
		want,
	) {
		t.Fatalf("AnnounceURLs = %v; want %v", m.AnnounceURLs, want)
	}

	if got, want := m.Comment, "test torrent"; got != want {
		t.Fatalf("Comment = %q; want %q", got, want)
	}
	if got, want := m.Encoding, "UTF-8"; got != want {
		t.Fatalf("Encoding = %q; want %q", got, want)
	}
	if got, want := m.CreationDate.Unix(), int64(1700000000); got != want {
		t.Fatalf("CreationDate = %d; want %d", got, want)
	}
	if m.Mode != FileModeSingle {
		t.Fatalf("Mode = %q; want %q", m.Mode, FileModeSingle)
	}
	if got, want := m.Size, uint64(12345); got != want {
		t.Fatalf("Size = %d; want %d", got, want)
	}

	// Info fields
	if got, want := m.Info.Name, "file.bin"; got != want {
		t.Fatalf("Info.Name = %q; want %q", got, want)
	}
	if m.Info.Files != nil {
		t.Fatalf("Info.Files should be nil for single-file torrents")
	}
	if got, want := m.Info.PieceLength, uint64(16384); got != want {
		t.Fatalf("PieceLength = %d; want %d", got, want)
	}
	if got := len(m.Info.Pieces); got != 2 { // 40 bytes / 20
		t.Fatalf("len(Pieces) = %d; want 2", got)
	}
	if m.Info.Private {
		t.Fatalf("Private = true; want false (absent)")
	}

	// Info hash is SHA-1 of re-encoded info dict
	var ibuf bytes.Buffer
	if err := bencode.NewEncoder(&ibuf).Encode(infoDict); err != nil {
		t.Fatalf("encode infoDict: %v", err)
	}

	wantHash := sha1.Sum(ibuf.Bytes())
	if m.Info.Hash != wantHash {
		t.Fatalf(
			"Info.Hash mismatch: got %x; want %x",
			m.Info.Hash,
			wantHash,
		)
	}
}

func TestNew_MultiFile(t *testing.T) {
	data, infoDict := buildMultiFileMeta(t)
	m, err := ParseMetainfo(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if m.Mode != FileModeMultiple {
		t.Fatalf("Mode = %q; want %q", m.Mode, FileModeMultiple)
	}
	if m.Info == nil || m.Info.Files == nil {
		t.Fatalf("expected Info.Files for multi-file torrent")
	}

	files := *m.Info.Files
	if len(files) != 2 {
		t.Fatalf("len(Files) = %d; want 2", len(files))
	}
	if files[0].Length != 100 ||
		!reflect.DeepEqual(files[0].Path, []string{"a.txt"}) {
		t.Fatalf(
			"file[0] = %+v; want Length=100 Path=[a.txt]",
			files[0],
		)
	}
	if files[1].Length != 200 ||
		!reflect.DeepEqual(files[1].Path, []string{"sub", "b.dat"}) {
		t.Fatalf(
			"file[1] = %+v; want Length=200 Path=[sub b.dat]",
			files[1],
		)
	}

	// Size is the sum of file lengths
	if got, want := m.Size, uint64(300); got != want {
		t.Fatalf("Size = %d; want %d", got, want)
	}

	// Private set via info.private=1
	if !m.Info.Private {
		t.Fatalf("Private = false; want true")
	}

	// Announce-list flattened and de-duplicated, preserving order
	wantURLs := []string{"http://t1/a", "http://t1/b", "http://t2/a"}
	if !reflect.DeepEqual(m.AnnounceURLs, wantURLs) {
		t.Fatalf("AnnounceURLs = %v; want %v", m.AnnounceURLs, wantURLs)
	}

	// Info hash check
	var ibuf bytes.Buffer
	if err := bencode.NewEncoder(&ibuf).Encode(infoDict); err != nil {
		t.Fatalf("encode infoDict: %v", err)
	}

	wantHash := sha1.Sum(ibuf.Bytes())
	if m.Info.Hash != wantHash {
		t.Fatalf(
			"Info.Hash mismatch: got %x; want %x",
			m.Info.Hash,
			wantHash,
		)
	}
}

func TestNew_Errors(t *testing.T) {
	t.Run("missing info", func(t *testing.T) {
		var buf bytes.Buffer

		if err := bencode.NewEncoder(&buf).Encode(map[string]any{"announce": "x"}); err != nil {
			t.Fatalf("encode: %v", err)
		}
		if _, err := ParseMetainfo(bytes.NewReader(buf.Bytes())); err == nil {
			t.Fatalf("expected error for missing info dictionary")
		}
	})

	t.Run("invalid pieces length", func(t *testing.T) {
		info := map[string]any{
			"name":         "x",
			"piece length": int64(1),
			"pieces": string(
				make([]byte, 21),
			), // not multiple of 20
			"length": int64(1),
		}

		var buf bytes.Buffer
		if err := bencode.NewEncoder(&buf).Encode(map[string]any{"info": info}); err != nil {
			t.Fatalf("encode: %v", err)
		}

		if _, err := ParseMetainfo(bytes.NewReader(buf.Bytes())); err == nil {
			t.Fatalf("expected error for invalid pieces length")
		}
	})

	t.Run("single-file missing length", func(t *testing.T) {
		info := map[string]any{
			"name":         "x",
			"piece length": int64(1),
			"pieces":       string(make([]byte, 20)),
			// no length, no files
		}

		var buf bytes.Buffer
		if err := bencode.NewEncoder(&buf).Encode(map[string]any{"info": info}); err != nil {
			t.Fatalf("encode: %v", err)
		}

		if _, err := ParseMetainfo(bytes.NewReader(buf.Bytes())); err == nil {
			t.Fatalf(
				"expected error for missing single-file length",
			)
		}
	})

	t.Run("multi-file invalid path element", func(t *testing.T) {
		info := map[string]any{
			"name":         "x",
			"piece length": int64(1),
			"pieces":       string(make([]byte, 20)),
			"files": []any{
				map[string]any{
					"length": int64(1),
					"path":   []any{"ok", int64(2)},
				},
			},
		}
		var buf bytes.Buffer

		if err := bencode.NewEncoder(&buf).Encode(map[string]any{"info": info}); err != nil {
			t.Fatalf("encode: %v", err)
		}
		if _, err := ParseMetainfo(bytes.NewReader(buf.Bytes())); err == nil {
			t.Fatalf("expected error for non-string path element")
		}
	})
}
