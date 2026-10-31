package boss

import (
	"context"
	"fmt"
	"time"
)

// Normal Act 2 boss scripts. Tags: V = read from the real game tables on 2026-10-28 (monstats, superuniques, levels,
// objects through internal/data; quest/act2_real_test.go re-checks the shared ids), S = Kolbot / Arreat, U = not
// run against the live game. Every number that shapes a fight (stand points, hazard radii, floors) is U.

const (
	NameRadament = "Radament"
	NameSummoner = "The Summoner"
	NameDuriel   = "Duriel"
)

// Monster classes (V monstats hcIdx): Duriel 211, The Summoner 250, Radament 229 (super unique no. 10, V).
const (
	ClassRadament = 229
	ClassSummoner = 250
	ClassDuriel   = 211

	uniqueNoRadament = 10
)

// Act 2 area ids (V levels.txt).
const (
	AreaSewers2     = 48
	AreaSewers3     = 49
	AreaCanyon      = 46
	AreaTalTomb1    = 66
	AreaTalTomb7    = 72
	AreaDurielsLair = 73
	AreaArcane      = 74
)

// presetJournal is the Summoner's journal object ("Tome", class 357: V it exists in objects.txt, S that it is the
// journal next to the Summoner). The stand point is the preset shifted by (-3,-3) (prior-art 02, S).
const presetJournal = 357

// Act2 returns fresh copies of the Normal Act 2 scripts keyed by boss name.
func Act2() map[string]*Script {
	return map[string]*Script{
		NameRadament: radament(),
		NameSummoner: summoner(),
		NameDuriel:   duriel(),
	}
}

// All returns the Act 1 and Act 2 scripts in one map, for Runner.Scripts (Runner defaults to Act1 only).
func All() map[string]*Script {
	m := Act1()
	for k, v := range Act2() {
		m[k] = v
	}
	return m
}

// Radament: Sewers 2 has a waypoint (found by walking from Sewers 1 the first time, nav handles the fallback); he
// stands in Sewers 3 and raises undead. The Book of Skill is a quest item and always taken. The Horadric Scroll chest
// in the same room is opened by the quest handler, not here.
func radament() *Script {
	return &Script{
		Name: NameRadament, Act: 2, WP: AreaSewers2, Route: []int{AreaSewers3}, Arena: AreaSewers3,
		Match:    Match{Class: ClassRadament, UniqueNo: uniqueNoRadament, Name: NameRadament},
		Anchor:   Anchor{Boss: true},
		Hazards:  []Hazard{NoPotionLowLife(30), RaisedAdds(5, 12, 10*time.Second)},
		MinLevel: [3]int{16, 45, 70}, // Normal 16 is his monster level (V monstats), the rest follows the Summoner's spacing (U)
		Timeout:  8 * time.Minute, EscapePct: 25, DrinkPct: 45, PostKillWait: 2 * time.Second,
		Loot: LootSpec{Radius: 25, Quest: []string{"Book of Skill"}, Kinds: []string{"any"}},
	}
}

// The Summoner: the Arcane Sanctuary is a maze of arms over a void; the waypoint is the arrival point and the
// Summoner stands at the end of one arm next to his journal. He casts and teleports and has few hit points in Normal
// (V: 129..172), so the run is mostly about reaching him: Teleport makes it short, walking needs the Explore budget.
// The red portal to the Canyon appears when the journal is touched, which the quest handler does.
func summoner() *Script {
	return &Script{
		Name: NameSummoner, Act: 2, WP: AreaArcane, Arena: AreaArcane,
		Match: Match{Class: ClassSummoner, UniqueNo: -1, Name: NameSummoner},
		Anchor: Anchor{
			Preset:   &PresetRef{Kind: PresetObject, ID: presetJournal, Off: Pt{-3, -3}},
			Teleport: true, // islands over a void: nav.TeleportTo plans the hops (Teleporter), GoTo is the fallback
		},
		Hazards:  []Hazard{NoPotionLowLife(30), RaisedAdds(6, 12, 10*time.Second)},
		MinLevel: [3]int{24, 50, 70}, // catalog.md boss.summoner
		Timeout:  12 * time.Minute, EscapePct: 25, DrinkPct: 45, PostKillWait: 2 * time.Second,
		Loot: LootSpec{Radius: 25, Kinds: []string{"any"}},
	}
}

// Duriel: Duriel's Lair (area 73). The lair is entered through the tomb after the Staff is placed, so the script
// starts inside it; RunWithEntry brings the character in. Teleport into the lair is forbidden by levels.txt (V,
// Teleport column 2), a town portal out is fine. Duriel is melee only, slow-ish (V velocity 10) and carries Holy Freeze
// (S): cold damage and slow while close, so the script steps away when hurt and kites between two relative points.
// Normal stats V: 4757 life, cold resist 50, level 22; floor 24 from runs.json.
func duriel() *Script {
	return &Script{
		Name: NameDuriel, Act: 2, Arena: AreaDurielsLair,
		Match:  Match{Class: ClassDuriel, UniqueNo: -1, Name: NameDuriel},
		Anchor: Anchor{Boss: true},
		Stand: []StandPoint{
			{Name: "west", Pos: Pt{-12, 12}, Rel: true},
			{Name: "east", Pos: Pt{12, 12}, Rel: true},
		},
		Kite:     &Kite{Range: 8, Every: 2 * time.Second, Away: 12},
		Hazards:  []Hazard{NoPotionLowLife(35), ColdAura(7, 12, 55), RaisedAdds(4, 10, 8*time.Second)},
		MinLevel: [3]int{24, 50, 70},
		Timeout:  15 * time.Minute, EscapePct: 30, ChickenPct: 12, DrinkPct: 50, PostKillWait: 3 * time.Second,
		Loot: LootSpec{Radius: 30, Kinds: []string{"rune", "key", "gem", "unique", "set", "rare", "any"}},
	}
}

// ColdAura steps away from the boss while it stands within near subtiles and life is under pct: Duriel's Holy Freeze
// chills and slows in melee range, so a hurt, chilled character leaves the aura instead of trading blows. Cooldown
// stops the script from dancing instead of attacking. U: the aura radius (near) and whether the chill state can be
// read (it cannot yet, so life stands in for it).
func ColdAura(near, away, pct int) Hazard {
	return Hazard{Name: "cold-aura", Cooldown: 4 * time.Second, Do: Response{Kind: RespStepAway, Radius: away},
		Detect: func(c Ctx) bool {
			return c.Boss != nil && c.Boss.Alive() && c.V.LifePct() < pct && c.V.Pos.Dist(c.Boss.Pos) <= near
		}}
}

// TombAreas are the seven Tal Rasha's Tomb levels of the Canyon of the Magi (V levels.txt 66..72); exactly one per
// game holds the orifice (quest/act2.go FindTomb).
func TombAreas() []int {
	var a []int
	for t := AreaTalTomb1; t <= AreaTalTomb7; t++ {
		a = append(a, t)
	}
	return a
}

// RunWithEntry runs the script, first calling enter when the character is not yet in the arena. For Duriel enter is
// the quest layer's tomb leg (find the tomb, place the Staff, use the lair door); the script itself cannot walk
// there because the lair has no exit from the Canyon. Without enter and outside the arena Run falls back to
// Actions.Travel(arena), which only works once a waypoint or portal exists.
func RunWithEntry(ctx context.Context, s *Script, w World, a Actions, o Opts, enter func(context.Context) error) (Result, error) {
	if enter != nil {
		v, err := w.View(ctx)
		if err != nil {
			return Result{Boss: s.Name}, err
		}
		if v.Area != s.Arena {
			if err := enter(ctx); err != nil {
				return Result{Boss: s.Name}, fmt.Errorf("boss: enter %s's arena: %w", s.Name, err)
			}
		}
	}
	return Run(ctx, s, w, a, o)
}
