package input

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

type Button int

const (
	Left Button = iota
	Right
)

func (b Button) letter() string {
	if b == Right {
		return "r"
	}
	return "l"
}

// Mod is a modifier applied to a click. ModShift on a left click on the ground makes the character
// stand still and attack; on a right click it casts in place.
type Mod string

const (
	ModShift Mod = "shift"
	ModCtrl  Mod = "ctrl"
)

// Target is something Interact can click: an NPC, object, monster or ground item.
type Target struct {
	Type uint32 // UnitAny.dwType (1 monster/NPC, 2 object, 4 item)
	ID   uint32
	Pos  Point // world position in subtiles
}

var (
	ErrOffscreen  = errors.New("input: target is not on screen")
	ErrMoveFailed = errors.New("input: did not arrive")
	ErrNotHovered = errors.New("input: pointer never hovered the target")
	ErrNoHover    = errors.New("input: hover detection unavailable")
)

// MoveOpts tunes MoveTo.
type MoveOpts struct {
	Tolerance float64       // arrive when within this many subtiles (default 2)
	Timeout   time.Duration // default 20 s
}

// Controller is everything the bot may do to the game. Implemented by *Game (live) and *Fake (tests).
type Controller interface {
	MoveMouse(x, y int) error
	Click(b Button, x, y int, mods ...Mod) error
	Key(name string) error
	Hold(key string) error
	Release(key string) error
	// Pos is the player's current position in subtiles.
	Pos() (Point, error)
	// ClickWorld left-clicks the screen position of a world point (walk toward it) and returns at once.
	ClickWorld(w Point) error
	// MoveTo walks until within Tolerance of w, re-clicking as needed.
	MoveTo(w Point, o MoveOpts) error
	SelectSkill(hotkey string) error
	// CastAt selects the skill on hotkey and right-clicks the world point w.
	CastAt(hotkey string, w Point) error
	// Interact hovers the target until the game reports it hovered, then left-clicks it.
	Interact(t Target) error
	// CloseAll presses Esc until no panel is open.
	CloseAll() error
}

// Clock is the time source of Game's own waits and deadlines (env.Clock satisfies it). nil means the wall clock.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type wallClock struct{}

func (wallClock) Now() time.Time        { return time.Now() }
func (wallClock) Sleep(d time.Duration) { time.Sleep(d) }

// Game drives the real game through the helper.
type Game struct {
	// Clock drives every wait and deadline of the controller; tests inject a fake one. nil: wall clock.
	Clock Clock
	// Ctx, if set, cancels the blocking calls (MoveTo, Interact, CloseAll) with ctx.Err(); the Controller interface has no
	// per-call context yet (docs/archive/review/findings.md F6).
	Ctx context.Context

	C      *Client
	T      Timing
	X      Transform
	R      Reader
	Hover  HoverFunc // nil disables hover verification (Interact then returns ErrNoHover)
	UIOpen UIFunc    // nil: CloseAll presses Esc once

	held map[string]bool
}

// HoverFunc returns the unit type and id under the pointer (id 0 when nothing).
type HoverFunc func() (typ, id uint32, err error)

// UIFunc reports whether any closable panel (inventory, NPC dialog, menu...) is open.
type UIFunc func() (bool, error)

func NewGame(c *Client, r Reader) *Game {
	g := &Game{C: c, T: DefaultTiming(), X: Calibrated, R: r, held: map[string]bool{}}
	g.UIOpen = r.UIOpen
	g.Hover = r.Hovered
	return g
}

func (g *Game) clk() Clock {
	if g.Clock == nil {
		return wallClock{}
	}
	return g.Clock
}

func (g *Game) sleep(d time.Duration) { g.clk().Sleep(d) }

func (g *Game) cancelled() error {
	if g.Ctx == nil {
		return nil
	}
	return g.Ctx.Err()
}

func (g *Game) gap() { g.sleep(g.T.Gap()) }

func ms(d time.Duration) string { return fmt.Sprintf("%d", d.Milliseconds()) }

func (g *Game) MoveMouse(x, y int) error {
	return g.C.Seq(fmt.Sprintf("move %d %d%s", x, y, g.heldMK()))
}

// heldMK adds the wParam bits for modifier keys currently held with Hold.
func (g *Game) heldMK() string {
	s := ""
	if g.held["shift"] {
		s += " mkshift"
	}
	if g.held["ctrl"] {
		s += " mkctrl"
	}
	return s
}

func (g *Game) Click(b Button, x, y int, mods ...Mod) error {
	flags := g.heldMK()
	for _, m := range mods {
		flags += " " + string(m)
	}
	press := ms(g.T.Press())
	var err error
	if len(mods) > 0 {
		// The helper wraps the whole click in the modifier key down/up.
		err = g.C.Seq(
			fmt.Sprintf("move %d %d%s", x, y, flags),
			"sleep "+ms(g.T.Settle()),
			fmt.Sprintf("click %s %d %d hold=%s%s", b.letter(), x, y, press, flags),
		)
	} else {
		// One round trip: move, let the game see the hover, press, release.
		err = g.C.Seq(
			fmt.Sprintf("move %d %d%s", x, y, flags),
			"sleep "+ms(g.T.Settle()),
			fmt.Sprintf("down %s %d %d%s", b.letter(), x, y, flags),
			"sleep "+press,
			fmt.Sprintf("up %s %d %d%s", b.letter(), x, y, flags),
		)
	}
	g.gap()
	return err
}

func (g *Game) Key(name string) error {
	err := g.C.Seq(fmt.Sprintf("key press %s hold=%s", name, ms(g.T.Press())))
	g.gap()
	return err
}

// TypeChar posts WM_CHAR for one character (digits into an edit box such as the gold amount dialog).
func (g *Game) TypeChar(ch rune) error {
	err := g.C.TypeChar(ch)
	g.gap()
	return err
}

func (g *Game) Hold(key string) error {
	if err := g.C.Seq("key down " + key); err != nil {
		return err
	}
	g.held[strings.ToLower(key)] = true
	return nil
}

func (g *Game) Release(key string) error {
	delete(g.held, strings.ToLower(key))
	return g.C.Seq("key up " + key)
}

func (g *Game) Pos() (Point, error) {
	p, err := g.R.Player()
	if err != nil {
		return Point{}, err
	}
	return Point{int(math.Round(p.X)), int(math.Round(p.Y))}, nil
}

// screenOf converts a world point (tile centre) to a clickable pixel, clamped into the world view.
// It uses the viewport variables, so it is right even while the camera lags a walking player;
// if they cannot be read it falls back to the player-relative transform.
func (g *Game) screenOf(w Point) (Screen, error) {
	wx, wy := float64(w.X)+0.5, float64(w.Y)+0.5
	if vx, vy, err := g.R.Viewport(); err == nil {
		sx, sy := g.X.WorldToScreenVP(vx, vy, wx, wy)
		return g.X.Clamp(Screen{int(math.Round(sx)), int(math.Round(sy))}), nil
	}
	p, err := g.R.Player()
	if err != nil {
		return Screen{}, err
	}
	sx, sy := g.X.WorldToScreenF([2]float64{p.X, p.Y}, wx, wy)
	return g.X.Clamp(Screen{int(math.Round(sx)), int(math.Round(sy))}), nil
}

func (g *Game) ClickWorld(w Point) error {
	s, err := g.screenOf(w)
	if err != nil {
		return err
	}
	return g.Click(Left, s.X, s.Y)
}

func (g *Game) MoveTo(w Point, o MoveOpts) error {
	if o.Tolerance <= 0 {
		o.Tolerance = 2
	}
	if o.Timeout <= 0 {
		o.Timeout = 20 * time.Second
	}
	clk := g.clk()
	deadline := clk.Now().Add(o.Timeout)
	dist := func(p PlayerState) float64 { return math.Hypot(p.X-float64(w.X), p.Y-float64(w.Y)) }
	stalls := 0
	for clk.Now().Before(deadline) {
		if err := g.cancelled(); err != nil {
			return err
		}
		p, err := g.R.Player()
		if err != nil {
			return err
		}
		if dist(p) <= o.Tolerance {
			return nil
		}
		if err := g.ClickWorld(w); err != nil {
			return err
		}
		// Follow the walk: return to re-click when the character stops, or after a second
		// (a clamped far target needs a new click once the view has scrolled).
		start := p
		last, lastMove := p, clk.Now()
		for clk.Now().Before(deadline) && clk.Now().Sub(lastMove) < 400*time.Millisecond && clk.Now().Sub(lastMove) < time.Second {
			if err := g.cancelled(); err != nil {
				return err
			}
			g.sleep(50 * time.Millisecond)
			cur, err := g.R.Player()
			if err != nil {
				return err
			}
			if dist(cur) <= o.Tolerance {
				return nil
			}
			if math.Hypot(cur.X-last.X, cur.Y-last.Y) > 0.2 {
				last, lastMove = cur, clk.Now()
			}
		}
		if math.Hypot(last.X-start.X, last.Y-start.Y) < 0.5 {
			if stalls++; stalls >= 3 {
				return fmt.Errorf("%w: stuck at (%.0f,%.0f) going to %v", ErrMoveFailed, last.X, last.Y, w)
			}
		} else {
			stalls = 0
		}
	}
	return fmt.Errorf("%w: timeout after %v going to %v", ErrMoveFailed, o.Timeout, w)
}

func (g *Game) SelectSkill(hotkey string) error { return g.Key(hotkey) }

func (g *Game) CastAt(hotkey string, w Point) error {
	if err := g.SelectSkill(hotkey); err != nil {
		return err
	}
	s, err := g.screenOf(w)
	if err != nil {
		return err
	}
	return g.Click(Right, s.X, s.Y)
}

// interactProbe lists pixel offsets from a unit's feet to try, bottom-up: sprites are drawn above
// their feet, and the hit box is the sprite. See docs/input.md for the measured NPC values.
var interactProbe = [][2]int{{0, -10}, {0, -20}, {0, -30}, {0, 0}, {0, -40}, {-8, -20}, {8, -20}, {0, -50}}

func (g *Game) Interact(t Target) error {
	if g.Hover == nil {
		return ErrNoHover
	}
	s, err := g.screenOf(t.Pos)
	if err != nil {
		return err
	}
	if !g.X.InView(s) {
		return ErrOffscreen
	}
	for _, off := range interactProbe {
		c := Screen{s.X + off[0], s.Y + off[1]}
		if !g.X.InView(c) {
			continue
		}
		if err := g.MoveMouse(c.X, c.Y); err != nil {
			return err
		}
		for i := 0; i < 4; i++ { // hover is evaluated on the game's next frame (40 ms)
			if err := g.cancelled(); err != nil {
				return err
			}
			g.sleep(50 * time.Millisecond)
			typ, id, err := g.Hover()
			if err != nil {
				return err
			}
			if id == t.ID && (t.Type == 0 || typ == t.Type) {
				return g.Click(Left, c.X, c.Y)
			}
		}
	}
	return ErrNotHovered
}

func (g *Game) CloseAll() error {
	if g.UIOpen == nil {
		return g.Key("esc")
	}
	for i := 0; i < 6; i++ {
		if err := g.cancelled(); err != nil {
			return err
		}
		open, err := g.UIOpen()
		if err != nil {
			return err
		}
		if !open {
			return nil
		}
		if err := g.Key("esc"); err != nil {
			return err
		}
		g.sleep(150 * time.Millisecond)
	}
	return errors.New("input: panels still open after 6 Esc")
}

// HoldClick presses the button on (x, y), keeps it down for d and releases it: a melee skill with a chain of hits
// (Zeal, VERIFIED LIVE by CLASSPAL: 19 kills in 55 s held 700 ms per tick against 7 kills in 30 s with single clicks).
func (g *Game) HoldClick(b Button, x, y int, d time.Duration) error {
	flags := g.heldMK()
	err := g.C.Seq(
		fmt.Sprintf("move %d %d%s", x, y, flags),
		"sleep "+ms(g.T.Settle()),
		fmt.Sprintf("down %s %d %d%s", b.letter(), x, y, flags),
		"sleep "+ms(d),
		fmt.Sprintf("up %s %d %d%s", b.letter(), x, y, flags),
	)
	g.gap()
	return err
}
