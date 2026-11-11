package leveler_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/leveler"
)

// Trash picks the item up and drops it beside the hero; the item ends on the ground, not in the pack and not on the
// cursor. Run for two classes: nothing here is class specific. A refused ground click that falls back to the pack is an
// error, and a fight nearby postpones.
func TestPlayerTrashDropsTheItem(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, lv, us := equipRig(t, "amu")
		w := r.W
		w.Player.Class = cls
		s, _ := w.Snapshot()
		if _, err := lv.CheckPlayerTrash(s, 999); !errors.Is(err, leveler.ErrNoItem) {
			t.Fatalf("%v: unknown item: %v", cls, err)
		}
		if err := lv.PlayerTrash(context.Background(), us[0].ID); err != nil {
			t.Fatalf("%v: %v", cls, err)
		}
		if inPack(w, us[0].ID) || w.Cursor != nil || len(w.Ground) != 1 {
			t.Fatalf("%v: pack %v cursor %v ground %d", cls, inPack(w, us[0].ID), w.Cursor != nil, len(w.Ground))
		}
	}
}

func TestPlayerTrashFallbackToPackIsAnError(t *testing.T) {
	r, lv, us := equipRig(t, "amu")
	r.W.DropRefuse = 99 // every ground click is refused: DropCursor puts the item back into the pack
	err := lv.PlayerTrash(context.Background(), us[0].ID)
	if err == nil || !inPack(r.W, us[0].ID) || r.W.Cursor != nil {
		t.Fatalf("err %v, in pack %v, cursor %v", err, inPack(r.W, us[0].ID), r.W.Cursor != nil)
	}
}
