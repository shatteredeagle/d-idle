package combat

// Scenario corpus, combat half (docs/scenarios.md): recorded fight situations replayed through Fighter.Clear as
// end-state tests. No knob guards these fixes (they are control-flow fixes), so there is no knob-off subtest: the
// scenario is proven by the harness counters the old code could not satisfy (a spin shows as walks without game time).

import (
	"context"
	"testing"
	"time"
)

// Situation: a crowd of ten with life at 34 percent, no portal scroll and no potion; every step-away walk fails at
// once with "nav: no path: empty path" and no game time passes. Evidence (docs/probes/navsoak-compare.md and the
// soakfull seed 11 and runab seed 42 logs cited in internal/combat/navstall_test.go): before the fix the fight loop
// spun forever without a tick; after it, navsoak counted 0 "combat no progress, holding" and the clock never froze
// (VERIFIED, 6455 "nav: no path" lines survived as noise). End state: the game clock advanced, the loop ran to its
// own timeout (20 s of game time) with a bounded number of failed walks, so it held or fought in place instead of spinning.
func TestScenario_CombatPinNoPath(t *testing.T) {
	h := newHarness(t)
	h.s.noTP = true
	h.s.life = 34
	h.s.healAmount = 0
	for i := 0; i < 10; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 102+float64(i%3), 98+float64(i%4), 4000, 0, 0)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mv := &emptyPathMover{Mover: h.s, cancel: cancel, limit: 5000}
	h.f.mv = mv
	t0 := h.s.Now()
	_, _ = h.f.Clear(ctx, ClearOpts{Timeout: 20 * time.Second})
	if mv.walks >= mv.limit {
		t.Fatalf("spin: %d failed walks", mv.walks)
	}
	if el := h.s.Now().Sub(t0); el < 15*time.Second {
		t.Fatalf("game time moved only %v", el)
	}
	t.Logf("walks %d casts %d game time %v", mv.walks, h.s.casts, h.s.Now().Sub(t0))
}

// Situation: three champions in view of a clvl 5 character, then two, then a clvl 8 one. Evidence (docs/probes/
// navsoak-compare.md: the first soak death was a clvl 4 pack fight, not a champion, so this pin is INFERRED from the
// champion rules of the combat-loop spec, not from a log): a low-level character avoids a champion group of three
// and fights smaller groups; from clvl 8 it fights the group. End state: the avoid flags of the hostile list.
func TestScenario_ChampionGroup(t *testing.T) {
	avoided := func(lvl, n int) int {
		s := baseSnap()
		for i := 0; i < n; i++ {
			s.Monsters = append(s.Monsters, champUnit(uint32(2+i), 108+2*i))
		}
		hc := hctx()
		hc.ClassLvl = lvl
		a := 0
		for _, h := range Hostiles(s, hc) {
			if h.Avoid {
				a++
			}
		}
		return a
	}
	for _, c := range []struct{ lvl, n, want int }{{5, 3, 3}, {5, 2, 0}, {5, 1, 0}, {8, 3, 0}} {
		if got := avoided(c.lvl, c.n); got != c.want {
			t.Errorf("%d champions at clvl %d: %d avoided, want %d", c.n, c.lvl, got, c.want)
		}
	}
}
