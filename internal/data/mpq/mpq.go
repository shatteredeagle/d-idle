// Package mpq is a small read-only MPQ (v1/v2 header, 32-bit offsets) reader,
// enough to extract Diablo II data files. Standard library only.
package mpq

import (
	"bytes"
	"compress/bzip2"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
)

const (
	flagImplode    = 0x00000100
	flagCompress   = 0x00000200
	flagEncrypted  = 0x00010000
	flagFixKey     = 0x00020000
	flagSingleUnit = 0x01000000
	flagDeleteMark = 0x02000000
	flagSectorCRC  = 0x04000000
	flagExists     = 0x80000000

	hashEmpty   = 0xFFFFFFFF
	hashDeleted = 0xFFFFFFFE
)

var ErrNotFound = errors.New("mpq: file not found")

var (
	cryptOnce  sync.Once
	cryptTable [0x500]uint32
)

func initCrypt() {
	var seed uint32 = 0x00100001
	for i1 := 0; i1 < 0x100; i1++ {
		i2 := i1
		for k := 0; k < 5; k++ {
			seed = (seed*125 + 3) % 0x2AAAAB
			t1 := (seed & 0xFFFF) << 16
			seed = (seed*125 + 3) % 0x2AAAAB
			t2 := seed & 0xFFFF
			cryptTable[i2] = t1 | t2
			i2 += 0x100
		}
	}
}

func hashString(s string, typ uint32) uint32 {
	cryptOnce.Do(initCrypt)
	var seed1, seed2 uint32 = 0x7FED7FED, 0xEEEEEEEE
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		if c == '/' {
			c = '\\'
		}
		seed1 = cryptTable[typ*0x100+uint32(c)] ^ (seed1 + seed2)
		seed2 = uint32(c) + seed1 + seed2 + seed2<<5 + 3
	}
	return seed1
}

func decrypt(data []byte, key uint32) {
	cryptOnce.Do(initCrypt)
	var seed2 uint32 = 0xEEEEEEEE
	for i := 0; i+4 <= len(data); i += 4 {
		seed2 += cryptTable[0x400+(key&0xFF)]
		v := binary.LittleEndian.Uint32(data[i:]) ^ (key + seed2)
		binary.LittleEndian.PutUint32(data[i:], v)
		key = (^key<<21 + 0x11111111) | key>>11
		seed2 = v + seed2 + seed2<<5 + 3
	}
}

type hashEntry struct {
	nameA, nameB uint32
	locale       uint16
	platform     uint16
	block        uint32
}

type blockEntry struct {
	pos, csize, fsize, flags uint32
}

// Archive is an open MPQ. Safe for concurrent reads.
type Archive struct {
	f          *os.File
	rd         io.ReaderAt
	base       int64 // offset of the MPQ header in the file
	sectorSize int
	hashes     []hashEntry
	blocks     []blockEntry
	path       string
}

// Open opens an MPQ archive file.
func Open(path string) (*Archive, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	a, err := OpenReaderAt(f)
	if err != nil {
		f.Close()
		return nil, err
	}
	a.f = f
	a.path = path
	return a, nil
}

// OpenReaderAt opens an archive from any ReaderAt (the caller keeps ownership).
func OpenReaderAt(r io.ReaderAt) (*Archive, error) {
	a := &Archive{}
	var hdr [32]byte
	found := false
	// The header sits on a 512-byte boundary; D2 archives have it at 0.
	for off := int64(0); off < 1<<24 && !found; off += 512 {
		if _, err := r.ReadAt(hdr[:], off); err != nil {
			break
		}
		switch string(hdr[:4]) {
		case "MPQ\x1a":
			a.base = off
			found = true
		case "MPQ\x1b": // user data header: offset of real header at +8
			ho := int64(binary.LittleEndian.Uint32(hdr[8:]))
			if _, err := r.ReadAt(hdr[:], off+ho); err == nil && string(hdr[:4]) == "MPQ\x1a" {
				a.base = off + ho
				found = true
			}
		}
	}
	if !found {
		return nil, errors.New("mpq: header not found")
	}
	hsize := binary.LittleEndian.Uint32(hdr[4:])
	if hsize < 32 {
		return nil, fmt.Errorf("mpq: header size %d too small", hsize)
	}
	ver := binary.LittleEndian.Uint16(hdr[12:])
	if ver > 1 {
		return nil, fmt.Errorf("mpq: format version %d unsupported", ver)
	}
	a.sectorSize = 512 << binary.LittleEndian.Uint16(hdr[14:])
	hpos := int64(binary.LittleEndian.Uint32(hdr[16:]))
	bpos := int64(binary.LittleEndian.Uint32(hdr[20:]))
	hn := int(binary.LittleEndian.Uint32(hdr[24:]))
	bn := int(binary.LittleEndian.Uint32(hdr[28:]))
	if hn <= 0 || hn > 1<<22 || bn < 0 || bn > 1<<22 {
		return nil, errors.New("mpq: implausible table sizes")
	}

	raw := make([]byte, hn*16)
	if _, err := r.ReadAt(raw, a.base+hpos); err != nil {
		return nil, fmt.Errorf("mpq: hash table: %w", err)
	}
	decrypt(raw, hashString("(hash table)", 3))
	a.hashes = make([]hashEntry, hn)
	for i := range a.hashes {
		b := raw[i*16:]
		a.hashes[i] = hashEntry{
			binary.LittleEndian.Uint32(b), binary.LittleEndian.Uint32(b[4:]),
			binary.LittleEndian.Uint16(b[8:]), binary.LittleEndian.Uint16(b[10:]),
			binary.LittleEndian.Uint32(b[12:]),
		}
	}
	raw = make([]byte, bn*16)
	if _, err := r.ReadAt(raw, a.base+bpos); err != nil {
		return nil, fmt.Errorf("mpq: block table: %w", err)
	}
	decrypt(raw, hashString("(block table)", 3))
	a.blocks = make([]blockEntry, bn)
	for i := range a.blocks {
		b := raw[i*16:]
		a.blocks[i] = blockEntry{
			binary.LittleEndian.Uint32(b), binary.LittleEndian.Uint32(b[4:]),
			binary.LittleEndian.Uint32(b[8:]), binary.LittleEndian.Uint32(b[12:]),
		}
	}
	a.rd = r
	return a, nil
}

// Close closes the underlying file when the archive was opened by Open.
func (a *Archive) Close() error {
	if a.f != nil {
		return a.f.Close()
	}
	return nil
}

func (a *Archive) find(name string) (blockEntry, bool) {
	n := uint32(len(a.hashes))
	start := hashString(name, 0) & (n - 1)
	ha, hb := hashString(name, 1), hashString(name, 2)
	for i := uint32(0); i < n; i++ {
		e := a.hashes[(start+i)&(n-1)]
		if e.block == hashEmpty {
			return blockEntry{}, false
		}
		if e.nameA == ha && e.nameB == hb && e.block != hashDeleted && int(e.block) < len(a.blocks) {
			b := a.blocks[e.block]
			if b.flags&flagExists != 0 && b.flags&flagDeleteMark == 0 {
				return b, true
			}
			return blockEntry{}, false
		}
	}
	return blockEntry{}, false
}

// Has reports whether the archive contains name (path with \ or / separators).
func (a *Archive) Has(name string) bool {
	_, ok := a.find(name)
	return ok
}

// Size returns the uncompressed size of name.
func (a *Archive) Size(name string) (int, bool) {
	b, ok := a.find(name)
	return int(b.fsize), ok
}

// ReadFile extracts and decompresses a file.
func (a *Archive) ReadFile(name string) ([]byte, error) {
	b, ok := a.find(name)
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, name)
	}
	data, err := a.extract(name, b)
	if err != nil {
		return nil, fmt.Errorf("mpq: %s: %w", name, err)
	}
	return data, nil
}

// ListFile returns the names in (listfile), if the archive has one.
func (a *Archive) ListFile() []string {
	data, err := a.ReadFile("(listfile)")
	if err != nil {
		return nil
	}
	var out []string
	for _, l := range strings.Split(string(data), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func (a *Archive) readAt(p []byte, off int64) error {
	_, err := a.rd.ReadAt(p, a.base+off)
	return err
}

func (a *Archive) extract(name string, b blockEntry) ([]byte, error) {
	if b.fsize == 0 {
		return []byte{}, nil
	}
	if b.flags&0x100000 != 0 {
		return nil, errors.New("patch files unsupported")
	}
	var key uint32
	if b.flags&flagEncrypted != 0 {
		base := name
		if i := strings.LastIndexAny(base, `\/`); i >= 0 {
			base = base[i+1:]
		}
		key = hashString(base, 3)
		if b.flags&flagFixKey != 0 {
			key = (key + b.pos) ^ b.fsize
		}
	}
	compressed := b.flags&(flagImplode|flagCompress) != 0
	if b.flags&flagSingleUnit != 0 {
		buf := make([]byte, b.csize)
		if err := a.readAt(buf, int64(b.pos)); err != nil {
			return nil, err
		}
		if key != 0 {
			decrypt(buf, key)
		}
		return a.unpackSector(buf, int(b.fsize), b.flags, compressed)
	}

	ss := a.sectorSize
	nsec := (int(b.fsize) + ss - 1) / ss
	if !compressed {
		// Plain sectors laid out back to back.
		buf := make([]byte, b.fsize)
		if err := a.readAt(buf, int64(b.pos)); err != nil {
			return nil, err
		}
		if key != 0 {
			for i := 0; i < nsec; i++ {
				end := min((i+1)*ss, len(buf))
				decrypt(buf[i*ss:end], key+uint32(i))
			}
		}
		return buf, nil
	}
	nent := nsec + 1
	if b.flags&flagSectorCRC != 0 {
		nent++
	}
	tab := make([]byte, nent*4)
	if err := a.readAt(tab, int64(b.pos)); err != nil {
		return nil, err
	}
	if key != 0 {
		decrypt(tab, key-1)
	}
	offs := make([]uint32, nent)
	for i := range offs {
		offs[i] = binary.LittleEndian.Uint32(tab[i*4:])
	}
	out := make([]byte, 0, b.fsize)
	for i := 0; i < nsec; i++ {
		start, end := offs[i], offs[i+1]
		if end < start || end > b.csize {
			return nil, fmt.Errorf("bad sector offsets %d..%d", start, end)
		}
		buf := make([]byte, end-start)
		if err := a.readAt(buf, int64(b.pos)+int64(start)); err != nil {
			return nil, err
		}
		if key != 0 {
			decrypt(buf, key+uint32(i))
		}
		want := min(ss, int(b.fsize)-i*ss)
		d, err := a.unpackSector(buf, want, b.flags, len(buf) < want)
		if err != nil {
			return nil, fmt.Errorf("sector %d: %w", i, err)
		}
		out = append(out, d...)
	}
	return out, nil
}

// unpackSector decompresses one sector (or a single-unit file). If compressed
// is false the data is returned as is. Sectors whose stored size equals the
// expected size are stored raw, with no mask byte.
func (a *Archive) unpackSector(buf []byte, want int, flags uint32, compressed bool) ([]byte, error) {
	if !compressed || len(buf) >= want {
		if len(buf) < want {
			return nil, io.ErrUnexpectedEOF
		}
		return buf[:want], nil
	}
	if flags&flagCompress != 0 {
		if len(buf) == 0 {
			return nil, io.ErrUnexpectedEOF
		}
		return decompressMask(buf[0], buf[1:], want)
	}
	// IMPLODE flag alone: raw DCL stream, no mask byte.
	out, err := explode(buf, want)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Compression mask bits.
const (
	cmpHuffman = 0x01
	cmpZlib    = 0x02
	cmpImplode = 0x08
	cmpBzip2   = 0x10
	cmpSparse  = 0x20
	cmpADPCM1  = 0x40
	cmpADPCM2  = 0x80
)

// maskHook, when set (tests), sees every compression mask byte decoded.
var maskHook func(mask byte)

func decompressMask(mask byte, in []byte, want int) ([]byte, error) {
	if maskHook != nil {
		maskHook(mask)
	}
	if mask&(cmpHuffman|cmpSparse|cmpADPCM1|cmpADPCM2) != 0 {
		return nil, fmt.Errorf("unsupported compression mask 0x%02x", mask)
	}
	data := in
	// Undo in the order StormLib does: bzip2, implode, zlib.
	var err error
	if mask&cmpBzip2 != 0 {
		if data, err = io.ReadAll(bzip2.NewReader(bytes.NewReader(data))); err != nil {
			return nil, fmt.Errorf("bzip2: %w", err)
		}
	}
	if mask&cmpImplode != 0 {
		if data, err = explode(data, want); err != nil {
			return nil, fmt.Errorf("implode: %w", err)
		}
	}
	if mask&cmpZlib != 0 {
		zr, zerr := zlib.NewReader(bytes.NewReader(data))
		if zerr != nil {
			return nil, fmt.Errorf("zlib: %w", zerr)
		}
		if data, err = io.ReadAll(zr); err != nil {
			return nil, fmt.Errorf("zlib: %w", err)
		}
	}
	if len(data) != want {
		return nil, fmt.Errorf("size mismatch: got %d want %d", len(data), want)
	}
	return data, nil
}
