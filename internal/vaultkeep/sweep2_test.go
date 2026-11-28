package vaultkeep

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Review 10 N4 (card sweep2): a direct Equip whose order cannot be written down is refused before the engine call. Before:
// the write error was dropped, the call ran, and a lost reply plus a restart left nothing to settle it with.
func TestSweep2DirectEquipRefusedWhenTheJournalCannotBeWritten(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			d := directRig(t, class)
			d.wearReal("head", "cap", "oldhead")
			id := d.add(1, "cap", "new")
			good := d.k.EquipPath
			block := filepath.Join(t.TempDir(), "file")
			if err := os.WriteFile(block, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			d.k.EquipPath = filepath.Join(block, "equip.json") // its directory is a file: every write fails
			res := d.k.EquipEnqueue(context.Background(), id, "", false)
			if res.Code != "journal" || len(d.df.Calls) != 0 {
				t.Fatalf("want the refusal journal and no engine call, got %+v calls %v", res, d.df.Calls)
			}
			if rec, _ := d.st.Get(id); rec.State != vault.Recorded || d.k.directOpen(id) {
				t.Fatalf("the refused order left the record %s or an open order behind", rec.State)
			}
			d.k.EquipPath = good
			if res := d.k.EquipEnqueue(context.Background(), id, "", false); !res.OK || len(d.df.Calls) != 1 {
				t.Fatalf("the next tap with a writable journal must work: %+v", res)
			}
		})
	}
}

// Review 10 N4 (card sweep2): an open order that carries an unreadable reply is settled by the bytes in the slot at the
// next repair, like one without a reply. Before: it stayed open for ever and every Equip of the record answered
// order_in_flight.
func TestSweep2OrderWithUnreadReplyIsSettledByTheLook(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
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
			d.k.setOrder(id, func(o *EquipOrder) { o.Reply = "ok equipped ?? (not parsed)" })
			d.ch.reset()
			d.df.DropReply, d.df.After = false, nil
			d.k.repairDirect(context.Background())
			if d.k.directOpen(id) {
				t.Fatal("the order is still open although the slot holds the old item")
			}
			if rec, _ := d.st.Get(id); rec.State != vault.Recorded || d.live("previously worn") != 0 {
				t.Fatal("nothing moved: the record must be back in the vault and nothing filed")
			}
			d.k.mu.Lock()
			kept := d.k.bookLocked().orders[id].Reply
			d.k.mu.Unlock()
			if kept == "" {
				t.Fatal("the reply text must stay in the order file")
			}
		})
	}
}

// Review 10 N6 (card sweep2): "this order filed it" holds for a record in any state. Before: only a live record counted,
// so a displaced item filed before a stop and sold before the repair was filed a second time.
func TestSweep2DisplacedItemIsFiledOncePerOrderInAnyState(t *testing.T) {
	ctx := context.Background()
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			d := directRig(t, class)
			rec := d2dctl.FakeRecord(3, "cap", 6, 20, "displaced")
			const tag = "eq-x-1"
			if _, where, ok := d.k.recordItem(ctx, d.st, rec, "Old Cap", tag, false); !ok || where != "vault" {
				t.Fatalf("not filed: %s %v", where, ok)
			}
			var id string
			for _, r := range d.st.Dump() {
				if r.Found.Area == tag {
					id = r.ID
				}
			}
			if err := d.st.Discard(id, "test: the player let it go"); err != nil {
				t.Fatal(err)
			}
			if _, where, ok := d.k.recordItem(ctx, d.st, rec, "Old Cap", tag, false); !ok || where != "vault" {
				t.Fatalf("the second pass: %s %v", where, ok)
			}
			if n := d.live("previously worn"); n != 0 {
				t.Fatalf("the second pass of one order filed the item again: %d live", n)
			}
			if _, _, ok := d.k.recordItem(ctx, d.st, rec, "Old Cap", "eq-y-2", false); !ok || d.live("previously worn") != 1 {
				t.Fatal("another order that displaces an equal item files it")
			}
		})
	}
}
