package d2dinput

import (
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/input"
)

// The cast verbs as input.Caster: "cmd skill <id> x y [unit] [left]", select and cast fused (V with a unit id on the
// right hand; U for the left hand without a unit and for every ground skill, docs/design/input-cast-verbs.md). There is
// no hold verb on any fork build (U, none in the caps list), so Controller is no input.HoldCaster and a hold request
// falls back to the caller's legacy path.

// CastUnit casts skill at unit (a living monster or a corpse).
func (c *Controller) CastUnit(skill int, left bool, at input.Point, unit uint32) error {
	x, y := c.cell(at)
	return c.C.CmdSkill(c.ctx(), skill, x, y, d2dctl.SkillOpts{Unit: int(unit), Left: left})
}

// CastPoint casts skill at the ground point at.
func (c *Controller) CastPoint(skill int, left bool, at input.Point) error {
	x, y := c.cell(at)
	return c.C.CmdSkill(c.ctx(), skill, x, y, d2dctl.SkillOpts{Left: left})
}

var _ input.Caster = (*Controller)(nil)
