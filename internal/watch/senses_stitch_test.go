package watch

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"sync"
	"testing"

	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/senses"
)

// Two small SYNTHETIC outdoor areas in the absolute frame: area 1 (the town) with area 2 (Blood Moor) on its west
// edge, in the geometry the recorded town image reports (neighbour 2 touches the west edge). Fixture only.
func stitchPair(t *testing.T) *senses.Publisher {
	t.Helper()
	townRect := d2map.Rect{X: 20, Y: 0, W: 10, H: 3}
	bmRect := d2map.Rect{X: 10, Y: 0, W: 10, H: 3}
	town := &d2map.LevelMap{No: 1, Rect: townRect, Grid: d2map.ParseASCII(20, 0, "##########", "#........#", "##########"),
		Adjacent: []d2map.Adjacent{{LevelNo: 2, Rect: bmRect}}}
	bm := &d2map.LevelMap{No: 2, Rect: bmRect, Grid: d2map.ParseASCII(10, 0, "##########", "#........#", "##########"),
		Adjacent: []d2map.Adjacent{{LevelNo: 1, Rect: townRect}}}
	adj, err := senses.ShippedAdjacency()
	if err != nil {
		t.Fatal(err)
	}
	p := &senses.Publisher{}
	p.SetAdjacency(adj)
	if err := p.Observe(town, "Rogue Encampment", d2map.Point{X: 21, Y: 1}); err != nil {
		t.Fatal(err)
	}
	if err := p.Observe(bm, "Blood Moor", d2map.Point{X: 18, Y: 1}); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestSensesStitchedGolden(t *testing.T) {
	p := stitchPair(t)
	w := httptest.NewRecorder()
	// the v1 golden stays as it was: a source without a trail answers without the additive keys
	(&Senses{Src: struct{ SensesSource }{p}, Lock: FixedTier(1)}).Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
	golden(t, "senses-stitched", w.Body.String())
}

func TestSensesNeighbourCacheAndGates(t *testing.T) {
	p := stitchPair(t)
	h := (&Senses{Src: p, Lock: FixedTier(1)}).Handler()
	_, a := getReply(t, h, "/api/senses")
	if len(a.Neighbours) != 1 || a.Neighbours[0].Map == nil || a.Neighbours[0].AreaID != 1 {
		t.Fatalf("%+v", a.Neighbours)
	}
	nb := fmt.Sprintf("1:%d", a.Neighbours[0].Version)
	_, b := getReply(t, h, fmt.Sprintf("/api/senses?v=%d&nb=%s", a.Version, nb))
	if b.Map != nil || len(b.Neighbours) != 1 || b.Neighbours[0].Map != nil || b.Neighbours[0].Version != a.Neighbours[0].Version {
		t.Fatalf("cached caller: %+v", b)
	}
	// a stale or malformed version still gets the map
	_, c := getReply(t, h, "/api/senses?nb=1:0,junk,x:y")
	if c.Neighbours[0].Map == nil {
		t.Fatal("stale nb version got no map")
	}
	// payload stays small
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest("GET", "/api/senses", nil))
	if w.Body.Len() > 16<<10 {
		t.Fatalf("payload %d bytes", w.Body.Len())
	}
	// tier 0 and tier 2 without tier 1 carry no neighbours (and no map)
	for name, lock := range map[string]Unlocker{"closed": FixedTier(0), "units-only": onlyTier(2)} {
		_, r := getReply(t, (&Senses{Src: p, Lock: lock}).Handler(), "/api/senses")
		if len(r.Neighbours) != 0 || r.Map != nil {
			t.Fatalf("%s: %+v", name, r)
		}
	}
}

type onlyTier int

func (o onlyTier) Allowed(t int) bool { return t == int(o) }

func TestSensesStitchRace(t *testing.T) {
	p := stitchPair(t)
	h := &Senses{Src: p, Lock: FixedTier(1)}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 200; j++ {
				r := h.ReplyNb(0, nil)
				if _, err := json.Marshal(r); err != nil {
					t.Error(err)
				}
			}
		}()
	}
	town := &d2map.LevelMap{No: 1, Rect: d2map.Rect{X: 20, Y: 0, W: 10, H: 3}, Grid: d2map.ParseASCII(20, 0, "##########", "#........#", "##########"),
		Adjacent: []d2map.Adjacent{{LevelNo: 2, Rect: d2map.Rect{X: 10, Y: 0, W: 10, H: 3}}}}
	for j := 0; j < 200; j++ {
		p.Observe(town, "Rogue Encampment", d2map.Point{X: 21 + j%7, Y: 1})
	}
	wg.Wait()
}
