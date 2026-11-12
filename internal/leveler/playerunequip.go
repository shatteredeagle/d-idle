package leveler

import (
	"context"
	"errors"

	"d2lod-bot/internal/d2"
)

// ErrNotWorn is returned when an unequip names an item that is not worn.
var ErrNotWorn = errors.New("the item is not worn")

// PlayerUnequip takes the worn item id off and puts it in the backpack through the normal click path (hand.Unequip:
// click the slot, put the cursor item in a free pack cell, each step proven from memory). Like PlayerEquip it is a
// player command: the caller picks the moment, a fight nearby returns ErrBusy, nothing writes game memory
// (docs/gearsets.md).
func (l *Leveler) PlayerUnequip(ctx context.Context, id uint32) error {
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
	worn := false
	for _, u := range s.Inventory.Container(d2.ContEquipment) {
		worn = worn || u.ID == id
	}
	if !worn {
		return ErrNotWorn
	}
	if l.o.Dry {
		return nil
	}
	err = h.Unequip(ctx, id)
	if cerr := h.CloseAll(ctx); cerr != nil {
		l.e.Log.Warn("leveler: closing panels after the player's unequip failed", "err", cerr)
	}
	if err == nil {
		l.logf("player unequip", "id", id)
	}
	return err
}

// IsTwoHanded says whether item id (in any container of the snapshot) takes both hands: the rule behind the
// two-hander and shield refusal of CheckPlayerEquip, used to order the steps of a gear set apply.
func (l *Leveler) IsTwoHanded(s *d2.Snapshot, id uint32) bool {
	if s == nil || s.Player == nil || s.Inventory == nil {
		return false
	}
	for _, u := range s.Inventory.Items {
		if u.ID == id && u.Item != nil {
			return l.cand(s.Player, u).TwoHanded
		}
	}
	return false
}
