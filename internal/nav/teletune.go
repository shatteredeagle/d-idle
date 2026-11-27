package nav

import (
	"math"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// TELETUNE and TELETUNE2: when a hop is worth its time and mana (docs/teletune.md, docs/teletune2.md). A hop is
// declined, and the caller walks, when
//  1. it would not shorten the planned walking path by more than the break-even leg (the hop costs more time than
//     the walk it skips), or
//  2. the mana left after it would fall below the reserve kept for fights.
// After a walk stall at this spot the leg rule is off and the reserve drops to the escape reserve (pocket escape).
// Nothing here waits for mana: a hop that mana does not allow is a walk (the regeneration wait of DRILLS22 is gone).

// teleHopTicks is the lockstep ticks one landed hop takes (V live, DRILLS24: 18 on 9 of 9 hops; the 64 of DRILLS22
// was mostly mana waiting, the nominal 33 was a guess).
const teleHopTicks = 18.0

// RunSubPerTick is the running speed (V live, DRILLS: about 0.5625 subtiles per tick).
const RunSubPerTick = 0.5625

// BreakEvenLeg is the walking distance (subtiles) at which walking takes as long as one hop of hopTicks ticks.
func BreakEvenLeg(subPerTick, hopTicks float64) float64 { return subPerTick * hopTicks }

// hopFacts are the inputs of one hop decision. WalkLen is the A* length of the walk to the landing cell in
// subtiles (+Inf when no known walk leads there: a wall or void in between).
type hopFacts struct {
	WalkLen             float64
	Mana, MaxMana, Cost int
	ReservePct, MinLeg  float64
	// Escape: a walk stalled here, so the leg rule is off and EscapePct replaces ReservePct.
	Escape    bool
	EscapePct float64
}

// hopWorth says whether to hop and, when not, why ("leg": too short, "mana": reserve). MaxMana 0 means the stat is
// unknown: only the leg rule applies. WalkLen is the walking distance the hop skips.
func hopWorth(h hopFacts) (bool, string) {
	if h.WalkLen < h.MinLeg && !h.Escape {
		return false, "leg"
	}
	if h.MaxMana > 0 {
		pct := h.ReservePct
		if h.Escape {
			pct = math.Min(h.EscapePct, h.ReservePct)
		}
		reserve := int(math.Ceil(pct / 100 * float64(h.MaxMana)))
		if h.Mana-h.Cost < reserve {
			return false, "mana"
		}
	}
	return true, ""
}

// TeleLegTicks models one leg of walkSub subtiles crossed with hops of hopSub subtiles each, hopTicks per hop, then
// walking the rest. Mana regenerates at regenPerSec and a hop waits when the pool is short (the old behaviour). It
// is the model behind the DRILLS22 table in docs/teletune.md; the code itself uses hopWorth.
func TeleLegTicks(walkSub, hopSub, hopTicks float64, hops int, mana, cost, regenPerSec, framesPerSec float64) float64 {
	t := 0.0
	left := walkSub
	for i := 0; i < hops && left > 0; i++ {
		if mana < cost {
			w := (cost - mana) / regenPerSec * framesPerSec
			t += w
			mana += w * regenPerSec / framesPerSec
		}
		mana -= cost
		t += hopTicks
		mana += hopTicks * regenPerSec / framesPerSec
		left -= hopSub
	}
	if left > 0 {
		t += left / RunSubPerTick
	}
	return t
}

// teleCost is the mana one cast needs: the skill data at the learned level when the bot wired TeleportCost, else
// the TeleportMana field, else the level 1 upper bound.
func (n *Navigator) teleCost(p *d2.Player) int {
	if n.TeleportCost != nil && p != nil {
		if c := n.TeleportCost(p); c > 0 {
			return c
		}
	}
	return n.teleMana()
}

// walkLen is the A* length from the character to c over known walkable cells, +Inf without a known walk.
func (n *Navigator) walkLen(o *obs, c input.Point) float64 {
	f := n.fieldFor(o.lm)
	if f.W == 0 {
		return math.Inf(1)
	}
	sx, sy, ok := f.nearestWalkable(o.pos.X, o.pos.Y, 8)
	if !ok || !f.Walkable(c.X, c.Y) {
		return math.Inf(1)
	}
	f.passMask = fWalk
	cells, partial := n.pl.Plan(f, sx, sy, c.X, c.Y, 0, 1)
	if partial || len(cells) == 0 {
		return math.Inf(1)
	}
	var sum float64
	px, py := f.pt(cells[0])
	for _, i := range cells[1:] {
		x, y := f.pt(i)
		sum += math.Hypot(float64(x-px), float64(y-py))
		px, py = x, y
	}
	return sum
}

// pathHop chooses the landing cell along the planned walking path: the path cell within hop range that skips the
// most path (the farthest along it), so a hop never lands where the walk from there is longer than from here. saved
// is the path length skipped. planned is false when no complete walk to the target is known (an island, a void in
// between): the caller falls back to the greedy cell.
func (n *Navigator) pathHop(o *obs, target input.Point) (cell input.Point, saved float64, found, planned bool) {
	f := n.fieldFor(o.lm)
	if f.W == 0 {
		return
	}
	sx, sy, ok := f.nearestWalkable(o.pos.X, o.pos.Y, 8)
	if !ok {
		return
	}
	tx, ty, ok := f.nearestWalkable(target.X, target.Y, 8)
	if !ok {
		return
	}
	f.passMask = fWalk
	cells, partial := n.pl.Plan(f, sx, sy, tx, ty, 0, 1)
	if partial || len(cells) < 2 {
		return
	}
	planned = true
	r := n.teleRange()
	px, py := f.pt(cells[0])
	cum := 0.0
	for _, i := range cells[1:] {
		x, y := f.pt(i)
		cum += math.Hypot(float64(x-px), float64(y-py))
		px, py = x, y
		c := input.Point{X: x, Y: y}
		if d := dist(c, o.pos); d > r || d < n.teleMinHop() || n.teleBad[[2]int{x, y}] > 0 {
			continue
		}
		if cum > saved {
			cell, saved, found = c, cum, true
		}
	}
	return
}

// stalledHere reports whether a walk stalled lately within stuckRadius of the character (stuckmem.go): the map
// says walkable but the walk does not get through, so a hop is the way out whatever its leg.
func (n *Navigator) stalledHere(o *obs) bool {
	now := n.clk.Now()
	for _, s := range n.spots {
		if s.level == o.level && dist(s.at, o.pos) <= stuckRadius && now.Sub(s.last) < stuckMemory {
			return true
		}
	}
	return false
}

// hopDeclined applies hopWorth to a candidate landing cell (saved: the walking distance the hop skips) and logs a
// decline once per reason change.
func (n *Navigator) hopDeclined(o *obs, saved float64) bool {
	t := n.tune()
	p := o.snap.Player
	esc := n.stalledHere(o)
	ok, why := hopWorth(hopFacts{
		WalkLen: saved, Mana: p.Mana, MaxMana: p.MaxMana, Cost: n.teleCost(p),
		ReservePct: t.TeleReservePct, MinLeg: t.TeleMinLeg, Escape: esc, EscapePct: t.TeleEscapeReservePct,
	})
	if esc && ok && n.teleWhy != "escape" {
		n.teleWhy = "escape"
		n.log.Info("nav teleport after a stall, leg rule off", "pos", o.pos)
	}
	if ok {
		return false
	}
	if n.teleWhy != why {
		n.teleWhy = why
		n.log.Info("nav teleport declined, walking", "why", why, "mana", p.Mana, "max", p.MaxMana)
	}
	return true
}
