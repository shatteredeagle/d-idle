package readiness

import (
	"os"
	"testing"

	"d2lod-bot/internal/data"
)

// TestRealAnchors checks the generated table against docs/difficulty.md section 2.1 and 3 (V tables) when an install
// is available; it skips otherwise.
func TestRealAnchors(t *testing.T) {
	if _, err := os.Stat("/tmp/d2mpq/d2data.mpq"); err != nil {
		t.Skip("no real install at /tmp/d2mpq")
	}
	db, err := data.Open("/tmp/d2mpq", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	at, err := BuildAreaTable(db)
	if err != nil {
		t.Fatal(err)
	}
	// docs/difficulty.md section 2.1: first-area monster levels (Normal, Nightmare, Hell).
	for _, c := range []struct {
		area string
		mlvl [3]int
	}{
		{"Blood Moor", [3]int{1, 36, 67}},
	} {
		a, err := at.Find(c.area)
		if err != nil {
			t.Fatal(err)
		}
		for d := 0; d < 3; d++ {
			if a.Diff[d].MLvl != c.mlvl[d] {
				t.Errorf("%s %s mlvl %d, doc says %d", c.area, diffNames[d], a.Diff[d].MLvl, c.mlvl[d])
			}
		}
	}
}
