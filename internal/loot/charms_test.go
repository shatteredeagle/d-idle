package loot_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
	"d2lod-bot/internal/town/sim"
)

// eval judges a fresh item through EvaluateUnit, so the looter's needs (cube held, free cells) come from the pack.
func eval(l *loot.Looter, w *sim.World, code string, q int, flags uint32, st d2.Stats) loot.Verdict {
	u := w.NewItem(code, q)
	u.Item.Flags = flags
	u.Addr = 0xC0DE
	w.Stats[u.Addr] = st
	return l.EvaluateUnit(u)
}

// M1LOOT (decision D4, SME 2026-10-31): what a sell pass does with charms, runes and gems. The loot package decides per
// item (stash, keep and needId survive the visit; sell goes to the vendor), so the pass is the verdict of every item.
func TestCharmsRunesGemsOnSellVisit(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Level = 10
	type row struct {
		name   string
		code   string
		q      int
		flags  uint32
		stats  d2.Stats
		want   loot.Action
		needID bool
	}
	var id uint32 = d2.FlagIdentified
	rows := []row{
		{"rune", "r07", 2, id, nil, loot.Stash, false},
		{"small charm with life", "cm1", 4, id, d2.Stats{7: 20 << 8}, loot.Stash, false},
		{"small charm with FCR", "cm1", 4, id, d2.Stats{105: 3}, loot.Stash, false},
		{"small charm with MF", "cm1", 4, id, d2.Stats{80: 7}, loot.Stash, false},
		{"unidentified small charm", "cm1", 4, 0, nil, loot.Keep, true},
		{"plain identified small charm", "cm1", 4, id, d2.Stats{21: 3}, loot.Sell, false},
		{"chipped gem, no Cube", "gcv", 2, id, nil, loot.Sell, false},
		{"flawed gem, no Cube", "gfv", 2, id, nil, loot.Sell, false},
		{"standard gem", "gsv", 2, id, nil, loot.Stash, false},
		{"flawless gem", "gzv", 2, id, nil, loot.Stash, false},
	}
	for _, c := range rows {
		v := eval(l, w, c.code, c.q, c.flags, c.stats)
		if v.Action != c.want || v.NeedID != c.needID {
			t.Errorf("%s: %v (rule %q), want %v needID %v", c.name, v.Action, v.Rule, c.want, c.needID)
		}
	}
}

// Large and grand charms need 3 free pack cells; a full pack sells them, as the SME does.
func TestBigCharmsNeedThreeFreeCells(t *testing.T) {
	l, r := newLooter(t)
	w := r.W
	w.Player.Level = 10
	st := d2.Stats{7: 30 << 8}
	if v := eval(l, w, "cm2", 4, d2.FlagIdentified, st); v.Rule != "big charms" { // large charm
		t.Fatalf("room: %v", v)
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 10; x++ {
			w.Add("cm1", d2.ContInventory, x, y)
		}
	}
	if v := eval(l, w, "cm2", 4, d2.FlagIdentified, st); v.Action != loot.Sell {
		t.Fatalf("full pack: %v", v)
	}
}

// With town.sellLowGems 0 the old rule returns: low gems are kept for the cube even without one.
func TestSellLowGemsKnobOffKeepsThem(t *testing.T) {
	l, r := newLooter(t)
	r.Env.Tune.Town.SellLowGems = 0
	if v := l.EvaluateUnit(r.W.NewItem("gcv", 2)); v.Action != loot.Stash {
		t.Fatalf("%v", v)
	}
}
