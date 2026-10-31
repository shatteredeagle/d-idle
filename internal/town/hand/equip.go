package hand

import (
	"context"
	"fmt"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// Equip wears the pack item with the base code (or display name) in the weapon slot: take it from the pack, click the
// weapon slot. A weapon already worn there goes to the cursor and is then put in a free pack cell (the click-swap is
// the slot behaviour V live for Unequip, docs/act2.md T0; swapping a worn weapon for a pack one is U live). A
// two-handed weapon next to a shield, and the weapon set on `w`, are not handled (the main set is made active first).
// No-op when the code is already worn there; ErrNoItem when the pack has none.
func (h *Hand) Equip(ctx context.Context, code string) error {
	if _, err := h.EnsureMainSet(ctx); err != nil {
		return err
	}
	s, err := h.OpenInventory(ctx)
	if err != nil {
		return err
	}
	match := func(u d2.Unit) bool {
		b := h.Base(u)
		return b != nil && (b.Code == code || strings.EqualFold(b.Display, code))
	}
	if w := s.Inventory.Equipped(d2.BodyRHand); w != nil && match(*w) {
		return nil
	}
	var it d2.Unit
	found := false
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if match(u) {
			it, found = u, true
			break
		}
	}
	if !found {
		return fmt.Errorf("%w: %s is not in the pack", ErrNoItem, code)
	}
	lay, err := h.Layout(s)
	if err != nil {
		return err
	}
	rect, ok := lay.Slots[SlotName(d2.BodyRHand)]
	if !ok {
		return fmt.Errorf("hand: no slot rectangle for the weapon slot")
	}
	if err := h.Take(ctx, it, lay.Inventory); err != nil {
		return err
	}
	if err := h.click(input.Left, rect.Center()); err != nil {
		return err
	}
	_, ok, err = h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool {
		u := s.Inventory.Equipped(d2.BodyRHand)
		return u != nil && u.ID == it.ID
	})
	if err != nil {
		return err
	}
	if !ok {
		_ = h.ClearCursor(ctx)
		return fmt.Errorf("hand: equipping %s not confirmed in memory", code)
	}
	return h.ClearCursor(ctx) // the displaced weapon, if any
}

// WornWeapon is the base code of the weapon in the active right hand ("" when empty).
func (h *Hand) WornWeapon(ctx context.Context) (string, error) {
	s, err := h.Fresh()
	if err != nil {
		return "", err
	}
	if u := s.Inventory.Equipped(d2.BodyRHand); u != nil {
		if b := h.Base(*u); b != nil {
			return b.Code, nil
		}
	}
	return "", nil
}
