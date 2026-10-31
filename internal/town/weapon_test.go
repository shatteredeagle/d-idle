package town_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
)

func worn(r *rig) string {
	for _, u := range r.W.Items {
		if u.Item.Where == d2.ContEquipment && u.Item.Body == d2.BodyRHand {
			return r.W.DB.Items[u.Class].Code
		}
	}
	return ""
}

func TestUseWeaponSwapsAndRestores(t *testing.T) {
	r := newRig(t)
	sw := r.W.Add("ssd", d2.ContEquipment, 0, 0)
	sw.Item.Body = d2.BodyRHand
	r.W.Items[len(r.W.Items)-1] = sw
	r.W.Add("hax", d2.ContInventory, 4, 0)
	var during string
	err := r.T.UseWeapon(context.Background(), "hax", func(context.Context) error { during = worn(r); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if during != "hax" || worn(r) != "ssd" {
		t.Fatalf("during %q after %q", during, worn(r))
	}
	if r.W.Cursor != nil {
		t.Fatal("cursor left")
	}
}

func TestUseWeaponRestoresAfterFailure(t *testing.T) {
	r := newRig(t)
	sw := r.W.Add("ssd", d2.ContEquipment, 0, 0)
	sw.Item.Body = d2.BodyRHand
	r.W.Items[len(r.W.Items)-1] = sw
	r.W.Add("hax", d2.ContInventory, 4, 0)
	boom := errors.New("boom")
	if err := r.T.UseWeapon(context.Background(), "hax", func(context.Context) error { return boom }); !errors.Is(err, boom) {
		t.Fatal(err)
	}
	if worn(r) != "ssd" {
		t.Fatalf("worn %q", worn(r))
	}
}

func TestUseWeaponEmptySlotTakesItOff(t *testing.T) {
	r := newRig(t)
	r.W.Add("hax", d2.ContInventory, 4, 0)
	if err := r.T.UseWeapon(context.Background(), "hax", func(context.Context) error { return nil }); err != nil {
		t.Fatal(err)
	}
	if worn(r) != "" {
		t.Fatalf("worn %q", worn(r))
	}
}

func TestUseWeaponMissing(t *testing.T) {
	r := newRig(t)
	if err := r.T.UseWeapon(context.Background(), "hax", func(context.Context) error { return nil }); err == nil {
		t.Fatal("want error")
	}
}
