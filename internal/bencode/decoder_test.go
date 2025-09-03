package bencode

import (
	"reflect"
	"strings"
	"testing"
)

func decodeString(t *testing.T, s string) any {
	t.Helper()

	v, err := NewDecoder(strings.NewReader(s)).Decode()
	if err != nil {
		t.Fatalf("Decode(%q) error = %v", s, err)
	}

	return v
}

func TestDecodeString(t *testing.T) {
	if got := decodeString(t, "4:spam"); got != "spam" {
		t.Fatalf("got %v, want %v", got, "spam")
	}

	if got := decodeString(t, "0:"); got != "" {
		t.Fatalf("got %v, want empty string", got)
	}

	if got := decodeString(t, "6:你好"); got != "你好" {
		t.Fatalf("got %v, want %v", got, "你好")
	}
}

func TestDecodeInteger(t *testing.T) {
	cases := map[string]int64{
		"i0e":  0,
		"i42e": 42,
		"i-7e": -7,
		"i-0e": 0, // allowed by implementation
	}

	for s, want := range cases {
		v, err := NewDecoder(strings.NewReader(s)).Decode()
		if err != nil {
			t.Fatalf("Decode(%q) error = %v", s, err)
		}

		if v != want {
			t.Fatalf("Decode(%q) = %v; want %v", s, v, want)
		}
	}
}

func TestDecodeList(t *testing.T) {
	cases := []struct {
		in   string
		want []any
	}{
		{"le", []any{}},
		{"l4:spam4:eggsi42ee", []any{"spam", "eggs", int64(42)}},
		{"l1:al1:b1:cee", []any{"a", []any{"b", "c"}}},
	}

	for _, tt := range cases {
		v, err := NewDecoder(strings.NewReader(tt.in)).Decode()
		if err != nil {
			t.Fatalf("Decode(%q) error = %v", tt.in, err)
		}

		if !reflect.DeepEqual(v, tt.want) {
			t.Fatalf(
				"Decode(%q) = %#v; want %#v",
				tt.in,
				v,
				tt.want,
			)
		}
	}
}

func TestDecodeDict(t *testing.T) {
	cases := []struct {
		in   string
		want map[string]any
	}{
		{"de", map[string]any{}},
		{
			"d3:bar4:spam3:fooi42ee",
			map[string]any{"bar": "spam", "foo": int64(42)},
		},
		{
			"d1:ad1:k1:ve1:zl1:aee",
			map[string]any{
				"a": map[string]any{"k": "v"},
				"z": []any{"a"},
			},
		},
	}

	for _, tt := range cases {
		v, err := NewDecoder(strings.NewReader(tt.in)).Decode()
		if err != nil {
			t.Fatalf("Decode(%q) error = %v", tt.in, err)
		}

		if !reflect.DeepEqual(v, tt.want) {
			t.Fatalf(
				"Decode(%q) = %#v; want %#v",
				tt.in,
				v,
				tt.want,
			)
		}
	}
}

func TestDecodeErrors(t *testing.T) {
	// Negative string length
	if _, err := NewDecoder(strings.NewReader("-1:")).Decode(); err == nil {
		t.Fatalf("expected error for negative string length")
	} else if err.Error() != "bencode: invalid string, length can't be negative" {
		t.Fatalf("unexpected error: %v", err)
	}

	// Non-numeric string length
	if _, err := NewDecoder(strings.NewReader("x:ab")).Decode(); err == nil {
		t.Fatalf("expected error for non-numeric string length")
	}

	// Missing ':' in string length
	if _, err := NewDecoder(strings.NewReader("3")).Decode(); err == nil {
		t.Fatalf("expected error for missing ':' in string length")
	}

	// Invalid integer: missing 'e'
	if _, err := NewDecoder(strings.NewReader("i42")).Decode(); err == nil {
		t.Fatalf("expected error for unterminated integer")
	}

	// Invalid integer content
	if _, err := NewDecoder(strings.NewReader("i4x2e")).Decode(); err == nil {
		t.Fatalf("expected error for invalid integer content")
	}

	// Unterminated list
	if _, err := NewDecoder(strings.NewReader("l4:spam")).Decode(); err == nil {
		t.Fatalf("expected error for unterminated list")
	}

	// Unterminated dict
	if _, err := NewDecoder(strings.NewReader("d3:bar4:spam")).Decode(); err == nil {
		t.Fatalf("expected error for unterminated dict")
	}

	// Dict key not a string (parser will fail attempting to read key as
	// string)
	if _, err := NewDecoder(strings.NewReader("di1e3:abce")).Decode(); err == nil {
		t.Fatalf("expected error when dict key is not a string")
	}
}
