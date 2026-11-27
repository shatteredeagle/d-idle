package combat

import (
	"context"
	"strings"
	"testing"
)

// cmpOf returns the comparison and agreement counts of one driver kind.
func cmpOf(t *testing.T, f *Fighter, kind string) (cmp, agree int64) {
	t.Helper()
	kinds, c, a, on := f.UpkeepCompare()
	if !on {
		t.Fatal("knob on, counters off")
	}
	for i, k := range kinds {
		if k == kind {
			return c[i], a[i]
		}
	}
	t.Fatalf("no kind %q", kind)
	return 0, 0
}

// Mode 1 with a Paladin aura held up: every keepAura comparison is counted and all of them agree (the opening one too:
// before the apply both sides say down), so a quiet shadow log is told apart from a comparison that never ran.
func TestUpkeepCompareCountsAuraAgreement(t *testing.T) {
	h, g := newHandGame(t)
	h.f.e.Tune.Combat.DeliveryUpkeep = 1
	ctx := context.Background()
	g.aura, g.p.RightSkill = 0, 0
	s, _ := g.snap()
	if ok, err := h.f.KeepAura(ctx, s, &Target{}, true); err != nil || !ok {
		t.Fatalf("opening apply ok=%v err=%v", ok, err)
	}
	c0, _ := cmpOf(t, h.f, "aura")
	for range 4 {
		s, _ = g.snap()
		if _, err := h.f.KeepAura(ctx, s, &Target{}, true); err != nil {
			t.Fatal(err)
		}
		s, _ = g.snap()
		if _, err := h.f.KeepAura(ctx, s, &Target{}, false); err != nil { // gated: the recheck comparison
			t.Fatal(err)
		}
	}
	c, a := cmpOf(t, h.f, "aura")
	if c < c0+8 || a != c {
		t.Fatalf("aura cmp %d (from %d) agree %d: want +8 comparisons, all agreeing", c, c0, a)
	}
	if h.f.UpkeepShadowDiffs() != 0 {
		t.Fatalf("aura up: no disagreement, got %d", h.f.UpkeepShadowDiffs())
	}
	if f := h.f.UpkeepCompareField(); !strings.Contains(f, "aura:") || !strings.HasPrefix(f, "buff:") {
		t.Fatalf("field %q", f)
	}
}

// A Sorceress has no aura: mode 1 counts no aura comparisons; mode 0 emits no field at all.
func TestUpkeepCompareSorceressAndOff(t *testing.T) {
	h := newHarness(t)
	h.f.e.Tune.Combat.DeliveryUpkeep = 1
	s, _ := h.s.snapshot()
	for range 3 {
		if _, err := h.f.KeepAura(context.Background(), s, &Target{}, true); err != nil {
			t.Fatal(err)
		}
	}
	if c, _ := cmpOf(t, h.f, "aura"); c != 0 {
		t.Fatalf("sorceress aura comparisons %d, want 0", c)
	}
	h.f.e.Tune.Combat.DeliveryUpkeep = 0
	if f := h.f.UpkeepCompareField(); f != "" {
		t.Fatalf("mode 0 field %q, want none", f)
	}
}
