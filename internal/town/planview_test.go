package town

import (
	"testing"

	"d2lod-bot/internal/supply"
)

func TestPlanViewBeforeAndAfterAVisit(t *testing.T) {
	tw := &Town{}
	if tw.PlanView().Known {
		t.Fatal("no visit solved a plan: NoView")
	}
	n, err := supply.NeedFor("town", "field", supply.Inputs{ManaNeed: 2, Capacity: map[supply.Line]int{supply.Healing: 4, supply.Mana: 4}})
	if err != nil {
		t.Fatal(err)
	}
	c := supply.Count{Purse: 100}
	tw.lastPlan.Store(&lastPlan{plan: supply.Solve(n, c, supply.Prices{supply.Healing: 30, supply.Mana: 30, supply.Portal: 100}, supply.Policy{}, 0), count: c})
	stops := []string{"akara"}
	tw.lastStops.Store(&stops)
	v := tw.PlanView()
	if !v.Known || len(v.Stops) != 1 || v.Budget.Purse != 100 {
		t.Fatalf("view %+v", v)
	}
}
