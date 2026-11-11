package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/repair"
	"d2lod-bot/internal/town"
)

// The repair trip (docs/repair.md): a requested repair that the pack cannot pay but the stash can takes the gold out of
// the stash first. These need the game tables (they skip without them, like the other repair tests of this package).

func fundedRepair(t *testing.T, class d2.Class, purse, bank int) (*rig, *withdrawGold, *repair.Controller) {
	t.Helper()
	r, w := withdrawRig(t, purse, bank, 11, true)
	r.W.Player.Class = class
	r.W.RepairBtn = town.RepairAll
	damage(r, 28)
	c, err := repair.New(repair.Config{Now: r.Env.Clock.Now,
		Gear:     func() []repair.Cond { return []repair.Cond{{Slot: "rhand", Pct: 28}} },
		Gold:     func() int { return r.W.Player.Gold },
		Estimate: func() int { s, _ := r.W.Snapshot(); return r.T.RepairEstimate(s) }})
	if err != nil {
		t.Fatal(err)
	}
	r.T.SetRepairGate(c)
	if _, err := c.Request(); err != nil {
		t.Fatal(err)
	}
	c.FarmReached() // a farm kind request becomes the trip; a repair kind one stays as it is
	c.BeginTrip(r.Env.Clock.Now())
	c.SetPhase(repair.PhaseRepairing, "")
	return r, w, c
}

func TestRepairWithdrawsFromTheStashWhenThePackIsShort(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, w, c := fundedRepair(t, class, 5, 5000)
		rep, err := r.T.Chores(context.Background(), town.ChorePlan{Repair: true, Gold: false})
		if err != nil {
			t.Fatal(err)
		}
		withdrew := false
		for _, m := range w.moves {
			withdrew = withdrew || m < 0
		}
		if !withdrew || !rep.Repaired {
			t.Fatalf("%v: withdrew %v repaired %v moves %v steps %+v", class, withdrew, rep.Repaired, w.moves, rep.Steps)
		}
		if q, _ := c.Active(); q.Phase != repair.PhaseReturning {
			t.Errorf("%v: the trip must be walking back: %+v", class, q)
		}
	}
}

func TestRepairStashTooSmallStaysShortAndWithdrawsNothing(t *testing.T) {
	r, w, c := fundedRepair(t, d2.Paladin, 0, 1)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Repair: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range w.moves {
		if m < 0 {
			t.Errorf("withdrew %v for a repair the stash cannot pay", w.moves)
		}
	}
	if rep.Repaired {
		t.Fatal("repaired without the gold")
	}
	if q, _ := c.Active(); q.Short <= 0 {
		t.Errorf("the shortfall must be recorded for the bot: %+v", q)
	}
}
