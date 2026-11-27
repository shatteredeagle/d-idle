package watch

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"d2lod-bot/internal/vault"
)

// The vault API at the size of a real vault (capacity 400..2400): what one poll (limit=1, every 6 s) and one page cost on the
// server and on the wire. See docs/probes/vault-perf-2026-11-23.md.
func benchVault(b *testing.B, n int, path string) {
	b.Helper()
	s := testVault(b, n, n+10)
	h := massAPI(b, s, &stubActor{st: s, free: true}, true).Handler()
	b.ReportAllocs()
	b.ResetTimer()
	var size int
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest("GET", path, nil))
		size = w.Body.Len()
	}
	b.ReportMetric(float64(size), "bytes")
}

func BenchmarkVaultPoll(b *testing.B) {
	for _, n := range []int{400, 2400, 5000, 20000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) { benchVault(b, n, "/api/vault?limit=1") })
	}
}

func BenchmarkVaultPage(b *testing.B) {
	for _, n := range []int{400, 2400, 5000, 20000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) { benchVault(b, n, "/api/vault?limit=200&facets=1") })
	}
}

func gzGet(h http.Handler, path string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("GET", path, nil)
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// Must-fail guards: the Vault's files are validated (304) and gzipped, a page of 200 rows stays small on the wire, a poll stays tiny.
func TestVaultWireWeight(t *testing.T) {
	s := testVault(t, 2400, 2500)
	h := massAPI(t, s, &stubActor{st: s, free: true}, true).Handler()
	gz := map[string]string{"Accept-Encoding": "gzip"}

	for _, p := range []string{"/watch/vault.js", "/watch/vault-logic.js", "/watch/vault.css", "/watch/vault.json"} {
		w := gzGet(h, p, gz)
		if w.Header().Get("Content-Encoding") != "gzip" || w.Header().Get("ETag") == "" || w.Header().Get("Cache-Control") == "no-store" {
			t.Errorf("%s: encoding %q etag %q cache %q", p, w.Header().Get("Content-Encoding"), w.Header().Get("ETag"), w.Header().Get("Cache-Control"))
		}
		raw := gzGet(h, p, nil)
		if raw.Header().Get("Content-Encoding") != "" || raw.Body.Len() <= w.Body.Len() {
			t.Errorf("%s: plain %d bytes, gzip %d", p, raw.Body.Len(), w.Body.Len())
		}
		if again := gzGet(h, p, map[string]string{"If-None-Match": w.Header().Get("ETag")}); again.Code != http.StatusNotModified || again.Body.Len() != 0 {
			t.Errorf("%s: revalidation answered %d with %d bytes", p, again.Code, again.Body.Len())
		}
	}

	page := gzGet(h, "/api/vault?limit=200&facets=1", gz)
	if page.Header().Get("Content-Encoding") != "gzip" {
		t.Fatal("the list is not gzipped")
	}
	wire := page.Body.Len()
	zr, err := gzip.NewReader(page.Body)
	if err != nil {
		t.Fatal(err)
	}
	plain, _ := io.ReadAll(zr)
	var v VaultView
	if err := json.Unmarshal(plain, &v); err != nil || len(v.Items) != 200 {
		t.Fatalf("decoded list: %v, %d items", err, len(v.Items))
	}
	t.Logf("200 rows: %d bytes plain, %d gzip", len(plain), wire)
	if len(plain) > 60_000 || wire > 8_000 {
		t.Errorf("a page of 200 rows weighs %d plain, %d gzip (ceilings 60000, 8000)", len(plain), wire)
	}
	if poll := gzGet(h, "/api/vault?limit=1", nil); poll.Body.Len() > 4_000 {
		t.Errorf("a poll weighs %d bytes (ceiling 4000)", poll.Body.Len())
	}
}

// A poll (every 6 s per open page) must not scale into milliseconds with the vault; the ceiling is ten times today's number.
func TestVaultPollCost(t *testing.T) {
	s := testVault(t, 2400, 2500)
	h := massAPI(t, s, &stubActor{st: s, free: true}, true).Handler()
	res := testing.Benchmark(func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			gzGet(h, "/api/vault?limit=1", nil)
		}
	})
	t.Logf("poll at 2400 records: %d ns/op", res.NsPerOp())
	if raceEnabled {
		t.Skip("the 10 ms ceiling is a plain-build number; the race detector makes a poll 5 to 10 times slower (12 to 14 ms on the builder, 2026-11-24)")
	}
	if res.NsPerOp() > 10_000_000 {
		t.Errorf("a poll costs %d ns at 2400 records (ceiling 10 ms)", res.NsPerOp())
	}
}

// The poll-time cache never serves a stale preview or facet list: after every kind of write the cached answer equals a
// freshly bound (uncached) desk's.
func TestVaultPollCacheFollowsWrites(t *testing.T) {
	s := testVault(t, 60, 200)
	a := &stubActor{st: s, free: true}
	cached := massAPI(t, s, a, true).Handler()
	same := func(step string) {
		t.Helper()
		for _, p := range []string{"/api/vault?limit=1", "/api/vault?limit=1&facets=1"} {
			got := gzGet(cached, p, nil).Body.String()
			want := gzGet(massAPI(t, s, a, true).Handler(), p, nil).Body.String()
			if got != want {
				t.Fatalf("%s %s: cached answer differs\n got %s\nwant %s", step, p, got, want)
			}
		}
	}
	same("start")
	same("again") // a hit
	recs, _ := s.Query(vault.Filter{Identified: ptrBool(false)})
	if len(recs) < 3 {
		t.Fatalf("fixture has %d unidentified", len(recs))
	}
	if err := s.Pin(recs[0].ID); err != nil {
		t.Fatal(err)
	}
	same("pin")
	if err := s.Discard(recs[1].ID, "test"); err != nil {
		t.Fatal(err)
	}
	same("discard")
	v0 := s.Version()
	if err := s.MarkSell(recs[2].ID); err != nil {
		t.Fatal(err)
	}
	if s.Version() == v0 {
		t.Fatal("a write left the store version unchanged")
	}
	same("sellq")
	a.free, a.charges = false, 1 // the price input changes with no store write
	same("price")
}

// A page read from the matching end equals the full sort, for every offset and both directions.
func TestVaultQueryWindowMatchesFullSort(t *testing.T) {
	s := testVault(t, 90, 200)
	for _, order := range []string{"", "new", "old"} {
		for _, f := range []vault.Filter{{}, {Quality: []int{3}}, {Type: "rune"}} {
			full := f
			full.Order = order
			all, total := s.Query(full)
			for _, off := range []int{0, 1, 7, 50, 500} {
				w := full
				w.Offset, w.Limit = off, 9
				got, n := s.Query(w)
				lo := min(off, len(all))
				want := all[lo:min(lo+9, len(all))]
				if n != total || len(got) != len(want) {
					t.Fatalf("order %q filter %+v offset %d: total %d/%d rows %d/%d", order, f, off, n, total, len(got), len(want))
				}
				for i := range got {
					if got[i].ID != want[i].ID {
						t.Fatalf("order %q filter %+v offset %d row %d: %s want %s", order, f, off, i, got[i].ID, want[i].ID)
					}
				}
			}
		}
	}
}

func ptrBool(b bool) *bool { return &b }
