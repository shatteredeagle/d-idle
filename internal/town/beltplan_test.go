package town_test

import (
	"context"
	"log/slog"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// planVisit runs a potion visit with the belt plan knob set; the belt starts empty.
func planVisit(t *testing.T, class d2.Class, tpl, gold int) (town.ChoreReport, *rig) {
	t.Helper()
	r := audit(t, gold, 12, false)
	r.W.Player.Class = class
	r.Env.Tune.Resource.KitPotionGold, r.Env.Tune.Resource.KitTPGold = 30, 100
	r.Env.Tune.Resource.BeltPlan = tpl
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	return rep, r
}

// TASK-164: the shopping list follows the plan template for two classes. The rig belt has four slots, so a column holds one.
func TestBeltPlanShoppingList(t *testing.T) {
	cases := []struct {
		name         string
		class        d2.Class
		tpl          int
		hp, mp, none bool // kinds that must / must not appear: hp, mp must be bought; none = nothing else
	}{
		{"sorceress all mana", d2.Sorceress, 3, false, true, true},
		{"barbarian all mana", d2.Barbarian, 3, false, true, true},
		{"sorceress all health", d2.Sorceress, 2, true, false, true},
		{"paladin 1:1", d2.Paladin, 5, true, true, true},
		{"barbarian default", d2.Barbarian, 0, true, false, true},
	}
	for _, c := range cases {
		rep, _ := planVisit(t, c.class, c.tpl, 1500)
		hp, mp := boughtLike(rep, "Healing"), boughtLike(rep, "Mana")
		if (hp > 0) != c.hp || (mp > 0) != c.mp {
			t.Errorf("%s: bought %v, want healing %v mana %v", c.name, rep.Bought, c.hp, c.mp)
		}
		if c.tpl == 3 && boughtLike(rep, "Rejuvenation") != 0 {
			t.Errorf("%s: bought rejuvenation %v", c.name, rep.Bought)
		}
	}
}

// The belt fill puts the mana potions in the belt of an all-mana plan and an unplanned healing potion does not take a column.
func TestBeltPlanFillResult(t *testing.T) {
	_, r := planVisit(t, d2.Sorceress, 3, 1500)
	if _, err := r.T.FillBelt(context.Background()); err != nil {
		t.Fatal(err)
	}
	s, _ := r.W.Snapshot()
	mp, hp := 0, 0
	for _, u := range s.Inventory.Container(d2.ContBelt) {
		switch r.Env.Data.Items[u.Class].Code[:2] {
		case "mp":
			mp++
		case "hp":
			hp++
		}
	}
	if mp == 0 || hp != 0 {
		t.Fatalf("belt holds %d mana, %d healing; want mana only", mp, hp)
	}
}

// An unknown template reaches the log (once) instead of a silent default.
func TestBeltPlanBadTemplateIsLogged(t *testing.T) {
	r := audit(t, 500, 12, false)
	var buf strings.Builder
	r.Env.Log = newBufLogger(&buf)
	r.Env.Tune.Resource.BeltPlan = 42
	s, _ := r.W.Snapshot()
	r.T.NeedsTrip(s)
	if !strings.Contains(buf.String(), "unknown belt plan template id 42") {
		t.Fatalf("log: %q", buf.String())
	}
}

func newBufLogger(w *strings.Builder) *slog.Logger { return slog.New(slog.NewTextHandler(w, nil)) }
