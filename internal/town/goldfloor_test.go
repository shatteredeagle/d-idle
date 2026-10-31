package town_test

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

func boughtLike(rep town.ChoreReport, sub string) int {
	n := 0
	for k, v := range rep.Bought {
		if strings.Contains(k, sub) {
			n += v
		}
	}
	return n
}

// goldfloor: gold 209 and 1 healing potion buys potions and no portal scroll; the knob at 0 restores the scroll.
func TestNoTPRebuyWhenPoor(t *testing.T) {
	for _, off := range []bool{false, true} {
		r := audit(t, 209, 8, false)
		r.W.Add("hp1", d2.ContBelt, 0, 0)
		if off {
			r.Env.Tune.Town.TripMinSpend = 0
		}
		rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Scrolls: true})
		if err != nil {
			t.Fatal(err)
		}
		tp := boughtLike(rep, "Town Portal")
		if !off && (tp != 0 || boughtLike(rep, "Healing") == 0) {
			t.Fatalf("poor visit bought %v", rep.Bought)
		}
		if off && tp == 0 {
			t.Fatalf("tripMinSpend 0 bought %v, want the old scroll", rep.Bought)
		}
	}
}

// goldfloor: below clvl 10 one mana potion per visit and no mana trip; clvl 12 and the knob at 0 are unchanged.
func TestManaCapBelowClvl10(t *testing.T) {
	r := audit(t, 600, 6, false)
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	r.W.Add("hp1", d2.ContBelt, 1, 0)
	r.W.Add("hp1", d2.ContBelt, 2, 0)
	r.W.Add("hp1", d2.ContBelt, 3, 0)
	s, _ := r.W.Snapshot()
	if bad, why := r.T.NeedsTrip(s); bad && strings.Contains(why, "mana") {
		t.Fatalf("clvl 6 mana belt raised a trip: %s", why)
	}
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	if n := boughtLike(rep, "Mana"); n != 1 {
		t.Fatalf("clvl 6 bought %d mana potions (%v), want 1", n, rep.Bought)
	}
	r = audit(t, 600, 6, false)
	r.Env.Tune.Town.MPCapLevel = 0
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	r.W.Add("hp1", d2.ContBelt, 1, 0)
	if rep, _ = r.T.Chores(context.Background(), town.ChorePlan{Potions: true}); boughtLike(rep, "Mana") < 2 {
		t.Fatalf("mpCapLevel 0 bought %v, want the old fill", rep.Bought)
	}
}

// PlannedSpend never exceeds the gold: the trip floor sees a 25-gold purse as a 25-gold plan.
func TestPlannedSpendCappedByGold(t *testing.T) {
	r := audit(t, 25, 6, false)
	s, _ := r.W.Snapshot()
	if n := r.T.PlannedSpend(s); n > 25 || n == 0 {
		t.Fatalf("planned %d with 25 gold", n)
	}
}
