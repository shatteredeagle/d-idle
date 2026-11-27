package town_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/belt"
)

// pickupRig is a town rig whose Looter is shared with the town (as the bot wires it), a planned belt row of the class
// at clvl 20 and nothing in the pack.
func pickupRig(t *testing.T, class d2.Class) (*rig, *loot.Looter) {
	r := newRig(t)
	w := r.W
	w.Player.Class, w.Player.Level, w.Player.Difficulty = class, 20, d2.Normal
	band := belt.PlanForClass(class, 20)
	code := map[belt.Kind]string{belt.HP: "hp3", belt.MP: "mp3", belt.RV: "rvs"}
	for col, k := range band.Cols {
		if c, ok := code[k]; ok {
			w.Add(c, d2.ContBelt, col, 0)
		}
	}
	l := loot.New(r.Env, r.Mv)
	l.UseSource(w.Snapshot, w.UnitStats)
	r.T.SetJudge(l)
	return r, l
}

func potionWanted(l *loot.Looter, u d2.Unit) bool {
	return l.EvaluateUnit(u).Rule == "healing and mana"
}

// packKeep is the plan's pack keep of a line for the current snapshot (target or min, whichever is higher, minus the belt).
func packKeep(t *testing.T, r *rig, profile string, line supply.Line) (int, bool) {
	s, err := r.W.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	pl, err := r.T.SupplyPlan(s, profile, 0)
	if err != nil {
		t.Fatal(err)
	}
	var c supply.Count
	for _, u := range s.Inventory.Container(d2.ContBelt) {
		switch belt.PotionKind(r.Env.Data.Items[u.Class]) {
		case belt.HP:
			c.Belt[supply.HP]++
		case belt.MP:
			c.Belt[supply.MP]++
		case belt.RV:
			c.Belt[supply.RV]++
		}
	}
	for _, ln := range pl.Lines {
		if ln.Line != line {
			continue
		}
		want := max(ln.Target, ln.Min)
		if line == supply.Healing {
			return max(0, want-c.Belt[supply.HP]-c.Belt[supply.RV]), true
		}
		return max(0, want-c.Belt[supply.MP]), true
	}
	return 0, false
}

// Stage 4 pickup (town-planner 5.3) on two builds: a ground healing potion the belt cannot take is wanted exactly while
// the usable pack count is under the plan's keep, the number the sell chore protects; a potion under the tier floor does
// not count; below stage 4 the pack buffer decides. Needs the game tables (skips without them).
func TestStage4PickupReadsPlanLine(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, l := pickupRig(t, class)
		w := r.W
		hp := w.NewItem("hp3", 2)
		if !potionWanted(l, hp) {
			t.Fatalf("%v: stage 0, empty pack: healing not wanted", class)
		}
		r.Env.Tune.Town.Plan = 4
		keep, ok := packKeep(t, r, "", supply.Healing)
		if !ok {
			t.Fatalf("%v: field plan has no healing line", class)
		}
		w.Add("hp1", d2.ContInventory, 9, 3) // under the clvl 20 tier floor: never counts toward the target
		for i := 0; i <= keep+1; i++ {
			if got, want := potionWanted(l, hp), i < keep; got != want {
				t.Fatalf("%v: %d usable in the pack, keep %d: wanted %v, want %v", class, i, keep, got, want)
			}
			w.Add("hp3", d2.ContInventory, i, 0)
		}
		// The field profile has no mana line: a mana potion keeps today's buffer, the same verdict as stage 0.
		mp := w.NewItem("mp3", 2)
		at4 := potionWanted(l, mp)
		r.Env.Tune.Town.Plan = 0
		if at0 := potionWanted(l, mp); at0 != at4 {
			t.Errorf("%v: mana verdict %v at stage 4, %v at stage 0 (no mana line in the field plan)", class, at4, at0)
		}
	}
}

// The pickup reads the excursion the last chores visit supplied: after a visit for the andariel profile the ground
// healing potion is wanted exactly while the pack is under that profile's keep.
func TestStage4PickupFollowsVisitProfile(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, l := pickupRig(t, class)
		r.Env.Tune.Town.Plan = 4
		if _, err := r.T.Chores(context.Background(), town.ChorePlan{Profile: "andariel"}); err != nil {
			t.Fatalf("%v: %v", class, err)
		}
		if got := r.T.ExcursionForTest(); got != "andariel" {
			t.Fatalf("%v: excursion %q after an andariel visit", class, got)
		}
		keep, _ := packKeep(t, r, "andariel", supply.Healing)
		hp := r.W.NewItem("hp3", 2)
		for i := 0; i <= keep; i++ {
			if got, want := potionWanted(l, hp), i < keep; got != want {
				t.Fatalf("%v: %d in the pack, andariel keep %d: wanted %v, want %v", class, i, keep, got, want)
			}
			r.W.Add("hp3", d2.ContInventory, i, 1)
		}
	}
}

// Scroll hoard (scenario town.plan.scroll-hoard, ground half): at stage 4 a full tome and the loose cap refuse ground
// portal scrolls, on two builds.
func TestStage4ScrollHoardRefused(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, l := pickupRig(t, class)
		r.Env.Tune.Town.Plan = 4
		w := r.W
		tb := w.Add("tbk", d2.ContInventory, 0, 0)
		w.Stats[tb.Addr] = d2.Stats{70: loot.TomeCap}
		if l.EvaluateUnit(w.NewItem("tsc", 2)).Rule == "town scrolls" {
			t.Errorf("%v: scroll wanted beside a full tome", class)
		}
		w.Stats[tb.Addr] = d2.Stats{70: 5}
		for i := range r.Env.Tune.Town.LooseScrollKeep {
			w.Add("tsc", d2.ContInventory, 1+i, 0)
		}
		if l.EvaluateUnit(w.NewItem("tsc", 2)).Rule == "town scrolls" {
			t.Errorf("%v: scroll wanted beyond the loose cap", class)
		}
	}
}
