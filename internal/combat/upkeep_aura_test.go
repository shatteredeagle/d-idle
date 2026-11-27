package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
)

// auraDropTick brings the aura up through KeepAura, then drops it the way a non-aura right pick does (the right hand
// moves to another skill and the active aura goes off) and calls KeepAura again inside combat.auraCheckMs.
// It returns whether the second call re-applied an aura, and the active aura after it.
func auraDropTick(t *testing.T, mode, crowd int) (*harness, *handGame, bool) {
	t.Helper()
	h, g := newHandGame(t)
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	h.f.auraCrowd = crowd
	ctx := context.Background()
	g.aura, g.p.RightSkill = 0, 0
	s, _ := g.snap()
	if ok, err := h.f.KeepAura(ctx, s, &Target{}, true); err != nil || !ok {
		t.Fatalf("mode %d: opening apply ok=%v err=%v", mode, ok, err)
	}
	g.p.RightSkill, g.aura = 107, 0 // a non-aura right pick: the aura is off
	s, _ = g.snap()
	ok, err := h.f.KeepAura(ctx, s, &Target{}, false)
	if err != nil {
		t.Fatal(err)
	}
	return h, g, ok
}

// Mode 2 re-picks the dropped aura at once, and it is the profile's choice in both cases (Might alone, Holy Fire in a
// pack); modes 0 and 1 wait for the check interval, and mode 1 logs the disagreement once.
func TestAuraUpkeepRepickAfterDrop(t *testing.T) {
	for _, tc := range []struct {
		crowd int
		want  int
	}{{0, 98}, {3, 102}} { // 98 Might, 102 Holy Fire in the palCatalog fixture
		h, g, ok := auraDropTick(t, 2, tc.crowd)
		if !ok || g.aura != tc.want || g.p.RightSkill != tc.want {
			t.Fatalf("mode 2 crowd %d: re-pick ok=%v aura %d right %d, want %d", tc.crowd, ok, g.aura, g.p.RightSkill, tc.want)
		}
		if h.f.UpkeepShadowDiffs() != 0 {
			t.Fatalf("mode 2 logs no shadow lines, got %d", h.f.UpkeepShadowDiffs())
		}
		for _, mode := range []int{0, 1} {
			h, g, ok := auraDropTick(t, mode, tc.crowd)
			if ok || g.aura != 0 {
				t.Fatalf("mode %d crowd %d: the old code waits for the interval, got ok=%v aura %d", mode, tc.crowd, ok, g.aura)
			}
			s, _ := g.snap()
			h.f.KeepAura(context.Background(), s, &Target{}, false) // the same lasting disagreement: logged once
			if want := map[int]int{0: 0, 1: 1}[mode]; h.f.UpkeepShadowDiffs() != want {
				t.Fatalf("mode %d: shadow diffs %d, want %d", mode, h.f.UpkeepShadowDiffs(), want)
			}
		}
	}
}

// Without an active-aura read the driver trusts the Skills.txt aurastate once learned: present = up, gone past the
// grace = re-pick, though the right hand still holds the aura (the d2d load case of the probe).
func TestAuraUpkeepLearnedState(t *testing.T) {
	h, g := newHandGame(t)
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	h.f.auraFn = func(*d2.Player) (int, bool) { return 0, false }
	const mightState = 33 // fixture value; the driver reads it from the row
	h.f.cat.(MapCatalog).Skills[98].StateID = mightState
	ctx := context.Background()
	g.p.RightSkill = 0
	s, _ := g.snap()
	if ok, _ := h.f.KeepAura(ctx, s, &Target{}, true); !ok || g.p.RightSkill != 98 {
		t.Fatalf("opening apply: right %d", g.p.RightSkill)
	}
	setState(g.p, mightState, true)
	s, _ = g.snap()
	if ok, _ := h.f.KeepAura(ctx, s, &Target{}, true); ok {
		t.Fatal("state up: no re-pick")
	}
	setState(g.p, mightState, false)
	h.s.Sleep(2 * upkeepGrace)
	s, _ = g.snap()
	if ok, _ := h.f.KeepAura(ctx, s, &Target{}, false); !ok {
		t.Fatal("learned state gone: the driver must re-pick")
	}
}

// A class without an aura in its profile never reaches the aura driver: no shadow line, no input.
func TestAuraUpkeepNonAuraClassSilent(t *testing.T) {
	b := upkeepBuilds()[1] // Sorceress
	h, w := upkeepSetup(t, b, 1)
	for i := 0; i < 5; i++ {
		s, _ := h.f.snapFn()
		if ok, err := h.f.KeepAura(context.Background(), s, &Target{}, i == 0); ok || err != nil {
			t.Fatalf("sorceress aura ok=%v err=%v", ok, err)
		}
	}
	if h.f.UpkeepShadowDiffs() != 0 || len(w.casts) != 0 {
		t.Fatalf("sorceress: shadow diffs %d casts %d", h.f.UpkeepShadowDiffs(), len(w.casts))
	}
}
