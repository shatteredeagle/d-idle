package vaultkeep

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/vault"
)

func kinds(fs []Finding) map[string]int {
	m := map[string]int{}
	for _, f := range fs {
		m[f.Kind]++
	}
	return m
}

// The check is read-only and reports the three things a repair would act on, for both hero classes.
func TestVault2CheckReportsAndChangesNothing(t *testing.T) {
	for _, class := range classes {
		t.Run(class, func(t *testing.T) {
			r := newSaleRig(t, class, 1000)
			dir := vault.HeroDir(r.dir, r.hero)
			r.k.SaleFile = func() string { return filepath.Join(dir, "sale.json") }
			r.k.EquipPath, r.k.CarryPath = filepath.Join(dir, "equip.json"), filepath.Join(dir, "carried.json")
			a, b, c, d := r.item(1, 2, 7, 100), r.item(2, 3, 9, 200), r.item(3, 3, 5, 50), r.item(4, 3, 5, 60)
			if fs, err := Check(dir, nil); err != nil || len(fs) != 0 {
				t.Fatalf("clean vault: %v %+v", err, fs)
			}
			// an unanswered credit: an open sale that holds its record (an intent, not an unreserved sale)
			r.pay.Fail = errors.New("timeout")
			r.sellNow(false, a)
			if got := kinds(mustCheck(t, dir, nil)); got[FindOrphanIntent] != 1 || got[FindUnreserved] != 0 || got[FindOrphanHold] != 0 {
				t.Fatalf("open sale: %v", got)
			}
			// the same order with its hold gone: credited-but-unreserved
			if _, err := r.st.Release(r.k.saleOrder().ID); err != nil {
				t.Fatal(err)
			}
			// a hold no sale owns, an open merc intent, an open carry intent, a carried charm whose record is still live
			if _, err := r.st.Reserve([]string{b}, "s-lost", 0); err != nil {
				t.Fatal(err)
			}
			must(t, r.k.mercPut(c, &mercIntent{ID: c, Order: "eqm-x-1", Name: "cap", Step: mercGive}))
			must(t, r.k.carryPut(c, &Carried{ID: c, Name: "charm", State: carryIntent}))
			must(t, r.k.carryPut(d, &Carried{ID: d, Name: "charm", Hex: r.state(d).Raw}))
			before := snapshotDir(t, dir)
			got := kinds(mustCheck(t, dir, nil))
			if got[FindUnreserved] != 1 || got[FindOrphanHold] != 1 || got[FindOrphanIntent] != 3 || got[FindDuplicate] != 2 {
				t.Fatalf("findings: %v", got)
			}
			// the hero's own items given: a live record with the same bytes is a duplicate
			if got := kinds(mustCheck(t, dir, []string{r.state(b).Raw})); got[FindDuplicate] != 2 {
				t.Fatalf("with hero items: %v", got)
			}
			if after := snapshotDir(t, dir); after != before {
				t.Fatalf("the check changed the directory:\n%s\n%s", before, after)
			}
			// the keeper's own repair clears what it may clear: the orphan hold goes, the credited order finishes
			r.pay.Fail = nil
			r.k.Run(context.Background())
			if got := kinds(mustCheck(t, dir, nil)); got[FindOrphanHold] != 0 {
				t.Fatalf("after the repair: %v", got)
			}
		})
	}
}

func mustCheck(t *testing.T, dir string, hero []string) []Finding {
	t.Helper()
	fs, err := Check(dir, hero)
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func snapshotDir(t *testing.T, dir string) string {
	t.Helper()
	es, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	s := ""
	for _, e := range es {
		b, _ := os.ReadFile(filepath.Join(dir, e.Name()))
		s += e.Name() + ":" + string(b) + "\n"
	}
	return s
}
