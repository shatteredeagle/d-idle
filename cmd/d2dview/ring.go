package main

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// The frame ring is a small file the d2d fork's `framefeed` verb keeps up to date (docs/viewer-fps.md): a 64-byte header
// and two pixel slots. d2d copies a drawn frame into slot (seq+1)%2, then stores seq+1; nothing is encoded and nothing
// is asked per frame, so a frame costs d2d one memcpy. The layout below is the contract with the fork (little-endian):
//
//	0  [4]byte "D2FF"      4  u32 version (1)      8  u32 width     12 u32 height
//	16 u32 stride (bytes per row)                  20 u32 format (1 = RGBA8, byte order R G B A, alpha 255)
//	24 u64 seq (frames published, 0 none; stored last)   32 u64 d2d frame counter   40 u64 game ms
//	48..63 reserved; slot 0 at 64, slot 1 at 64 + stride*height
const (
	ringMagic   = "D2FF"
	ringVersion = 1
	ringHdr     = 64
	ringRGBA8   = 1
	ringMaxSide = 4096
)

type ringHeader struct {
	W, H, Stride int
	Seq          uint64
	Frame        uint64
	GameMS       uint64
}

func (h ringHeader) slot() int64 { return ringHdr + int64(h.Seq%2)*int64(h.Stride*h.H) }

func parseRingHeader(b []byte) (ringHeader, error) {
	if len(b) < ringHdr || string(b[:4]) != ringMagic {
		return ringHeader{}, errors.New("framefeed: not a frame ring")
	}
	le := binary.LittleEndian
	if v := le.Uint32(b[4:]); v != ringVersion {
		return ringHeader{}, fmt.Errorf("framefeed: ring version %d", v)
	}
	if f := le.Uint32(b[20:]); f != ringRGBA8 {
		return ringHeader{}, fmt.Errorf("framefeed: pixel format %d", f)
	}
	h := ringHeader{W: int(le.Uint32(b[8:])), H: int(le.Uint32(b[12:])), Stride: int(le.Uint32(b[16:])),
		Seq: le.Uint64(b[24:]), Frame: le.Uint64(b[32:]), GameMS: le.Uint64(b[40:])}
	if h.W <= 0 || h.H <= 0 || h.W > ringMaxSide || h.H > ringMaxSide || h.Stride < 4*h.W || h.Stride > 4*ringMaxSide {
		return ringHeader{}, fmt.Errorf("framefeed: bad ring geometry %dx%d stride %d", h.W, h.H, h.Stride)
	}
	return h, nil
}

// readRing copies the newest frame out of the ring into buf (grown as needed). A frame overwritten while it was read
// (the seq moved) is retried; the d2d writes at most 25 a second and a copy takes well under a millisecond.
func readRing(f *os.File, buf []byte) (ringHeader, []byte, error) {
	var hb [ringHdr]byte
	for try := 0; try < 3; try++ {
		if _, err := f.ReadAt(hb[:], 0); err != nil {
			return ringHeader{}, buf, err
		}
		h, err := parseRingHeader(hb[:])
		if err != nil || h.Seq == 0 {
			return h, buf, err
		}
		n := h.Stride * h.H
		if cap(buf) < n {
			buf = make([]byte, n)
		}
		buf = buf[:n]
		if _, err := f.ReadAt(buf, h.slot()); err != nil && !errors.Is(err, io.EOF) {
			return h, buf, err
		}
		if _, err := f.ReadAt(hb[24:32], 24); err != nil {
			return h, buf, err
		}
		if binary.LittleEndian.Uint64(hb[24:]) == h.Seq {
			return h, buf, nil
		}
	}
	return ringHeader{}, buf, errors.New("framefeed: ring kept moving during the read")
}

// writeRing is the writer side of the contract (what the fork does in C++), for tests and fakes.
func writeRing(f *os.File, w, h int, pix []byte, frame, gameMS uint64) error {
	var hb [ringHdr]byte
	seq := uint64(0)
	if _, err := f.ReadAt(hb[:], 0); err == nil {
		if old, err := parseRingHeader(hb[:]); err == nil && old.W == w && old.H == h {
			seq = old.Seq
		}
	}
	le := binary.LittleEndian
	copy(hb[:], ringMagic)
	le.PutUint32(hb[4:], ringVersion)
	le.PutUint32(hb[8:], uint32(w))
	le.PutUint32(hb[12:], uint32(h))
	le.PutUint32(hb[16:], uint32(4*w))
	le.PutUint32(hb[20:], ringRGBA8)
	le.PutUint64(hb[24:], seq)
	le.PutUint64(hb[32:], frame)
	le.PutUint64(hb[40:], gameMS)
	next := ringHeader{W: w, H: h, Stride: 4 * w, Seq: seq + 1}
	if _, err := f.WriteAt(pix[:4*w*h], next.slot()); err != nil {
		return err
	}
	if _, err := f.WriteAt(hb[:24], 0); err != nil {
		return err
	}
	if _, err := f.WriteAt(hb[32:48], 32); err != nil {
		return err
	}
	le.PutUint64(hb[24:], next.Seq)
	_, err := f.WriteAt(hb[24:32], 24)
	return err
}
