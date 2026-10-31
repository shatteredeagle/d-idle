package d2map

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sort"

	"d2lod-bot/internal/mem"
)

// Recorder wraps a Source and remembers every byte range successfully read through it, so a
// live session can be saved as a fixture (Image) and replayed offline. Later reads of the same
// address overwrite earlier ones.
type Recorder struct {
	Src  mem.Source
	segs map[uint32][]byte
}

func NewRecorder(src mem.Source) *Recorder { return &Recorder{Src: src, segs: map[uint32][]byte{}} }

func (r *Recorder) ReadAt(p []byte, addr uint32) error {
	if err := r.Src.ReadAt(p, addr); err != nil {
		return err
	}
	r.segs[addr] = append(r.segs[addr][:0], p...)
	return nil
}

// Image freezes the recorded ranges into a replayable memory image.
func (r *Recorder) Image() *Image {
	img := &Image{}
	for a, b := range r.segs {
		img.segs = append(img.segs, seg{a, append([]byte(nil), b...)})
	}
	img.normalize()
	return img
}

type seg struct {
	addr uint32
	data []byte
}

// Image is an immutable-after-load memory image that implements mem.Source.
type Image struct{ segs []seg }

func (m *Image) normalize() {
	sort.Slice(m.segs, func(i, j int) bool { return m.segs[i].addr < m.segs[j].addr })
	var out []seg
	for _, s := range m.segs {
		if n := len(out); n > 0 {
			last := &out[n-1]
			end := uint64(last.addr) + uint64(len(last.data))
			if uint64(s.addr) <= end {
				// overlapping or adjacent: merge, the later (higher) segment wins on overlap
				need := int(uint64(s.addr) + uint64(len(s.data)) - uint64(last.addr))
				if need > len(last.data) {
					last.data = append(last.data, make([]byte, need-len(last.data))...)
				}
				copy(last.data[s.addr-last.addr:], s.data)
				continue
			}
		}
		out = append(out, seg{s.addr, append([]byte(nil), s.data...)})
	}
	m.segs = out
}

func (m *Image) ReadAt(p []byte, addr uint32) error {
	i := sort.Search(len(m.segs), func(i int) bool { return m.segs[i].addr > addr }) - 1
	if i >= 0 {
		s := m.segs[i]
		if uint64(addr)+uint64(len(p)) <= uint64(s.addr)+uint64(len(s.data)) {
			copy(p, s.data[addr-s.addr:])
			return nil
		}
	}
	return fmt.Errorf("d2map.Image: unmapped read of %d bytes at 0x%x", len(p), addr)
}

// Put adds bytes at addr (for hand-built images).
func (m *Image) Put(addr uint32, b []byte) {
	m.segs = append(m.segs, seg{addr, append([]byte(nil), b...)})
	m.normalize()
}

// Size is the total number of bytes held.
func (m *Image) Size() int {
	n := 0
	for _, s := range m.segs {
		n += len(s.data)
	}
	return n
}

const imageMagic = "D2MI1"

// Save writes the image in a small gzip-compressed format.
func (m *Image) Save(w io.Writer) error {
	zw := gzip.NewWriter(w)
	var buf bytes.Buffer
	buf.WriteString(imageMagic)
	_ = binary.Write(&buf, binary.LittleEndian, uint32(len(m.segs)))
	for _, s := range m.segs {
		_ = binary.Write(&buf, binary.LittleEndian, s.addr)
		_ = binary.Write(&buf, binary.LittleEndian, uint32(len(s.data)))
		buf.Write(s.data)
	}
	if _, err := zw.Write(buf.Bytes()); err != nil {
		return err
	}
	return zw.Close()
}

// LoadImage reads an image written by Save.
func LoadImage(r io.Reader) (*Image, error) {
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(zr)
	if err != nil {
		return nil, err
	}
	if len(raw) < len(imageMagic)+4 || string(raw[:len(imageMagic)]) != imageMagic {
		return nil, errors.New("d2map: not a memory image")
	}
	n := int(binary.LittleEndian.Uint32(raw[len(imageMagic):]))
	p := len(imageMagic) + 4
	m := &Image{}
	for i := 0; i < n; i++ {
		if p+8 > len(raw) {
			return nil, errors.New("d2map: truncated image")
		}
		addr := binary.LittleEndian.Uint32(raw[p:])
		l := int(binary.LittleEndian.Uint32(raw[p+4:]))
		p += 8
		if p+l > len(raw) {
			return nil, errors.New("d2map: truncated image")
		}
		m.segs = append(m.segs, seg{addr, raw[p : p+l]})
		p += l
	}
	m.normalize()
	return m, nil
}
