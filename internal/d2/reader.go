package d2

import (
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/mem"
)

// ErrNotInGame is returned when the player unit or its chain is not readable (menu/loading).
var ErrNotInGame = errors.New("d2: not in game")

// ErrBadPointer is returned when a pointer fails the sanity check.
var ErrBadPointer = errors.New("d2: implausible pointer")

// Reader decodes game state from a Source. It holds no state; it is safe to share.
type Reader struct {
	S mem.Source
	// Retries is how many extra attempts Snapshot makes when it detects a torn read (default 3).
	Retries int
	// Sleep is called between attempts (default time.Sleep(15ms); tests set a no-op).
	Sleep func(time.Duration)
}

func New(s mem.Source) *Reader {
	return &Reader{S: s, Retries: 3, Sleep: time.Sleep}
}

// okPtr: a plausible user-space heap/image pointer in a 32-bit Wine process.
func okPtr(v uint32) bool { return v >= 0x10000 && v < 0x7fff0000 }

func (r *Reader) u8(a uint32) (uint8, error)   { return mem.U8(r.S, a) }
func (r *Reader) u16(a uint32) (uint16, error) { return mem.U16(r.S, a) }
func (r *Reader) u32(a uint32) (uint32, error) { return mem.U32(r.S, a) }
func (r *Reader) i32(a uint32) (int32, error)  { return mem.I32(r.S, a) }

// ptr reads a pointer at a and checks it; a zero or implausible value is ErrBadPointer.
func (r *Reader) ptr(a uint32) (uint32, error) {
	v, err := r.u32(a)
	if err != nil {
		return 0, err
	}
	if !okPtr(v) {
		return 0, fmt.Errorf("%w: 0x%x at 0x%x", ErrBadPointer, v, a)
	}
	return v, nil
}

// optPtr is like ptr but a zero value is returned as (0, nil).
func (r *Reader) optPtr(a uint32) (uint32, error) {
	v, err := r.u32(a)
	if err != nil {
		return 0, err
	}
	if v == 0 {
		return 0, nil
	}
	if !okPtr(v) {
		return 0, fmt.Errorf("%w: 0x%x at 0x%x", ErrBadPointer, v, a)
	}
	return v, nil
}

func (r *Reader) bytes(a uint32, n int) ([]byte, error) {
	b := make([]byte, n)
	if err := r.S.ReadAt(b, a); err != nil {
		return nil, err
	}
	return b, nil
}

// cstr reads up to n bytes at a and returns the NUL-terminated string (latin-1 bytes kept as is).
func (r *Reader) cstr(a uint32, n int) (string, error) {
	b, err := r.bytes(a, n)
	if err != nil {
		return "", err
	}
	for i, c := range b {
		if c == 0 {
			return string(b[:i]), nil
		}
	}
	return string(b), nil
}

// wstr reads a NUL-terminated UTF-16LE string of at most n code units (ASCII/BMP only).
func (r *Reader) wstr(a uint32, n int) (string, error) {
	b, err := r.bytes(a, n*2)
	if err != nil {
		return "", err
	}
	var rs []rune
	for i := 0; i+1 < len(b); i += 2 {
		c := rune(b[i]) | rune(b[i+1])<<8
		if c == 0 {
			break
		}
		rs = append(rs, c)
	}
	return string(rs), nil
}
