package boss

import (
	"context"
	"fmt"
	"time"
)

// Farm-only boss scripts (post-campaign runs, docs/farm.md): Pindleskin behind Anya's red portal and Eldritch in the
// Frigid Highlands. All of it is U: nothing here ran against the live game. Names, super unique numbers and area ids
// are from docs/merc-farm.md section 2.3/2.4 (V from the extracted superuniques.tsv/strings.tsv: Pindleskin 52,
// Eldritch 49); the red portal object class is S (objects.txt "Red Portal", from memory) and the stand points are none
// on purpose: both are found by the monster list and Explore like Shenk.
const (
	NamePindleskin = "Pindleskin"
	NameEldritch   = "Eldritch the Rectifier"

	uniquePindleskin = 52
	uniqueEldritch   = 49

	AreaHarrogath       = 109
	AreaFrigidHighlands = 111
	AreaNihlathakTemple = 121 // runs.json final_area of the Pindleskin run
	objRedPortal        = 60  // S: objects.txt "Red Portal" (Anya's portal to the temple), U for the live id
	redPortalWait       = 10 * time.Second
)

// FarmScripts returns fresh copies of the scripts the farm runs add to the campaign ones.
func FarmScripts() map[string]*Script {
	return map[string]*Script{
		NamePindleskin: pindleskin(),
		NameEldritch:   eldritch(),
	}
}

// pindleskin stands beside the portal arrival in the temple area (runs.json note). Arena is the temple, so Run starts
// from there once RunPindleskin crossed the portal; never go on to the Halls of Anguish (122).
func pindleskin() *Script {
	return &Script{
		Name: NamePindleskin, Act: 5, Arena: AreaNihlathakTemple,
		Match:    Match{UniqueNo: uniquePindleskin, Name: NamePindleskin},
		Anchor:   Anchor{Boss: true},
		Hazards:  []Hazard{NoPotionLowLife(35), RaisedAdds(7, 12, 12*time.Second)},
		MinLevel: [3]int{25, 50, 70}, MinPotions: 3,
		Timeout: 6 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 2 * time.Second, PostClear: 20,
		Loot: LootSpec{Radius: 30, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
		Exit: ExitTown,
	}
}

// eldritch: the Frigid Highlands waypoint area is the arena (he stands near the waypoint side, S Kolbot), so there is
// no route. The run leaves the character in the Highlands for Shenk (Bloody Foothills, next door), so Exit is Stay.
func eldritch() *Script {
	return &Script{
		Name: NameEldritch, Act: 5, WP: AreaFrigidHighlands, Arena: AreaFrigidHighlands,
		Match:  Match{UniqueNo: uniqueEldritch, Name: NameEldritch},
		Anchor: Anchor{Boss: true},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-12, 12}, Rel: true},
			{Name: "east", Pos: Pt{12, 12}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 12},
		Hazards:  []Hazard{NoPotionLowLife(35), RaisedAdds(7, 12, 12*time.Second)},
		MinLevel: [3]int{25, 50, 70}, MinPotions: 3,
		Timeout: 8 * time.Minute, ExploreBudget: 2 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 2 * time.Second, PostClear: 25,
		Loot: LootSpec{Radius: 30, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// RunPindleskin walks to Harrogath if needed, clicks Anya's red portal, waits for the temple and runs the Pindleskin
// script there. It can also start inside the temple. Without ops the run fails with ErrAborted before any walk.
func RunPindleskin(ctx context.Context, w World, a Actions, ops ObjectOps, o Opts) (Result, error) {
	s := pindleskin()
	out := Result{Boss: s.Name}
	v, err := w.View(ctx)
	if err != nil {
		return out, err
	}
	if v.Dead {
		return out, ErrDead
	}
	if f := s.MinLevel[clampDiff(v.Difficulty)]; f > 0 && v.Level < f {
		return out, fmt.Errorf("%w: %s needs clvl %d, have %d", ErrTooWeak, s.Name, f, v.Level)
	}
	t0 := w.Now()
	if v.Area != AreaNihlathakTemple {
		if ops == nil {
			return out, fmt.Errorf("%w: no object capability for the red portal", ErrAborted)
		}
		if v.Area != AreaHarrogath {
			if err := a.Travel(ctx, AreaHarrogath); err != nil {
				return out, fmt.Errorf("boss: travel to Harrogath: %w", err)
			}
		}
		ps, _ := w.Presets(ctx)
		at, ok := presetPos(ps, objRedPortal)
		if !ok {
			out.Notes = append(out.Notes, "red portal preset not found, clicking where the character stands")
			if nv, err := w.View(ctx); err == nil {
				at = nv.Pos
			}
		} else if err := a.GoTo(ctx, at); err != nil {
			if fe := passFatal(err); fe != nil {
				return out, fe
			}
		}
		if err := ops.UseObject(ctx, objRedPortal, at); err != nil {
			return out, fmt.Errorf("boss: red portal: %w", err)
		}
		if err := waitArea(ctx, w, AreaNihlathakTemple, redPortalWait); err != nil {
			return out, err
		}
		out.Phases = append(out.Phases, "portal")
	}
	res, err := Run(ctx, s, w, a, o)
	res.Phases = append(out.Phases, res.Phases...)
	res.Notes = append(out.Notes, res.Notes...)
	res.Elapsed = w.Now().Sub(t0)
	return res, err
}

// FarmRunner is Act45Runner plus the farm-only bosses; it satisfies the same KillBoss/LastResult pair the task layer
// and the farm loop use. Scripts must hold the campaign scripts (AllActs) for the names it delegates, e.g. Shenk.
type FarmRunner struct {
	Act45Runner
	Farm map[string]*Script // nil = FarmScripts()
}

// KillBoss runs the named boss: Pindleskin through the red portal, Eldritch from his script, the rest as Act45Runner.
func (r *FarmRunner) KillBoss(ctx context.Context, name string) error {
	switch name {
	case NamePindleskin:
		res, err := RunPindleskin(ctx, r.W, r.A, r.Ops, r.Opts)
		r.Last = res
		return err
	case NameEldritch:
		if r.Farm == nil {
			r.Farm = FarmScripts()
		}
		res, err := Run(ctx, r.Farm[NameEldritch], r.W, r.A, r.Opts)
		r.Last = res
		return err
	}
	if r.Scripts == nil {
		r.Scripts = AllActs()
	}
	return r.Act45Runner.KillBoss(ctx, name)
}

// LastResult is the result of the latest KillBoss.
func (r *FarmRunner) LastResult() Result { return r.Last }
