package cheat

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/cheat/d2s"
	"d2lod-bot/internal/mem"
)

type fakeRules struct{}

func (fakeRules) ExpToLevel(l int) int64 { return int64(l-1) * 1000 }
func (fakeRules) Gains(int) Gains {
	return Gains{LifePerLevel: 4, ManaPerLevel: 8, StaminaPerLevel: 4, LifePerVit: 8, StaminaPerVit: 4, ManaPerEnergy: 8, StatPerLevel: 5}
}

func testEnv(t *testing.T) *Env {
	dir := t.TempDir()
	return &Env{
		Ledger:    NewLedger(filepath.Join(dir, "state", "cheats.jsonl")),
		Instance:  "i5",
		Now:       func() time.Time { return time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC) },
		SaveDir:   filepath.Join(dir, "saves"),
		BackupDir: filepath.Join(dir, "backups"),
		DataDir:   filepath.Join(dir, "Data"),
		InGame:    func() (bool, error) { return false, nil },
		Rules:     func() (Rules, error) { return fakeRules{}, nil },
		Open:      func() (mem.Writer, func(), error) { return nil, nil, errors.New("no game in tests") },
	}
}

// writeSave builds a synthetic v96 save with d2s's own writer (no real save data is ever committed).
func writeSave(t *testing.T, env *Env, name string) string {
	t.Helper()
	os.MkdirAll(env.SaveDir, 0o755)
	f := synthFile(t, name)
	p := filepath.Join(env.SaveDir, name+".d2s")
	if err := os.WriteFile(p, f, 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestReasonRequired(t *testing.T) {
	env := testEnv(t)
	_, err := env.Do(context.Background(), Request{Kind: KindHeal})
	if !errors.Is(err, ErrNoReason) {
		t.Fatalf("err %v", err)
	}
	es, _ := env.Ledger.List(Filter{})
	if len(es) != 1 || es[0].Success || es[0].Reason != "(none)" {
		t.Fatalf("refusal not recorded: %+v", es)
	}
}

func TestSaveEditEndToEnd(t *testing.T) {
	env := testEnv(t)
	p := writeSave(t, env, "Multi")
	orig, _ := os.ReadFile(p)
	res, err := env.Do(context.Background(), Request{Kind: KindSaveEdit, Actor: "tester", Reason: "unit test",
		Params: map[string]any{"character": "Multi", "level": 25, "gold": 5000, "waypoints": 1}})
	if err != nil {
		t.Fatal(err)
	}
	if res.Entry.Backup == "" {
		t.Fatal("no backup")
	}
	if b, _ := os.ReadFile(res.Entry.Backup); string(b) != string(orig) {
		t.Fatal("backup differs from the original")
	}
	b, _ := os.ReadFile(p)
	f, err := d2s.Parse(b)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := f.Stat(d2s.StatLevel); v != 25 || f.Level() != 25 {
		t.Fatal("level")
	}
	if v, _ := f.Stat(d2s.StatExp); v != 24000 {
		t.Fatalf("exp %d", v)
	}
	if v, _ := f.Stat(d2s.StatPoints); v != 120 { // 24 levels x 5
		t.Fatalf("points %d", v)
	}
	if v, _ := f.Stat(d2s.StatSkillPoints); v != 24 {
		t.Fatalf("skill points %d", v)
	}
	if v, _ := f.Stat(d2s.StatMaxLife); v != 10240+24*256 { // 4 quarter points = 1 per level
		t.Fatalf("max life %d", v)
	}
	es, _ := env.Ledger.List(Filter{Kind: KindSaveEdit, OnlySuccess: true})
	if len(es) != 1 || es[0].Actor != "tester" || es[0].Character != "Multi" || es[0].Before["level"] == es[0].After["level"] {
		t.Fatalf("ledger %+v", es)
	}
}

func TestSavePolicy(t *testing.T) {
	env := testEnv(t)
	writeSave(t, env, "Stranger")
	if _, err := env.Do(context.Background(), Request{Kind: KindSaveEdit, Reason: "x", Params: map[string]any{"character": "Stranger", "gold": 1}}); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("allowlist: %v", err)
	}
	writeSave(t, env, "Botty")
	env.InGame = func() (bool, error) { return true, nil }
	if _, err := env.Do(context.Background(), Request{Kind: KindSaveEdit, Reason: "x", Params: map[string]any{"character": "Botty", "gold": 1}}); !errors.Is(err, ErrGameRunning) {
		t.Fatalf("running game: %v", err)
	}
	env.InGame = func() (bool, error) { return false, nil }
	env.BackupDir = "/proc/cannot-create-here" // backup failure must stop the edit
	p := filepath.Join(env.SaveDir, "Botty.d2s")
	before, _ := os.ReadFile(p)
	if _, err := env.Do(context.Background(), Request{Kind: KindSaveEdit, Reason: "x", Params: map[string]any{"character": "Botty", "gold": 1}}); err == nil {
		t.Fatal("edit went ahead without a backup")
	}
	if after, _ := os.ReadFile(p); string(after) != string(before) {
		t.Fatal("save changed although backup failed")
	}
	es, _ := env.Ledger.List(Filter{})
	if len(es) != 3 {
		t.Fatalf("want 3 ledger entries (all refusals recorded), got %d", len(es))
	}
	for _, e := range es {
		if e.Success {
			t.Fatalf("refusal logged as success: %+v", e)
		}
	}
}

func TestLedgerFilterSummaryCheated(t *testing.T) {
	l := NewLedger(filepath.Join(t.TempDir(), "x", "l.jsonl"))
	t0 := time.Date(2026, 10, 28, 10, 0, 0, 0, time.UTC)
	add := func(dt time.Duration, char string, k Kind, ok bool) {
		if err := l.Append(Entry{Time: t0.Add(dt), Actor: "a", Character: char, Kind: k, Reason: "r", Success: ok}); err != nil {
			t.Fatal(err)
		}
	}
	add(0, "Multi", KindHeal, true)
	add(time.Hour, "Botty", KindGold, false)
	add(2*time.Hour, "Multi", KindLevel, true)
	if err := l.Append(Entry{}); err == nil {
		t.Fatal("entry without reason accepted")
	}
	got, _ := l.List(Filter{Character: "multi"})
	if len(got) != 2 {
		t.Fatalf("filter: %d", len(got))
	}
	got, _ = l.List(Filter{Since: t0.Add(30 * time.Minute), OnlySuccess: true})
	if len(got) != 1 || got[0].Kind != KindLevel {
		t.Fatalf("since+success: %+v", got)
	}
	all, _ := l.List(Filter{})
	if s := Summarize(all); s.Total != 3 || s.Failed != 1 || s.ByKind[KindHeal] != 1 || s.ByCharacter["Multi"] != 2 {
		t.Fatalf("summary %+v", s)
	}
	if !l.Cheated(t0.Add(90*time.Minute), "Multi") || l.Cheated(t0.Add(3*time.Hour), "") || l.Cheated(t0.Add(30*time.Minute), "Botty") {
		t.Fatal("Cheated")
	}
	os.WriteFile(l.Path, append(mustRead(t, l.Path), []byte("not json\n")...), 0o644)
	if _, err := l.List(Filter{}); err == nil {
		t.Fatal("corrupt line not reported")
	}
}

func mustRead(t *testing.T, p string) []byte {
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// fakeGame lays out a unit with a stat list (base 4 stats/cap 6, merged 4/cap 5) in a mem.Fake.
func fakeGame(t *testing.T) (*game, *mem.Fake) {
	f := mem.NewFake()
	const unit, sl, base, merged = 0x1000, 0x2000, 0x3000, 0x4000
	f.PutU32(unit+uoStats, sl)
	f.PutU32(sl+slFlags, slFlagEx)
	put := func(addr uint32, ents [][3]uint32) {
		b := make([]byte, 8*8)
		for i, e := range ents {
			binary.LittleEndian.PutUint16(b[i*8:], uint16(e[1]))
			binary.LittleEndian.PutUint16(b[i*8+2:], uint16(e[0]))
			binary.LittleEndian.PutUint32(b[i*8+4:], e[2])
		}
		f.Put(addr, b)
	}
	ents := [][3]uint32{{6, 0, 1000}, {7, 0, 2000}, {12, 0, 5}, {172, 0, 1}}
	put(base, ents)
	put(merged, ents)
	f.Put(sl+slVec, []byte{0, 0x30, 0, 0, 4, 0, 6, 0}) // vec 0x3000, count 4, cap 6
	f.Put(sl+slVecEx, []byte{0, 0x40, 0, 0, 4, 0, 5, 0})
	return &game{w: f, client: unit, server: unit}, f
}

func TestStatSetAndInsert(t *testing.T) {
	g, _ := fakeGame(t)
	if err := g.set(UnitServer, VecBoth, 6, 77); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{VecBase, VecMerged} {
		a, _ := g.slot(g.server, v, 6, 0)
		if x, _ := mem.I32(g.w, a); x != 77 {
			t.Fatalf("%s life %d", v, x)
		}
	}
	if err := g.set(UnitServer, VecBoth, 80, 5); !errors.Is(err, errAbsent) {
		t.Fatalf("absent stat: %v", err)
	}
	// insert keeps the vector sorted: 80 goes between 12 and 172
	if err := g.insertStat(g.server, VecMerged, 80, 300); err != nil {
		t.Fatal(err)
	}
	if v, ok := g.get(UnitServer, VecMerged, 80); !ok || v != 300 {
		t.Fatalf("inserted stat %d %v", v, ok)
	}
	if v, ok := g.get(UnitServer, VecMerged, 172); !ok || v != 1 {
		t.Fatal("shifted entry lost")
	}
	if err := g.insertStat(g.server, VecMerged, 81, 1); err == nil {
		t.Fatal("insert into a full vector must fail") // cap 5, now count 5
	}
	if err := g.insertStat(g.server, VecMerged, 80, 1); err == nil {
		t.Fatal("duplicate insert accepted")
	}
}

func TestGateDisabledByDefault(t *testing.T) {
	g := Gate{Env: testEnv(t)}
	if _, err := g.Do(context.Background(), KindHeal, nil, "x"); !errors.Is(err, ErrCheatsDisabled) {
		t.Fatalf("err %v", err)
	}
	if err := g.God(context.Background(), "x")(); !errors.Is(err, ErrCheatsDisabled) {
		t.Fatalf("god err %v", err)
	}
}

func saveEdit(t *testing.T, env *Env, params map[string]any) (*d2s.File, error) {
	t.Helper()
	params["character"] = "Multi"
	if _, err := env.Do(context.Background(), Request{Kind: KindSaveEdit, Actor: "tester", Reason: "unit test", Params: params}); err != nil {
		return nil, err
	}
	b, _ := os.ReadFile(filepath.Join(env.SaveDir, "Multi.d2s"))
	return d2s.Parse(b) // also proves size + checksum
}

func TestSaveQuestsWaypointsAct(t *testing.T) {
	env := testEnv(t)
	writeSave(t, env, "Multi")
	f, err := saveEdit(t, env, map[string]any{"quest": "1.den_of_evil:done,1/2.radament:started,2/w0:0x1234", "wp": "0/cold_plains+9,1/all"})
	if err != nil {
		t.Fatal(err)
	}
	if w, _ := f.QuestWord(0, 1); w != 0x8001 {
		t.Fatalf("den %#x", w)
	}
	if w, _ := f.QuestWord(1, 9); w != 0x000c {
		t.Fatalf("radament %#x", w)
	}
	if w, _ := f.QuestWord(2, 0); w != 0x1234 {
		t.Fatalf("raw word %#x", w)
	}
	if f.WaypointMask(0) != 1<<1|1<<9 || f.WaypointMask(1) != d2s.AllWaypoints {
		t.Fatalf("waypoints %x %x", f.WaypointMask(0), f.WaypointMask(1))
	}
	if f.WaypointMask(2) != 0 {
		t.Fatal("hell touched")
	}
	// all 39 bits: act I-IV done so the panel draws all five tabs; hell untouched; act-1 only bits keep act I open
	if w, ok := questWords(f)["act5_intro"]; !ok {
		t.Fatal("no act5_intro")
	} else if v, _ := f.QuestWord(1, w); v != 0x8001 {
		t.Fatalf("act5_intro %#x", v)
	}
	for _, a := range []int{1, 2, 3, 4} {
		w, ok := questWords(f)[fmt.Sprintf("act%d_done", a)]
		if !ok {
			t.Fatalf("no act%d_done word", a)
		}
		if v, _ := f.QuestWord(1, w); v != 0x8001 {
			t.Fatalf("nightmare act%d_done %#x", a, v)
		}
		if v, _ := f.QuestWord(2, w); v == 0x8001 {
			t.Fatalf("hell act%d_done set", a)
		}
	}

	f, err = saveEdit(t, env, map[string]any{"wp": "0/none", "questclear": 1, "unlock": 2, "startat": "2/3"})
	if err != nil {
		t.Fatal(err)
	}
	if f.Progression() != 10 || f.Difficulty(2) != 0x82 || f.Difficulty(0)&0x80 != 0 {
		t.Fatalf("progression %d diff %x %x", f.Progression(), f.Difficulty(2), f.Difficulty(0))
	}
	if f.WaypointMask(0) != 0 || f.WaypointMask(2) != 1<<18 { // Kurast Docks = first act 3 bit
		t.Fatalf("waypoints %x %x", f.WaypointMask(0), f.WaypointMask(2))
	}
	if w, _ := f.QuestWord(0, 1); w != 0 {
		t.Fatal("questclear did not clear normal")
	}
	if w, _ := f.QuestWord(2, 7); w != 0x8001 { // act1_done
		t.Fatalf("act1_done %#x", w)
	}
	if w, _ := f.QuestWord(2, 16); w != 0x8001 { // act3_intro
		t.Fatalf("act3_intro %#x", w)
	}

	// ledger carries the quest words before and after
	es, _ := env.Ledger.List(Filter{Kind: KindSaveEdit, OnlySuccess: true})
	if len(es) != 2 || es[0].After["quests0"] != "1=0x8001 7=0x8001" {
		t.Fatalf("ledger %+v", es)
	}
}

func TestSaveProgressErrors(t *testing.T) {
	env := testEnv(t)
	p := writeSave(t, env, "Multi")
	orig, _ := os.ReadFile(p)
	for _, params := range []map[string]any{
		{"quest": "1.nope:done"}, {"quest": "2.den_of_evil:done"}, {"quest": "1.den_of_evil:bogus"},
		{"quest": "9/1.den_of_evil"}, {"wp": "0/atlantis"}, {"wp": "0/39"}, {"unlock": 3}, {"startat": "6"},
	} {
		if _, err := saveEdit(t, env, params); err == nil {
			t.Errorf("%v accepted", params)
		}
	}
	if b, _ := os.ReadFile(p); string(b) != string(orig) {
		t.Fatal("a refused edit changed the save")
	}
	// no reason, no edit
	if _, err := env.Do(context.Background(), Request{Kind: KindSaveEdit, Params: map[string]any{"character": "Multi", "quest": "1.den_of_evil"}}); err == nil {
		t.Fatal("edit without a reason")
	}
}

func TestAllowListConfigDriven(t *testing.T) {
	t.Setenv(AllowFileEnv, filepath.Join(t.TempDir(), "none.txt"))
	t.Setenv(AllowEnv, "")
	for _, n := range []string{"Palli", "palli2", "Navi"} {
		if err := CheckCharacter(n); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
	}
	if err := CheckCharacter("Stranger"); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("Stranger: %v", err)
	}
	t.Setenv(AllowEnv, " Envi , ")
	f := filepath.Join(t.TempDir(), "allow.txt")
	os.WriteFile(f, []byte("# throwaways\nFilie  # i3\n\n"), 0o644)
	t.Setenv(AllowFileEnv, f)
	for _, n := range []string{"Envi", "filie"} {
		if err := CheckCharacter(n); err != nil {
			t.Fatalf("%s: %v", n, err)
		}
	}
	if err := CheckCharacter("# throwaways"); err == nil {
		t.Fatal("comment lines must not allow anything")
	}
}

// godFake is fakeGame with mana/stamina so heal works; life 1000, max life 2000 (raw values).
func godFake(t *testing.T) (*game, *mem.Fake) {
	g, f := fakeGame(t)
	ents := [][3]uint32{{6, 0, 1000}, {7, 0, 2000}, {8, 0, 500}, {9, 0, 600}, {10, 0, 700}, {11, 0, 800}}
	for _, base := range []uint32{0x3000, 0x4000} {
		b := make([]byte, len(ents)*8)
		for i, e := range ents {
			binary.LittleEndian.PutUint16(b[i*8+2:], uint16(e[0]))
			binary.LittleEndian.PutUint32(b[i*8+4:], e[2])
		}
		f.Put(base, b)
	}
	f.Put(0x2000+slVec, []byte{0, 0x30, 0, 0, 6, 0, 8, 0})
	f.Put(0x2000+slVecEx, []byte{0, 0x40, 0, 0, 6, 0, 8, 0})
	g.name = "Navi"
	return g, f
}

func maxLife(g *game) int32 { v, _ := g.get(UnitServer, VecMerged, statMaxLife); return v }

func TestGodRestoresOnCancelAndRemovesMarker(t *testing.T) {
	g, _ := godFake(t)
	env := testEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	seen := false
	env.Sleep = func(time.Duration) {
		if maxLife(g) == godBuffer {
			seen = true
			if _, err := os.Stat(env.markerPath()); err != nil {
				t.Errorf("marker missing while god is on: %v", err)
			}
			cancel() // the "signal"
		}
	}
	if _, err := env.god(ctx, g, Request{Params: map[string]any{"on": true, "for": "1h"}}); err != nil {
		t.Fatal(err)
	}
	if !seen {
		t.Fatal("god never inflated max life")
	}
	if maxLife(g) != 2000 {
		t.Fatalf("max life %d after cancel", maxLife(g))
	}
	if l, _ := g.get(UnitServer, VecMerged, statLife); l > 2000 {
		t.Fatalf("life %d above restored max", l)
	}
	if _, err := os.Stat(env.markerPath()); !os.IsNotExist(err) {
		t.Fatalf("marker not removed: %v", err)
	}
}

func TestGodRestoresOnPanic(t *testing.T) {
	g, _ := godFake(t)
	env := testEnv(t)
	env.Sleep = func(time.Duration) { panic("boom") }
	func() {
		defer func() { recover() }()
		env.god(context.Background(), g, Request{Params: map[string]any{"on": true, "for": "1h"}})
	}()
	if maxLife(g) != 2000 {
		t.Fatalf("max life %d after panic", maxLife(g))
	}
}

// A SIGKILLed run leaves the buffer and the marker; the next god start restores before it begins.
func TestGodRepairsKilledRun(t *testing.T) {
	g, _ := godFake(t)
	env := testEnv(t)
	if err := env.writeMarker(godMarker{Character: "Navi", Instance: "i5", MaxLife: 2000, Life: 1000}); err != nil {
		t.Fatal(err)
	}
	if err := g.setBoth(statMaxLife, godBuffer); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	msg, err := env.god(ctx, g, Request{Params: map[string]any{"on": true}})
	if err != nil || !strings.Contains(msg, "restored leftover god state") {
		t.Fatalf("%q %v", msg, err)
	}
	if maxLife(g) != 2000 {
		t.Fatalf("max life %d", maxLife(g))
	}
}

func TestGodRestoreCommandAndGuards(t *testing.T) {
	g, _ := godFake(t)
	env := testEnv(t)
	// inflated, no marker: god and restore without --maxlife refuse, with it they force
	_ = g.setBoth(statMaxLife, godBuffer)
	if _, err := env.god(context.Background(), g, Request{Params: map[string]any{"on": true}}); !errors.Is(err, ErrInflatedLife) {
		t.Fatalf("god on inflated: %v", err)
	}
	if _, err := env.godRestore(g, Request{}); !errors.Is(err, ErrInflatedLife) {
		t.Fatalf("restore without value: %v", err)
	}
	if _, err := env.godRestore(g, Request{Params: map[string]any{"maxlife": 9000}}); !errors.Is(err, ErrInflatedLife) {
		t.Fatalf("restore above the sane bound: %v", err)
	}
	if _, err := env.godRestore(g, Request{Params: map[string]any{"maxlife": 7}}); err != nil {
		t.Fatal(err)
	}
	if maxLife(g) != 7*256 {
		t.Fatalf("max life %d", maxLife(g))
	}
	// marker of another character is ignored, nothing to restore
	_ = env.writeMarker(godMarker{Character: "Other", MaxLife: 99})
	if msg, err := env.godRestore(g, Request{}); err != nil || msg != "nothing to restore" {
		t.Fatalf("%q %v", msg, err)
	}
	if m, _ := env.PendingGod(); m == nil || m.Character != "Other" {
		t.Fatal("foreign marker was removed")
	}
}

func TestGateGodStopIdempotent(t *testing.T) {
	g := Gate{Allow: false}
	stop := g.God(context.Background(), "x")
	if stop() == nil || stop() == nil {
		t.Fatal("disabled gate must keep returning its error")
	}
}

// A stale marker (clean save, game died before saving the buffer) must not lower a levelled character.
func TestStaleMarkerIsDroppedNotRestored(t *testing.T) {
	g, _ := godFake(t)
	env := testEnv(t)
	cur := maxLife(g)
	if err := env.writeMarker(godMarker{Character: "Navi", Instance: "i5", MaxLife: 500, Life: 500}); err != nil {
		t.Fatal(err)
	}
	msg, err := env.godRestore(g, Request{})
	if err != nil || !strings.Contains(msg, "dropped stale god marker") {
		t.Fatalf("%q %v", msg, err)
	}
	if maxLife(g) != cur {
		t.Fatalf("max life changed %d -> %d", cur, maxLife(g))
	}
	if m, _ := env.PendingGod(); m != nil {
		t.Fatal("marker kept")
	}
}
