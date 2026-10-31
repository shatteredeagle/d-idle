package combat

import (
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// tomeHarness turns the harness scroll class (unit 77) into a tome whose stack count is qty (-1: no stat list = empty).
func tomeHarness(t *testing.T, qty int) (*harness, *portalGame) {
	h, g := newPortalHarness(t)
	cat := h.f.cat.(MapCatalog)
	cat.Items[itTSC] = ItemClass{Portal: true, Tome: true}
	h.f.cat = cat
	h.f.statsFn = func(uint32) (d2.Stats, error) {
		if qty < 0 {
			return nil, errors.New("no stat list")
		}
		return d2.Stats{statQuantity: int32(qty)}, nil
	}
	return h, g
}

func TestEmptyTomeIsNoPortal(t *testing.T) {
	h, g := tomeHarness(t, -1)
	if err := h.f.openPortal(t.Context()); !errors.Is(err, ErrNoPortal) {
		t.Fatalf("err %v", err)
	}
	if len(g.clicks) != 0 {
		t.Errorf("clicked an empty tome %d times", len(g.clicks))
	}
	s, _ := g.snap()
	if h.f.portalScrolls() != 0 || h.f.portalItem(s) != nil {
		t.Errorf("empty tome counted as a portal")
	}
}

func TestFilledTomeIsAPortal(t *testing.T) {
	h, g := tomeHarness(t, 3)
	if h.f.portalScrolls() != 3 {
		t.Errorf("scrolls %d", h.f.portalScrolls())
	}
	if err := h.f.openPortal(t.Context()); err != nil || !g.portal {
		t.Fatalf("err %v portal %v", err, g.portal)
	}
}

func TestFailedCastBacksOffTheSafetyAction(t *testing.T) {
	h, g := newPortalHarness(t)
	g.failClicks = 99
	h.s.life = 40
	h.s.addMon(10, clsFallen, 104, 100, 500, 5, 6)
	s, _ := g.snap()
	if a := h.f.Safety(s); a != TownPortal {
		t.Fatalf("before: %v", a)
	}
	if err := h.f.openPortal(t.Context()); err == nil {
		t.Fatal("want error")
	}
	if a := h.f.Safety(s); a == TownPortal {
		t.Errorf("TownPortal is offered again right after a failed cast")
	}
	h.s.mu.Lock()
	h.s.t = h.s.t.Add(portalBackoff + time.Second)
	h.s.mu.Unlock()
	if a := h.f.Safety(s); a != TownPortal {
		t.Errorf("after the backoff: %v", a)
	}
}
