package bencode

import (
	"bufio"
	"errors"
	"io"
	"strconv"
)

type Decoder struct {
	r *bufio.Reader
}

type bType byte

const (
	bInteger bType = 'i'
	bDict    bType = 'd'
	bList    bType = 'l'
	bDelim   bType = 'e'
)

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

func (d *Decoder) Decode() (any, error) {
	btype, err := d.r.ReadByte()
	if err != nil {
		return nil, err
	}

	var val any

	switch btype {
	case byte(bInteger):
		val, err = d.decodeInteger()
	case byte(bList):
		val, err = d.decodeList()
	case byte(bDict):
		val, err = d.decodeDict()
	default:
		if err := d.r.UnreadByte(); err != nil {
			return nil, err
		}

		val, err = d.decodeString()
	}

	if err != nil {
		return nil, err
	}
	return val, nil
}

func (d *Decoder) decodeInteger() (int64, error) {
	return d.readInteger(bDelim)
}

func (d *Decoder) decodeString() (string, error) {
	size, err := d.readInteger(':')
	if err != nil {
		return "", err
	}
	if size == 0 {
		return "", nil
	}
	if size < 0 {
		return "", errors.New(
			"bencode: invalid string, length can't be negative",
		)
	}

	buf := make([]byte, size)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (d *Decoder) decodeList() ([]any, error) {
	list := make([]any, 0)

	for {
		peek, err := d.r.Peek(1)
		if err != nil {
			return nil, err
		}
		if peek[0] == byte(bDelim) {
			d.r.ReadByte()
			break
		}

		v, err := d.Decode()
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}

	return list, nil
}

func (d *Decoder) decodeDict() (map[string]any, error) {
	dict := make(map[string]any)

	for {
		peek, err := d.r.Peek(1)
		if err != nil {
			return nil, err
		}
		if peek[0] == byte(bDelim) {
			d.r.ReadByte()
			break
		}

		key, err := d.decodeString()
		if err != nil {
			return nil, err
		}
		val, err := d.Decode()
		if err != nil {
			return nil, err
		}

		dict[key] = val
	}

	return dict, nil
}

func (d *Decoder) readInteger(delim bType) (int64, error) {
	read, err := d.r.ReadBytes(byte(delim))
	if err != nil {
		return 0, err
	}

	sint := string(read[:len(read)-1])
	return strconv.ParseInt(sint, 10, 64)
}
