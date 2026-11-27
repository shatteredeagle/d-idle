package combat

import (
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// useGame is the portal game plus the use-item verb. effect false is a build that takes the verb and opens nothing
// (the world refuses silently while busy); fail is a transport error on the send.
type useGame struct {
	*portalGame
	effect   bool
	fail     error
	used     []uint32
	tome     int  // charges of the tome (unit 77) when the harness made it a tome
	noObject bool // the portal object is not listed yet
}

func (g *useGame) UseItem(id uint32) error {
	g.used = append(g.used, id)
	if g.fail != nil {
		return g.fail
	}
	if g.effect && id == 77 {
		g.portal = !g.noObject
		if g.tome > 0 {
			g.tome--
		} else {
			g.scrollGone = true
		}
	}
	return nil
}

func useHarness(t *testing.T, class d2.Class, on bool, caps ...string) (*harness, *useGame) {
	h, pg := newPortalHarness(t)
	h.s.class = class
	g := &useGame{portalGame: pg, effect: true}
	h.f.e.In = g
	h.f.e.Planner = &input.Planner{Enabled: func() bool { return on }, Caps: &input.BackendCaps{Verbs: caps}}
	return h, g
}

// I3 (docs/design/input-i3.md): with the knob on and the capability listed, the cast is one verb, no panel key and no click.
func TestPortalByUseVerbNoPanelNoClick(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		h, g := useHarness(t, class, true, input.CapUseItem)
		if err := h.f.openPortal(t.Context()); err != nil {
			t.Fatalf("%v: %v", class, err)
		}
		if len(g.used) != 1 || g.used[0] != 77 || len(g.keys) != 0 || len(g.clicks) != 0 || !g.portal {
			t.Fatalf("%v: used %v keys %v clicks %v portal %v", class, g.used, g.keys, g.clicks, g.portal)
		}
		if v, k, c, _ := h.f.e.Planner.Counts(); v != 1 || k+c != 0 {
			t.Fatalf("%v: counts %d/%d/%d", class, v, k, c)
		}
	}
}

func TestPortalVerbOffOrNoCapKeepsClick(t *testing.T) {
	for _, tc := range []struct {
		on   bool
		caps []string
	}{{false, []string{input.CapUseItem}}, {true, nil}} {
		h, g := useHarness(t, d2.Necromancer, tc.on, tc.caps...)
		if err := h.f.openPortal(t.Context()); err != nil {
			t.Fatal(err)
		}
		if len(g.used) != 0 || len(g.clicks) != 1 {
			t.Fatalf("on %v caps %v: used %v clicks %v", tc.on, tc.caps, g.used, g.clicks)
		}
	}
}

func TestPortalVerbTransportErrorFallsBackToClick(t *testing.T) {
	h, g := useHarness(t, d2.Sorceress, true, input.CapUseItem)
	g.fail = errors.New("d2d: unknown cmd 'use'")
	if err := h.f.openPortal(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(g.used) != 1 || len(g.clicks) != 1 || !g.portal {
		t.Fatalf("used %v clicks %v portal %v", g.used, g.clicks, g.portal)
	}
	if v, _, _, _ := h.f.e.Planner.Counts(); v != 0 {
		t.Fatalf("a failed verb was counted")
	}
}

// A sent verb that opens nothing is never followed by the pack click: the click is the dropped-cast incident path.
func TestPortalVerbNoEffectNeverClicks(t *testing.T) {
	h, g := useHarness(t, d2.Paladin, true, input.CapUseItem)
	g.effect = false
	if err := h.f.openPortal(t.Context()); !errors.Is(err, errPortalVerbNoEffect) {
		t.Fatalf("err %v", err)
	}
	if len(g.used) != portalAttempts || len(g.clicks) != 0 {
		t.Fatalf("used %v clicks %v", g.used, g.clicks)
	}
}

// Stat-aware check: a tome stays in the pack, so the cast shows as its quantity stat going down even before the
// portal object is listed.
func TestPortalVerbTomeQuantityVerifies(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		h, pg := tomeHarness(t, 5)
		h.s.class = class
		g := &useGame{portalGame: pg, effect: true, tome: 5}
		h.f.e.In = g
		h.f.e.Planner = &input.Planner{Enabled: func() bool { return true }, Caps: &input.BackendCaps{Verbs: []string{input.CapUseItem}}}
		h.f.statsFn = func(uint32) (d2.Stats, error) { return d2.Stats{statQuantity: int32(g.tome)}, nil }
		g.noObject = true
		if err := h.f.openPortal(t.Context()); err != nil {
			t.Fatalf("%v: %v", class, err)
		}
		if g.tome != 4 || len(g.clicks) != 0 {
			t.Fatalf("%v: tome %d clicks %v", class, g.tome, g.clicks)
		}
	}
}
