package d2dctl

import (
	"context"
	"fmt"
	"math"
)

// Combat helpers on top of the typed cmd verbs. Everything here is a thin composition; the channel facts are in
// docs/d2d-combat.md (V = seen on the private fork, U = not).

// Alive reports a monster that can still be fought: not dead, not dying (mode DT) and with life left.
func (m Monster) Alive() bool { return !m.Dead() && m.Mode != "DT" }

// DistCells is the Euclidean distance in cells from the monster to the cell position (x, y).
func (m Monster) DistCells(x, y float64) float64 { return math.Hypot(m.CellX-x, m.CellY-y) }

// NearestAlive returns the living monster closest to the cell position (x, y) within maxCells (maxCells <= 0: any).
func NearestAlive(ms []Monster, x, y, maxCells float64) (Monster, bool) {
	var best Monster
	bd, found := math.MaxFloat64, false
	for _, m := range ms {
		if !m.Alive() {
			continue
		}
		if d := m.DistCells(x, y); d < bd && (maxCells <= 0 || d <= maxCells) {
			best, bd, found = m, d, true
		}
	}
	return best, found
}

// CastUnit casts skill id at a monster by unit id (cmd skill id x y unit). Unit ids are d2d's, V as the target of "cmd skill".
func (c *Client) CastUnit(ctx context.Context, skill int, m Monster) error {
	if m.Unit <= 0 {
		return fmt.Errorf("d2dctl: monster %q has no unit id", m.Name)
	}
	return c.CmdSkill(ctx, skill, m.CellX, m.CellY, SkillOpts{Unit: m.Unit})
}

// CastCell casts skill id at a ground cell position with no unit.
func (c *Client) CastCell(ctx context.Context, skill int, x, y float64) error {
	return c.CmdSkill(ctx, skill, x, y, SkillOpts{})
}

// Melee is the basic attack (skill 0, left button) on a monster. U: whether skill 0 is accepted as the normal attack
// (the d2dinput Interact path assumes it, docs/d2d-backend.md).
func (c *Client) Melee(ctx context.Context, m Monster) error {
	if m.Unit <= 0 {
		return fmt.Errorf("d2dctl: monster %q has no unit id", m.Name)
	}
	return c.CmdSkill(ctx, 0, m.CellX, m.CellY, SkillOpts{Unit: m.Unit, Left: true})
}

// Drink drinks the potion of belt column slot (0 to 3, the key 1..4 of the original).
func (c *Client) Drink(ctx context.Context, slot int) error {
	if slot < 0 || slot > 3 {
		return fmt.Errorf("d2dctl: belt column %d out of range 0..3", slot)
	}
	return c.CmdBelt(ctx, slot, false)
}

// SetSkill puts a skill on the right (default) or left button through the debug picker (V verb, "err not usable"
// when the character cannot use it). Unlike CmdSelect it does not depend on the skill list UI.
func (c *Client) SetSkill(ctx context.Context, skill int, left bool) error {
	side := "right"
	if left {
		side = "left"
	}
	return c.dbg(ctx, "skill %s %d", side, skill)
}
