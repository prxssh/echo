package bencode

import (
	"bytes"
	"reflect"
	"testing"
)

func TestDecode(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    any
		wantErr bool
	}{
		{name: "integer positive", input: "i42e", want: int64(42)},
		{name: "integer zero", input: "i0e", want: int64(0)},
		{name: "integer negative", input: "i-7e", want: int64(-7)},
		{name: "string", input: "4:spam", want: "spam"},
		{name: "empty string error", input: "0:", want: ""},
		{
			name:  "list of ints",
			input: "li1ei2ei3ee",
			want:  []any{int64(1), int64(2), int64(3)},
		},
		{
			name:  "nested list",
			input: "lli1ei2ee3:abce",
			want:  []any{[]any{int64(1), int64(2)}, "abc"},
		},
		{
			name:  "dict simple",
			input: "d3:foo3:bare",
			want:  map[string]any{"foo": "bar"},
		},
		{
			name:  "dict nested",
			input: "d4:listli1ei2eee",
			want:  map[string]any{"list": []any{int64(1), int64(2)}},
		},
		{name: "invalid prefix", input: "xe", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := NewDecoder(bytes.NewReader([]byte(tt.input)))
			got, err := d.Decode()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Decode() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
