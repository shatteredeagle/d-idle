package main

import (
	"bytes"
	"encoding/binary"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/savesafe"
)

// saveImage is the smallest character file image savesafe.Valid accepts (class byte at offset 40).
func saveImage(name string, class, seed byte) []byte {
	b := make([]byte, savesafe.MinSize+16)
	binary.LittleEndian.PutUint32(b, d2s.Signature)
	binary.LittleEndian.PutUint32(b[4:], 96)
	copy(b[20:36], name)
	b[40], b[savesafe.MinSize] = class, seed
	binary.LittleEndian.PutUint32(b[8:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[12:], d2s.Checksum(b))
	return b
}

// The incident path end to end for two classes: a backup exists, the fork leaves a zero-length file, the next start
// puts the hero back before classForStart can read the empty file as "no save", and the verb and notices say so.
func TestStartupRepairBeforeClassForStart(t *testing.T) {
	for _, h := range []struct {
		name  string
		class byte
	}{{"Sorcy", 1}, {"Questy", 3}} {
		t.Run(h.name, func(t *testing.T) {
			home, state := t.TempDir(), t.TempDir()
			dir := d2dSaveFolder(home)
			if err := os.MkdirAll(dir, 0o755); err != nil {
				t.Fatal(err)
			}
			live := filepath.Join(dir, h.name+".d2s")
			img := saveImage(h.name, h.class, 9)
			os.WriteFile(live, img, 0o644)
			if _, err := savesafe.Save(dir, state, h.name, nil, time.Now()); err != nil {
				t.Fatal(err)
			}
			var check bytes.Buffer
			if code, err := opsSave([]string{"check", "--save", dir, "--state", state}, &check); code != 0 || err != nil {
				t.Fatalf("check on a good save: %d %v\n%s", code, err, check.String())
			}
			os.WriteFile(live, nil, 0o644)
			check.Reset()
			if code, _ := opsSave([]string{"check", "--char", h.name, "--save", dir, "--state", state}, &check); code != 1 || !strings.Contains(check.String(), "INVALID: ") || !strings.Contains(check.String(), "backup\t") {
				t.Fatalf("check on a truncated save: %d\n%s", code, check.String())
			}
			if st, _ := os.Stat(live); st.Size() != 0 {
				t.Fatal("check changed the live file")
			}
			var logs bytes.Buffer
			log := slog.New(slog.NewTextHandler(&logs, nil))
			repairSaveAtStart(home, state, h.name, log)
			if got, _ := os.ReadFile(live); !bytes.Equal(got, img) {
				t.Fatal("not restored")
			}
			if !strings.Contains(logs.String(), "save repair: restored "+live+" from ") {
				t.Fatalf("log: %s", logs.String())
			}
			if c := classForStart(int(h.class), home, h.name, slog.New(slog.NewTextHandler(io.Discard, nil))); c != -1 {
				t.Fatalf("classForStart = %d: d2d would create a new hero over the restored one", c)
			}
			if n := saveNotices(); len(n) == 0 || !strings.Contains(n[0], "save repair: restored") {
				t.Fatalf("notices for ops status --why and the viewer: %q", n)
			}
			if m, _ := filepath.Glob(live + ".broken-*"); len(m) != 1 {
				t.Fatalf("broken file kept: %v", m)
			}
		})
	}
	if code, _ := opsSave([]string{"repair"}, io.Discard); code != 2 {
		t.Fatalf("unknown save verb: %d", code)
	}
}
