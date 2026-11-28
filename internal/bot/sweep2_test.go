package bot

import (
	"testing"

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
