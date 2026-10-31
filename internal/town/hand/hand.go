// Package hand moves items with the mouse and proves every move by re-reading memory: pick an item up
// (cursor holds it), put it down in a grid cell, right-click it (use, sell, buy). Nothing here decides
// WHAT to move; town and loot do. All waits go through env.Clock so tests never sleep.
package hand

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/panel"
)

var (
	ErrNoItem    = errors.New("hand: item not found")
	ErrCursor    = errors.New("hand: unexpected cursor state")
	ErrNoSpace   = errors.New("hand: no free space in target")
	ErrNotOpen   = errors.New("hand: panel did not open")
	ErrNotInGame = errors.New("hand: not in game")
	ErrMoveFail  = errors.New("hand: move not confirmed in memory")
)

// Hand is the verified mouse layer. Snap is the one seam tests replace.
type Hand struct {
	E    *env.Env
	Snap func() (*d2.Snapshot, error)
	lay  *panel.Layout
}

func New(e *env.Env) *Hand { return &Hand{E: e, Snap: e.Snapshot} }

// Tune shortcuts.
func (h *Hand) settle() time.Duration {
	return time.Duration(h.E.Tune.Town.ClickSettleMs) * time.Millisecond
}
func (h *Hand) uiTimeout() time.Duration {
	return time.Duration(h.E.Tune.Town.UITimeoutMs) * time.Millisecond
}

// Layout returns the panel layout for the running window and character (cached).
func (h *Hand) Layout(s *d2.Snapshot) (*panel.Layout, error) {
	if h.lay != nil && h.lay.Screen == (panel.Pt{X: s.Screen.X, Y: s.Screen.Y}) {
		return h.lay, nil
	}
	if s.Player == nil {
		return nil, ErrNotInGame
	}
	l, err := panel.NewLayout(h.E.Data, s.Player.Class, panel.Pt{X: s.Screen.X, Y: s.Screen.Y}, true)
	if err != nil {
		return nil, err
	}
	h.lay = l
	return l, nil
}

// Fresh reads a snapshot and insists the player is in a game.
func (h *Hand) Fresh() (*d2.Snapshot, error) {
	s, err := h.Snap()
	if err != nil {
		return nil, err
	}
	if s.State != d2.StateInGame || s.Player == nil || s.Inventory == nil {
		return nil, ErrNotInGame
	}
	return s, nil
}

// Wait polls fn on fresh snapshots every settle interval until it returns true or the timeout passes.
func (h *Hand) Wait(ctx context.Context, timeout time.Duration, fn func(*d2.Snapshot) bool) (*d2.Snapshot, bool, error) {
	deadline := h.E.Clock.Now().Add(timeout)
	for {
		s, err := h.Fresh()
		if err != nil {
			return nil, false, err
		}
		if fn(s) {
			return s, true, nil
		}
		if !h.E.Clock.Now().Before(deadline) {
			return s, false, nil
		}
		if err := ctx.Err(); err != nil {
			return s, false, err
		}
		h.E.Clock.Sleep(h.settle())
	}
}

// WaitFlag waits until the UI flag idx equals want.
func (h *Hand) WaitFlag(ctx context.Context, idx int, want bool) (*d2.Snapshot, error) {
	s, ok, err := h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool { return s.UI.Open(idx) == want })
	if err != nil {
		return s, err
	}
	if !ok {
		return s, fmt.Errorf("%w: ui flag 0x%x never became %v", ErrNotOpen, idx, want)
	}
	return s, nil
}

// OpenInventory opens the backpack with the hotkey if it is not open.
func (h *Hand) OpenInventory(ctx context.Context) (*d2.Snapshot, error) {
	s, err := h.Fresh()
	if err != nil {
		return nil, err
	}
	if s.UI.Open(d2.UIInventory) {
		return s, nil
	}
	if err := h.E.In.Key("i"); err != nil {
		return nil, err
	}
	return h.WaitFlag(ctx, d2.UIInventory, true)
}

// CloseAll closes every panel and makes sure nothing is left on the cursor (it is returned to the pack
// when possible). Returns an error if panels remain open.
func (h *Hand) CloseAll(ctx context.Context) error {
	if s, err := h.Fresh(); err == nil && s.Inventory.Cursor != nil {
		_ = h.ClearCursor(ctx)
	}
	if err := h.E.In.CloseAll(); err != nil {
		return err
	}
	s, ok, err := h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool { return !PanelOpen(s.UI) })
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("hand: panels still open: %v", s.UI)
	}
	return nil
}

// Size is the grid footprint of an item unit (1x1 when the base is unknown).
func (h *Hand) Size(u d2.Unit) (w, hgt int) {
	if db := h.E.Data; db != nil && u.Class >= 0 && u.Class < len(db.Items) {
		b := db.Items[u.Class]
		if b.Width > 0 && b.Height > 0 {
			return b.Width, b.Height
		}
	}
	return 1, 1
}

// Base is the data row of an item unit, or nil.
func (h *Hand) Base(u d2.Unit) *data.ItemBase {
	if db := h.E.Data; db != nil && u.Class >= 0 && u.Class < len(db.Items) {
		return db.Items[u.Class]
	}
	return nil
}

// Occupancy builds the cell map of one container of the player's inventory (backpack, stash, cube).
func (h *Hand) Occupancy(inv *d2.Inventory, c d2.Container, g panel.Grid) *panel.Occupancy {
	o := panel.NewOccupancy(g.Cols, g.Rows)
	for _, u := range inv.Container(c) {
		w, hh := h.Size(u)
		o.Place(u.X, u.Y, w, hh)
	}
	return o
}

// Find locates an item unit by id in the player's inventory lists (cursor included).
func Find(inv *d2.Inventory, id uint32) (d2.Unit, bool) {
	for _, u := range inv.Items {
		if u.ID == id {
			return u, true
		}
	}
	if inv.Cursor != nil && inv.Cursor.ID == id {
		return *inv.Cursor, true
	}
	return d2.Unit{}, false
}

func (h *Hand) click(b input.Button, p panel.Pt, mods ...input.Mod) error {
	return h.E.In.Click(b, p.X, p.Y, mods...)
}

// Take left-clicks u in grid g (its panel must be open) and confirms the cursor now holds it.
func (h *Hand) Take(ctx context.Context, u d2.Unit, g panel.Grid) error {
	w, hh := h.Size(u)
	if err := h.click(input.Left, g.ItemCenter(u.X, u.Y, w, hh)); err != nil {
		return err
	}
	_, ok, err := h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool {
		return s.Inventory.Cursor != nil && s.Inventory.Cursor.ID == u.ID
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: cursor never held item %d", ErrCursor, u.ID)
	}
	return nil
}

// Put left-clicks the footprint centre of cell (col,row) of g with the cursor item (size w x hh) and
// confirms the cursor is empty and the item sits in container want at (col,row).
func (h *Hand) Put(ctx context.Context, id uint32, g panel.Grid, col, row, w, hh int, want d2.Container) error {
	if err := h.click(input.Left, g.ItemCenter(col, row, w, hh)); err != nil {
		return err
	}
	_, ok, err := h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool {
		if s.Inventory.Cursor != nil {
			return false
		}
		it, found := Find(s.Inventory, id)
		return found && it.Item != nil && it.Item.Where == want && it.X == col && it.Y == row
	})
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("%w: item %d not at %v (%d,%d)", ErrMoveFail, id, want, col, row)
	}
	return nil
}

// Transfer moves an item already in the player's inventory lists from one container grid to another:
// pick up, find a spot in dst, put down. Each step is verified; on a failed put the item is put back where
// it came from. Both panels must be open.
func (h *Hand) Transfer(ctx context.Context, u d2.Unit, src panel.Grid, dst panel.Grid, dstCont d2.Container) error {
	var last error
	for try := 0; try < h.E.Tune.Town.MoveTries; try++ {
		if try > 0 {
			h.E.Clock.Sleep(h.settle())
		}
		s, err := h.Fresh()
		if err != nil {
			return err
		}
		cur, ok := Find(s.Inventory, u.ID)
		if !ok {
			return ErrNoItem
		}
		if cur.Item != nil && cur.Item.Where == dstCont {
			return nil
		}
		if s.Inventory.Cursor != nil && s.Inventory.Cursor.ID != u.ID {
			return fmt.Errorf("%w: cursor holds a different item", ErrCursor)
		}
		w, hh := h.Size(cur)
		occ := h.Occupancy(s.Inventory, dstCont, dst)
		c, r, found := occ.FindSpot(w, hh)
		if !found {
			return ErrNoSpace
		}
		if s.Inventory.Cursor == nil {
			if last = h.Take(ctx, cur, src); last != nil {
				continue
			}
		}
		if last = h.Put(ctx, u.ID, dst, c, r, w, hh, dstCont); last == nil {
			return nil
		}
		// Put failed: if the item is still on the cursor, return it to its source cell.
		if s2, err := h.Fresh(); err == nil && s2.Inventory.Cursor != nil && s2.Inventory.Cursor.ID == u.ID {
			_ = h.click(input.Left, src.ItemCenter(cur.X, cur.Y, w, hh))
			h.E.Clock.Sleep(h.settle())
		}
	}
	return fmt.Errorf("hand: transfer of item %d failed: %w", u.ID, last)
}

// RightClick right-clicks an item in grid g: buys from a vendor panel, sells from the pack while a vendor
// panel is open, uses a potion or scroll otherwise. The caller verifies the effect.
func (h *Hand) RightClick(u d2.Unit, g panel.Grid) error {
	w, hh := h.Size(u)
	return h.click(input.Right, g.ItemCenter(u.X, u.Y, w, hh))
}

// ShiftClick shift-left-clicks an item in grid g. On a potion in the pack D2 sends it to the belt (U: live check
// pending); the caller verifies the effect.
func (h *Hand) ShiftClick(u d2.Unit, g panel.Grid) error {
	w, hh := h.Size(u)
	return h.click(input.Left, g.ItemCenter(u.X, u.Y, w, hh), input.ModShift)
}

// ClearCursor puts whatever the cursor holds into the first free backpack cell (the backpack must be open
// for the click to land; it is opened when needed).
func (h *Hand) ClearCursor(ctx context.Context) error {
	s, err := h.Fresh()
	if err != nil {
		return err
	}
	cur := s.Inventory.Cursor
	if cur == nil {
		return nil
	}
	if s, err = h.OpenInventory(ctx); err != nil {
		return err
	}
	lay, err := h.Layout(s)
	if err != nil {
		return err
	}
	w, hh := h.Size(*cur)
	c, r, ok := h.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory).FindSpot(w, hh)
	if !ok {
		return ErrNoSpace
	}
	return h.Put(ctx, cur.ID, lay.Inventory, c, r, w, hh, d2.ContInventory)
}

// PanelOpen reports whether a panel, dialog or menu that blocks clicks is open. The panel table in internal/d2/ui.go
// already leaves out the quest-log button flag 0x11 (stays set after a quest update; VERIFIED LIVE, docs/town.md).
func PanelOpen(u d2.UIState) bool { return u.BlockingPanelOpen() }
