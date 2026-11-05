package town_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town"
)

// restockVisit runs a potion visit for a class with one healing potion in the belt and no mana potion.
func restockVisit(t *testing.T, class d2.Class, gold, clvl int) (town.ChoreReport, string) {
	t.Helper()
	r := audit(t, gold, clvl, false)
	r.W.Player.Class = class
	r.Env.Tune.Resource.KitPotionGold, r.Env.Tune.Resource.KitTPGold = 30, 100
	r.W.Add("hp1", d2.ContBelt, 0, 0)
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	var buf bytes.Buffer
	r.Env.Log = slog.New(slog.NewTextHandler(&buf, nil))
	rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
	if err != nil {
		t.Fatal(err)
	}
	return rep, buf.String()
}

// TASK-A: with gold for the plan the caster ends with the first mana potions and the healing plan filled (the rig's
// 4-slot belt holds 2 healing + 2 mana; its vendor prices are higher than the game's), and logs the restock.
func TestCasterBuysFirstManaBeforeHealingFill(t *testing.T) {
	rep, log := restockVisit(t, d2.Sorceress, 500, 6)
	if mp, hp := boughtLike(rep, "Mana"), boughtLike(rep, "Healing"); mp < 2 || hp < 1 {
		t.Fatalf("500 gold bought %v, want 2 mana and the missing healing potion", rep.Bought)
	}
	if !strings.Contains(log, "town: restocked mana potions") {
		t.Errorf("no restock line in %q", log)
	}
}

// With 20 gold nothing is affordable: no mana bought, and the no-mana line comes once per gold value.
func TestCasterPoorBuysNoManaAndLogs(t *testing.T) {
	rep, log := restockVisit(t, d2.Sorceress, 20, 6)
	if boughtLike(rep, "Mana") != 0 {
		t.Fatalf("20 gold bought mana: %v", rep.Bought)
	}
	if !strings.Contains(log, "caster: no mana, gold 20") {
		t.Errorf("no caster line in %q", log)
	}
}

// A second build: the Barbarian has no mana column, so the visit is the old healing fill and logs nothing about mana.
func TestMeleeBuildUnchangedByManaRestock(t *testing.T) {
	rep, log := restockVisit(t, d2.Barbarian, 300, 6)
	if boughtLike(rep, "Mana") != 0 || boughtLike(rep, "Healing") < 3 {
		t.Fatalf("barbarian bought %v", rep.Bought)
	}
	if strings.Contains(log, "mana") && strings.Contains(log, "restocked") || strings.Contains(log, "caster:") {
		t.Errorf("melee build logged a mana line: %q", log)
	}
}

// The trip trigger: a caster at the kit gate level with one mana potion and gold for another goes to town; a
// caster without the gold for one, and a Barbarian (no mana column), do not.
func TestCasterTripAtFewerThanFirstMana(t *testing.T) {
	for _, tc := range []struct {
		name  string
		class d2.Class
		gold  int
		want  bool
	}{{"sorceress", d2.Sorceress, 300, true}, {"sorceress poor", d2.Sorceress, 20, false}, {"barbarian", d2.Barbarian, 300, false}} {
		r := audit(t, tc.gold, 12, false)
		r.W.Player.Class = tc.class
		r.W.Add("hp1", d2.ContBelt, 0, 0)
		r.W.Add("hp1", d2.ContBelt, 1, 0)
		r.W.Add("hp1", d2.ContBelt, 3, 0)
		r.W.Add("mp1", d2.ContBelt, 2, 0)
		r.W.Add("tsc", d2.ContInventory, 9, 3)
		s, _ := r.W.Snapshot()
		if got, why := r.T.NeedsTrip(s); got != tc.want {
			t.Errorf("%s: NeedsTrip = %v (%s), want %v", tc.name, got, why, tc.want)
		}
	}
}
