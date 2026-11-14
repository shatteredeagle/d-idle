package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// sellskip (hero14 finding 4): with the player deciding gear nothing wears the "better" armor the looter keeps below
// clvl 15, so the sell chore skipped 157 of 158 visits while 9 items sat in the pack. A poor character with plain
// armor in the pack and a vendor visit must sell it.
func TestPoorPackOfPlainArmorIsSoldWhenThePlayerDecidesGear(t *testing.T) {
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		r := earlyRig(t, cls, 14)
		r.Env.Tune.Resource.GearDecisions = 0
		w := r.W
		w.Player.Gold = 40
		dressed(w) // worn: dagger and quilted armor; helm, belt and boots slots are empty
		for i, code := range []string{"cap", "lbt", "qui", "qui"} {
			w.Add(code, d2.ContInventory, i*2, 0)
		}
		rep, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
		if err != nil {
			t.Fatal(err)
		}
		if n := count(w, d2.ContInventory); n != 0 {
			t.Errorf("%v: %d items left in the pack, steps %+v", cls, n, rep.Steps)
		}
		if w.Player.Gold <= 40 {
			t.Errorf("%v: gold %d, nothing sold", cls, w.Player.Gold)
		}
	}
}

// sellskip (hero14 finding 5): 950 gold went to a tome and portal scrolls, then the 600-gold merc revive had no money.
// With ReserveGold set the optional buys (tome, spare scrolls) stay above it; without it they buy as before.
func TestShopKeepsTheMercReviveGold(t *testing.T) {
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, reserve := range []int{600, 0} {
			r := earlyRig(t, cls, 14)
			r.W.Player.Gold = 1050
			dressed(r.W)
			rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true, Scrolls: true, ReserveGold: reserve})
			if err != nil {
				t.Fatal(err)
			}
			bought := rep.Bought["Tome of Town Portal"]
			if reserve > 0 && (bought != 0 || r.W.Player.Gold < reserve) {
				t.Errorf("%v: reserve %d, tome %d, gold left %d", cls, reserve, bought, r.W.Player.Gold)
			}
			if reserve == 0 && bought == 0 {
				t.Errorf("%v: no reserve and no tome bought: %+v", cls, rep.Bought)
			}
		}
	}
}
