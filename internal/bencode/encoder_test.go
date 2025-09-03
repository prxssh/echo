package bencode

import (
	"bytes"
	"errors"
	"reflect"
	"testing"
)

type errWriter struct{ err error }

func (e errWriter) Write(p []byte) (int, error) { return 0, e.err }

func TestEncodeString(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "0:"},
		{"spam", "4:spam"},
		{"你好", "6:你好"}, // UTF-8 length in bytes
		{"a:b", "3:a:b"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)

		if err := enc.Encode(tt.in); err != nil {
			t.Fatalf("Encode(%q) error = %v", tt.in, err)
		}

		if got := buf.String(); got != tt.want {
			t.Fatalf(
				"Encode(%q) = %q; want %q",
				tt.in,
				got,
				tt.want,
			)
		}
	}
}

func TestEncodeInteger(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{0, "i0e"},
		{42, "i42e"},
		{-7, "i-7e"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)

		if err := enc.Encode(tt.in); err != nil {
			t.Fatalf("Encode(%d) error = %v", tt.in, err)
		}

		if got := buf.String(); got != tt.want {
			t.Fatalf(
				"Encode(%d) = %q; want %q",
				tt.in,
				got,
				tt.want,
			)
		}
	}
}

func TestEncodeList(t *testing.T) {
	tests := []struct {
		in   []any
		want string
	}{
		{[]any{}, "le"},
		{[]any{"spam", "eggs", int64(42)}, "l4:spam4:eggsi42ee"},
		{[]any{"a", []any{"b", "c"}}, "l1:al1:b1:cee"},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)

		if err := enc.Encode(tt.in); err != nil {
			t.Fatalf("Encode(%v) error = %v", tt.in, err)
		}

		if got := buf.String(); got != tt.want {
			t.Fatalf(
				"Encode(%v) = %q; want %q",
				tt.in,
				got,
				tt.want,
			)
		}
	}
}

func TestEncodeDict(t *testing.T) {
	tests := []struct {
		in   map[string]any
		want string
	}{
		{map[string]any{}, "de"},
		{
			map[string]any{"bar": "spam", "foo": int64(42)},
			"d3:bar4:spam3:fooi42ee",
		},
		{
			map[string]any{
				"z": []any{"a"},
				"a": map[string]any{"k": "v"},
			},
			"d1:ad1:k1:ve1:zl1:aee",
		},
	}

	for _, tt := range tests {
		var buf bytes.Buffer
		enc := NewEncoder(&buf)

		if err := enc.Encode(tt.in); err != nil {
			t.Fatalf("Encode(%v) error = %v", tt.in, err)
		}

		if got := buf.String(); got != tt.want {
			t.Fatalf(
				"Encode(%v) = %q; want %q",
				tt.in,
				got,
				tt.want,
			)
		}
	}
}

func TestEncodeUnsupportedType(t *testing.T) {
	var buf bytes.Buffer
	enc := NewEncoder(&buf)

	cases := []any{1, true, nil}
	for _, c := range cases {
		if err := enc.Encode(c); err == nil {
			t.Fatalf("Encode(%T) expected error, got nil", c)
		}
	}
}

func TestEncodeWriterError(t *testing.T) {
	ew := errWriter{err: errors.New("write failed")}
	enc := NewEncoder(ew)

	if err := enc.Encode(int64(1)); err == nil {
		t.Fatalf("Encode(int64) expected error on writer, got nil")
	}
	if err := enc.Encode("a"); err == nil {
		t.Fatalf("Encode(string) expected error on writer, got nil")
	}
	if err := enc.Encode([]any{}); err == nil {
		t.Fatalf("Encode(list) expected error on writer, got nil")
	}
	if err := enc.Encode(map[string]any{}); err == nil {
		t.Fatalf("Encode(dict) expected error on writer, got nil")
	}
}

func TestRoundTripEncodeDecode(t *testing.T) {
	cases := []any{
		"spam",
		"",
		int64(0),
		int64(-10),
		[]any{"a", int64(1), []any{"b"}},
		map[string]any{
			"foo":  int64(1),
			"bar":  "baz",
			"list": []any{"x", int64(2)},
		},
	}

	for _, c := range cases {
		var buf bytes.Buffer

		if err := NewEncoder(&buf).Encode(c); err != nil {
			t.Fatalf("encode error for %v: %v", c, err)
		}

		got, err := NewDecoder(bytes.NewReader(buf.Bytes())).Decode()
		if err != nil {
			t.Fatalf("decode error for %v: %v", c, err)
		}

		if !reflect.DeepEqual(got, c) {
			t.Fatalf(
				"round-trip mismatch: got %v (%T), want %v (%T)",
				got,
				got,
				c,
				c,
			)
		}
	}
}
