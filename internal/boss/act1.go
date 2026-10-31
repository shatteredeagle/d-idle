package boss

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/env"
)

// Boss names, equal to the task package constants (BossBR ...), so task.Actions.KillBoss(name) maps directly.
const (
	NameBloodRaven = "Blood Raven"
	NameCountess   = "The Countess"
	NameSmith      = "The Smith"
	NameAndariel   = "Andariel"
)

// Monster class ids (monstats hcIdx). V per the DATA2 work: Andariel 156 (docs/data.md), Blood Raven 267, Smith 402.
// The Countess is a super unique: resolve her UniqueNo with db.SuperUniqueByName("The Countess") and call WithUniqueNo.
const (
	ClassAndariel   = 156
	ClassBloodRaven = 267
	ClassSmith      = 402
)

// Area ids (levels.txt order, R per runs.json; task/world.go has the same ones for 1..40). U until levels.txt is read.
const (
	AreaColdPlains     = 3
	AreaBlackMarsh     = 6
	AreaBurialGrounds  = 17
	AreaForgottenTower = 20
	AreaTowerCellar1   = 21
	AreaTowerCellar5   = 25
	AreaOuterCloister  = 27
	AreaBarracks       = 28
	AreaCatacombs2     = 35
	AreaCatacombs3     = 36
	AreaCatacombs4     = 37
)

// Blood Raven's monster preset id (Burial Grounds): VERIFIED LIVE (i4, 2026-10-28) "805@5568,4954", exactly where she stood, in three
// different games of one character. It is NOT her monstats class (267); Kolbot uses 805 too (S).
const presetBloodRaven = 805

// Object class of the Countess' chest (Kolbot SuperChest 580, S; U for 1.14d). The chest is never opened.
const presetCountessChest = 580

// The Countess' monster preset on Tower Cellar 5 (kind 1): VERIFIED LIVE (i2, Combi, 2026-10-29) "836@12534,11040" in the
// presets of area 25; she then stood at (12542,11036) (one game). Presets 833@12520,11099 and 834@12603,11043 are other monsters.
const presetCountessSpawn = 836

const (
	presetSmith = 754 // U, see smith()
	objMalus    = 108
)

// Act1 returns fresh copies of the four Act 1 scripts, keyed by boss name. Every number is U until the live
// checks of docs/boss.md ran; Kolbot coordinates are S.
func Act1() map[string]*Script {
	return map[string]*Script{
		NameBloodRaven: bloodRaven(),
		NameCountess:   countess(),
		NameSmith:      smith(),
		NameAndariel:   andariel(),
	}
}

// Lookup returns the script for a boss name.
func Lookup(name string) (*Script, bool) {
	s, ok := Act1()[name]
	return s, ok
}

// WithUniqueNo returns a copy whose Match also recognises the super unique number (from superuniques.txt).
func (s *Script) WithUniqueNo(no int) *Script {
	c := *s
	c.Match.UniqueNo = no
	return &c
}

// Blood Raven: Burial Grounds, reached from the Cold Plains waypoint. She shoots fire arrows and raises skeletons;
// she moves around the graveyard, so there is no preset: wait for her in the monster list, else explore.
func bloodRaven() *Script {
	return &Script{
		Name: NameBloodRaven, Act: 1, WP: AreaColdPlains, Route: []int{AreaBurialGrounds}, Arena: AreaBurialGrounds,
		Match: Match{Class: ClassBloodRaven, UniqueNo: -1, Name: NameBloodRaven},
		// Her spawn is a monster preset (kind 1, id 805, V live) when the map data has one: walk there first, because the client
		// only lists units within about 60 subtiles (V live: 58 listed her, 95 did not), so "wait for her to show" fails
		// whenever the arrival point is further away. Without the preset: the boss list, then Explore.
		Anchor: Anchor{Boss: true, Preset: &PresetRef{Kind: PresetNPC, ID: presetBloodRaven}},
		// No lure (V live, i4 2026-10-28): she stands still at her spawn until aggroed, 58 subtiles from the arrival point, and the
		// client drops her from the unit list once the player walks back, so luring ended every run in "boss vanished".
		Hazards:  []Hazard{NoPotionLowLife(30), Skeletons(3, 14, 10*time.Second), RaisedAdds(6, 14, 12*time.Second)},
		MinLevel: [3]int{6, 40, 60},
		// Tactic (docs/sme/blood-raven-caster.md section 3, U live): thin the ring around the entry first, and when hurt walk
		// back to the entry (the fence gap we came in by, the only chokepoint the script knows without area geometry) and
		// drink; the second retreat is a town portal. MaxRetreats is overridden from the boss.brMaxRetreats knob.
		OuterClear: OuterClear{Radius: 28, Budget: 45 * time.Second}, RetreatPct: 40, MaxRetreats: 2,
		Timeout: 8 * time.Minute, EscapePct: 25, DrinkPct: 45, PostKillWait: 2 * time.Second,
		Loot: LootSpec{Radius: 25, Kinds: []string{"any"}},
	}
}

// The Countess: Black Marsh waypoint, then Forgotten Tower and Tower Cellar 1 to 5 (no waypoints inside, stairs only).
// Level 5 has her chest preset; the stand point depends on which of the two known layouts the chest X shows (S).
// Her chest is not opened. Runes are why this run exists: the loot spec asks for runes and keys first.
func countess() *Script {
	hint := Pt{12578, 11043}
	return &Script{
		Name: NameCountess, Act: 1, WP: AreaBlackMarsh,
		Route: []int{AreaForgottenTower, AreaTowerCellar1, 22, 23, 24, AreaTowerCellar5}, Arena: AreaTowerCellar5,
		Match: Match{UniqueNo: -1, Name: NameCountess},
		Anchor: Anchor{
			Preset: &PresetRef{Kind: PresetNPC, ID: presetCountessSpawn},
			Hint:   &hint,
		},
		Hazards:  []Hazard{NoPotionLowLife(30), RaisedAdds(5, 12, 10*time.Second)},
		MinLevel: [3]int{14, 42, 67}, RouteMinLifePct: 40, MinPotions: 2,
		Timeout: 10 * time.Minute, EscapePct: 25, DrinkPct: 40,
		PostKillWait: 2 * time.Second, PostClear: 20,
		Loot: LootSpec{Radius: 30, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare"}},
	}
}

// The Smith: Barracks, one hop from the Outer Cloister waypoint. The Barracks are crowded: retreat to the arrival
// point when hurt among many. The Horadric Malus is a quest item and is always taken.
func smith() *Script {
	return &Script{
		Name: NameSmith, Act: 1, WP: AreaOuterCloister, Route: []int{AreaBarracks}, Arena: AreaBarracks,
		Match: Match{Class: ClassSmith, UniqueNo: -1, Name: NameSmith},
		// Preset 754 = 734 + The Smith's superuniques hcIdx 20 (S: Kolbot's superunique presets run from 734; V: 754@15119,4389
		// is in the Barracks preset dump of BOSSLIVE2 and no other class/preset row fits). The Malus object (objects.txt 108,
		// V real table) sits in his room (S). Neither is verified as the spawn point (U): Explore stays the fallback.
		Anchor:   Anchor{Preset: &PresetRef{Kind: PresetNPC, ID: presetSmith}, Also: []*PresetRef{{Kind: PresetObject, ID: objMalus}}},
		Hazards:  []Hazard{NoPotionLowLife(30), CrowdedAndHurt(6, 15, 50), RaisedAdds(8, 12, 10*time.Second)},
		MinLevel: [3]int{8, 40, 60}, MinPotions: 1,
		Timeout: 8 * time.Minute, ExploreBudget: 4 * time.Minute, EscapePct: 25, DrinkPct: 45, PostKillWait: 2 * time.Second,
		Loot: LootSpec{Radius: 25, Quest: []string{"Horadric Malus"}, Kinds: []string{"any"}},
	}
}

// Andariel: Catacombs 2 waypoint, Catacombs 3, Catacombs 4. Poison is the danger: antidote when poisoned (and one
// before the first hit when poison resist is under 75), step off poison clouds, keep 10+ subtiles from her by
// alternating sides (Kolbot's dance, S). Side points are relative to where she is first seen, not Kolbot's world X.
func andariel() *Script {
	hint := Pt{22549, 9520}
	return &Script{
		Name: NameAndariel, Act: 1, WP: AreaCatacombs2, Route: []int{AreaCatacombs3, AreaCatacombs4}, Arena: AreaCatacombs4,
		Match:  Match{Class: ClassAndariel, UniqueNo: -1, Name: NameAndariel},
		Anchor: Anchor{Boss: true, Hint: &hint},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-14, 14}, Rel: true},
			{Name: "east", Pos: Pt{14, 14}, Rel: true},
		},
		Kite:     &Kite{Range: 10, Every: 1500 * time.Millisecond, Away: 12},
		Hazards:  []Hazard{NoPotionLowLife(35), PoisonStatus(), PoisonCloud(6, 10), RaisedAdds(5, 12, 10*time.Second)},
		MinLevel: [3]int{env.DefaultAndarielMinLevel, 45, 65}, PreAntidoteBelow: 75, MinPotions: 3,
		Timeout: 12 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50,
		PostKillWait: 2 * time.Second,
		Loot:         LootSpec{Radius: 25, Kinds: []string{"any"}},
	}
}

// Runner is the adapter-facing entry point: construct once with the live World and Actions, then the task layer
// calls KillBoss(ctx, name) from its own Actions.KillBoss.
type Runner struct {
	W       World
	A       Actions
	Opts    Opts
	Scripts map[string]*Script // nil = Act1()
	Last    Result             // result of the latest call, for logs and events
}

// KillBoss runs the named boss script and returns its error unchanged (errors.Is with the Err* values).
func (r *Runner) KillBoss(ctx context.Context, name string) error {
	sc := r.Scripts
	if sc == nil {
		sc = Act1()
	}
	s, ok := sc[name]
	if !ok {
		return fmt.Errorf("boss: no script for %q", name)
	}
	res, err := Run(ctx, s, r.W, r.A, r.Opts)
	r.Last = res
	return err
}
