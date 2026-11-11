package watch

import (
	"testing"
	"time"

	"d2lod-bot/internal/goals"
)

// The runner reports experience inside the level (observe.go updateCharacter); /api/hero must show totals and a
// progress that is not null (live regression 2026-11-11: the readout showed "-").
func TestHeroXPFromRunnerStatus(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		r := newRig(t, "sorceress")
		r.sim.Hub.xpSeen = false // forget the simulator's own first sample
		b := &GoalsOrders{Hub: r.sim.Hub}
		// level 11 starts at 72144 and level 12 at 90180 (experience.txt): 7307 into a 18036 wide level
		b.OnStatus(goals.Status{State: goals.RunnerRunning, Character: goals.Character{Class: class, Level: 11, XP: 7307, XPNext: 18036, Area: "Black Marsh", AreaID: 6}})
		var v HeroView
		r.get(HeroRoute, &v)
		if v.XP != 79451 || v.XPLevel != 72144 || v.XPNext != 90180 {
			t.Fatalf("%s: xp %d level %d next %d", class, v.XP, v.XPLevel, v.XPNext)
		}
		if v.XPPct == nil || *v.XPPct != 40.5 {
			t.Fatalf("%s: pct %v", class, v.XPPct)
		}
		if v.XPPerHour != nil {
			t.Errorf("%s: rate before the first interval: %v", class, *v.XPPerHour)
		}
		r.sim.Step(time.Hour)
		b.OnStatus(goals.Status{State: goals.RunnerRunning, Character: goals.Character{Class: class, Level: 11, XP: 7307 + 6000, XPNext: 18036, Area: "Black Marsh", AreaID: 6}})
		r.get(HeroRoute, &v)
		if v.XPPerHour == nil || v.XPPerHour != nil && *v.XPPerHour < 5000 || *v.XPPerHour > 6000 {
			t.Errorf("%s: xp/h %v xp %d", class, func() float64 {
				if v.XPPerHour == nil {
					return -1
				}
				return *v.XPPerHour
			}(), v.XP)
		}
	}
}

func TestAbsoluteXPUnknownLevel(t *testing.T) {
	if xp, next := AbsoluteXP(0, 5, 9); xp != 5 || next != 9 {
		t.Errorf("%d %d", xp, next)
	}
	if xp, next := AbsoluteXP(1, 30, 0); xp != 30 || next != 0 {
		t.Errorf("level 1 without span: %d %d", xp, next)
	}
}
