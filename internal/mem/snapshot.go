package mem

import (
	"bufio"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
)

// PageSize is the Snapshot granularity.
const PageSize = 4096

const snapMagic = "D2SN"
const snapVersion uint16 = 1

// Snapshot is a Source over sparse 4 KiB pages. A read that touches a page that is not present
// fails with ErrUnmapped, exactly like an unmapped address in the live process.
//
// File format (version 1): "D2SN", u16 version (LE), then a gzip stream of: u32 page count,
// then per page (ascending): u32 page address, PageSize bytes.
type Snapshot struct {
	pages map[uint32]*[PageSize]byte
}

func NewSnapshot() *Snapshot { return &Snapshot{pages: map[uint32]*[PageSize]byte{}} }

// Put copies b to addr, creating zeroed pages as needed.
func (s *Snapshot) Put(addr uint32, b []byte) {
	for i := range b {
		a := addr + uint32(i)
		base := a &^ (PageSize - 1)
		pg := s.pages[base]
		if pg == nil {
			pg = new([PageSize]byte)
			s.pages[base] = pg
		}
		pg[a-base] = b[i]
	}
}

// NumPages returns the number of stored pages.
func (s *Snapshot) NumPages() int { return len(s.pages) }

func (s *Snapshot) ReadAt(p []byte, addr uint32) error {
	for i := 0; i < len(p); {
		a := addr + uint32(i)
		base := a &^ (PageSize - 1)
		pg := s.pages[base]
		if pg == nil {
			return &ReadError{Addr: addr, Len: len(p), Err: ErrUnmapped}
		}
		i += copy(p[i:], pg[a-base:])
	}
	return nil
}

func (s *Snapshot) WriteTo(w io.Writer) (int64, error) {
	cw := &countWriter{w: w}
	var hdr [6]byte
	copy(hdr[:], snapMagic)
	binary.LittleEndian.PutUint16(hdr[4:], snapVersion)
	if _, err := cw.Write(hdr[:]); err != nil {
		return cw.n, err
	}
	zw, _ := gzip.NewWriterLevel(cw, gzip.BestCompression)
	bw := bufio.NewWriter(zw)
	keys := make([]uint32, 0, len(s.pages))
	for k := range s.pages {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	var w4 [4]byte
	binary.LittleEndian.PutUint32(w4[:], uint32(len(keys)))
	bw.Write(w4[:])
	for _, k := range keys {
		binary.LittleEndian.PutUint32(w4[:], k)
		bw.Write(w4[:])
		bw.Write(s.pages[k][:])
	}
	if err := bw.Flush(); err != nil {
		return cw.n, err
	}
	err := zw.Close()
	return cw.n, err
}

type countWriter struct {
	w io.Writer
	n int64
}

func (c *countWriter) Write(b []byte) (int, error) {
	n, err := c.w.Write(b)
	c.n += int64(n)
	return n, err
}

// ReadSnapshot parses the file format written by WriteTo.
// It also accepts a "pages" envelope (see envelope.go) and unwraps it.
func ReadSnapshot(r io.Reader) (*Snapshot, error) {
	br0 := bufio.NewReader(r)
	if h, err := br0.Peek(4); err == nil && string(h) == EnvMagic {
		hd, pr, err := OpenEnvelope(br0)
		if err != nil {
			return nil, err
		}
		if hd.Kind != KindPages {
			return nil, fmt.Errorf("mem: fixture kind %q is not a page snapshot", hd.Kind)
		}
		return ReadSnapshot(pr)
	}
	r = br0
	var hdr [6]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return nil, fmt.Errorf("mem: snapshot header: %w", err)
	}
	if string(hdr[:4]) != snapMagic {
		return nil, errors.New("mem: not a snapshot file")
	}
	if v := binary.LittleEndian.Uint16(hdr[4:]); v != snapVersion {
		return nil, fmt.Errorf("mem: unsupported snapshot version %d", v)
	}
	zr, err := gzip.NewReader(r)
	if err != nil {
		return nil, err
	}
	br := bufio.NewReader(zr)
	var w4 [4]byte
	if _, err := io.ReadFull(br, w4[:]); err != nil {
		return nil, err
	}
	n := binary.LittleEndian.Uint32(w4[:])
	s := NewSnapshot()
	for i := uint32(0); i < n; i++ {
		if _, err := io.ReadFull(br, w4[:]); err != nil {
			return nil, err
		}
		addr := binary.LittleEndian.Uint32(w4[:])
		if addr&(PageSize-1) != 0 {
			return nil, fmt.Errorf("mem: unaligned snapshot page 0x%x", addr)
		}
		pg := new([PageSize]byte)
		if _, err := io.ReadFull(br, pg[:]); err != nil {
			return nil, err
		}
		s.pages[addr] = pg
	}
	return s, nil
}

// LoadSnapshot reads a snapshot file.
func LoadSnapshot(path string) (*Snapshot, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return ReadSnapshot(bufio.NewReader(f))
}

// Save writes the snapshot to path.
func (s *Snapshot) Save(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	if _, err := s.WriteTo(f); err != nil {
		f.Close()
		return err
	}
	return f.Close()
}

// EachPage calls fn for every stored page in ascending address order; b is a copy.
func (s *Snapshot) EachPage(fn func(addr uint32, b []byte)) {
	keys := make([]uint32, 0, len(s.pages))
	for k := range s.pages {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	for _, k := range keys {
		fn(k, append([]byte(nil), s.pages[k][:]...))
	}
}
