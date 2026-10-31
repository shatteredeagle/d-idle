package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// avoidSpy records what the gate hands the navigator.
type avoidSpy struct {
	*fakeMover
	set map[int]bool
}

func (a *avoidSpy) SetAvoidLevels(m map[int]bool) { a.set = m }

// kitRig is a recovering character of the given level with the kit parts asked for.
func kitRig(t *testing.T, level int, belt, tp bool, knob int) (*rig, *d2.Snapshot) {
	t.Helper()
	g := newRig(t)
	g.e.Tune.Resource.RecoveryKitGate = knob
	g.w.set(func(w *world) { w.level = level })
	if !belt {
		g.w.belt()
	}
	if tp {
		g.w.set(func(w *world) {
			w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
		})
	}
	g.r.loop.broke.on = true
	s, _ := g.w.snapshot()
	return g, s
}

// Scenario "Recovery area without the kit" (openspec/changes/kitgate/specs/bot-loop/spec.md).
func TestKitGateRecoveryArea(t *testing.T) {
	for _, tc := range []struct {
		name     string
		level    int
		belt, tp bool
	}{
		{"no belt, no portal, clvl 7", 7, false, false},
		{"no belt, portal, clvl 7", 7, false, true},
		{"belt, no portal, clvl 7", 7, true, false},
		{"no belt, portal, clvl 14", 14, false, true},
		{"belt, no portal, clvl 14", 14, true, false},
	} {
		g, s := kitRig(t, tc.level, tc.belt, tc.tp, 1)
		if !g.r.kitMissing(s) {
			t.Fatalf("%s: kit must be missing", tc.name)
		}
		if a, ok := g.r.recoveryArea(s); ok && !kitSafeTier[a] {
			t.Errorf("%s: gate on picked area %d outside Blood Moor / Cold Plains", tc.name, a)
		}
	}
}

// Scenario "Kit present keeps the old choice": the gate changes nothing for a character with belt and portal, and the
// knob 0 restores the old choice for one without.
func TestKitGateKeepsOldChoices(t *testing.T) {
	for _, level := range []int{3, 7, 14} {
		on, sOn := kitRig(t, level, true, true, 1)
		off, sOff := kitRig(t, level, true, true, 0)
		if on.r.kitMissing(sOn) {
			t.Fatal("belt and portal: kit present")
		}
		a1, ok1 := on.r.recoveryArea(sOn)
		a2, ok2 := off.r.recoveryArea(sOff)
		if a1 != a2 || ok1 != ok2 {
			t.Errorf("clvl %d with the kit: gate on %d/%v, off %d/%v", level, a1, ok1, a2, ok2)
		}
	}
	// the gate is the only difference without the kit: clvl 14, portal, empty belt
	g, s := kitRig(t, 14, false, true, 0)
	if a, ok := g.r.recoveryArea(s); !ok || kitSafeTier[a] {
		t.Fatalf("knob 0 must give the old XP pick (not Blood Moor or Cold Plains), got %d %v", a, ok)
	}
}

// Both fields cleared or banned: no area (the goal waits at home) instead of a fall-through to the quest ladder.
func TestKitGateNoSafeFieldWaits(t *testing.T) {
	g, s := kitRig(t, 14, false, true, 1)
	for id := range kitSafeTier {
		g.r.avoid[id] = g.clock.Now().Add(20 * 60e9)
	}
	if a, ok := g.r.recoveryArea(s); ok {
		t.Fatalf("both safe fields banned, got %d", a)
	}
}

// Scenario "Walks avoid the dungeons": the navigator gets the dungeon list while the gate is on and loses it after.
func TestKitGateNavAvoid(t *testing.T) {
	g, s := kitRig(t, 7, false, true, 1)
	spy := &avoidSpy{fakeMover: g.mover}
	g.r.d.Mover = spy
	g.r.syncKitGate(s)
	for _, a := range []int{areaDenOfEvil, 9, 13, 17, 18, 19} {
		if !spy.set[a] {
			t.Errorf("area %d missing from the avoid set %v", a, spy.set)
		}
	}
	if spy.set[2] || spy.set[3] || spy.set[4] {
		t.Errorf("fields must not be avoided: %v", spy.set)
	}
	g.w.belt(101, 102)
	g.w.set(func(w *world) {})
	s2, _ := g.w.snapshot()
	g.r.syncKitGate(s2)
	// belt back but still no portal in the pack? the rig gave one, so the kit is complete
	if len(spy.set) != 0 {
		t.Errorf("kit complete: the avoid set must clear, got %v", spy.set)
	}
}
