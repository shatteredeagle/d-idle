package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// Problem 8 of docs/probes/sandbox-log-analysis-2026-11-27.md: a Paladin at clvl 4 with 15 gold and an empty belt died in Cold
// Plains to a Fallen pack while "Recovery: farming gold" (V deaths.jsonl); Questy twice at clvl 12 in Stony Field. With no healing
// potion the recovery farm takes the safest area that still pays: the lowest monster level the plan allows, not the best XP.
// Must fail while the pick is by XP alone.
func TestRecoveryFarmWithoutPotionsTakesTheSafestArea(t *testing.T) {
	for _, tc := range []struct {
		class       d2.Class
		level, gold int
	}{{d2.Paladin, 4, 15}, {d2.Sorceress, 4, 15}, {d2.Paladin, 9, 7}, {d2.Necromancer, 12, 7}} {
		g := newRig(t)
		g.r.e.Tune.Route.EarlyRoute = 1
		g.w.set(func(w *world) { w.class, w.level, w.gold0 = tc.class, tc.level, tc.gold })
		g.w.belt()
		g.w.set(func(w *world) { // a portal scroll: the kit gate is not what limits the pick at clvl 12
			w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
		})
		g.r.loop.broke.on = true
		s, _ := g.w.snapshot()
		a, ok := g.r.recoveryArea(s)
		if !ok {
			t.Fatalf("%v clvl %d: no recovery area", tc.class, tc.level)
		}
		ml := g.r.recoveryMLvl(a, d2.Normal)
		if limit := tc.level - 3; ml > limit && ml > g.r.recoveryMLvl(2, d2.Normal) { // the first field is the floor
			t.Errorf("%v clvl %d, no potions: picked area %d, monster level %d above clvl-3 = %d", tc.class, tc.level, a, ml, limit)
		}
	}
}
