package saves

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/cheat/d2s"
)

// fixup makes the size and checksum fields agree with b so a corruption reaches the deeper parser.
func fixup(b []byte) {
	if len(b) < 0x10 {
		return
	}
	binary.LittleEndian.PutUint32(b[8:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[0xC:], d2s.Checksum(b))
}

// mustNotPanic decodes b every way the bot does and returns nothing; a panic fails the test.
func decodeAll(t testing.TB, b []byte) {
	t.Helper()
	defer func() {
		if x := recover(); x != nil {
			t.Fatalf("panic on %d-byte save: %v", len(b), x)
		}
	}()
	if f, err := Verify(b); err == nil {
		p := ProgressOf(f)
		_ = p.Difficulty[0].Quest("eve_of_destruction")
		out := f.Bytes()
		if _, err := d2s.Parse(out); err != nil {
			t.Fatalf("re-encoded file does not parse: %v", err)
		}
	}
	Decode(b)
	MapFileSeed(b)
}

func TestSaveCorruptionTable(t *testing.T) {
	base := build("Fuzz", 7, nil)
	rng := rand.New(rand.NewSource(1))
	// every truncation, with and without a repaired header
	for n := 0; n <= len(base); n++ {
		b := append([]byte(nil), base[:n]...)
		decodeAll(t, b)
		fixup(b)
		decodeAll(t, b)
	}
	// byte flips and random garbage tails, header repaired so the stat block parser runs
	for i := 0; i < 20000; i++ {
		b := append([]byte(nil), base...)
		for k := rng.Intn(6); k >= 0; k-- {
			b[rng.Intn(len(b))] = byte(rng.Intn(256))
		}
		if rng.Intn(3) == 0 {
			b = append(b[:0x2FD+rng.Intn(len(b)-0x2FD)], make([]byte, rng.Intn(40))...)
		}
		if rng.Intn(5) > 0 {
			fixup(b)
		}
		decodeAll(t, b)
	}
	// pure noise and empty input
	for i := 0; i < 2000; i++ {
		b := make([]byte, rng.Intn(1200))
		rng.Read(b)
		decodeAll(t, b)
	}
	decodeAll(t, nil)
}

func FuzzDecodeSave(f *testing.F) {
	base := build("Fuzz", 7, nil)
	f.Add(base)
	f.Add(base[:0x2FD])
	f.Add([]byte{})
	f.Fuzz(func(t *testing.T, b []byte) {
		decodeAll(t, b)
		c := append([]byte(nil), b...)
		fixup(c)
		decodeAll(t, c)
	})
}

func TestSnapshotRestoreGarbage(t *testing.T) {
	o := Options{AtMenu: true}
	// hostile names never leave the directory
	for _, n := range []string{"", ".", "..", "../x", `a\b`, "a/b", ".hid", "\x00", "a\x00b"} {
		e := newEnv(t, "Fix", build("Fix", 1, nil))
		if _, err := Snapshot(e.save, e.bak, n, o, t0); err == nil {
			t.Errorf("Snapshot accepted name %q", n)
		}
		if _, err := Restore(e.save, e.bak, n, "", o, t0); err == nil {
			t.Errorf("Restore accepted name %q", n)
		}
		List(e.bak, n)
	}
	// a corrupt .d2s is refused at snapshot time or at restore time and never overwrites the live file
	good := build("Fix", 1, nil)
	e := newEnv(t, "Fix", good)
	if _, err := Snapshot(e.save, e.bak, "Fix", o, t0); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(e.bak, "Fix", t0.Format(stampLayout))
	for _, bad := range [][]byte{nil, good[:100], bytes.Repeat([]byte{0xff}, 4000)} {
		must(t, os.WriteFile(filepath.Join(dir, "Fix.d2s"), bad, 0o644))
		if _, err := Restore(e.save, e.bak, "Fix", "", o, t0.Add(time.Hour)); err == nil {
			t.Fatalf("restored corrupt snapshot of %d bytes", len(bad))
		}
		cur, _ := os.ReadFile(filepath.Join(e.save, "Fix.d2s"))
		if !bytes.Equal(cur, good) {
			t.Fatal("live save modified by a failed restore")
		}
	}
	// junk entries in the backup dir are ignored by List/Prune
	for _, n := range []string{"zzz", "20269999T999999Z", "20260101T000000Z-x"} {
		os.MkdirAll(filepath.Join(e.bak, "Fix", n), 0o755)
	}
	os.WriteFile(filepath.Join(e.bak, "Fix", "file"), []byte("x"), 0o644)
	if _, err := List(e.bak, "Fix"); err != nil {
		t.Fatal(err)
	}
	if _, err := Prune(e.bak, "Fix", o, t0); err != nil {
		t.Fatal(err)
	}
}
