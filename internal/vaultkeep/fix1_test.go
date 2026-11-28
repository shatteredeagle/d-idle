package vaultkeep

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Card fix1: every test here MUST FAIL when the displaced item (or the carried charm) can vanish or double.

func (d *drig) reopen() {
	d.open()
	d.k.Hand, d.k.EquipPath = d.h, d.dir+"/equip.json"
	d.k.Direct, d.k.DirectOn = d.df, func() bool { return true }
}

// A failure AFTER the engine swapped (the ledger line of the reply) must not drop the displaced item.
func TestFix1LateFailureFilesDisplaced(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			d := directRig(t, class)
			d.wearOld("head", "cap", "Old Cap")
			id := d.add(1, "cap", "new")
			d.df.LateFail = errors.New("ledger: disk full")
			res := d.k.EquipEnqueue(context.Background(), id, "", false)
			if !res.OK || d.live("previously worn") != 1 {
				t.Fatalf("displaced item not filed after a late failure: %+v live=%d", res, d.live("previously worn"))
			}
			if rec, _ := d.st.Get(id); rec.State == vault.Recorded {
				t.Fatal("the worn record is still in the vault (duplicate)")
			}
		})
	}
}

// The reply is journaled before the keeper's own steps: a stop right after the engine answered is repaired from the file,
// and a second tap after the restart neither swaps again nor overwrites the saved bytes.
func TestFix1RestartAfterReplyThenSecondTap(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			d := directRig(t, class)
			old := d.wearOld("head", "cap", "Old Cap")
			id := d.add(1, "cap", "new")
			// the order as the journal hook left it: applied, bytes saved, nothing filed (the process died there)
			d.k.putEq(&EquipOrder{ID: id, Name: "cap 1", Slot: SlotAuto, Group: []string{"head"}, State: EqEquipping, Direct: true,
				Order: "eq-x-1", Applied: true, Worn: "head", Removed: []string{old}})
			d.df.Worn["head"] = createHex(mustRec(t, d, "new"))
			d.reopen()
			res := d.k.EquipEnqueue(context.Background(), id, "", false)
			if res.OK || len(d.df.Calls) != 0 {
				t.Fatalf("second tap swapped again: %+v calls %v", res, d.df.Calls)
			}
			if d.live("previously worn") != 1 {
				t.Fatalf("the saved displaced item was not filed: %d", d.live("previously worn"))
			}
			if rec, _ := d.st.Get(id); rec.State == vault.Recorded {
				t.Fatal("the worn record is still in the vault (duplicate)")
			}
		})
	}
}

// A lost reply (timeout after the engine ran): the copy read before the call is filed, the record leaves, no second swap.
func TestFix1DroppedReplyResolvedByLooking(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			d := directRig(t, class)
			unit := d.h.wear("head", "cap", "oldhead") // the game's worn unit, readable before the swap
			var norm string
			for _, c := range d.api.VaultFake.Carried {
				if c.Rec.UnitID == unit {
					norm = c.Rec.Norm
				}
			}
			d.df.Worn["head"] = norm
			if d.api.VaultFake.Bytes == nil {
				d.api.VaultFake.Bytes = map[string]d2dctl.VaultRecord{}
			}
			d.api.VaultFake.Bytes[norm] = d2dctl.FakeRecord(0, "cap", 6, 20, "oldhead")
			id := d.add(1, "cap", "new")
			d.df.DropReply = true
			d.df.After = func([]d2dctl.DirectResult) { // the engine removed the old unit from the game
				d.api.VaultFake.Remove(unit)
				delete(d.h.body, unit)
				d.mirror() // and the new item is worn (card fix1b)
			}
			res := d.k.EquipEnqueue(context.Background(), id, "", false)
			if d.live("previously worn") != 1 {
				t.Fatalf("lost reply dropped the displaced item: %+v", res)
			}
			if rec, _ := d.st.Get(id); rec.State == vault.Recorded {
				t.Fatal("the worn record is still in the vault (duplicate)")
			}
			d.df.DropReply = false
			if res = d.k.EquipEnqueue(context.Background(), id, "", false); res.OK || len(d.df.Calls) != 1 {
				t.Fatalf("second tap swapped again: %+v %v", res, d.df.Calls)
			}
		})
	}
}

// An unanswered call the bot cannot judge stays OPEN: no "nothing happened", and no new order for the record.
func TestFix1UnknownOutcomeBlocksSecondTap(t *testing.T) {
	d := directRig(t, classes[0])
	unit := d.h.wear("head", "cap", "oldhead")
	id := d.add(1, "cap", "new")
	d.df.DropReply = true
	d.df.After = func([]d2dctl.DirectResult) { delete(d.h.body, unit) } // off the body, still readable: not judged
	d.k.EquipEnqueue(context.Background(), id, "", false)
	d.df.DropReply = false
	if res := d.k.EquipEnqueue(context.Background(), id, "", false); res.Code != CodeInFlight || len(d.df.Calls) != 1 {
		t.Fatalf("an unjudged order was replaced: %+v %v", res, d.df.Calls)
	}
}

// An identical item already in the vault: the displaced twin is NOT counted as filed; it goes back to the hero.
func TestFix1TwinDisplacedIsNotSilentlyDropped(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			d := directRig(t, class)
			old := d.wearOld("rring", "rin", "Angelic Halo")
			rec := d.api.VaultFake.Bytes[old]
			if _, err := d.st.Add(vault.Item{Raw: old, Code: "rin", Name: "Angelic Halo", Quality: 5, Ilvl: 20, Identified: true}); err != nil {
				t.Fatal(err)
			}
			_ = rec
			id := d.add(1, "rin", "new")
			before := len(d.api.VaultFake.Carried)
			res := d.k.EquipEnqueue(context.Background(), id, SlotRight, false)
			if !res.OK || res.Where == "vault" || len(d.api.VaultFake.Carried) != before+1 {
				t.Fatalf("the twin vanished: %+v carried %d->%d", res, before, len(d.api.VaultFake.Carried))
			}
		})
	}
}

// Carry: a stop after the create and before the record is retired is completed at the next start (no live record next
// to the carried charm); a refused create rolls the intent back.
func TestFix1CarryIntentRepaired(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r, h := carryRig(t, class)
			ctx := context.Background()
			id := r.charm(1, "cm1", 4, true)
			rec, _ := r.st.Get(id)
			// the journal as a stop after the create left it
			cr, err := r.api.ItemCreate(ctx, createHex(rec), d2dctl.CreateOpts{Pack: true, Key: carryKey + id})
			if err != nil {
				t.Fatal(err)
			}
			_ = cr
			if err := r.k.carryPut(id, &Carried{ID: id, Name: rec.Name, Code: "cm1", Cells: 1, State: carryIntent, Hex: createHex(rec)}); err != nil {
				t.Fatal(err)
			}
			r.open()
			r.k.Hand, r.k.CarryPath = h, r.dir+"/carried.json"
			r.k.repairDirect(ctx)
			if got, _ := r.st.Get(id); got.State == vault.Recorded {
				t.Fatal("charm in the pack and its record still live: sellable duplicate")
			}
			if l := r.k.CarriedList(); len(l) != 1 || l[0].State != "" || len(h.protected) != 1 {
				t.Fatalf("carried %+v protected %d", l, len(h.protected))
			}
			// intent with no charm in the pack: rolled back, the record stays
			id2 := r.charm(2, "cm1", 4, true)
			rec2, _ := r.st.Get(id2)
			_ = r.k.carryPut(id2, &Carried{ID: id2, State: carryIntent, Hex: createHex(rec2), Code: "cm1"})
			r.k.repairDirect(ctx)
			if got, _ := r.st.Get(id2); got.State != vault.Recorded || len(r.k.CarriedList()) != 1 {
				t.Fatalf("rollback: %+v %+v", got.State, r.k.CarriedList())
			}
		})
	}
}

// Stow after a game restart: the unit id in the file is stale; the charm is found by its bytes and the record exists
// before the charm leaves the pack.
func TestFix1StowFindsCharmByBytes(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r, _ := carryRig(t, class)
			ctx := context.Background()
			id := r.charm(1, "cm1", 4, true)
			if res := r.k.Carry(ctx, id); !res.OK {
				t.Fatalf("%+v", res)
			}
			c := r.k.CarriedList()[0]
			c.Unit += 1000 // the restart renumbered the units
			_ = r.k.carryPut(id, &c)
			res := r.k.Stow(ctx, id)
			if !res.OK || r.carried(d2dctl.FakePack) != 0 || len(r.k.CarriedList()) != 0 {
				t.Fatalf("stow by bytes: %+v pack %d", res, r.carried(d2dctl.FakePack))
			}
			n := 0
			for _, x := range r.st.Dump() {
				if x.State == vault.Recorded && x.Found.Zone == zoneWorn {
					n++
				}
			}
			if n != 1 {
				t.Fatalf("stowed records: %d", n)
			}
		})
	}
}
