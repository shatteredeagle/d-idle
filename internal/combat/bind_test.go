package combat

import (
	"context"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// popupGame simulates the right-skill popup: its button opens it, clicking an icon selects that skill and closes it,
// an F key pressed over an icon binds, a bound F key outside the popup selects the skill.
type popupGame struct {
	*sim
	open   bool
	hover  input.Screen
	bound  map[string]int
	icons  map[input.Screen]int
	broken bool // the key press never binds
	opened int
}

func (g *popupGame) Click(b input.Button, x, y int, m ...input.Mod) error {
	at := input.Screen{X: x, Y: y}
	switch {
	case g.open:
		if id, ok := g.icons[at]; ok {
			g.right = id
		}
		g.open = false
	case x == Popup.BtnX && y == Popup.BtnY:
		g.open = true
		g.opened++
	}
	return nil
}
func (g *popupGame) MoveMouse(x, y int) error { g.hover = input.Screen{X: x, Y: y}; return nil }
func (g *popupGame) Key(name string) error {
	if len(name) == 2 && name[0] == 'f' {
		if g.open {
			if id, ok := g.icons[g.hover]; ok && !g.broken {
				g.bound[name] = id
			}
		} else if id, ok := g.bound[name]; ok {
			g.right = id
		}
	}
	return nil
}
func (g *popupGame) CloseAll() error { g.open = false; return nil }
func (g *popupGame) snap() (*d2.Snapshot, error) {
	s, _ := g.sim.snapshot()
	s.Player.RightSkill = g.right
	return s, nil
}

func newPopupHarness(t *testing.T) (*harness, *popupGame) {
	h := newHarness(t)
	h.s.skills = []d2.Skill{{ID: 0, Level: 1}, {ID: 2, Level: 1}, {ID: 36, Bonus: 1}, {ID: 38, Level: 3}}
	h.f.SetListRows(map[int]int{0: 0, 2: 0, 36: 1, 38: 2})
	g := &popupGame{sim: h.s, bound: map[string]int{}, icons: map[input.Screen]int{}}
	// rows present: 0 (Attack, Throw), 1 (Fire Bolt), 2 (Charged Bolt); rows are packed bottom up
	p := Popup
	g.icons[input.Screen{X: p.Col0X, Y: p.Row0Y}] = 0
	g.icons[input.Screen{X: p.Col0X - p.ColDX, Y: p.Row0Y}] = 2
	g.icons[input.Screen{X: p.Col0X, Y: p.Row0Y - p.RowDY}] = 36
	g.icons[input.Screen{X: p.Col0X, Y: p.Row0Y - 2*p.RowDY}] = 38
	g.right = 36
	h.f.e.In = g
	h.f.snapFn = g.snap
	h.f.SetHotkeys(nil)
	return h, g
}

func TestEnsureHotkeysBindsThroughPopup(t *testing.T) {
	h, g := newPopupHarness(t)
	buf := logTo(h)
	if err := h.f.EnsureHotkeys(context.Background()); err != nil {
		t.Fatal(err)
	}
	// BUILDV2: one summary line per class and call, the live run's check that every class tried its keys
	if l := buf.String(); !strings.Contains(l, "combat bind") || !strings.Contains(l, "wanted=2") || !strings.Contains(l, "failed=0") {
		t.Fatalf("no combat bind summary line: %s", l)
	}
	if g.bound["f8"] != 36 || g.bound["f4"] != 38 {
		t.Fatalf("bindings %v", g.bound)
	}
	if k := h.f.Hotkeys(); k[36] != "f8" || k[38] != "f4" {
		t.Fatalf("hotkey map %v", k)
	}
	// second run: keys already select the skills, nothing is rebound
	before := len(g.bound)
	g.broken = true
	h.f.SetHotkeys(nil)
	if err := h.f.EnsureHotkeys(context.Background()); err != nil || len(g.bound) != before || len(h.f.Hotkeys()) != 2 {
		t.Fatalf("rerun: %v %v", err, h.f.Hotkeys())
	}
}

func TestEnsureHotkeysDetectsBrokenBinding(t *testing.T) {
	h, g := newPopupHarness(t)
	g.broken = true
	err := h.f.EnsureHotkeys(context.Background())
	if !errors.Is(err, ErrBindFailed) || len(h.f.Hotkeys()) != 0 {
		t.Fatalf("a key that never binds must be reported, not recorded: %v %v", err, h.f.Hotkeys())
	}
}

func TestEnsureHotkeysNeedsRows(t *testing.T) {
	h, _ := newPopupHarness(t)
	h.f.SetListRows(nil)
	if err := h.f.EnsureHotkeys(context.Background()); !errors.Is(err, ErrBindFailed) {
		t.Fatal(err)
	}
}

func TestPopupCellsSkipEmptyRows(t *testing.T) {
	h, _ := newPopupHarness(t)
	p := baseSnap().Player
	p.Skills = []d2.Skill{{ID: 0, Level: 1}, {ID: 2, Level: 1}, {ID: 38, Level: 1}} // no fire row
	c := h.f.popupCells(p, 38)
	if len(c) != 1 || c[0].Y != Popup.Row0Y-Popup.RowDY {
		t.Fatalf("Charged Bolt is the second occupied row: %v", c)
	}
	if c := h.f.popupCells(p, 0); len(c) != 2 || c[1].X != Popup.Col0X-Popup.ColDX {
		t.Fatalf("row 0 has two cells: %v", c)
	}
	if h.f.popupCells(p, 999) != nil {
		t.Fatal("unknown skill")
	}
}

func TestWantedBindings(t *testing.T) {
	p := baseSnap().Player
	p.Skills = []d2.Skill{{ID: 36, Bonus: 1}, {ID: 38, Level: 1}, {ID: 42, Level: 1}, {ID: 99, Level: 1}}
	w := WantedBindings(p, testCatalog(), BuildBlizzard)
	if len(w) != 3 || w[0].Key != "f1" || w[0].Skill.Name != "Static Field" || w[2].Key != "f8" {
		t.Fatalf("%+v", w)
	}
}
