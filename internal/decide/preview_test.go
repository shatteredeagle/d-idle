package decide

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func pvRow(t *testing.T, id string) (*PolicyRow, *PreviewSpecs) {
	t.Helper()
	cat, err := Shipped()
	if err != nil {
		t.Fatal(err)
	}
	tbl, err := ShippedPolicy(cat)
	if err != nil {
		t.Fatal(err)
	}
	row, err := tbl.Row(id)
	if err != nil {
		t.Fatal(err)
	}
	sp, err := ShippedPreviewSpecs()
	if err != nil {
		t.Fatal(err)
	}
	return row, sp
}

var pvBase = map[string]float64{"combat.drinkLifePct": 55, "combat.chickenPct": 25, "loot.pickStrictness": 1}

func pvBaseline(k string) (float64, bool) { v, ok := pvBase[k]; return v, ok }

func fights(lows ...int) []PreviewEvent {
	var out []PreviewEvent
	for i, l := range lows {
		out = append(out, PreviewEvent{T: float64(100 + i*10), Kind: PvFight, LifePct: l})
	}
	return out
}

func delta(t *testing.T, r PreviewResult, key string) int {
	t.Helper()
	for _, l := range r.Lines {
		if l.Key == key {
			if l.Delta == nil {
				t.Fatalf("line %s has no delta: %+v", key, l)
			}
			return *l.Delta
		}
	}
	t.Fatalf("no line %s in %+v", key, r.Lines)
	return 0
}

// Known deltas from a synthetic log, two classes whose stance numbers differ.
func TestPreviewStanceKnownDeltas(t *testing.T) {
	row, sp := pvRow(t, "stance")
	lows := fights(30, 40, 60, 58, 70, 28, 90, 24, 52, 21)
	for _, c := range []struct {
		class, to    string
		leave, drink int
	}{
		{"paladin", "cautious", 2, 2}, // (25,35] = 30,28; (55,65] = 60,58
		{"sorceress", "cautious", 2, 2},
		{"paladin", "aggressive", -1, -1},   // (22,25] = 24; (50,55] = 52
		{"sorceress", "aggressive", -2, -1}, // (20,25] = 24,21
	} {
		r, err := sp.Preview(PreviewInput{Row: row, Class: c.class, Current: "default", New: c.to, Baseline: pvBaseline, Events: lows, Now: 1000, Window: 3600})
		if err != nil {
			t.Fatalf("%s/%s: %v", c.class, c.to, err)
		}
		if !r.Supported || !r.Enough {
			t.Fatalf("%s/%s: %+v", c.class, c.to, r)
		}
		if got := delta(t, r, "leave"); got != c.leave {
			t.Errorf("%s/%s leave %d want %d", c.class, c.to, got, c.leave)
		}
		if got := delta(t, r, "drink"); got != c.drink {
			t.Errorf("%s/%s drink %d want %d", c.class, c.to, got, c.drink)
		}
		for _, l := range r.Lines {
			if l.Based != 10 || !strings.Contains(l.Basis, "based on 10 fights in the last 17 minutes") {
				t.Errorf("basis %+v", l)
			}
			if l.Confidence != "medium" {
				t.Errorf("a bound with 10 events is medium, got %q", l.Confidence)
			}
		}
	}
}

func TestPreviewNotEnoughAndWindow(t *testing.T) {
	row, sp := pvRow(t, "stance")
	r, err := sp.Preview(PreviewInput{Row: row, Class: "paladin", Current: "default", New: "cautious", Baseline: pvBaseline, Now: 1000, Window: 3600})
	if err != nil || !r.Supported || r.Enough || len(r.Lines) != 0 || !strings.HasPrefix(r.Reason, "not enough events") {
		t.Fatalf("empty history: %+v %v", r, err)
	}
	// events outside the window do not count
	old := fights(30, 40, 60, 58)
	r, _ = sp.Preview(PreviewInput{Row: row, Class: "paladin", Current: "default", New: "cautious", Baseline: pvBaseline, Events: old, Now: 100000, Window: 600})
	if r.Enough || r.Window.Events != 0 {
		t.Fatalf("old events counted: %+v", r)
	}
}

func TestPreviewWhitesFilteredList(t *testing.T) {
	row, sp := pvRow(t, "whites")
	evs := []PreviewEvent{
		{T: 10, Kind: PvDrop, Item: "Short Sword", Quality: "normal", Type: "sword"},
		{T: 20, Kind: PvDrop, Item: "El Rune", Quality: "normal", Type: "rune"},
		{T: 30, Kind: PvDrop, Item: "Cap", Quality: "normal", Type: "helm"},
		{T: 40, Kind: PvDrop, Item: "Amulet", Quality: "magic", Type: "amulet"},
		{T: 50, Kind: PvDrop, Item: "Wirt's Leg", Quality: "unique", Type: "club"},
	}
	r, err := sp.Preview(PreviewInput{Row: row, Class: "sorceress", Current: "default", New: "keepers", Baseline: pvBaseline, Events: evs, Now: 100, Window: 3600})
	if err != nil {
		t.Fatal(err)
	}
	if got := delta(t, r, "pickups"); got != -2 || r.FilteredTotal != 2 || len(r.Filtered) != 2 {
		t.Fatalf("strictness 2: delta %d total %d list %v", got, r.FilteredTotal, r.Filtered)
	}
	if r.Filtered[0].Item != "Cap" || !strings.Contains(r.Filtered[0].Why, "pickit may still have kept") {
		t.Errorf("filtered %+v", r.Filtered)
	}
	// looser: no number
	r, _ = sp.Preview(PreviewInput{Row: row, Class: "sorceress", Current: "default", New: "sellable", Baseline: pvBaseline, Events: evs, Now: 100, Window: 3600})
	if r.Lines[0].Delta != nil || r.Lines[0].Confidence != "none" || r.FilteredTotal != 0 {
		t.Errorf("looser must not invent a number: %+v", r.Lines)
	}
}

func TestPreviewFilteredIsBounded(t *testing.T) {
	row, sp := pvRow(t, "whites")
	var evs []PreviewEvent
	for i := 0; i < 80; i++ {
		evs = append(evs, PreviewEvent{T: float64(i), Kind: PvDrop, Item: "Cap", Quality: "normal", Type: "helm"})
	}
	r, _ := sp.Preview(PreviewInput{Row: row, Class: "paladin", Current: "default", New: "keepers", Baseline: pvBaseline, Events: evs, Now: 100, Window: 3600})
	if r.FilteredTotal != 80 || len(r.Filtered) != MaxFiltered {
		t.Fatalf("total %d list %d", r.FilteredTotal, len(r.Filtered))
	}
}

func TestPreviewNoPreviewRowsAndErrors(t *testing.T) {
	cat, _ := Shipped()
	tbl, _ := ShippedPolicy(cat)
	sp, _ := ShippedPreviewSpecs()
	// every row of the table has an explicit entry: a new row must decide
	for _, r := range tbl.Rows {
		if !sp.Has(r.ID) {
			t.Errorf("row %q has no preview entry", r.ID)
		}
	}
	for id := range sp.f.Rows {
		if _, err := tbl.Row(id); err != nil {
			t.Errorf("preview entry %q is not a row", id)
		}
	}
	row, _ := tbl.Row("sell")
	r, err := sp.Preview(PreviewInput{Row: row, Class: "paladin", Current: "default", New: "keep", Baseline: pvBaseline, Now: 10, Window: 60})
	if err != nil || r.Supported || !strings.HasPrefix(r.Reason, "no preview for this row yet: ") || len(r.Lines) != 0 {
		t.Fatalf("sell: %+v %v", r, err)
	}
	if _, err := sp.Preview(PreviewInput{Row: row, Current: "default", New: "melt", Now: 10, Window: 60}); err == nil || !strings.Contains(err.Error(), "melt") {
		t.Errorf("unknown option: %v", err)
	}
	if _, err := sp.Preview(PreviewInput{Now: 10, Window: 60}); !errors.Is(err, ErrPreviewInput) {
		t.Errorf("no row: %v", err)
	}
	empty, _ := ParsePreviewSpecs([]byte(`{"version":1,"minEvents":3,"rows":{}}`))
	if _, err := empty.Preview(PreviewInput{Row: row, Current: "default", New: "keep", Now: 10, Window: 60}); !errors.Is(err, ErrPreviewRow) {
		t.Errorf("unknown row: %v", err)
	}
	// a missing baseline for a knob the option does not set is an error naming the knob
	srow, _ := tbl.Row("stance")
	_, err = sp.Preview(PreviewInput{Row: srow, Class: "paladin", Current: "default", New: "cautious", Events: fights(1, 2, 3), Now: 1000, Window: 3600})
	if !errors.Is(err, ErrPreviewKnob) || !strings.Contains(err.Error(), "combat.chickenPct") {
		t.Errorf("missing baseline: %v", err)
	}
	// an unknown class is an error naming it, no silent fallback
	_, err = sp.Preview(PreviewInput{Row: srow, Class: "necromancer", Current: "default", New: "cautious", Baseline: pvBaseline, Events: fights(1, 2, 3), Now: 1000, Window: 3600})
	if err == nil || !strings.Contains(err.Error(), "necromancer") {
		t.Errorf("unknown class: %v", err)
	}
	for _, bad := range []string{`{"version":1,"minEvents":3,"rows":{"x":{"kind":"zzz"}}}`, `{"version":1,"minEvents":3,"rows":{"x":{"kind":"none"}}}`, `{"version":2,"minEvents":3,"rows":{}}`} {
		if _, err := ParsePreviewSpecs([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

// Preview reads only: the same input gives the same answer from many goroutines and the input is untouched.
func TestPreviewRace(t *testing.T) {
	row, sp := pvRow(t, "stance")
	lows := fights(30, 40, 60, 58, 70, 28)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			r, err := sp.Preview(PreviewInput{Row: row, Class: "paladin", Current: "default", New: "cautious", Baseline: pvBaseline, Events: lows, Now: 1000, Window: 3600})
			if err != nil || delta(t, r, "leave") != 2 {
				t.Errorf("%+v %v", r, err)
			}
		}()
	}
	wg.Wait()
}
