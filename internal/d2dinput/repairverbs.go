package d2dinput

import "d2lod-bot/internal/input"

// Repair sends the repair verb (input.Repairer); id input.RepairAllItems repairs everything.
func (c *Controller) Repair(id int) error { return c.C.CmdRepair(c.ctx(), id) }

// MenuChoose picks the open dialog's line containing text (input.MenuChooser).
func (c *Controller) MenuChoose(text string) error { return c.C.MenuChoose(c.ctx(), text) }

var (
	_ input.Repairer    = (*Controller)(nil)
	_ input.MenuChooser = (*Controller)(nil)
)
