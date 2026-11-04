package town_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/gear/geartest"
)

// tomefix (docs/tomefix.md): the town half of the verdict. NeedsTrip raises "no town portal scroll" only when
// scrollCount (town.qty) reads the pack as empty; a tome with charges must never do that.
func tomeTripRig(t *testing.T) (*rig, d2.Unit) {
	t.Helper()
	geartest.Need(t)
	r := emptySourceRig(t)
	var tome d2.Unit
	for _, u := range r.W.Items {
		if r.W.DB.Items[u.Class].Code == "tbk" {
			tome = u
		}
	}
	return r, tome
}

func wantNoScrollTrip(t *testing.T, r *rig, want bool, label string) {
	t.Helper()
	s, err := r.W.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	ok, why := r.T.NeedsTrip(s)
	if got := ok && why == "no town portal scroll"; got != want {
		t.Errorf("%s: trip=%v %q, want scroll trip %v", label, ok, why, want)
	}
}

func TestTomeFixTownTomeWithChargesRaisesNoTrip(t *testing.T) {
	r, tome := tomeTripRig(t)
	r.W.Stats[tome.Addr] = d2.Stats{70: 13}
	wantNoScrollTrip(t, r, false, "13 charges")
}

// Stale or shifted snapshot: the stats come from another look, the cell differs. Keyed by address, not cell.
func TestTomeFixTownShiftedTomeStillCounts(t *testing.T) {
	r, tome := tomeTripRig(t)
	r.W.Stats[tome.Addr] = d2.Stats{70: 13}
	for i := range r.W.Items {
		if r.W.Items[i].Addr == tome.Addr {
			r.W.Items[i].X, r.W.Items[i].Y = 0, 0
			r.W.Items[i].Item.GX, r.W.Items[i].Item.GY = 0, 0
		}
	}
	wantNoScrollTrip(t, r, false, "moved cell")
}

// A scroll next to an unreadable tome is a source on its own (tpTool/scrollCount count the scroll).
func TestTomeFixTownScrollBesideEmptyTome(t *testing.T) {
	r, tome := tomeTripRig(t)
	r.W.Stats[tome.Addr] = d2.Stats{} // empty tome
	r.W.Add("tsc", d2.ContInventory, 8, 3)
	wantNoScrollTrip(t, r, false, "scroll + empty tome")
}

// Contract: the tome whose stats cannot be read, or has no stat 70, IS empty to the town (real empty tome looks the
// same). So the fix for a dropout has to be on the producer side; the town cannot tell.
func TestTomeFixTownNoStatReadsEmpty(t *testing.T) {
	r, tome := tomeTripRig(t)
	delete(r.W.Stats, tome.Addr)
	wantNoScrollTrip(t, r, true, "no stat list")
}
