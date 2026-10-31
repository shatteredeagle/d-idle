package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func TestRecoveryXPModel(t *testing.T) {
	// V-d2d means at clvl 12: Cold Plains under 3, Stony Field about 15, Dark Wood about 35.
	if x := areaXPPerKill(12, 2); x >= 3 {
		t.Fatalf("cold plains %.1f", x)
	}
	if x := areaXPPerKill(12, 4); x < 12 || x > 20 {
		t.Fatalf("stony %.1f", x)
	}
	if x := areaXPPerKill(12, 5); x < 28 || x > 40 {
		t.Fatalf("dark wood %.1f", x)
	}
	if xpLevelPct(30, 36) != 100*30.0/36 || xpLevelPct(5, 15) != 1.95 || xpLevelPct(5, 5) != 100 {
		t.Fatal("penalty table")
	}
}

func xpPick(g *rig, level int, ex map[int]bool) (int, bool) {
	g.w.set(func(w *world) { w.level = level })
	s, _ := g.w.snapshot()
	return g.r.recoveryXPArea(s, ex)
}

func TestRecoveryXPAreaByLevel(t *testing.T) {
	g := newRig(t)
	if a, ok := xpPick(g, 14, nil); !ok || (a != 5 && a != 4 && a != 6) {
		t.Fatalf("clvl 14 picked %d %v", a, ok)
	}
	if a, _ := xpPick(g, 14, nil); a == 3 || a == 2 {
		t.Fatalf("clvl 14 must not farm %d", a)
	}
	if a, ok := xpPick(g, 6, nil); !ok || (a != 2 && a != 3 && a != 9) {
		t.Fatalf("clvl 6 picked %d", a)
	}
	// the cap: with +0 a clvl 6 character takes nothing above monster level 6
	g.e.Tune.Resource.RecoveryMaxAreaAbove = 0
	if a, _ := xpPick(g, 9, nil); a == 6 {
		t.Fatalf("clvl 9 capped at +0 picked Black Marsh")
	}
}

func TestRecoveryXPAreaExcluded(t *testing.T) {
	g := newRig(t)
	a, _ := xpPick(g, 14, nil)
	b, ok := xpPick(g, 14, map[int]bool{a: true})
	if !ok || b == a {
		t.Fatalf("excluded %d still picked %d", a, b)
	}
	all := map[int]bool{}
	for id := range recoveryXPTier {
		all[id] = true
	}
	if _, ok := xpPick(g, 14, all); ok {
		t.Fatal("everything excluded must report nothing")
	}
}

func TestRecoveryAreaBannedFallsThrough(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.level = 14 })
	g.w.belt()
	s, _ := g.w.snapshot()
	for id := range recoveryXPTier {
		g.r.avoid[id] = g.clock.Now().Add(20 * 60e9)
	}
	if a, ok := g.r.recoveryArea(s); ok {
		t.Fatalf("all banned must leave no area (noidle wait), got %d", a)
	}
}

func TestRecoveryAreaUsesXPPick(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.level = 14 })
	g.w.set(func(w *world) {
		w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
	})
	s, _ := g.w.snapshot()
	if !g.r.resInput(s).HasTP {
		t.Fatal("rig has no town portal")
	}
	a, ok := g.r.recoveryArea(s)
	if !ok || a == 2 || a == 3 {
		t.Fatalf("clvl 14 recovery area %d %v", a, ok)
	}
}
