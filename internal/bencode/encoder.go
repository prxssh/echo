package bencode

import (
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

func (e *Encoder) Encode(v any) error {
	switch vt := v.(type) {
	case string:
		return e.encodeString(vt)
	case []any:
		return e.encodeList(vt)
	case map[string]any:
		return e.encodeDict(vt)
	case int64:
		return e.encodeInteger(vt)
	default:
		return fmt.Errorf("bencode: unsupported type '%T'", vt)
	}
}

func (e *Encoder) encodeInteger(v int64) error {
	buf := []byte{byte(bInteger)}
	buf = append(buf, strconv.FormatInt(v, 10)...)
	buf = append(buf, byte(bDelim))

	_, err := e.w.Write(buf)
	return err
}

func (e *Encoder) encodeString(v string) error {
	buf := []byte(strconv.Itoa(len(v)))
	buf = append(buf, byte(bDelim))
	buf = append(buf, v...)

	_, err := e.w.Write(buf)
	return err
}

func (e *Encoder) encodeList(list []any) error {
	if _, err := e.w.Write([]byte{byte(bList)}); err != nil {
		return err
	}

	for _, l := range list {
		if err := e.Encode(l); err != nil {
			return err
		}
	}

	_, err := e.w.Write([]byte{byte(bDelim)})
	return err
}

func (e *Encoder) encodeDict(dict map[string]any) error {
	if _, err := e.w.Write([]byte{byte(bDict)}); err != nil {
		return err
	}

	keys := make([]string, 0, len(dict))
	for k := range dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := e.Encode(k); err != nil {
			return err
		}
		if err := e.Encode(dict[k]); err != nil {
			return err
		}
	}

	_, err := e.w.Write([]byte{byte(bDelim)})
	return err
}
