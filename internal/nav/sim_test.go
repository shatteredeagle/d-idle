package nav

import (
	"io"
	"log/slog"
	"math"
	"os"
	"sort"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

// simLevel is one level of the simulated world.
type simLevel struct {
	no    int
	rect  d2map.Rect
	known *d2map.Grid
	lm    *d2map.LevelMap
}

// sim is a tiny game: one truth grid, levels as rectangles of it, a player that walks straight toward the last
// clicked point and stops at the first cell the truth grid blocks, 40x40 rooms that join the known grid when the
// player comes near, and a fake clock that moves the world whenever something sleeps. It is the world source, the
// state source, the input controller and the clock of a Navigator under test.
type sim struct {
	neverReveal []d2map.Rect // rooms touching these never load (models unloadable regions)
	mode        int          // player mode: 1 neutral, 0/17 dead
	travelled   float64
	onStep      func(s *sim)
	knownSrc    *d2map.Grid // what rooms show the navigator (defaults to truth); differs to model invisible obstacles
	truth       *d2map.Grid
	levels      []*simLevel
	px, py      float64
	target      *input.Point
	speed       float64
	now         time.Time
	reveal      int
	loadTicks   int // observations left that report the loading state

	units         []d2.Unit
	objects       []d2.Unit
	ui            d2.UIState
	life, maxLife int

	mana, maxMana int     // maxMana 0: not simulated
	manaRegen     float64 // per second
	casts         []input.Point
	castCost      int
	noTeleport    bool // casts do nothing
	voidUnknown   bool // truth cells that are not in the truth grid never become known

	clicks       []input.Point
	rawClicks    [][2]int
	interacts    []input.Target
	keys         []string
	closeAlls    int
	onInteract   func(s *sim, t input.Target) error
	onClick      func(s *sim, x, y int)
	onClickWorld func(s *sim, p input.Point)
	solidUnits   bool
	revealedRoom map[[3]int]bool
	updates      int
	cursor       *d2.Unit // item held on the cursor
}

func newSim(truth *d2map.Grid, start input.Point, levels ...*simLevel) *sim {
	s := &sim{truth: truth, px: float64(start.X) + 0.5, py: float64(start.Y) + 0.5, speed: 6, reveal: 45,
		now: time.Unix(1000, 0), life: 100, maxLife: 100, mode: 1, revealedRoom: map[[3]int]bool{}}
	if len(levels) == 0 {
		b := truth.Bounds()
		levels = []*simLevel{{no: 2, rect: b}}
	}
	for _, l := range levels {
		l.known = d2map.NewGrid()
		l.lm = &d2map.LevelMap{No: l.no, Rect: l.rect, Grid: l.known}
	}
	s.levels = levels
	for _, a := range levels {
		for _, b := range levels {
			if a == b {
				continue
			}
			c := contact(a.rect, b.rect)
			if !c.Empty() {
				a.lm.Adjacent = append(a.lm.Adjacent, d2map.Adjacent{LevelNo: b.no, Rect: b.rect, Contact: c})
			}
		}
	}
	return s
}

func contact(a, b d2map.Rect) d2map.Rect {
	x0, y0 := max(a.X, b.X-1), max(a.Y, b.Y-1)
	x1, y1 := min(a.Right(), b.Right()+1), min(a.Bottom(), b.Bottom()+1)
	if x1 <= x0 || y1 <= y0 {
		return d2map.Rect{}
	}
	return d2map.Rect{X: x0, Y: y0, W: x1 - x0, H: y1 - y0}
}

func (s *sim) levelAt(x, y int) *simLevel {
	for _, l := range s.levels {
		if l.rect.Contains(x, y) {
			return l
		}
	}
	return s.levels[0]
}

func (s *sim) cur() *simLevel { return s.levelAt(int(s.px), int(s.py)) }

func (s *sim) pos() input.Point {
	return input.Point{X: int(math.Floor(s.px)), Y: int(math.Floor(s.py))}
}

// revealRooms copies the 40x40 rooms near the player from the truth grid into the level grids.
func (s *sim) revealRooms() {
	b := s.truth.Bounds()
	for ry := (int(s.py) - s.reveal) / 40 * 40; ry <= int(s.py)+s.reveal; ry += 40 {
		for rx := (int(s.px) - s.reveal) / 40 * 40; rx <= int(s.px)+s.reveal; rx += 40 {
			if rx < b.X-40 || ry < b.Y-40 || rx >= b.Right() || ry >= b.Bottom() {
				continue
			}
			if s.never(rx, ry) {
				continue
			}
			for _, l := range s.levels {
				key := [3]int{rx, ry, l.no}
				if s.revealedRoom[key] {
					continue
				}
				r := d2map.Rect{X: max(rx, l.rect.X), Y: max(ry, l.rect.Y)}
				r.W = min(rx+40, l.rect.Right()) - r.X
				r.H = min(ry+40, l.rect.Bottom()) - r.Y
				r.W, r.H = min(r.W, b.Right()-r.X), min(r.H, b.Bottom()-r.Y)
				if r.Empty() {
					continue
				}
				if s.voidUnknown { // cells missing from the truth stay unknown, like the Arcane void
					for y := r.Y; y < r.Bottom(); y++ {
						for x := r.X; x < r.Right(); x++ {
							if f, ok := s.knownGrid().Flags(x, y); ok {
								l.known.SetBlock(d2map.Rect{X: x, Y: y, W: 1, H: 1}, []uint16{f})
							}
						}
					}
					s.revealedRoom[key] = true
					continue
				}
				cells := make([]uint16, 0, r.W*r.H)
				for y := r.Y; y < r.Bottom(); y++ {
					for x := r.X; x < r.Right(); x++ {
						f, _ := s.knownGrid().Flags(x, y)
						cells = append(cells, f)
					}
				}
				l.known.SetBlock(r, cells)
				s.revealedRoom[key] = true
			}
		}
	}
}

func (s *sim) knownGrid() *d2map.Grid {
	if s.knownSrc != nil {
		return s.knownSrc
	}
	return s.truth
}

func (s *sim) never(rx, ry int) bool {
	for _, r := range s.neverReveal {
		if r.X < rx+40 && rx < r.Right() && r.Y < ry+40 && ry < r.Bottom() {
			return true
		}
	}
	return false
}

func (s *sim) walkableTruth(x, y int) bool {
	if !s.truth.Walkable(x, y) {
		return false
	}
	if s.solidUnits {
		for _, u := range s.units {
			if math.Hypot(float64(u.X-x), float64(u.Y-y)) < 1.5 {
				return false
			}
		}
	}
	return true
}

// step moves the player toward the target for dt.
func (s *sim) step(dt time.Duration) {
	s.now = s.now.Add(dt)
	if s.maxMana > 0 {
		s.mana = min(s.maxMana, s.mana+int(s.manaRegen*dt.Seconds()+0.5))
	}
	if s.onStep != nil {
		s.onStep(s)
	}
	if s.target == nil {
		return
	}
	tx, ty := float64(s.target.X)+0.5, float64(s.target.Y)+0.5
	left := s.speed * dt.Seconds()
	for left > 0 {
		dx, dy := tx-s.px, ty-s.py
		d := math.Hypot(dx, dy)
		if d < 0.05 {
			s.target = nil
			return
		}
		k := math.Min(d, math.Min(left, 0.25))
		nx, ny := s.px+dx/d*k, s.py+dy/d*k
		// a diagonal step may not cut a blocked corner
		if !s.walkableTruth(int(nx), int(ny)) || !s.walkableTruth(int(nx), int(s.py)) || !s.walkableTruth(int(s.px), int(ny)) {
			s.target = nil
			return
		}
		s.travelled += math.Hypot(nx-s.px, ny-s.py)
		s.px, s.py = nx, ny
		left -= k
	}
}

// --- env.Clock
func (s *sim) Now() time.Time        { return s.now }
func (s *sim) Sleep(d time.Duration) { s.step(d) }

// --- world source
func (s *sim) Update() (d2map.State, error) {
	s.updates++
	if s.loadTicks > 0 {
		return d2map.State{}, d2map.ErrNotInGame
	}
	s.revealRooms()
	l := s.cur()
	return d2map.State{Pos: d2map.Point{X: int(s.px), Y: int(s.py)}, LevelNo: l.no}, nil
}
func (s *sim) Current() *d2map.LevelMap { return s.cur().lm }
func (s *sim) Level(no int) *d2map.LevelMap {
	for _, l := range s.levels {
		if l.no == no {
			return l.lm
		}
	}
	return nil
}

// --- state source
func (s *sim) Snapshot() (*d2.Snapshot, error) {
	if s.loadTicks > 0 {
		s.loadTicks--
		return &d2.Snapshot{State: d2.StateLoading}, nil
	}
	p := &d2.Player{X: int(math.Floor(s.px)), Y: int(math.Floor(s.py)), Area: s.cur().no, Mode: s.mode, Life: s.life, MaxLife: s.maxLife, Mana: s.mana, MaxMana: s.maxMana}
	return &d2.Snapshot{State: d2.StateInGame, Player: p, Monsters: append([]d2.Unit(nil), s.units...),
		Objects: append([]d2.Unit(nil), s.objects...), UI: s.ui, Inventory: s.inventory()}, nil
}

// --- input.Controller
func (s *sim) MoveMouse(x, y int) error { return nil }
func (s *sim) Click(b input.Button, x, y int, mods ...input.Mod) error {
	s.rawClicks = append(s.rawClicks, [2]int{x, y})
	if s.onClick != nil {
		s.onClick(s, x, y)
	}
	return nil
}
func (s *sim) Key(name string) error     { s.keys = append(s.keys, name); return nil }
func (s *sim) Hold(string) error         { return nil }
func (s *sim) Release(string) error      { return nil }
func (s *sim) Pos() (input.Point, error) { return s.pos(), nil }
func (s *sim) ClickWorld(w input.Point) error {
	s.clicks = append(s.clicks, w)
	if s.onClickWorld != nil {
		s.onClickWorld(s, w)
	}
	t := w
	s.target = &t
	return nil
}
func (s *sim) MoveTo(w input.Point, _ input.MoveOpts) error { return s.ClickWorld(w) }
func (s *sim) SelectSkill(string) error                     { return nil }
func (s *sim) CastAt(_ string, w input.Point) error {
	s.casts = append(s.casts, w)
	if s.noTeleport || !s.walkableTruth(w.X, w.Y) || s.maxMana > 0 && s.mana < s.castCost {
		return nil
	}
	s.mana -= s.castCost
	s.px, s.py = float64(w.X)+0.5, float64(w.Y)+0.5
	s.target = nil
	return nil
}
func (s *sim) Interact(t input.Target) error {
	s.interacts = append(s.interacts, t)
	if s.onInteract != nil {
		return s.onInteract(s, t)
	}
	return nil
}
func (s *sim) CloseAll() error { s.closeAlls++; s.ui = d2.UIState{}; return nil }

var _ input.Controller = (*sim)(nil)

// navigator builds a Navigator wired to the sim with default tuning.
func (s *sim) navigator() *Navigator {
	tune := env.DefaultTuning()
	var lg *slog.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	if os.Getenv("NAVLOG") != "" {
		lg = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug}))
	}
	e := &env.Env{ID: "test", Log: lg, In: s, Clock: s, Tune: &tune}
	n := New(e)
	n.w, n.st = s, s
	return n
}

// tuning returns the sim navigator's tuning for modification.
func tuningOf(n *Navigator) *env.Tuning {
	return n.e.Tune
}

// asciiTruth builds a truth grid from rows ('#' wall, '.' floor) at origin (x0,y0).
func asciiTruth(x0, y0 int, rows ...string) *d2map.Grid { return d2map.ParseASCII(x0, y0, rows...) }

// open returns a w x h open field at (x0,y0) surrounded by nothing (all walkable).
func openField(x0, y0, w, h int) *d2map.Grid {
	g := d2map.NewGrid()
	cells := make([]uint16, w*h)
	g.SetBlock(d2map.Rect{X: x0, Y: y0, W: w, H: h}, cells)
	return g
}

func blockRect(g *d2map.Grid, r d2map.Rect) {
	cells := make([]uint16, r.W*r.H)
	for i := range cells {
		cells[i] = d2map.BlockWalk
	}
	g.SetBlock(r, cells)
}

func sortedKeys[T any](m map[[2]int]T) [][2]int {
	var ks [][2]int
	for k := range m {
		ks = append(ks, k)
	}
	sort.Slice(ks, func(i, j int) bool { return ks[i][0] < ks[j][0] || ks[i][0] == ks[j][0] && ks[i][1] < ks[j][1] })
	return ks
}

func d2Unit(id uint32, class, x, y int) d2.Unit {
	return d2.Unit{ID: id, Type: d2.TypeObject, Class: class, X: x, Y: y}
}

func (s *sim) inventory() *d2.Inventory {
	if s.cursor == nil {
		return nil
	}
	return &d2.Inventory{Cursor: s.cursor}
}
