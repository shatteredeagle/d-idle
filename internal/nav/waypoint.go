package nav

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
)

// wpLevels lists the waypoint levels of each act in the order of the panel (levels.txt ids).
var wpLevels = [5][]int{
	{1, 3, 4, 5, 6, 27, 29, 32, 35},
	{40, 48, 42, 57, 43, 44, 52, 74, 46},
	{75, 76, 77, 78, 79, 80, 81, 83, 101},
	{103, 106, 107},
	{109, 111, 112, 113, 115, 123, 117, 118, 129},
}

// waypointSlot returns the act tab and the row of a waypoint level.
func waypointSlot(levelNo int) (act, row int, ok bool) {
	for a, l := range wpLevels {
		for r, id := range l {
			if id == levelNo {
				return a, r, true
			}
		}
	}
	return 0, 0, false
}

// Waypoint panel geometry at 800x600 (client pixels). Measured from a screenshot of the opened panel in the
// Rogue Encampment (VERIFIED LIVE for act 1, incl. clicking a row to travel: tab "I" at (115,77), area names at x 160..300 and y 135 + 34*row,
// nine fixed rows, inactive ones grey). Tabs of acts 2-5 only appear once a waypoint there is activated, so their x positions are an
// extrapolation at the same pitch (UNVERIFIED).
var wpPanel = struct {
	TabX        [5]int // centre of each act tab
	TabY        int
	RowX, RowY0 int // text of row 0
	RowDY       int // row pitch
	IconX       int // the row icon column; IconDY is its y offset from the text line
	IconDY      int
	IconFrom    int // rows from this one on ignore clicks on the text and only react to the icon
}{
	TabX: [5]int{116, 173, 230, 287, 344}, TabY: 77, RowX: 230, RowY0: 135, RowDY: 34,
	// V live (act2live5, Act 2): rows 0..6 respond at x=230, rows 7 (Arcane) and 8 (Canyon) do nothing at x=230 or
	// x=350 and travel when the icon is clicked at x=112, y=376 / 410. Icons of rows 0..6 are U.
	IconX: 112, IconDY: 2, IconFrom: 7,
}

// ErrCursorItem: an item is on the cursor and could not be put away; game.exe does not open the waypoint panel then
// (D2D-V: FUN_00535060 busy test).
var ErrCursorItem = errors.New("nav: item on the cursor blocks the waypoint panel")

// wpRetryWindow: a row click that did nothing is retried while the character arrived this recently (U: the 10 s
// cooldown is measured by the game from its own clock, ours starts at our first look at the level).
const wpRetryWindow = 8 * time.Second

// noteLevel remembers when the character was first seen in a level; the waypoint cooldown counts from there.
func (n *Navigator) noteLevel(s *d2.Snapshot) {
	if s != nil && s.State == d2.StateInGame && s.Player != nil && s.Player.Area != 0 && s.Player.Area != n.wpLevel {
		n.wpLevel, n.wpSince = s.Player.Area, n.clk.Now()
	}
}

// waitCooldown sleeps until game.exe will accept a waypoint travel again (10 s since the last level change, D2D-V
// FUN_0054c5d0, silently ignored otherwise). It returns how long it waited.
func (n *Navigator) waitCooldown(ctx context.Context) (time.Duration, error) {
	cool := time.Duration(n.tune().WpCooldownSec*float64(time.Second)) + time.Second
	start := n.clk.Now()
	for {
		left := cool - n.clk.Now().Sub(n.wpSince)
		if left <= 0 || n.wpSince.IsZero() {
			return n.clk.Now().Sub(start), nil
		}
		if err := ctx.Err(); err != nil {
			return n.clk.Now().Sub(start), err
		}
		n.clk.Sleep(min(left, time.Second))
	}
}

// TakeWaypoint walks to the nearest waypoint on the current level, opens its panel and picks levelNo.
// Panel geometry and tabs: see wpPanel (calibrated live, docs/nav.md). The click on a level whose waypoint was
// never touched does nothing, which shows up as ErrNoWaypoint (the level did not change).
func (n *Navigator) TakeWaypoint(ctx context.Context, levelNo int) error {
	t := n.tune()
	act, row, ok := waypointSlot(levelNo)
	if !ok {
		return fmt.Errorf("%w: level %d has no waypoint", ErrNoWaypoint, levelNo)
	}
	o, err := n.observeWait(ctx)
	if err != nil {
		return err
	}
	if o.level == levelNo {
		return nil
	}
	if err := n.clearCursor(ctx, o.snap); err != nil {
		return err
	}
	from := o.level
	n.wpClicks = 0
	if err := n.openWaypoint(ctx); err != nil {
		return err
	}
	if n.wpClicks > 1 {
		n.log.Info("wp: activated", "area", from, "clicks", n.wpClicks)
	}
	// After the walk and the panel, so the walk counts toward the cooldown.
	waited, err := n.waitCooldown(ctx)
	if err != nil {
		_ = n.in.CloseAll()
		return err
	}
	// the panel opens on the current act's tab; click the target act's tab, then the row
	if err := n.in.Click(input.Left, wpPanel.TabX[act], wpPanel.TabY); err != nil {
		return err
	}
	n.clk.Sleep(250 * time.Millisecond)
	// Which column works depends on the row (wpPanel); try the likelier one first and fall back to the other while
	// the panel is still open. A click on a dead row does nothing, so the area change is the verdict. A second round
	// follows for a character that just arrived: the game eats a row click inside its cooldown.
	y := wpPanel.RowY0 + row*wpPanel.RowDY
	cols := [2][2]int{{wpPanel.RowX, y}, {wpPanel.IconX, y + wpPanel.IconDY}}
	if row >= wpPanel.IconFrom {
		cols[0], cols[1] = cols[1], cols[0]
	}
	age0 := n.clk.Now().Sub(n.wpSince) // how long the character has stood here when the first row is clicked
	for round := 0; round < 2; round++ {
		if round > 0 {
			if s, err := n.st.Snapshot(); err != nil || !s.UI.Open(d2.UIWaypoint) || age0 > wpRetryWindow+time.Duration(t.WpCooldownSec*float64(time.Second)) {
				break
			}
			n.clk.Sleep(2 * time.Second)
			if _, err := n.waitCooldown(ctx); err != nil {
				return err
			}
		}
		for i, c := range cols {
			if i > 0 || round > 0 {
				if s, err := n.st.Snapshot(); err != nil || !s.UI.Open(d2.UIWaypoint) {
					break // the panel closed on its own: nothing left to click
				}
			}
			if err := n.in.Click(input.Left, c[0], c[1]); err != nil {
				return err
			}
			n.wpClicks++
			wait := t.LoadWait()
			if i == 0 {
				wait = min(wait, 3*time.Second)
			}
			if err := n.waitArea(ctx, wait, levelNo); err == nil {
				n.log.Info(fmt.Sprintf("wp: used %d -> %d (%d clicks, waited %d ms)", from, levelNo, n.wpClicks, waited.Milliseconds()))
				return nil
			} else if !errors.Is(err, ErrNoWaypoint) {
				return err
			}
		}
	}
	_ = n.in.CloseAll()
	return fmt.Errorf("%w: level %d not reached (waypoint not activated?)", ErrNoWaypoint, levelNo)
}

// waitArea polls until the player stands in the level; ErrNoWaypoint on timeout. A loading screen extends the
// deadline (V live ACT2LIVE6: the Canyon load outlasted the fixed wait) up to LoadWait beyond it.
func (n *Navigator) waitArea(ctx context.Context, wait time.Duration, levelNo int) error {
	t := n.tune()
	end := n.clk.Now().Add(wait)
	hard := end.Add(t.LoadWait())
	for n.clk.Now().Before(end) {
		if err := ctx.Err(); err != nil {
			return err
		}
		n.clk.Sleep(t.Poll())
		s, err := n.st.Snapshot()
		if err != nil {
			return err
		}
		n.noteLevel(s)
		if s.State == d2.StateInGame && s.Player != nil && s.Player.Area == levelNo {
			n.clk.Sleep(500 * time.Millisecond)
			return nil
		}
		if s.State != d2.StateInGame && end.Before(n.clk.Now().Add(2*time.Second)) {
			end = n.clk.Now().Add(2 * time.Second)
			if end.After(hard) {
				end = hard
			}
		}
	}
	return ErrNoWaypoint
}

func (n *Navigator) nearestWaypointUnit(o *obs) *d2.Unit {
	var best *d2.Unit
	bd := 1e9
	for i := range o.snap.Objects {
		u := &o.snap.Objects[i]
		if !d2map.IsWaypointObject(u.Class) {
			continue
		}
		if d := dist(o.pos, input.Point{X: u.X, Y: u.Y}); d < bd {
			best, bd = u, d
		}
	}
	return best
}

func nearestPreset(p input.Point, ps []d2map.Preset) d2map.Preset {
	best := ps[0]
	for _, q := range ps[1:] {
		if dist(p, input.Point{X: q.Pos.X, Y: q.Pos.Y}) < dist(p, input.Point{X: best.Pos.X, Y: best.Pos.Y}) {
			best = q
		}
	}
	return best
}

// openWaypoint walks to the nearest waypoint of the current level and interacts until its panel is open.
// Touching a waypoint activates it (UNVERIFIED until the live run in docs/nav.md).
func (n *Navigator) openWaypoint(ctx context.Context) error {
	t := n.tune()
	o, err := n.observeWait(ctx)
	if err != nil {
		return err
	}
	var wp *d2.Unit
	for attempt := 0; attempt < 3 && wp == nil; attempt++ {
		wp = n.nearestWaypointUnit(o)
		if wp == nil {
			// the live unit appears when the character is near the preset: walk toward it
			pre := o.lm.Waypoints()
			if len(pre) == 0 {
				// The preset is missing until its room has streamed in (V live: Lut Gholein, "none on level 40" right
				// after loading, fine after a walk across the town): walk to the known position, then look again.
				kp, ok := knownWaypointPos[o.level]
				if !ok || attempt > 0 && dist(o.pos, kp) < unitTolerance+2 {
					return fmt.Errorf("%w: none on level %d", ErrNoWaypoint, o.level)
				}
				pre = []d2map.Preset{{Kind: d2map.PresetObject, Pos: d2map.Point{X: kp.X, Y: kp.Y}}}
			}
			p := nearestPreset(o.pos, pre)
			// Not onto the preset: the object's footprint blocks, the stuck ladder then clicks the waypoint and opens its
			// panel, and the next walk click lands on the panel's first row (VERIFIED LIVE: Cold Plains, the character
			// was sent to the Rogue Encampment). Stop inside interact range like GoToUnit does.
			pp := input.Point{X: p.Pos.X, Y: p.Pos.Y}
			if err := n.walk(ctx, func(*obs) (input.Point, float64, error) { return pp, unitTolerance + 1, nil }, walkOpts{}); err != nil {
				return err
			}
			if o, err = n.observeWait(ctx); err != nil {
				return err
			}
		}
	}
	if wp == nil {
		return fmt.Errorf("%w: unit not found", ErrNoWaypoint)
	}
	if err := n.GoToUnitOf(ctx, d2.TypeObject, wp.ID); err != nil {
		return err
	}
	opened := false
	for try := 0; try < 3 && !opened; try++ {
		o, err = n.observeWait(ctx)
		if err != nil {
			return err
		}
		if u := findUnit(o.snap, d2.TypeObject, wp.ID); u != nil {
			wp = u
		}
		n.wpClicks++
		if err := n.in.Interact(input.Target{Type: uint32(d2.TypeObject), ID: wp.ID, Pos: input.Point{X: wp.X, Y: wp.Y}}); err != nil {
			n.log.Info("nav waypoint interact", "try", try, "err", err)
			n.clk.Sleep(300 * time.Millisecond)
			// Something else under the pointer (a monster standing on the waypoint) or a bad angle: come at it from another side.
			if errors.Is(err, input.ErrNotHovered) {
				if off, ok := n.stepOff(o, input.Point{X: wp.X, Y: wp.Y}); ok {
					_ = n.GoTo(ctx, off)
					_ = n.GoToUnitOf(ctx, d2.TypeObject, wp.ID)
				}
			}
			// Plain clicks on the object (V live: Lut Gholein waypoint, Interact alternates between "never hovered" and
			// success). A second click on an open panel would close it again, so each is followed by a panel check.
			if errors.Is(err, input.ErrNotHovered) || errors.Is(err, input.ErrNoHover) {
				for _, off := range [][2]int{{0, 0}, {-1, -1}, {1, 1}} {
					n.wpClicks++
					if opened = n.clickWaypoint(input.Point{X: wp.X + off[0], Y: wp.Y + off[1]}); opened {
						break
					}
				}
			}
			continue
		}
		opened = n.panelOpens(t.PanelWait())
	}
	if !opened {
		_ = n.in.CloseAll()
		return fmt.Errorf("%w: panel did not open", ErrNoWaypoint)
	}
	return nil
}

func (n *Navigator) panelOpens(wait time.Duration) bool {
	end := n.clk.Now().Add(wait)
	for n.clk.Now().Before(end) {
		n.clk.Sleep(150 * time.Millisecond)
		if s, err := n.st.Snapshot(); err == nil && s.State == d2.StateInGame && s.UI.Open(d2.UIWaypoint) {
			return true
		}
	}
	return false
}

func (n *Navigator) clickWaypoint(p input.Point) bool {
	if err := n.in.ClickWorld(p); err != nil {
		return false
	}
	return n.panelOpens(n.tune().PanelWait())
}

// ActivateWaypoint finds the waypoint of the current level (exploring until its preset shows up), walks onto
// it, opens the panel (that is what activates it) and closes the panel again.
func (n *Navigator) ActivateWaypoint(ctx context.Context) error {
	o, err := n.observeWait(ctx)
	if err != nil {
		return err
	}
	if len(o.lm.Waypoints()) == 0 && n.nearestWaypointUnit(o) == nil {
		n.log.Info("nav no waypoint known yet, exploring", "level", o.level)
		err := n.explore(ctx, func() bool {
			lm := n.w.Current()
			return lm != nil && len(lm.Waypoints()) > 0
		}, nil)
		if errors.Is(err, ErrExplored) {
			return fmt.Errorf("%w: none found on level %d", ErrNoWaypoint, o.level)
		}
		if err != nil {
			return err
		}
	}
	if err := n.openWaypoint(ctx); err != nil {
		return err
	}
	n.clk.Sleep(300 * time.Millisecond)
	return n.in.CloseAll()
}

// ActivateWaypointNear is the opportunistic form of ActivateWaypoint: it touches the waypoint of the current level only
// when the waypoint is already known (live unit or preset) and at most maxDist subtiles away in a straight line, and
// never explores. touched is false with a nil error when there is nothing near enough to touch.
func (n *Navigator) ActivateWaypointNear(ctx context.Context, maxDist float64) (touched bool, err error) {
	o, err := n.observeWait(ctx)
	if err != nil {
		return false, err
	}
	near := false
	if u := n.nearestWaypointUnit(o); u != nil {
		near = dist(o.pos, input.Point{X: u.X, Y: u.Y}) <= maxDist
	} else if pre := o.lm.Waypoints(); len(pre) > 0 {
		p := nearestPreset(o.pos, pre)
		near = dist(o.pos, input.Point{X: p.Pos.X, Y: p.Pos.Y}) <= maxDist
	}
	if !near {
		return false, nil
	}
	if err := n.openWaypoint(ctx); err != nil {
		return false, err
	}
	n.clk.Sleep(300 * time.Millisecond)
	return true, n.in.CloseAll()
}

// clearCursor puts a held item away through the ClearCursor hook; without one the trip is refused.
func (n *Navigator) clearCursor(ctx context.Context, s *d2.Snapshot) error {
	if s == nil || s.Inventory == nil || s.Inventory.Cursor == nil {
		return nil
	}
	if n.ClearCursor == nil {
		return ErrCursorItem
	}
	if err := n.ClearCursor(ctx); err != nil {
		return fmt.Errorf("%w: %v", ErrCursorItem, err)
	}
	if s, err := n.st.Snapshot(); err == nil && s.Inventory != nil && s.Inventory.Cursor != nil {
		return ErrCursorItem
	}
	return nil
}
