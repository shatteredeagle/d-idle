package vaultkeep

import (
	"context"
	"fmt"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

func carryRig(t *testing.T, class string) (*rig, *fakeHand) {
	r, h := equipRig(t, class)
	r.k.CarryPath = r.dir + "/carried.json"
	r.api.VaultFake.Sizes = map[string][2]int{"cm1": {1, 1}, "cm2": {1, 2}, "cm3": {1, 3}}
	return r, h
}

func (r *rig) charm(i int, code string, q int, identified bool) string {
	r.t.Helper()
	rec := d2dctl.FakeRecord(i, code, q, 30, fmt.Sprintf("charm%d", i))
	if r.api.VaultFake.Bytes == nil {
		r.api.VaultFake.Bytes = map[string]d2dctl.VaultRecord{}
	}
	r.api.VaultFake.Bytes[rec.Norm] = rec // `item decode` knows these bytes
	it, err := r.st.Add(vault.Item{Raw: rec.Norm, Code: code, Name: fmt.Sprintf("%s %d", code, i), Quality: q, Ilvl: 30, Identified: identified})
	if err != nil {
		r.t.Fatal(err)
	}
	return it.ID
}

// MUST FAIL if Carry leaves the record in the vault (two copies), skips the keep mark, or Stow loses the item.
func TestCarryAndStow(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r, h := carryRig(t, class)
			ctx := context.Background()
			ids := []string{r.charm(1, "cm1", 4, true), r.charm(2, "cm3", 7, true)}
			for _, id := range ids {
				if res := r.k.Carry(ctx, id); !res.OK {
					t.Fatalf("carry %s: %+v", id, res)
				}
				if rec, ok := r.st.Get(id); ok && rec.State.Live() {
					t.Fatalf("%s is still a live record", id)
				}
			}
			if r.carried(d2dctl.FakePack) != 2 || len(h.protected) != 2 {
				t.Fatalf("pack %d protected %d", r.carried(d2dctl.FakePack), len(h.protected))
			}
			list := r.k.CarriedList()
			if len(list) != 2 || list[1].Cells != 3 || list[0].Cells != 1 {
				t.Fatalf("carried %+v", list)
			}
			// the book survives a restart
			r.open()
			r.k.Hand, r.k.CarryPath = h, r.dir+"/carried.json"
			if len(r.k.CarriedList()) != 2 {
				t.Fatalf("book lost on restart: %+v", r.k.CarriedList())
			}
			res := r.k.Stow(ctx, ids[1])
			if !res.OK || res.Where != "vault" || r.carried(d2dctl.FakePack) != 1 || len(r.k.CarriedList()) != 1 {
				t.Fatalf("stow %+v pack %d notes %v", res, r.carried(d2dctl.FakePack), r.notes)
			}
			if n := r.st.Status().Count; n != 1 {
				t.Fatalf("vault holds %d records, want the stowed one", n)
			}
			if again := r.k.Stow(ctx, ids[1]); again.OK || again.Code != "missing" {
				t.Fatalf("a second stow: %+v", again)
			}
		})
	}
}

func TestCarryRefusals(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r, _ := carryRig(t, class)
			ctx := context.Background()
			ring := r.addID(1, "rin")
			unid := r.charm(2, "cm1", 4, false)
			rare := r.charm(3, "cm1", 6, true) // no rare charm exists
			for id, code := range map[string]string{ring: "kind", unid: "reveal", rare: "kind", "nope": "missing"} {
				if res := r.k.Carry(ctx, id); res.OK || res.Code != code {
					t.Errorf("%s: %+v, want %s", id, res, code)
				}
			}
			if r.carried(d2dctl.FakePack) != 0 {
				t.Error("a refused carry created something")
			}
			r.k.Measured = func() bool { return true }
			if res := r.k.Carry(ctx, r.charm(4, "cm2", 4, true)); res.OK || res.Code != "off" {
				t.Errorf("measured run: %+v", res)
			}
		})
	}
}

// A full backpack refuses with room and changes nothing; the record stays in the vault.
func TestCarryNeedsAFreeCell(t *testing.T) {
	r, _ := carryRig(t, "sorceress")
	ctx := context.Background()
	for i := 0; i < 40; i++ { // fill the 10x4 pack with 1x1 items
		rec := d2dctl.FakeRecord(0, "cm1", 4, 20, fmt.Sprint("fill", i))
		if _, err := r.api.ItemCreate(ctx, rec.Norm, d2dctl.CreateOpts{Pack: true}); err != nil {
			t.Fatal(err)
		}
	}
	id := r.charm(1, "cm3", 4, true)
	res := r.k.Carry(ctx, id)
	if res.OK || res.Code != "room" {
		t.Fatalf("%+v", res)
	}
	if rec, _ := r.st.Get(id); rec.State != vault.Recorded || len(r.k.CarriedList()) != 0 {
		t.Fatalf("record %s carried %v", rec.State, r.k.CarriedList())
	}
}

// Sell all never takes a carried charm (it is no vault record) and never a pinned one (the player's keep mark in the vault).
func TestSellAllLeavesCarriedAndPinnedCharms(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r := newSellAllRig(t, class, 1000)
			h := newHand(r.api.VaultFake, class)
			r.k.Hand, r.k.CarryPath = h, r.dir+"/carried.json"
			ctx := context.Background()
			pinned := r.charm(21, "cm1", 4, true)
			if err := r.st.Pin(pinned); err != nil {
				t.Fatal(err)
			}
			if res := r.k.Carry(ctx, r.charm(22, "cm2", 4, true)); !res.OK {
				t.Fatalf("carry: %+v", res)
			}
			out := r.sellAll("tap-charm")
			if out.View == nil || out.View.Done != 4 {
				t.Fatalf("sale %+v", out)
			}
			if g, ok := r.st.Get(pinned); !ok || g.State != vault.Recorded {
				t.Fatalf("pinned record %v", g.State)
			}
			if len(r.k.CarriedList()) != 1 || r.carried(d2dctl.FakePack) != 1 {
				t.Fatalf("the carried charm was touched: %v pack %d", r.k.CarriedList(), r.carried(d2dctl.FakePack))
			}
		})
	}
}
