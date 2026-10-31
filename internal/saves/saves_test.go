package saves

import (
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/cheat/d2s"
)

// build makes a minimal valid v96 file by hand (no real save data). Offsets are the ones in
// internal/cheat/d2s (V on real files).
func build(name string, mapID uint32, edit func(h []byte)) []byte {
	h := make([]byte, 0x2FD)
	binary.LittleEndian.PutUint32(h[0:], d2s.Signature)
	binary.LittleEndian.PutUint32(h[4:], d2s.Version)
	copy(h[0x14:], name)
	h[0x24] = d2s.FlagExpansion
	h[0x28] = 1
	h[0x2B] = 1
	h[0xA8] = 0x80
	binary.LittleEndian.PutUint32(h[0xAB:], mapID)
	copy(h[0x14F:], "Woo!")
	copy(h[0x279:], "WS")
	copy(h[0x279+8:], []byte{2, 1, 1}) // normal: Rogue Encampment waypoint (V on fresh chars)
	copy(h[0x2CA:], "w4")
	if edit != nil {
		edit(h)
	}
	b := append(h, 'g', 'f', 0xFF, 0x01) // empty stat block: 9 bits of 0x1FF
	b = append(b, 'i', 'f')
	b = append(b, make([]byte, 30)...)
	b = append(b, 'J', 'M', 0, 0)
	binary.LittleEndian.PutUint32(b[8:], uint32(len(b)))
	binary.LittleEndian.PutUint32(b[0xC:], d2s.Checksum(b))
	return b
}

func setQ(h []byte, diff, word int, v uint16) {
	binary.LittleEndian.PutUint16(h[0x14F+10+diff*96+word*2:], v)
}

func TestVerify(t *testing.T) {
	b := build("Fix", 7, nil)
	if _, err := Verify(b); err != nil {
		t.Fatal(err)
	}
	bad := append([]byte(nil), b...)
	bad[0x30] ^= 1
	if _, err := Verify(bad); !errors.Is(err, d2s.ErrChecksum) {
		t.Fatalf("flipped byte: %v", err)
	}
	if _, err := Verify(b[:len(b)-3]); err == nil {
		t.Fatal("truncated file passed")
	}
	sig := append([]byte(nil), b...)
	sig[0] = 0
	if _, err := Verify(sig); err == nil {
		t.Fatal("bad signature passed")
	}
}

func TestProgress(t *testing.T) {
	b := build("Fix", 0x08ec30b2, func(h []byte) {
		setQ(h, 0, 1, 0x0c)    // den: started + left town
		setQ(h, 0, 9, 0x0001)  // radament reward granted
		setQ(h, 0, 40, 0x8000) // eve of destruction completed before -> nightmare unlocked
		h[0xA9] = 0x84         // nightmare last played, act V
		h[0x279+8+24] = 2
		h[0x279+8+24+1] = 1
		h[0x279+8+24+2] = 0x03 // nightmare: bits 0,1
	})
	p, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if p.MapID != 0x08ec30b2 || p.Name != "Fix" || !p.Expansion {
		t.Fatalf("%+v", p)
	}
	n := p.Difficulty[0]
	if q := n.Quest("den_of_evil"); !q.Started || q.Done || q.Word != 0x0c {
		t.Fatalf("den %+v", q)
	}
	if !n.Quest("radament").Done || n.Quest("lam_esen").Done {
		t.Fatal("reward quests")
	}
	if got := n.RewardsEarned(); len(got) != 1 || got[0] != "radament" {
		t.Fatalf("rewards %v", got)
	}
	if a := n.Active(); len(a) != 1 || a[0].Name != "Rogue Encampment" || a[0].Level != 1 {
		t.Fatalf("wp %+v", a)
	}
	if !n.Unlocked || !p.Difficulty[1].Unlocked || p.Difficulty[2].Unlocked {
		t.Fatalf("unlocked %v %v %v", n.Unlocked, p.Difficulty[1].Unlocked, p.Difficulty[2].Unlocked)
	}
	nm := p.Difficulty[1]
	if !nm.LastPlayed || nm.Act != 5 || len(nm.Active()) != 2 || nm.Active()[1].Name != "Cold Plains" {
		t.Fatalf("nm %+v", nm)
	}
	if len(waypointTable) != 39 {
		t.Fatal("table")
	}
	seen := map[int]bool{}
	for _, w := range waypointTable {
		if seen[w.Level] {
			t.Fatalf("dup level %d", w.Level)
		}
		seen[w.Level] = true
	}
	words := map[int]bool{}
	for _, q := range questTable {
		if words[q.Word] || q.Word > 47 {
			t.Fatalf("word clash %s", q.ID)
		}
		words[q.Word] = true
	}
	for _, id := range RewardQuests {
		if n.Quest(id).ID == "" {
			t.Fatalf("reward quest %s not in table", id)
		}
	}
	if p2, _ := Decode(build("Fix", 1, func(h []byte) { h[0x25] = 5 })); !p2.Difficulty[1].Unlocked {
		t.Fatal("progression byte 5 should unlock nightmare")
	}
}

func TestMapFileSeed(t *testing.T) {
	// Layout V on Botty.map: 0c000000 01000000 <seed> + zeros.
	b := []byte{0x0c, 0, 0, 0, 1, 0, 0, 0, 0xb2, 0x30, 0xec, 0x08, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	if s, err := MapFileSeed(b); err != nil || s != 0x08ec30b2 {
		t.Fatal(s, err)
	}
}

type env struct{ save, bak string }

func newEnv(t *testing.T, name string, d2sBytes []byte) env {
	e := env{t.TempDir(), t.TempDir()}
	for ext, b := range map[string][]byte{"d2s": d2sBytes, "key": []byte("key"), "map": []byte("map-data-1234"), "ma0": []byte("ma0"), "ma1": []byte("ma1")} {
		must(t, os.WriteFile(filepath.Join(e.save, name+"."+ext), b, 0o644))
	}
	must(t, os.WriteFile(filepath.Join(e.save, "Other.d2s"), []byte("other"), 0o644))
	return e
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func TestSnapshotGateAndCopy(t *testing.T) {
	e := newEnv(t, "Fix", build("Fix", 1, nil))
	if _, err := Snapshot(e.save, e.bak, "Fix", Options{}, t0); !errors.Is(err, ErrNoGameState) {
		t.Fatalf("no state: %v", err)
	}
	in := GameFunc(func() (bool, error) { return true, nil })
	if _, err := Snapshot(e.save, e.bak, "Fix", Options{Game: in}, t0); !errors.Is(err, ErrInGame) {
		t.Fatalf("in game: %v", err)
	}
	if _, err := Snapshot(e.save, e.bak, "Fix", Options{Game: GameFunc(func() (bool, error) { return false, errors.New("x") })}, t0); err == nil {
		t.Fatal("state error must fail closed")
	}
	// told it is at a menu: allowed even though the probe says in game
	r, err := Snapshot(e.save, e.bak, "Fix", Options{Game: in, AtMenu: true}, t0)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Files) != 5 {
		t.Fatalf("files %v (Other.d2s must not be copied)", r.Files)
	}
	if _, err := os.Stat(filepath.Join(r.Dir, "Fix.ma1")); err != nil {
		t.Fatal(err)
	}
	// identical content: no new snapshot
	r2, err := Snapshot(e.save, e.bak, "Fix", Options{AtMenu: true}, t0.Add(time.Minute))
	if err != nil || !r2.Unchanged || r2.ID != r.ID {
		t.Fatalf("unchanged: %+v %v", r2, err)
	}
	for _, bad := range []string{"", "../x", "a/b", ".hid"} {
		if _, err := Snapshot(e.save, e.bak, bad, Options{AtMenu: true}, t0); !errors.Is(err, ErrBadName) {
			t.Fatalf("name %q: %v", bad, err)
		}
	}
}

func TestSnapshotRefusesCorrupt(t *testing.T) {
	good := build("Fix", 1, nil)
	e := newEnv(t, "Fix", good)
	if _, err := Snapshot(e.save, e.bak, "Fix", Options{AtMenu: true}, t0); err != nil {
		t.Fatal(err)
	}
	bad := append([]byte(nil), good...)
	bad[0x40] ^= 0xff
	must(t, os.WriteFile(filepath.Join(e.save, "Fix.d2s"), bad, 0o644))
	if _, err := Snapshot(e.save, e.bak, "Fix", Options{AtMenu: true}, t0.Add(time.Hour)); err == nil {
		t.Fatal("corrupt source snapshotted")
	}
	if l, _ := List(e.bak, "Fix"); len(l) != 1 {
		t.Fatalf("snapshots %d", len(l))
	}
	// restore brings back the good one and keeps the corrupt file aside
	if _, err := Restore(e.save, e.bak, "Fix", "", Options{AtMenu: true}, t0.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(e.save, "Fix.d2s"))
	if string(got) != string(good) {
		t.Fatal("not restored")
	}
	ents, _ := os.ReadDir(filepath.Join(e.bak, "Fix"))
	found := false
	for _, x := range ents {
		if len(x.Name()) > 9 && x.Name()[:9] == "replaced-" {
			found = true
		}
	}
	if !found {
		t.Fatal("replaced files not kept")
	}
	if _, err := Restore(e.save, e.bak, "Fix", "", Options{Game: GameFunc(func() (bool, error) { return true, nil })}, t0); !errors.Is(err, ErrInGame) {
		t.Fatalf("restore in game: %v", err)
	}
}

func TestRotation(t *testing.T) {
	e := newEnv(t, "Fix", nil)
	o := Options{AtMenu: true, Keep: 3, Days: 5}
	// 4 changes per day over 10 days
	now := t0
	for day := 0; day < 10; day++ {
		for k := 0; k < 4; k++ {
			now = t0.AddDate(0, 0, day).Add(time.Duration(k) * time.Hour)
			must(t, os.WriteFile(filepath.Join(e.save, "Fix.d2s"), build("Fix", uint32(day*10+k+1), nil), 0o644))
			if _, err := Snapshot(e.save, e.bak, "Fix", o, now); err != nil {
				t.Fatal(err)
			}
		}
	}
	l, _ := List(e.bak, "Fix")
	// newest 3 are on the last day (k=3,2,1); plus newest of each of the last 5 days: days 9..5 (cutoff now-5d)
	days := map[string]int{}
	for _, in := range l {
		days[in.Time.Format("20060102")]++
	}
	if len(l) != 3+4 { // last day has 3 kept (newest-of-day is among them), 4 more days, one each
		t.Fatalf("kept %d: %v", len(l), days)
	}
	if l[0].Time.Hour() != 15 {
		t.Fatalf("newest %v", l[0].Time)
	}
	for _, in := range l[3:] {
		if in.Time.Hour() != 15 {
			t.Fatalf("older kept snapshot should be end-of-day: %v", in.Time)
		}
	}
}

func TestSnapshotSameSecondNoClobber(t *testing.T) {
	e := newEnv(t, "Fix", build("Fix", 1, nil))
	a, _ := Snapshot(e.save, e.bak, "Fix", Options{AtMenu: true}, t0)
	must(t, os.WriteFile(filepath.Join(e.save, "Fix.d2s"), build("Fix", 2, nil), 0o644))
	b, err := Snapshot(e.save, e.bak, "Fix", Options{AtMenu: true}, t0)
	if err != nil || a.ID == b.ID {
		t.Fatalf("%v %v %v", a.ID, b.ID, err)
	}
}

// D2S_REAL=<file.d2s or folder with .d2s copies>: read-only check of real copies; skipped otherwise.
func TestRealCopies(t *testing.T) {
	p := os.Getenv("D2S_REAL")
	if p == "" {
		t.Skip("set D2S_REAL to a .d2s copy or a folder of copies")
	}
	files := []string{p}
	if fi, err := os.Stat(p); err == nil && fi.IsDir() {
		files, _ = filepath.Glob(filepath.Join(p, "*.d2s"))
	}
	if len(files) == 0 {
		t.Skip("no .d2s found")
	}
	for _, f := range files {
		pr, err := ProgressFile(f)
		if err != nil {
			t.Errorf("%s: %v", f, err)
			continue
		}
		t.Logf("%s: level %d class %d map 0x%08x progression %d normal act %d wp %d quests-nonzero %d",
			pr.Name, pr.Level, pr.Class, pr.MapID, pr.Progression, pr.Difficulty[0].Act, len(pr.Difficulty[0].Active()), nonzero(pr.Difficulty[0]))
		if mb, err := os.ReadFile(f[:len(f)-4] + ".map"); err == nil {
			s, err := MapFileSeed(mb)
			if err != nil || s != pr.MapID {
				t.Errorf("%s: .map seed 0x%x != header map id 0x%x (%v)", f, s, pr.MapID, err)
			}
		}
	}
}

func nonzero(d Difficulty) (n int) {
	for _, q := range d.Quests {
		if q.Word != 0 {
			n++
		}
	}
	return
}

func TestExpOf(t *testing.T) {
	b := build("Fix", 7, nil)
	f, err := d2s.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if err := f.SetStat(d2s.StatExp, 23300); err != nil {
		t.Fatal(err)
	}
	lvl, exp, err := ExpOf(f.Bytes())
	if err != nil || lvl != 1 || exp != 23300 {
		t.Fatalf("ExpOf = %d %d %v", lvl, exp, err)
	}
	if _, exp, _ := ExpOf(b); exp != 0 {
		t.Fatalf("absent exp stat = %d, want 0", exp)
	}
	bad := f.Bytes()
	bad[0x30] ^= 1
	if _, _, err := ExpOf(bad); err == nil {
		t.Fatal("corrupt save read")
	}
	p := filepath.Join(t.TempDir(), "Fix.d2s")
	os.WriteFile(p, f.Bytes(), 0o600)
	if _, exp, err := ExpFile(p); err != nil || exp != 23300 {
		t.Fatalf("ExpFile = %d %v", exp, err)
	}
}
