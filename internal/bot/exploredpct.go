package bot

// ExploredPctMin and ExploredPctMax bound the knob bot.exploredPct (docs/exploredpct.md).
const (
	ExploredPctMin = 30
	ExploredPctMax = 100
)

// ExploredThreshold is the knob bot.exploredPct clamped to 30..100; an unset (0) or too-high value means 100, the
// old "every coverage point visited" behaviour. The same for every class.
func ExploredThreshold(pct int) int {
	switch {
	case pct <= 0 || pct > ExploredPctMax:
		return ExploredPctMax
	case pct < ExploredPctMin:
		return ExploredPctMin
	}
	return pct
}

// ExploredReached reports whether visited of total coverage points meet the threshold (total 0: never).
func ExploredReached(visited, total, pct int) bool {
	return total > 0 && visited*100 >= ExploredThreshold(pct)*total
}

// ExploredPercent is the visited share rounded down to a whole percent (0 when total is 0).
func ExploredPercent(visited, total int) int {
	if total <= 0 {
		return 0
	}
	return min(100, visited*100/total)
}

// Explored is the roam coverage of the current zone for the watch screen and the minute line: visited and total
// points, the percent, and the threshold at which a clear run moves on. ok is false when no roam coverage exists
// (town, a backend that does not roam).
func (r *Runner) Explored() (visited, total, pct, threshold int, ok bool) {
	c, has := r.d.Mover.(interface{ Coverage() (int, int) })
	if !has {
		return 0, 0, 0, 0, false
	}
	visited, total = c.Coverage()
	if total <= 0 {
		return 0, 0, 0, 0, false
	}
	return visited, total, ExploredPercent(visited, total), ExploredThreshold(r.e.Tune.Bot.ExploredPct), true
}

// exploredEnough is the early stop of a clear roam: true once the zone's coverage reaches bot.exploredPct. Quest and
// boss runs keep roaming (keepRoaming), and the default 100 percent leaves the roam to end by itself.
func (r *Runner) exploredEnough(area int) bool {
	th := ExploredThreshold(r.e.Tune.Bot.ExploredPct)
	if th >= ExploredPctMax || r.loop.keepRoaming(area) {
		return false
	}
	v, m, _, _, ok := r.Explored()
	if !ok || !ExploredReached(v, m, th) {
		return false
	}
	r.e.Log.Info("explored enough", "area", area, "name", r.areaName(area), "visited", v, "points", m, "pct", ExploredPercent(v, m), "threshold", th)
	return true
}
