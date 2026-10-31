package town_test

import (
	"context"
	"d2lod-bot/internal/gear/geartest"
	"math/rand"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town"
)

// dropKind is one entry of the simulated loot table.
type dropKind struct {
	code    string
	quality int
	unid    bool
	weight  int
}

// lootTable is a mixed bag of what the field drops: the junk the bot sells, the stash-worthy finds, supplies and
// the things the bot never sells (runes, quest items). Codes are items.txt codes the rest of the town tests use.
var lootTable = []dropKind{
	{"hax", 2, false, 12}, {"ssd", 2, false, 12}, {"qui", 2, false, 8}, {"cap", 3, false, 8}, {"buc", 2, false, 6},
	{"lbt", 2, false, 6}, {"ssd", 4, false, 10}, {"hax", 6, false, 4}, {"cap", 4, true, 6}, {"jew", 4, false, 4},
	{"r07", 2, false, 3}, {"r08", 2, false, 2}, {"vps", 2, false, 8}, {"isc", 2, false, 6}, {"key", 2, false, 3},
	{"hp1", 2, false, 4}, {"tsc", 2, false, 4},
}

// simPickup mimics the looter's room rule (loot/pickup.go candidates): valuables (prio 70+) may take the last cells,
// the rest keeps FreeCellsKeep free, a Sell verdict sellKeepExtra more. It returns false when the item stays.
func simPickup(r *rig, j town.Judge, u d2.Unit) (picked bool, valuable bool) {
	w := r.W
	v := j.EvaluateUnit(u)
	if v.Action == loot.Drop {
		return false, false
	}
	b := w.DB.Items[u.Class]
	if tome := map[string]string{"tsc": "tbk", "isc": "ibk"}[b.Code]; tome != "" && v.Action == loot.Keep {
		for _, it := range w.Items { // a matching tome absorbs the scroll on pickup (Kolbot comment, S)
			if it.Item.Where == d2.ContInventory && w.DB.Items[it.Class].Code == tome {
				return true, false
			}
		}
	}
	iw, ih := max(1, b.Width), max(1, b.Height)
	keep := r.Env.Tune.Loot.FreeCellsKeep
	if v.Prio >= 70 {
		keep = 0
	} else if v.Action == loot.Sell {
		keep += 4
	}
	g := w.Lay.Inventory
	occ := make([]bool, g.Cols*g.Rows)
	free := g.Cols * g.Rows
	for _, it := range w.Items {
		if it.Item.Where != d2.ContInventory {
			continue
		}
		ib := w.DB.Items[it.Class]
		for y := it.Y; y < it.Y+max(1, ib.Height); y++ {
			for x := it.X; x < it.X+max(1, ib.Width); x++ {
				occ[y*g.Cols+x] = true
				free--
			}
		}
	}
	val := v.Prio >= loot.LeftBehindMinPrio
	if free-iw*ih < keep {
		return false, val
	}
	for y := 0; y+ih <= g.Rows; y++ {
		for x := 0; x+iw <= g.Cols; x++ {
			ok := true
			for yy := y; yy < y+ih && ok; yy++ {
				for xx := x; xx < x+iw; xx++ {
					if occ[yy*g.Cols+xx] {
						ok = false
						break
					}
				}
			}
			if ok {
				w.Put(u, d2.ContInventory, x, y)
				return true, val
			}
		}
	}
	return false, val
}

func packPct(r *rig) int {
	used := 0
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContInventory {
			b := r.W.DB.Items[u.Class]
			used += max(1, b.Width) * max(1, b.Height)
		}
	}
	return used * 100 / (r.W.Lay.Inventory.Cols * r.W.Lay.Inventory.Rows)
}

// TestInventoryCycles runs 200 field drops through the looter's room rule and a town trip whenever the pack is 80
// percent full or a valuable was left behind. It asserts the audit's three rules: a valuable is never left on the
// ground twice in a row (the trip makes room), the pack is below 70 percent after every trip, and the Cube and
// quest items are never sold. Runes and gems may pile up in the stash but never vanish.
func TestInventoryCycles(t *testing.T) { runCycles(t, lootTable, 200) }

// TestInventoryCyclesStashOverflow drives 600 stash-worthy drops (jewels, magic ring-class junk, a few runes) so the
// 48-cell stash fills up and relieveStash has to sell the lowest-priority stash items; nothing may stall.
func TestInventoryCyclesStashOverflow(t *testing.T) {
	geartest.Need(t)
	runCycles(t, []dropKind{{"jew", 4, false, 20}, {"r07", 2, false, 1}, {"ssd", 4, false, 4}, {"hax", 2, false, 4}}, 600)
}

func runCycles(t *testing.T, lootTable []dropKind, drops int) {
	geartest.Need(t)
	total := 0
	for _, e := range lootTable {
		total += e.weight
	}
	for seed := int64(1); seed <= 3; seed++ {
		rng := rand.New(rand.NewSource(seed))
		r := newRig(t)
		w := r.W
		dressed(w)
		w.Player.Gold = 5000
		w.Add("box", d2.ContInventory, 0, 0) // the Cube (2x2)
		w.Add("msf", d2.ContInventory, 2, 0) // a quest item
		j := loot.New(r.Env, r.Mv)
		j.UseSource(w.Snapshot, w.UnitStats)
		runes := 0
		left := 0 // consecutive trips after which a valuable was still left
		trips := 0
		for n := 0; n < drops; {
			var leftBehind bool
			for k := 0; k < 4 && n < drops; k, n = k+1, n+1 {
				pick := rng.Intn(total)
				var d dropKind
				for _, e := range lootTable {
					if pick < e.weight {
						d = e
						break
					}
					pick -= e.weight
				}
				u := w.NewItem(d.code, d.quality)
				if d.unid {
					u.Item.Flags &^= d2.FlagIdentified
				}
				ok, val := simPickup(r, j, u)
				if ok && (d.code == "r07" || d.code == "r08") {
					runes++
				}
				if !ok && val {
					leftBehind = true
				}
			}
			if leftBehind && packPct(r) >= 80 || packPct(r) >= 80 {
				trips++
				rep, err := r.T.Chores(context.Background(), town.ChorePlan{Sell: true, Stash: true, Identify: true, Potions: true, Scrolls: true})
				if err != nil || rep.Errors != 0 {
					t.Fatalf("seed %d trip %d: %v %+v", seed, trips, err, rep.Steps)
				}
				if p := packPct(r); p >= 70 {
					t.Errorf("seed %d trip %d: pack still %d%% full after the chores: %v", seed, trips, p, packDump(r, j))
				}
				if leftBehind {
					left++
				} else {
					left = 0
				}
				if left > 1 {
					t.Fatalf("seed %d: valuables left on the ground after %d trips in a row", seed, left)
				}
			}
			if m := cubeCodes(r, d2.ContInventory); m["box"] != 1 || m["msf"] != 1 {
				t.Fatalf("seed %d: Cube or quest item gone from the pack: %v", seed, m)
			}
		}
		got := 0
		for _, u := range w.Items {
			if c := w.DB.Items[u.Class].Code; (c == "r07" || c == "r08") && u.Item.Where != d2.ContEquipment {
				got++
			}
		}
		if got != runes {
			t.Errorf("seed %d: picked %d runes, %d left in pack and stash", seed, runes, got)
		}
		t.Logf("seed %d: %d trips, pack %d%%, stash %d items", seed, trips, packPct(r), count(w, d2.ContStash))
	}
}

// packDump lists the pack as code/quality/verdict counts for failure messages.
func packDump(r *rig, j town.Judge) map[string]int {
	m := map[string]int{}
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContInventory {
			m[r.W.DB.Items[u.Class].Code+"/q"+string(rune('0'+u.Item.Quality))+"/"+j.EvaluateUnit(u).String()]++
		}
	}
	return m
}
