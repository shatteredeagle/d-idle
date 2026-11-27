package vault

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The batch sale leaves exactly what the same sales one by one leave (state, archive lines, replay after a reopen), skips a
// record that cannot be sold and says so.
func TestSellNowBatchMatchesSingle(t *testing.T) {
	fill := func(dir string) (*Store, []SoldLine) {
		s := open(t, dir, "sorc", 50)
		var ls []SoldLine
		for i := 1; i <= 12; i++ {
			r, err := s.Add(mk(i, "cap", 4, 10))
			if err != nil {
				t.Fatal(err)
			}
			ls = append(ls, SoldLine{ID: r.ID, Gold: 10 * i})
		}
		return s, ls
	}
	d1, d2 := t.TempDir(), t.TempDir()
	a, la := fill(d1)
	b, lb := fill(d2)
	if err := b.Pin(lb[3].ID); err != nil {
		t.Fatal(err)
	}
	if err := a.Pin(la[3].ID); err != nil {
		t.Fatal(err)
	}
	for _, l := range la {
		_ = a.SellNow(l.ID, l.Gold, "o1") // the pinned one fails
	}
	done, failed, err := b.SellNowBatch(append(lb, SoldLine{ID: "v-nope"}), "o1")
	if done != 11 || failed != 2 || err == nil {
		t.Fatalf("batch: done %d failed %d err %v", done, failed, err)
	}
	if dump(a) != dump(b) {
		t.Fatalf("state differs:\n%s\n%s", dump(a), dump(b))
	}
	ar := func(d string) string { x, _ := os.ReadFile(filepath.Join(d, "sorc", ArchiveFile)); return string(x) }
	if got, want := strings.Count(ar(d2), "\n"), strings.Count(ar(d1), "\n"); got != want || got != 11 {
		t.Fatalf("archive lines %d, want %d", got, want)
	}
	a.Close()
	b.Close()
	b2 := open(t, d2, "sorc", 50)
	a2 := open(t, d1, "sorc", 50)
	if dump(a2) != dump(b2) || b2.Status().Count != 1 {
		t.Fatalf("replay differs or live != 1: %+v", b2.Status())
	}
}
