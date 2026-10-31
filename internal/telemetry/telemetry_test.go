package telemetry

import (
	"math"
	"testing"
	"time"
)

func newT() (*Tracker, *FakeClock) {
	c := &FakeClock{T: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
	return New(c, Config{ExpForLevel: DefaultExpForLevel}), c
}

func count(t *Tracker, kind string) int { return len(t.Events(10000, kind)) }

func TestRing(t *testing.T) {
	r := NewRing[int](3)
	for i := 1; i <= 5; i++ {
		r.Push(i)
	}
	if s := r.Slice(); len(s) != 3 || s[0] != 3 || s[2] != 5 {
		t.Fatalf("got %v", s)
	}
	if l := r.Last(2); l[0] != 4 || l[1] != 5 {
		t.Fatalf("got %v", l)
	}
}

func TestXPRateWindows(t *testing.T) {
	tr, c := newT()
	f := Frame{InGame: true, Area: 2, Level: 5, Exp: 10000}
	tr.Observe(f)
	for i := 0; i < 600; i++ { // 10 minutes, 100 exp per 1s tick
		c.Advance(time.Second)
		f.Exp += 100
		tr.Observe(f)
	}
	o := tr.Overview()
	want := 100.0 * 3600
	if got := o.Rates[0].XP; math.Abs(got-want)/want > 0.02 {
		t.Fatalf("5min xp/h = %v want %v", got, want)
	}
	if o.Rates[0].Partial {
		t.Fatal("5 min window should be full")
	}
	if !o.Rates[1].Partial {
		t.Fatal("1 h window should be partial after 10 min")
	}
	if got := o.Rates[2].XP; math.Abs(got-want)/want > 0.02 {
		t.Fatalf("session xp/h = %v", got)
	}
	// idle for 5 minutes: 5-minute rate collapses to ~0
	for i := 0; i < 300; i++ {
		c.Advance(time.Second)
		tr.Observe(f)
	}
	if got := tr.Overview().Rates[0].XP; got > 1 {
		t.Fatalf("idle 5min rate = %v", got)
	}
}

func TestETA(t *testing.T) {
	tr, c := newT()
	f := Frame{InGame: true, Area: 2, Level: 5, Exp: 7875}
	tr.Observe(f)
	c.Advance(time.Minute)
	f.Exp += 630 // 630/min -> 37800/h; need 14175-8505 = 5670 -> 9 min
	tr.Observe(f)
	eta := tr.Overview().ETASecs
	if math.Abs(eta-540) > 20 {
		t.Fatalf("eta %v", eta)
	}
}

func TestKillsNeedTransition(t *testing.T) {
	tr, c := newT()
	f := Frame{InGame: true, Area: 2, Level: 1, Exp: 1}
	f.Monsters = []Monster{{ID: 1, Mode: 1}, {ID: 2, Mode: 12}, {ID: 3, Mode: 0, Friendly: true}}
	tr.Observe(f) // baseline: corpse #2 not counted
	c.Advance(time.Second)
	f.Monsters = []Monster{{ID: 1, Mode: 8}, {ID: 2, Mode: 12}, {ID: 3, Mode: 1, Friendly: true}}
	tr.Observe(f)
	f.Monsters = []Monster{{ID: 1, Mode: 0}, {ID: 2, Mode: 12}}
	tr.Observe(f)
	f.Monsters = []Monster{{ID: 1, Mode: 12}, {ID: 2, Mode: 12}}
	tr.Observe(f) // 0 -> 12 must not double count
	if n := count(tr, KindKill); n != 1 {
		t.Fatalf("kills = %d", n)
	}
	if tr.Overview().TotalKills != 1 {
		t.Fatal("total kills")
	}
}

func TestDropsPotionsDeathsLevelUps(t *testing.T) {
	tr, c := newT()
	f := Frame{InGame: true, Area: 2, Level: 5, Exp: 8000, Gold: 100,
		Ground: []GroundItem{{ID: 9}}, Belt: Belt{Health: 4, Mana: 2}}
	tr.Observe(f)
	c.Advance(time.Second)
	f.Ground = []GroundItem{{ID: 9}, {ID: 10, Name: "Short Sword", Quality: "magic"}}
	f.Belt = Belt{Health: 3, Mana: 2}
	f.Gold = 150
	f.Dead = true
	f.Level, f.Exp = 6, 15000
	tr.Observe(f)
	f.Belt = Belt{Health: 3, Mana: 2}
	tr.Observe(f) // still dead: no second death
	o := tr.Overview()
	if o.TotalDrops != 1 || o.Potions != 1 || o.Deaths != 1 || o.LevelUps != 1 || o.TotalGold != 50 {
		t.Fatalf("%+v", o)
	}
	if d := tr.Drops(5); len(d) != 1 || d[0].Extra != "magic" {
		t.Fatalf("drops %+v", d)
	}
}

func TestTownPotionsIgnoredAndAreaChange(t *testing.T) {
	tr, c := newT()
	f := Frame{InGame: true, InTown: true, Area: 1, Level: 1, Belt: Belt{Health: 4}}
	tr.Observe(f)
	f.Belt.Health = 0
	tr.Observe(f)
	if tr.Overview().Potions != 0 {
		t.Fatal("town belt change counted")
	}
	c.Advance(10 * time.Second)
	f.Area, f.InTown = 2, false
	f.Ground = []GroundItem{{ID: 1}}
	tr.Observe(f) // ground item present on arrival is baseline, not a drop
	c.Advance(5 * time.Second)
	tr.Observe(f)
	if tr.Overview().TotalDrops != 0 {
		t.Fatal("baseline item counted")
	}
	as := tr.Areas()
	if len(as) != 2 || as[0].Area != 2 && as[0].Area != 1 {
		t.Fatalf("%+v", as)
	}
	var town AreaStat
	for _, a := range as {
		if a.Area == 1 {
			town = a
		}
	}
	if town.Secs != 10 || town.Visits != 1 {
		t.Fatalf("town %+v", town)
	}
}

func TestRunsActivityAndBoundedMemory(t *testing.T) {
	tr, c := newT()
	var got []Event
	tr.OnEvent(func(e Event) { got = append(got, e) })
	tr.SetActivity("fighting", "Blood Moor")
	tr.RunStart("Blood Moor")
	c.Advance(90 * time.Second)
	tr.RunEnd()
	c.Advance(time.Second)
	tr.SetActivity("town", "")
	o := tr.Overview()
	if o.Runs != 1 || o.AvgRunSecs != 90 {
		t.Fatalf("%+v", o)
	}
	segs := tr.Segments(10)
	if len(segs) != 2 || segs[0].End == nil || segs[1].End != nil {
		t.Fatalf("%+v", segs)
	}
	if len(got) < 4 {
		t.Fatalf("callbacks %d", len(got))
	}
	f := Frame{InGame: true, Area: 2, Level: 1}
	for i := 0; i < 20000; i++ {
		c.Advance(time.Second)
		f.Exp++
		f.Ground = []GroundItem{{ID: uint32(i)}}
		f.Monsters = []Monster{{ID: uint32(i), Mode: 1}}
		tr.Observe(f)
	}
	if tr.events.Len() > 1000 || tr.xp.ring.Len() > 4096 || len(tr.monst) > 2200 || len(tr.ground) > 4100 {
		t.Fatal("unbounded growth")
	}
	if n := len(tr.Series(5)); n == 0 || n > 1440 {
		t.Fatalf("series %d", n)
	}
}

// Spot checks against experience.txt dumped from the real d2data.mpq (level N -> total exp to reach it).
func TestDefaultExpTableTo99(t *testing.T) {
	for lvl, want := range map[int]uint32{2: 500, 30: 4663553, 31: 5493363, 60: 117772849, 98: 3229426756, 99: 3520485254} {
		if got, ok := DefaultExpForLevel(lvl); !ok || got != want {
			t.Errorf("level %d = %d %v, want %d", lvl, got, ok, want)
		}
	}
	if _, ok := DefaultExpForLevel(100); ok {
		t.Error("level 100 must be unknown")
	}
	prev := uint32(0)
	for l := 2; l <= 99; l++ {
		if v, _ := DefaultExpForLevel(l); v <= prev {
			t.Fatalf("table not increasing at %d", l)
		} else {
			prev = v
		}
	}
}
