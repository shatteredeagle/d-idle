// Package d2dinput drives a d2d game through its control channel (internal/d2dctl). Controller implements
// input.Controller and input.SemanticInput; Lifecycle (lifecycle.go) starts and stops the process.
// The mapping table and the gaps are in docs/d2d-backend.md, "Phase 2b: input status".
package d2dinput

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/input"
)

// Controller is an input.Controller on top of a d2dctl.Client.
type Controller struct {
	C *d2dctl.Client
	// Ctx bounds every channel call (the Controller interface has no context). nil: Background.
	Ctx context.Context
	// Clock drives MoveTo's waits; nil: wall clock.
	Clock input.Clock
	// X is the iso transform, used only for the pixel fallbacks (CastAt without a skill id, portals). U: not
	// re-measured on d2d (docs/d2d-spike.md section 7).
	X input.Transform
	// Origin is added to d2d's level-relative cell positions (in subtiles), the same switch as d2dsrc.Options.AddOrigin.
	Origin input.Point
	// Skills maps a hotkey the bot uses ("f1") to a skill id, so SelectSkill and CastAt use "cmd select|skill"
	// instead of keys. Hotkeys not in the map fall back to the key and a right click.
	Skills map[string]int
	// Poll is MoveTo's polling interval (default 100 ms).
	Poll time.Duration

	// Run, when set, manages the run toggle by stamina before every move (run.go). nil: always walk, as before.
	Run *RunPolicy

	lastLevel int
	primed    bool
	running   bool // our belief of d2d's run toggle
}

// New returns a Controller with default transform and polling.
func New(c *d2dctl.Client) *Controller {
	return &Controller{C: c, X: input.Calibrated, Poll: 100 * time.Millisecond}
}

var (
	_ input.Controller    = (*Controller)(nil)
	_ input.SemanticInput = (*Controller)(nil)
)

type wall struct{}

func (wall) Now() time.Time        { return time.Now() }
func (wall) Sleep(d time.Duration) { time.Sleep(d) }

// Wait sleeps on the controller's clock: virtual ticks under lockstep, wall time otherwise.
func (c *Controller) Wait(d time.Duration) { c.clk().Sleep(d) }

func (c *Controller) ctx() context.Context {
	if c.Ctx != nil {
		return c.Ctx
	}
	return context.Background()
}

func (c *Controller) clk() input.Clock {
	if c.Clock != nil {
		return c.Clock
	}
	return wall{}
}

func (c *Controller) poll() time.Duration {
	if c.Poll > 0 {
		return c.Poll
	}
	return 100 * time.Millisecond
}

func unsupported(what string) error {
	return fmt.Errorf("%w: %s", input.ErrUnsupported, what)
}

// cell converts a subtile (centre of the subtile) to d2d cells.
func (c *Controller) cell(p input.Point) (x, y float64) {
	return (float64(p.X-c.Origin.X) + 0.5) / 5, (float64(p.Y-c.Origin.Y) + 0.5) / 5
}

// ---- pixels ----

// MoveMouse hovers the pointer ("move x y"). Game pixels are our client coordinates: the frame is 800x600 (V, "info").
func (c *Controller) MoveMouse(x, y int) error { return c.C.Hover(c.ctx(), x, y) }

// Click sends "click" or "rclick". Modifiers are not on the channel (V: "key shift" is an error, P3 of the
// fork is not done), so any modifier is ErrUnsupported; shift-drinking has Belt.
func (c *Controller) Click(b input.Button, x, y int, mods ...input.Mod) error {
	if len(mods) > 0 {
		return unsupported(fmt.Sprintf("%v-click: the channel has no click modifiers (use Belt for potions)", mods))
	}
	if b == input.Right {
		return c.C.RClick(c.ctx(), x, y)
	}
	return c.C.Click(c.ctx(), x, y)
}

// Key sends "key <d2d name>" (see KeyName for the table and its tags).
func (c *Controller) Key(name string) error {
	k, err := KeyName(name)
	if err != nil {
		return err
	}
	return c.C.Key(c.ctx(), k)
}

// TypeChar sends one character with "text" (name entry, gold amount).
func (c *Controller) TypeChar(ch rune) error { return c.C.Text(c.ctx(), string(ch)) }

// Hold and Release need key down/up, which the channel lacks (V: only "key <name>", a press).
func (c *Controller) Hold(key string) error    { return unsupported("Hold " + key) }
func (c *Controller) Release(key string) error { return unsupported("Release " + key) }

// ---- position and walking ----

// Pos is the first player's position in subtiles ("debug players", a debug verb that is read-only, V).
func (c *Controller) Pos() (input.Point, error) {
	pl, err := c.C.Players(c.ctx())
	if err != nil {
		return input.Point{}, err
	}
	if len(pl.Items) == 0 {
		return input.Point{}, errors.New("d2dinput: no player listed")
	}
	p := pl.Items[0]
	for _, q := range pl.Items {
		if q.Seat == -1 {
			p = q
			break
		}
	}
	return input.Point{X: d2dctl.SubTile(p.X) + c.Origin.X, Y: d2dctl.SubTile(p.Y) + c.Origin.Y}, nil
}

// prime sends the priming click when "cmd move" would be ignored: the first move on this controller and
// the first one after the level id changed (V quirk after a load or goto; the client itself also kicks after
// its own Goto/Warp/Enter). U whether a change of level through a walk needs it too: stall handling in MoveTo
// covers that.
func (c *Controller) prime() error {
	l, err := c.C.Level(c.ctx())
	if err != nil {
		return err
	}
	if c.primed && l.ID == c.lastLevel {
		return nil
	}
	if err := c.C.Kick(c.ctx()); err != nil {
		return err
	}
	c.primed, c.lastLevel = true, l.ID
	return nil
}

// ClickWorld starts "cmd move" toward w and returns at once.
func (c *Controller) ClickWorld(w input.Point) error { return c.Move(w) }

// Move is ClickWorld by its semantic name.
func (c *Controller) Move(w input.Point) error {
	if err := c.prime(); err != nil {
		return err
	}
	c.manageRun()
	x, y := c.cell(w)
	return c.C.CmdMove(c.ctx(), x, y)
}

// MoveTo re-issues "cmd move" until the player is within Tolerance of w (a long path stops short, V). A
// stalled first attempt gets one extra Kick (the move-ignored quirk).
func (c *Controller) MoveTo(w input.Point, o input.MoveOpts) error {
	if o.Tolerance <= 0 {
		o.Tolerance = 2
	}
	if o.Timeout <= 0 {
		o.Timeout = 20 * time.Second
	}
	clk := c.clk()
	deadline := clk.Now().Add(o.Timeout)
	dist := func(p input.Point) float64 { return math.Hypot(float64(p.X-w.X), float64(p.Y-w.Y)) }
	stalls, kicked := 0, false
	for clk.Now().Before(deadline) {
		if err := c.ctx().Err(); err != nil {
			return err
		}
		start, err := c.Pos()
		if err != nil {
			return err
		}
		if dist(start) <= o.Tolerance {
			return nil
		}
		if err := c.Move(w); err != nil {
			return err
		}
		last, lastMove := start, clk.Now()
		for clk.Now().Before(deadline) && clk.Now().Sub(lastMove) < 400*time.Millisecond {
			clk.Sleep(c.poll())
			cur, err := c.Pos()
			if err != nil {
				return err
			}
			if dist(cur) <= o.Tolerance {
				return nil
			}
			if math.Hypot(float64(cur.X-last.X), float64(cur.Y-last.Y)) > 1 {
				last, lastMove = cur, clk.Now()
			}
		}
		if math.Hypot(float64(last.X-start.X), float64(last.Y-start.Y)) >= 2 {
			stalls = 0
			continue
		}
		stalls++
		if !kicked {
			kicked = true
			if err := c.C.Kick(c.ctx()); err != nil {
				return err
			}
			continue
		}
		if stalls >= 3 {
			return fmt.Errorf("%w: stuck at %v going to %v", input.ErrMoveFailed, last, w)
		}
	}
	return fmt.Errorf("%w: timeout after %v going to %v", input.ErrMoveFailed, o.Timeout, w)
}

// ---- skills ----

// SelectSkill puts the hotkey's skill on the right button ("cmd select <id> 0") when Skills knows it, else presses the key.
func (c *Controller) SelectSkill(hotkey string) error {
	if id, ok := c.Skills[strings.ToLower(hotkey)]; ok {
		return c.C.CmdSelect(c.ctx(), id, false)
	}
	return c.Key(hotkey)
}

// CastAt casts the hotkey's skill at w: "cmd skill" when the id is known, else key + right click on the pixel
// computed from the player position (U: transform not calibrated on d2d).
func (c *Controller) CastAt(hotkey string, w input.Point) error {
	if id, ok := c.Skills[strings.ToLower(hotkey)]; ok {
		return c.Cast(id, w, 0)
	}
	if err := c.Key(hotkey); err != nil {
		return err
	}
	s, err := c.screenOf(w)
	if err != nil {
		return err
	}
	return c.C.RClick(c.ctx(), s.X, s.Y)
}

func (c *Controller) screenOf(w input.Point) (input.Screen, error) {
	p, err := c.Pos()
	if err != nil {
		return input.Screen{}, err
	}
	sx, sy := c.X.WorldToScreenF([2]float64{float64(p.X), float64(p.Y)}, float64(w.X)+0.5, float64(w.Y)+0.5)
	return c.X.Clamp(input.Screen{X: int(math.Round(sx)), Y: int(math.Round(sy))}), nil
}

// Cast is "cmd skill <id> x y [unit]" (V with a unit id; U for ground-targeted skills and which button).
func (c *Controller) Cast(skill int, w input.Point, unit uint32) error {
	x, y := c.cell(w)
	return c.C.CmdSkill(c.ctx(), skill, x, y, d2dctl.SkillOpts{Unit: int(unit)})
}

// ---- interaction ----

// Interact maps a Target by the synthetic ids of d2dsrc: NPC -> "cmd interact <index>", ground item (type 4) ->
// "cmd pickup", a monster -> "cmd skill 0 x y <unit> left" (normal attack; U), portal rows -> UsePortal.
// Objects (chest, waypoint, stash) have no verb: ErrUnsupported, as is the merc.
func (c *Controller) Interact(t input.Target) error {
	id := t.ID
	switch {
	case id >= d2dsrc.PortalIDBase:
		return c.UsePortal(int(id - d2dsrc.PortalIDBase))
	case id == d2dsrc.MercID:
		return unsupported("interact with the mercenary")
	case id >= d2dsrc.NPCIDBase:
		return c.InteractNPC(int(id - d2dsrc.NPCIDBase))
	}
	switch t.Type {
	case 4:
		return c.Pickup(id)
	case 1:
		x, y := c.cell(t.Pos)
		return c.C.CmdSkill(c.ctx(), 0, x, y, d2dctl.SkillOpts{Unit: int(id), Left: true})
	}
	return unsupported(fmt.Sprintf("interact with unit type %d (objects have no cmd verb)", t.Type))
}

func (c *Controller) InteractNPC(index int) error {
	if err := c.prime(); err != nil {
		return err
	}
	return c.C.CmdInteract(c.ctx(), index)
}

func (c *Controller) Pickup(unit uint32) error { return c.C.CmdPickup(c.ctx(), int(unit)) }

func (c *Controller) Belt(slot int, merc bool) error { return c.C.CmdBelt(c.ctx(), slot, merc) }

// ChooseMenu clicks the NPC menu line whose text equals text (case-insensitive; the header, line 0, is skipped).
func (c *Controller) ChooseMenu(text string) error {
	m, err := c.C.Menu(c.ctx())
	if err != nil {
		return err
	}
	for i, l := range m.Items {
		if i > 0 && strings.EqualFold(strings.TrimSpace(l.Text), text) {
			return c.C.Click(c.ctx(), l.X, l.Y)
		}
	}
	return fmt.Errorf("d2dinput: menu has no line %q", text)
}

// UsePortal walks to the portal (row of "debug portals", the end that is in the current level) and clicks it.
// U: the click point is computed with the uncalibrated transform and "debug portals" has no screen position.
func (c *Controller) UsePortal(row int) error {
	ps, err := c.C.Portals(c.ctx())
	if err != nil {
		return err
	}
	if row < 0 || row >= len(ps.Items) {
		return fmt.Errorf("d2dinput: no portal row %d (%d listed)", row, len(ps.Items))
	}
	p := ps.Items[row]
	w := input.Point{X: d2dctl.SubTile(p.X) + c.Origin.X, Y: d2dctl.SubTile(p.Y) + c.Origin.Y}
	if err := c.MoveTo(w, input.MoveOpts{Tolerance: 8}); err != nil {
		return err
	}
	s, err := c.screenOf(w)
	if err != nil {
		return err
	}
	return c.C.Click(c.ctx(), s.X, s.Y)
}

// CloseAll closes what d2d lets us close: an NPC menu through its "cancel" line (not Escape: V, a second
// Escape opens the game menu and "cmd move" is then ignored) and the game menu through "Return to Game"
// (400,308, V). Open panels (inventory, stash) cannot be seen on the channel yet (U): they stay open.
func (c *Controller) CloseAll() error {
	for i := 0; i < 4; i++ {
		st, _, err := c.C.State(c.ctx())
		if err != nil {
			return err
		}
		if st.GameMenuOpen() {
			if err := c.C.Click(c.ctx(), 400, 308); err != nil {
				return err
			}
			c.clk().Sleep(150 * time.Millisecond)
			continue
		}
		// A vendor window has no cancel line: one Escape closes it (V docs/d2d-town.md smoke; a second one would open the
		// game menu, which is handled above on the next round). Seen live (D2DWIRE): without this the town's CloseAll
		// failed with the shop flag still set until the failsafe ladder restarted the game.
		// Same for the stash, cube, inventory, skill tree and waypoint panels (seen live in D2DLOOP: after the stash chore
		// the inventory and stash flags stayed set and post-death chores looped on "panels still open").
		if ui, err := c.C.UI(c.ctx()); err == nil && (ui.Vendor || c.C.CharacterHint() || ui.Stash || ui.Cube || ui.Inventory || ui.SkillTree || ui.Waypoint) {
			if err := c.C.Key(c.ctx(), "Escape"); err != nil {
				return err
			}
			c.clk().Sleep(200 * time.Millisecond)
			continue
		}
		m, err := c.C.Menu(c.ctx())
		if err != nil {
			return err
		}
		clicked := false
		for _, l := range m.Items {
			if strings.EqualFold(strings.TrimSpace(l.Text), "cancel") {
				if err := c.C.Click(c.ctx(), l.X, l.Y); err != nil {
					return err
				}
				clicked = true
				break
			}
		}
		if !clicked {
			return nil
		}
		c.clk().Sleep(150 * time.Millisecond)
	}
	return errors.New("d2dinput: menus still open after 4 rounds")
}
