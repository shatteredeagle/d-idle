package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/admit"
	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/questboard"
)

// Review 13 L2 (card sweep2): the leveling picks never offer the area of a boss whose chunk the board parked. Before:
// the zone choice took Tower Cellar Level 4 while forgotten_tower/countess was parked.
func TestParkedBossChunkAreaIsNoLevelingCandidate(t *testing.T) {
	const burial = 17
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		t.Run(class.String(), func(t *testing.T) {
			g, x, st := smBoardRig(t, class, nil)
			g.w.set(func(w *world) { w.level = 20 })
			areas := map[string]int{"blood_raven": burial, "countess": 25}
			g.r.SetQuestMargin(func(s *d2.Snapshot, boss, diff string, potions int) (questboard.Hero, *questboard.Encounter, bool) {
				h := questboard.Hero{Class: class.String(), Level: s.Player.Level, MercHired: true, MercAlive: true, Potions: 8}
				if boss == "" {
					return h, nil, true
				}
				return h, &questboard.Encounter{Name: boss, Kind: "boss", Best: 3, Low: 2, Known: true, Area: areas[boss]}, true
			})
			g.run()
			defer g.finish()
			g.waitFor("snapshot", func() bool { return g.r.QuestClass() != "" })
			if err := x.Accept("sisters_burial_grounds", "blood_raven"); err != nil {
				t.Fatal(err)
			}
			ex := map[int]bool{}
			g.r.parkedBossAreas(ex)
			if len(ex) != 0 {
				t.Fatalf("a running boss chunk excludes nothing, got %v", ex)
			}
			d := questboard.Decision{Quest: "sisters_burial_grounds", Chunk: "blood_raven"}
			x.Report(d, questboard.StepResult{Outcome: questboard.StepDied, Why: "Blood Raven"})
			x.Report(d, questboard.StepResult{Outcome: questboard.StepDied, Why: "Blood Raven"})
			if !boardParked(t, st) {
				t.Fatal("not parked")
			}
			g.clock.Sleep(parkedBossEvery)
			ex = map[int]bool{}
			g.r.parkedBossAreas(ex)
			if !ex[burial] || len(ex) != 1 {
				t.Fatalf("want only the parked boss area %d, got %v", burial, ex)
			}
			ex = map[int]bool{}
			g.r.guardedAreasNoFarm(g.r.latest(), ex) // the exclusion every leveling, farm and recovery pick starts from
			if !ex[burial] {
				t.Fatalf("the pickers still offer area %d: %v", burial, ex)
			}
			x.Unblock("sisters_burial_grounds", "blood_raven")
			g.clock.Sleep(parkedBossEvery)
			ex = map[int]bool{}
			g.r.parkedBossAreas(ex)
			if ex[burial] {
				t.Fatal("Try again must give the area back")
			}
		})
	}
}

// Review 11 N8 (card sweep2): every road row's level and life floor is asked before any row's kit. Before: a caster under
// Duriel's life floor and short of potions was told "kit" by the tombs row, made the trip and was then refused for life.
// The function beside the old judge answers the same (shadow differ 0). The melee floor sits on the first row: control.
func TestEarlyRoadAsksEveryFloorBeforeAnyKit(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := admitRig(t, class, 30, 46, 176, 0, false, 0)
		s := g.r.latest()
		why, kitOnly := g.r.earlyRouteCheck(s, 73)
		if !strings.Contains(why, "life 220 needed") || kitOnly {
			t.Fatalf("%v: want the life floor and no kit trip, got %q kitOnly=%v", class, why, kitOnly)
		}
		if got := g.r.admitEnter(s, 73); got != why {
			t.Fatalf("%v: admitEnter %q, old %q", class, got, why)
		}
		if ex := g.r.admissionExplain(); ex.Differ != 0 {
			t.Fatalf("%v: counters %+v, want differ 0", class, ex)
		}
	}
}

// Review 7 admission (card sweep2): the kit of an early road is an entry rule in the function too. Before: the old judge
// let a hero inside the Maggot Lair go on with a thinned kit and the function refused it (a new-only cell at every ask),
// so bot.admission 1 would have brought back the park of the 2026-11-28 lab night.
func TestAdmissionKitIsAnEntryRuleLikeTheOldJudge(t *testing.T) {
	const lair1, lair3 = 62, 64
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, admission := range []int{0, 1} {
			g := admitRig(t, class, 30, lair1, 1400, 4, false, admission)
			g.w.set(func(w *world) { w.maxL = 1400 })
			g.r.poll()
			s := g.r.latest()
			if old := g.r.earlyRouteWhy(s, lair3); old != "" {
				t.Fatalf("%v: the old judge closes the road from inside: %q", class, old)
			}
			if got := g.r.admitEnter(s, lair3); got != "" {
				t.Fatalf("%v admission %d: admitEnter refuses a hero on the road: %q", class, admission, got)
			}
			if ex := g.r.admissionExplain(); ex.Differ != 0 || ex.NewOnly != 0 {
				t.Fatalf("%v admission %d: counters %+v, want differ 0", class, admission, ex)
			}
		}
	}
}

// Review 14 H3: both judges keep the potion half of the kit at the road's boss level; with a portal it is a trip.
func TestOnRoadKitSkipStopsAtTheBossLevelBothJudges(t *testing.T) {
	const lair2, lair3 = 63, 64
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, admission := range []int{0, 1} {
			g := admitRig(t, class, 30, lair2, 1400, 0, true, admission)
			g.w.set(func(w *world) { w.maxL = 1400 })
			g.r.poll()
			s := g.r.latest()
			if why, kit := g.r.earlyRouteCheck(s, lair3); why == "" || !kit {
				t.Fatalf("%v: the old judge lets a potionless hero into the boss level: %q kit=%v", class, why, kit)
			}
			if got := g.r.admitEnter(s, lair3); got == "" {
				t.Fatalf("%v admission %d: admitEnter lets a potionless hero into the boss level", class, admission)
			}
			if ex := g.r.admissionExplain(); ex.Differ != 0 || ex.NewOnly != 0 {
				t.Fatalf("%v admission %d: counters %+v, want differ 0", class, admission, ex)
			}
		}
	}
}

// Review 11 N8: the "both refuse with different words" cell has its own counter.
func TestAdmissionShadowCountsTheWordsCell(t *testing.T) {
	g := admitRig(t, d2.Paladin, 12, 2, 176, 4, true, 0)
	ref := admit.Verdict{Refused: true, Why: "new words", Row: "row", Need: "needs_potions"}
	g.r.admitNote(3, admit.Level, "old words", ref, false)
	g.r.admitNote(3, admit.Level, "", ref, false)
	if ex := g.r.admissionExplain(); ex.Words != 1 || ex.Differ != 2 || ex.NewOnly != 1 {
		t.Fatalf("counters %+v", ex)
	}
}

// Review 11 N1c (card sweep2): a controlled hook that declines is asked again after arbiter.DeclineMax at the latest.
// Before: every decline counted as a failure and the refill was silent for up to 30 minutes. Two classes.
func TestArbiterDeclinedHookIsAskedAgainAfterTheCap(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g, _ := farmRig(t, 6, class)
		g.w.set(func(w *world) { w.gold0 = 900 })
		g.w.belt()
		g.r.e.Tune.Bot.ArbiterShadow, g.r.e.Tune.Bot.ArbiterControl, g.r.e.Tune.Bot.ArbiterFamilies = 1, 1, 3|4|64
		for i := 0; i < 8; i++ {
			g.r.setLastTrip(g.clock.Now()) // the trip cooldown: the chores hook declines
			g.r.poll()
			g.clock.Sleep((arbiter.DeclineMax + 1) * time.Second)
			g.r.setLastTrip(g.clock.Now())
			g.r.runGoalShadowed(context.Background(), g.r.latest(), campControl())
			if d, _ := g.r.Arbiter(); d.Chosen == nil || arbiter.Family(d.Chosen.Source) != "chores" {
				t.Fatalf("%v: decline %d still silences the refill %d s later: chosen %+v rejected %+v", class, i, arbiter.DeclineMax+1, d.Chosen, d.Rejected)
			}
		}
	}
}
