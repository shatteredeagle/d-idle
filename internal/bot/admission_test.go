package bot

import (
	"strings"
	"testing"

	"d2lod-bot/internal/admit"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/task"
)

// admitRig is a rig for one class with the given kit, life and area; EarlyRoute 1 so every row is on.
func admitRig(t *testing.T, class d2.Class, level, area, life, potions int, tp bool, admission int) *rig {
	t.Helper()
	g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) {
		tu.Route.EarlyRoute = 1
		tu.Bot.Admission = admission
	})
	g.w.set(func(w *world) {
		w.class, w.level, w.area, w.life, w.maxL = class, level, area, life, 176
		var keep []d2.Unit
		for _, u := range w.items {
			if u.Class != 1 && u.Class != 5 {
				keep = append(keep, u)
			}
		}
		w.items = keep
		if tp {
			w.items = append(w.items, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
		}
		for i := 0; i < potions; i++ {
			w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
		}
	})
	g.r.poll()
	return g
}

// The function beside the old judges (shadow) never disagrees on the four entry rows, and bot.admission 1 hands the same answers
// out: the entry words of areaGuardWhy are equal in both modes. Two classes, kit and life cases of the Questy incidents.
func TestAdmissionShadowEqualsOldJudges(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			level, area, life, potions int
			tp                         bool
		}{{12, 1, 176, 1, true}, {12, 2, 176, 0, true}, {12, 2, 90, 1, false}, {12, 2, 170, 1, false}, {12, 2, 90, 4, false}, {5, 1, 176, 0, true}, {8, 1, 176, 0, false}, {20, 2, 176, 6, true}} {
			var ans [2][]string
			for mode := 0; mode <= 1; mode++ {
				g := admitRig(t, class, tc.level, tc.area, tc.life, tc.potions, tc.tp, mode)
				s := g.r.latest()
				for _, a := range []int{2, 3, 4, 5, 10} {
					if a == tc.area {
						continue
					}
					ans[mode] = append(ans[mode], g.r.areaGuardWhy(s, a))
				}
				ex := g.r.admissionExplain()
				if ex.Differ != 0 {
					t.Errorf("%v %+v mode %d: the function differs from the old judges %d times (last %q)", class, tc, mode, ex.Differ, ex.Last)
				}
				if wantMode := []string{"shadow", "control"}[mode]; ex.Mode != wantMode {
					t.Errorf("mode %q, want %q", ex.Mode, wantMode)
				}
			}
			for i := range ans[0] {
				if ans[0][i] != ans[1][i] {
					t.Errorf("%v %+v: old %q, control %q", class, tc, ans[0][i], ans[1][i])
				}
			}
		}
	}
}

// A refusal reaches ops status --why with its row, need and wake; the town exit asks the quest-walk row.
func TestAdmissionExplainLine(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		g := admitRig(t, class, 12, 2, 176, 0, true, 0)
		s := g.r.latest()
		if why := g.r.areaGuardWhy(s, 4); why == "" {
			t.Fatalf("%v: no potions must close area 4", class)
		}
		ex := g.r.Explain().Admission
		if ex.Row == "" || ex.Need != "needs_potions" || ex.Wake == "" || ex.Last == "" || ex.Agree == 0 {
			t.Errorf("%v: admission line %+v", class, ex)
		}
	}
	g := admitRig(t, d2.Paladin, 8, 1, 176, 0, true, 1)
	why, kit := g.r.admitLeave(g.r.latest())
	oldWhy, oldKit := g.r.questWalkCheck(g.r.latest())
	if why != oldWhy || kit != oldKit || why == "" {
		t.Errorf("quest walk exit: control %q %v, old %q %v", why, kit, oldWhy, oldKit)
	}
}

// Fable review part 7, section 4, difference 2: Duriel's Lair (area 73) is on the tombs road and has its own row with the
// caster life floor of 220. The old judge took the first row only, so a caster with max life 176 walked in while the
// function refused: a new-only differ. Both now ask every row; the pair agrees and both refuse. The Paladin (melee floor
// 220 on both rows) was refused by both before and after: the second build.
func TestAdmissionDurielLairAsksEveryRoadRow(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		g := admitRig(t, class, 30, 46, 176, 6, true, 0)
		s := g.r.latest()
		old := g.r.earlyRouteWhy(s, 73)
		if !strings.Contains(old, "life 220 needed") {
			t.Fatalf("%v: the old judge lets max life 176 into Duriel's Lair: %q", class, old)
		}
		if got := g.r.admitEnter(s, 73); got != old {
			t.Fatalf("%v: admitEnter %q, old %q", class, got, old)
		}
		if ex := g.r.admissionExplain(); ex.Differ != 0 || ex.AgreeRefused != 1 {
			t.Fatalf("%v: counters %+v, want differ 0 and one refused/refused agreement", class, ex)
		}
		w := &task.World{}
		g.r.fillEarly(w)
		if _, blocked := w.EarlyBlocked[73]; !blocked {
			t.Fatalf("%v: the task layer says Duriel's Lair is open while the hop refuses", class)
		}
	}
}

// Difference 1: the lean entry row under the owner's try anyway. The old judge held (no exemption), the function let
// through: an old-only differ. Both exempt now. Without the try both still hold (the must-fail half). Two classes.
func TestAdmissionLeanRowIsExemptUnderTry(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, try := range []bool{false, true} {
			task.SetTry(try)
			g := admitRig(t, class, 12, 2, 90, 1, false, 0)
			got := g.r.admitEnter(g.r.latest(), 3)
			ex := g.r.admissionExplain()
			task.SetTry(false)
			if held := got != ""; held == try || ex.Differ != 0 {
				t.Fatalf("%v try %v: entry words %q, counters %+v", class, try, got, ex)
			}
		}
	}
}

// The shadow window can fail: each mixed cell is counted on its own and logged as a differ line, and an agreement on a
// refusal is told from the both-open case (a window with agreeRefused 0 exercised no row).
func TestAdmissionShadowCountsTheMixedCells(t *testing.T) {
	g := admitRig(t, d2.Paladin, 12, 2, 176, 4, true, 0)
	ref := admit.Verdict{Refused: true, Why: "new words", Row: "row", Need: "needs_potions"}
	g.r.admitNote(3, admit.Level, "", admit.Allow, false)          // both open
	g.r.admitNote(3, admit.Level, "new words", ref, false)         // both refuse, same words
	g.r.admitNote(3, admit.Level, "old words", admit.Allow, false) // old-only
	g.r.admitNote(3, admit.Level, "", ref, false)                  // new-only
	g.r.admitNote(3, admit.Level, "old words", ref, false)         // words
	ex := g.r.admissionExplain()
	if ex.Agree != 2 || ex.AgreeRefused != 1 || ex.Differ != 3 || ex.OldOnly != 1 || ex.NewOnly != 1 {
		t.Fatalf("counters %+v", ex)
	}
}
