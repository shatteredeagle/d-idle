package main

import (
	"fmt"
	"log/slog"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2/fake"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/senses"
)

func unitCat() combat.MapCatalog {
	mk := func(name string) *data.Monster {
		return &data.Monster{Key: name, Name: name, Killable: true, Enabled: true}
	}
	return combat.MapCatalog{Monsters: map[int]*data.Monster{
		5: mk("Fallen"), 6: mk("Zombie"), 7: mk("Corpsefire-class"),
		150: {Key: "kashya", Name: "Kashya", NPC: true}, 271: {Key: "rogue", Name: "Rogue", AI: "Hireable"},
	}}
}

func modName(id int) string {
	if n, ok := map[int]string{5: "Extra Strong", 6: "Extra Fast", 7: "Cursed"}[id]; ok {
		return n
	}
	return fmt.Sprintf("unknown mod #%d", id)
}

func mon(id uint32, class, x, y int, mi *d2.MonsterInfo) d2.Unit {
	return d2.Unit{ID: id, Type: d2.TypeMonster, Class: class, Mode: d2.MonModeStand, X: x, Y: y, Monster: mi}
}

func snapOf(class d2.Class, merc bool, hover uint32) *d2.Snapshot {
	ms := []d2.Unit{
		mon(1, 5, 12, 10, &d2.MonsterInfo{LifePct: 40, Life: 20, MaxLife: 50}),
		mon(2, 150, 8, 10, &d2.MonsterInfo{}),
		mon(3, 6, 14, 10, &d2.MonsterInfo{LifePct: 100, Life: 9, MaxLife: 9}),
		{ID: 4, Type: d2.TypeMonster, Class: 6, Mode: d2.MonModeDeath, X: 15, Y: 10}, // dead: dropped
		mon(5, 289, 9, 11, nil), // clay golem: a summon, not a dot
		// champion (flags 0x0c) with Extra Strong, Cursed and an id above the monumod range handled as unknown mods
		mon(6, 7, 16, 10, &d2.MonsterInfo{Flags: 0x0c, Enchants: [9]uint8{5, 7}, LifePct: 100, Life: 90, MaxLife: 90}),
		// superunique (0x0a) with its own name
		mon(7, 7, 17, 10, &d2.MonsterInfo{Flags: 0x0a, Enchants: [9]uint8{6, 33}, Name: "Corpsefire", LifePct: 100, Life: 90, MaxLife: 90}),
		mon(8, 7, 18, 10, &d2.MonsterInfo{Flags: 0x0c, Enchants: [9]uint8{99}, LifePct: 100, Life: 5, MaxLife: 5}), // garbage enchants
	}
	if merc {
		ms = append(ms, mon(9, 271, 10, 10, &d2.MonsterInfo{LifePct: 100, Life: 5, MaxLife: 5}))
	}
	s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, Class: class}, Monsters: ms}
	if hover != 0 {
		s.Hovered = d2.Hover{Active: true, Type: d2.TypeMonster, ID: hover}
	}
	return s
}

func byID(us []senses.Unit, id uint32) (senses.Unit, bool) {
	for _, u := range us {
		if u.ID == id {
			return u, true
		}
	}
	return senses.Unit{}, false
}

func TestUnitSourceTwoClasses(t *testing.T) {
	for _, tc := range []struct {
		class d2.Class
		merc  bool
	}{{d2.Sorceress, false}, {d2.Paladin, true}} {
		us := unitSource{Read: &fake.Reader{Snap: snapOf(tc.class, tc.merc, 1)}, Cat: unitCat(), ModName: modName}.Units()
		if _, ok := byID(us, 4); ok {
			t.Fatal("dead unit listed")
		}
		if _, ok := byID(us, 5); ok {
			t.Fatal("summon listed")
		}
		if u, _ := byID(us, 2); u.Kind != senses.KindNPC {
			t.Fatalf("npc: %+v", u)
		}
		m, ok := byID(us, 9)
		if ok != tc.merc || (ok && m.Kind != senses.KindMerc) {
			t.Fatalf("%v: merc %+v present=%v", tc.class, m, ok)
		}
		if u, _ := byID(us, 1); u.Kind != senses.KindMonster || !u.Target || u.LifePct != 40 || u.Name != "Fallen" {
			t.Fatalf("target: %+v", u)
		}
		if u, _ := byID(us, 3); u.Target {
			t.Fatalf("second monster is a target: %+v", u)
		}
		c, _ := byID(us, 6)
		if c.Elite != senses.EliteChampion || len(c.Mods) != 2 || c.Mods[0] != "Extra Strong" || c.Mods[1] != "Cursed" {
			t.Fatalf("champion: %+v", c)
		}
		su, _ := byID(us, 7)
		if su.Elite != senses.EliteSuperunique || su.Title != "Corpsefire" || su.Mods[1] != "unknown mod #33" {
			t.Fatalf("superunique: %+v", su)
		}
		// An enchant byte outside the monumod range is a bad read: the champion stays, with no mods.
		if g, _ := byID(us, 8); g.Elite != senses.EliteChampion || len(g.Mods) != 0 {
			t.Fatalf("garbage enchants: %+v", g)
		}
	}
}

func TestUnitSourceNotInGame(t *testing.T) {
	if us := (unitSource{Read: &fake.Reader{}}).Units(); us != nil {
		t.Fatalf("units while loading: %v", us)
	}
	if us := (unitSource{}).Units(); us != nil {
		t.Fatal("nil reader")
	}
}

func TestSensesFeedUnitsThrottleAndRace(t *testing.T) {
	pub := &senses.Publisher{}
	clk := time.Unix(0, 0)
	rd := &fake.Reader{Snap: snapOf(d2.Sorceress, false, 0)}
	hook := sensesFeedUnits(pub, func() time.Time { return clk }, slog.Default(), &unitSource{Read: rd, Cat: unitCat()})
	hook(strip(1, 40), d2map.Point{X: 10, Y: 1})
	s, _ := pub.Snapshot()
	n := len(s.Unit)
	if n == 0 {
		t.Fatal("no units published")
	}
	// Within the throttle window nothing is re-read: a changed world is not visible yet.
	rd.Snap = &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1}}
	clk = clk.Add(100 * time.Millisecond)
	hook(strip(1, 40), d2map.Point{X: 10, Y: 1})
	if s, _ = pub.Snapshot(); len(s.Unit) != n {
		t.Fatal("units re-read inside the throttle window")
	}
	clk = clk.Add(time.Second)
	hook(strip(1, 40), d2map.Point{X: 10, Y: 1})
	if s, _ = pub.Snapshot(); len(s.Unit) != 0 {
		t.Fatalf("stale units after the window: %d", len(s.Unit))
	}
	// Concurrent pollers against the hook.
	rd.Snap = snapOf(d2.Paladin, true, 1)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				if g == 0 {
					clk = clk.Add(time.Second)
					hook(strip(1, 40), d2map.Point{X: 10, Y: 1})
				} else {
					pub.Snapshot()
				}
			}
		}(g)
	}
	wg.Wait()
}

func TestUseSensesNoUnitsWithoutReader(t *testing.T) {
	e := &env.Env{}
	useSenses(&ladderRig{Senses: &senses.Publisher{}}, e, nil, slog.Default())
	if e.OnObserve == nil {
		t.Fatal("hook not set")
	}
}

// recentFake is a reader whose Snapshot must not be called when Recent has a snapshot (WALKSMOOTH2).
type recentFake struct {
	fake.Reader
	recent *d2.Snapshot
	reads  int
}

func (r *recentFake) Snapshot() (*d2.Snapshot, error) { r.reads++; return r.Reader.Snapshot() }
func (r *recentFake) Recent(time.Duration) (*d2.Snapshot, bool) {
	return r.recent, r.recent != nil
}

func TestUnitSourceReusesTheWalkersSnapshot(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		rf := &recentFake{Reader: fake.Reader{Snap: snapOf(class, false, 0)}, recent: snapOf(class, false, 0)}
		us := unitSource{Read: rf, Cat: unitCat(), ModName: modName}.Units()
		if rf.reads != 0 || len(us) == 0 {
			t.Fatalf("%v: a recent snapshot existed, yet Snapshot was read %d times (units %d)", class, rf.reads, len(us))
		}
		rf.recent = nil
		unitSource{Read: rf, Cat: unitCat(), ModName: modName}.Units()
		if rf.reads != 1 {
			t.Fatalf("%v: without a recent snapshot one fresh read is expected, got %d", class, rf.reads)
		}
	}
}

// TASK-186: d2d never sets Snapshot.Hovered; the fighter's attack target flags the unit instead.
func TestUnitSourceAttackTarget(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		snap := snapOf(class, false, 0) // no hover, as on d2d
		var att uint32
		src := unitSource{Read: &fake.Reader{Snap: snap}, Cat: unitCat(), Attacking: func() uint32 { return att }}
		for _, m := range src.Units() {
			if m.Target {
				t.Fatalf("%v: target flagged outside a fight: %+v", class, m)
			}
		}
		att = 3
		us := src.Units()
		if u, _ := byID(us, 3); !u.Target || u.LifePct != 100 || u.Name == "" {
			t.Fatalf("%v: attacked unit: %+v", class, u)
		}
		if u, _ := byID(us, 1); u.Target {
			t.Fatalf("%v: other unit flagged: %+v", class, u)
		}
		att = 2 // an NPC id never becomes a target
		if u, _ := byID(src.Units(), 2); u.Target {
			t.Fatalf("%v: npc flagged: %+v", class, u)
		}
	}
}
