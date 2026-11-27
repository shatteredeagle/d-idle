package bot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/supply"
	"d2lod-bot/internal/task"
)

type manaCatalog struct{ fakeCatalog }

func (c manaCatalog) Item(class int) (ItemMeta, bool) {
	if class == 6 {
		return ItemMeta{Name: "Mana Potion", W: 1, H: 1, Potion: PotionMana}, true
	}
	return c.fakeCatalog.Item(class)
}

// fixedPlanner answers every guard question with one plan, to see what the bot hands the guards.
type fixedPlanner struct {
	Townsman
	pl supply.Plan
}

func (f fixedPlanner) SupplyPlan(*d2.Snapshot, string, int) (supply.Plan, error) { return f.pl, nil }

// Mana counts belt plus pack at every stage (the quest gate used to see the belt only: a Necromancer with its potions in the pack
// was refused "0 mana potions" for 29 minutes while the shop saw nothing to buy); stage 2 also hands the guards the plan.
func TestFillKitStages(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin, d2.Necromancer} {
		for _, stage := range []int{0, 1, 2, 10} { // 10: stage 0 with quests.boardSM (the guard reads the plan then)
			g := newRig(t)
			g.r.d.Catalog = manaCatalog{}
			g.r.e.Tune.Town.Plan = stage
			if stage == 10 {
				g.r.e.Tune.Town.Plan, g.r.e.Tune.Quests.BoardSM = 0, 1
			}
			g.w.set(func(w *world) {
				w.class = class
				w.items = []d2.Unit{
					{ID: 10, Type: d2.TypeItem, Class: 6, Item: &d2.Item{Where: d2.ContBelt}},
					{ID: 11, Type: d2.TypeItem, Class: 6, Item: &d2.Item{Where: d2.ContInventory}},
					{ID: 12, Type: d2.TypeItem, Class: 6, Item: &d2.Item{Where: d2.ContInventory}},
				}
			})
			pl := supply.Plan{Lines: []supply.PlanLine{{Line: supply.Healing, Min: 3}, {Line: supply.Mana, Min: 2}, {Line: supply.Portal, Min: 2}}, Verdict: supply.ShortGold, Open: "healing", Short: 60}
			g.r.d.Town = fixedPlanner{Townsman: g.r.d.Town, pl: pl}
			g.r.poll()
			w := g.r.guardWorld(g.r.latest())
			wantMana, wantPlan := 3, false
			if stage >= 2 { // stage 10 attaches the plan like stage 2
				wantPlan = true
			}
			if w.ManaPotions != wantMana || (w.KitPlan != nil) != wantPlan {
				t.Fatalf("%v stage %d: mana %d plan %v, want %d and %v", class, stage, w.ManaPotions, w.KitPlan, wantMana, wantPlan)
			}
			if wantPlan && w.KitPlan.Hand != "needs 60 more gold for healing: recovery" {
				t.Fatalf("%v: hand %q", class, w.KitPlan.Hand)
			}
			if wantPlan && (w.KitPlan.Healing != 3 || w.KitPlan.Short != "short of gold: needs 60 more for healing") {
				t.Fatalf("%v: plan %+v", class, *w.KitPlan)
			}
		}
	}
}

// The failing case of the Necromancer drill: no mana potion in the belt, all of them in the pack, the quest board's world
// (taskWorld) at the default stage 0.
func TestTaskWorldCountsPackMana(t *testing.T) {
	for _, class := range []d2.Class{d2.Necromancer, d2.Sorceress, d2.Paladin} {
		g := newRig(t)
		g.r.d.Catalog = manaCatalog{}
		g.w.set(func(w *world) {
			w.class = class
			w.items = []d2.Unit{
				{ID: 11, Type: d2.TypeItem, Class: 6, Item: &d2.Item{Where: d2.ContInventory}},
				{ID: 12, Type: d2.TypeItem, Class: 6, Item: &d2.Item{Where: d2.ContInventory}},
				{ID: 13, Type: d2.TypeItem, Class: 6, Item: &d2.Item{Where: d2.ContInventory}},
			}
		})
		g.r.poll()
		w, err := taskEnv{g.r}.World(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if w.ManaPotions != 3 {
			t.Errorf("%v: quest gate world counts %d mana potions, want 3 (pack only)", class, w.ManaPotions)
		}
		if gw := g.r.guardWorld(g.r.latest()); gw.ManaPotions != 3 {
			t.Errorf("%v: route guard world counts %d, want 3", class, gw.ManaPotions)
		}
	}
}

type profilePlanner struct {
	Townsman
	got *string
}

func (f profilePlanner) SupplyPlan(_ *d2.Snapshot, profile string, _ int) (supply.Plan, error) {
	*f.got = profile
	return supply.Plan{Lines: []supply.PlanLine{{Line: supply.Healing, Min: 8}, {Line: supply.GoldCarry, Min: 3000}}}, nil
}

// V live 2026-11-24: the boss chunk kept the road's kit (4 healing) and the purse fell to 188. Within two levels of Andariel's floor
// in Normal, before she is killed, the guards and the town read the "andariel" profile (8 healing, the boss purse); an earlier
// hero keeps the road's. Two classes.
func TestKitPlanUsesTheBossProfileNearAndariel(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Necromancer} {
		for _, tc := range []struct {
			level int
			want  string
		}{{20, "andariel"}, {5, kitProfile}} {
			g := newRig(t)
			g.r.e.Tune.Quests.BoardSM = 1
			g.r.e.Tune.Boss.AndarielHold = 0
			var got string
			g.r.d.Town = profilePlanner{Townsman: g.r.d.Town, got: &got}
			g.w.set(func(w *world) { w.class = class; w.level = tc.level })
			g.r.poll()
			k := g.r.kitPlanOf(g.r.latest())
			if got != tc.want {
				t.Errorf("%v level %d: profile %q, want %q", class, tc.level, got, tc.want)
			}
			if tc.want == "andariel" && (k == nil || k.GoldCarry != 3000 || g.r.bossPurse() != 3000) {
				t.Errorf("%v: boss purse not handed on: %+v", class, k)
			}
			// the boss floor is what belt and pack carry: no buffer above it, the shop could never close it (V live 2026-11-24)
			if h, _ := g.r.routeKit(); tc.want == "andariel" && h != 8 {
				t.Errorf("%v: the trip buys up to %d healing, want the plan's 8 without a buffer", class, h)
			}
		}
	}
}

// While boss.andarielHold holds Andariel back (and no owner try is on), the boss kit floor is not asked of a hero that only levels
// (V live 2026-11-25: 815 refusals "route guard (4 healing potions < 5)"). Two classes.
func TestBossKitIsNotAskedWhileAndarielIsHeld(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Necromancer} {
		g := newRig(t)
		g.r.e.Tune.Quests.BoardSM = 1
		g.r.e.Tune.Boss.AndarielHold = 1
		var got string
		g.r.d.Town = profilePlanner{Townsman: g.r.d.Town, got: &got}
		g.w.set(func(w *world) { w.class = class; w.level = 20 })
		g.r.poll()
		g.r.kitPlanOf(g.r.latest())
		if got != kitProfile {
			t.Errorf("%v: profile %q while Andariel is held, want %q", class, got, kitProfile)
		}
		if g.r.bossPurse() != 0 {
			t.Errorf("%v: boss purse while held", class)
		}
	}
}

// Questy bed 2026-11-25: a clvl 4 Paladin with no potions walked Stony Field, the Underground Passage and toward the Dark Wood and
// died. The early road (earlyroutes.json) is closed below the class family's level and short of 2 healing potions and a portal
// scroll; the owner's try anyway opens it. Two classes (melee 7, caster 8).
func TestEarlyRouteClosesBelowTheLevelAndTheKit(t *testing.T) {
	for _, tc := range []struct {
		class   d2.Class
		level   int
		potions int
		maxLife int
		want    string // "" open, else a part of the reason
	}{
		{d2.Paladin, 4, 3, 200, "level 7 needed"},
		{d2.Paladin, 7, 0, 200, "healing potions"},
		{d2.Paladin, 7, 2, 200, ""},
		{d2.Paladin, 8, 3, 69, "life 180 needed"}, // the Questy hero: clvl 8, nothing in vitality
		{d2.Sorceress, 7, 3, 120, "level 8 needed"},
		{d2.Sorceress, 8, 2, 120, ""},
		{d2.Sorceress, 8, 3, 60, "life 100 needed"},
	} {
		g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
		g.w.set(func(w *world) {
			w.class, w.level, w.area, w.maxL, w.life = tc.class, tc.level, 3, tc.maxLife, tc.maxLife
			var keep []d2.Unit
			for _, u := range w.items {
				if u.Class != 1 && u.Class != 5 { // the rig's own potions and scrolls go: the case sets them
					keep = append(keep, u)
				}
			}
			w.items = keep
		})
		for i := 0; i < tc.potions; i++ {
			g.w.set(func(w *world) {
				w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
			})
		}
		g.w.set(func(w *world) {
			w.items = append(w.items, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}}) // a portal scroll
		})
		g.r.poll()
		got := g.r.earlyRouteWhy(g.r.latest(), 10)
		if tc.want == "" && got != "" || tc.want != "" && !strings.Contains(got, tc.want) {
			t.Errorf("%v clvl %d potions %d: %q, want %q", tc.class, tc.level, tc.potions, got, tc.want)
		}
	}
	if (&Runner{}).earlyRouteWhy(nil, 10) != "" {
		t.Error("no snapshot must not close the road")
	}
}

// The walk itself is refused: a quest travel from Stony Field through the Underground Passage to the Dark Wood returns the gate
// error for a clvl 4 hero (the board shows the reason and the hero levels), and a walk to town is still fine. Two classes.
func TestEarlyRoadRefusesTheQuestWalk(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
		g.w.set(func(w *world) { w.class, w.level, w.area = class, 4, 4 })
		g.r.poll()
		s := g.r.latest()
		if err := g.r.routeAllowed(s, 4, 5, []int{10, 5}); !errors.Is(err, errAreaGated) {
			t.Errorf("%v: walk to the Dark Wood: %v, want the gate error", class, err)
		}
		if err := g.r.routeAllowed(s, 4, 1, []int{3, 2, 1}); err != nil {
			t.Errorf("%v: the walk home was refused: %v", class, err)
		}
	}
}

// fillEarly hands the closed early road to the task layer: the Dark Wood is blocked for a clvl 4 hero (level) and for a clvl 8 hero
// without potions (kit only); an open hero blocks nothing. Two classes.
func TestFillEarlyHandsTheRoadToTheTaskLayer(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			level, life int
			potions     bool
			want        bool
			kit         bool
		}{{4, 200, true, true, false}, {8, 200, false, true, true}, {9, 200, true, false, false}} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			g.w.set(func(w *world) {
				w.class, w.level, w.area, w.maxL, w.life = class, tc.level, 3, tc.life, tc.life
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 && u.Class != 5 {
						keep = append(keep, u)
					}
				}
				w.items = append(keep, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
				if tc.potions {
					for i := 0; i < 3; i++ {
						w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
					}
				}
			})
			g.r.poll()
			var w task.World
			g.r.fillEarly(&w)
			b, got := w.EarlyBlocked[5]
			if got != tc.want || got && b.KitOnly != tc.kit {
				t.Errorf("%v clvl %d potions %v: blocked %v %+v, want %v kitOnly %v", class, tc.level, tc.potions, got, b, tc.want, tc.kit)
			}
		}
	}
}

// Questy bed 2026-11-26: clvl 12, 0 potions, gold 7: the poverty farm took it to Stony Field (Dark Rangers) and it died. A hero short of
// the kit farms the best measured gold per minute of povertyfarm.json among the areas the kit floor leaves open: with two potions
// (tier: monsters far below the hero) Cave level 1 first, then Stony Field, Den of Evil; with no potion the first field only (the kit
// floor); with four potions the rule does not apply. Two classes.
func TestPovertyFarmPicksTheBestMeasuredArea(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			potions int
			cleared []int
			want    int
		}{{2, nil, 9}, {2, []int{9}, 4}, {2, []int{9, 4}, 8}, {0, nil, 2}, {1, nil, 2}, {4, nil, 0}} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			g.r.mlvlFn = func(a int, _ d2.Difficulty) int { return 1 } // every monster is far below clvl 12
			g.w.set(func(w *world) {
				w.class, w.level, w.area = class, 12, 1
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 {
						keep = append(keep, u)
					}
				}
				w.items = append(keep, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
				for i := 0; i < tc.potions; i++ {
					w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
				}
			})
			g.r.poll()
			for _, a := range tc.cleared {
				g.r.loop.cleared[a] = true
			}
			got, ok := g.r.povertyFarmArea(g.r.latest())
			if tc.potions == 2 && len(tc.cleared) == 0 {
				if a, _ := g.r.starveArea(g.r.latest()); a != tc.want {
					t.Errorf("%v: starveArea %d, want the measured best %d", class, a, tc.want)
				}
			}
			if got != tc.want || ok != (tc.want != 0) {
				t.Errorf("%v potions %d cleared %v: area %d ok %v, want %d", class, tc.potions, tc.cleared, got, ok, tc.want)
			}
		}
	}
}

// A hero with no healing potion does not walk into an area whose monsters are above clvl minus the gap (3): clvl 8, Dark Wood
// (monster level 8) is closed, Stony Field (5) is open; with a potion both are; the owner's try anyway is not asked here. Two classes.
func TestNoPotionsKeepsTheHeroOutOfTooHighAreas(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, potions := range []int{0, 2} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			g.r.mlvlFn = func(a int, _ d2.Difficulty) int { return map[int]int{5: 8, 4: 5}[a] }
			g.w.set(func(w *world) {
				w.class, w.level, w.area = class, 8, 1
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 {
						keep = append(keep, u)
					}
				}
				w.items = keep
				for i := 0; i < potions; i++ {
					w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
				}
			})
			g.r.poll()
			s := g.r.latest()
			high, low := g.r.povertyAreaWhy(s, 5), g.r.povertyAreaWhy(s, 4)
			if (potions == 0) != (high != "") || low != "" {
				t.Errorf("%v potions %d: Dark Wood %q, Stony Field %q", class, potions, high, low)
			}
		}
	}
}

// The quest-walk kit (earlyroutes.json questWalk): from clvl 5 to 14 in Normal a hero with no healing potions does not walk a
// quest: a chores trip when 60 gold pays them, a gold farm when it does not; at clvl 15 and over, or with the kit, nothing.
// Two classes.
func TestQuestWalkNeedsTheKit(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			level, gold int
			potions     int
			want        string
			kitOnly     bool
		}{
			{12, 11, 0, "farm gold first", false},
			{12, 500, 0, "healing potions < 2", true},
			{12, 500, 3, "", false},
			{3, 0, 0, "", false},
			{15, 0, 0, "", false},
		} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			g.w.set(func(w *world) {
				w.class, w.level, w.area, w.gold0 = class, tc.level, 1, tc.gold
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 {
						keep = append(keep, u)
					}
				}
				w.items = keep
				for i := 0; i < tc.potions; i++ {
					w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
				}
				w.items = append(w.items, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
			})
			g.r.poll()
			s := g.r.latest()
			s.Player.Gold = tc.gold
			why, kit := g.r.questWalkCheck(s)
			if tc.want == "" && why != "" || tc.want != "" && !strings.Contains(why, tc.want) || kit != tc.kitOnly {
				t.Errorf("%v clvl %d gold %d potions %d: %q kit %v, want %q kit %v", class, tc.level, tc.gold, tc.potions, why, kit, tc.want, tc.kitOnly)
			}
		}
	}
}

// Questy 2026-11-26: no town portal and 1 potion, it entered Stony Field at 90/176 life and walked home through 24 hostiles. A
// lean hero (no portal, fewer than noPortalLean.potionsBelow potions) counts the first field only for the walk home and does not
// enter another field below the entry life line; a portal, four potions or a healthy hero change that. Two classes.
func TestLeanHeroStaysNearHomeAndHealthy(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			tp      bool
			potions int
			life    int
			lean    bool
			blocked bool
		}{{false, 1, 90, true, true}, {false, 1, 170, true, false}, {true, 1, 90, false, false}, {false, 4, 90, false, false}, {false, 0, 90, true, true}} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			g.w.set(func(w *world) {
				w.class, w.level, w.area, w.life, w.maxL = class, 12, 2, tc.life, 176
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 && u.Class != 5 {
						keep = append(keep, u)
					}
				}
				w.items = keep
				if tc.tp {
					w.items = append(w.items, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
				}
				for i := 0; i < tc.potions; i++ {
					w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
				}
			})
			g.r.poll()
			s := g.r.latest()
			if got := g.r.leanHero(s); got != tc.lean {
				t.Errorf("%v tp %v potions %d: lean %v, want %v", class, tc.tp, tc.potions, got, tc.lean)
			}
			if why := g.r.leanEntryWhy(s, 4); (why != "") != tc.blocked {
				t.Errorf("%v tp %v potions %d life %d: entry %q, blocked want %v", class, tc.tp, tc.potions, tc.life, why, tc.blocked)
			}
			if why := g.r.leanEntryWhy(s, 2); why != "" {
				t.Errorf("%v: the current area is never an entry: %q", class, why)
			}
		}
	}
}

// Questy 2026-11-26: it carried a portal scroll and 0 to 1 potions and walked into Blood Moor and Stony Field (invariant
// field-entry-no-kit, seven hits) and died in Cave Level 1. Below povertyfarm.json kitFloor (4 potions, from clvl 8, Normal) a hero
// picks the first field only, scroll or not; four potions, a lower level or the area it stands in change that. Two classes.
func TestKitFloorKeepsThePoorHeroInTheFirstField(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			level, potions, area int
			want4, want2         bool // area 4 / area 2 closed
		}{{12, 1, 1, true, false}, {12, 0, 1, true, false}, {12, 4, 1, false, false}, {5, 0, 1, false, false}, {12, 1, 4, false, false}} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			g.w.set(func(w *world) {
				w.class, w.level, w.area = class, tc.level, tc.area
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 {
						keep = append(keep, u)
					}
				}
				w.items = append(keep, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
				for i := 0; i < tc.potions; i++ {
					w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
				}
			})
			g.r.poll()
			s := g.r.latest()
			if got := g.r.kitFloorWhy(s, 4) != ""; got != tc.want4 {
				t.Errorf("%v clvl %d potions %d in %d: area 4 closed %v, want %v", class, tc.level, tc.potions, tc.area, got, tc.want4)
			}
			if got := g.r.kitFloorWhy(s, 2) != ""; got != tc.want2 {
				t.Errorf("%v clvl %d potions %d in %d: first field closed %v, want %v", class, tc.level, tc.potions, tc.area, got, tc.want2)
			}
		}
	}
}

// portalFighter reports a fixed safety verdict and whether a portal cast is under way.
type portalFighter struct {
	Fighter
	verdict  SafetyAction
	inFlight bool
}

func (f portalFighter) Safety(*d2.Snapshot) SafetyAction { return f.verdict }
func (f portalFighter) PortalInFlight() bool             { return f.inFlight }

// Questy death 15: the guard cancelled the action (the fight loop was casting the town portal) with the very verdict the loop was
// carrying out, so the cast died "before the click". With a cast in flight the guard keeps watching; without one it cancels as before.
// Two classes.
func TestGuardLeavesARunningPortalCastAlone(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, inFlight := range []bool{true, false} {
			g := newRig(t)
			g.w.set(func(w *world) { w.class, w.level, w.area, w.life = class, 12, 2, 50 })
			g.r.poll()
			g.r.d.Fighter = portalFighter{Fighter: g.r.d.Fighter, verdict: SafetyTownPortal, inFlight: inFlight}
			cancelled := false
			got := g.r.guardOnce(context.Background(), func(error) { cancelled = true }, StateClearing)
			if got == inFlight || cancelled == inFlight {
				t.Errorf("%v inFlight %v: guard returned %v cancelled %v", class, inFlight, got, cancelled)
			}
		}
	}
}

// The poverty deadlock of Questy 2026-11-26: 3 potions and 14 gold at clvl 13 with only Blood Moor open (1.4 gold per minute, 106
// refusals). The kitFloor tiers (admission.json): two potions open the first two fields and any area whose monsters are six levels
// under the hero; below two potions the first field only; four potions anything. The poverty farm then picks the best measured
// gold per minute among the open areas. Two classes.
func TestKitFloorTiersEndThePovertyDeadlock(t *testing.T) {
	mlvl := map[int]int{2: 1, 3: 3, 4: 5, 9: 3, 5: 8, 8: 4}
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			potions   int
			farm      int          // the poverty farm pick (0: none)
			open      map[int]bool // area -> open
			mlvlTweak map[int]int
		}{
			{3, 9, map[int]bool{2: true, 3: true, 4: true, 9: true, 5: false}, nil},
			{3, 4, map[int]bool{2: true, 3: true, 4: true, 9: false}, map[int]int{9: 9}},
			{1, 2, map[int]bool{2: true, 3: false, 4: false, 9: false}, nil},
			{4, 0, map[int]bool{2: true, 3: true, 4: true, 9: true, 5: true}, nil},
		} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			ml := map[int]int{}
			for k, v := range mlvl {
				ml[k] = v
			}
			for k, v := range tc.mlvlTweak {
				ml[k] = v
			}
			g.r.mlvlFn = func(a int, _ d2.Difficulty) int { return ml[a] }
			g.w.set(func(w *world) {
				w.class, w.level, w.area, w.gold0 = class, 13, 1, 14
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 {
						keep = append(keep, u)
					}
				}
				w.items = append(keep, d2.Unit{ID: 799, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}})
				for i := 0; i < tc.potions; i++ {
					w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
				}
			})
			g.r.poll()
			s := g.r.latest()
			for a, want := range tc.open {
				if got := g.r.kitFloorWhy(s, a) == ""; got != want {
					t.Errorf("%v potions %d: area %d open %v, want %v (%s)", class, tc.potions, a, got, want, g.r.kitFloorWhy(s, a))
				}
			}
			if a, ok := g.r.povertyFarmArea(s); a != tc.farm || ok != (tc.farm != 0) {
				t.Errorf("%v potions %d: farm area %d ok %v, want %d", class, tc.potions, a, ok, tc.farm)
			}
		}
	}
}

// The quest-walk kit rule (no portal scroll, gold under the price: "farm gold first") never closes the first field: the hero must
// be able to go and earn the gold (Questy 2026-11-26, 28 minutes of frozen xp). Other fields stay closed. Two classes.
func TestQuestWalkRuleLeavesTheFirstFieldOpen(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
		g.w.set(func(w *world) {
			w.class, w.level, w.area, w.gold0 = class, 13, 1, 19
			var keep []d2.Unit
			for _, u := range w.items {
				if u.Class != 1 && u.Class != 5 {
					keep = append(keep, u)
				}
			}
			w.items = keep
			for i := 0; i < 3; i++ {
				w.items = append(w.items, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
			}
		})
		g.r.poll()
		var w task.World
		g.r.fillEarly(&w)
		if w.WalkBlocked.Why == "" || w.FirstField == 0 {
			t.Fatalf("%v: walk %+v first field %d", class, w.WalkBlocked, w.FirstField)
		}
		first, other := w.FirstField, w.FirstField+1
		w.Area = 1
		if f := task.StepClear("c", "clear", first, func(task.World) bool { return false }, 60*1e9).Ready(w); f != nil {
			t.Errorf("%v: the first field is closed: %+v", class, f)
		}
		if f := task.StepClear("c", "clear", other, func(task.World) bool { return false }, 60*1e9).Ready(w); f == nil {
			t.Errorf("%v: another field is open without the kit", class)
		}
	}
}
