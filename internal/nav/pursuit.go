package nav

import (
	"context"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// pursuitCooldown keeps a wander from handing over to the fight again at once after a hand-over that left the monster
// standing (unreachable, ignored by the fighter): the next hand-over waits this long.
const pursuitCooldown = 3 * time.Second

// pursuers lists the living non-NPC monsters within r subtiles of the character.
func (n *Navigator) pursuers(o *obs, r float64) []d2.Unit {
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

// pursuit is the turn-and-fight rule of the wanders (explore, roam, exit walks; walkOpts.pursuit). A monster that
// stays within nav.pursuitRadius for nav.pursuitPolls observations in a row is following the hero: the walk hands over
// to the fight (the Yield hook filters friends and fights with earlier ignore decisions lifted) at life at or above
// nav.pursuitMinLifePct. Before this a wander only yielded when it had stalled, so a hero at full life walked on with
// followers behind it (d2play 2026-11-10, docs/pursuers.md). done reports that the walk must plan again.
func (n *Navigator) pursuit(ctx context.Context, o *obs, w *walkState) (done bool, err error) {
	t := n.tune()
	if t.PursuitRadius <= 0 || n.Yield == nil || o.snap.Player == nil || o.snap.Player.InTown() {
		w.followed = 0
		return false, nil
	}
	if float64(o.snap.Player.LifePct()) < t.PursuitMinLifePct {
		w.followed = 0
		return false, nil
	}
	near := n.pursuers(o, t.PursuitRadius)
	if len(near) == 0 {
		w.followed = 0
		return false, nil
	}
	w.followed++
	if float64(w.followed) < t.PursuitPolls || w.yields >= maxYields || n.clk.Now().Before(n.pursuitAt) {
		return false, nil
	}
	w.yields++
	w.followed = 0
	n.log.Info("nav: followed, turning to fight", "pos", o.pos, "near", len(near), "life", o.snap.Player.LifePct())
	if err := n.Yield(ctx, near); err != nil {
		return false, err
	}
	n.pursuitAt = n.clk.Now().Add(pursuitCooldown)
	n.invalidate()
	w.have, w.clicked, w.rung = false, false, 0
	w.anchorPos, w.anchorTime = o.pos, n.clk.Now()
	return true, nil
}
