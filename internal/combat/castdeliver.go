package combat

import (
	"sync"
	"time"

	capdata "d2lod-bot/data/capabilities"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/delivery"
	"d2lod-bot/internal/input"
)

// deliverer is a backend that sends each cast as one cast request (d2dcombat.Input.Deliver, card casthold,
// docs/design/input-cast-verbs.md): the cast verb when the connected build lists its capability, else the backend's
// old single command, never both. Delivers is the knob input.planner: off, cast keeps its old calls unchanged.
type deliverer interface {
	Delivers() bool
	Deliver(id int, use *delivery.Use, at input.Point, hold time.Duration) error
}

// capRows is every row of the capability table by skill name (the table is validated by its own tests; a load error
// leaves no rows, so every cast keeps its old calls).
var capRows = sync.OnceValue(func() map[string]delivery.Entry {
	t, err := delivery.Load(capdata.JSON)
	if err != nil {
		return nil
	}
	return t.All()
})

// useRow is the use object of the capability row named name (hand, target, press); nil when there is no castable row.
func useRow(name string) *delivery.Use {
	if e, ok := capRows()[name]; ok && e.Delivery != nil {
		return e.Delivery.Use
	}
	return nil
}

// deliverCast sends sp through the backend's Deliver when the backend delivers now and the skill has a row. The hand,
// target and press come from the row; a repeat or channel press holds for the profile's hold time. sent false leaves
// the cast to the old calls (and nothing was sent).
func (f *Fighter) deliverCast(p *d2.Player, sp SkillSpec, at input.Point) (sent bool, err error) {
	d, ok := f.e.In.(deliverer)
	if !ok || !d.Delivers() {
		return false, nil
	}
	use := useRow(sp.Name)
	if use == nil {
		return false, nil
	}
	var hold time.Duration
	if use.Press != "tap" {
		hold = f.pressSlice(p, sp)
	}
	return true, d.Deliver(sp.ID, use, at, hold)
}
