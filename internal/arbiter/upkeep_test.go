package arbiter

import (
	"strings"
	"testing"

	"d2lod-bot/internal/supply"
)

func upkeepPlan(t *testing.T, mana int, purse int) supply.Plan {
	t.Helper()
	in := supply.Inputs{ManaNeed: mana, Capacity: map[supply.Line]int{supply.Healing: 4, supply.Mana: 4}}
	n, err := supply.NeedFor("quest", "monastery_road", in)
	if err != nil {
		t.Fatal(err)
	}
	return supply.Solve(n, supply.Count{Purse: purse, StashKnown: true}, supply.Prices{supply.Healing: 30, supply.Mana: 30, supply.Portal: 100}, supply.Policy{}, 0)
}

// one upkeep source stands for trip and resupply: the plan's open min line sends the hero to town, and the same intent
// serves a build with and without mana potions.
func TestUpkeepSourceOneIntent(t *testing.T) {
	for name, mana := range map[string]int{"caster": 2, "melee": 0} {
		facts := PlanFacts(upkeepPlan(t, mana, 5000))
		if facts[FactPlanBuy] == 0 || facts[FactPlanMinOpen] != 1 || facts[FactPlanVerdict] != PlanReady {
			t.Fatalf("%s: facts %v", name, facts)
		}
		a := New(DefaultConfig(), UpkeepSources()...)
		d := a.Decide(View{Now: 100, Area: "dark_wood", Facts: facts})
		if d.Chosen == nil || d.Chosen.Kind != "upkeep" || d.Chosen.NeedsArea != Town || Family(d.Chosen.Source) != "chores" {
			t.Fatalf("%s: chosen %+v", name, d.Chosen)
		}
		// no trip or resupply source remains
		for _, s := range UpkeepSources() {
			if s.Name() == SrcTrip || s.Name() == SrcResupply {
				t.Errorf("%s: %s still present", name, s.Name())
			}
		}
	}
}

func TestUpkeepShortGoldSellsFirstOrLeavesItToRecovery(t *testing.T) {
	for name, mana := range map[string]int{"caster": 2, "melee": 0} {
		f := PlanFacts(upkeepPlan(t, mana, 0)) // nothing in the purse: short of gold
		f[FactPlanBuy] = 1                     // a stub of what the poor plan could still buy
		if f[FactPlanVerdict] != PlanShort {
			t.Fatalf("%s: verdict %d", name, f[FactPlanVerdict])
		}
		if in := upkeepIntent(View{Facts: f}); in != nil {
			t.Errorf("%s: nothing to sell must leave it to recovery, got %+v", name, in)
		}
		f[FactSellable] = 6
		in := upkeepIntent(View{Facts: f})
		if in == nil || in.Urgency != 0.85 || len(in.Steps) == 0 || in.Steps[0] != "sell_junk" || !strings.Contains(in.Reason, "sell 6") {
			t.Errorf("%s: %+v", name, in)
		}
	}
}

func TestUpkeepTripAndNothingDue(t *testing.T) {
	if in := upkeepIntent(View{Facts: map[string]int{}}); in != nil {
		t.Errorf("nothing due: %+v", in)
	}
	in := upkeepIntent(View{Facts: map[string]int{FactTrip: 1, FactTripEmerg: 1}})
	if in == nil || in.Urgency != 0.9 {
		t.Errorf("emergency: %+v", in)
	}
	if in := upkeepIntent(View{Facts: map[string]int{FactPlanBuy: 1, FactPlanMinOpen: 1, FactPlanVerdict: PlanBlocked}}); in != nil {
		t.Errorf("blocked: %+v", in)
	}
}

// Review 11, N1: declines of "the plan would buy" (the hook said not now) must not silence the same source when a trip
// runs or emergency chores are on: the running trip outranks the recovery farm whatever the back-off says. A second
// decline of the same cause still backs off, and player orders and reflexes are never backed off. Two builds.
func TestBackoffNeverRefusesARunningOrEmergencyTrip(t *testing.T) {
	for name, mana := range map[string]int{"caster": 2, "melee": 0} {
		facts := PlanFacts(upkeepPlan(t, mana, 5000))
		facts[FactRecovery], facts[FactObjective] = 1, 1
		a := New(DefaultConfig(), UpkeepSources()...)
		for _, now := range []float64{0, 30, 90, 210} {
			d := a.Decide(View{Now: now, Area: "dark_wood", Facts: facts})
			if d.Chosen == nil || d.Chosen.Source != SrcUpkeep {
				t.Fatalf("%s t=%v: setup, chosen %+v", name, now, d.Chosen)
			}
			a.Report(*d.Chosen, now, false)
		}
		if d := a.Decide(View{Now: 250, Area: "dark_wood", Facts: facts}); d.Chosen != nil && d.Chosen.Source == SrcUpkeep {
			t.Fatalf("%s: the declined plan intent must stay backed off: %+v", name, d.Chosen)
		}
		for _, emerg := range []int{0, 1} {
			f := map[string]int{FactTrip: 1, FactTripEmerg: emerg, FactInTown: 1}
			for k, v := range facts {
				f[k] = v
			}
			d := a.Decide(View{Now: 300, Area: Town, Facts: f})
			if d.Chosen == nil || d.Chosen.Source != SrcUpkeep {
				t.Fatalf("%s emergency %d: a running trip was refused: chosen %+v, rejected %+v", name, emerg, d.Chosen, d.Rejected)
			}
			a.Report(*d.Chosen, 300, false) // a trip leg that did nothing this tick is not backed off either
			if d := a.Decide(View{Now: 301, Area: Town, Facts: f}); d.Chosen == nil || d.Chosen.Source != SrcUpkeep {
				t.Fatalf("%s emergency %d: a failed trip leg backed the trip off: %+v", name, emerg, d.Rejected)
			}
		}
	}
	a := New(DefaultConfig(), StdSources()...)
	for _, f := range []map[string]int{{FactPlayerOrder: 1}, {FactSafety: 1, FactHostile: 1}} {
		d := a.Decide(View{Now: 0, Area: "dark_wood", Facts: f})
		a.Report(*d.Chosen, 0, false)
		if len(a.fails) != 0 {
			t.Fatalf("a %s intent was backed off", d.Chosen.Class)
		}
	}
}
