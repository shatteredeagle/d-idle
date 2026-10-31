package combat

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/panel"
)

// portalGame fakes the inventory panel: "i" toggles it, a right click on the scroll cell while it is open
// consumes the scroll and opens a portal. Clicks elsewhere do nothing.
type portalGame struct {
	*sim
	open       bool
	scrollGone bool
	portal     bool
	keys       []string
	clicks     []input.Screen
	failClicks int // the first n correct clicks are ignored
	panelStuck bool
	scrollCell input.Screen
	uiOnStart  bool
}

func (g *portalGame) Key(name string) error {
	g.keys = append(g.keys, name)
	if name == "i" {
		g.open = !g.open
	}
	return nil
}
func (g *portalGame) CloseAll() error          { g.open = false; return nil }
func (g *portalGame) MoveMouse(x, y int) error { return nil }
func (g *portalGame) Click(b input.Button, x, y int, m ...input.Mod) error {
	at := input.Screen{X: x, Y: y}
	g.clicks = append(g.clicks, at)
	if g.open && b == input.Right && at == g.scrollCell {
		if g.failClicks > 0 {
			g.failClicks--
			return nil
		}
		g.scrollGone, g.portal = true, true
	}
	return nil
}
func (g *portalGame) snap() (*d2.Snapshot, error) {
	s, _ := g.sim.snapshot()
	if g.open {
		s.UI[d2.UIInventory] = 1
	}
	s.Screen = d2.Point{X: 800, Y: 600}
	if g.scrollGone {
		var keep []d2.Unit
		for _, u := range s.Inventory.Items {
			if u.ID != 77 {
				keep = append(keep, u)
			}
		}
		s.Inventory.Items = keep
	}
	for i := range s.Inventory.Items {
		if s.Inventory.Items[i].ID == 77 {
			s.Inventory.Items[i].Item.GX, s.Inventory.Items[i].Item.GY = 4, 1
		}
	}
	if g.portal {
		s.Objects = append(s.Objects, d2.Unit{Type: d2.TypeObject, Class: objTownPortal, X: s.Player.X + 3, Y: s.Player.Y})
	}
	return s, nil
}

func newPortalHarness(t *testing.T) (*harness, *portalGame) {
	h := newHarness(t)
	g := &portalGame{sim: h.s}
	// 800x600 grid: left 417+80 style numbers are made up, the test checks the offset is part of the grid
	grid := panel.Grid{Name: "inventory", Cols: 10, Rows: 4, Left: 417 + 80, Top: 315 + 60, CellW: 29, CellH: 29}
	g.scrollCell = input.Screen{X: grid.Left + 4*29 + 14, Y: grid.Top + 1*29 + 14}
	h.f.e.In = g
	h.f.snapFn = g.snap
	h.f.gridFn = func(*d2.Snapshot) (panel.Grid, error) { return grid, nil }
	h.f.Portal = nil
	return h, g
}

func TestOpenPortalClicksScroll(t *testing.T) {
	h, g := newPortalHarness(t)
	if err := h.f.openPortal(t.Context()); err != nil {
		t.Fatal(err)
	}
	if !g.portal || len(g.keys) != 1 || g.keys[0] != "i" {
		t.Errorf("portal %v keys %v", g.portal, g.keys)
	}
}

func TestOpenPortalPanelAlreadyOpen(t *testing.T) {
	h, g := newPortalHarness(t)
	g.open = true
	if err := h.f.openPortal(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(g.keys) != 0 {
		t.Errorf("pressed %v with the panel open (would close it)", g.keys)
	}
}

func TestOpenPortalRetriesIgnoredClick(t *testing.T) {
	h, g := newPortalHarness(t)
	g.failClicks = 2
	if err := h.f.openPortal(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(g.clicks) != 3 {
		t.Errorf("clicks %d", len(g.clicks))
	}
}

func TestOpenPortalGivesUp(t *testing.T) {
	h, g := newPortalHarness(t)
	g.failClicks = 99
	if err := h.f.openPortal(t.Context()); err == nil {
		t.Fatal("want error")
	}
	if len(g.clicks) != portalAttempts {
		t.Errorf("clicks %d", len(g.clicks))
	}
}

func TestOpenPortalNoScroll(t *testing.T) {
	h, g := newPortalHarness(t)
	g.scrollGone = true
	if err := h.f.openPortal(t.Context()); !errors.Is(err, ErrNoPortal) {
		t.Errorf("err %v", err)
	}
}

func TestFailedPortalLeavesGame(t *testing.T) {
	h := newHarness(t)
	h.s.life = 50
	h.f.Portal = func(context.Context) error { return errors.New("combat: no portal appeared after the click") }
	h.s.addMon(10, clsFallen, 104, 100, 500, 5, 6)
	_, err := h.f.Clear(context.Background(), ClearOpts{})
	if !errors.Is(err, ErrChicken) || !h.s.left {
		t.Fatalf("want chicken after a failed portal, got %v left=%v", err, h.s.left)
	}
}

// Fresha i5 / Soaky i4 soak 2026-10-30 (docs/potion-bug.md): a failing town portal cast took about 6 s of three
// tries while a crowd chewed on the character, and no potion was drunk in between.
func TestOpenPortalDrinksBetweenTries(t *testing.T) {
	h, g := newPortalHarness(t)
	g.failClicks = 99
	h.s.life = 40
	h.s.stock(itHP1, 0, 2)
	h.s.addMon(10, clsFallen, 104, 100, 500, 0, 0)
	if err := h.f.openPortal(t.Context()); err == nil {
		t.Fatal("want error")
	}
	n := 0
	for _, k := range g.keys {
		if k == "1" {
			n++
		}
	}
	if n == 0 {
		t.Fatalf("no healing potion key during the failed cast, keys %v", g.keys)
	}
}
