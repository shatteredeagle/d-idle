package town_test

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/repair"
	"d2lod-bot/internal/town"
)

// Repair is the player's call (docs/repair.md): with the gate installed and the default manual mode the town repairs
// only after a "Repair now" request, even for a Force plan; a mode that was bought repairs on its own. Both classes run
// the same table-driven path (the town code has no class rule).

func damage(r *rig, pct int32) {
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContEquipment {
			r.W.Stats[u.Addr] = d2.Stats{72: pct, 73: 100}
		}
	}
}

func gated(t *testing.T, class d2.Class, mode string) (*rig, *repair.Controller) {
	t.Helper()
	r := audit(t, 400, 1, true)
	r.W.Player.Class = class
	r.W.RepairBtn = town.RepairAll
	damage(r, 28)
	c, err := repair.New(repair.Config{Mode: func() string { return mode }, Now: r.Env.Clock.Now,
		Gear: func() []repair.Cond { return []repair.Cond{{Slot: "rhand", Pct: 28}} }})
	if err != nil {
		t.Fatal(err)
	}
	r.T.SetRepairGate(c)
	return r, c
}

func repairStep(rep town.ChoreReport) (ran bool) {
	for _, s := range rep.Steps {
		if s.Name == "repair" {
			return s.Ran
		}
	}
	return false
}

func TestRepairManualDoesNothingWithoutRequest(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, _ := gated(t, class, "manual")
		plan := townPlan()
		plan.Force = true // a recovery round forces every chore; the player's policy still wins
		rep, err := r.T.Chores(t.Context(), plan)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Repaired || repairStep(rep) {
			t.Errorf("%v: repaired without a request: %+v", class, rep.Steps)
		}
	}
}

func TestRepairRequestedIsDoneAtTheVisitAndReported(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, c := gated(t, class, "manual")
		if _, err := c.Request(); err != nil {
			t.Fatal(err)
		}
		damage(r, 10) // under the trip knob
		s, _ := r.W.Snapshot()
		if need, why := r.T.NeedsTrip(s); need && strings.Contains(why, "durability") {
			t.Errorf("%v: a request must not send the hero to town by itself: %s", class, why)
		}
		rep := run(t, r)
		if !rep.Repaired {
			t.Fatalf("%v: requested repair not done: %+v", class, rep.Steps)
		}
		if st := c.View().Request; st == nil || st.State != repair.StateDone {
			t.Errorf("%v: request state %+v", class, st)
		}
	}
}

func TestRepairRequestWithoutGoldStaysRequested(t *testing.T) {
	r, c := gated(t, d2.Paladin, "manual")
	r.W.Player.Gold = 0
	c.Request()
	rep := run(t, r)
	if rep.Repaired {
		t.Fatal("repaired with no gold")
	}
	if st := c.View().Request; st == nil || st.State != repair.StateRequested {
		t.Errorf("request state %+v", st)
	}
}

func TestRepairBoughtModeRepairsOnItsOwn(t *testing.T) {
	r, _ := gated(t, d2.Sorceress, "always") // the controller has no ladder here: every mode is open
	if rep := run(t, r); !rep.Repaired {
		t.Fatalf("always mode did not repair: %+v", rep.Steps)
	}
}

func TestRepairModeMakesTheTripManualDoesNot(t *testing.T) {
	trip := func(mode string) bool {
		r, _ := gated(t, d2.Paladin, mode)
		damage(r, 10)
		r.W.Add("tsc", d2.ContInventory, 9, 3)
		s, _ := r.W.Snapshot()
		need, why := r.T.NeedsTrip(s)
		return need && strings.Contains(why, "durability")
	}
	if trip("manual") {
		t.Error("manual mode sent the hero to town for durability")
	}
	if !trip("always") {
		t.Error("always mode did not send the hero to town for durability")
	}
}

// A requested repair is paid before the potions: with just enough gold for the repair, the shop does not spend it first.
func TestRepairRequestedIsPaidBeforeSupplies(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, c := gated(t, class, "manual")
		r.W.Player.Gold = 180
		damage(r, 0)
		c.Request()
		plan := townPlan()
		rep, err := r.T.Chores(t.Context(), plan)
		if err != nil {
			t.Fatal(err)
		}
		if !rep.Repaired {
			t.Errorf("%v: requested repair lost to the supplies: %+v bought %v", class, rep.Steps, rep.Bought)
		}
	}
}

func TestRepairRequestShortOfGoldSaysHowMuch(t *testing.T) {
	r, c := gated(t, d2.Paladin, "manual")
	r.W.Player.Gold = 1
	damage(r, 0)
	c.Request()
	run(t, r)
	if q := c.View().Request; q == nil || q.Short <= 0 || q.State != repair.StateRequested {
		t.Errorf("request %+v", q)
	}
}
