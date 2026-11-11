package main

import (
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/senses"
)

func TestActivityFor(t *testing.T) {
	for code, want := range map[string]uint8{"walk": senses.ActWalk, "fight": senses.ActCombat, "town_chores": senses.ActTown, "farm_gold": senses.ActFarm,
		"guard_exit": senses.ActFlee, "wait_heal": senses.ActRest, "": senses.ActWalk, "never_heard_of_it": senses.ActWalk} {
		if got := activityFor(code, false); got != want {
			t.Errorf("%q: %d, want %d", code, got, want)
		}
	}
	if activityFor("town_trip", true) != senses.ActCombat {
		t.Error("a fighter with a target is combat whatever the plan says")
	}
	// every code of the whyline has a mapping decision: the enum stays inside the legend
	if len(senses.ActivityNames) != 6 {
		t.Fatal("legend and enum out of step")
	}
}

func TestHistoryEvent(t *testing.T) {
	k, txt, big, ok := historyEvent(goals.Event{Type: goals.EvKill, Monster: "Fallen Shaman", Boss: true})
	if !ok || k != senses.EvKill || txt != "Fallen Shaman killed" || !big {
		t.Fatalf("%s %s %v %v", k, txt, big, ok)
	}
	if _, _, _, ok := historyEvent(goals.Event{Type: goals.EvAreaChange}); ok {
		t.Fatal("area change is not a marker (zone entries come from the trail)")
	}
	if k, _, big, _ := historyEvent(goals.Event{Type: goals.EvDrop, Item: &goals.Item{Name: "Shako", Quality: "unique"}}); k != senses.EvLoot || !big {
		t.Fatal("a unique drop is a big loot marker")
	}
	fi := fightInfo(combat.FightSummary{End: "clear", Kills: 3, MinLifePct: 88, LifePct: 92, Sec: 2.14})
	if fi.End != "clear" || fi.Sec != 2.1 || fi.MinLife != 88 {
		t.Fatalf("%+v", fi)
	}
}
