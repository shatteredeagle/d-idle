package vaultkeep

import (
	"context"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/vault"
)

// Card vault2: the paths card fix1 left without a test (Wear set failures, twin Keep) and the two-hander's shield on a
// lost reply. Every test runs on both hero classes and MUST FAIL when an item can vanish or double.

// wearReal puts a readable unit on the body: the game's own worn item, known to the hand, the direct fake and the decoder.
func (d *drig) wearReal(slot, code, extra string) (unit int, norm string) {
	unit = d.h.wear(slot, code, extra)
	for _, c := range d.api.VaultFake.Carried {
		if c.Rec.UnitID == unit {
			norm = c.Rec.Norm
		}
	}
	d.df.Worn[slot] = norm
	if d.api.VaultFake.Bytes == nil {
		d.api.VaultFake.Bytes = map[string]d2dctl.VaultRecord{}
	}
	d.api.VaultFake.Bytes[norm] = d2dctl.FakeRecord(0, code, 6, 20, extra)
	return unit, norm
}

// gone is the engine side of a swap: the units leave the body and the game, the new item is worn.
func (d *drig) gone(units ...int) func([]d2dctl.DirectResult) {
	return func([]d2dctl.DirectResult) {
		for _, u := range units {
			d.api.VaultFake.Remove(u)
			delete(d.h.body, u)
		}
		d.mirror() // and the new item is worn (card fix1b: the verdict reads the slot's bytes)
	}
}

func (d *drig) retired(ids ...string) bool {
	for _, id := range ids {
		if r, _ := d.st.Get(id); r.State == vault.Recorded {
			return false
		}
	}
	return true
}

func TestVault2TwoHanderLostReplyFilesWeaponAndShield(t *testing.T) {
	for _, class := range classes {
		t.Run(class+"/weapon and shield", func(t *testing.T) {
			d := directRig(t, class)
			w, _ := d.wearReal("rhand", "wsp", "oldweapon")
			s, _ := d.wearReal("lhand", "shd", "oldshield")
			id := d.add(1, "two", "two")
			d.df.DropReply, d.df.After = true, d.gone(w, s)
			res := d.k.EquipEnqueue(context.Background(), id, "", false)
			if d.live("previously worn") != 2 || !d.retired(id) {
				t.Fatalf("lost reply: %d filed, record retired %v: %+v", d.live("previously worn"), d.retired(id), res)
			}
			d.df.DropReply = false
			if res = d.k.EquipEnqueue(context.Background(), id, "", false); res.OK || len(d.df.Calls) != 1 {
				t.Fatalf("second tap swapped again: %+v %v", res, d.df.Calls)
			}
		})
		t.Run(class+"/empty hand and shield", func(t *testing.T) {
			d := directRig(t, class)
			s, _ := d.wearReal("lhand", "shd", "oldshield")
			id := d.add(1, "two", "two")
			d.df.DropReply, d.df.After = true, d.gone(s)
			d.k.EquipEnqueue(context.Background(), id, "", false)
			if d.live("previously worn") != 1 || !d.retired(id) {
				t.Fatalf("the shield was not filed: %d retired %v", d.live("previously worn"), d.retired(id))
			}
		})
		t.Run(class+"/shield not judged stays open", func(t *testing.T) {
			d := directRig(t, class)
			w, _ := d.wearReal("rhand", "wsp", "oldweapon")
			s, _ := d.wearReal("lhand", "shd", "oldshield")
			id := d.add(1, "two", "two")
			d.df.DropReply = true
			d.df.After = func(r []d2dctl.DirectResult) { d.gone(w)(r); delete(d.h.body, s) } // the shield: off the body, still readable
			d.k.EquipEnqueue(context.Background(), id, "", false)
			d.df.DropReply = false
			if res := d.k.EquipEnqueue(context.Background(), id, "", false); res.Code != CodeInFlight || len(d.df.Calls) != 1 {
				t.Fatalf("an unjudged two-hander order was replaced: %+v %v", res, d.df.Calls)
			}
		})
	}
}

func TestVault2WearSetFailuresAfterTheSwap(t *testing.T) {
	for _, class := range classes {
		t.Run(class+"/late failure", func(t *testing.T) {
			d := directRig(t, class)
			d.wearOld("head", "cap", "Old Cap")
			d.wearOld("torso", "arm", "Old Armor")
			a, b := d.add(1, "cap", "s1"), d.add(2, "arm", "s2")
			d.df.LateFail = errors.New("ledger: disk full")
			out := d.k.EquipSet(context.Background(), []string{a, b}, true)
			if !out[0].OK || !out[1].OK || d.live("previously worn") != 2 || !d.retired(a, b) {
				t.Fatalf("late failure of a set: %+v filed %d", out, d.live("previously worn"))
			}
		})
		t.Run(class+"/lost reply", func(t *testing.T) {
			d := directRig(t, class)
			h, _ := d.wearReal("head", "cap", "oldcap")
			w, _ := d.wearReal("rhand", "wsp", "oldweapon")
			s, _ := d.wearReal("lhand", "shd", "oldshield")
			a, b, c := d.add(1, "cap", "s1"), d.add(2, "wsp", "s2"), d.add(3, "shd", "s3")
			d.df.DropReply, d.df.After = true, d.gone(h, w, s)
			d.k.EquipSet(context.Background(), []string{a, b, c}, true)
			// three old pieces, each filed once although both hand orders read the other hand before the call
			if d.live("previously worn") != 3 || !d.retired(a, b, c) {
				t.Fatalf("lost reply of a set: filed %d, retired %v", d.live("previously worn"), d.retired(a, b, c))
			}
			d.df.DropReply = false
			if out := d.k.EquipSet(context.Background(), []string{a, b, c}, true); len(d.df.Calls) != 1 || out[0].OK {
				t.Fatalf("second tap swapped again: %+v %v", out, d.df.Calls)
			}
		})
		t.Run(class+"/one piece refused, late failure", func(t *testing.T) {
			d := directRig(t, class)
			d.wearOld("head", "cap", "Old Cap")
			a, b := d.add(1, "cap", "s1"), d.add(2, "arm", "strfail")
			d.df.LateFail = errors.New("ledger: disk full")
			out := d.k.EquipSet(context.Background(), []string{a, b}, true)
			if !out[0].OK || out[1].OK || d.live("previously worn") != 1 || !d.retired(a) || d.retired(b) {
				t.Fatalf("%+v filed %d", out, d.live("previously worn"))
			}
		})
		t.Run(class+"/vault full", func(t *testing.T) {
			d := directRig(t, class)
			d.wearOld("head", "cap", "Old Cap")
			d.wearOld("torso", "arm", "Old Armor")
			a, b := d.add(1, "cap", "s1"), d.add(2, "arm", "s2")
			d.cap = 2
			before := len(d.api.VaultFake.Carried)
			out := d.k.EquipSet(context.Background(), []string{a, b}, true)
			onHero := len(d.api.VaultFake.Carried) - before
			if worn := okCount(out); worn > 0 && d.live("previously worn")+onHero != worn {
				t.Fatalf("%d pieces worn, %d old pieces filed and %d put back on the hero: %+v", worn, d.live("previously worn"), onHero, out)
			}
		})
	}
}

// Twin Keep (review finding F6): a kept ring was displaced and filed again as a second record with the same bytes; its
// Keep is in flight when the bot restarts. The item is never created a second time, and a Keep the bot cannot judge by
// counting ends visibly.
func TestVault2TwinKeepNeverCreatesTwice(t *testing.T) {
	for _, class := range classes {
		for _, created := range []bool{false, true} {
			name := class + "/not created"
			if created {
				name = class + "/created before the stop"
			}
			t.Run(name, func(t *testing.T) {
				ctx := context.Background()
				r := newRig(t, class)
				a := r.add(1, "rin", 3)
				r.enq(a)
				r.run()
				recA := r.state(a)
				if recA.State != vault.Kept || len(r.api.VaultFake.Carried) != 1 {
					t.Fatalf("setup keep: %s", recA.State)
				}
				// the ring was worn and displaced: it left the hero and came back as a second record
				r.api.VaultFake.Remove(r.api.VaultFake.Carried[0].Rec.UnitID)
				twin, err := r.st.Add(vault.Item{Raw: recA.Raw, Code: "rin", Name: recA.Name, Quality: 3, Ilvl: recA.Ilvl, Identified: true,
					Found: vault.Found{Zone: zoneWorn, Area: "eq-x-1"}})
				must(t, err)
				r.enq(twin.ID)
				must(t, r.st.BeginKeep(twin.ID))
				if created {
					if _, err := r.api.VaultFake.ItemCreate(ctx, createHex(r.state(twin.ID)), d2dctl.CreateOpts{}); err != nil {
						t.Fatal(err)
					}
				}
				r.open()
				for i := 0; i < 3; i++ {
					r.run()
				}
				copies := 0
				for _, c := range r.api.VaultFake.Carried {
					if strings.EqualFold(c.Rec.Norm, createHex(recA)) {
						copies++
					}
				}
				got := r.state(twin.ID)
				if copies != 1 || got.InFlight {
					t.Fatalf("the ring exists %d times on the hero, twin %+v", copies, got)
				}
				if created && got.State != vault.Kept && !strings.Contains(strings.Join(r.notes, "|"), "identical") && Reason(got.Reason) != ReasonTwin {
					t.Fatalf("an unjudged twin keep ended silently: %+v notes %v", got, r.notes)
				}
				if !created && got.State != vault.Kept {
					t.Fatalf("a twin that was never created was not kept: %+v", got)
				}
			})
		}
	}
}
