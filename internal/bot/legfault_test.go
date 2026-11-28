package bot

import (
	"context"
	"fmt"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/nav"
)

// L1 (live review 2026-11-28): the way out of the level the hero stands in is cut off. That is one fact about where it
// stands, so it is charged once: the repeat leaves by town portal instead of banning the next target (live: ten zones
// banned in 80 s from one cell of Tower Cellar Level 3, then the failsafe ladder).
func TestCutOffExitIsChargedOncePerLeg(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := campaignRig(t, &fakeQuests{}, 12)
		const pocket = 9
		g.mover.exitErr = fmt.Errorf("%w: gate to level 3 is on the map but not reachable (cause=cut_off, no way out found)", nav.ErrNoPath)
		g.mover.exitErrIn = pocket
		g.w.set(func(w *world) { w.area, w.class = pocket, class })
		g.r.poll()
		acts := taskActs{r: g.r, c: control{goal: &goals.Goal{}}}
		fails := 0
		for i := 0; i < 5; i++ {
			if err := acts.LevelTo(context.Background(), 14); err != nil {
				fails++
			}
			g.r.poll()
			if i == 1 && g.w.count("totown") != 1 {
				t.Fatalf("%v: the second failure of the same leg must leave by town portal, calls: %s", class, g.w.log())
			}
		}
		if n := len(g.r.avoid); n > 1 {
			t.Errorf("%v: %d zones skipped for one cut-off exit, want one: %v", class, n, g.r.avoidWhy)
		}
		if fails > 1 {
			t.Errorf("%v: %d failed slices feed the failsafe ladder, want at most one", class, fails)
		}
	}
}

// L3: a quest step's travel that fails twice on the same exit steps off by town portal before the next retry.
func TestQuestTravelStepsOffAfterTheSameExitFailsTwice(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := campaignRig(t, &fakeQuests{}, 18)
		const pocket = 9
		g.mover.exitErr = fmt.Errorf("%w: gate to level 3 is on the map but not reachable (cause=cut_off, no way out found)", nav.ErrNoPath)
		g.mover.exitErrIn = pocket
		g.w.set(func(w *world) { w.area, w.class = pocket, class })
		g.r.poll()
		acts := taskActs{r: g.r, c: control{goal: &goals.Goal{}}}
		if err := acts.Travel(context.Background(), 3); err == nil || g.w.count("totown") != 0 {
			t.Fatalf("%v: first try: err %v, calls %s", class, err, g.w.log())
		}
		if err := acts.Travel(context.Background(), 3); err == nil || g.w.count("totown") != 1 {
			t.Fatalf("%v: the second failure on the same exit must leave by town portal: err %v, calls %s", class, err, g.w.log())
		}
	}
}
