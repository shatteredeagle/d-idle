package bot

import (
	"d2lod-bot/internal/resource"
	"d2lod-bot/internal/task"
)

// OrderGuardWhy says why the hero's own guards close game area id right now ("" = open, or nothing is known yet). It is
// the read-only fact behind the "too hard now" mark of the orders map (internal/watch, docs/research/orders-ux-design.md
// section 5). It reads the latest snapshot only: no game read, no ladder question (the learned t0.area.toohard rule is
// not asked here, so the map shows the base guard and never opens a Thought), no log line.
// Sources: banned after repeated deaths, the Monastery road and the Forgotten Tower guards, the gear gate.
func (r *Runner) OrderGuardWhy(id int) string {
	s := r.latest()
	if s == nil || s.Player == nil || id <= 0 || r.d.Plan.IsTown(id) {
		return ""
	}
	if r.banned(id) {
		return "banned after repeated deaths"
	}
	if w := r.guardWorld(s); w != nil {
		switch {
		case task.RouteGuarded(task.Area(id)):
			if why := task.RouteWhy(*w); why != "" {
				return why
			}
		case task.TowerGuarded(task.Area(id)):
			if why := task.TowerWhy(*w); why != "" {
				return why
			}
		}
	}
	// gearVerdictFor never logs (gearVerdict does).
	if v, ok := r.gearVerdictFor(s, 0); ok && v.Gated && id != r.safeFarmArea(s) {
		return v.Reason + ": only the first field is open"
	}
	if v, ok := r.gearVerdictFor(s, id); ok && v.Gated && r.e.Tune.Resource.GearGateMode != resource.GateHard && id != r.safeFarmArea(s) {
		return "gear risk high here (" + v.Reason + "): only the first field is open"
	}
	return ""
}
