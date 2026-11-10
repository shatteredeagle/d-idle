package bot

import (
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/plandata"
	"d2lod-bot/internal/task"
)

// fillAndariel sets the Andariel level inputs of w: the class row of plandata.BossLevels (recommended and floor, with
// and without a merc), each overridden by its knob when that is set (boss.andarielMinLevel*, 0 = the class row). A class
// without a row is logged once and takes the old single-number defaults, never another class's row.
func (r *Runner) fillAndariel(w *task.World, class d2.Class) {
	b := r.e.Tune.Boss
	w.AndarielMinLevel, w.AndarielMinLevelMerc = b.AndarielMinLevel, b.AndarielMinLevelMerc
	row, err := plandata.BossLevels(plandata.GateAndariel, strings.ToLower(class.String()))
	if err != nil {
		if s := err.Error(); s != r.loop.task.andyRowErr {
			r.loop.task.andyRowErr = s
			r.e.Log.Warn("andariel level row missing, using the default floors", "err", s)
		}
		if w.AndarielMinLevel <= 0 {
			w.AndarielMinLevel = env.DefaultAndarielMinLevel
		}
		if w.AndarielMinLevelMerc <= 0 {
			w.AndarielMinLevelMerc = env.DefaultAndarielMinLevelMerc
		}
		return
	}
	if w.AndarielMinLevel <= 0 {
		w.AndarielMinLevel = row.Floor
	}
	if w.AndarielMinLevelMerc <= 0 {
		w.AndarielMinLevelMerc = row.FloorMerc
	}
	w.AndarielRec, w.AndarielRecMerc = row.Recommended, row.RecommendedMerc
}

// andarielLow is the lowest Andariel floor of the class (the merc one), for the antidote shopping window.
func (r *Runner) andarielLow(class d2.Class) int {
	var w task.World
	r.fillAndariel(&w, class)
	return min(w.AndarielMinLevel, w.AndarielMinLevelMerc)
}
