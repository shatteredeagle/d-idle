package savesafe

import (
	"bytes"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/cheat/d2s"
)

// fixture builds the smallest image Valid accepts: header fields, class byte (offset 40, V: internal/cheat/d2s),
// a body that varies with seed, size and checksum.
func fixture(name string, class byte, seed byte) []byte {
	b := make([]byte, MinSize+40)
	binary.LittleEndian.PutUint32(b, d2s.Signature)
	binary.LittleEndian.PutUint32(b[4:], 96)
	copy(b[20:36], name)
	b[40] = class
	for i := MinSize; i < len(b); i++ {
		b[i] = seed + byte(i)
	}
	binary.LittleEndian.PutUint32(b[offSize:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[offChecksum:], d2s.Checksum(b))
	return b
}

var heroes = []struct {
	name  string
	class byte
}{{"Sorcy", 1}, {"Questy", 3}} // a Sorceress and a Paladin

func plenty(string) (uint64, error) { return 100 << 30, nil }

func write(t *testing.T, path string, b []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestValid(t *testing.T) {
	for _, h := range heroes {
		good := fixture(h.name, h.class, 1)
		if err := Valid(good); err != nil {
			t.Fatalf("%s: %v", h.name, err)
		}
		if f, err := d2s.Parse(good); err == nil && f.Class() != int(h.class) {
			t.Fatalf("%s: class %d", h.name, f.Class())
		}
		bad := map[string][]byte{"empty": nil, "short": good[:700], "truncated": good[:len(good)-1]}
		flip := bytes.Clone(good)
		flip[MinSize+3] ^= 0xff
		bad["flipped"] = flip
		sig := bytes.Clone(good)
		sig[0] = 0
		bad["signature"] = sig
		for k, b := range bad {
			if err := Valid(b); !errors.Is(err, ErrInvalid) {
				t.Errorf("%s %s: got %v", h.name, k, err)
			}
		}
	}
}

func TestSaveAndZeroByteRestore(t *testing.T) {
	for _, h := range heroes {
		t.Run(h.name, func(t *testing.T) {
			save, state := t.TempDir(), t.TempDir()
			live := filepath.Join(save, h.name+".d2s")
			now := time.Unix(1_800_000_000, 0)
			v1 := fixture(h.name, h.class, 1)
			write(t, live, v1)
			write(t, filepath.Join(save, h.name+".key"), []byte("key"))
			write(t, filepath.Join(save, h.name+".ma0"), []byte("automap"))
			write(t, filepath.Join(save, "shared.d2x"), []byte("stash"))
			r, err := Save(save, state, h.name, plenty, now)
			if err != nil || r.Unchanged {
				t.Fatalf("first save: %+v %v", r, err)
			}
			if want := filepath.Join(state, Dir, h.name+".1800000000.d2s"); r.Path != want {
				t.Fatalf("path %s, want %s", r.Path, want)
			}
			if r, err = Save(save, state, h.name, plenty, now.Add(time.Minute)); err != nil || !r.Unchanged {
				t.Fatalf("same bytes must not be copied again: %+v %v", r, err)
			}
			v2 := fixture(h.name, h.class, 2)
			write(t, live, v2)
			if r, err = Save(save, state, h.name, plenty, now.Add(2*time.Minute)); err != nil || r.Unchanged {
				t.Fatalf("second save: %+v %v", r, err)
			}
			newest := r.Path

			// The incident: the fork truncates the file on a full disk.
			write(t, live, nil)
			write(t, filepath.Join(save, h.name+".key"), nil)
			if _, err := Save(save, state, h.name, plenty, now.Add(3*time.Minute)); !errors.Is(err, ErrInvalid) {
				t.Fatalf("a zero-length live file must not be backed up: %v", err)
			}
			if l, _ := List(state, h.name); len(l) != 2 {
				t.Fatalf("backups %d, want 2", len(l))
			}
			rep, ok, err := Repair(save, state, h.name, plenty, now.Add(4*time.Minute))
			if err != nil || !ok {
				t.Fatalf("repair: %v %v", ok, err)
			}
			if rep.From != newest || !strings.HasPrefix(rep.String(), "save repair: restored "+live+" from "+newest) {
				t.Fatalf("repair record %q", rep)
			}
			if got, _ := os.ReadFile(live); !bytes.Equal(got, v2) {
				t.Fatal("the live file is not the newest backup")
			}
			if st, err := os.Stat(rep.Broken); err != nil || st.Size() != 0 || !strings.Contains(rep.Broken, ".d2s.broken-") {
				t.Fatalf("broken file not kept: %q %v", rep.Broken, err)
			}
			if got, _ := os.ReadFile(filepath.Join(save, h.name+".key")); string(got) != "key" {
				t.Fatalf("empty key file not restored: %q", got)
			}
			if got, _ := os.ReadFile(filepath.Join(save, "shared.d2x")); string(got) != "stash" {
				t.Fatalf("a present shared stash must stay: %q", got)
			}
			if n := Notices(state, plenty, now.Add(time.Hour)); len(n) != 1 || !strings.Contains(n[0], "save repair: restored") {
				t.Fatalf("notices %q", n)
			}
			if n := Notices(state, plenty, now.Add(8*24*time.Hour)); len(n) != 0 {
				t.Fatalf("the repair notice must expire: %q", n)
			}
			if _, ok, err := Repair(save, state, h.name, plenty, now.Add(5*time.Minute)); ok || err != nil {
				t.Fatalf("a valid live file must be left alone: %v %v", ok, err)
			}
		})
	}
}

func TestRepairMissingCorruptAndNoBackup(t *testing.T) {
	save, state := t.TempDir(), t.TempDir()
	now := time.Unix(1_800_000_000, 0)
	if _, ok, err := Repair(save, state, "Questy", plenty, now); ok || err != nil {
		t.Fatalf("no save and no backup is a new hero: %v %v", ok, err)
	}
	live := filepath.Join(save, "Questy.d2s")
	write(t, live, []byte("garbage"))
	if _, ok, err := Repair(save, state, "Questy", plenty, now); ok || err == nil {
		t.Fatalf("broken without backup must report: %v %v", ok, err)
	}
	if b, _ := os.ReadFile(live); string(b) != "garbage" {
		t.Fatal("the broken file must stay when there is no backup")
	}
	good := fixture("Questy", 3, 7)
	write(t, live, good)
	if _, err := Save(save, state, "Questy", plenty, now); err != nil {
		t.Fatal(err)
	}
	// A corrupt backup newer than the good one is skipped.
	write(t, filepath.Join(state, Dir, "Questy.1800000500.d2s"), good[:800])
	os.Remove(live)
	rep, ok, err := Repair(save, state, "Questy", plenty, now.Add(time.Hour))
	if err != nil || !ok || rep.Broken != "" || rep.Why != "live file missing" {
		t.Fatalf("missing: %+v %v %v", rep, ok, err)
	}
	if b, _ := os.ReadFile(live); !bytes.Equal(b, good) {
		t.Fatal("not restored from the valid backup")
	}
	corrupt := bytes.Clone(good)
	corrupt[900-140] ^= 1
	write(t, live, corrupt)
	full := func(string) (uint64, error) { return 10 << 20, nil }
	if _, ok, err := Repair(save, state, "Questy", full, now.Add(2*time.Hour)); ok || !errors.Is(err, ErrDiskLow) {
		t.Fatalf("a full disk must refuse the restore: %v %v", ok, err)
	}
	if b, _ := os.ReadFile(live); !bytes.Equal(b, corrupt) {
		t.Fatal("a refused restore must not move the live file")
	}
	if _, ok, err := Repair(save, state, "Questy", plenty, now.Add(3*time.Hour)); !ok || err != nil {
		t.Fatalf("corrupt: %v %v", ok, err)
	}
}

func TestDiskLowRefusal(t *testing.T) {
	save, state := t.TempDir(), t.TempDir()
	write(t, filepath.Join(save, "Sorcy.d2s"), fixture("Sorcy", 1, 1))
	free := uint64(MinFree - 1)
	probe := func(string) (uint64, error) { return free, nil }
	now := time.Unix(1_800_000_000, 0)
	if _, err := Save(save, state, "Sorcy", probe, now); !errors.Is(err, ErrDiskLow) {
		t.Fatalf("got %v", err)
	}
	if l, _ := List(state, "Sorcy"); len(l) != 0 {
		t.Fatalf("nothing may be written: %v", l)
	}
	var logs []string
	g := &Guard{State: state, SaveDir: func() string { return save }, Name: func() string { return "Sorcy" },
		Now: func() time.Time { return now }, Free: probe, Logf: func(msg string, kv ...any) { logs = append(logs, msg) }}
	for i := 0; i < 5; i++ {
		g.Observe(i%2 == 0) // arrivals in town: each one is a backup attempt
	}
	if len(logs) != 1 || logs[0] != "save backup: skipped" || !g.DiskLow() {
		t.Fatalf("the refusal must be logged once: %q low=%t", logs, g.DiskLow())
	}
	if n := Notices(state, probe, now); len(n) != 1 || !strings.Contains(n[0], "no save backups") {
		t.Fatalf("notices %q", n)
	}
	if n := Notices(state, func(string) (uint64, error) { return 1 << 30, nil }, now); len(n) != 1 || strings.Contains(n[0], "no save backups") {
		t.Fatalf("under 2 GB warns only: %q", n)
	}
	free = 1 << 40
	g.Observe(false)
	g.Observe(true)
	if l, _ := List(state, "Sorcy"); len(l) != 1 || g.DiskLow() {
		t.Fatalf("with space back the backup is written: %v", l)
	}
}

func TestGuardCadence(t *testing.T) {
	save, state := t.TempDir(), t.TempDir()
	live := filepath.Join(save, "Questy.d2s")
	now := time.Unix(1_800_000_000, 0)
	g := &Guard{State: state, SaveDir: func() string { return save }, Name: func() string { return "Questy" },
		Now: func() time.Time { return now }, Free: plenty}
	count := func() int { l, _ := List(state, "Questy"); return len(l) }
	g.Observe(false) // no file yet: silent
	write(t, live, fixture("Questy", 3, 1))
	now = now.Add(Every - time.Second)
	g.Observe(false)
	if count() != 0 {
		t.Fatal("not due yet")
	}
	now = now.Add(time.Second)
	g.Observe(false)
	if count() != 1 {
		t.Fatal("due after Every")
	}
	write(t, live, fixture("Questy", 3, 2))
	now = now.Add(time.Minute)
	g.Observe(false)
	if count() != 1 {
		t.Fatal("in the field, not due")
	}
	g.Observe(true)
	if count() != 2 {
		t.Fatal("town arrival must back up")
	}
	write(t, live, fixture("Questy", 3, 3))
	now = now.Add(time.Minute)
	g.Observe(true)
	if count() != 2 {
		t.Fatal("standing in town is not an arrival")
	}
}

func TestRotation(t *testing.T) {
	save, state := t.TempDir(), t.TempDir()
	live := filepath.Join(save, "Sorcy.d2s")
	start := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	now := start
	// Ten days, three saves a day, then a burst of 20 on the last day.
	seed := byte(0)
	step := func(d time.Duration) {
		now = now.Add(d)
		seed++
		write(t, live, fixture("Sorcy", 1, seed))
		write(t, filepath.Join(save, "Sorcy.key"), []byte{seed})
		if _, err := Save(save, state, "Sorcy", plenty, now); err != nil {
			t.Fatal(err)
		}
	}
	for i := 0; i < 30; i++ {
		step(8 * time.Hour)
	}
	for i := 0; i < 20; i++ {
		step(time.Minute)
	}
	l, err := List(state, "Sorcy")
	if err != nil {
		t.Fatal(err)
	}
	days := map[string]int{}
	for i, b := range l {
		if b.Err != nil {
			t.Fatalf("invalid backup kept: %v", b.Err)
		}
		if i < KeepNewest {
			continue
		}
		days[time.Unix(b.Unix, 0).UTC().Format("2006-01-02")]++
	}
	// The newest 12 are all from the burst; beyond them one a day for the days inside the window.
	if len(l) < KeepNewest+KeepDays-2 || len(l) > KeepNewest+KeepDays {
		t.Fatalf("kept %d sets", len(l))
	}
	for d, n := range days {
		if n != 1 {
			t.Errorf("day %s kept %d beyond the newest %d", d, n, KeepNewest)
		}
	}
	if oldest := time.Unix(l[len(l)-1].Unix, 0); now.Sub(oldest) > KeepDays*24*time.Hour {
		t.Errorf("oldest kept set is %s old", now.Sub(oldest))
	}
	ents, _ := os.ReadDir(filepath.Join(state, Dir))
	if len(ents) != 2*len(l) {
		t.Fatalf("side files must rotate with their set: %d files for %d sets", len(ents), len(l))
	}
}

func TestCheckReadOnly(t *testing.T) {
	save, state := t.TempDir(), t.TempDir()
	now := time.Unix(1_800_000_000, 0)
	write(t, filepath.Join(save, "Questy.d2s"), fixture("Questy", 3, 1))
	if _, err := Save(save, state, "Questy", plenty, now); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(save, "Questy.d2s"), nil)
	r, err := Check(save, state, "Questy", plenty, now)
	if err != nil || r.OK() || len(r.Backups) != 1 || r.Backups[0].Err != nil {
		t.Fatalf("%+v %v", r, err)
	}
	if st, _ := os.Stat(filepath.Join(save, "Questy.d2s")); st.Size() != 0 {
		t.Fatal("check must not repair")
	}
	if _, err := Check(save, state, "../x", plenty, now); err == nil {
		t.Fatal("bad name accepted")
	}
}
