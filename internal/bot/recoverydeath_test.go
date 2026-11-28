package bot

import (
	"context"
	"errors"
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

// Card fix3 (review 3 finding 2): the same rule without a portal scroll (the incident hero could not own one). That
// path left through starveStep before the no-potion exclusion ran, and the own-level pick among the waypoint areas
// (homeBandArea) took an area of the hero's own level with an empty belt. The first field of the act is the floor.
func TestRecoveryFarmWithoutPotionsOrPortalTakesTheSafestArea(t *testing.T) {
	for _, tc := range []struct {
		class      d2.Class
		level, act int
	}{{d2.Sorceress, 7, 0}, {d2.Paladin, 7, 0}, {d2.Sorceress, 8, 0}, {d2.Paladin, 6, 0}, {d2.Sorceress, 16, 1}, {d2.Paladin, 16, 1}, {d2.Necromancer, 2, 0}} {
		g := newRig(t)
		g.r.e.Tune.Route.EarlyRoute = 0 // the lean rule (early route) is not what keeps the hero on the first field
		g.r.e.Tune.Resource.RecoveryKitGate = 0
		g.w.set(func(w *world) { w.class, w.level, w.gold0, w.act = tc.class, tc.level, 7, tc.act })
		g.w.belt()
		g.r.loop.broke.on = true
		s, _ := g.w.snapshot()
		if g.r.resInput(s).HasTP || !g.r.noPotions(s) {
			t.Fatal("the rig hero has a portal scroll or a potion")
		}
		g.r.activeWaypoints(s)
		first, areas := 0, 0
		g.r.loop.task.mu.Lock()
		for _, st := range g.r.d.Plan.Steps["normal"] {
			if g.r.d.Plan.IsTown(st.Area) || g.r.d.Plan.Act(st.Area) != tc.act+1 {
				continue
			}
			if first == 0 {
				first = st.Area
			}
			g.r.loop.task.wp[st.Area] = true // every area of the act has its waypoint
			areas++
		}
		g.r.loop.task.mu.Unlock()
		if areas < 2 {
			t.Fatalf("act %d: the rig plan has %d areas, the case needs a choice", tc.act+1, areas)
		}
		a, ok := g.r.recoveryArea(s)
		if !ok {
			t.Fatalf("%v clvl %d act %d: no recovery area", tc.class, tc.level, tc.act+1)
		}
		if ml := g.r.recoveryMLvl(a, d2.Normal); ml > tc.level-3 && a != first {
			t.Errorf("%v clvl %d act %d, no potions, no portal: picked area %d, monster level %d above clvl-3 and not the first field %d", tc.class, tc.level, tc.act+1, a, ml, first)
		}
	}
}

// Review 11, N4: the floor of the no-potion rule was the first field of the hero's CURRENT act. For a hero in act 2 with
// no potion and no scroll that field's monster level is above the limit it is the floor of (every act 2 area is). The
// floor is then act 1's first field (the town waypoint is free). Lean (no waypoint reach) and not lean; two classes.
// Act 1 heroes keep their floor: nothing lower exists.
func TestNoPotionFloorIsNeverAboveTheLimit(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, lean := range []int{1, 0} {
			const level = 15
			g := newRig(t)
			g.r.e.Tune.Route.EarlyRoute = lean
			g.r.e.Tune.Resource.RecoveryKitGate = 0
			g.r.mlvlFn = func(area int, _ d2.Difficulty) int {
				if g.r.d.Plan.Act(area) == 1 {
					return normalMLvl[area]
				}
				return 12 + g.r.d.Plan.Act(area) // above clvl-3 in every later act
			}
			town := 0
			for a, ar := range g.r.d.Plan.Areas {
				if ar.Town && ar.Act == 2 {
					town = a
				}
			}
			g.w.set(func(w *world) { w.class, w.level, w.gold0, w.act, w.area = class, level, 7, 1, town })
			g.w.belt()
			g.r.loop.broke.on = true
			s, _ := g.w.snapshot()
			if g.r.resInput(s).HasTP || !g.r.noPotions(s) || town == 0 {
				t.Fatal("the rig hero has a portal scroll or a potion, or the plan has no act 2 town")
			}
			g.r.activeWaypoints(s)
			g.r.loop.task.mu.Lock()
			for _, st := range g.r.d.Plan.Steps["normal"] {
				if !g.r.d.Plan.IsTown(st.Area) && g.r.d.Plan.Act(st.Area) == 2 {
					g.r.loop.task.wp[st.Area] = true
				}
			}
			g.r.loop.task.mu.Unlock()
			a, ok := g.r.recoveryArea(s)
			if !ok {
				continue // a refusal is allowed, farming above the limit is not
			}
			if ml := g.r.recoveryMLvl(a, d2.Normal); ml > level-3 {
				t.Errorf("%v lean %d: clvl %d in act 2, no potion, no scroll: picked area %d (act %d), monster level %d above clvl-3",
					class, lean, level, a, g.r.d.Plan.Act(a), ml)
			}
			// ...and the way there exists: no walk leads west, the town waypoint does
			g.mover.routeFn = func(from, to int) ([]int, error) {
				if g.r.d.Plan.Act(from) != g.r.d.Plan.Act(to) {
					return nil, errors.New("no exit to that level")
				}
				g.w.set(func(w *world) { w.act = g.r.d.Plan.Act(from) - 1 }) // the game: arrived in the other act
				g.r.poll()
				return []int{to}, nil
			}
			g.r.poll()
			if err := g.r.travelTo(context.Background(), a); err != nil || g.w.count("wp 1") != 1 {
				t.Errorf("%v lean %d: travel from the act 2 town to area %d: %v (calls %s)", class, lean, a, err, g.w.log())
			}
		}
	}
}

// With a scroll, an exclusion that empties the plan pick (the low areas are cleared) falls back the same way.
func TestRecoveryFarmFallbackKeepsTheNoPotionRule(t *testing.T) {
	for _, tc := range []struct {
		class d2.Class
		level int
	}{{d2.Sorceress, 12}, {d2.Paladin, 12}, {d2.Sorceress, 3}, {d2.Paladin, 2}} {
		class, level := tc.class, tc.level
		g := newRig(t)
		g.r.e.Tune.Route.EarlyRoute = 0
		g.r.e.Tune.Resource.RecoveryKitGate = 0
		g.w.set(func(w *world) {
			w.class, w.level, w.gold0 = class, level, 7
			w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
		})
		g.w.belt()
		g.r.loop.broke.on = true
		s, _ := g.w.snapshot()
		g.r.activeWaypoints(s)
		first := 0
		g.r.loop.task.mu.Lock()
		for _, st := range g.r.d.Plan.Steps["normal"] {
			if g.r.d.Plan.IsTown(st.Area) || g.r.d.Plan.Act(st.Area) != 1 {
				continue
			}
			if first == 0 {
				first = st.Area
				continue
			}
			g.r.loop.task.wp[st.Area] = true
			if g.r.recoveryMLvl(st.Area, d2.Normal) <= level-3 {
				g.r.loop.cleared[st.Area] = true // the allowed low areas are spent
			}
		}
		g.r.loop.task.mu.Unlock()
		a, ok := g.r.recoveryArea(s)
		if !ok {
			t.Fatalf("%v: no recovery area", class)
		}
		if ml := g.r.recoveryMLvl(a, d2.Normal); ml > level-3 && a != first {
			t.Errorf("%v clvl %d, low areas cleared: picked area %d, monster level %d, not the first field %d", class, level, a, ml, first)
		}
	}
}
