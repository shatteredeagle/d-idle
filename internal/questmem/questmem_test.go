package questmem

import (
	"encoding/binary"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/mem"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/saves"
)

// regSrc adds the region list that a process would get from /proc/<pid>/maps.
type regSrc struct {
	mem.Source
	regs []d2.Region
}

func (r *regSrc) Regions() []d2.Region { return r.regs }

func regions(spans ...uint32) []d2.Region {
	var out []d2.Region
	for i := 0; i+1 < len(spans); i += 2 {
		out = append(out, d2.Region{Lo: spans[i], Hi: spans[i+1]})
	}
	return out
}

func fixed(s *Source) *Source {
	t0 := time.Unix(1000, 0)
	s.Now = func() time.Time { t0 = t0.Add(time.Second); return t0 } // every read is older than MaxAge
	return s
}

// The recorded fixture (cmd/d2qmem record=, Multi on i5, 2026-10-28) holds a character whose save was set to
// Normal sisters 0x1c, Den 0x0c (a real Akara talk + leaving town wrote it), Nightmare den 0x0c and tools 0x8001,
// Hell radament 0x1001 and siege 0x0004, Normal waypoints Rogue Encampment + Cold Plains (real touch).
func fixture(t *testing.T) *Source {
	t.Helper()
	snap, err := mem.LoadSnapshot("testdata/quests-i5.snap")
	if err != nil {
		t.Fatal(err)
	}
	src := &regSrc{Source: snap, regs: regions(0x3a42000, 0x3a43000)}
	return fixed(New(d2.New(src)))
}

func TestFixtureWords(t *testing.T) {
	s := fixture(t)
	want := []struct {
		diff int
		id   string
		w    uint16
	}{
		{0, "den_of_evil", 0x000c}, {0, "sisters_burial_grounds", 0x001c}, {0, "act1_intro", 0},
		{1, "den_of_evil", 0x000c}, {1, "tools_of_the_trade", 0x8001},
		{2, "radament", 0x1001}, {2, "siege_harrogath", 0x0004},
	}
	for _, c := range want {
		if w, ok := s.Word(c.diff, c.id); !ok || w != c.w {
			t.Errorf("Word(%d,%s) = %#x,%v want %#x", c.diff, c.id, w, ok, c.w)
		}
	}
	if _, ok := s.Word(0, "no_such_quest"); ok {
		t.Error("unknown quest answered")
	}
	if _, ok := s.Word(3, "den_of_evil"); ok {
		t.Error("difficulty 3 answered")
	}
	if s.Scans() != 1 {
		t.Errorf("scans %d", s.Scans())
	}
}

func TestFixtureWaypoints(t *testing.T) {
	s := fixture(t)
	for _, c := range []struct {
		diff, area int
		on         bool
	}{{0, 1, true}, {0, 3, true}, {0, 4, false}, {1, 1, true}, {1, 3, false}, {2, 3, false}} {
		if on, known := s.Waypoint(c.diff, c.area); !known || on != c.on {
			t.Errorf("Waypoint(%d,%d) = %v,%v", c.diff, c.area, on, known)
		}
	}
	if _, known := s.Waypoint(0, 2); known {
		t.Error("Blood Moor has no waypoint")
	}
}

func TestFixtureThroughReader(t *testing.T) {
	rd := quest.NewReader(fixture(t), quest.SaveSource{P: saves.Progress{}})
	q := rd.Quest(2, "radament")
	if q.Status != quest.Done || q.Evidence != quest.EvMemory || !rd.Skippable(2, "radament") {
		t.Errorf("%+v", q)
	}
	if q := rd.Quest(0, "den_of_evil"); q.Status != quest.Started {
		t.Errorf("den %+v", q)
	}
	if q := rd.Quest(0, "forgotten_tower"); q.Status != quest.Needed || q.Evidence != quest.EvMemory {
		t.Errorf("tower %+v", q)
	}
}

// Index tables come from internal/saves; the checks pin the live-verified positions.
func TestIndexTables(t *testing.T) {
	for id, idx := range map[string]int{"den_of_evil": 1, "sisters_burial_grounds": 2, "golden_bird": 20, "siege_harrogath": 35, "eve_of_destruction": 40} {
		if got, ok := QuestIndex(id); !ok || got != idx {
			t.Errorf("QuestIndex(%s) = %d,%v want %d", id, got, ok, idx)
		}
	}
	if len(questIdx) < 35 || len(wpBit) != 39 {
		t.Errorf("%d quests, %d waypoints", len(questIdx), len(wpBit))
	}
	if b, ok := WaypointBit(3); !ok || b != 1 {
		t.Errorf("Cold Plains bit %d", b)
	}
}

// ---- hand-built memory ----

const (
	client = 0x100000
	srvA   = 0x200000
	srvB   = 0x300000
)

type world struct {
	f *mem.Fake
	// where the player is: inGame false leaves the menu canary
}

func put32(f *mem.Fake, a uint32, v ...uint32) { f.PutU32(a, v...) }

// The Fake serves a read from one segment only, so each unit lives in one zeroed page that is patched with
// WriteAt (a region scan then reads the page in one go, like a process mapping).
func mustW(f *mem.Fake, a uint32, b []byte) {
	if err := f.WriteAt(b, a); err != nil {
		panic(err)
	}
}

func w32(f *mem.Fake, a uint32, v ...uint32) {
	b := make([]byte, 4*len(v))
	for i, x := range v {
		binary.LittleEndian.PutUint32(b[4*i:], x)
	}
	mustW(f, a, b)
}

// buildUnit writes a player UnitAny at base with its stat list and PlayerData; qw/wp feed the quest pointers
// (client copies get zero buffers, like the real client unit).
func buildUnit(f *mem.Fake, base uint32, qw [3][48]uint16, wp [3]uint64, extended bool) {
	f.Put(base, make([]byte, 0x1000))
	u := make([]byte, 0x100)
	binary.LittleEndian.PutUint32(u[0x04:], 1)          // Sorceress
	binary.LittleEndian.PutUint32(u[0x0C:], 1)          // id
	binary.LittleEndian.PutUint32(u[0x14:], base+0x400) // PlayerData
	binary.LittleEndian.PutUint32(u[0x2C:], base+0x500) // path
	binary.LittleEndian.PutUint32(u[0x5C:], base+0x600) // stat list
	mustW(f, base, u)
	sl := make([]byte, 0x60)
	if extended {
		binary.LittleEndian.PutUint32(sl[0x10:], 0x80000000)
	}
	mustW(f, base+0x600, sl)
	pd := make([]byte, 0x40)
	copy(pd, "Multi")
	for d := 0; d < 3; d++ {
		qi := base + 0x700 + uint32(d)*0x20
		buf := base + 0x800 + uint32(d)*0x80
		binary.LittleEndian.PutUint32(pd[0x10+4*d:], qi)
		w32(f, qi, buf, 0x300)
		b := make([]byte, 0x80)
		for i, w := range qw[d] {
			binary.LittleEndian.PutUint16(b[2*i:], w)
		}
		mustW(f, buf, b)
		wa := base + 0xA00 + uint32(d)*0x10
		binary.LittleEndian.PutUint32(pd[0x1C+4*d:], wa)
		w := make([]byte, 0x10)
		w[0], w[1] = 2, 1
		for i := 0; i < 5; i++ {
			w[2+i] = byte(wp[d] >> (8 * i))
		}
		mustW(f, wa, w)
	}
	mustW(f, base+0x400, pd)
}

// buildWorld: in game (canary + area chain, area 1) with the client unit at `client` and one server unit.
func buildWorld(srv uint32, qw [3][48]uint16, wp [3]uint64) *mem.Fake {
	f := mem.NewFake()
	put32(f, mem.Base+0x3A6A70, client)
	put32(f, mem.Base+0x3D55BC, 0)
	var empty [3][48]uint16
	buildUnit(f, client, empty, [3]uint64{1, 1, 1}, true)
	w32(f, client+0x500+0x1C, client+0xC00) // path -> room1
	w32(f, client+0xC00+0x10, client+0xC40) // room1 -> room2
	w32(f, client+0xC40+0x58, client+0xC80) // room2 -> level
	w32(f, client+0xC80+0x1D0, 1)           // levelNo
	buildUnit(f, srv, qw, wp, true)
	return f
}

func world1(srv uint32) (*mem.Fake, [3][48]uint16) {
	var qw [3][48]uint16
	qw[0][1], qw[1][3], qw[2][9] = 0x000c, 0x8001, 0x0001
	return buildWorld(srv, qw, [3]uint64{0x3, 0x1, 0}), qw
}

func TestHandBuilt(t *testing.T) {
	f, _ := world1(srvA)
	src := &regSrc{Source: f, regs: regions(client, client+0x1000, srvA, srvA+0x1000)}
	s := fixed(New(d2.New(src)))
	if w, ok := s.Word(0, "den_of_evil"); !ok || w != 0x000c {
		t.Fatalf("den %#x %v", w, ok)
	}
	if w, ok := s.Word(1, "tools_of_the_trade"); !ok || w != 0x8001 {
		t.Fatalf("tools %#x %v", w, ok)
	}
	if w, ok := s.Word(2, "radament"); !ok || w != 1 {
		t.Fatalf("radament %#x %v", w, ok)
	}
	if on, _ := s.Waypoint(0, 3); !on {
		t.Fatal("cold plains")
	}
	// the client copy (empty words) must never be taken for the server unit
	if s.unit != srvA {
		t.Fatalf("unit %#x", s.unit)
	}
	// cached: no further scan while the unit stays valid; a live word change shows up on the next read
	f.WriteAt([]byte{0x01, 0x00}, srvA+0x800+2) // difficulty 0 word 1 := 1 (Den done)
	if w, _ := s.Word(0, "den_of_evil"); w != 1 {
		t.Fatalf("after change %#x", w)
	}
	if s.Scans() != 1 {
		t.Fatalf("scans %d", s.Scans())
	}
}

func TestMaxAgeCache(t *testing.T) {
	f, _ := world1(srvA)
	src := &regSrc{Source: f, regs: regions(client, client+0x1000, srvA, srvA+0x1000)}
	s := New(d2.New(src))
	now := time.Unix(5000, 0)
	s.Now = func() time.Time { return now }
	s.Word(0, "den_of_evil")
	f.WriteAt([]byte{0x01, 0x00}, srvA+0x800+2)
	if w, _ := s.Word(0, "den_of_evil"); w != 0x000c {
		t.Fatalf("expected cached value, got %#x", w)
	}
	now = now.Add(MaxAge + time.Millisecond)
	if w, _ := s.Word(0, "den_of_evil"); w != 1 {
		t.Fatalf("expected fresh value, got %#x", w)
	}
}

// A new game or a reallocated unit: the cached address no longer validates, the Source rescans and finds the
// unit at its new place; the old copy is left behind with a wrong id.
func TestStaleUnitRescan(t *testing.T) {
	f, _ := world1(srvA)
	src := &regSrc{Source: f, regs: regions(client, client+0x1000, srvA, srvA+0x1000, srvB, srvB+0x1000)}
	s := fixed(New(d2.New(src)))
	if _, ok := s.Word(0, "den_of_evil"); !ok || s.Scans() != 1 {
		t.Fatal("first read")
	}
	var qw [3][48]uint16
	qw[0][1] = 0x0010
	buildUnit(f, srvB, qw, [3]uint64{1, 0, 0}, true)
	f.WriteAt([]byte{9, 0, 0, 0}, srvA+0x0C) // old unit's id changes (freed and reused)
	if w, ok := s.Word(0, "den_of_evil"); !ok || w != 0x0010 {
		t.Fatalf("after move %#x %v", w, ok)
	}
	if s.Scans() != 2 || s.unit != srvB {
		t.Fatalf("scans %d unit %#x", s.Scans(), s.unit)
	}
}

// Torn pointer chain behind a still-valid unit (buffers freed): fall back to a rescan, then to "unknown".
func TestBrokenChain(t *testing.T) {
	f, _ := world1(srvA)
	src := &regSrc{Source: f, regs: regions(client, client+0x1000, srvA, srvA+0x1000)}
	s := fixed(New(d2.New(src)))
	if _, ok := s.Word(0, "den_of_evil"); !ok {
		t.Fatal("first read")
	}
	f.WriteAt([]byte{0, 0, 0, 0}, srvA+0x700) // QuestInfo.buf := NULL
	if _, ok := s.Word(0, "den_of_evil"); ok {
		t.Fatal("answered from a null buffer")
	}
	if _, err := s.Progress(); err == nil {
		t.Fatal("no error")
	}
}

func TestNotInGame(t *testing.T) {
	f, _ := world1(srvA)
	src := &regSrc{Source: f, regs: regions(client, client+0x1000, srvA, srvA+0x1000)}
	s := fixed(New(d2.New(src)))
	if _, ok := s.Word(0, "den_of_evil"); !ok {
		t.Fatal("in game")
	}
	// main menu: no player unit, a first control
	put32(f, mem.Base+0x3A6A70, 0)
	put32(f, mem.Base+0x3D55BC, 0x1234)
	if _, ok := s.Word(0, "den_of_evil"); ok {
		t.Fatal("answered in the menu")
	}
	if _, err := s.Progress(); !errors.Is(err, d2.ErrNoServerPlayer) {
		t.Fatalf("err %v", err)
	}
	// loading screen: player pointer set but pUpdateUnit pending
	put32(f, mem.Base+0x3A6A70, client)
	put32(f, mem.Base+0x3D55BC, 0)
	w32(f, client+0x70, 0x1)
	if _, ok := s.Word(0, "den_of_evil"); ok {
		t.Fatal("answered while loading")
	}
	// back in game: reads again (the unit address is rescanned after a not-in-game spell)
	w32(f, client+0x70, 0)
	if w, ok := s.Word(0, "den_of_evil"); !ok || w != 0x000c {
		t.Fatalf("after load %#x %v", w, ok)
	}
	if s.Scans() != 2 {
		t.Fatalf("scans %d", s.Scans())
	}
}

func TestNoServerCopy(t *testing.T) {
	f, _ := world1(srvA)
	src := &regSrc{Source: f, regs: regions(client, client+0x1000)} // server unit outside the scan
	s := fixed(New(d2.New(src)))
	if _, err := s.Progress(); !errors.Is(err, d2.ErrNoServerPlayer) {
		t.Fatalf("err %v", err)
	}
}

func TestIsASource(t *testing.T) {
	var _ quest.Source = New(d2.New(mem.NewFake()))
	if New(nil).Evidence() != quest.EvMemory {
		t.Fatal("evidence")
	}
}
