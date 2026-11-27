package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

func packAt(w *nworld, x, y, n int) {
	for i := 0; i < n; i++ {
		w.snap.Monsters = append(w.snap.Monsters, mon(uint32(600+len(w.snap.Monsters)), 1, x+i%2, y+i%3))
	}
}

// Scenario combat.whirlwind-clears (offline part): a Barbarian with Whirlwind spins a leg through the pack at 2, only
// logs at 1 and does nothing at 0; the leg ends past the pack, within the row's range.
func TestWhirlLegBarbarian(t *testing.T) {
	for _, mode := range []int{0, 1, 2} {
		h, w := moveWorld(t, d2.Barbarian, "", mode, idWhirl)
		packAt(w, 107, 100, 5)
		acted, err := h.f.whirlTick(w.snap, newFightState())
		if err != nil {
			t.Fatal(err)
		}
		if acted != (mode == 2) || (len(w.casts) > 0) != (mode == 2) {
			t.Fatalf("mode %d: acted %v casts %v", mode, acted, w.casts)
		}
		if mode == 2 {
			at := w.castAt[0]
			if d := pdist(input.Point{X: 100, Y: 100}, at); at.X < 108 || d > 10 || h.f.WhirlLegs() != 1 {
				t.Fatalf("leg to %v (%v away), legs %d", at, d, h.f.WhirlLegs())
			}
		}
		if mode == 1 && h.f.MoveShadowCasts() != 1 {
			t.Fatalf("shadow %d", h.f.MoveShadowCasts())
		}
	}
}

// The life floor aborts: under whirlAbortLifePct no leg starts; a pack too small or too far gets none either; an unlearned
// skill is a refusal, not a cast.
func TestWhirlHolds(t *testing.T) {
	h, w := moveWorld(t, d2.Barbarian, "", 2, idWhirl)
	packAt(w, 107, 100, 5)
	w.snap.Player.Life = 150 // 37 percent, row floor 50
	if acted, _ := h.f.whirlTick(w.snap, newFightState()); acted || len(w.casts) > 0 || h.f.WhirlAborts() != 1 {
		t.Fatalf("spun under the life floor: %v aborts %d", w.casts, h.f.WhirlAborts())
	}
	h, w = moveWorld(t, d2.Barbarian, "", 2, idWhirl)
	packAt(w, 107, 100, 2) // below whirlPack
	if acted, _ := h.f.whirlTick(w.snap, newFightState()); acted {
		t.Fatal("spun against two")
	}
	h, w = moveWorld(t, d2.Barbarian, "", 2, idWhirl)
	packAt(w, 140, 100, 5) // beyond the range
	if acted, _ := h.f.whirlTick(w.snap, newFightState()); acted {
		t.Fatal("spun at a far pack")
	}
	h, w = moveWorld(t, d2.Barbarian, "", 2) // not learned
	packAt(w, 107, 100, 5)
	if acted, _ := h.f.whirlTick(w.snap, newFightState()); acted || len(w.casts) > 0 {
		t.Fatal("cast an unknown skill")
	}
}

// Legs move between pack positions: the second leg turns to the other pack, not back into the one just crossed.
func TestWhirlLegsAlternate(t *testing.T) {
	h, w := moveWorld(t, d2.Barbarian, "", 2, idWhirl)
	packAt(w, 106, 100, 4)
	packAt(w, 94, 100, 3)
	for i := 0; i < 2; i++ {
		if acted, err := h.f.whirlTick(w.snap, newFightState()); !acted || err != nil {
			t.Fatalf("leg %d: %v %v", i, acted, err)
		}
	}
	if w.castAt[0].X <= 100 || w.castAt[1].X >= 100 {
		t.Fatalf("legs %v did not alternate across the packs", w.castAt)
	}
}

// Second build: the planner is data driven. A Paladin profile given a whirl row for a different skill and numbers spins
// by its own row (the test row is not shipped data).
func TestWhirlSecondBuild(t *testing.T) {
	h, w := moveWorld(t, d2.Paladin, "", 2, idZeal)
	pr := h.f.profileFor(w.snap.Player)
	saved := pr.Moves
	t.Cleanup(func() { pr.Moves = saved })
	pr.Moves = append(append([]MoveRule(nil), saved...), MoveRule{Name: "Zeal", Uses: []string{MoveWhirl}, Range: 6, MinHop: 2,
		WhirlPack: 2, WhirlRadius: 3, WhirlAbortLifePct: 70, Source: "U test row"})
	if err := validateMoves([]Profile{*pr}); err != nil {
		t.Fatal(err)
	}
	packAt(w, 104, 100, 3)
	if acted, err := h.f.whirlTick(w.snap, newFightState()); !acted || err != nil || len(w.casts) != 1 {
		t.Fatalf("paladin leg: %v %v %v", acted, err, w.casts)
	}
	if d := pdist(input.Point{X: 100, Y: 100}, w.castAt[0]); d > 6 {
		t.Fatalf("leg %v beyond the row range", d)
	}
	w.snap.Player.Life = 250 // 62 percent, this row's floor is 70
	if acted, _ := h.f.whirlTick(w.snap, newFightState()); acted || h.f.WhirlAborts() != 1 {
		t.Fatalf("paladin spun under its own floor")
	}
}

func TestValidateWhirl(t *testing.T) {
	ok := MoveRule{Name: "Whirlwind", Uses: []string{MoveWhirl}, Range: 10, MinHop: 3, WhirlPack: 3, WhirlRadius: 4, WhirlAbortLifePct: 50, Source: "U test"}
	if err := validateMoves([]Profile{{Class: "x", Moves: []MoveRule{ok}}}); err != nil {
		t.Fatal(err)
	}
	for _, mod := range []func(*MoveRule){
		func(r *MoveRule) { r.WhirlPack = 1 },
		func(r *MoveRule) { r.WhirlRadius = 0 },
		func(r *MoveRule) { r.WhirlAbortLifePct = 0 },
		func(r *MoveRule) { r.WhirlAbortLifePct = 101 },
	} {
		r := ok
		mod(&r)
		if validateMoves([]Profile{{Class: "x", Moves: []MoveRule{r}}}) == nil {
			t.Fatalf("accepted %+v", r)
		}
	}
}
