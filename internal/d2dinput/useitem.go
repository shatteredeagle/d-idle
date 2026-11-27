package d2dinput

import "d2lod-bot/internal/input"

// UseItem sends the use-item verb for carried item id (input.ItemUser): the message the pack right-click sends, no
// panel and no pixel. Callers ask the planner for input.use-item first and verify the effect.
func (c *Controller) UseItem(id uint32) error {
	return c.C.CmdUse(c.ctx(), int(id))
}

var _ input.ItemUser = (*Controller)(nil)
