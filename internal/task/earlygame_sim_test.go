package task_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/task"
	"d2lod-bot/internal/task/sim"
)

// EARLYGAME: a fresh character (clvl 1, no gold, the 4 starter potions, one town portal scroll, no mana potions) of
// every class runs the Normal Act 1 campaign in the fake world. The fake knows nothing about classes except the Caster
// flag the boss gate reads, so this checks the gates and the planner for clvl 1-8, not the combat. Rules inside the fake
// are U (docs/earlygame.md).
func freshChain(caster bool, gold int) *chain {
	c := newChain(true, "normal", 1)
	g := c.cur.(*sim.Game)
	g.W.Gold, g.W.Potions, g.W.ManaPotions, g.W.Caster, g.W.HasTP, g.W.Antidotes = gold, 4, 0, caster, true, 0
	return c
}

func act1Only() *task.Campaign {
	c := task.Template("normal_all", false)
	var e []task.Entry
	for _, x := range c.Entries {
		if x.Act == 1 {
			e = append(e, x)
		}
	}
	c.Entries = e
	return c
}

func TestFreshCharacterStartsWithTheDenAndNeverStallsInAct1(t *testing.T) {
	for _, caster := range []bool{false, true} {
		for _, gold := range []int{0, 300} {
			c := freshChain(caster, gold)
			ex := task.NewExecutor(c, c)
			var first string
			var tail []string
			done := false
			for i := 0; i < 3000 && !done; i++ {
				r, err := ex.Tick(context.Background(), act1Only())
				if err != nil {
					t.Fatalf("caster=%v gold=%d tick %d: %v", caster, gold, i, err)
				}
				if first == "" && r.Group != "" && r.Outcome == task.OutRan {
					first = r.Group
				}
				done = r.Outcome == task.OutDone
				tail = append(tail, r.StepID+"/"+string(rune('0'+int(r.Outcome))))
			}
			w, _ := c.World(context.Background())
			t.Logf("caster=%v gold=%d: first group %q, done=%v, clvl %d, act %d, log tail %v", caster, gold, first, done, w.Level, w.Act, c.Log[max(0, len(c.Log)-8):])
			if !done {
				t.Errorf("caster=%v gold=%d: Act 1 did not finish (stuck at clvl %d); tail %v", caster, gold, w.Level, c.Log[max(0, len(c.Log)-8):])
			}
			if first != task.QDen {
				t.Errorf("caster=%v gold=%d: first quest group %q, want the Den of Evil", caster, gold, first)
			}
			if v := c.violations(); len(v) > 0 {
				t.Errorf("violations: %v", v)
			}
		}
	}
}
