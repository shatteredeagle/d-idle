package town

import (
	"context"
	"errors"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/town/hand"
)

// ErrNoRoom: the pack has no free w x h area and selling/stashing did not make one.
var ErrNoRoom = errors.New("town: no room in the pack")

// PackFits reports whether the backpack has a free w x h rectangle (the game places a bought item at the first
// free spot, so a 2x3 armor needs a real 2x3 hole, not 6 loose cells; VERIFIED LIVE on i4, 82% full pack).
func (t *Town) PackFits(s *d2.Snapshot, w, h int) (bool, error) {
	lay, err := t.layout(s)
	if err != nil {
		return false, err
	}
	_, _, ok := t.h.Occupancy(s.Inventory, d2.ContInventory, lay.Inventory).FindSpot(w, h)
	return ok, nil
}

// MakeRoom frees a w x h area in the pack. A vendor window must be open (selling is a right-click in the pack while
// it is). Order: sell what the caller calls junk (extra) and what the pickit would sell or drop; if there is still no
// hole, close the vendor, stash the pickit's stash items and then the unsold extra junk. The vendor window is CLOSED
// after the stash step, so callers check d2.UIShop and reopen the trade. Returns whether the hole exists now.
func (t *Town) MakeRoom(ctx context.Context, w, h int, extra []d2.Unit) (bool, error) {
	s, err := t.current()
	if err != nil {
		return false, err
	}
	if ok, err := t.PackFits(s, w, h); ok || err != nil {
		return ok, err
	}
	var rep ChoreReport
	var junk []d2.Unit
	seen := map[uint32]bool{}
	for _, u := range append(append([]d2.Unit(nil), extra...), t.sellableIf(s, true)...) {
		if !seen[u.ID] {
			seen[u.ID] = true
			junk = append(junk, u)
		}
	}
	var unsold []d2.Unit
	if s.UI.Open(d2.UIShop) {
		if unsold, err = t.sellTo(ctx, junk, &rep); err != nil {
			return false, err
		}
		t.e.Log.Info("town: make room sold", "sold", rep.Sold, "gold", rep.Earned, "refused", len(unsold))
		if s, err = t.current(); err != nil {
			return false, err
		}
		if ok, err := t.PackFits(s, w, h); ok || err != nil {
			return ok, err
		}
	} else {
		unsold = junk
	}
	a := ActOfLevel(s.Player.Area)
	if a == nil {
		return false, ErrNotInTown
	}
	if err := t.h.CloseAll(ctx); err != nil {
		return false, err
	}
	if err := t.openStash(ctx, a); err != nil {
		return false, err
	}
	defer func() { _ = t.h.CloseAll(ctx) }()
	if err := t.stashItems(ctx, &rep, false); err != nil {
		return false, err
	}
	for _, u := range unsold {
		if s, err = t.current(); err != nil {
			return false, err
		}
		if ok, err := t.PackFits(s, w, h); ok || err != nil {
			return ok, err
		}
		lay, err := t.layout(s)
		if err != nil {
			return false, err
		}
		fresh, ok := findIn(s.Inventory, u.ID)
		if !ok {
			continue
		}
		if err := t.h.Transfer(ctx, fresh, lay.Inventory, lay.Stash, d2.ContStash); err != nil {
			if errors.Is(err, hand.ErrNoSpace) {
				break
			}
			return false, err
		}
		rep.Stashed++
	}
	t.e.Log.Info("town: make room stashed", "stashed", rep.Stashed)
	if s, err = t.current(); err != nil {
		return false, err
	}
	return t.PackFits(s, w, h)
}
