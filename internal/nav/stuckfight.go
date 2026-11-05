package nav

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// stuckFight is the low-life rule of the stuck ladder (Gate 2 g2c seed 13: the ladder side-stepped through a boss pack
// from 65 % life to death). From rung 2 on, with hostiles inside the fight radius and life below
// nav.stuckFightLifePct, the walk step ends and the fight runs (the Yield hook, a bounded clear); once the fights of
// this walk are used up, or there is no Yield hook, the town portal (Escape) is used, once per walk. done reports that
// the loop must start over (replan) instead of climbing the ladder.
func (n *Navigator) stuckFight(ctx context.Context, o *obs, w *walkState) (done bool, err error) {
	pct := n.tune().StuckFightLifePct
	if pct <= 0 || w.rung < 2 || o.snap.Player == nil {
		return false, nil
	}
	life := o.snap.Player.LifePct()
	if float64(life) >= pct {
		return false, nil
	}
	near := n.hostilesWithin(o, float64(n.e.Tune.Combat.FightRadius))
	if len(near) == 0 {
		return false, nil
	}
	if n.Yield != nil && w.fights < maxYields {
		w.fights++
		n.log.Info("nav: stuck ladder -> fight", "rung", w.rung, "life", life, "hostiles", len(near), "pos", o.pos)
		if err := n.Yield(ctx, near); err != nil {
			return false, err
		}
		n.invalidate()
		w.have, w.clicked, w.rung = false, false, 0
		w.anchorPos, w.anchorTime = o.pos, n.clk.Now()
		return true, nil
	}
	if n.Escape == nil || w.escTried {
		return false, nil
	}
	n.log.Info("nav: stuck ladder -> portal", "rung", w.rung, "life", life, "hostiles", len(near), "pos", o.pos)
	if err := n.walkEscape(ctx, o, w); err != nil {
		return false, err
	}
	return false, nil
}

// hostilesWithin lists living non-NPC monsters within r subtiles of the character (friends are filtered by Yield).
func (n *Navigator) hostilesWithin(o *obs, r float64) []d2.Unit {
	var out []d2.Unit
	for i := range o.snap.Monsters {
		u := &o.snap.Monsters[i]
		if !u.Alive() || u.Type != d2.TypeMonster || n.isTownNPC(u) {
			continue
		}
		if dist(input.Point{X: u.X, Y: u.Y}, o.pos) <= r {
			out = append(out, *u)
		}
	}
	return out
}

// crossNote remembers a walk that ended in another level so the walk that comes straight back can say why.
type crossNote struct {
	from, to int
	at       time.Time
	reason   string
}

// crossReason names what the walk was doing when the level changed: a recovery rung (sidestep, wide-sidestep, ...)
// or the plan trigger (first, goal, map, monsters, deviation, partial) of the plan it was following.
func (w *walkState) crossReason() string {
	if w.method != "" {
		return w.method
	}
	if w.why != "" {
		return "replan-" + w.why
	}
	return "walk"
}

const borderBackWindow = 2 * time.Second

// noteBorderBack logs a walk that starts in the level the previous walk just left (Gate -> Tamoe -> Gate within 0.3 s).
func (n *Navigator) noteBorderBack(level int) {
	c := n.lastCross
	if c.at.IsZero() || level != c.from || n.clk.Now().Sub(c.at) > borderBackWindow {
		return
	}
	n.log.Info("nav: area border crossed back", "area", c.from, "from", c.to, "reason", c.reason,
		"after", n.clk.Now().Sub(c.at).Round(10*time.Millisecond))
	n.lastCross = crossNote{}
}
