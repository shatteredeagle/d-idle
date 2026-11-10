// Package art turns the user's own Diablo II archives into PNGs for the watch UI at runtime.
// Nothing here ships or stores Blizzard art: images are decoded on request from the install the
// user already owns and cached under the data directory (docs/paperdoll-art-build.md).
package art

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"image/color"
)

// Frame is one decoded DC6 frame. Pix holds palette indexes, 0 = transparent; len(Pix) = W*H, top row first.
type Frame struct {
	W, H       int
	OffX, OffY int
	Pix        []byte
}

const (
	dc6HeaderSize = 24
	dc6FrameHdr   = 32
	maxDim        = 4096
)

// DecodeDC6 decodes every frame of a one-direction (or multi-direction, frames concatenated) DC6 file.
// Format source: OpenDiablo2 d2fileformats/d2dc6; verified against the 1.14d item and panel files
// (docs/paperdoll-art-build.md section 1).
func DecodeDC6(b []byte) ([]Frame, error) {
	if len(b) < dc6HeaderSize+4 {
		return nil, errors.New("art: dc6 too short")
	}
	if v := binary.LittleEndian.Uint32(b[0:]); v != 6 {
		return nil, fmt.Errorf("art: dc6 version %d", v)
	}
	dirs := int(binary.LittleEndian.Uint32(b[16:]))
	per := int(binary.LittleEndian.Uint32(b[20:]))
	n := dirs * per
	if dirs < 1 || per < 1 || n > 1024 || dc6HeaderSize+4*n > len(b) {
		return nil, errors.New("art: dc6 frame table out of range")
	}
	out := make([]Frame, 0, n)
	for i := 0; i < n; i++ {
		off := int(binary.LittleEndian.Uint32(b[dc6HeaderSize+4*i:]))
		f, err := decodeFrame(b, off)
		if err != nil {
			return nil, fmt.Errorf("art: dc6 frame %d: %w", i, err)
		}
		out = append(out, f)
	}
	return out, nil
}

func decodeFrame(b []byte, off int) (Frame, error) {
	if off < 0 || off+dc6FrameHdr > len(b) {
		return Frame{}, errors.New("frame header out of range")
	}
	h := b[off:]
	flip := binary.LittleEndian.Uint32(h[0:])
	w := int(binary.LittleEndian.Uint32(h[4:]))
	ht := int(binary.LittleEndian.Uint32(h[8:]))
	f := Frame{W: w, H: ht, OffX: int(int32(binary.LittleEndian.Uint32(h[12:]))), OffY: int(int32(binary.LittleEndian.Uint32(h[16:])))}
	length := int(binary.LittleEndian.Uint32(h[28:]))
	if w < 0 || ht < 0 || w > maxDim || ht > maxDim {
		return f, errors.New("frame size out of range")
	}
	start := off + dc6FrameHdr
	if length < 0 || start+length > len(b) {
		return f, errors.New("frame data out of range")
	}
	data := b[start : start+length]
	f.Pix = make([]byte, w*ht)
	// Scanlines are stored bottom-up unless the flip flag is set.
	y, step := ht-1, -1
	if flip != 0 {
		y, step = 0, 1
	}
	x := 0
	for i := 0; i < len(data) && y >= 0 && y < ht; {
		c := data[i]
		i++
		switch {
		case c == 0x80:
			x, y = 0, y+step
		case c&0x80 != 0:
			x += int(c & 0x7f)
		default:
			n := int(c)
			if i+n > len(data) {
				return f, errors.New("literal run past end of data")
			}
			for k := 0; k < n; k++ {
				if x < w {
					f.Pix[y*w+x] = data[i+k]
				}
				x++
			}
			i += n
		}
	}
	return f, nil
}

// Palette is 256 RGB entries.
type Palette [256][3]uint8

// ParsePalette reads a pal.dat (768 bytes, B G R per entry; the order is confirmed by the item icons
// rendering with the right colours, docs/paperdoll-art-build.md).
func ParsePalette(b []byte) (*Palette, error) {
	if len(b) < 768 {
		return nil, fmt.Errorf("art: palette has %d bytes, want 768", len(b))
	}
	var p Palette
	for i := range p {
		p[i] = [3]uint8{b[3*i+2], b[3*i+1], b[3*i]}
	}
	return &p, nil
}

// Image renders the frame with the palette; index 0 is transparent.
func (f Frame) Image(p *Palette) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, f.W, f.H))
	for i, c := range f.Pix {
		if c == 0 {
			continue
		}
		e := p[c]
		img.SetNRGBA(i%f.W, i/f.W, color.NRGBA{e[0], e[1], e[2], 255})
	}
	return img
}
