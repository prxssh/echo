package bencode

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
)

type Decoder struct {
	r *bufio.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{r: bufio.NewReader(r)}
}

func (d *Decoder) Decode() (any, error) {
	// consume first byte
	b, err := d.r.ReadByte()
	if err != nil {
		return nil, err
	}

	switch b {
	case 'i':
		return d.decodeInt()
	case 'l':
		return d.decodeList()
	case 'd':
		return d.decodeDict()
	default:
		if '0' <= b && b <= '9' {
			if err := d.r.UnreadByte(); err != nil {
				return nil, fmt.Errorf("bencode: failed unread byte '%v'", err)
			}

			return d.decodeString()
		}

		return nil, fmt.Errorf("bencode: invalid prefix '%c'", b)
	}
}

func (d *Decoder) decodeInt() (int64, error) {
	rest, err := d.r.ReadBytes('e')
	if err != nil {
		return 0, err
	}

	// drop trailing 'e'
	s := string(rest[:len(rest)-1])
	if s == "" {
		return 0, fmt.Errorf("bencode: empty integer")
	}
	size := len(s)

	if s[0] == '-' {
		if size < 2 || s[1] == '0' {
			return 0, fmt.Errorf("bencode: invalid integer '%q'", s)
		}
	} else if size > 1 && s[0] == '0' {
		return 0, fmt.Errorf("bencode: leading zero in integer '%q'", s)
	}
	return strconv.ParseInt(s, 10, 64)
}

func (d *Decoder) decodeString() (string, error) {
	ssize, err := d.r.ReadBytes(':')
	if err != nil {
		return "", err
	}

	s := string(ssize[:len(ssize)-1])
	if s == "" {
		return "", fmt.Errorf("bencode: empty string")
	}
	if s[0] == '-' {
		return "", fmt.Errorf("bencode: invalid string length %q", s)
	}

	slen, err := strconv.Atoi(s)
	if err != nil {
		return "", fmt.Errorf("bencode: invalid string length %q: %w", s, err)
	}

	buf := make([]byte, slen)
	if _, err := io.ReadFull(d.r, buf); err != nil {
		return "", err
	}
	return string(buf), nil
}

func (d *Decoder) decodeList() ([]any, error) {
	var list []any

	for {
		peek, err := d.r.Peek(1)
		if err != nil {
			return nil, err
		}
		if peek[0] == 'e' {
			// consume 'e'
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
		if peek[0] == 'e' {
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

		dict[string(key)] = val

	}

	return dict, nil
}
