package hand

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
)

// Weapon swap (the `w` key). VERIFIED LIVE (CLASSPAL, Paladin Palli, 2026-10-28): `w` flips every worn weapon and
// shield between item body 4/5 (the ACTIVE set) and 11/12 (the INACTIVE set), and the left/right skills read 0/0 on an
// empty active set. So Equipped(BodyRHand) is nil while the character stands on an empty swap set although a full
// set sits at 11/12: code that reads only 4/5 sees empty slots and buys gear it already owns. Whether the skills of
// the other set come back by themselves after swapping again is VERIFIED (106/100 returned after two presses).

// WeaponSets says which weapon sets hold items. active = body 4/5, alt = body 11/12.
type WeaponSets struct {
	Active, Alt bool
}

// SetsOf reads the two sets from an inventory.
func SetsOf(inv *d2.Inventory) WeaponSets {
	if inv == nil {
		return WeaponSets{}
	}
	return WeaponSets{
		Active: inv.Equipped(d2.BodyRHand) != nil || inv.Equipped(d2.BodyLHand) != nil,
		Alt:    inv.Equipped(d2.BodyAltR) != nil || inv.Equipped(d2.BodyAltL) != nil,
	}
}

// OnEmptySet: the active set is empty but the other one is not, i.e. the character stands on the wrong set.
func (w WeaponSets) OnEmptySet() bool { return !w.Active && w.Alt }

// FilledEither is the "does this body slot hold gear" test rebuy and the gear pass must use: the slot counts as
// filled when it or its twin in the other set holds an item, so a set swapped away is never bought again.
func FilledEither(inv *d2.Inventory, body int) bool {
	if inv.Equipped(body) != nil {
		return true
	}
	switch body {
	case d2.BodyRHand:
		return inv.Equipped(d2.BodyAltR) != nil
	case d2.BodyLHand:
		return inv.Equipped(d2.BodyAltL) != nil
	}
	return false
}

// EnsureMainSet presses `w` once when the character stands on an empty set while the other set holds gear, and proves
// the gear is active afterwards. It returns whether it swapped. A swap that did not take is an error (nothing is
// bought or equipped on top of a wrong set).
func (h *Hand) EnsureMainSet(ctx context.Context) (swapped bool, err error) {
	s, err := h.Fresh()
	if err != nil {
		return false, err
	}
	if !SetsOf(s.Inventory).OnEmptySet() {
		return false, nil
	}
	if PanelOpen(s.UI) {
		if err := h.CloseAll(ctx); err != nil {
			return false, err
		}
	}
	if err := h.E.In.Key("w"); err != nil {
		return false, err
	}
	_, ok, err := h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool { return !SetsOf(s.Inventory).OnEmptySet() })
	if err != nil {
		return false, err
	}
	if !ok {
		return false, fmt.Errorf("hand: weapon swap (w) did not bring the gear set back")
	}
	return true, nil
}
