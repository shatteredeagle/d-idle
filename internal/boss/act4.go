package boss

import (
	"context"
	"fmt"
	"time"
)

// Normal Act 4 boss scripts (docs/act4.md section 6). Tags as in act3.go: V = read from the real 1.14d tables
// (monstats, superuniques, levels, objects), S = Kolbot (Izual.js, Hephasto.js, Diablo.js, Common.js) or Arreat,
// U = never run against the live game. Every stand point, radius and floor that shapes a fight is U.

const (
	NameIzual    = "Izual"
	NameHephasto = "Hephasto the Armorer"
	NameDiablo   = "Diablo"
)

// Monster classes (V monstats hcIdx): Izual 256, Hephasto 409 (SU 41), Diablo 243.
const (
	ClassIzual    = 256
	ClassHephasto = 409
	ClassDiablo   = 243

	uniqueHephasto = 41
)

// Act 4 area ids (V levels.txt).
const (
	AreaCityOfTheDamned = 106 // waypoint 28
	AreaPlainsOfDespair = 105 // Izual, no waypoint
	AreaRiverOfFlame    = 107 // Hellforge, waypoint 29
	AreaChaosSanctuary  = 108 // data name "Chaos Sanctum"
)

// Object classes the scripts anchor on (V objects.txt, roles S): the Hellforge.
const presetHellforge = 376

// Act4 returns fresh copies of the Normal Act 4 scripts keyed by boss name. Diablo is in the map for floors, route
// and loot; it must be run through RunDiablo, because the seals have to be opened before he spawns.
func Act4() map[string]*Script {
	return map[string]*Script{
		NameIzual:    izual(),
		NameHephasto: hephasto(),
		NameDiablo:   diablo(),
	}
}

// Izual: City of the Damned waypoint, then a level crossing to the Plains of Despair (no waypoint there, V). Kolbot
// walks to the preset monster and stands 50..60 subtiles off before engaging (S, RushThread moveIntoPos); the stand
// points below approximate that with a ranged side step and are U. All resists are 30 and cold is 75 (V), so cold
// builds are weak against him; his Holy Freeze-like chill is old lore (U), so the script steps away when hurt like
// Duriel's. Floor 30 (runs.json R).
func izual() *Script {
	return &Script{
		Name: NameIzual, Act: 4, WP: AreaCityOfTheDamned, Route: []int{AreaPlainsOfDespair}, Arena: AreaPlainsOfDespair,
		Match:  Match{Class: ClassIzual, UniqueNo: -1},
		Anchor: Anchor{Preset: &PresetRef{Kind: PresetNPC, ID: ClassIzual}},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-14, 14}, Rel: true},
			{Name: "east", Pos: Pt{14, 14}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 12},
		Hazards:  []Hazard{NoPotionLowLife(35), ColdAura(7, 12, 55), RaisedAdds(6, 12, 10*time.Second)},
		MinLevel: [3]int{30, 55, 78}, MinPotions: 4, RouteMinLifePct: 50,
		Timeout: 15 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 2 * time.Second, PostClear: 25,
		Loot: LootSpec{Radius: 30, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// Hephasto the Armorer: super unique 41 standing at the Hellforge in the River of Flame (waypoint 29). The preset of
// the forge object is the anchor (S Kolbot Hephasto.js). He carries the Hell Forge Hammer, a quest item that is
// always taken. Normal poison resist is -50 (V). The task layer only sends the character when the Soulstone is in
// the pack.
func hephasto() *Script {
	return &Script{
		Name: NameHephasto, Act: 4, WP: AreaRiverOfFlame, Arena: AreaRiverOfFlame,
		Match:  Match{Class: ClassHephasto, UniqueNo: uniqueHephasto, Name: "Hephasto the Armorer"},
		Anchor: Anchor{Preset: &PresetRef{Kind: PresetObject, ID: presetHellforge}},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-12, 12}, Rel: true},
			{Name: "east", Pos: Pt{12, 12}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 12},
		Hazards:  []Hazard{NoPotionLowLife(35), RaisedAdds(6, 12, 10*time.Second)},
		MinLevel: [3]int{28, 55, 78}, MinPotions: 3,
		Timeout: 10 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 2 * time.Second, PostClear: 20,
		Loot: LootSpec{Radius: 30, Quest: []string{"Hell Forge Hammer"}, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// FireWall steps away from a Diablo fire wall (Fire Storm / Lightning Hose pools, S) when the player stands within
// near subtiles of one. U: the adapter must produce Danger{Kind: DangerFireWall} from missiles/objects.
func FireWall(near, away int) Hazard {
	return Hazard{Name: "fire-wall", Cooldown: 1500 * time.Millisecond, Do: Response{Kind: RespStepAway, Radius: away},
		Detect: func(c Ctx) bool { _, ok := nearestDanger(c.V, DangerFireWall, near); return ok }}
}

// Diablo: the arena is the Chaos Sanctuary (River of Flame waypoint, then the Vis link 107 -> 108, V). Once the five
// seals are open he spawns at the star (7791,5293; S Kolbot). Normal resists are 33 fire/light/cold and 50 poison
// (V), so physical and magic builds do best. Fire walls are the killer (S): FireWall steps away, NoPotionLowLife
// escapes. Floor 30 (runs.json R, progression-normal). The script is run through RunDiablo.
func diablo() *Script {
	star := ChaosStar
	return &Script{
		Name: NameDiablo, Act: 4, WP: AreaRiverOfFlame, Route: []int{AreaChaosSanctuary}, Arena: AreaChaosSanctuary,
		Match:  Match{Class: ClassDiablo, UniqueNo: -1, Name: NameDiablo},
		Anchor: Anchor{Boss: true, Hint: &star},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-16, 14}, Rel: true},
			{Name: "east", Pos: Pt{16, 14}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 14},
		Hazards:  []Hazard{NoPotionLowLife(35), FireWall(6, 14), RaisedAdds(8, 12, 10*time.Second)},
		MinLevel: [3]int{30, 55, 80}, MinPotions: 5, RouteMinLifePct: 50,
		Timeout: 20 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 3 * time.Second, PostClear: 30,
		Loot: LootSpec{Radius: 35, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// RunDiablo is the whole Terror's End leg: reach the Sanctuary (River of Flame waypoint, then the Vis hop), open the
// five seals with OpenSeals (killing the three seal bosses), then fight Diablo with the shared runner. OpenSeals
// resumes where a previous run stopped, so calling it again after a chicken or a restart in the same game is safe.
// The Opts.Hooks are passed to the Diablo fight only. A result from the seal phase is merged into Result.Notes.
func RunDiablo(ctx context.Context, w World, a Actions, ops ObjectOps, so SealOpts, o Opts) (Result, error) {
	s := diablo()
	v, err := w.View(ctx)
	if err != nil {
		return Result{Boss: s.Name}, err
	}
	if v.Dead {
		return Result{Boss: s.Name}, ErrDead
	}
	if f := s.MinLevel[clampDiff(v.Difficulty)]; f > 0 && v.Level < f {
		return Result{Boss: s.Name}, fmt.Errorf("%w: %s needs clvl %d, have %d", ErrTooWeak, s.Name, f, v.Level)
	}
	if ops == nil {
		return Result{Boss: s.Name}, fmt.Errorf("%w: no object capability for the seals", ErrAborted)
	}
	if err := reachArena(ctx, s, w, a, v); err != nil {
		return Result{Boss: s.Name}, err
	}
	t0 := w.Now()
	sr, err := OpenSeals(ctx, w, a, ops, so)
	pre := Result{Boss: s.Name, Phases: []string{"seals"}, Notes: sr.Notes}
	pre.Notes = append(pre.Notes, fmt.Sprintf("layouts %v opened %v bosses %v", sr.Layouts, sr.Opened, sr.Bosses))
	if err != nil {
		pre.Elapsed = w.Now().Sub(t0)
		return pre, err
	}
	res, err := Run(ctx, s, w, a, o)
	res.Phases = append(pre.Phases, res.Phases...)
	res.Notes = append(pre.Notes, res.Notes...)
	res.Elapsed = w.Now().Sub(t0)
	return res, err
}

// reachArena is the manual form of the runner's approach for composite runs that need work between arrival and the
// fight: potion check, waypoint travel (skipped when already there), then ExitTo each route area, then the arena.
func reachArena(ctx context.Context, s *Script, w World, a Actions, v View) error {
	return walkChain(ctx, s, w, a, v, append(append([]int{}, s.Route...), s.Arena))
}

func walkChain(ctx context.Context, s *Script, w World, a Actions, v View, hops []int) error {
	if v.Area == hops[len(hops)-1] {
		return nil
	}
	if s.MinPotions > 0 && v.Potions < s.MinPotions {
		return fmt.Errorf("%w: %s needs %d potions, have %d", ErrAborted, s.Name, s.MinPotions, v.Potions)
	}
	if s.WP > 0 && v.Area != s.WP {
		inRoute := false
		for _, h := range hops {
			inRoute = inRoute || h == v.Area
		}
		if !inRoute {
			if err := a.Travel(ctx, s.WP); err != nil {
				return fmt.Errorf("boss: travel to %d: %w", s.WP, err)
			}
		}
	} else if s.WP == 0 && v.Area != hops[0] {
		if err := a.Travel(ctx, hops[0]); err != nil {
			return fmt.Errorf("boss: travel to %d: %w", hops[0], err)
		}
		hops = hops[1:]
	}
	for _, h := range hops {
		nv, err := w.View(ctx)
		if err != nil {
			return err
		}
		if nv.Dead {
			return ErrDead
		}
		if nv.Area == h {
			continue
		}
		if err := a.ExitTo(ctx, h); err != nil {
			return fmt.Errorf("boss: exit to %d: %w", h, err)
		}
		if nv, err = w.View(ctx); err != nil {
			return err
		}
		if nv.Area != h {
			return fmt.Errorf("%w: wanted area %d, in %d", ErrLeftArena, h, nv.Area)
		}
		if s.RouteMinLifePct > 0 && nv.LifePct() < s.RouteMinLifePct {
			return fmt.Errorf("%w: life %d%% in area %d", ErrAborted, nv.LifePct(), nv.Area)
		}
	}
	return nil
}
