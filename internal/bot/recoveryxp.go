package bot

import (
	"fmt"

	"d2lod-bot/internal/d2"
)

// recoveryXPTier is the farm list of a recovering Normal act 1 character who can pick by XP: the old safe tier plus Dark
// Wood, Black Marsh and Burial Grounds, which pay several times more XP per kill from clvl 10 on. Gold per kill is area
// independent (V-d2d, docs/d2d-fidelity.md), so the XP is the only thing that differs between the areas.
// Burial Grounds (17) is NOT a candidate: Blood Raven lives there and her fight is gated on a full kit (BRKILL); a broke character must not wander in.
var recoveryXPTier = map[int]bool{2: true, 3: true, 4: true, 5: true, 6: true, 9: true, 13: true}

// normalMLvl is the Normal monster level of the act 1 farm areas for a run without install data (offline tests, a missing
// MPQ cache). V: Blood Moor 1, Cold Plains 2, Stony Field 4, Dark Wood 5, Black Marsh 6 (plandata progression-normal.json);
// S (wiki recollection): Den 2, Cave 1 2, Cave 2 3, Burial Grounds 3.
var normalMLvl = map[int]int{2: 1, 3: 2, 4: 4, 5: 5, 6: 6, 8: 2, 9: 2, 13: 3, 17: 3}

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

// areaXPPerKill estimates the XP of one kill in an area of monster level mlvl: a base of 10 per monster level times the
// level-difference share. S: the base is a fit, not data; it reproduces the d2d means at clvl 12 (Cold Plains about 1,
// Stony Field 17 against 15, Dark Wood 31 against 35, V-d2d) and only the ranking matters here.
func areaXPPerKill(clvl, mlvl int) float64 {
	return float64(mlvl) * 10 * xpLevelPct(clvl, mlvl) / 100
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
	best, bestML, bestXP := 0, 0, -1.0
	for _, st := range r.d.Plan.Steps[diff] {
		if !recoveryXPTier[st.Area] || ex[st.Area] || r.d.Plan.IsTown(st.Area) {
			continue
		}
		if (acts != nil && !acts[r.d.Plan.Act(st.Area)]) || (st.KitLvl > clvl && !full) || st.enter(melee)+t.LevelMargin > clvl {
			continue
		}
		ml := r.recoveryMLvl(st.Area, s.Player.Difficulty)
		if ml == 0 || ml > clvl+r.e.Tune.Resource.RecoveryMaxAreaAbove {
			continue
		}
		if xp := areaXPPerKill(clvl, ml); xp > bestXP || (xp == bestXP && ml < bestML) {
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
