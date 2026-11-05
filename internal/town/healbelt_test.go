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

// healBeltRig is a belt of three mana potions and one Rejuvenation (Gate 2 g2b seed 13) for class.
func healBeltRig(t *testing.T, class d2.Class, gold int) (*rig, *bytes.Buffer) {
	t.Helper()
	r := audit(t, gold, 4, false)
	r.W.Player.Class = class
	r.Env.Tune.Resource.KitPotionGold, r.Env.Tune.Resource.KitTPGold = 30, 100
	for i := 0; i < 3; i++ {
		r.W.Add("mp1", d2.ContBelt, i, 0)
	}
	r.W.Add("rvs", d2.ContBelt, 3, 0)
	r.W.Add("tsc", d2.ContInventory, 9, 3)
	var buf bytes.Buffer
	r.Env.Log = slog.New(slog.NewTextHandler(&buf, nil))
	return r, &buf
}

// Rejuvenation is no healing potion: the shop buys one even though no belt column is free.
func TestShopBuysHealingWhenOnlyRejuvenationAndMana(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		r, _ := healBeltRig(t, class, 88)
		rep, err := r.T.Chores(context.Background(), town.ChorePlan{Potions: true})
		if err != nil {
			t.Fatal(err)
		}
		if boughtLike(rep, "Healing") < 1 {
			t.Errorf("class %v: bought %v, want a healing potion", class, rep.Bought)
		}
	}
}

// An empty healing belt raises a trip when the gold buys a potion; without the gold there is none and one log line.
func TestEmptyHealingBeltTrip(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		r, buf := healBeltRig(t, class, 88)
		s, _ := r.W.Snapshot()
		if got, why := r.T.NeedsTrip(s); !got || !strings.Contains(why, "no healing potion") {
			t.Errorf("class %v rich: NeedsTrip = %v (%s)", class, got, why)
		}
		r, buf = healBeltRig(t, class, 0)
		s, _ = r.W.Snapshot()
		for i := 0; i < 3; i++ {
			if got, why := r.T.NeedsTrip(s); got && strings.Contains(why, "no healing potion") {
				t.Errorf("class %v poor: trip %s", class, why)
			}
		}
		if n := strings.Count(buf.String(), "gold buys none"); n != 1 {
			t.Errorf("class %v poor: %d log lines, want 1: %q", class, n, buf)
		}
	}
}
