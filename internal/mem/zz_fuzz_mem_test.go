package mem

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"math/rand"
	"testing"
)

func parseAny(t testing.TB, b []byte) {
	t.Helper()
	defer func() {
		if x := recover(); x != nil {
			t.Fatalf("panic on %d bytes: %v", len(b), x)
		}
	}()
	if s, err := ReadSnapshot(bytes.NewReader(b)); err == nil {
		var p [8]byte
		s.ReadAt(p[:], 0x400000)
	}
	if h, r, err := OpenEnvelope(bytes.NewReader(b)); err == nil {
		_ = h
		UnwrapJSONL(r)
	}
}

func gz(b []byte) []byte {
	var out bytes.Buffer
	w := gzip.NewWriter(&out)
	w.Write(b)
	w.Close()
	return out.Bytes()
}

func validSnap() []byte {
	s := NewSnapshot()
	s.Put(0x400000, []byte("hello"))
	s.Put(0x7ff000, bytes.Repeat([]byte{7}, 5000))
	var buf bytes.Buffer
	s.WriteTo(&buf)
	return buf.Bytes()
}

func TestSnapshotFileGarbage(t *testing.T) {
	base := validSnap()
	rng := rand.New(rand.NewSource(5))
	for n := 0; n <= len(base) && n < 200; n++ {
		parseAny(t, base[:n])
	}
	for i := 0; i < 3000; i++ {
		b := append([]byte(nil), base...)
		for k := rng.Intn(5); k >= 0; k-- {
			b[rng.Intn(min(len(b), 120))] = byte(rng.Intn(256))
		}
		parseAny(t, b)
	}
	// a gzip body that claims 4 billion pages but holds none must fail fast without allocating them
	hdr := append([]byte("D2SN"), 1, 0)
	body := make([]byte, 4)
	binary.LittleEndian.PutUint32(body, 0xffffffff)
	parseAny(t, append(append([]byte(nil), hdr...), gz(body)...))
	// envelope with a huge or zero header length, and nested envelopes
	for _, n := range []uint32{0, 1, 1 << 20, 0xffffffff} {
		pre := make([]byte, 10)
		copy(pre, EnvMagic)
		binary.LittleEndian.PutUint16(pre[4:], EnvSchema)
		binary.LittleEndian.PutUint32(pre[6:], n)
		parseAny(t, pre)
		parseAny(t, append(pre, bytes.Repeat([]byte("{"), 100)...))
	}
	parseAny(t, nil)
}

func FuzzReadSnapshot(f *testing.F) {
	f.Add(validSnap())
	f.Add([]byte("D2SN\x01\x00"))
	f.Add([]byte(EnvMagic))
	f.Fuzz(func(t *testing.T, b []byte) { parseAny(t, b) })
}

// Source readers must tolerate any address and length: wrap-around, zero length, overlap.
func TestSourceEdgeReads(t *testing.T) {
	f := NewFake()
	f.Put(0xfffffffc, []byte{1, 2, 3, 4})
	f.Put(0x1000, []byte{9, 9})
	f.Put(0x1001, []byte{8, 8, 8})
	s := NewSnapshot()
	s.Put(0xfffffffc, []byte{1, 2, 3, 4})
	for _, src := range []Source{f, s, NewRecording(f)} {
		for _, a := range []uint32{0, 1, 0xfffffffc, 0xfffffffd, 0xffffffff, 0x1000, 0x1001, 0x7fffffff} {
			for _, n := range []int{0, 1, 3, 4, 5, 4096, 70000} {
				p := make([]byte, n)
				func() {
					defer func() {
						if x := recover(); x != nil {
							t.Fatalf("%T addr 0x%x len %d: panic %v", src, a, n, x)
						}
					}()
					src.ReadAt(p, a)
				}()
			}
			U8(src, a)
			U16(src, a)
			U32(src, a)
			I32(src, a)
		}
	}
	if err := f.WriteAt([]byte{1}, 0x5000); err == nil {
		t.Fatal("write outside any segment must fail")
	}
}
