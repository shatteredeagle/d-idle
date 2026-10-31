package town

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/hand"
)

// Entry is a line of an NPC dialog.
type Entry int

const (
	EntryTalk Entry = iota
	EntryTrade
	EntryHeal // not a line in 1.14d: healers heal when the dialog opens; kept for completeness
	EntryGamble
	EntryCancel
	EntryResurrect // hireling NPC with a dead merc (U: never seen live)
)

// NPCMenu places the dialog lines. The box floats above the NPC and is anchored at its bottom: the last line
// (Cancel) is Cancel pixels below the NPC's feet on screen (negative = above) and the lines above it are
// Step2/2 px apart; the box is centred on the NPC x. VERIFIED LIVE at 800x600 on Gheed (Cancel -113,
// lines 15-16 px apart, three positions) and Akara (Cancel -117); the offset varies by a few pixels with the
// NPC's sprite, so pickEntry also tries nearby offsets and remembers the one that worked per NPC class.
var NPCMenu = struct{ DX, Cancel, Step2 int }{DX: 0, Cancel: -115, Step2: 31}

// MenuOrder lists the dialog lines top to bottom per NPC class. Default is Talk, Trade, Cancel.
// Gheed (Talk, Trade, Gamble, Cancel) and Akara (Talk, Trade, Cancel) are VERIFIED LIVE.
var MenuOrder = map[int][]Entry{
	147: {EntryTalk, EntryTrade, EntryGamble, EntryCancel},
	150: {EntryTalk, EntryCancel},
}

var defaultMenu = []Entry{EntryTalk, EntryTrade, EntryCancel}

// menuLine is the index of entry e in NPC class's dialog and the number of lines.
func menuLine(class int, e Entry) (idx, n int, ok bool) {
	order, found := MenuOrder[class]
	if !found {
		order = defaultMenu
	}
	for i, x := range order {
		if x == e {
			return i, len(order), true
		}
	}
	return 0, len(order), false
}

// entryPoint is the click position of line idx of n for an NPC standing at world (x,y) with the given viewport.
func entryPoint(x, y int, vp d2.Point, idx, n int) input.Point {
	f := d2.WorldToScreen(x, y, vp)
	return input.Point{X: f.X + NPCMenu.DX, Y: f.Y + 8 + NPCMenu.Cancel - (n-1-idx)*NPCMenu.Step2/2}
}

// menuAdj are the pixel adjustments pickEntry tries, in order; the one that worked is remembered per class.
// +-15 is one dialog line: Akara's main menu has a fourth line ("Reset Stat/Skill Points", VERIFIED LIVE on i4 after the Den
// reward) that moves Trade one line up in the bottom-anchored box, which the fixed Talk/Trade/Cancel order does not know.
var menuAdj = []int{0, 5, -5, 9, -9, -15, 15, -30}

// Adjustments that stay on the same line (the click may have missed or the hover was late) and those that move a whole
// line (the box has one line more or less than assumed). A click that CLOSED the dialog hit another line (Talk, Cancel,
// Imbue), so the next try jumps to a line-sized adjustment instead of nudging a few pixels (live: Charsi, repeated
// clicks on the same wrong line).
var (
	menuNudge = []int{0, 5, -5, 9, -9}
	menuLines = []int{-15, 15, -30}
)

// nextAdj picks the next untried adjustment: from the line-sized ones after a click that closed the dialog, else the
// nudges first.
func nextAdj(tried map[int]bool, closed bool) (int, bool) {
	order := [][]int{menuNudge, menuLines}
	if closed {
		order = [][]int{menuLines, menuNudge}
	}
	for _, l := range order {
		for _, a := range l {
			if !tried[a] {
				return a, true
			}
		}
	}
	return 0, false
}

// findNPC returns the nearest living monster-table unit whose class is in classes.
func findNPC(s *d2.Snapshot, classes []int) (d2.Unit, bool) {
	var best d2.Unit
	bd, found := 1<<30, false
	for _, m := range s.Monsters {
		if !hasClass(classes, m.Class) || !m.Alive() {
			continue
		}
		if d := int(dist(m.X, m.Y, s.Player.X, s.Player.Y)); d < bd {
			best, bd, found = m, d, true
		}
	}
	return best, found
}

// findObject returns the nearest object of the class.
func findObject(s *d2.Snapshot, class int) (d2.Unit, bool) {
	var best d2.Unit
	bd, found := 1<<30, false
	for _, o := range s.Objects {
		if o.Class != class {
			continue
		}
		if d := int(dist(o.X, o.Y, s.Player.X, s.Player.Y)); d < bd {
			best, bd, found = o, d, true
		}
	}
	return best, found
}

// approach walks to a unit (or to a fallback point when the unit table does not list it yet).
func (t *Town) approach(ctx context.Context, u d2.Unit, have bool, fallback input.Point) error {
	if have && t.noWalk {
		return nil // the game walks to the NPC itself when it is clicked (Greiz stands next to the Sewers stairs, merc.go)
	}
	if have {
		if err := t.mv.GoToUnit(ctx, u.ID); err != nil {
			return t.mv.GoTo(ctx, input.Point{X: u.X, Y: u.Y})
		}
		return nil
	}
	if fallback == (input.Point{}) {
		return ErrNoNPC
	}
	return t.mv.GoTo(ctx, fallback)
}

// offscreenStep is how many subtiles short of an off-screen NPC the nudge stops (U: the screen shows about 20 subtiles
// from the character to the edge; 12 keeps the NPC well inside it without walking next to him).
const offscreenStep = 12

// stopShort is the point on the line from (px,py) to (nx,ny) that is gap subtiles before the target, or the player's
// own position when already closer than that.
func stopShort(px, py, nx, ny int, gap float64) input.Point {
	d := dist(px, py, nx, ny)
	if d <= gap {
		return input.Point{X: px, Y: py}
	}
	f := (d - gap) / d
	return input.Point{X: px + int(float64(nx-px)*f), Y: py + int(float64(ny-py)*f)}
}

// maxInteractDist is the farthest the character may stand from an NPC when he is clicked, in subtiles. A farther click
// is only a walk command, and the greeting wait after it never sees the menu (V live G2 log: dist 7 failed after 13 s,
// dist 2 to 5 opened at once).
const maxInteractDist = 4

// openNPC walks to the first NPC of classes found, clicks it with hover confirmation and waits for the
// dialog. approx is the fallback point for a unit that is not loaded. On success the dialog is open.
func (t *Town) openNPC(ctx context.Context, classes []int, approx input.Point) (d2.Unit, error) {
	var last error = ErrNoNPC
	start := t.e.Clock.Now()
	for try := 0; try < t.tune().NPCOpenTries; try++ {
		if err := ctx.Err(); err != nil {
			return d2.Unit{}, err
		}
		s, err := t.current()
		if err != nil {
			return d2.Unit{}, err
		}
		if hand.PanelOpen(s.UI) {
			t.e.Log.Info("npc: panel open before interact, closing", "classes", classes, "try", try+1, "ui", fmt.Sprint(s.UI))
			if err := t.h.CloseAll(ctx); err != nil {
				return d2.Unit{}, err
			}
			t.shopClass = 0
			if s, err = t.current(); err != nil {
				return d2.Unit{}, err
			}
		}
		npc, ok := findNPC(s, classes)
		if err := t.approach(ctx, npc, ok, approx); err != nil {
			last = fmt.Errorf("%w: %v", ErrNoNPC, err)
			t.e.Log.Warn("npc: walk failed", "classes", classes, "try", try+1, "seen", ok, "err", err)
			continue
		}
		if s, err = t.current(); err != nil { // the NPC may have wandered while we walked
			return d2.Unit{}, err
		}
		if npc, ok = findNPC(s, classes); !ok {
			last = ErrNoNPC
			t.e.Log.Warn("npc: not in the unit table after the walk", "classes", classes, "try", try+1)
			continue
		}
		d := int(dist(npc.X, npc.Y, s.Player.X, s.Player.Y))
		if d > maxInteractDist && !t.noWalk {
			// The walk stopped early (blocked path, wandering NPC). A click from afar makes the game walk the rest
			// while the greeting waits (live G2: dist 7 cost 13 s and a failed try): walk again first.
			t.e.Log.Info("npc: too far to interact, walking closer", "class", npc.Class, "try", try+1, "dist", d)
			if err := t.approach(ctx, npc, true, approx); err != nil {
				last = fmt.Errorf("%w: %v", ErrNoNPC, err)
				continue
			}
			if s, err = t.current(); err != nil {
				return d2.Unit{}, err
			}
			if npc, ok = findNPC(s, classes); !ok {
				last = ErrNoNPC
				continue
			}
			d = int(dist(npc.X, npc.Y, s.Player.X, s.Player.Y))
		}
		t.e.Log.Info("npc: interact", "class", npc.Class, "try", try+1, "dist", d, "npc", [2]int{npc.X, npc.Y})
		clickAt := t.e.Clock.Now()
		npc, err = t.interactFresh(ctx, classes, npc)
		if err != nil {
			last = fmt.Errorf("click %d: %w", npc.Class, err)
			if errors.Is(err, input.ErrOffscreen) && t.noWalk {
				// A click-from-afar NPC that is off the screen cannot be clicked, and approach() will not walk (noWalk):
				// step part of the way so he is on screen, never next to him (U live, docs/act2.md X5).
				if gerr := t.mv.GoTo(ctx, stopShort(s.Player.X, s.Player.Y, npc.X, npc.Y, offscreenStep)); gerr != nil {
					last = fmt.Errorf("%w: nudge toward the NPC: %v", last, gerr)
				}
				continue
			}
			if errors.Is(err, input.ErrOffscreen) || errors.Is(err, input.ErrNotHovered) || errors.Is(err, input.ErrMoveFailed) {
				continue
			}
			return d2.Unit{}, last
		}
		if d > maxInteractDist && !t.noWalk {
			// Not next to him: the click was only a walk command, there is no speech to skip.
			s, err = t.h.WaitFlag(ctx, d2.UINPCMenu, true)
		} else {
			s, npc, err = t.awaitMenu(ctx, classes, npc, clickAt)
		}
		if err != nil {
			last = err
			t.e.Log.Warn("npc: menu did not open", "class", npc.Class, "try", try+1, "err", err)
			continue
		}
		t.e.Log.Info("npc: menu open", "class", npc.Class, "try", try+1, "ms", t.e.Clock.Now().Sub(start).Milliseconds())
		// The dialog box is drawn where the NPC stood when it opened; a wandering NPC moves on afterwards,
		// so the position from this snapshot is the one the entry clicks are computed from.
		for _, m := range s.Monsters {
			if m.ID == npc.ID {
				npc = m
			}
		}
		return npc, nil
	}
	return d2.Unit{}, last
}

// interactFresh clicks the NPC at the position the unit table has right now, not at the position of an older
// snapshot (Akara wanders several subtiles within seconds; a click computed from an old position misses the sprite). A hover
// miss re-reads the position once and retries with it when it changed. It returns the unit as last read.
func (t *Town) interactFresh(ctx context.Context, classes []int, npc d2.Unit) (d2.Unit, error) {
	click := func(u d2.Unit) error {
		return t.h.InteractNPC(ctx, input.Target{Type: TypeMonster, ID: u.ID, Pos: input.Point{X: u.X, Y: u.Y}})
	}
	if s, err := t.current(); err == nil {
		if u, ok := findNPC(s, classes); ok && u.ID == npc.ID {
			npc = u
		}
	}
	err := click(npc)
	if !errors.Is(err, input.ErrNotHovered) {
		return npc, err
	}
	s, serr := t.current()
	if serr != nil {
		return npc, err
	}
	u, ok := findNPC(s, classes)
	if !ok || u.ID != npc.ID {
		return npc, err
	}
	t.e.Log.Info("npc: hover missed, re-reading position", "class", npc.Class, "old", [2]int{npc.X, npc.Y}, "new", [2]int{u.X, u.Y})
	if u.X == npc.X && u.Y == npc.Y {
		return npc, err // same spot: hover is dead, not stale; the caller decides (re-approach or keep waiting)
	}
	return u, click(u)
}

// greetingSlice is how long one wait for the menu lasts before Esc skips the greeting (live: the whole first Akara talk
// took about 2.5 min with 20 s slices; the speech plays once, so the re-click after Esc finds the menu, U).
const greetingSlice = 4 * time.Second

// waitGreeting waits one greetingSlice for the NPC menu; a slice without it ends with an Esc (the greeting is
// dismissed, the caller clicks again). The error is non-nil when the menu did not come.
func (t *Town) waitGreeting(ctx context.Context) (*d2.Snapshot, error) {
	s, ok, err := t.h.Wait(ctx, greetingSlice, func(s *d2.Snapshot) bool { return s.UI.Open(d2.UINPCMenu) })
	if err != nil {
		return s, err
	}
	if ok {
		return s, nil
	}
	_ = t.e.In.CloseAll()
	return nil, errors.New("town: NPC greeting never gave way to the menu")
}

func (t *Town) greetingWait() time.Duration {
	return time.Duration(t.tune().GreetingWaitS) * time.Second
}

// maxMenuCloses is how often a click may close the dialog without the proof flag before pickEntry gives up: a click
// that closes the menu hit Talk (a speech) or Cancel, and clicking the same spot again does not change that.
const maxMenuCloses = 2

// Failure budget of one pickEntry. V from the i5 Fresha log and the G2 soak log of 2026-10-30: 198 of 223 first clicks
// opened the window at the modelled spot within p95 303 ms (max 3.1 s), and in all 18 visits that failed anyway every
// one of 7 further clicks, from 30 px above to 15 px below the line, left the dialog flag set and changed nothing
// (35 s per visit): the dialog ignores clicks, a pixel ladder cannot fix that. So a click gets pickClickWait, the second
// and third click hover slowly, and an ignored third click or pickBudget ends the visit.
const (
	pickClickWait  = 1500 * time.Millisecond
	pickBudget     = 9 * time.Second
	maxIgnoredPick = 3
)

// menuBias is the first pixel adjustment per NPC class. V from the logs: Gheed's four-line box needs +5 (11 of 11
// successes, 7 of 9 first clicks at 0 missed); Akara and Charsi succeed at 0 (197 of 197).
var menuBias = map[int]int{147: 5}

func (e Entry) String() string {
	switch e {
	case EntryTalk:
		return "talk"
	case EntryTrade:
		return "trade"
	case EntryGamble:
		return "gamble"
	case EntryCancel:
		return "cancel"
	}
	return fmt.Sprintf("entry%d", int(e))
}

// slowClick hovers one step off the line, then on it, and clicks after the pointer rested (the game ignores a click
// that comes right after the move, see PickLine).
func (t *Town) slowClick(p input.Point) error {
	if err := t.e.In.MoveMouse(p.X-3, p.Y); err != nil {
		return err
	}
	t.e.Clock.Sleep(150 * time.Millisecond)
	if err := t.e.In.MoveMouse(p.X, p.Y); err != nil {
		return err
	}
	t.e.Clock.Sleep(450 * time.Millisecond)
	return t.e.In.Click(input.Left, p.X, p.Y)
}

// pickEntry clicks a line of the open dialog and waits for the flag that proves the result. Every attempt re-reads
// the game first (is the menu still open, where is the NPC now, did the window open anyway) and logs what it saw.
// A click after which the dialog stays open with unchanged flags was ignored: the next one hovers slowly, and after
// maxIgnoredPick the visit ends instead of laddering pixels. A click that closed the dialog hit another line: reopen
// and jump a line (at most maxMenuCloses times). pickBudget caps the whole thing. flag < 0: just click.
func (t *Town) pickEntry(ctx context.Context, npc d2.Unit, e Entry, flag int) error {
	idx, n, ok := menuLine(npc.Class, e)
	if !ok {
		return fmt.Errorf("town: npc %d has no dialog entry %d", npc.Class, e)
	}
	tried := map[int]bool{}
	start := t.e.Clock.Now()
	closes, ignored, closed := 0, 0, false
	var ui d2.UIState
	fail := func(why string) error {
		t.e.Log.Warn("npc: "+e.String()+" entry not reached because "+why, "class", npc.Class, "ui", fmt.Sprint(ui), "lines", n, "clicks", len(tried), "closes", closes, "ignored", ignored, "ms", t.e.Clock.Now().Sub(start).Milliseconds())
		return fmt.Errorf("town: dialog entry %d of npc %d not reached after %d clicks: %s", e, npc.Class, len(tried), why)
	}
	for try := 0; try < len(menuAdj); try++ {
		adj, have := 0, false
		if a, ok := t.menuAdj[npc.Class]; ok && try == 0 {
			adj, have = a, true // what worked for this NPC last time
		} else if b, ok := menuBias[npc.Class]; ok && try == 0 {
			adj, have = b, true
		} else {
			adj, have = nextAdj(tried, closed)
		}
		if !have {
			break
		}
		tried[adj] = true
		if err := ctx.Err(); err != nil {
			return err
		}
		if try > 0 && t.e.Clock.Now().Sub(start) > pickBudget {
			return fail(fmt.Sprintf("the %v budget is spent", pickBudget))
		}
		s, err := t.current()
		if err != nil {
			return err
		}
		ui = s.UI
		if flag >= 0 && s.UI.Open(flag) { // the window is open already (an earlier click was slow to show it)
			t.e.Log.Info("npc: target window already open", "class", npc.Class, "entry", e, "try", try+1)
			return nil
		}
		if !s.UI.Open(d2.UINPCMenu) { // an earlier click closed it
			closes++
			t.e.Log.Warn("npc: menu closed by the last click, reopening", "class", npc.Class, "entry", e, "try", try+1, "closes", closes, "ui", fmt.Sprint(s.UI))
			if closes > maxMenuCloses {
				return fail(fmt.Sprintf("the dialog closed after %d clicks (the click hits Talk or Cancel)", closes-1))
			}
			approx := t.npcPreset(npc.Class)
			if npc, err = t.openNPC(ctx, []int{npc.Class}, approx); err != nil {
				return err
			}
			if s, err = t.current(); err != nil {
				return err
			}
			ui = s.UI
		}
		for _, m := range s.Monsters { // the NPC may have wandered since the last click: the box follows him
			if m.ID == npc.ID {
				npc = m
			}
		}
		p := entryPoint(npc.X, npc.Y, s.Viewport, idx, n)
		if t.menuEntry != nil {
			if mp, ok := t.menuEntry(e); ok {
				p = mp
			}
		}
		p.Y += adj
		t.e.Log.Info("npc: menu click", "class", npc.Class, "entry", e, "line", idx, "lines", n, "adj", adj, "at", [2]int{p.X, p.Y}, "npc", [2]int{npc.X, npc.Y}, "try", try+1, "slow", ignored > 0)
		if ignored > 0 {
			err = t.slowClick(p)
		} else {
			err = t.e.In.Click(input.Left, p.X, p.Y)
		}
		if err != nil {
			return err
		}
		if flag < 0 {
			return nil
		}
		s2, ok, err := t.h.Wait(ctx, pickClickWait, func(s *d2.Snapshot) bool { return s.UI.Open(flag) })
		if err != nil {
			return err
		}
		if ok {
			if t.menuAdj == nil {
				t.menuAdj = map[int]int{}
			}
			t.menuAdj[npc.Class] = adj
			t.e.Log.Info("npc: window open", "class", npc.Class, "entry", e, "adj", adj, "tries", try+1, "ms", t.e.Clock.Now().Sub(start).Milliseconds())
			return nil
		}
		closed = !s2.UI.Open(d2.UINPCMenu)
		if !closed && s2.UI == ui {
			ignored++
		}
		ui = s2.UI
		t.e.Log.Info("npc: no window after click", "class", npc.Class, "entry", e, "adj", adj, "menuStillOpen", !closed, "ui", fmt.Sprint(s2.UI), "ignored", ignored)
		if ignored >= maxIgnoredPick {
			_ = t.h.CloseAll(ctx)
			return fail(fmt.Sprintf("the dialog ignored %d clicks (flag still set, nothing changed)", ignored))
		}
	}
	return fail("no pixel adjustment worked")
}

// PickLine clicks line idx (0 = top) of the open dialog, which has n lines in total, at whatever menu level it is
// (the box is bottom-anchored, so the count places it). It does not wait for a proof: the caller checks the effect.
// Uses the click offset that worked for this NPC before (pickEntry). UNVERIFIED for submenus beyond Akara's Talk menu.
func (t *Town) PickLine(ctx context.Context, npc d2.Unit, idx, n int) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	if !s.UI.Open(d2.UINPCMenu) {
		return errors.New("town: no NPC dialog is open")
	}
	p := entryPoint(npc.X, npc.Y, s.Viewport, idx, n)
	if t.menuIndex != nil {
		if mp, ok := t.menuIndex(idx); ok {
			p = mp
		}
	}
	p.Y += t.menuAdj[npc.Class]
	// Hover first: the game ignores a click that comes right after the move (LIVE: Akara's Den of Evil row was clicked
	// at once, the menu stayed open with the pointer on the row; same as the vendor tabs).
	// A move from afar sometimes does not register as a hover change (LIVE: two Den clicks with 1 s hover kept the
	// highlight on the previous line); approaching from 3 px away makes the game see two positions.
	if err := t.e.In.MoveMouse(p.X-3, p.Y); err != nil {
		return err
	}
	t.e.Clock.Sleep(150 * time.Millisecond)
	if err := t.e.In.MoveMouse(p.X, p.Y); err != nil {
		return err
	}
	t.e.Clock.Sleep(450 * time.Millisecond)
	if err := t.e.In.Click(input.Left, p.X, p.Y); err != nil {
		return err
	}
	t.e.Clock.Sleep(time.Duration(t.tune().ClickSettleMs) * time.Millisecond)
	return nil
}

// openShopWindow returns the NPC whose trade window is still open from an earlier chore of this trip, when he is one of
// classes: chores of one vendor share one dialog instead of opening and closing it each (live G2: Charsi 4 visits and
// Akara 3 in 13 minutes). The game flag is checked, so a window closed in between is never assumed open.
func (t *Town) openShopWindow(classes []int) (d2.Unit, bool) {
	if t.shopClass == 0 || !hasClass(classes, t.shopClass) {
		return d2.Unit{}, false
	}
	s, err := t.current()
	if err != nil || !s.UI.Open(d2.UIShop) {
		return d2.Unit{}, false
	}
	return findNPC(s, []int{t.shopClass})
}

// openShop opens the trade window of the first NPC of classes.
func (t *Town) openShop(ctx context.Context, classes []int, approx input.Point) (d2.Unit, error) {
	start := t.e.Clock.Now()
	if npc, ok := t.openShopWindow(classes); ok {
		t.e.Log.Info("npc visit: vendor window reused", "class", npc.Class)
		return npc, nil
	}
	t.shopClass = 0
	npc, err := t.openNPC(ctx, classes, approx)
	if err != nil {
		t.e.Log.Warn("npc visit failed", "classes", classes, "stage", "open", "ms", t.e.Clock.Now().Sub(start).Milliseconds(), "err", err)
		return d2.Unit{}, err
	}
	t.tab = 0
	if err := t.pickEntry(ctx, npc, EntryTrade, d2.UIShop); err != nil {
		t.e.Log.Warn("npc visit failed", "class", npc.Class, "stage", "trade", "ms", t.e.Clock.Now().Sub(start).Milliseconds(), "err", err)
		_ = t.h.CloseAll(ctx)
		return d2.Unit{}, fmt.Errorf("trade window: %w", err)
	}
	t.e.Log.Info("npc visit: vendor window open", "class", npc.Class, "ms", t.e.Clock.Now().Sub(start).Milliseconds())
	t.shopClass = npc.Class
	t.pages = t.pages[:0]
	if stock, err := t.Stock(npc); err == nil {
		seen := map[int]bool{}
		for _, u := range stock {
			if !seen[u.Item.Page] {
				seen[u.Item.Page] = true
				t.pages = append(t.pages, u.Item.Page)
			}
		}
		sort.Ints(t.pages)
	}
	return npc, nil
}

// presetPos is the fallback for a unit that is not in the unit table yet: its preset position from the map
// reader (absolute subtiles of THIS game, so it follows the seed). Zero Point when unknown. presetsFn is a test seam.
func (t *Town) presetPos(kind int, ids []int) input.Point {
	var ps []d2map.Preset
	switch {
	case t.presetsFn != nil:
		ps = t.presetsFn()
	case t.e.World != nil:
		_, _ = t.e.World.Update()
		if lm := t.e.World.Current(); lm != nil {
			ps = lm.Presets
		}
	}
	for _, id := range ids {
		for _, p := range ps {
			if p.Kind == kind && p.ID == id {
				return input.Point{X: p.Pos.X, Y: p.Pos.Y}
			}
		}
	}
	return input.Point{}
}

func (t *Town) npcPreset(classes ...int) input.Point { return t.presetPos(d2map.PresetNPC, classes) }
func (t *Town) objPreset(class int) input.Point      { return t.presetPos(d2map.PresetObject, []int{class}) }

// spotOfClass and spotOfObject are the preset-based fallback positions (seed independent).
func (t *Town) spotOfClass(class int) input.Point  { return t.npcPreset(class) }
func (t *Town) spotOfObject(class int) input.Point { return t.objPreset(class) }
