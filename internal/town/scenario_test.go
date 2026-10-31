package town_test

// Scenario corpus, town half (docs/scenarios.md): the seed-8 poverty loop replayed through Town.Chores. The bot half
// (no one-item sell trip) is TestScenario_PovertyLoop in internal/bot.

import (
	"fmt"
	"strings"
	"testing"
)

// Situation: the character has 100 to 190 gold, an empty belt and no portal scroll, clvl 4, and visits town again and
// again (needs the extracted game tables like the other town audits: skipped without them, run it on the worker
// host). Evidence (docs/d2d-xp-gap.md, VERIFIED from the 24 h baseline: 64 portal scrolls were 62 % of 10 270 gold
// spent, 42 emergency and 34 poverty trips, seed 8 the known poverty loop at 8.5 trips/h; the d2-sme order of D3 is
// potions first). End state of every visit: healing potions bought (1 to 3), no portal scroll, no identify scroll,
// nothing banked (the kit line is the reserve), and gold only goes down by what was spent.
func TestScenario_PovertyLoop(t *testing.T) {
	visit := func(t *testing.T, gold int, potionsFirst int) (bought map[string]int, hp, after int) {
		r := audit(t, gold, 4, false)
		r.Env.Tune.Town.PotionsFirst = potionsFirst
		rep := run(t, r)
		for k, n := range rep.Bought {
			if strings.Contains(k, "Healing") {
				hp += n
			}
		}
		if got := rep.GoldBefore - rep.Spent + rep.Sold; got != rep.GoldAfter || r.W.Player.Gold != rep.GoldAfter {
			t.Fatalf("gold %d -> %d with spent %d sold %d: something was banked below the reserve", rep.GoldBefore, rep.GoldAfter, rep.Spent, rep.Sold)
		}
		return rep.Bought, hp, rep.GoldAfter
	}

	for _, gold := range []int{100, 130, 150, 190} {
		t.Run(fmt.Sprintf("fixed gold %d", gold), func(t *testing.T) {
			bought, hp, _ := visit(t, gold, 1)
			if hp == 0 || hp > 3 {
				t.Fatalf("gold %d: healing potions %d, bought %v", gold, hp, bought)
			}
			if bought["Scroll of Town Portal"] != 0 || bought["Scroll of Identify"] != 0 {
				t.Fatalf("gold %d: a scroll before the potions: %v", gold, bought)
			}
		})
	}

	// Knob off (town.potionsFirst 0, with the old tpScrollPotions): a poor character spends on the scroll before the
	// kit potions. The corpus must see it at every recorded gold value.
	t.Run("knob off: old scroll-first order reproduces", func(t *testing.T) {
		for _, gold := range []int{100, 130, 150, 190} {
			r := audit(t, gold, 4, false)
			r.Env.Tune.Town.PotionsFirst, r.Env.Tune.Town.TPScrollPotions, r.Env.Tune.Town.TPFirstLevel = 0, 0, 4
			rep := run(t, r)
			if rep.Bought["Scroll of Town Portal"] == 0 {
				t.Fatalf("gold %d: the old order did not buy the scroll, the scenario cannot detect it: %v", gold, rep.Bought)
			}
		}
	})
}
