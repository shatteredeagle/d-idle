package d2dcombat

import (
	"errors"
	"fmt"
	"math"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/delivery"
	"d2lod-bot/internal/input"
)

// The cast verbs on the combat input (card dlv-cast-verbs, docs/design/input-cast-verbs.md). Every cast leaves through
// send: the cast verb when the knob input.planner is on and the connected build lists its capability, the old command
// otherwise; never both for one cast (input.Planner.RunCast).

// ErrNoCorpse is returned for a corpse-target cast with no corpse near the aim: nothing is sent (no cast into the air).
var ErrNoCorpse = errors.New("d2dcombat: no corpse near the aim")

// send runs one cast and counts it. melee counts the cast as a click too (the left-hand swing, as before).
func (i *Input) send(c input.Cast, legacy func() error, melee bool) error {
	_, err := i.Planner.RunCast(c, i.Controller, legacy, input.PathKey, nil)
	if err == nil {
		i.mu.Lock()
		if melee {
			i.stats.Clicks++
		}
		if i.stats.Casts == nil {
			i.stats.Casts = map[int]int{}
		}
		i.stats.Casts[c.Skill]++
		i.mu.Unlock()
	}
	return err
}

// unitNear is the unit id of the monster nearest w within maxCells: living ones, or corpses (dead, with a unit id).
// A failed monster read gives no unit (the cast goes to the point), as CastAt always did.
func (i *Input) unitNear(w input.Point, maxCells float64, corpse bool) (uint32, bool) {
	x, y := (float64(w.X)+0.5)/5, (float64(w.Y)+0.5)/5
	ml, err := i.C.Monsters(i.ctx)
	if err != nil {
		return 0, false
	}
	if !corpse {
		m, ok := d2dctl.NearestAlive(ml.Items, x, y, maxCells)
		return uint32(m.Unit), ok && m.Unit > 0
	}
	var best uint32
	bd := math.MaxFloat64
	for _, m := range ml.Items {
		if !m.Dead() || m.Unit <= 0 {
			continue
		}
		if d := m.DistCells(x, y); d <= maxCells && d < bd {
			best, bd = uint32(m.Unit), d
		}
	}
	return best, best != 0
}

// Deliver casts skill id as its capability row says (data/capabilities/capabilities.json, the delivery use object):
// the hand, the target kind (a living unit near the aim, a corpse near the aim, or the point) and, for repeat and
// channel presses, hold as the press time. Nothing here knows a class or a skill; the row is the data.
func (i *Input) Deliver(id int, use *delivery.Use, at input.Point, hold time.Duration) error {
	if use == nil {
		return fmt.Errorf("d2dcombat: skill %d has no use row", id)
	}
	c := input.Cast{Skill: id, Left: use.Hand == "left", At: at}
	switch use.Target {
	case "unit":
		c.Unit, _ = i.unitNear(at, 1.5, false)
	case "corpse":
		u, ok := i.unitNear(at, 2, true)
		if !ok {
			return ErrNoCorpse
		}
		c.Unit = u
	}
	if use.Press != "tap" {
		c.Hold = hold
	}
	x, y := i.cell(at)
	legacy := func() error { // the old single command; a hold has no older form than one cast
		return i.C.CmdSkill(i.ctx, id, x, y, d2dctl.SkillOpts{Unit: int(c.Unit), Left: c.Left})
	}
	return i.send(c, legacy, c.Left)
}

// Delivers reports whether casts go through Deliver: the knob input.planner is on. Off, the Fighter keeps CastAt,
// Swing and Click exactly as before (internal/combat castdeliver.go).
func (i *Input) Delivers() bool { return i.Planner.On() }

func (i *Input) cell(w input.Point) (x, y float64) {
	return (float64(w.X-i.Origin.X) + 0.5) / 5, (float64(w.Y-i.Origin.Y) + 0.5) / 5
}
