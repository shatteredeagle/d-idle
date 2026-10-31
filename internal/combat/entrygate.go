package combat

import (
	"fmt"

	"d2lod-bot/internal/d2"
)

// Recommended minimum character level to enter an act, per difficulty (docs/difficulty.md section 2.2, "R": derived
// from the install's levels.tsv monster levels, V, minus the usual margin; the planner may override with tests).
// Index 0 is act 1. Normal has no gate. Hell act 1 is the wilderness; caves and crypts need hellDungeonFloor.
var entryFloors = map[d2.Difficulty][5]int{
	d2.Nightmare: {36, 44, 50, 56, 58},
	d2.Hell:      {66, 72, 75, 80, 82},
}

const hellDungeonFloor = 75 // Hell act 1 caves/crypts (monster level 69-85)

// Monster-level margins: enter an area at clvl >= area monster level - margin (docs/difficulty.md 2.2, U).
var entryMargin = map[d2.Difficulty]int{d2.Normal: 2, d2.Nightmare: 5, d2.Hell: 8}

// EntryDecision is the result of EntryGate.
type EntryDecision struct {
	OK    bool
	Floor int // the level that was required
	Why   string
}

// EntryGate says whether a character of level clvl may enter an area of act (1-5) in difficulty diff. areaMonLvl is
// the area's monster level (levels.tsv, 0 when unknown) and dungeon marks Hell caves/crypts. Normal is never refused.
// Refusal means: stay in the easier difficulty or area and level up.
func EntryGate(diff d2.Difficulty, act, clvl, areaMonLvl int, dungeon bool) EntryDecision {
	floors, gated := entryFloors[diff]
	if !gated {
		return EntryDecision{OK: true}
	}
	if act < 1 || act > 5 {
		return EntryDecision{OK: false, Why: fmt.Sprintf("unknown act %d", act)}
	}
	floor := floors[act-1]
	if diff == d2.Hell && act == 1 && dungeon {
		floor = max(floor, hellDungeonFloor)
	}
	if areaMonLvl > 0 {
		floor = max(floor, areaMonLvl-entryMargin[diff])
	}
	if clvl < floor {
		return EntryDecision{Floor: floor, Why: fmt.Sprintf("clvl %d below %d for act %d (monster level %d)", clvl, floor, act, areaMonLvl)}
	}
	return EntryDecision{OK: true, Floor: floor}
}
