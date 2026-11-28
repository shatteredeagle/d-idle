package vaultkeep

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// lostVault runs the removal in the game and loses its reply once.
type lostVault struct {
	*flaky
	lose *bool
}

func (l lostVault) ItemVault(ctx context.Context, id int) (d2dctl.VaultRecord, error) {
	rec, err := l.flaky.ItemVault(ctx, id)
	if err == nil && *l.lose {
		*l.lose = false
		return d2dctl.VaultRecord{}, errLost
	}
	return rec, err
}

var fix14Kinds = map[string]string{"sorceress": "desert", "paladin": "rogue"}

// Review 14 H1: the mercenary refuses, the copy is removed in the game and the reply is lost. The copy is gone, the
// mercenary does not wear it: the vault record must stay (before: "gone = given", the record was retired).
func TestMercRefusalWithLostTakebackKeepsTheRecord(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			ctx := context.Background()
			d, mf := mercRig(t, class, fix14Kinds[class])
			mf.rule = func(string) *d2dctl.ItemRefused {
				return &d2dctl.ItemRefused{Code: d2dctl.ItemErrRequirements, Why: "level 30"}
			}
			lose := true
			d.k.API = lostVault{d.api, &lose}
			id := d.add(1, "cap", "new")
			if res, _ := d.k.EquipMerc(ctx, id); res.OK || lose {
				t.Fatalf("want a refusal whose take-back reply was lost: %+v lose=%v", res, lose)
			}
			for i := 0; i < 3; i++ {
				d.k.repairMerc(ctx)
			}
			if r, ok := d.st.Get(id); !ok || r.State != vault.Recorded {
				t.Fatalf("the vault record was lost: %+v", r)
			}
			if open := d.k.mercIntents(); len(open) != 0 {
				t.Fatalf("intent left open: %+v", open)
			}
		})
	}
}

// Review 14 H1, second half: a copy that left the pack at step give is "given" only when the mercenary's slots show
// it. A hand that cannot show them, or slots without the piece (the copy was sold or dropped), keep the record.
func TestMercCopyGoneIsNotGivenWithoutTheSlotBytes(t *testing.T) {
	for _, class := range classes {
		for _, reader := range []bool{false, true} {
			ctx := context.Background()
			d, mf := mercRig(t, class, fix14Kinds[class])
			if !reader {
				d.k.Hand = mhand{dhand: d.h, kind: fix14Kinds[class], hired: true}
			}
			id := d.add(1, "cap", "new")
			rec, _ := d.st.Get(id)
			in := mercIntent{ID: id, Order: "eqm-test", Name: rec.Name, Hex: createHex(rec), Step: mercGive}
			if err := d.k.mercPut(id, &in); err != nil {
				t.Fatal(err)
			}
			out := d.k.repairMerc(ctx)[id]
			if r, _ := d.st.Get(id); out.given || r.State != vault.Recorded {
				t.Fatalf("%s reader=%v: judged given without proof (%+v), record %v", class, reader, out, r.State)
			}
			_ = mf
		}
	}
}

// Review 14 H2: after a proven give the repair files the one new backpack piece of the slot the copy went to. A new
// piece the hero wears, and new pieces of other slots, stay with the hero.
func TestMercRepairFilesOnlyTheDisplacedPiece(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			ctx := context.Background()
			kind := fix14Kinds[class]
			d, mf := mercRig(t, class, kind)
			id := d.add(1, "cap", "new")
			rec, _ := d.st.Get(id)
			in := mercIntent{ID: id, Order: "eqm-test", Name: rec.Name, Hex: createHex(rec), Step: mercGive}
			if err := d.k.mercPut(id, &in); err != nil { // the pack snapshot is empty: everything below is new
				t.Fatal(err)
			}
			mf.worn[1] = createHex(rec) // the give ran, its reply was lost
			old := d2dctl.FakeRecord(0, "cap", 6, 20, "oldcap")
			if _, err := d.api.ItemCreate(ctx, old.Norm, d2dctl.CreateOpts{Pack: true}); err != nil { // came off the mercenary
				t.Fatal(err)
			}
			wornCap, _ := d.wearReal("head", "cap", "looted-cap") // the hero put on a looted helm
			_, armNorm := 0, d2dctl.FakeRecord(0, "arm", 6, 20, "looted-arm").Norm
			if _, err := d.api.ItemCreate(ctx, armNorm, d2dctl.CreateOpts{Pack: true}); err != nil { // loot of another slot, in the pack
				t.Fatal(err)
			}
			out := d.k.repairMerc(ctx)[id]
			if !out.given || len(out.names) != 1 {
				t.Fatalf("outcome %+v, want given with one piece filed", out)
			}
			if d.copies(old.Norm) != 0 || d.liveBytes(old.Norm) != 1 {
				t.Fatalf("the displaced piece: pack %d vault %d, want 0 and 1", d.copies(old.Norm), d.liveBytes(old.Norm))
			}
			if d.copies(armNorm) != 1 || d.liveBytes(armNorm) != 0 {
				t.Fatalf("the hero's looted armour was filed: pack %d vault %d", d.copies(armNorm), d.liveBytes(armNorm))
			}
			if d.h.WornUnit("head") != uint32(wornCap) {
				t.Fatal("the hero's worn helm was touched")
			}
			for _, r := range d.st.Dump() {
				if r.State == vault.Recorded && r.Raw != old.Norm && r.Found.Area == "eqm-test" {
					t.Fatalf("a record of the hero's own gear was written: %+v", r.Item)
				}
			}
		})
	}
}
