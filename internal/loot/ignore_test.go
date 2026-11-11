package loot_test

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
)

// An item the player trashed lies on the ground and is not picked up again; the same kind of item elsewhere still is.
// Runs for two classes: the ignore is not class specific.
func TestPickUpLeavesTrashedItems(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		l, r := newLooter(t)
		w := r.W
		dress(w)
		w.Player.Class = cls
		w.Player.Gold = 5000
		trashed := w.AddGround("r03", 2, 6030, 4930)
		wanted := w.AddGround("r04", 2, 6021, 4931)
		l.SetIgnore(func(u d2.Unit) bool { return u.ID == trashed.ID })
		rep, err := l.PickUp(context.Background(), loot.PickOpts{})
		if err != nil {
			t.Fatal(err)
		}
		if len(rep.Picked) != 1 || rep.Picked[0].ID != wanted.ID || len(w.Ground) != 1 || w.Ground[0].ID != trashed.ID {
			t.Fatalf("%v: picked %+v ground %d", cls, rep.Picked, len(w.Ground))
		}
		for _, sk := range rep.Skipped {
			if sk.ID == trashed.ID {
				t.Errorf("%v: the trashed item is reported as skipped", cls)
			}
		}
		l.SetIgnore(nil)
		if rep, _ = l.PickUp(context.Background(), loot.PickOpts{}); len(rep.Picked) != 1 {
			t.Errorf("%v: without the ignore the item is picked up: %+v", cls, rep.Picked)
		}
	}
}
