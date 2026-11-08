package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// TASK-164: an all-mana belt plan is not an incomplete kit for lacking healing potions, for two classes; a plan with a
// healing column is. The emergency hold (no portal source and no healing) stays for every plan.
func TestKitGateAllManaPlan(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			name       string
			tpl, hp    int
			tp         bool
			wantMiss   bool
			wantHold14 bool
		}{
			{"default, no healing", 0, 0, true, true, false},
			{"all mana, no healing", 3, 0, true, false, false},
			{"all health, no healing", 2, 0, true, true, false},
			{"rejuv only, no healing", 6, 0, true, true, false},
			{"all mana, no portal", 3, 0, false, true, true},
		} {
			g, s := kitField(t, 14, tc.hp, tc.tp, 500)
			g.w.set(func(w *world) { w.class = class })
			s, _ = g.w.snapshot()
			g.e.Tune.Resource.BeltPlan = tc.tpl
			k, known := g.r.kitRead(s)
			if !known {
				t.Fatalf("%v %s: kit unknown", class, tc.name)
			}
			if got := len(k.missing()) > 0; got != tc.wantMiss {
				t.Errorf("%v %s: missing %v (%v), want %v", class, tc.name, got, k.missing(), tc.wantMiss)
			}
			if got := g.r.kitHolds(s); got != tc.wantHold14 {
				t.Errorf("%v %s: holds %v, want %v", class, tc.name, got, tc.wantHold14)
			}
		}
	}
}
