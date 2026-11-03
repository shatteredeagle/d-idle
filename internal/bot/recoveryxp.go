package bot

import (
	"fmt"

	"d2lod-bot/internal/d2"
)

// recoveryXPTier is the farm list of a recovering Normal act 1 character who can pick by XP: the old safe tier plus Dark
// Wood, Black Marsh and Burial Grounds, which pay several times more XP per kill from clvl 10 on. Gold per pile grows with the
// monster level (3.5 ilvl - 0.5 coins, treasure-trace B2) but a plain kill drops a pile only 13 percent of the time,
// so XP still decides the pick; docs/botsupply.md has the supply numbers.
// Burial Grounds (17) is NOT a candidate: Blood Raven lives there and her fight is gated on a full kit (BRKILL); a broke character must not wander in.
var recoveryXPTier = map[int]bool{2: true, 3: true, 4: true, 5: true, 6: true, 9: true, 13: true}

// normalMLvl is the Normal monster level of the act 1 farm areas for a run without install data (offline tests, a missing
// MPQ cache). V: Blood Moor 1, Cold Plains 2, Stony Field 4, Dark Wood 5, Black Marsh 6 (plandata progression-normal.json);
// Levels.txt MonLvl1Ex (D, docs/research/monster-stats-trace.md B2): Den 1, Cave 1 2, Cave 2 2, Burial Grounds 3
// (the earlier wiki recollection of Den 2 and Cave 2 3 was wrong).
var normalMLvl = map[int]int{2: 1, 3: 2, 4: 4, 5: 5, 6: 6, 8: 1, 9: 2, 13: 2, 17: 3}

// recoveryMLvl is the area's monster level from the install data, else the fallback table (Normal only).
func (r *Runner) recoveryMLvl(area int, d d2.Difficulty) int {
	if ml := r.areaMLvl(area, d); ml > 0 {
		return ml
	}
	if d == d2.Normal {
		return normalMLvl[area]
	}
	return 0
}

// xpLevelPct is the share of the XP a monster of level mlvl gives a character of level clvl, in percent (V, docs/sme/mechanics-progression.md
// section 2.1/2.2, B15).
func xpLevelPct(clvl, mlvl int) float64 {
	d := mlvl - clvl
	if clvl >= 25 && d > 0 {
		return 100 * float64(clvl) / float64(mlvl)
	}
	switch {
	case d >= 10:
		return 1.95
	case d >= 6:
		return [...]float64{87.9, 68.0, 35.9, 14.8}[d-6]
	case d >= -5:
		return 100
	case d >= -9:
		return [...]float64{80.9, 62.1, 43.0, 23.8}[-d-6]
	}
	return 5.08
}

// areaXPPerKill estimates the XP of one kill in an area of monster level mlvl with no measured table row: a base of 10
// per monster level times the level-difference share. S: the base is a fit, not data (0.45x to 1.15x of the real
// means, docs/research/monster-stats-trace.md B1); areaXP prefers the traced table.
func areaXPPerKill(clvl, mlvl int) float64 {
	return float64(mlvl) * 10 * xpLevelPct(clvl, mlvl) / 100
}

// areaXPRow is a traced Normal area: the mean experience of a plain monster of its real population (before the level
// gap) and the mean unit level. V run, monster-stats-trace.md section 4 (12 seeds of the real room populations, bosses
// and their packs excluded; the mean is per unit, not per kill the bot makes).
type areaXPRow struct {
	exp  float64
	mlvl int // round(mean level)
}

var areaXPTable = map[int]areaXPRow{
	2: {22.0, 1}, 3: {25.6, 2}, 4: {37.3, 3}, 5: {47.0, 5}, 6: {52.0, 6}, 8: {26.1, 1}, 9: {35.2, 2}, 13: {27.2, 2},
	17: {40.2, 2}, 27: {76.4, 8}, 35: {97.2, 11}, 41: {159.3, 14}, 44: {218.6, 17}, 47: {140.0, 13},
}

// areaXP is the XP of one plain kill of a Normal area for a character of level clvl: the traced mean times the
// level-gap share at the area's mean unit level; an area without a row (or a harder difficulty) falls back to
// areaXPPerKill on the area's monster level.
func areaXP(clvl, area, mlvl int, d d2.Difficulty) float64 {
	if row, ok := areaXPTable[area]; ok && d == d2.Normal {
		return row.exp * xpLevelPct(clvl, row.mlvl) / 100
	}
	return areaXPPerKill(clvl, mlvl)
}

// stepXP is the XP estimate of one kill in a plan step for a character of level clvl (PickOptions.XP). An area without
// a monster level in the install data or the fallback table is rated as if its monsters were 3 levels under the entry
// level (U: only the order between areas matters).
func (r *Runner) stepXP(clvl int, d d2.Difficulty) func(Step) float64 {
	return func(st Step) float64 {
		ml := r.recoveryMLvl(st.Area, d)
		if ml == 0 {
			ml = max(1, st.EnterLvl-3)
		}
		return areaXP(clvl, st.Area, ml, d)
	}
}

// recoveryXPArea picks the safe area with the best XP per kill for a recovering character, among the plan steps the
// level picker would allow (entry level, acts, exclusions) whose monster level is at most clvl+recoveryMaxAreaAbove.
// ties go to the lower monster level. ok is false when nothing qualifies or the data has no area levels.
func (r *Runner) recoveryXPArea(s *d2.Snapshot, ex map[int]bool) (int, bool) {
	diff := difficultyName(s.Player.Difficulty)
	clvl := s.Player.Level
	t := r.e.Tune.Bot
	acts := r.allowedActs(s)
	melee, full := MeleeTank(s.Player.Class), r.fullKit(s)
	early := r.e.Tune.Resource.RecoveryXPEarly // XPGAP knob: enter this many levels before the plan says
	best, bestML, bestXP := 0, 0, -1.0
	for _, st := range r.d.Plan.Steps[diff] {
		if !recoveryXPTier[st.Area] || ex[st.Area] || r.d.Plan.IsTown(st.Area) {
			continue
		}
		if (acts != nil && !acts[r.d.Plan.Act(st.Area)]) || (st.KitLvl-early > clvl && !full) || st.enter(melee)+t.LevelMargin-early > clvl {
			continue
		}
		ml := r.recoveryMLvl(st.Area, s.Player.Difficulty)
		if ml == 0 || ml > clvl+r.e.Tune.Resource.RecoveryMaxAreaAbove {
			continue
		}
		if xp := areaXP(clvl, st.Area, ml, s.Player.Difficulty); xp > bestXP || (xp == bestXP && ml < bestML) {
			best, bestML, bestXP = st.Area, ml, xp
		}
	}
	if best == 0 {
		return 0, false
	}
	if r.loop.broke.areaNote != best {
		r.loop.broke.areaNote = best
		r.e.Log.Info(fmt.Sprintf("recovery: farm area %s (xp/kill est %.0f at clvl %d)", r.areaName(best), bestXP, clvl), "area", best, "gold", s.Player.Gold)
	}
	return best, true
}
