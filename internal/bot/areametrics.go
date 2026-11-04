package bot

import (
	"time"

	"d2lod-bot/internal/d2"
)

// Per-area metrics (change areametrics): logging only, nothing here feeds a decision.
// goldPickup is the positive gold change seen between two snapshots in the same non-town area (S: a shrine or
// a refund inside the field would count too; a sale in town never does).
// xp is the sum of positive experience changes in the area (a restart that drops memory exp does not subtract).
type areaAcc struct {
	id           int
	xp, gold     int64
	killsAtEntry int
	enteredAt    time.Time
	prevGold     int
	prevExp      uint32
	known        bool
}

// noteAreaMetrics runs once per snapshot before the area-change block and writes the summary line of the area
// being left. Caller holds obMu.
func (r *Runner) noteAreaMetrics(p *d2.Player, now time.Time) {
	o := &r.ob
	a := &o.am
	if !a.known || a.id != p.Area {
		if a.known {
			mins := int(now.Sub(a.enteredAt).Round(time.Minute) / time.Minute)
			r.e.Log.Info("area summary", "area", a.id, "minutes", mins, "xp", a.xp, "kills", o.kills-a.killsAtEntry, "goldPickup", a.gold)
		}
		*a = areaAcc{id: p.Area, killsAtEntry: o.kills, enteredAt: now, prevGold: p.Gold, prevExp: p.Exp, known: true}
		return
	}
	if p.Exp > a.prevExp {
		a.xp += int64(p.Exp - a.prevExp)
	}
	if p.Gold > a.prevGold && !r.d.Plan.IsTown(p.Area) {
		d := int64(p.Gold - a.prevGold)
		a.gold += d
		o.goldPickupMin += d
	}
	a.prevGold, a.prevExp = p.Gold, p.Exp
}
