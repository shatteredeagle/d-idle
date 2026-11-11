package leveler

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town/hand"
)

// Player-requested trash (docs/inventory-actions.md): the watch screen queues "drop this backpack item on the ground";
// the bot calls PlayerTrash at an open opportunity. Click path: open the pack, pick the item up (hand.Take), put it on
// the ground beside the hero (hand.DropCursor: panels closed, left click on a free playfield cell), then prove from
// memory that the item left the pack and the cursor is empty. Only real input; nothing writes game memory.

// ErrNoRoomToDrop is returned when the ground clicks did not drop the item and it went back into the pack.
var ErrNoRoomToDrop = errors.New("the item could not be dropped and is back in the pack")

// CheckPlayerTrash says why the pack item id cannot be trashed now; nil means it can. It returns the item's name.
func (l *Leveler) CheckPlayerTrash(s *d2.Snapshot, id uint32) (name string, err error) {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return "", ErrNoPlayer
	}
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		if u.ID == id && u.Item != nil {
			name = "item"
			if b := l.newHand().Base(u); b != nil && b.Name != "" {
				name = b.Name
			}
			return name, nil
		}
	}
	return "", ErrNoItem
}

// PlayerTrash drops pack item id on the ground. The caller picks the moment (town, or the field with nothing
// hostile near); a fight nearby returns ErrBusy. The item is either on the ground afterwards (nil) or back in the pack
// (an error), never left on the cursor.
func (l *Leveler) PlayerTrash(ctx context.Context, id uint32) error {
	h := l.newHand()
	if !l.o.Dry {
		if _, err := h.EnsureMainSet(ctx); err != nil {
			return err
		}
	}
	s, err := h.Fresh()
	if err != nil {
		return err
	}
	if l.busy(s) {
		return ErrBusy
	}
	if _, err := l.CheckPlayerTrash(s, id); err != nil {
		return err
	}
	if l.o.Dry {
		return nil
	}
	err = l.trash(ctx, h, id)
	if cerr := h.CloseAll(ctx); cerr != nil {
		l.e.Log.Warn("leveler: closing panels after the player's trash failed", "err", cerr)
	}
	if err == nil {
		l.logf("player trash", "item", id)
	}
	return err
}

func (l *Leveler) trash(ctx context.Context, h *hand.Hand, id uint32) error {
	s, err := h.OpenInventory(ctx)
	if err != nil {
		return err
	}
	lay, err := h.Layout(s)
	if err != nil {
		return err
	}
	if s.Inventory.Cursor != nil {
		if err := h.ClearCursor(ctx); err != nil {
			return fmt.Errorf("cursor not empty: %w", err)
		}
	}
	u, ok := hand.Find(s.Inventory, id)
	if !ok || u.Item == nil || u.Item.Where != d2.ContInventory {
		return ErrNoItem
	}
	if err := h.Take(ctx, u, lay.Inventory); err != nil {
		return err
	}
	if err := h.DropCursor(ctx); err != nil {
		return err
	}
	// DropCursor falls back to the pack when no ground click worked: that is not a trash.
	_, gone, err := h.Wait(ctx, 2*time.Second, func(s *d2.Snapshot) bool {
		_, still := hand.Find(s.Inventory, id)
		return !still && s.Inventory.Cursor == nil
	})
	if err != nil {
		return err
	}
	if !gone {
		return ErrNoRoomToDrop
	}
	return nil
}
