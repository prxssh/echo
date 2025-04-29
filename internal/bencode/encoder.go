package bencode

import (
	"bytes"
	"fmt"
	"io"
	"sort"
	"strconv"
)

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func Marshal(v any) ([]byte, error) {
	buf := &bytes.Buffer{}
	enc := NewEncoder(buf)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (e *Encoder) Encode(v any) error {
	switch vv := v.(type) {
	case int:
		return e.encodeInt(int64(vv))
	case int64:
		return e.encodeInt(vv)
	case string:
		return e.encodeString(vv)
	case []any:
		return e.encodeList(vv)
	case map[string]any:
		return e.encodeDict(vv)
	default:
		return fmt.Errorf("bencode: unsupported type %T", vv)
	}
}

func (e *Encoder) encodeInt(i int64) error {
	_, err := e.w.Write([]byte("i" + strconv.FormatInt(i, 10) + "e"))
	return err
}

func (e *Encoder) encodeString(s string) error {
	_, err := e.w.Write([]byte(strconv.Itoa(len(s)) + ":" + s))
	return err
}

func (e *Encoder) encodeList(list []any) error {
	if _, err := e.w.Write([]byte("l")); err != nil {
		return err
	}

	for _, item := range list {
		if err := e.Encode(item); err != nil {
			return err
		}
	}

	_, err := e.w.Write([]byte("e"))
	return err
}

func (e *Encoder) encodeDict(dict map[string]any) error {
	if _, err := e.w.Write([]byte("d")); err != nil {
		return err
	}

	// Bencode spec requires keys sorted lexicographically
	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := e.encodeString(k); err != nil {
			return err
		}
		if err := e.Encode(dict[k]); err != nil {
			return err
		}
	}

	_, err := e.w.Write([]byte("e"))
	return err
}
