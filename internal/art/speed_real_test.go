package art

import (
	"sort"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/data/mpq"
)

// realItemKeys lists n item-art keys that exist in the real install, spread over the archive.
func realItemKeys(t testing.TB, dir string, n int) []string {
	t.Helper()
	p, err := findFold(dir, "d2data.mpq")
	if err != nil {
		t.Skip("no d2data.mpq")
	}
	a, err := mpq.Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	var all []string
	for _, f := range a.ListFile() {
		f = strings.ToLower(f)
		if rest, ok := strings.CutPrefix(f, `data\global\items\inv`); ok && strings.HasSuffix(rest, ".dc6") {
			k := "item-inv" + strings.TrimSuffix(rest, ".dc6")
			if itemKey.MatchString(k) {
				all = append(all, k)
			}
		}
	}
	sort.Strings(all)
	if len(all) < n {
		t.Skipf("only %d icons", len(all))
	}
	var out []string
	for i := 0; i < n; i++ {
		out = append(out, all[i*len(all)/n])
	}
	return out
}

// TestRealColdWarm prints where the time goes for a typical 30 icon inventory (read-only on the install).
func TestRealColdWarm(t *testing.T) {
	dir := realInstall(t)
	keys := append([]string{"panel-inv"}, realItemKeys(t, dir, 30)...)
	cache := t.TempDir()

	t0 := time.Now()
	s, err := Open(dir, cache)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Open (archives, patch sha, palette): %v", time.Since(t0))

	var total time.Duration
	for _, k := range keys {
		t1 := time.Now()
		if _, err := s.PNG(k); err != nil {
			t.Logf("%s: %v", k, err)
			continue
		}
		d := time.Since(t1)
		total += d
		if k == "panel-inv" || d > 5*time.Millisecond {
			t.Logf("cold %s %v", k, d)
		}
	}
	t.Logf("cold %d keys: %v", len(keys), total)

	t1 := time.Now()
	for _, k := range keys {
		s.PNG(k)
	}
	t.Logf("warm memory %d keys: %v", len(keys), time.Since(t1))
	s.Close()

	t2 := time.Now()
	s2, _ := Open(dir, cache)
	t.Logf("reopen: %v", time.Since(t2))
	t3 := time.Now()
	for _, k := range keys {
		s2.PNG(k)
	}
	t.Logf("warm disk %d keys: %v", len(keys), time.Since(t3))
	s2.Close()
}
