package bitfield

import (
	"testing"
)

func TestNewAndLen(t *testing.T) {
	tests := []struct {
		nBits    int
		expBytes int
	}{
		{1, 1},
		{8, 1},
		{9, 2},
		{16, 2},
		{0, 0},
	}
	for _, tt := range tests {
		bf := New(tt.nBits)
		if len(bf) != tt.expBytes {
			t.Errorf("New(%d) length = %d bytes; want %d", tt.nBits, len(bf), tt.expBytes)
		}
		if bf.Len() != tt.expBytes*8 {
			t.Errorf("Len() = %d bits; want %d bits", bf.Len(), tt.expBytes*8)
		}
	}
}

func TestFromBytesIndependentCopy(t *testing.T) {
	orig := []byte{0xFF, 0x00}
	bf := FromBytes(orig)
	// modify original
	orig[0] = 0x00
	if bf[0] != 0xFF {
		t.Errorf("FromBytes did not copy slice properly: bf[0]=%02x; want ff", bf[0])
	}
}

func TestSetClearHasBit(t *testing.T) {
	bf := New(16)
	// initially no bits set
	for i := 0; i < bf.Len(); i++ {
		if bf.HasBit(i) {
			t.Errorf("HasBit(%d) = true; want false on fresh Bitfield", i)
		}
	}
	// set some bits
	indices := []int{0, 5, 8, 15}
	for _, idx := range indices {
		bf.SetBit(idx)
		if !bf.HasBit(idx) {
			t.Errorf("HasBit(%d) = false after SetBit; want true", idx)
		}
	}
	// clear bits
	for _, idx := range indices {
		bf.ClearBit(idx)
		if bf.HasBit(idx) {
			t.Errorf("HasBit(%d) = true after ClearBit; want false", idx)
		}
	}
}

func TestString(t *testing.T) {
	// construct known byte pattern: 10100000 00010010
	data := []byte{0b10100000, 0b00010010}
	bf := FromBytes(data)
	want := "1010000000010010"
	got := bf.String()
	if got != want {
		t.Errorf("String() = %q; want %q", got, want)
	}
}

func TestOutOfRangeHasBit(t *testing.T) {
	bf := New(8)
	// negative index
	if bf.HasBit(-1) {
		t.Error("HasBit(-1) = true; want false")
	}
	// index >= Len
	if bf.HasBit(bf.Len()) {
		t.Errorf("HasBit(%d) = true; want false", bf.Len())
	}
}

func TestOutOfRangeSetClearBit(t *testing.T) {
	bf := New(8)
	// should not panic
	func() {
		defer func() {
			recover()
		}()
		bf.SetBit(100)
		bf.ClearBit(100)
	}()
}
