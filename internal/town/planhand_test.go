package town

import (
	"regexp"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/town/belt"
)

// Card town7 on two builds: with 130 gold and an empty belt the plan buys potions first, names the dropped portal line, and
// the plan line carries the sell count and the hand-over (the fields the scenarios gold-short-priorities and
// short-gold-handover read). No game tables.
func TestPlanTailShortGoldTwoBuilds(t *testing.T) {
	lt, rs := env.DefaultLootTuning(), env.DefaultResourceTuning()
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		band := belt.PlanForClass(class, 20)
		c := supply.Count{Purse: 130, StashKnown: true}
		pl, err := solveShadow(class, "", band, belt.Columns, c, ChorePlan{Profile: "countess"}, &lt, &rs)
		if err != nil {
			t.Fatal(err)
		}
		if pl.Verdict != supply.ShortGold {
			t.Fatalf("%v: verdict %s: %s", class, pl.Verdict, pl.Trace())
		}
		line := "town plan: purse keep=0 spend=0 verdict=" + string(pl.Verdict) + planTail(pl, 6, false)
		if !regexp.MustCompile(`^town plan: .*verdict=short-gold.*portal`).MatchString(line) {
			t.Errorf("%v: %q misses the dropped portal line", class, line)
		}
		if !strings.Contains(line, " sell=6") || !strings.Contains(line, "sell, then recovery") {
			t.Errorf("%v: %q wants the sale and the hand-over", class, line)
		}
		if strings.Contains(planTail(pl, 0, false), "sell, then") {
			t.Errorf("%v: nothing to sell, yet the hand-over names sell", class)
		}
	}
}
