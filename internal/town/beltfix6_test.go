package town_test

import (
	"context"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/town/belt"
)

// beltCols returns the potion kinds by belt box.
func beltBoxes(r *rig) map[int]string {
	out := map[int]string{}
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContBelt {
			out[u.Item.GY*4+u.Item.GX] = r.W.DB.Items[u.Class].Code
		}
	}
	return out
}

// BELTFIX (docs/beltfix.md): the sandbox state, mana in three of four columns, healing in the pack. The town visit moves
// mana out of the surplus columns and fills healing, for a Paladin (three healing columns) and a Sorceress (two).
func TestTownMovesSurplusManaOutAndFillsHealing(t *testing.T) {
	geartest.Need(t)
	for _, tc := range []struct {
		name    string
		class   d2.Class
		minHeal int // healing boxes after the visit on a four-box belt
	}{
		{"paladin", d2.Paladin, 3},
		{"sorceress", d2.Sorceress, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			dressed(r.W)
			r.W.Player.Class = tc.class
			for c := 0; c < 3; c++ {
				r.W.Add("mp1", d2.ContBelt, c, 0)
			}
			addPack(r, "hp1", 4)
			rep, err := r.T.FillBelt(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			hp, mp := 0, 0
			for _, c := range beltBoxes(r) {
				if strings.HasPrefix(c, "hp") {
					hp++
				}
				if strings.HasPrefix(c, "mp") {
					mp++
				}
			}
			if hp < tc.minHeal || rep.Repaired == 0 {
				t.Fatalf("healing %d (want %d), mana %d, repaired %d, belt %v", hp, tc.minHeal, mp, rep.Repaired, beltBoxes(r))
			}
			if mp == 0 {
				t.Errorf("every mana potion left the belt: %v", beltBoxes(r))
			}
			if n := count(r.W, d2.ContBelt) + packPotions(r, "mp"); n < 3+0 {
				t.Errorf("a mana potion vanished: belt+pack %d", n)
			}
			// no oscillation: a second visit changes nothing
			n0 := len(r.W.Clicks)
			if rep2, err := r.T.FillBelt(context.Background()); err != nil || rep2.Moved != 0 || rep2.Repaired != 0 || len(r.W.Clicks) != n0 {
				t.Errorf("second visit moved things: %+v err %v clicks %d", rep2, err, len(r.W.Clicks)-n0)
			}
		})
	}
}

func packPotions(r *rig, prefix string) int {
	n := 0
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContInventory && strings.HasPrefix(r.W.DB.Items[u.Class].Code, prefix) {
			n++
		}
	}
	return n
}

// A belt the player set up on purpose (all mana) is not repaired, and a belt with healing already in every planned
// column is left alone.
func TestBeltRepairLeavesPlayerPlanAlone(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	dressed(r.W)
	r.Env.Tune.Resource.BeltPlan = 3 // all mana
	for c := 0; c < 4; c++ {
		r.W.Add("mp1", d2.ContBelt, c, 0)
	}
	addPack(r, "hp1", 3)
	n0 := len(r.W.Clicks)
	rep, err := r.T.FillBelt(context.Background())
	if err != nil || rep.Repaired != 0 || len(r.W.Clicks) != n0 {
		t.Fatalf("all-mana plan was touched: %+v err %v", rep, err)
	}
	_ = belt.HP
}
