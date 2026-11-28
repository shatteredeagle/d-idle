package arbiter

import (
	"strings"
	"testing"
)

type fnSource struct {
	name string
	f    func(View) *Intent
}

func (s fnSource) Name() string           { return s.name }
func (s fnSource) Propose(v View) *Intent { return s.f(v) }

// L5 (live review 2026-11-28): the hero leaves field F for a town trip, the trip is done (Report true), and the quest
// step that needs F is then refused: "travel budget: going to F undoes F->town 0s ago and nothing progressed since".
// A finished trip is progress of its own. An unfinished one still keeps the budget.
func TestTravelBudgetAcceptsTheFieldAfterAFinishedTownTrip(t *testing.T) {
	for _, finished := range []bool{true, false} {
		const field = "Tamoe Highland"
		trip := true
		quest := fnSource{"quest", func(View) *Intent {
			return &Intent{Kind: "step", Class: Objective, NeedsArea: field, Reason: "quest step"}
		}}
		upkeep := fnSource{"upkeep", func(v View) *Intent {
			if !trip {
				return nil
			}
			return &Intent{Kind: "trip", Class: Upkeep, NeedsArea: Town, Reason: "equipment at 0% durability"}
		}}
		a := New(DefaultConfig(), quest, upkeep)
		d := a.Decide(View{Now: 0, Area: field})
		if d.Chosen == nil || d.Chosen.Kind != "trip" {
			t.Fatalf("want the trip chosen in the field, got %+v", d.Chosen)
		}
		a.Decide(View{Now: 30, Area: Town}) // the area change is recorded
		trip = false
		if finished {
			a.Report(*d.Chosen, 31, true)
		}
		d = a.Decide(View{Now: 32, Area: Town})
		refused := false
		for _, r := range d.Rejected {
			refused = refused || strings.Contains(r.Why, "travel budget")
		}
		if got := d.Chosen != nil && d.Chosen.Kind == "step"; got != finished || refused == finished {
			t.Fatalf("finished=%v: chosen %+v, refused by the travel budget: %v", finished, d.Chosen, refused)
		}
	}
}
