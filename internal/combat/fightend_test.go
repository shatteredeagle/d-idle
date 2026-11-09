package combat

import (
	"context"
	"testing"
	"time"
)

func TestNoteFightEndSkipsQuickUnhurtAndNamesHowItEnded(t *testing.T) {
	var got []FightEnd
	f := &Fighter{OnFight: func(e FightEnd) { got = append(got, e) }}
	ft := func(dmg int) *FightTelemetry {
		return &FightTelemetry{Area: 3, Diff: 1, MinLifePct: 40, Damage: dmg, have: true}
	}
	f.noteFightEnd(ft(0), ClearResult{Kills: 1, Elapsed: 2 * time.Second}, nil) // quick and unhurt: skipped
	f.noteFightEnd(ft(0), ClearResult{Kills: 5, Elapsed: 4 * time.Second}, nil)
	f.noteFightEnd(ft(30), ClearResult{Elapsed: time.Second, Potions: 2}, ErrChicken)
	f.noteFightEnd(&FightTelemetry{Area: 3}, ClearResult{Elapsed: time.Minute}, nil) // never sampled: skipped
	if len(got) != 2 || got[0].Kills != 5 || got[0].Left != "" || got[1].Left != "chicken" || got[1].Potions != 2 || got[1].MinLifePct != 40 {
		t.Fatalf("%+v", got)
	}
	(&Fighter{}).noteFightEnd(ft(9), ClearResult{}, nil) // no hook: nothing
}

func TestRealFightCallsOnFightOnce(t *testing.T) {
	h := newHarness(t)
	var got []FightEnd
	h.f.OnFight = func(e FightEnd) { got = append(got, e) }
	h.s.stock(itHP1, 0, 2)
	h.s.addMon(10, clsFallen, 120, 100, 90, 22, 6)
	if _, err := h.f.Clear(context.Background(), ClearOpts{}); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Kills != 1 || got[0].Damage == 0 || got[0].MinLifePct <= 0 || got[0].MinLifePct >= 100 {
		t.Fatalf("want one event for the one fight: %+v", got)
	}
}
