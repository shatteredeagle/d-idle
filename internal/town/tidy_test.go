package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
	"d2lod-bot/internal/town"
)

func equippedCount(r *rig) int {
	n := 0
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContEquipment {
			n++
		}
	}
	return n
}

func countIn(r *rig, code string, c d2.Container) int {
	n := 0
	for _, u := range r.W.Items {
		if u.Item.Where == c && r.Env.Data.Items[u.Class].Code == code {
			n++
		}
	}
	return n
}

// The Tidy chore stashes a rune, leaves the worn gear and a keep-marked rune alone, and runs the same for a caster and
// a melee class (the policy table has no class literal).
func TestTidyStashesByPolicyAndNeverTouchesEquipmentOrMarks(t *testing.T) {
	geartest.Need(t)
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r := newRig(t)
		r.W.Player.Class = cls
		dressed(r.W)
		worn := equippedCount(r)
		r.W.Add("r07", d2.ContInventory, 0, 0)
		marked := r.W.Add("r07", d2.ContInventory, 1, 0)
		r.T.SetItemMarks(fakeMarks{marked.ID: "keep"})
		rep, err := r.T.Chores(context.Background(), town.ChorePlan{Stash: true})
		if err != nil {
			t.Fatal(err)
		}
		if got := countIn(r, "r07", d2.ContStash); got != 1 {
			t.Errorf("%v: %d runes in the stash, want 1 (steps %+v)", cls, got, rep.Steps)
		}
		if got := countIn(r, "r07", d2.ContInventory); got != 1 {
			t.Errorf("%v: %d runes left in the pack, want the marked one", cls, got)
		}
		if equippedCount(r) != worn {
			t.Errorf("%v: equipment changed", cls)
		}
	}
}

// With tidy off nothing runs.
func TestTidyOffDoesNothing(t *testing.T) {
	geartest.Need(t)
	r := newRig(t)
	r.Env.Tune.Town.TidyEnabled = 0
	dressed(r.W)
	r.W.Add("r07", d2.ContInventory, 0, 0)
	rep, _ := r.T.Chores(context.Background(), town.ChorePlan{Sell: true})
	if rep.Tidied != 0 {
		t.Errorf("tidied %d", rep.Tidied)
	}
}
