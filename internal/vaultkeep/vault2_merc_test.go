package vaultkeep

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Card vault2: Equip on the mercenary with a failure injected after every game step, on two hero classes and two merc
// kinds. MUST FAIL when the given piece or the piece that came off can vanish or exist twice.

// cmfake puts the give behind the same chaos gate as the vault verbs.
type cmfake struct {
	*mfake
	c *chaos
}

func (m cmfake) MercEquipDirect(ctx context.Context, id uint32, order string) (d2dctl.ItemResult, error) {
	run, lost := m.c.gate()
	if !run {
		return d2dctl.ItemResult{}, errLost
	}
	r, err := m.mfake.MercEquipDirect(ctx, id, order)
	if lost {
		return d2dctl.ItemResult{}, errLost
	}
	return r, err
}

func (d *drig) copies(norm string) (n int) {
	for _, c := range d.api.VaultFake.Carried {
		if strings.EqualFold(c.Rec.Norm, norm) {
			n++
		}
	}
	return n
}

func (d *drig) liveBytes(norm string) (n int) {
	for _, r := range d.st.Dump() {
		if r.State.Live() && strings.EqualFold(r.Raw, norm) {
			n++
		}
	}
	return n
}

func TestVault2MercStopAfterEveryStep(t *testing.T) {
	kinds := map[string]string{"sorceress": "desert", "paladin": "rogue"}
	for _, class := range classes {
		for _, lost := range []bool{false, true} {
			for limit := 0; limit <= 6; limit++ {
				t.Run(fmt.Sprintf("%s/lost%v/%d", class, lost, limit), func(t *testing.T) {
					ctx := context.Background()
					d, mf := mercRig(t, class, kinds[class])
					wire := func() {
						d.k.Direct = cmfake{mf, d.ch}
						d.k.Hand = mhand{dhand: d.h, kind: kinds[class], hired: true, worn: mf.worn}
					}
					wire()
					old := d2dctl.FakeRecord(0, "cap", 6, 20, "oldcap")
					mf.worn[1] = old.Norm
					id := d.add(1, "cap", "new")
					piece := createHex(mustRec(t, d, "new"))
					d.ch.limit, d.ch.lost = limit, lost
					res, _ := d.k.EquipMerc(ctx, id) // dies somewhere
					d.ch.limit = 1 << 30
					d.reopen() // the bot restarts; the game and its mercenary do not
					wire()
					for i := 0; i < 3; i++ {
						d.k.repairDirect(ctx)
					}
					worn := 0
					if strings.EqualFold(mf.worn[1], piece) {
						worn = 1
					}
					if n := worn + d.copies(piece) + d.liveBytes(piece); n != 1 {
						t.Fatalf("the piece exists %d times (worn %d, pack %d, vault %d) after %+v", n, worn, d.copies(piece), d.liveBytes(piece), res)
					}
					oldWorn := 1 - worn
					if n := oldWorn + d.copies(old.Norm) + d.liveBytes(old.Norm); n != 1 {
						t.Fatalf("the old piece exists %d times (worn %d, pack %d, vault %d)", n, oldWorn, d.copies(old.Norm), d.liveBytes(old.Norm))
					}
					if worn == 1 && d.copies(old.Norm) != 0 {
						t.Fatal("the displaced piece was left in the backpack with no record")
					}
					if open := d.k.mercIntents(); len(open) != 0 {
						t.Fatalf("intent left open: %+v", open)
					}
					for _, r := range d.st.Dump() {
						if r.State == vault.Recorded && strings.EqualFold(r.Raw, old.Norm) && !strings.HasPrefix(r.Found.Area, "eqm-") {
							t.Fatalf("displaced record not filed under its order id: %+v", r.Found)
						}
					}
				})
			}
		}
	}
}

// A lost reply of the give in a process that lives on: settled at once by looking, and a second tap gives nothing twice.
func TestVault2MercLostReplySettledAndSecondTap(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			ctx := context.Background()
			d, mf := mercRig(t, class, "rogue")
			old := d2dctl.FakeRecord(0, "cap", 6, 20, "oldcap")
			mf.worn[1] = old.Norm
			id := d.add(1, "cap", "new")
			drop := true
			d.k.Direct = dropOnce{mf, &drop}
			res, _ := d.k.EquipMerc(ctx, id)
			if !res.OK || len(res.Removed) != 1 || res.Where != "vault" || d.live("previously worn") != 1 || d.copies(old.Norm) != 0 {
				t.Fatalf("lost reply not settled: %+v worn-records %d pack %d", res, d.live("previously worn"), d.copies(old.Norm))
			}
			if res, _ := d.k.EquipMerc(ctx, id); res.OK || mf.calls != 1 {
				t.Fatalf("second tap gave again: %+v calls %d", res, mf.calls)
			}
		})
	}
}

type dropOnce struct {
	*mfake
	drop *bool
}

func (m dropOnce) MercEquipDirect(ctx context.Context, id uint32, order string) (d2dctl.ItemResult, error) {
	r, err := m.mfake.MercEquipDirect(ctx, id, order)
	if *m.drop {
		*m.drop = false
		return d2dctl.ItemResult{}, errLost
	}
	return r, err
}

// The intent cannot be written: nothing is created. A measured run leaves an open intent alone (no game write).
func TestVault2MercJournalFirstAndMeasured(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			ctx := context.Background()
			d, mf := mercRig(t, class, "rogue")
			id := d.add(1, "cap", "new")
			blocker := filepath.Join(d.dir, "blocker")
			if err := os.WriteFile(blocker, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			good := d.k.EquipPath
			d.k.EquipPath = filepath.Join(blocker, "equip.json")
			res, _ := d.k.EquipMerc(ctx, id)
			if res.OK || res.Code != "journal" || d.api.calls != 0 || mf.calls != 0 {
				t.Fatalf("moved without a journal: %+v creates %d gives %d", res, d.api.calls, mf.calls)
			}
			d.k.EquipPath = good
			d.k.mb = mercBook{}
			// an intent a stop left (copy in the pack), then a measured run starts
			rec := mustRec(t, d, "new")
			if _, err := d.api.VaultFake.ItemCreate(ctx, createHex(rec), d2dctl.CreateOpts{Pack: true}); err != nil {
				t.Fatal(err)
			}
			must(t, d.k.mercPut(id, &mercIntent{ID: id, Order: "eqm-x-1", Name: "cap", Hex: createHex(rec), Step: mercCreate}))
			d.k.Measured = func() bool { return true }
			d.k.repairDirect(ctx)
			if len(d.k.mercIntents()) != 1 || d.copies(createHex(rec)) != 1 || len(d.api.removedID) != 0 {
				t.Fatal("the repair wrote the game during a measured run")
			}
			if res, ok := d.k.EquipMerc(ctx, id); ok && res.OK {
				t.Fatalf("a measured run equipped the mercenary: %+v", res)
			}
			d.k.Measured = nil
			d.k.repairDirect(ctx)
			if len(d.k.mercIntents()) != 0 || d.copies(createHex(rec)) != 0 || d.liveBytes(createHex(rec)) != 1 {
				t.Fatal("the stray copy was not taken back after the measured run")
			}
		})
	}
}
