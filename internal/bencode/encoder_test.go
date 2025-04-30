package bencode

import (
	"bytes"
	"testing"
)

func TestEncode(t *testing.T) {
	tests := []struct {
		name    string
		value   any
		want    string
		wantErr bool
	}{
		{"int positive", int64(42), "i42e", false},
		{"int from int", 7, "i7e", false},
		{"int zero", int64(0), "i0e", false},
		{"int negative", int64(-5), "i-5e", false},
		{"string", "spam", "4:spam", false},
		{"empty string", "", "0:", false},
		{
			"list ints",
			[]any{int64(1), int64(2), int64(-3)},
			"li1ei2ei-3ee",
			false,
		},
		{"list mixed", []any{"foo", int64(1)}, "l3:fooi1ee", false},
		{
			"nested list",
			[]any{[]any{int64(1), int64(2)}, "abc"},
			"lli1ei2ee3:abce",
			false,
		},
		{"dict simple", map[string]any{"foo": "bar"}, "d3:foo3:bare", false},
		{
			"dict key order",
			map[string]any{"b": "B", "a": "A"},
			"d1:a1:A1:b1:Be",
			false,
		},
		{
			"nested dict",
			map[string]any{"list": []any{int64(1), int64(2)}},
			"d4:listli1ei2eee",
			false,
		},
		{
			"dict mixed",
			map[string]any{"int": int64(1), "str": "x"},
			"d3:inti1e3:str1:xe",
			false,
		},
		{"unsupported", 3.14, "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			enc := NewEncoder(&buf)
			err := enc.Encode(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Encode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			got := buf.String()
			if got != tt.want {
				t.Errorf("Encode() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMarshal(t *testing.T) {
	b, err := Marshal(map[string]any{"foo": "bar"})
	if err != nil {
		t.Fatalf("Marshal() error = %v", err)
	}
	want := "d3:foo3:bare"
	if string(b) != want {
		t.Errorf("Marshal() = %q, want %q", string(b), want)
	}
}

func TestMarshal_UnsupportedType(t *testing.T) {
	if _, err := Marshal(struct{}{}); err == nil {
		t.Fatal("Marshal() expected error for unsupported type, got nil")
	}
}
