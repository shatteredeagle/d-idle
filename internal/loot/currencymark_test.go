package loot_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/loot"
)

// The Raven's Mark (overlay didle-v1, code mrk) is carried and never sold or dropped, for any build.
// The simulated item table gets the overlay row as the Go loader would produce it.
func TestCurrencyMarkIsKept(t *testing.T) {
	for _, build := range []string{"", "sorceress_blizzard_endgame90", "barbarian_warcry_endgame90"} {
		l, r := newLooter(t)
		r.W.DB.Items = append(r.W.DB.Items, &data.ItemBase{Kind: data.KindMisc, Name: "DI Mark", Code: "mrk", NameKey: "didle_mrk", Type: "key", Width: 1, Height: 1, Stackable: true, MinStack: 1, MaxStack: 12})
		l.SetBuild(build)
		u := r.W.Add("mrk", d2.ContInventory, 0, 0)
		for _, gold := range []int{0, 5000} {
			v := l.EvaluateWith(u, loot.Ctx{CLvl: 30, Gold: gold})
			if v.Action != loot.Keep || v.Rule != "currency marks" || v.Tier != "S" {
				t.Errorf("build %q gold %d: %v", build, gold, v)
			}
		}
	}
}
