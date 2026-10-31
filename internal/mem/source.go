// Package mem is the only way the bot reads the game's memory.
// Everything above it (internal/d2, internal/d2map, ...) takes a mem.Source, so all logic can be
// unit-tested against a Fake or a recorded Snapshot without a running game.
package mem

import (
	"encoding/binary"
	"sort"
)

// Source reads the 32-bit address space of Game.exe. Addresses are the game's own
// virtual addresses (Game.exe is mapped at 0x400000). Reads never write or attach.
type Source interface {
	// ReadAt fills p from addr. A short or unmapped read is an error (all-or-nothing).
	ReadAt(p []byte, addr uint32) error
}

// Writer is a Source that can also be written. Only internal/cheat uses it.
type Writer interface {
	Source
	WriteAt(p []byte, addr uint32) error
}

// Base is where Game.exe 1.14.3.71 is mapped (preferred image base, no ASLR under Wine).
const Base uint32 = 0x400000

func U8(s Source, addr uint32) (uint8, error) {
	var b [1]byte
	err := s.ReadAt(b[:], addr)
	return b[0], err
}

func U16(s Source, addr uint32) (uint16, error) {
	var b [2]byte
	err := s.ReadAt(b[:], addr)
	return binary.LittleEndian.Uint16(b[:]), err
}

func U32(s Source, addr uint32) (uint32, error) {
	var b [4]byte
	err := s.ReadAt(b[:], addr)
	return binary.LittleEndian.Uint32(b[:]), err
}

func I32(s Source, addr uint32) (int32, error) {
	v, err := U32(s, addr)
	return int32(v), err
}

// Fake is an in-memory Source for tests: Put byte ranges, ReadAt returns them.
type Fake struct {
	segs map[uint32][]byte // start -> bytes
}

func NewFake() *Fake { return &Fake{segs: map[uint32][]byte{}} }

// Put stores b at addr (later Puts may overlap earlier ones; the most specific start wins on read).
func (f *Fake) Put(addr uint32, b []byte) { f.segs[addr] = append([]byte(nil), b...) }

// PutU32 stores little-endian words starting at addr.
func (f *Fake) PutU32(addr uint32, vals ...uint32) {
	b := make([]byte, 4*len(vals))
	for i, v := range vals {
		binary.LittleEndian.PutUint32(b[i*4:], v)
	}
	f.Put(addr, b)
}

func (f *Fake) ReadAt(p []byte, addr uint32) error {
	starts := make([]uint32, 0, len(f.segs))
	for s := range f.segs {
		starts = append(starts, s)
	}
	sort.Slice(starts, func(i, j int) bool { return starts[i] < starts[j] })
	for i := len(starts) - 1; i >= 0; i-- {
		s := starts[i]
		seg := f.segs[s]
		if addr >= s && uint64(addr)+uint64(len(p)) <= uint64(s)+uint64(len(seg)) {
			copy(p, seg[addr-s:])
			return nil
		}
	}
	return &ReadError{Addr: addr, Len: len(p), Err: ErrUnmapped}
}

// WriteAt overwrites existing bytes of f (a Put range must cover the whole write); for cheat tests.
func (f *Fake) WriteAt(p []byte, addr uint32) error {
	for s, seg := range f.segs {
		if addr >= s && uint64(addr)+uint64(len(p)) <= uint64(s)+uint64(len(seg)) {
			copy(seg[addr-s:], p)
			return nil
		}
	}
	return &ReadError{Addr: addr, Len: len(p), Err: ErrUnmapped}
}
