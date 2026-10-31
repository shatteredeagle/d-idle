package mpq

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"encoding/hex"
	"strings"
	"testing"
)

func encrypt(data []byte, key uint32) { encryptBlock(data, key) }

// bitWriter writes LSB-first bits like the PKWARE format.
type bitWriter struct {
	b   []byte
	cur uint32
	n   uint
}

func (w *bitWriter) put(v int, n uint) {
	for i := uint(0); i < n; i++ {
		w.cur |= uint32(v>>i&1) << w.n
		w.n++
		if w.n == 8 {
			w.b = append(w.b, byte(w.cur))
			w.cur, w.n = 0, 0
		}
	}
}

func (w *bitWriter) bytes() []byte {
	if w.n > 0 {
		return append(w.b, byte(w.cur))
	}
	return w.b
}

// putCode emits the bit string that the decoder resolves to sym.
func (w *bitWriter) putCode(t *testing.T, h *huff, sym int) {
	for l := 1; l <= maxBits; l++ {
		for v := 0; v < 1<<l; v++ {
			br := &bitReader{in: pack(v, l)}
			got, err := br.decode(h)
			if err == nil && got == sym && br.pos*8-int(br.cnt) == l {
				w.put(v, uint(l))
				return
			}
		}
	}
	t.Fatalf("no code for symbol %d", sym)
}

func pack(v, n int) []byte {
	w := &bitWriter{}
	w.put(v, uint(n))
	return append(w.bytes(), 0, 0)
}

// implodeSim builds a DCL stream: coded literals for s[:k], a copy of 8 bytes
// from 4 back, then the end code.
func implodeSim(t *testing.T, lit []byte, coded bool, copies int) []byte {
	w := &bitWriter{}
	if coded {
		w.put(1, 8)
	} else {
		w.put(0, 8)
	}
	w.put(4, 8) // dictionary: 4 extra distance bits
	for _, c := range lit {
		w.put(0, 1)
		if coded {
			w.putCode(t, litCode, int(c))
		} else {
			w.put(int(c), 8)
		}
	}
	for i := 0; i < copies; i++ {
		// copy length 8 (sym 6: base 8, 0 extra), distance 4: code 0, low 3
		w.put(1, 1)
		w.putCode(t, lenCode, 6)
		w.putCode(t, distCode, 0)
		w.put(3, 4)
	}
	// end: length symbol 15, extra 255 -> 264+255 = 519
	w.put(1, 1)
	w.putCode(t, lenCode, 15)
	w.put(255, 8)
	return w.bytes()
}

func TestExplode(t *testing.T) {
	for _, coded := range []bool{false, true} {
		got, err := explode(implodeSim(t, []byte("abcd"), coded, 1), 12)
		if err != nil || string(got) != "abcdabcdabcd" {
			t.Fatalf("coded=%v: %q %v", coded, got, err)
		}
	}
	if _, err := explode([]byte{0, 9}, 0); err == nil {
		t.Error("bad dict should fail")
	}
	if _, err := explode([]byte{0, 4, 0}, 0); err == nil {
		t.Error("truncated should fail")
	}
}

type sfile struct {
	name  string
	flags uint32
	data  []byte // stored bytes (sector table included for sector files)
	fsize uint32
}

func buildArchive(files []sfile, shift uint16, hashSize int) []byte {
	var body bytes.Buffer
	body.Write(make([]byte, 32))
	blocks := make([]blockEntry, len(files))
	for i, f := range files {
		blocks[i] = blockEntry{uint32(body.Len()), uint32(len(f.data)), f.fsize, f.flags | flagExists}
		body.Write(f.data)
	}
	hashes := make([]hashEntry, hashSize)
	for i := range hashes {
		hashes[i] = hashEntry{0xFFFFFFFF, 0xFFFFFFFF, 0xFFFF, 0xFFFF, hashEmpty}
	}
	for i, f := range files {
		h := hashString(f.name, 0) & uint32(hashSize-1)
		for hashes[h].block != hashEmpty {
			h = (h + 1) & uint32(hashSize-1)
		}
		hashes[h] = hashEntry{hashString(f.name, 1), hashString(f.name, 2), 0, 0, uint32(i)}
	}
	hpos := body.Len()
	hb := make([]byte, hashSize*16)
	for i, h := range hashes {
		o := hb[i*16:]
		binary.LittleEndian.PutUint32(o, h.nameA)
		binary.LittleEndian.PutUint32(o[4:], h.nameB)
		binary.LittleEndian.PutUint16(o[8:], h.locale)
		binary.LittleEndian.PutUint16(o[10:], h.platform)
		binary.LittleEndian.PutUint32(o[12:], h.block)
	}
	encrypt(hb, hashString("(hash table)", 3))
	body.Write(hb)
	bpos := body.Len()
	bb := make([]byte, len(blocks)*16)
	for i, b := range blocks {
		o := bb[i*16:]
		binary.LittleEndian.PutUint32(o, b.pos)
		binary.LittleEndian.PutUint32(o[4:], b.csize)
		binary.LittleEndian.PutUint32(o[8:], b.fsize)
		binary.LittleEndian.PutUint32(o[12:], b.flags)
	}
	encrypt(bb, hashString("(block table)", 3))
	body.Write(bb)
	out := body.Bytes()
	copy(out, "MPQ\x1a")
	binary.LittleEndian.PutUint32(out[4:], 32)
	binary.LittleEndian.PutUint32(out[8:], uint32(len(out)))
	binary.LittleEndian.PutUint16(out[12:], 0)
	binary.LittleEndian.PutUint16(out[14:], shift)
	binary.LittleEndian.PutUint32(out[16:], uint32(hpos))
	binary.LittleEndian.PutUint32(out[20:], uint32(bpos))
	binary.LittleEndian.PutUint32(out[24:], uint32(hashSize))
	binary.LittleEndian.PutUint32(out[28:], uint32(len(blocks)))
	return out
}

func zl(b []byte) []byte {
	var buf bytes.Buffer
	w := zlib.NewWriter(&buf)
	w.Write(b)
	w.Close()
	return buf.Bytes()
}

func sectorFile(secs [][]byte) []byte {
	// offsets table then sector data
	n := len(secs)
	tab := make([]byte, (n+1)*4)
	off := uint32(len(tab))
	var data []byte
	for i, s := range secs {
		binary.LittleEndian.PutUint32(tab[i*4:], off)
		off += uint32(len(s))
		data = append(data, s...)
	}
	binary.LittleEndian.PutUint32(tab[n*4:], off)
	return append(tab, data...)
}

func TestSyntheticArchive(t *testing.T) {
	// 3 sectors of 512 bytes (+ a short tail) with mixed methods.
	s0 := bytes.Repeat([]byte("zlib sector "), 60)[:512]
	s1 := make([]byte, 512)
	for i := range s1 {
		s1[i] = byte(i*7 + i>>3) // stored raw
	}
	s2 := bytes.Repeat([]byte("abcd"), 128)
	s3 := bytes.Repeat([]byte("Z"), 100) // short tail sector
	multi := append(append(append(append([]byte{}, s0...), s1...), s2...), s3...)
	secs := [][]byte{
		append([]byte{cmpZlib}, zl(s0)...),
		s1, // incompressible: stored as is, same size as sector
		append([]byte{cmpZlib}, zl(s2)...),
		append([]byte{cmpZlib}, zl(s3)...),
	}
	bz, _ := hex.DecodeString("425a6839314159265359586bdd480000239980400010001264c010200020a551a327a8400c385d97c5a2d974585c2c161617e2ee48a70a120b0d7ba900")
	bzWant := strings.Repeat("hello bzip2 hello bzip2 hello bzip2 ", 4)
	small := bytes.Repeat([]byte("single unit zlib "), 10)
	implWant := "abcdabcdabcd"
	// encrypted, fix-key file
	encPlain := bytes.Repeat([]byte("secret "), 20)
	encBlock := append([]byte{}, encPlain...)
	arc := []sfile{
		{`dir\multi.bin`, flagCompress, sectorFile(secs), uint32(len(multi))},
		{`single.txt`, flagCompress | flagSingleUnit, append([]byte{cmpZlib}, zl(small)...), uint32(len(small))},
		{`bz.txt`, flagCompress | flagSingleUnit, append([]byte{cmpBzip2}, bz...), uint32(len(bzWant))},
		{`impl.txt`, flagCompress | flagSingleUnit, append([]byte{cmpImplode}, implodeSim(t, []byte("abcd"), true, 1)...), uint32(len(implWant))},
		{`raw.txt`, flagSingleUnit, []byte("not compressed"), 14},
		{`empty.txt`, 0, nil, 0},
		{`audio.wav`, flagCompress | flagSingleUnit, []byte{0x41, 1, 2, 3}, 100},
	}
	// FIX_KEY mixes the block position into the key: build once to learn it.
	arc = append(arc, sfile{`dir\secret.txt`, flagEncrypted | flagFixKey | flagSingleUnit, encBlock, uint32(len(encPlain))})
	// first pass to learn position
	tmp := buildArchive(arc, 0, 16)
	a0, err := OpenReaderAt(bytes.NewReader(tmp))
	if err != nil {
		t.Fatal(err)
	}
	blk, _ := a0.find(`dir\secret.txt`)
	key := (hashString("secret.txt", 3) + blk.pos) ^ blk.fsize
	enc := append([]byte{}, encPlain...)
	encrypt(enc, key)
	arc[len(arc)-1].data = enc
	img := buildArchive(arc, 0, 16)

	a, err := OpenReaderAt(bytes.NewReader(img))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		name string
		want string
	}{
		{`dir\multi.bin`, string(multi)},
		{`DIR/MULTI.BIN`, string(multi)}, // case and slash insensitive
		{`single.txt`, string(small)},
		{`bz.txt`, bzWant},
		{`impl.txt`, implWant},
		{`raw.txt`, "not compressed"},
		{`empty.txt`, ""},
		{`dir\secret.txt`, string(encPlain)},
	} {
		got, err := a.ReadFile(c.name)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if string(got) != c.want {
			t.Errorf("%s: got %d bytes, want %d", c.name, len(got), len(c.want))
		}
	}
	if _, err := a.ReadFile(`audio.wav`); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("wav: %v", err)
	}
	if _, err := a.ReadFile(`nope.txt`); err == nil {
		t.Error("expected not found")
	}
	if a.Has(`nope.txt`) || !a.Has(`single.txt`) {
		t.Error("Has")
	}
}

func TestOpenGarbage(t *testing.T) {
	if _, err := OpenReaderAt(bytes.NewReader(make([]byte, 4096))); err == nil {
		t.Error("expected error")
	}
	// truncated: header only
	img := buildArchive([]sfile{{`a`, flagSingleUnit, []byte("x"), 1}}, 0, 4)
	if _, err := OpenReaderAt(bytes.NewReader(img[:40])); err == nil {
		t.Error("expected error for truncated tables")
	}
	// corrupt sector offsets must not panic
	bad := sectorFile([][]byte{{cmpZlib, 1, 2, 3}})
	binary.LittleEndian.PutUint32(bad[4:], 99999)
	img = buildArchive([]sfile{{`a`, flagCompress, bad, 600}}, 0, 4)
	a, err := OpenReaderAt(bytes.NewReader(img))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := a.ReadFile("a"); err == nil {
		t.Error("expected error for corrupt offsets")
	}
}

func TestBuild(t *testing.T) {
	files := map[string][]byte{
		`data\a.txt`: bytes.Repeat([]byte("hello "), 100),
		`b.bin`:      {1, 2, 3},
		`empty`:      {},
	}
	a, err := OpenReaderAt(bytes.NewReader(Build(files)))
	if err != nil {
		t.Fatal(err)
	}
	for n, want := range files {
		got, err := a.ReadFile(n)
		if err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s: %v", n, err)
		}
	}
}
