// Package d2dworld builds the world nav needs (level rectangle, collision grid by subtile, exits, presets) from d2d
// control verbs instead of Game.exe memory. *World satisfies nav's unexported worldSource interface
// (Update, Current, Level) and is additive: nothing in the Game.exe path uses it. See docs/d2d-backend.md
// "Phase 3: world source status".
//
// Honesty rules: a cell the grid verb did not answer with '.' or '#' is unknown (Grid.Known false, which nav treats
// as not walkable). Any failing verb leaves the previous grid untouched or, on a first fetch, an empty one.
package d2dworld

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsrc"
	"d2lod-bot/internal/d2map"
)

// Verbs is the part of *d2dctl.Client the world source uses.
type Verbs interface {
	Level(ctx context.Context) (d2dctl.LevelInfo, error)
	Grid(ctx context.Context, n int) ([]string, error)
	Warps(ctx context.Context) (d2dctl.List[d2dctl.Warp], error)
	Presets(ctx context.Context) (d2dctl.List[d2dctl.Preset], error)
	NPCs(ctx context.Context) (d2dctl.List[d2dctl.NPC], error)
}

// OriginMode says how d2d cell coordinates relate to the absolute act subtiles of the Game.exe backend.
type OriginMode int

const (
	// OriginNone: cells are level-relative and used as they are (x_sub = floor(5*cell)). This matches
	// d2dsrc.Options{AddOrigin: false}, the default, and is what the spike shows (V: debug players == debug level
	// position, town 57x41 cells).
	OriginNone OriginMode = iota
	// OriginTiles: add 5*wx, 5*wy of "debug level" (the act-tile origin, 5 subtiles per tile). Use together with
	// d2dsrc.Options{AddOrigin: true}.
	OriginTiles
	// OriginSubtiles: add wx, wy as they are, if the origin turns out to be in subtiles (U).
	OriginSubtiles
)

// Options tune a World.
type Options struct {
	Origin OriginMode
	// MaxCells caps the grid cells (subtiles) of one level; 0 is 4096*4096, nav's maxFieldCells and the d2map.Grid cap.
	// A level rectangle (or a reply) larger than that yields no grid rather than a truncated one.
	MaxCells int
	// RefetchDist is how far (subtiles) the player may move from the spot of the last grid fetch before the grid is
	// fetched again (d2d lays rooms near the player, U); 0 is 60. Negative: never, only Refresh.
	RefetchDist int
	// MinRefetch is the least time between two grid fetches of one level; 0 is 2 s.
	MinRefetch time.Duration
	// Timeout bounds the verbs of one Update (0: 10 s); the LinkWait of an arrival comes on top of it. (V live, D2DEXIT:
	// when the bound covered the wait, the bot's 10 s default cut a 90 s LinkWait short and nav reported "no exit",
	// while d2dwalk, which set Timeout = LinkWait+30 s, crossed.)
	Timeout time.Duration
	// LinkWait is how long (wall time) an arrival in a level waits for d2d to link its neighbours, when some are still
	// pending. A lockstep game links them only while it is stepped (V live: 80 s of wall time with no step linked
	// nothing; a few hundred ticks did), so Sleep must advance the game (d2dwalk passes the Pacer's). 0: no wait; the
	// pending retries of Update look again later, and nav may give up on an exit before the neighbour exists.
	LinkWait time.Duration
	Now      func() time.Time                 // test seam (virtual or fake clock for MinRefetch)
	Logf     func(format string, args ...any) // arrival diagnostics (nil: silent)
	Sleep    func(time.Duration)              // LinkWait pause between polls (step the game here), nil: time.Sleep
}

const maxLinkTries = 20

// World is a nav.worldSource over one d2d socket. Not safe for concurrent use, like d2map.Tracker.
type World struct {
	v    Verbs
	o    Options
	levs map[int]*lvl
	cur  *lvl

	resetNext atomic.Bool // ResetNext: forget the levels at the start of the next update, on the reader's goroutine

	// Diagnostics for tools and tests.
	GridFetches    int
	LastLinked     []int // levels the last neighbour list named (diagnostics)
	LinkPolls      int   // neighbour-list polls made while waiting on arrival (LinkWait)
	LastGridErr    error
	LastGridNote   string
	LastBorderNote string
}

type lvl struct {
	m         *d2map.LevelMap
	fetchedAt time.Time
	fetchedP  d2map.Point
	fetched   bool
	ox, oy    int      // origin in subtiles added to cell positions
	names     []string // preset names, parallel to m.Presets
	own       Placement
	unlinked  int // abutting levels missing from the neighbour list (subset of pending)
	pending   int // abutting levels d2d had not linked at the last fetch
	tries     int // refetches spent waiting for them
}

// New wraps a verb source (usually *d2dctl.Client).
func New(v Verbs, o Options) *World {
	if o.MaxCells <= 0 {
		o.MaxCells = 4096 * 4096
	}
	if o.RefetchDist == 0 {
		o.RefetchDist = 60
	}
	if o.MinRefetch == 0 {
		o.MinRefetch = 2 * time.Second
	}
	if o.Timeout == 0 {
		o.Timeout = 10 * time.Second
	}
	if o.Now == nil {
		o.Now = time.Now
	}
	return &World{v: v, o: o, levs: map[int]*lvl{}}
}

func (w *World) logf(f string, a ...any) {
	if w.o.Logf != nil {
		w.o.Logf(f, a...)
	}
}

// Current is the map of the player's level (nil before the first good Update).
func (w *World) Current() *d2map.LevelMap {
	if w.cur == nil {
		return nil
	}
	return w.cur.m
}

// Level returns the cached map of a level visited this game.
func (w *World) Level(no int) *d2map.LevelMap {
	if l := w.levs[no]; l != nil {
		return l.m
	}
	return nil
}

// Reset forgets every level (after a new game: d2d rebuilds levels per game).
func (w *World) Reset() { w.levs, w.cur = map[int]*lvl{}, nil }

// ResetNext is Reset for callers on another goroutine (the d2d restart hook runs in combat's chicken): the maps are
// swapped by the next update, so the reader never sees a half-replaced map. Without any reset after a restart the roam
// coverage of the old game said "everything visited", the planner started a fresh game, and so on: 71 restarts in a row
// (D2DLOOP r21, V live).
func (w *World) ResetNext() { w.resetNext.Store(true) }

// Sub converts a d2d cell position to absolute subtiles for the given origin mode and level origin.
func Sub(mode OriginMode, ox, oy int, cx, cy float64) d2map.Point {
	ax, ay := originSub(mode, ox, oy)
	return d2map.Point{X: ax + d2dctl.SubTile(cx), Y: ay + d2dctl.SubTile(cy)}
}

func originSub(mode OriginMode, ox, oy int) (int, int) {
	switch mode {
	case OriginTiles:
		return 5 * ox, 5 * oy
	case OriginSubtiles:
		return ox, oy
	}
	return 0, 0
}

// Update reads the level and player position; on a level change or when the player has moved far it refetches the
// grid, warps and presets. d2map.ErrNotInGame is returned when the level verb fails (menu, loading screen).
func (w *World) Update() (d2map.State, error) {
	ctx, cancel := w.ctx()
	defer cancel()
	return w.update(ctx, false)
}

func (w *World) ctx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), w.o.Timeout+w.o.LinkWait)
}

// Refresh forces a refetch of the current level's grid and statics on the next call, ignoring the rate limit.
func (w *World) Refresh() (d2map.State, error) {
	ctx, cancel := w.ctx()
	defer cancel()
	return w.update(ctx, true)
}

func (w *World) update(ctx context.Context, force bool) (d2map.State, error) {
	if w.resetNext.Swap(false) {
		w.Reset()
	}
	li, err := w.v.Level(ctx)
	if err != nil || li.W <= 0 || li.H <= 0 || li.ID < 0 {
		if err == nil || !errors.Is(err, context.Canceled) {
			err = d2map.ErrNotInGame
		}
		return d2map.State{}, err
	}
	ox, oy := originSub(w.o.Origin, li.OriginX, li.OriginY)
	rect := d2map.Rect{X: ox, Y: oy, W: 5 * li.W, H: 5 * li.H}
	pos := Sub(w.o.Origin, li.OriginX, li.OriginY, li.X, li.Y)
	st := d2map.State{Pos: pos, LevelNo: li.ID, Act: d2dsrc.ActOf(li.ID)}

	l := w.levs[li.ID]
	if l == nil {
		l = &lvl{m: &d2map.LevelMap{No: li.ID, Rect: rect, Grid: d2map.NewGrid()}}
		w.levs[li.ID] = l
	}
	l.m.Rect, l.ox, l.oy = rect, ox, oy
	if w.cur != l {
		// Level change: the rooms of a level are laid when the player arrives, so whatever was fetched on an earlier
		// visit may be stale; fetch again at once.
		st.LevelChanged = true
		w.cur = l
		force = true
	}
	if force || w.due(l, pos) {
		l.own = Placement{Level: li.ID, X: li.OriginX, Y: li.OriginY, W: li.W, H: li.H}
		st.NewRooms = w.fetch(ctx, l, rect, pos)
		if st.LevelChanged {
			w.logf("d2dworld: arrived in level %d: %s (pending %d, linkwait %v)", li.ID, w.LastBorderNote, l.pending, w.o.LinkWait)
		}
		if st.LevelChanged && l.pending > 0 && w.o.LinkWait > 0 {
			t0 := time.Now()
			w.waitLinked(ctx, l)
			w.logf("d2dworld: link wait over after %v, %d polls: %s (pending %d, linked %v)", time.Since(t0).Round(time.Millisecond), w.LinkPolls, w.LastBorderNote, l.pending, w.LastLinked)
		}
	}
	return st, nil
}

// waitLinked polls the neighbour list in wall time until every abutting level is linked and its border is readable,
// or LinkWait has passed. A newly linked neighbour changes OUR grid too (V live: its edge strip was '#' in the grid
// fetched on arrival and open after the link), so every change of the list refetches the level.
func (w *World) waitLinked(ctx context.Context, l *lvl) {
	sleep := w.o.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	end := time.Now().Add(w.o.LinkWait)
	stale := false // the list changed and the level was not refetched yet
	for time.Now().Before(end) && ctx.Err() == nil {
		if l.pending == 0 || (l.unlinked == 0 && !stale) {
			return // done, or everything is linked and a refetch after the last link still shows closed walls
		}
		sleep(500 * time.Millisecond)
		w.LinkPolls++
		before := len(w.LastLinked)
		w.borders(ctx, l)
		if len(w.LastLinked) != before {
			stale = true
		}
		if stale && l.unlinked == 0 || stale && w.LinkPolls%20 == 0 {
			w.fetch(ctx, l, l.m.Rect, l.fetchedP)
			stale = false
		}
	}
}

func (w *World) due(l *lvl, pos d2map.Point) bool {
	if !l.fetched {
		return true
	}
	if l.pending > 0 && l.tries < maxLinkTries && w.o.Now().Sub(l.fetchedAt) >= w.o.MinRefetch {
		l.tries++ // a neighbour level was still being built: look again soon, not only after moving
		return true
	}
	if w.o.RefetchDist < 0 || w.o.Now().Sub(l.fetchedAt) < w.o.MinRefetch {
		return false
	}
	return math.Hypot(float64(pos.X-l.fetchedP.X), float64(pos.Y-l.fetchedP.Y)) >= float64(w.o.RefetchDist)
}

// fetch loads grid, warps and presets of l. It returns the number of newly known cells, and keeps old data on errors.
func (w *World) fetch(ctx context.Context, l *lvl, rect d2map.Rect, pos d2map.Point) int {
	l.fetchedAt, l.fetchedP, l.fetched = w.o.Now(), pos, true
	before := l.m.Grid.KnownCount()
	if rows, err := w.v.Grid(ctx, 5); err != nil {
		w.LastGridErr, w.LastGridNote = err, "grid verb failed"
	} else {
		w.GridFetches++
		w.LastGridErr = nil
		ng := d2map.NewGrid()
		w.LastGridNote = mergeRows(ng, rect, rows, w.o.MaxCells)
		// nav keys its cached planner field on the Grid pointer and KnownCount only (nav.go fieldFor), so a
		// refetch that changes cells (a door, a relaid room) must hand out a new Grid; an identical one keeps the old.
		if ng.KnownCount() > 0 && !sameGrid(l.m.Grid, ng) {
			l.m.Grid = ng
		}
	}
	w.borders(ctx, l)
	sub := func(x, y float64) d2map.Point {
		return d2map.Point{X: l.ox + d2dctl.SubTile(x), Y: l.oy + d2dctl.SubTile(y)}
	}
	if ws, err := w.v.Warps(ctx); err == nil {
		l.m.Exits = nil
		for _, x := range ws.Items {
			l.m.Exits = append(l.m.Exits, d2map.Exit{Dest: x.To, Pos: sub(x.X, x.Y)})
		}
	}
	if ps, err := w.v.Presets(ctx); err == nil {
		l.m.Presets, l.names = nil, nil
		for _, p := range ps.Items {
			l.m.Presets = append(l.m.Presets, d2map.Preset{Kind: d2map.PresetObject, ID: p.ObjectID, Pos: sub(p.X, p.Y)})
			l.names = append(l.names, p.Name)
		}
	}
	return max(0, l.m.Grid.KnownCount()-before)
}

// borders fills LevelMap.Adjacent from the act layout (fork verbs, see Layouter).
func (w *World) borders(ctx context.Context, l *lvl) {
	lay, ok := w.v.(Layouter)
	if !ok {
		return
	}
	places, err := lay.Layout(ctx)
	if err != nil || len(places) == 0 {
		w.LastBorderNote = "no layout verb: no outdoor borders"
		return
	}
	var linked map[int]bool
	if near, err := lay.Nearby(ctx); err == nil {
		linked = map[int]bool{}
		for _, n := range near {
			linked[n.Level] = true
		}
	}
	var across func(side, sx, sy int) bool
	if p, ok := w.v.(prober); ok && linked != nil {
		across = func(side, sx, sy int) bool {
			// d2d's grid is a tile larger than the layout (V live), so look past the grid's own edge, where only the
			// neighbour's collision can answer.
			px, py := sx, sy
			switch side {
			case 0:
				py = l.m.Rect.Bottom()
			case 1:
				px = l.m.Rect.X - 1
			case 2:
				py = l.m.Rect.Y - 1
			default:
				px = l.m.Rect.Right()
			}
			c, err := p.Blocked(ctx, (float64(px-l.ox)+0.5)/5, (float64(py-l.oy)+0.5)/5)
			return err == nil && !c // an error is closed, never open
		}
	}
	own := l.own
	for _, p := range places {
		if p.Level == own.Level {
			own.W, own.H = p.W, p.H // the layout's size, not the grid's
		}
	}
	lr := d2map.Rect{X: l.m.Rect.X, Y: l.m.Rect.Y, W: 5 * own.W, H: 5 * own.H}
	l.m.Adjacent, l.pending, l.unlinked = adjacent(own, lr, places, linked, l.m.Grid, across)
	if l.pending == 0 {
		l.tries = 0
	}
	w.LastLinked = w.LastLinked[:0]
	for n := range linked {
		w.LastLinked = append(w.LastLinked, n)
	}
	sort.Ints(w.LastLinked)
	w.LastBorderNote = fmt.Sprintf("%d adjacent entries, %d levels not linked yet", len(l.m.Adjacent), l.pending)
}

func sameGrid(a, b *d2map.Grid) bool {
	if a.KnownCount() != b.KnownCount() || a.KnownCount() == 0 {
		return false
	}
	r := a.Bounds()
	if r != b.Bounds() {
		return false
	}
	for y := r.Y; y < r.Bottom(); y++ {
		for x := r.X; x < r.Right(); x++ {
			fa, oa := a.Flags(x, y)
			fb, ob := b.Flags(x, y)
			if fa != fb || oa != ob {
				return false
			}
		}
	}
	return true
}

// mergeRows writes the '.'/'#' rows into g with row 0 / column 0 at the level rectangle's corner. Other characters
// leave the cell unknown. It refuses oversized replies and returns a short note for diagnostics.
func mergeRows(g *d2map.Grid, rect d2map.Rect, rows []string, maxCells int) string {
	width := 0
	for _, r := range rows {
		width = max(width, len(r))
	}
	if len(rows) == 0 || width == 0 {
		return "empty grid reply"
	}
	if int64(len(rows))*int64(width) > int64(maxCells) {
		return "grid reply over the cell cap, ignored"
	}
	note := ""
	if width != rect.W || len(rows) != rect.H {
		note = "grid size differs from the level rectangle (origin or scale U)"
	}
	for y, row := range rows {
		for x := 0; x < len(row); {
			if row[x] != '.' && row[x] != '#' {
				x++
				continue
			}
			x0 := x
			var cells []uint16
			for ; x < len(row) && (row[x] == '.' || row[x] == '#'); x++ {
				c := uint16(0)
				if row[x] == '#' {
					c = d2map.BlockWalk
				}
				cells = append(cells, c)
			}
			g.SetBlock(d2map.Rect{X: rect.X + x0, Y: rect.Y + y, W: len(cells), H: 1}, cells)
		}
	}
	return note
}

// Doors lists the preset positions of the level whose name says door (U: d2d names; a closed door is probably
// blocked in the grid until it is opened).
func (w *World) Doors(no int) []d2map.Point { return w.byName(no, "door") }

// Waypoints lists waypoint positions by object id (d2map.IsWaypointObject) or by name.
func (w *World) Waypoints(no int) []d2map.Point {
	l := w.levs[no]
	if l == nil {
		return nil
	}
	var out []d2map.Point
	for i, p := range l.m.Presets {
		if d2map.IsWaypointObject(p.ID) || strings.Contains(strings.ToLower(l.names[i]), "waypoint") {
			out = append(out, p.Pos)
		}
	}
	return out
}

func (w *World) byName(no int, sub string) []d2map.Point {
	l := w.levs[no]
	if l == nil {
		return nil
	}
	var out []d2map.Point
	for i, p := range l.m.Presets {
		if strings.Contains(strings.ToLower(l.names[i]), sub) {
			out = append(out, p.Pos)
		}
	}
	return out
}

// NPC is a town NPC in absolute subtiles.
type NPC struct {
	Name  string
	Index int // for cmd interact
	Pos   d2map.Point
}

// NPCs asks d2d for the NPCs in view now (positions converted like everything else).
func (w *World) NPCs(ctx context.Context) ([]NPC, error) {
	if w.cur == nil {
		return nil, d2map.ErrNotInGame
	}
	r, err := w.v.NPCs(ctx)
	if err != nil {
		return nil, err
	}
	var out []NPC
	for _, n := range r.Items {
		out = append(out, NPC{n.Name, n.Index, d2map.Point{X: w.cur.ox + d2dctl.SubTile(n.CellX), Y: w.cur.oy + d2dctl.SubTile(n.CellY)}})
	}
	return out, nil
}

// ASCII renders the level grid for manual checks: '.' walkable, '#' blocked, ' ' unknown, 'E' exit, 'W' waypoint (by object id),
// 'D' door, '@' not drawn. One character per step subtiles so that at most maxW x maxH characters come out.
func ASCII(lm *d2map.LevelMap, maxW, maxH int) string {
	r := lm.Rect
	if r.Empty() || maxW <= 0 || maxH <= 0 {
		return ""
	}
	step := max((r.W+maxW-1)/maxW, (r.H+maxH-1)/maxH, 1)
	cols, rows := (r.W+step-1)/step, (r.H+step-1)/step
	buf := make([][]byte, rows)
	for y := range buf {
		buf[y] = make([]byte, cols)
		for x := range buf[y] {
			ch := byte(' ')
			for dy := 0; dy < step; dy++ {
				for dx := 0; dx < step; dx++ {
					px, py := r.X+x*step+dx, r.Y+y*step+dy
					if lm.Grid.Walkable(px, py) {
						ch = '.'
					} else if lm.Grid.Known(px, py) && ch != '.' {
						ch = '#'
					}
				}
			}
			buf[y][x] = ch
		}
	}
	mark := func(p d2map.Point, ch byte) {
		if r.Contains(p.X, p.Y) {
			buf[(p.Y-r.Y)/step][(p.X-r.X)/step] = ch
		}
	}
	for _, p := range lm.Presets {
		switch {
		case d2map.IsWaypointObject(p.ID):
			mark(p.Pos, 'W')
		}
	}
	for _, e := range lm.Exits {
		mark(e.Pos, 'E')
	}
	var sb strings.Builder
	for _, row := range buf {
		sb.Write(row)
		sb.WriteByte('\n')
	}
	return sb.String()
}

// FullyKnown tells nav the level maps arrive whole (`debug level`/layout), so there is nothing to reveal by
// exploring and clearing has to roam instead (nav.roam).
func (w *World) FullyKnown() bool { return true }
