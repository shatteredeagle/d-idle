package hand

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// DropCursor puts the held item on the ground beside the character: a left click on the playfield with an item on the
// cursor drops it (U live: D2 semantics, not yet seen in this bot). It is the last resort for an item no pack cell
// takes (ClearCursor returned ErrNoSpace); a cursor item persists through a game re-entry (U), so leaving it held
// blocks every later pickup, shop and waypoint use. No-op when the cursor is empty. Panels are closed first so the
// click lands on the playfield.
func (h *Hand) DropCursor(ctx context.Context) error {
	s, err := h.Fresh()
	if err != nil {
		return err
	}
	if s.Inventory.Cursor == nil {
		return nil
	}
	if err := h.E.In.CloseAll(); err != nil {
		return err
	}
	for try := 0; try < 2; try++ {
		if err := h.E.In.ClickWorld(input.Point{X: s.Player.X + 2, Y: s.Player.Y + 2}); err != nil {
			return err
		}
		ns, ok, err := h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool { return s.Inventory.Cursor == nil })
		if err != nil {
			return err
		}
		if ok {
			return nil
		}
		s = ns
	}
	return fmt.Errorf("%w: item still on the cursor after a ground click", ErrCursor)
}
