package boss

import (
	"context"
	"errors"
	"time"
)

// Normal Act 3 boss scripts. Tags as in act2.go: V = read from the real 1.14d tables (docs/act3.md section 6), S =
// Kolbot (Travincal.js, Mephisto.js, RushThread.js) or Arreat, U = never run against the live game. Every stand
// point, radius and floor that shapes a fight is U.

const (
	NameCouncil  = "High Council"
	NameCouncil2 = "High Council 2"
	NameCouncil3 = "High Council 3"
	NameMephisto = "Mephisto"
)

// Monster classes. V monstats hcIdx: council members 345/346/347, Mephisto 242. Which of the six council super uniques
// (SU 26..31) holds which class, and which three spawn per game, is U; matching by class covers all of them.
const (
	ClassCouncil1 = 345
	ClassCouncil2 = 346
	ClassCouncil3 = 347
	ClassMephisto = 242
)

// Act 3 area ids (V levels.txt).
const (
	AreaTravincal  = 83
	AreaDurance2   = 101 // waypoint
	AreaDurance3   = 102 // Mephisto
	modeAttacking2 = 6   // S: D2 monster mode ATTACK2 (0 death, 1 neutral, 2 walk, 3 run, 4 gethit, 5 attack1, 6 attack2, 7 block, 8 cast)
)

// Act3 returns fresh copies of the Normal Act 3 scripts keyed by boss name.
func Act3() map[string]*Script {
	return map[string]*Script{
		NameCouncil:  council(NameCouncil, ClassCouncil1),
		NameCouncil2: council(NameCouncil2, ClassCouncil2),
		NameCouncil3: council(NameCouncil3, ClassCouncil3),
		NameMephisto: mephisto(),
	}
}

// CouncilNames are the scripts RunCouncil runs, in order. The task layer asks for "High Council"; a Runner that only
// looks scripts up by name should go through RunCouncil so all three members die.
func CouncilNames() []string { return []string{NameCouncil, NameCouncil2, NameCouncil3} }

// council is one member. Travincal has a waypoint (V, wp 25) and the three stand close together near the temple steps
// (Kolbot fights at waypoint + (101,-56), S), so PostClear after each kill sweeps the other two and their packs. The
// quest item Khalim's Flail drops from one member (U which), so every member's loot spec asks for it by name.
func council(name string, class int) *Script {
	return &Script{
		Name: name, Act: 3, WP: AreaTravincal, Arena: AreaTravincal,
		Match:  Match{Class: class, UniqueNo: -1},
		Anchor: Anchor{Boss: true},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-12, 12}, Rel: true},
			{Name: "east", Pos: Pt{12, 12}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 12},
		Hazards:  []Hazard{NoPotionLowLife(35), RaisedAdds(7, 12, 12*time.Second)},
		MinLevel: [3]int{24, 48, 70}, // runs.json floors (V that they are in the repo; sourcing U)
		Timeout:  15 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50, MinPotions: 3,
		PostKillWait: 2 * time.Second, PostClear: 40,
		Loot: LootSpec{Radius: 35, Quest: []string{"Khalim's Flail"}, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// RunCouncil kills the three members one after another and returns the merged result. The first member is required
// (its errors are returned). The other two are best effort: if the game did not spawn them or the sweep already killed
// them the shorter run ends in ErrBossNotFound or ErrTimeout, which is recorded as a note and not an error (U: whether
// all three always spawn; Kolbot clears the area instead of waiting for fixed monsters, S).
func RunCouncil(ctx context.Context, w World, a Actions, o Opts) (Result, error) {
	sc := Act3()
	first, err := Run(ctx, sc[NameCouncil], w, a, o)
	first.Boss = NameCouncil
	if err != nil {
		return first, err
	}
	for _, n := range []string{NameCouncil2, NameCouncil3} {
		s := sc[n]
		s.Timeout = 3 * time.Minute
		r, err := Run(ctx, s, w, a, o)
		first.Loot += r.Loot
		first.Phases = append(first.Phases, r.Phases...)
		first.Notes = append(first.Notes, r.Notes...)
		first.Elapsed += r.Elapsed
		switch {
		case err == nil:
		case errors.Is(err, ErrBossNotFound) || errors.Is(err, ErrTimeout):
			first.Notes = append(first.Notes, n+": "+err.Error()+" (not counted as a failure)")
		default:
			return first, err
		}
	}
	return first, nil
}

// Mephisto: Durance of Hate 3 via the waypoint on level 2 (S Kolbot: useWaypoint(101), then the stairs). He stands
// near the arrival point (S Kolbot stands at (17566,8069), kept as a hint only). He is a caster; Normal resists are 25..50
// percent (V), and at Normal there are no immunities so any build works. Kolbot sidesteps when his mode is
// Attacking2 (S, U what that attack is: the notes say lightning/hydra source), which the hazard below does. The moat
// trick (teleport sorceress casts from across the water at (17610,8094)) is a stand point that is skipped unless the
// caller sets Opts.CanTeleport (U). The quest item Mephisto's Soulstone is picked up for the Act 4 Hellforge.
// Floor 28 (runs.json).
func mephisto() *Script {
	hint := Pt{17566, 8069}
	return &Script{
		Name: NameMephisto, Act: 3, WP: AreaDurance2, Route: []int{AreaDurance3}, Arena: AreaDurance3,
		Match:  Match{Class: ClassMephisto, UniqueNo: -1, Name: NameMephisto},
		Anchor: Anchor{Boss: true, Hint: &hint},
		Stand: []StandPoint{
			{Name: "moat", Pos: Pt{17610, 8094}, NeedsTeleport: true},
			{Name: "west", Pos: Pt{-14, 14}, Rel: true},
			{Name: "east", Pos: Pt{14, 14}, Rel: true},
		},
		Kite:     &Kite{Range: 7, Every: 2 * time.Second, Away: 14},
		Hazards:  []Hazard{NoPotionLowLife(35), SidestepOnMode(modeAttacking2, 18), RaisedAdds(6, 12, 10*time.Second)},
		MinLevel: [3]int{28, 55, 75}, MinPotions: 4, RouteMinLifePct: 50,
		Timeout: 20 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 3 * time.Second, PostClear: 25,
		Loot: LootSpec{Radius: 30, Quest: []string{"Mephisto's Soulstone"}, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// SidestepOnMode steps away from the boss when its animation mode equals mode and the player is closer than away.
// Cooldown stops the script from dancing between attacks. U: the mode value and whether the adapter reads Mon.Mode
// every loop (the Act 1 scripts only use it for alive/dead).
func SidestepOnMode(mode, away int) Hazard {
	return Hazard{Name: "boss-mode-sidestep", Cooldown: 3 * time.Second, Do: Response{Kind: RespStepAway, Radius: away},
		Detect: func(c Ctx) bool {
			return c.Boss != nil && c.Boss.Alive() && c.Boss.Mode == mode && c.V.Pos.Dist(c.Boss.Pos) < away
		}}
}

// AllActs is All() plus the Act 3, 4 and 5 scripts (All is in act2.go and keeps its Acts 1+2 contract). Diablo, the
// Ancients and Baal are in the map for their data but run through RunDiablo, RunAncients and RunBaal (Act45Runner).
func AllActs() map[string]*Script {
	m := All()
	for _, act := range []map[string]*Script{Act3(), Act4(), Act5()} {
		for k, v := range act {
			m[k] = v
		}
	}
	return m
}
