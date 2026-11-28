package arbiter

import "testing"

// Review 11 N1c (card sweep2): a declined intent ("the hook had nothing to do") is asked again after DeclineMax at the
// latest however often it declined; a failure after cost still backs off to BackoffMax. Before: the bot reported every
// decline as a failure and upkeep was silent for up to 1800 s. Two builds.
func TestDeclineBackoffIsCapped(t *testing.T) {
	for name, mana := range map[string]int{"caster": 2, "melee": 0} {
		facts := PlanFacts(upkeepPlan(t, mana, 5000))
		facts[FactRecovery], facts[FactObjective] = 1, 1
		for _, decline := range []bool{true, false} {
			a := New(DefaultConfig(), UpkeepSources()...)
			now, longest := 0.0, 0.0
			for i := 0; i < 8; i++ {
				d := a.Decide(View{Now: now, Area: "dark_wood", Facts: facts})
				if d.Chosen == nil || d.Chosen.Source != SrcUpkeep {
					t.Fatalf("%s: setup at t=%v, chosen %+v", name, now, d.Chosen)
				}
				if decline {
					a.Decline(*d.Chosen, now)
				} else {
					a.Report(*d.Chosen, now, false)
				}
				b := a.fails[d.Chosen.backoffKey()]
				longest = max(longest, b.until-now)
				now = b.until
			}
			if decline && longest != DeclineMax {
				t.Fatalf("%s: eight declines backed off for %v s, want the cap %v", name, longest, float64(DeclineMax))
			}
			if !decline && longest != DefaultConfig().BackoffMax {
				t.Fatalf("%s: eight failures backed off for %v s, want %v", name, longest, DefaultConfig().BackoffMax)
			}
		}
	}
}

// Review 14 M1: declines and failures count apart. One real failure after six declines backs off for BackoffBase, not
// BackoffMax. Two builds.
func TestFailureAfterDeclinesStartsAtBase(t *testing.T) {
	for name, mana := range map[string]int{"caster": 2, "melee": 0} {
		facts := PlanFacts(upkeepPlan(t, mana, 5000))
		facts[FactRecovery], facts[FactObjective] = 1, 1
		a := New(DefaultConfig(), UpkeepSources()...)
		now := 0.0
		var in Intent
		for i := 0; i < 6; i++ {
			d := a.Decide(View{Now: now, Area: "dark_wood", Facts: facts})
			if d.Chosen == nil || d.Chosen.Source != SrcUpkeep {
				t.Fatalf("%s: setup at t=%v, chosen %+v", name, now, d.Chosen)
			}
			in = *d.Chosen
			a.Decline(in, now)
			now += 120
		}
		a.Report(in, now, false)
		if got := a.fails[in.backoffKey()].until - now; got != DefaultConfig().BackoffBase {
			t.Fatalf("%s: a first failure after six declines backs off %v s, want BackoffBase %v", name, got, DefaultConfig().BackoffBase)
		}
	}
}
