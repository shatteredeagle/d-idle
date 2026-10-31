package hand

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town/panel"
)

// SlotNames is the ONE table from a body location (d2.Body*) to the inventory.txt equipment slot name; the slot
// rectangles come from the layout (panel.Layout.Slots), so no pixel coordinate lives in code.
//
//	rArm (weapon) and neck (amulet): V live (docs/act2.md P1: the Shaft and the Top of the Staff were found worn at
//	body 4 and 2; T0: a slot click with the item on the cursor re-equips it).
//	torso, head, lArm, gloves: V live for clicking in (leveler equips "Heavy Gloves -> gloves", "Short Staff -> rArm",
//	soak 2026-10-29 i4).
//	rHand/lHand are the RING slots of inventory.txt (columns rHand/lHand, 23x24 boxes beside the belt): U whether
//	rHand is the body location 6 (rring) slot; the leveler accepts a ring in either (swapLanded). belt, feet: U.
//
// Live equips into the small 23x24 neck and ring boxes (and only those) failed in the 2026-10-29 soak ("did not reach
// slot neck/rHand", the item stayed on the cursor); the cause is not proven (see docs/town.md, Equip into small slots).
var SlotNames = map[int]string{
	d2.BodyHead: "head", d2.BodyNeck: "neck", d2.BodyTorso: "torso", d2.BodyRHand: "rArm", d2.BodyLHand: "lArm",
	d2.BodyRRing: "rHand", d2.BodyLRing: "lHand", d2.BodyBelt: "belt", d2.BodyFeet: "feet", d2.BodyGloves: "gloves",
}

// SlotName is the slot name of a body location ("" when unknown).
func SlotName(body int) string { return SlotNames[body] }

// SlotPoint is the click point for try 0, 1, 2...: the centre first, then points inside the box that move the
// click away from the centre in case the game's real box is offset from inventory.txt (U). Always inside r.
func SlotPoint(r panel.Rect, try int) panel.Pt {
	c := r.Center()
	w, h := r.R-r.L, r.B-r.T
	switch try % 3 {
	case 1:
		return panel.Pt{X: c.X - w/4, Y: c.Y - h/4}
	case 2:
		return panel.Pt{X: c.X + w/4, Y: c.Y + h/4}
	}
	return c
}
