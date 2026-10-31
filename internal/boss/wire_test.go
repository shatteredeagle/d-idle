package boss

import (
	"errors"
	"strings"
	"testing"
	"time"
)

// Countess, Smith and Andariel under failure: entry, route, drink and antidote trouble, timeouts, quest loot.

func TestPotionFloorRefusesApproach(t *testing.T) {
	for _, n := range []string{NameCountess, NameSmith, NameAndariel} {
		f := newFake()
		f.level, f.potions = 30, 0
		_, err := run(t, n, f, Opts{})
		if !errors.Is(err, ErrAborted) || len(f.log) != 0 {
			t.Fatalf("%s: %v %v", n, err, f.log)
		}
	}
}

func TestPotionFloorSkippedWhenAlreadyInArena(t *testing.T) {
	f := newFake()
	f.level, f.potions, f.area = 30, 0, AreaBarracks
	b := f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{110, 100}, UniqueNo: -1}, AreaBarracks, 20)
	b.dmg = 0
	f.lootN = 1
	if res, err := run(t, NameSmith, f, Opts{}); err != nil || !res.Killed {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestCountessHopFailureInsideTowerGoesToTown(t *testing.T) {
	f := newFake()
	f.level = 20
	f.exitFail = map[int]bool{23: true}
	_, err := run(t, NameCountess, f, Opts{})
	if err == nil || !strings.Contains(err.Error(), "exit to 23") || f.townCalls != 1 {
		t.Fatalf("%v town %d %v", err, f.townCalls, f.log)
	}
}

func TestCountessFirstHopFailureNoTownTrip(t *testing.T) {
	f := newFake()
	f.level = 20
	f.exitFail = map[int]bool{AreaForgottenTower: true}
	f.arriveFn = func(f *fake, a int) {}
	_, err := run(t, NameCountess, f, Opts{})
	// the character stands in Black Marsh (the WP), not in the dungeon: nothing to abort
	if err == nil || f.townCalls != 0 {
		t.Fatalf("%v %d", err, f.townCalls)
	}
}

func TestCountessDiesOnRoute(t *testing.T) {
	f := newFake()
	f.level = 20
	f.arriveFn = func(f *fake, a int) {
		if a == 22 {
			f.life = 0
		}
	}
	_, err := run(t, NameCountess, f, Opts{})
	if !errors.Is(err, ErrDead) {
		t.Fatalf("%v", err)
	}
}

func TestCountessPresetErrorIsANoteNotFatal(t *testing.T) {
	f := newFake()
	f.level, f.area = 20, AreaTowerCellar5
	f.presetErr = errors.New("map offline")
	f.addBoss(Mon{Class: 45, UniqueNo: -1, Name: "The Countess", Pos: Pt{9000, 9000}}, AreaTowerCellar5, 30).vis = 20
	f.exploreFn = func(f *fake) { f.pos = Pt{9005, 9005} }
	res, err := run(t, NameCountess, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("%v", err)
	}
	if !strings.Contains(strings.Join(res.Notes, "|"), "presets: map offline") {
		t.Fatalf("%v", res.Notes)
	}
}

func TestCountessUnseenTimesOutInsteadOfExploringForever(t *testing.T) {
	f := newFake()
	f.level, f.area = 20, AreaTowerCellar5
	f.exploreFn = func(f *fake) { f.now = f.now.Add(20 * time.Minute) }
	_, err := run(t, NameCountess, f, Opts{})
	if !errors.Is(err, ErrTimeout) && !errors.Is(err, ErrBossNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestCountessIgnoresLookalikeOfAnotherName(t *testing.T) {
	f := newFake()
	f.level, f.area = 20, AreaTowerCellar5
	f.addBoss(Mon{Class: 45, UniqueNo: -1, Name: "Treehead Woodfist", Pos: Pt{100, 100}}, AreaTowerCellar5, 30)
	_, err := run(t, NameCountess, f, Opts{Explore: time.Second, LocatePolls: 2})
	if !errors.Is(err, ErrBossNotFound) {
		t.Fatalf("%v", err)
	}
}

func TestCountessLootNeverAsksForChests(t *testing.T) {
	s, _ := Lookup(NameCountess)
	for _, k := range s.Loot.Kinds {
		if strings.Contains(k, "chest") {
			t.Fatalf("loot kind %q", k)
		}
	}
	if s.Loot.Kinds[0] != "rune" {
		t.Fatalf("runes first: %v", s.Loot.Kinds)
	}
}

func TestDrinkFailureBelowEscapeLeavesFight(t *testing.T) {
	f := newFake()
	f.level, f.area = 30, AreaBarracks
	f.life, f.drinkErr = 20, errors.New("belt column empty")
	f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{110, 100}, UniqueNo: -1}, AreaBarracks, 100)
	res, err := run(t, NameSmith, f, Opts{})
	if !errors.Is(err, ErrEscaped) || f.townCalls != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
	if !strings.Contains(strings.Join(res.Notes, "|"), "drink failed") {
		t.Fatalf("%v", res.Notes)
	}
}

func TestDrinkFailureAboveEscapeKeepsFighting(t *testing.T) {
	f := newFake()
	f.level, f.area = 30, AreaBarracks
	f.life, f.drinkErr = 40, errors.New("belt column empty") // under DrinkPct 45, over EscapePct 25
	b := f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{110, 100}, UniqueNo: -1}, AreaBarracks, 20)
	b.dmg = 0
	f.lootN = 1
	if res, err := run(t, NameSmith, f, Opts{}); err != nil || !res.Killed {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestSmithQuestLootRetriedOnce(t *testing.T) {
	f := newFake()
	f.level, f.area = 30, AreaBarracks
	f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{110, 100}, UniqueNo: -1}, AreaBarracks, 20).dmg = 0
	f.lootSeq = []int{0, 1}
	res, err := run(t, NameSmith, f, Opts{})
	if err != nil || res.Loot != 1 || f.did("loot") != 2 {
		t.Fatalf("%v loot %d %v", err, res.Loot, f.log)
	}
}

func TestSmithQuestItemMissingIsNoted(t *testing.T) {
	f := newFake()
	f.level, f.area = 30, AreaBarracks
	f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{110, 100}, UniqueNo: -1}, AreaBarracks, 20).dmg = 0
	res, err := run(t, NameSmith, f, Opts{})
	if err != nil || !res.Killed || f.did("loot") != 2 || !strings.Contains(strings.Join(res.Notes, "|"), "still missing") {
		t.Fatalf("%v %v %v", err, f.log, res.Notes)
	}
}

func TestSmithIgnoresFriendlyCrowd(t *testing.T) {
	// Hostile comes from combat.IsFriendly in the adapter; friendly units (pets, mercenary) must not trip the crowd rule.
	f := newFake()
	f.level, f.area, f.life = 30, AreaBarracks, 45
	f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{110, 100}, UniqueNo: -1}, AreaBarracks, 20).dmg = 0
	for i := 0; i < 8; i++ {
		m := f.addBoss(Mon{Class: 70, Pos: Pt{101 + i, 100}, UniqueNo: -1}, AreaBarracks, 5)
		m.Hostile, m.Mon.ID = false, uint32(200+i)
	}
	f.lootN = 1
	res, err := run(t, NameSmith, f, Opts{})
	if err != nil || has(res, "hazard:crowded") || has(res, "hazard:adds") {
		t.Fatalf("%v %v", err, res.Phases)
	}
}

func TestAndarielAntidoteFailureDoesNotEndRun(t *testing.T) {
	f := newFake()
	f.level, f.area, f.antidotes = 30, AreaCatacombs4, 2
	f.antiErr = errors.New("no antidote in belt")
	f.poisoned = true
	b := f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{200, 100}, UniqueNo: -1}, AreaCatacombs4, 60)
	b.dmg = 0
	f.lootN = 1
	res, err := run(t, NameAndariel, f, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("%v %v", err, f.log)
	}
	if !strings.Contains(strings.Join(res.Notes, "|"), "antidote") {
		t.Fatalf("%v", res.Notes)
	}
}

func TestAndarielChickenKeepsCharacterAliveWithoutTP(t *testing.T) {
	f := newFake()
	f.level, f.area = 30, AreaCatacombs4
	f.life = 10 // under ChickenPct 12
	f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{200, 100}, UniqueNo: -1}, AreaCatacombs4, 100)
	_, err := run(t, NameAndariel, f, Opts{})
	if !errors.Is(err, ErrChicken) || f.did("chicken") != 1 {
		t.Fatalf("%v %v", err, f.log)
	}
}

func TestAndarielFightTimeout(t *testing.T) {
	f := newFake()
	f.level, f.area, f.potions = 30, AreaCatacombs4, 9
	f.antidotes = 3
	b := f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{200, 100}, UniqueNo: -1}, AreaCatacombs4, 1000000)
	b.dmg = 0
	_, err := run(t, NameAndariel, f, Opts{})
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("%v", err)
	}
}

func TestAndarielKiteRelativeSidesStayAwayFromBoss(t *testing.T) {
	f := newFake()
	f.level, f.area, f.potions, f.antidotes = 30, AreaCatacombs4, 9, 3
	f.pos = Pt{1000, 1000}
	b := f.addBoss(Mon{Class: ClassAndariel, Name: "Andariel", Pos: Pt{1004, 1000}, UniqueNo: -1}, AreaCatacombs4, 100)
	b.dmg = 0
	f.lootN = 1
	res, err := run(t, NameAndariel, f, Opts{})
	if err != nil || !has(res, "kite") {
		t.Fatalf("%v %v", err, res.Phases)
	}
	for _, l := range f.log {
		if l == "goto 1004,1000" {
			t.Fatalf("walked onto the boss: %v", f.log)
		}
	}
}

// The Smith is not in sight on arrival (vis 30): locate must walk to the preset, then to the Malus object if the
// monster preset is missing, before exploring.
func TestSmithLocatedByPreset(t *testing.T) {
	for name, ps := range map[string][]Preset{
		"monster preset": {{Kind: PresetNPC, ID: presetSmith, Pos: Pt{500, 500}}},
		"malus object":   {{Kind: PresetObject, ID: objMalus, Pos: Pt{500, 500}}},
	} {
		f := newFake()
		f.level, f.area, f.pos, f.presets = 30, AreaBarracks, Pt{100, 100}, ps
		sm := f.addBoss(Mon{Class: ClassSmith, Name: "The Smith", Pos: Pt{505, 500}, UniqueNo: -1}, AreaBarracks, 20)
		sm.dmg, sm.vis = 0, 30
		res, err := run(t, NameSmith, f, Opts{})
		if err != nil || !res.Killed || f.did("explore") != 0 {
			t.Fatalf("%s: %v killed=%v %v", name, err, res.Killed, f.log)
		}
	}
}
