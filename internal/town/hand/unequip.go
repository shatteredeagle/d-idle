package hand

import (
	"context"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Unequip takes a worn item off: click its slot (the item goes to the cursor), then put it in a free pack cell.
// V live (docs/act2.md P1/T0): a second click on the slot with the item still on the cursor re-equips it, so the put
// goes straight to the pack grid. No-op when the item is not worn. The pack must have room (ErrNoSpace otherwise).
func (h *Hand) Unequip(ctx context.Context, id uint32) error {
	s, err := h.OpenInventory(ctx)
	if err != nil {
		return err
	}
	u, ok := Find(s.Inventory, id)
	if !ok || u.Item == nil || u.Item.Where != d2.ContEquipment {
		return nil
	}
	lay, err := h.Layout(s)
	if err != nil {
		return err
	}
	r, ok := lay.Slots[SlotName(u.Item.Body)]
	if !ok {
		return fmt.Errorf("hand: no slot rectangle for body location %d", u.Item.Body)
	}
	c := r.Center()
	if err := h.E.In.Click(input.Left, c.X, c.Y); err != nil {
		return err
	}
	_, ok, err = h.Wait(ctx, 2*time.Second, func(s *d2.Snapshot) bool { return s.Inventory.Cursor != nil })
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("hand: slot click at %v took nothing off", c)
	}
	return h.ClearCursor(ctx)
}
