package hand

import (
	"context"
	"fmt"
	"log/slog"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// dropOffsets are the cells (subtiles from the character) DropCursor clicks, in order; town.dropTries of them are
// used. The first is the old fixed target. (0,0) is deliberately absent: a click on the character's own cell does not
// drop (U). Why one cell refuses the drop is unknown (a non-walkable cell, a ground item or monster under the click,
// a coordinate mismatch are the candidates; Gate 2 g2b seed 13), so every try uses a new cell.
var dropOffsets = [...][2]int{{2, 2}, {-2, 2}, {2, -2}, {-2, -2}, {0, 3}, {3, 0}, {-3, 0}, {0, -3}}

// DropCursor puts the held item on the ground beside the character: a left click on the playfield with an item on the
// cursor drops it (U live: D2 semantics, not yet seen in this bot). It is the last resort for an item no pack cell
// takes (ClearCursor returned ErrNoSpace); a cursor item persists through a game re-entry (U), so leaving it held
// blocks every later pickup, shop and waypoint use. No-op when the cursor is empty. Panels are closed first so the
// click lands on the playfield.
//
// Each failed click is followed by a new cell (dropOffsets, never the same twice, at most town.dropTries). If the
// cursor is still occupied, the item goes to a free pack cell (ClearCursor) when there is one; otherwise ErrCursor is
// returned and the failsafe ladder takes over.
func (h *Hand) DropCursor(ctx context.Context) error {
	s, err := h.Fresh()
	if err != nil {
		return err
	}
	if s.Inventory.Cursor == nil {
		return nil
	}
	name := fmt.Sprintf("class%d", s.Inventory.Cursor.Class)
	if b := h.Base(*s.Inventory.Cursor); b != nil {
		name = b.Name
	}
	if err := h.E.In.CloseAll(); err != nil {
		return err
	}
	tries := min(max(h.E.Tune.Town.DropTries, 1), len(dropOffsets))
	for try := 1; try <= tries; try++ {
		off := dropOffsets[try-1]
		at := input.Point{X: s.Player.X + off[0], Y: s.Player.Y + off[1]}
		click := h.E.In.ClickWorld
		if wd, ok := h.E.In.(input.WorldDropper); ok {
			click = wd.DropClick // d2d: ClickWorld is a walk order and drops nothing
		}
		if err := click(at); err != nil {
			return err
		}
		ns, ok, err := h.Wait(ctx, h.uiTimeout(), func(s *d2.Snapshot) bool { return s.Inventory.Cursor == nil })
		if err != nil {
			return err
		}
		if ok {
			h.log().Info("cursor: dropped", "item", name, "try", try, "offset", fmt.Sprintf("(%d,%d)", off[0], off[1]))
			return nil
		}
		s = ns
	}
	if h.ClearCursor(ctx) == nil {
		h.log().Info("cursor: no drop worked, put the item into the pack", "item", name, "tries", tries)
		return nil
	}
	h.log().Warn(fmt.Sprintf("cursor: could not clear after %d tries", tries), "items", name)
	return fmt.Errorf("%w: item still on the cursor after %d ground clicks", ErrCursor, tries)
}

func (h *Hand) log() *slog.Logger {
	if h.E.Log != nil {
		return h.E.Log
	}
	return slog.Default()
}
