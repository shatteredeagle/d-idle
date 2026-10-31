package gear

import (
	"sort"

	"d2lod-bot/internal/d2"
)

// Pure scoring helpers for the leveler's item scorer (internal/leveler/score.go calls these instead of its flat
// constants; docs/items.md section 8). Scores are defence-equivalents like the leveler's.

// Difficulty indexes the resistance penalty.
const (
	Normal = iota
	Nightmare
	Hell
)

// ResistPenalty is the resistance penalty of a difficulty: 0 / -40 / -100 (S, common D2 knowledge; U live on this install).
func ResistPenalty(diff int) int { return d2.Difficulty(clamp(diff, 0, 2)).ResistPenalty() }

// MaxResist is the usual 75 cap; the character shows (raw + penalty) capped at it.
const MaxResist = 75

// overflow is the share of a point above the cap that still counts (curse and sunder buffer).
const overflow = 0.1

// ResistValue is the worth of adding add points to one resistance whose raw total (all gear, no penalty) is have.
// Points that move the effective value toward the cap count in full, points above the cap count overflow.
// weight is the score of one useful point.
func ResistValue(have, add, diff int, weight float64) float64 {
	capRaw := MaxResist - ResistPenalty(diff)
	// Effective resistance below -100 is not meaningful; do not credit the climb out of an absurd hole beyond that.
	useful := min(have+add, capRaw) - min(have, capRaw)
	over := add - useful
	if useful < 0 { // a negative add (a curse on the item) is a straight loss
		return float64(add) * weight
	}
	return float64(useful)*weight + float64(max(over, 0))*weight*overflow
}

// fcrBreakpoints are the 1.14 faster-cast-rate breakpoints per class (S: widely published tables, U on this install).
var fcrBreakpoints = map[string][]int{
	"sor": {9, 20, 37, 63, 105, 200},
	"nec": {9, 18, 30, 48, 75, 125},
	"pal": {9, 18, 30, 48, 75, 125},
	"dru": {4, 10, 19, 30, 46, 68, 99, 163},
	"ama": {7, 14, 22, 32, 48, 68, 99, 152},
	"bar": {9, 20, 37, 63, 105, 200},
	"ass": {8, 16, 27, 42, 65, 102, 174},
}

// FCRBreakpoints returns the breakpoint list for a class (any spelling ClassCode accepts), nil when unknown.
func FCRBreakpoints(class string) []int { return fcrBreakpoints[ClassCode(class)] }

// FCRValue scores add points of faster cast rate on top of have: each breakpoint crossed is worth perBreakpoint,
// the points themselves a small residual so the next breakpoint stays attractive.
func FCRValue(class string, have, add int, perBreakpoint, perPoint float64) float64 {
	bps := FCRBreakpoints(class)
	reached := func(v int) int { return sort.Search(len(bps), func(i int) bool { return bps[i] > v }) }
	crossed := reached(have+add) - reached(have)
	return float64(crossed)*perBreakpoint + float64(add)*perPoint
}

// EtherealRepairs says whether the bot repairs ethereal gear. UNVERIFIED (docs/items.md Q6: does Repair All skip it?);
// default false is the cautious reading. TODO live probe: repair an ethereal item at Charsi.
var EtherealRepairs = false

// EtherealFactor replaces the flat +25% the leveler gave ethereal armour: ethereal armour has +50% defence (S) but
// cannot be repaired, so unless the bot repairs it the net is a small gain, not a bonus.
func EtherealFactor() float64 {
	if EtherealRepairs {
		return 1.25
	}
	return 0.9
}

// runewordScore is a flat defence-equivalent for a finished runeword whose stat list could not be read (name from the
// item's runeword flag, U: the name is not readable from memory yet). Values are tuning guesses (U), ordered by how
// much each helps a leveling bot (docs/items.md section 5).
var runewordScore = map[string]float64{
	"Stealth": 35, "Lore": 30, "Ancient's Pledge": 60, "Rhyme": 40, "Leaf": 50, "Spirit": 90, "Insight": 70,
	"Smoke": 55, "Fortitude": 150, "Enigma": 170, "Heart of the Oak": 160, "Call to Arms": 140, "Grief": 150,
}

// RunewordScore returns the flat score of a named runeword, 0 when unknown.
func RunewordScore(name string) float64 { return runewordScore[name] }

func clamp(v, lo, hi int) int { return max(lo, min(hi, v)) }
