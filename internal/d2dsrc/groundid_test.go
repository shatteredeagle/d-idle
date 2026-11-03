package d2dsrc

import (
	"context"
	"testing"
)

// A ground item and a carried item can share a unit id (loot and character count separately, both from 1; docs/tomestat.md).
// The carried item's quantity must survive: this is the Tome of Town Portal losing stat 70 in the field.
func TestGroundIDDoesNotShadowCarriedItem(t *testing.T) {
	r := lateReplies(t)
	r["ground"] = "gld\t8800 Gold\t371\t331\t#11\t40.9\t11.5\nhp1\tMinor Healing Potion\t300\t300\t#9\t41.0\t12.0\nok\n"
	_, c := startScript(t, r)
	res, err := New(c, Options{Tables: fakeTables{}, Logf: quiet}).Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if ex := res.Extras.Items[11]; ex.Qty != 20 || ex.Code != "tsc" {
		t.Errorf("carried id 11 lost by a ground item with the same id: %+v", ex)
	}
	if ex := res.Extras.Items[9]; ex.Code != "wnd" {
		t.Errorf("carried id 9 shadowed: %+v", ex)
	}
	if ex := res.Extras.Ground[11]; ex.Code != "gld" {
		t.Errorf("ground extras %+v", res.Extras.Ground)
	}
	if len(res.Snap.Ground) != 2 {
		t.Errorf("ground %d", len(res.Snap.Ground))
	}
}
