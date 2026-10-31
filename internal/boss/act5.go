package boss

import (
	"context"
	"fmt"
	"time"
)

// Normal Act 5 boss scripts (docs/act5.md section 6). Tags as in act4.go: V = real 1.14d tables, S = Kolbot
// (Shenk, Nihlathak, Ancients, Baal; Common.js) or Arreat, U = never run against the live game.

const (
	NameShenk     = "Shenk the Overseer"
	NameNihlathak = "Nihlathak"
	NameAncients  = "The Ancients"
	NameTalic     = "Talic"
	NameMadawc    = "Madawc"
	NameKorlic    = "Korlic"
	NameBaal      = "Baal"
)

// Monster classes (V monstats hcIdx) and super unique numbers (V superuniques).
const (
	ClassShenk      = 479
	ClassNihlathak  = 526
	ClassTalic      = 540
	ClassMadawc     = 541
	ClassKorlic     = 542
	ClassBaalThrone = 543
	ClassBaal       = 544
	ClassBaalClone  = 570 // V exists (`baalclone`)
	ClassCouncilA5  = 557 // wave 3 (S)
	ClassLister     = 571 // wave 5, with minions 572/573 (S)

	uniqueShenk     = 42
	uniqueNihlathak = 60
	uniqueTalic     = 43
	uniqueMadawc    = 44
	uniqueKorlic    = 45
)

// Act 5 area ids (V levels.txt; docs/act5.md corrects the older Halls ids: Pain 123, Vaught 124).
const (
	AreaBloodyFoothills = 110 // Shenk, no waypoint
	AreaAncientsWay     = 118 // waypoint 37
	AreaArreatSummit    = 120
	AreaHallsOfPain     = 123 // waypoint 35
	AreaHallsOfVaught   = 124
	AreaWorldstoneKeep2 = 129 // waypoint 38
	AreaWorldstoneKeep3 = 130
	AreaThrone          = 131
	AreaWorldstoneChamb = 132
)

// Objects (V objects.txt, roles S): Nihlathak's platform 462, the Ancients' altar 546, Baal's portal 563.
const (
	presetNihlathakPlatform = 462
	objAncientsAltar        = 546
	objBaalPortal           = 563
)

// TombViperClasses are the monster classes of revived Tomb Vipers, whose presence aborts the Nihlathak run (S Kolbot).
// U: the class ids were not dumped, so the list is empty and the hazard never fires until a live reading fills it.
var TombViperClasses []int

// Act5 returns fresh copies of the scripts of Normal Act 5 keyed by boss name. The Ancients and Baal need composite
// runs (RunAncients, RunBaal) because of the altar, the no-portal rule, the waves and the chamber portal; their
// entries here (Talic, Madawc, Korlic, Baal) carry the data for those runs.
func Act5() map[string]*Script {
	return map[string]*Script{
		NameShenk:     shenk(),
		NameNihlathak: nihlathak(),
		NameTalic:     ancient(NameTalic, ClassTalic, uniqueTalic),
		NameMadawc:    ancient(NameMadawc, ClassMadawc, uniqueMadawc),
		NameKorlic:    ancient(NameKorlic, ClassKorlic, uniqueKorlic),
		NameBaal:      baal(),
	}
}

// Shenk the Overseer: Bloody Foothills has no waypoint (V); travel from town walks the level crossing. Kolbot's
// positions disagree by up to 40 subtiles ((3846,5120), (3883,5113), (3876,5130), S), so the monster list is the
// anchor and the first one is only a hint. Floor 25 (runs.json R).
func shenk() *Script {
	hint := Pt{3883, 5113}
	return &Script{
		Name: NameShenk, Act: 5, Arena: AreaBloodyFoothills,
		Match:  Match{Class: ClassShenk, UniqueNo: uniqueShenk, Name: NameShenk},
		Anchor: Anchor{Boss: true, Hint: &hint, HintSlop: 80},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-12, 12}, Rel: true},
			{Name: "east", Pos: Pt{12, 12}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 12},
		Hazards:  []Hazard{NoPotionLowLife(35), RaisedAdds(7, 12, 12*time.Second)},
		MinLevel: [3]int{25, 50, 72}, MinPotions: 3,
		Timeout: 12 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 2 * time.Second, PostClear: 30,
		Loot: LootSpec{Radius: 35, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// AbortOnClasses aborts the run when at least min living hostiles of the given classes are in view. With no classes
// it never fires.
func AbortOnClasses(name string, classes []int, min int) Hazard {
	return Hazard{Name: name, Do: Response{Kind: RespAbort},
		Detect: func(c Ctx) bool {
			n := 0
			for _, m := range c.V.Hostiles() {
				if containsInt(classes, m.Class) {
					n++
				}
			}
			return len(classes) > 0 && n >= min
		}}
}

// Nihlathak: Halls of Pain waypoint (35), one hop to the Halls of Vaught, preset 462 is his platform (S). His Corpse
// Explosion on raised corpses is lethal (V Arreat text), so adds are cleared first (RaisedAdds) and the run aborts
// on revived Tomb Vipers (S, classes U). Normal cold and poison resist is 70 (V). Floor 30 (U; Nihlathak's area is
// level 36 in the table, V).
func nihlathak() *Script {
	return &Script{
		Name: NameNihlathak, Act: 5, WP: AreaHallsOfPain, Route: []int{AreaHallsOfVaught}, Arena: AreaHallsOfVaught,
		Match:  Match{Class: ClassNihlathak, UniqueNo: uniqueNihlathak, Name: NameNihlathak},
		Anchor: Anchor{Preset: &PresetRef{Kind: PresetObject, ID: presetNihlathakPlatform}},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-14, 12}, Rel: true},
			{Name: "east", Pos: Pt{14, 12}, Rel: true},
		},
		Kite:     &Kite{Range: 9, Every: 2 * time.Second, Away: 14},
		Hazards:  []Hazard{NoPotionLowLife(35), AbortOnClasses("revived-tomb-vipers", TombViperClasses, 1), RaisedAdds(5, 12, 10*time.Second)},
		MinLevel: [3]int{30, 55, 78}, MinPotions: 4, RouteMinLifePct: 50,
		Timeout: 12 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 2 * time.Second, PostClear: 25,
		Loot: LootSpec{Radius: 30, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// ancient is one of Talic, Madawc, Korlic. No town portal, never leave the area (V text); NoTP turns every escape
// into a chicken. The three attack together once the altar is touched, so each script stands where the previous
// fight ended. Hard floors are the credit levels 20/40/60 (V Arreat text); the planner uses a higher soft floor.
// Normal resists (V): Talic phys/fire/poison 70, Madawc lightning/cold 70, Korlic phys/fire/poison 70.
func ancient(name string, class, unique int) *Script {
	return &Script{
		Name: name, Act: 5, WP: AreaAncientsWay, Route: []int{AreaArreatSummit}, Arena: AreaArreatSummit,
		Match:  Match{Class: class, UniqueNo: unique, Name: name},
		Anchor: Anchor{Boss: true},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-10, 10}, Rel: true},
			{Name: "east", Pos: Pt{10, 10}, Rel: true},
		},
		Kite:     &Kite{Range: 7, Every: 2 * time.Second, Away: 10},
		Hazards:  []Hazard{NoPotionLowLife(30)},
		MinLevel: [3]int{20, 40, 60}, MinPotions: 6, RouteMinLifePct: 60, NoTP: true,
		Timeout: 12 * time.Minute, ChickenPct: 12, DrinkPct: 55, EscapePct: 25,
		PostKillWait: 0,
		Loot:         LootSpec{Radius: 30, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// DefaultAncientsOrder fights Korlic first (poison/physical/fire walls vs cold/lightning casters), then Talic, then
// Madawc. U: the best order depends on the build; callers pass their own.
func DefaultAncientsOrder() []int { return []int{ClassKorlic, ClassTalic, ClassMadawc} }

// noLoot wraps Actions so that an intermediate kill does not walk off to loot while the other Ancients are alive.
type noLoot struct{ Actions }

func (noLoot) Loot(context.Context, LootSpec) (int, error) { return 0, nil }

// RunAncients is the whole Rite of Passage leg: reach the Arreat Summit, touch the altar (preset 546) unless an
// Ancient is already in view or the altar reads as used, then kill the three in the given order without leaving the
// area (the scripts have NoTP, so a bad moment is a Chicken, not a town portal). Only the last kill loots. A death
// or chicken returns ErrDead/ErrChicken; the Ancients reset when the summit is empty (V text), so the caller starts
// the leg over in a new game.
func RunAncients(ctx context.Context, w World, a Actions, ops ObjectOps, order []int, o Opts) (Result, error) {
	if len(order) == 0 {
		order = DefaultAncientsOrder()
	}
	first := ancientByClass(order[0])
	out := Result{Boss: NameAncients}
	v, err := w.View(ctx)
	if err != nil {
		return out, err
	}
	if v.Dead {
		return out, ErrDead
	}
	if f := first.MinLevel[clampDiff(v.Difficulty)]; f > 0 && v.Level < f {
		return out, fmt.Errorf("%w: %s needs clvl %d, have %d", ErrTooWeak, NameAncients, f, v.Level)
	}
	if err := reachArena(ctx, first, w, a, v); err != nil {
		return out, err
	}
	t0 := w.Now()
	if v, err = w.View(ctx); err != nil {
		return out, err
	}
	if !anyAncient(v) {
		if ops == nil {
			return out, fmt.Errorf("%w: no object capability for the altar", ErrAborted)
		}
		ps, _ := w.Presets(ctx)
		at, ok := presetPos(ps, objAncientsAltar)
		if !ok {
			out.Notes = append(out.Notes, "altar preset not found, clicking where the character stands")
			at = v.Pos
		}
		if mode, err := ops.ObjectMode(ctx, objAncientsAltar, at); err != nil || mode == 0 {
			if err := a.GoTo(ctx, at); err != nil {
				if fe := passFatal(err); fe != nil {
					return out, fe
				}
			}
			if err := ops.UseObject(ctx, objAncientsAltar, at); err != nil {
				return out, fmt.Errorf("boss: altar: %w", err)
			}
			out.Phases = append(out.Phases, "altar")
		}
	}
	for i, class := range order {
		s := ancientByClass(class)
		act := a
		if i < len(order)-1 {
			act = noLoot{a}
		}
		r, err := Run(ctx, s, w, act, o)
		out.Loot += r.Loot
		out.Phases = append(out.Phases, s.Name+":"+fmt.Sprint(r.Phases))
		out.Notes = append(out.Notes, r.Notes...)
		if err != nil {
			out.Elapsed = w.Now().Sub(t0)
			return out, err
		}
	}
	out.Killed = true
	out.Elapsed = w.Now().Sub(t0)
	return out, nil
}

func ancientByClass(class int) *Script {
	switch class {
	case ClassTalic:
		return ancient(NameTalic, ClassTalic, uniqueTalic)
	case ClassMadawc:
		return ancient(NameMadawc, ClassMadawc, uniqueMadawc)
	}
	return ancient(NameKorlic, ClassKorlic, uniqueKorlic)
}

func anyAncient(v View) bool {
	for _, m := range v.Monsters {
		if (m.Class == ClassTalic || m.Class == ClassMadawc || m.Class == ClassKorlic) && m.Alive() {
			return true
		}
	}
	return false
}

func presetPos(ps []Preset, class int) (Pt, bool) {
	for _, p := range ps {
		if p.Kind == PresetObject && p.ID == class {
			return p.Pos, true
		}
	}
	return Pt{}, false
}

// baal is the chamber fight: Baal (class 544) at the end of the Worldstone Chamber, standing point (15134,5923) (S).
// Normal resists are 33 fire/light/cold and 50 poison (V). Duplicates (class 570) are ignored: the Match is the
// real class only (V Arreat text: do not waste time on the fake). The script's arena is the chamber, which is entered
// through the portal object after the waves, so it is run through RunBaal. Floor 32 (runs.json R).
func baal() *Script {
	hint := Pt{15134, 5923}
	return &Script{
		Name: NameBaal, Act: 5, WP: AreaWorldstoneKeep2, Route: []int{AreaWorldstoneKeep3, AreaThrone}, Arena: AreaWorldstoneChamb,
		Match:  Match{Class: ClassBaal, UniqueNo: -1},
		Anchor: Anchor{Boss: true, Hint: &hint},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-14, 14}, Rel: true},
			{Name: "east", Pos: Pt{14, 14}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 14},
		Hazards:  []Hazard{NoPotionLowLife(35), RaisedAdds(8, 12, 10*time.Second)},
		MinLevel: [3]int{32, 58, 80}, MinPotions: 5, RouteMinLifePct: 50,
		Timeout: 20 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 3 * time.Second, PostClear: 30,
		Loot: LootSpec{Radius: 35, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}
