package town

import (
	"context"
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/belt"
	"d2lod-bot/internal/town/hand"
	"d2lod-bot/internal/town/panel"
)

// maxRepairColumns bounds the columns one FillBelt call empties (a chore step, not an overhaul).
const maxRepairColumns = 2

// beltRepair says which kind lacks a belt column (the plan has room for it, the pack holds some, and PlaceFor finds no
// box) and which belt boxes of another kind to empty for it (belt.Surplus). Healing first. ok is false when the layout is
// fine or has nothing to give. Pure: NeedsBeltFill and FillBelt share it.
func (t *Town) beltRepair(s *d2.Snapshot) (starved belt.Kind, boxes []int, from belt.Kind, ok bool) {
	if s == nil || s.Inventory == nil || s.Player == nil || !s.Player.InTown() {
		return
	}
	band := t.plan(s.Player)
	st := belt.Read(t.e.Data, s.Inventory)
	placed := belt.PlacedItems(t.e.Data, s.Inventory)
	for _, k := range []belt.Kind{belt.HP, belt.RV, belt.MP} {
		if band.Capacity(k, st.Slots)-st.Count[k] <= 0 || belt.PackCount(t.e.Data, s.Inventory, k) == 0 {
			continue
		}
		if boxes, from, ok = belt.Surplus(st.Slots, placed, k, band); ok {
			return k, boxes, from, true
		}
	}
	return
}

// repairBelt empties the surplus columns found by beltRepair, one potion at a time from the top of the column, into the
// pack (the pack must be open). It only touches potions in the belt, never equipment, and only in town. Logs
// `belt layout` with the state before it acts. Returns how many potions it moved.
func (t *Town) repairBelt(ctx context.Context, lay *panel.Layout, rep *BeltFillReport) int {
	moved := 0
	for col := 0; col < maxRepairColumns; col++ {
		s, err := t.current()
		if err != nil {
			return moved
		}
		starved, boxes, from, ok := t.beltRepair(s)
		if !ok {
			return moved
		}
		st := belt.Read(t.e.Data, s.Inventory)
		t.e.Log.Info(fmt.Sprintf("belt layout: violated, no column for %s (belt hp=%d mp=%d rv=%d slots=%d, pack %s=%d), emptying %d %s potions",
			starved, st.Count[belt.HP], st.Count[belt.MP], st.Count[belt.RV], st.Slots, starved, belt.PackCount(t.e.Data, s.Inventory, starved), len(boxes), from),
			"boxes", fmt.Sprint(boxes))
		for _, box := range boxes {
			if ctx.Err() != nil {
				return moved
			}
			u, found := beltUnitAt(s.Inventory, box)
			if !found {
				continue
			}
			if err := t.beltToPack(ctx, lay, u, st.Slots); err != nil {
				t.e.Log.Warn("belt layout: could not move a potion out", "item", t.nameOf(u), "box", box, "err", err)
				return moved
			}
			moved++
			rep.Repaired++
			if s, err = t.current(); err != nil {
				return moved
			}
		}
	}
	return moved
}

func beltUnitAt(inv *d2.Inventory, box int) (d2.Unit, bool) {
	for _, u := range inv.Container(d2.ContBelt) {
		if u.Item != nil && u.Item.GY*belt.Columns+u.Item.GX == box {
			return u, true
		}
	}
	return d2.Unit{}, false
}

// beltToPack takes one potion out of its belt box into the first free pack cell: the item verb when the backend has one
// (U: the verb is documented for pack and stash; a belt item is the same ToCursor/ToGrid pair), else a click on the box
// and a put in the pack (U for d2d: does a left click on a belt box lift the potion; game.exe does).
func (t *Town) beltToPack(ctx context.Context, lay *panel.Layout, u d2.Unit, slots int) error {
	if t.mover != nil {
		if done, err := t.mover.Move(ctx, u.ID, false); done {
			return err
		}
	}
	slot, ok := lay.Slots["belt"]
	if !ok {
		return fmt.Errorf("town: no belt slot in the layout")
	}
	box := u.Item.GY*belt.Columns + u.Item.GX
	rows := max(slots/belt.Columns, 1)
	cw, ch := (slot.R-slot.L)/belt.Columns, (slot.B-slot.T)/rows
	p := panel.Pt{X: slot.L + box%belt.Columns*cw + cw/2, Y: slot.B - box/belt.Columns*ch - ch/2}
	if err := t.h.E.In.Click(input.Left, p.X, p.Y); err != nil {
		return err
	}
	if _, ok, err := t.h.Wait(ctx, t.uiTimeout(), func(s *d2.Snapshot) bool {
		return s.Inventory.Cursor != nil && s.Inventory.Cursor.ID == u.ID
	}); err != nil {
		return err
	} else if !ok {
		return fmt.Errorf("%w: belt box %d did not lift %s", hand.ErrCursor, box, t.nameOf(u))
	}
	if err := t.h.ClearCursor(ctx); err != nil {
		return err
	}
	cur, err := t.current()
	if err != nil {
		return err
	}
	if it, ok := hand.Find(cur.Inventory, u.ID); !ok || it.Item == nil || it.Item.Where != d2.ContInventory {
		return fmt.Errorf("town: %s is not in the pack after the move", t.nameOf(u))
	}
	return nil
}
