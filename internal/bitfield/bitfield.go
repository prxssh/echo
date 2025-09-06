package bitfield

import (
	"bytes"
	"math/bits"
)

// Bitfield represents a sequence of bits stored in a byte slice.
// The bit at index 0 is the MSB of byte 0, following BEP 3 convention.
type Bitfield []byte

// New creates a Bitfield capable of holding n bits. The underlying storage
// is rounded up to the nearest byte.
func New(n int64) Bitfield {
	size := int((n + 7) / 8)
	if size < 0 {
		size = 0
	}
	return make(Bitfield, size)
}

// FromBytes returns a copy of the provided byte slice as a Bitfield.
func FromBytes(b []byte) Bitfield {
	bf := make(Bitfield, len(b))
	copy(bf, b)
	return bf
}

// ToBytes returns a copy of the underlying bytes representing this bitfield.
func (bf Bitfield) ToBytes() []byte {
	out := make([]byte, len(bf))
	copy(out, bf)
	return out
}

// HasBit returns true if the bit at the given index is set.
func (bf Bitfield) HasBit(index int) bool {
	byteIndex, offset := index/8, index%8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return false
	}
	return bf[byteIndex]>>(7-offset)&1 != 0
}

// SetBit sets the bit at the given index to 1.
func (bf Bitfield) SetBit(index int) {
	byteIndex, offset := index/8, index%8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}
	bf[byteIndex] |= 1 << (7 - offset)
}

// ClearBit clears the bit at the given index to 0.
func (bf Bitfield) ClearBit(index int) {
	byteIndex, offset := index/8, index%8
	if byteIndex < 0 || byteIndex >= len(bf) {
		return
	}
	bf[byteIndex] &^= 1 << (7 - offset)
}

// Len returns the total number of bits the Bitfield can hold.
func (bf Bitfield) Len() int { return len(bf) * 8 }

// Count returns the number of set bits in the Bitfield.
func (bf Bitfield) Count() int {
	c := 0
	for _, b := range bf {
		c += bits.OnesCount8(b)
	}
	return c
}

// Equals reports whether two bitfields have identical contents.
func (bf Bitfield) Equals(other Bitfield) bool {
	return bytes.Equal(bf, other)
}

// String returns a human-readable representation of the bitfield as a
// sequence of '0' and '1' characters, big-endian within each byte.
func (bf Bitfield) String() string {
	var buf bytes.Buffer
	for i := 0; i < bf.Len(); i++ {
		if bf.HasBit(i) {
			buf.WriteByte('1')
		} else {
			buf.WriteByte('0')
		}
	}
	return buf.String()
}
