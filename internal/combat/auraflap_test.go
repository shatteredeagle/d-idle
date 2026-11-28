package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// auraFlapRun plays ticks 1.5 s apart: fights of fightLen ticks whose crowd count follows crowds, then gapLen ticks
// of upkeep calls with no target (Buff, rest). It returns the aura changes seen per fight (a fight with its gap).
func auraFlapRun(t *testing.T, h *harness, g *handGame, crowds []int, fightLen, gapLen, ticks int) (changes []int, total int) {
	t.Helper()
	ctx := context.Background()
	last, cur, n := g.aura, 0, 0
	for tick := 0; tick < ticks; tick++ {
		h.s.Sleep(1500 * time.Millisecond)
		pos := tick % (fightLen + gapLen)
		if pos == 0 && tick > 0 {
			changes = append(changes, cur)
			cur = 0
		}
		s, _ := g.snap()
		if pos < fightLen {
			h.f.auraCrowd = crowds[n%len(crowds)]
			n++
			if _, err := h.f.KeepAura(ctx, s, &Target{}, false); err != nil {
				t.Fatal(err)
			}
		} else if _, err := h.f.KeepAura(ctx, s, nil, true); err != nil {
			t.Fatal(err)
		}
		if g.aura != last {
			cur++
			total++
			last = g.aura
		}
	}
	return append(changes, cur), total
}

// One decider owns the standing aura: the profile's ladder, and its Holy Fire (crowd) entry does not flip with each
// monster that dies or walks in (class A on Questy: 95 Holy Fire and 97 Might presses in 70 minutes).
func TestAuraDoesNotFlapWithCrowd(t *testing.T) {
	h, g := newHandGame(t)
	g.aura, g.p.RightSkill = 0, 0
	changes, total := auraFlapRun(t, h, g, []int{4, 2, 3, 1, 3, 0, 4, 2}, 10, 2, 200)
	for i, c := range changes {
		if c > 2 {
			t.Fatalf("fight %d: %d aura changes, at most one at its start and one at its end", i, c)
		}
	}
	if max := 2 * len(changes); total > max {
		t.Fatalf("%d aura changes in 200 ticks, want at most %d", total, max)
	}
}

// A second build: Hammerdin (Concentration ahead of Holy Fire and Might) never changes its aura at all after the
// first apply, with the same crowd pattern.
func TestAuraStableHammerdin(t *testing.T) {
	h, g := newHandGame(t)
	cat := palCatalog()
	for id, name := range map[int]string{120: "Concentration", 121: "Blessed Hammer"} {
		sp := *cat.Skills[102]
		sp.ID, sp.Key, sp.Name = id, name, name
		if name == "Blessed Hammer" {
			sp.Aura, sp.LeftSkill, sp.Range = false, true, "h2h"
		}
		cat.Skills[id] = &sp
	}
	h.f.SetCatalog(cat)
	g.p.Level = 18
	g.p.Skills = append(g.p.Skills, d2.Skill{ID: 120, Level: 3}, d2.Skill{ID: 121, Level: 3})
	h.f.SetHotkeys(map[int]string{102: "f2", 98: "f1", 120: "f3"})
	g.keys["f3"] = 120
	g.aura, g.p.RightSkill = 0, 0
	_, total := auraFlapRun(t, h, g, []int{4, 2, 3, 1, 3, 0, 4, 2}, 10, 2, 200)
	if total > 1 {
		t.Fatalf("%d aura changes, want the first apply only", total)
	}
}
