package vaultkeep

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Card fix1b (review round 2, N1 N2 N3 N5): every test here MUST FAIL when a unit id is trusted across a new game, or
// when an item under an open order can be sold, carried or handed to the mercenary.

// mirror is the game side of a direct swap: what the direct fake wears becomes a worn unit the hand and the item list show.
func (d *drig) mirror() {
	for slot, hx := range d.df.Worn {
		if d.h.wornIn(slot) != 0 {
			continue
		}
		cr, err := d.api.VaultFake.ItemCreate(context.Background(), hx, d2dctl.CreateOpts{Pack: true})
		if err != nil {
			d.t.Fatal(err)
		}
		for i := range d.api.VaultFake.Carried {
			if d.api.VaultFake.Carried[i].Rec.UnitID == cr.ID {
				d.api.VaultFake.Carried[i].Where = d2dctl.FakeWorn
			}
		}
		d.h.body[cr.ID] = slot
	}
}

// renumber is a new game (or a game restart): every unit of the hero gets another id. With by > 0 the ids shift, so a
// unit can take the number another one had.
func renumber(api *d2dctl.VaultFake, h *fakeHand, by int) {
	for i := range api.Carried {
		api.Carried[i].Rec.UnitID += by
	}
	body, names := map[int]string{}, map[int]string{}
	for u, s := range h.body {
		body[u+by] = s
	}
	for u, n := range h.names {
		names[u+by] = n
	}
	h.body, h.names = body, names
}

func (d *drig) wornNorm(slot string) string {
	u := d.h.wornIn(slot)
	for _, c := range d.api.VaultFake.Carried {
		if c.Rec.UnitID == u {
			return c.Rec.Norm
		}
	}
	return ""
}

// N2: the reply is lost, the look fails too (the order stays open), then a new game renumbers the units. The verdict must
// come from the item bytes. Scenario A: nothing was swapped and the ids shifted. Scenario B: the swap ran and the new item
// got the old item's number.
func TestFix1bOpenOrderJudgedByBytesAfterRestart(t *testing.T) {
	for _, class := range classes {
		t.Run(class+"/not swapped, ids shifted", func(t *testing.T) {
			d := directRig(t, class)
			_, old := d.wearReal("head", "cap", "oldhead")
			id := d.add(1, "cap", "new")
			d.df.DropReply = true
			d.df.After = func([]d2dctl.DirectResult) { // the engine never ran the command; the look after the timeout fails
				d.df.Worn["head"] = old
				d.ch.dead = true
			}
			if res := d.k.EquipEnqueue(context.Background(), id, "", false); res.Code != CodeInFlight {
				t.Fatalf("the order did not stay open: %+v", res)
			}
			d.ch.reset()
			renumber(d.api.VaultFake, d.h.fakeHand, 7)
			d.k.repairDirect(context.Background())
			if rec, _ := d.st.Get(id); rec.State != vault.Recorded {
				t.Fatal("the new item was never worn and its record is gone: lost")
			}
			if d.live("previously worn") != 0 {
				t.Fatal("the old item is still worn and was filed as a record: duplicate")
			}
			if d.k.directOpen(id) {
				t.Fatal("the order is still open although the slot holds the old item")
			}
		})
		t.Run(class+"/swapped, new item has the old number", func(t *testing.T) {
			d := directRig(t, class)
			unit, old := d.wearReal("head", "cap", "oldhead")
			id := d.add(1, "cap", "new")
			d.df.DropReply = true
			d.df.After = func(r []d2dctl.DirectResult) {
				d.gone(unit)(r)
				d.ch.dead = true
			}
			if res := d.k.EquipEnqueue(context.Background(), id, "", false); res.Code != CodeInFlight {
				t.Fatalf("the order did not stay open: %+v", res)
			}
			d.ch.reset()
			renumber(d.api.VaultFake, d.h.fakeHand, unit-d.h.wornIn("head")) // the reload gives the worn new item the old number
			if d.h.wornIn("head") != unit {
				t.Fatal("test setup: the new item should carry the old unit id")
			}
			d.k.repairDirect(context.Background())
			if !d.retired(id) {
				t.Fatal("the new item is worn and its record is still live: duplicate")
			}
			n := 0
			for _, r := range d.st.Dump() {
				if r.State == vault.Recorded && r.Raw == old {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("the old item must be filed exactly once, got %d", n)
			}
		})
		t.Run(class+"/empty slot, swapped", func(t *testing.T) { // review N4, first shape: the filled slot proves the swap
			d := directRig(t, class)
			id := d.add(1, "cap", "new")
			d.df.DropReply = true
			d.df.After = func([]d2dctl.DirectResult) { d.mirror() }
			d.k.EquipEnqueue(context.Background(), id, "", false)
			if !d.retired(id) {
				t.Fatal("the item is worn in the empty slot and its record is still live: duplicate")
			}
		})
		t.Run(class+"/another item in the slot stays open", func(t *testing.T) {
			d := directRig(t, class)
			unit, _ := d.wearReal("head", "cap", "oldhead")
			id := d.add(1, "cap", "new")
			d.df.DropReply = true
			d.df.After = func(r []d2dctl.DirectResult) {
				d.gone(unit)(r)
				for i := range d.api.VaultFake.Carried { // neither the old nor the new bytes: not judged
					if d.api.VaultFake.Carried[i].Rec.UnitID == d.h.wornIn("head") {
						d.api.VaultFake.Carried[i].Rec.Norm = d2dctl.FakeRecord(0, "cap", 6, 20, "third").Norm
					}
				}
			}
			d.k.EquipEnqueue(context.Background(), id, "", false)
			if !d.k.directOpen(id) || d.retired(id) || d.live("previously worn") != 0 {
				t.Fatalf("an order the bytes cannot judge was settled: open %v retired %v filed %d", d.k.directOpen(id), d.retired(id), d.live("previously worn"))
			}
		})
	}
}
