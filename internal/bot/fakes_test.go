package bot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/launch"
)

// vclock is a virtual clock: Sleep advances time and yields briefly so goroutines interleave.
type vclock struct {
	mu  sync.Mutex
	now time.Time
}

func newClock() *vclock { return &vclock{now: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)} }

func (c *vclock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }
func (c *vclock) Sleep(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	c.mu.Unlock()
	time.Sleep(50 * time.Microsecond)
}

// world is the fake game: one mutable state behind a Snapshot function and the fake behaviors.
type world struct {
	mu         sync.Mutex
	snaps      atomic.Int64 // snapshots taken, so tests can wait until the bot has looked again
	state      d2.State
	area       int
	x, y       int
	life, maxL int
	level      int
	exp        uint32
	mode       int
	diff       d2.Difficulty
	act        int
	mons       []d2.Unit
	items      []d2.Unit
	calls      []string
	statPts    int
	skillPts   int
	snapErr    error
	gold0      int
	ground     []d2.Unit
	bossImmune bool
}

func newWorld() *world {
	w := &world{state: d2.StateInGame, area: 1, life: 100, maxL: 100, level: 1, mode: 1, x: 100, y: 100, gold0: 500}
	// weapon and body armor on: a character without them is gated into the first field (geargate.go)
	for i, body := range []int{d2.BodyRHand, d2.BodyTorso} {
		w.items = append(w.items, d2.Unit{ID: uint32(9000 + i), Type: d2.TypeItem, Class: 2, Item: &d2.Item{Where: d2.ContEquipment, Body: body}})
	}
	return w
}

func (w *world) call(f string, a ...any) {
	w.mu.Lock()
	w.calls = append(w.calls, fmt.Sprintf(f, a...))
	w.mu.Unlock()
}

func (w *world) log() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return strings.Join(w.calls, " | ")
}

func (w *world) count(prefix string) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	n := 0
	for _, c := range w.calls {
		if strings.HasPrefix(c, prefix) {
			n++
		}
	}
	return n
}

func (w *world) gold(n int) { w.gold0 = n }

func (w *world) set(f func(w *world)) { w.mu.Lock(); f(w); w.mu.Unlock() }

func (w *world) snapshot() (*d2.Snapshot, error) {
	w.snaps.Add(1)
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.snapErr != nil {
		return nil, w.snapErr
	}
	s := &d2.Snapshot{State: w.state}
	if w.state != d2.StateInGame {
		return s, nil
	}
	s.Player = &d2.Player{ID: 1, Name: "Testy", Class: d2.Sorceress, Area: w.area, X: w.x, Y: w.y, Life: w.life, MaxLife: w.maxL,
		Mana: 50, MaxMana: 50, Level: w.level, Exp: w.exp, Mode: w.mode, Difficulty: w.diff, Act: w.act, StatPoints: w.statPts, SkillPoints: w.skillPts, Gold: w.gold0,
		Resist: d2.Resists{Fire: 75, Cold: 75, Lightning: 75, Poison: 75}}
	s.Inventory = &d2.Inventory{Items: append([]d2.Unit(nil), w.items...)}
	s.Monsters = append([]d2.Unit(nil), w.mons...)
	s.Ground = append([]d2.Unit(nil), w.ground...)
	return s, nil
}

func (w *world) move(area int) { w.mu.Lock(); w.area = area; w.x += 50; w.mu.Unlock() }

func (w *world) belt(ids ...uint32) {
	w.mu.Lock()
	defer w.mu.Unlock()
	var keep []d2.Unit
	for _, u := range w.items {
		if u.Item.Where != d2.ContBelt {
			keep = append(keep, u)
		}
	}
	for _, id := range ids {
		keep = append(keep, d2.Unit{ID: id, Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
	}
	w.items = keep
}

// addMonster places a monster at an offset from the player.
func (w *world) addMonster(id uint32, class, dx, dy int) {
	w.mu.Lock()
	w.mons = append(w.mons, d2.Unit{ID: id, Type: d2.TypeMonster, Class: class, X: w.x + dx, Y: w.y + dy, Mode: d2.MonModeStand})
	w.mu.Unlock()
}

// ---- fake catalog ----

type fakeCatalog struct{}

const goldClass = 523 // the gold pile item class seen live (docs/combat.md)

var graph = map[int][]int{
	1: {2}, 2: {1, 3, 8}, 3: {2, 4, 17, 9}, 4: {3, 5}, 5: {4, 6}, 6: {5}, 8: {2}, 9: {3}, 17: {3},
}

func (fakeCatalog) Item(class int) (ItemMeta, bool) {
	switch class {
	case 1:
		return ItemMeta{Name: "Healing Potion", GoalType: "potion", W: 1, H: 1, Potion: PotionHealing}, true
	case 2:
		return ItemMeta{Name: "El Rune", GoalType: "rune", W: 1, H: 1}, true
	case 3:
		return ItemMeta{Name: "Plate Mail", GoalType: "armor", W: 2, H: 3}, true
	case 5:
		return ItemMeta{Name: "Scroll of Town Portal", GoalType: "scroll", W: 1, H: 1}, true
	case goldClass:
		return ItemMeta{Name: "Gold", GoalType: "gold", W: 1, H: 1}, true
	case 4:
		return ItemMeta{Name: "Full Plate", GoalType: "armor", W: 2, H: 3, Cost: 3000}, true
	}
	return ItemMeta{}, false
}
func (fakeCatalog) Monster(class int) (MonsterMeta, bool) {
	switch class {
	case 10:
		return MonsterMeta{Name: "Fallen", Killable: true}, true
	case 11:
		return MonsterMeta{Name: "Akara", NPC: true}, true
	case 12:
		return MonsterMeta{Name: "Andariel", Killable: true, Boss: true}, true
	}
	return MonsterMeta{}, false
}
func (fakeCatalog) AreaName(id int) string {
	return map[int]string{1: "Rogue Encampment", 2: "Blood Moor", 3: "Cold Plains", 4: "Stony Field", 5: "Dark Wood", 6: "Black Marsh", 8: "Den of Evil", 9: "Cave Level 1", 17: "Burial Grounds"}[id]
}
func (c fakeCatalog) AreaByName(n string) (int, bool) {
	for id := 1; id < 40; id++ {
		if strings.EqualFold(c.AreaName(id), n) {
			return id, true
		}
	}
	return 0, false
}
func (fakeCatalog) ExpToReach(level int) int64 { return int64(level-1) * 1000 }

// ---- fake behaviors ----

type fakeMover struct {
	exitDelay time.Duration
	w         *world
	wpErr     error
	exitBlock bool // ExitTo never arrives
	exploreFn func(ctx context.Context, done func() bool) error
	routeFn   func(from, to int) ([]int, error) // overrides the graph search
}

// Route is a breadth-first search over the fake level graph (undirected), like nav.Route without costs.
func (m *fakeMover) Route(from, to int) ([]int, error) {
	if m.routeFn != nil {
		return m.routeFn(from, to)
	}
	if from == to {
		return nil, nil
	}
	rev := map[int][]int{}
	for a, ns := range graph {
		for _, b := range ns {
			rev[a], rev[b] = append(rev[a], b), append(rev[b], a)
		}
	}
	prev := map[int]int{from: from}
	queue := []int{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			var path []int
			for n := to; n != from; n = prev[n] {
				path = append([]int{n}, path...)
			}
			return path, nil
		}
		for _, n := range rev[cur] {
			if _, seen := prev[n]; !seen {
				prev[n] = cur
				queue = append(queue, n)
			}
		}
	}
	return nil, errors.New("no route")
}

func (m *fakeMover) GoTo(ctx context.Context, p input.Point) error {
	m.w.call("goto %d,%d", p.X, p.Y)
	return nil
}
func (m *fakeMover) GoToUnit(ctx context.Context, id uint32) error {
	m.w.call("gotounit %d", id)
	return nil
}
func (m *fakeMover) ExitTo(ctx context.Context, l int) error {
	m.w.call("exit %d", l)
	if m.exitBlock {
		<-ctx.Done()
		return ctx.Err()
	}
	if m.exitDelay > 0 {
		select {
		case <-time.After(m.exitDelay):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	m.w.move(l)
	return nil
}
func (m *fakeMover) TakeWaypoint(ctx context.Context, l int) error {
	m.w.call("wp %d", l)
	if m.wpErr != nil {
		return m.wpErr
	}
	m.w.move(l)
	return nil
}
func (m *fakeMover) Explore(ctx context.Context, done func() bool) error {
	m.w.call("explore")
	if m.exploreFn != nil {
		return m.exploreFn(ctx, done)
	}
	return nil
}
func (m *fakeMover) PathTo(p input.Point) ([]input.Point, error) { return []input.Point{p}, nil }

type fakeFighter struct {
	w            *world
	safety       func(s *d2.Snapshot) SafetyAction
	actErr       error
	clearErr     error
	clearNothing bool
	onAct        func(a SafetyAction)
}

func (f *fakeFighter) Clear(ctx context.Context, o ClearOpts) (ClearResult, error) {
	f.w.call("clear r=%d", o.Radius)
	if err := f.clearErr; err != nil {
		f.clearErr = nil
		f.w.set(func(w *world) { w.area = 1 }) // the portal was taken
		return ClearResult{}, err
	}
	if f.clearNothing { // unreachable monsters: the fighter kills nothing
		return ClearResult{Remaining: 1}, nil
	}
	f.w.mu.Lock()
	n := 0
	for i := range f.w.mons {
		if f.w.mons[i].Alive() && f.w.mons[i].Class != 11 && !(f.w.mons[i].Class == 12 && f.w.bossImmune) { // NPCs (and immune bosses) survive
			f.w.mons[i].Mode = d2.MonModeDeath
			n++
		}
	}
	f.w.exp += uint32(10 * n)
	f.w.mu.Unlock()
	time.Sleep(2 * time.Millisecond) // let a poll see the corpses
	return ClearResult{Killed: n}, nil
}
func (f *fakeFighter) Kill(ctx context.Context, id uint32) error { f.w.call("kill %d", id); return nil }
func (f *fakeFighter) Buff(ctx context.Context) error            { f.w.call("buff"); return nil }
func (f *fakeFighter) Safety(s *d2.Snapshot) SafetyAction {
	if f.safety != nil {
		return f.safety(s)
	}
	return SafetyNone
}
func (f *fakeFighter) Act(ctx context.Context, a SafetyAction) error {
	f.w.call("act %s", a)
	if f.onAct != nil {
		f.onAct(a)
	}
	return f.actErr
}

type fakeTown struct {
	backErr     func(area int) // runs inside BackToField while armed
	backReturn  error          // returned instead of errBoom when set
	armed       atomic.Bool
	w           *world
	choresErr   error
	toTownBlock bool  // ToTown never returns
	toTownErr   error // ToTown fails (no portal, no route)
	needs       bool  // NeedsTrip answer
	needsWhy    string
}

func (t *fakeTown) ToTown(ctx context.Context) error {
	t.w.call("totown")
	if t.toTownBlock {
		<-ctx.Done()
		return ctx.Err()
	}
	if t.toTownErr != nil {
		return t.toTownErr
	}
	t.w.set(func(w *world) { w.area = 1 })
	return nil
}
func (t *fakeTown) Chores(ctx context.Context, p ChorePlan) (ChoreReport, error) {
	t.w.call("chores %s", p.Reason)
	if t.choresErr != nil {
		return ChoreReport{}, t.choresErr
	}
	t.w.belt(101, 102, 103, 104)
	t.w.set(func(w *world) { w.life = w.maxL })
	return ChoreReport{Done: []string{"potions"}}, nil
}
func (t *fakeTown) BackToField(ctx context.Context, l int) error {
	t.w.call("back %d", l)
	if t.backErr != nil && t.armed.CompareAndSwap(true, false) {
		t.backErr(l)
		if t.backReturn != nil {
			return t.backReturn
		}
		return errBoom
	}
	t.w.move(l)
	return nil
}

type fakeLoot struct {
	w   *world
	err error
}

func (l fakeLoot) PickUp(ctx context.Context, o PickOpts) (PickReport, error) {
	l.w.call("pickup")
	if l.err != nil {
		return PickReport{}, l.err
	}
	return PickReport{}, nil
}
func (l fakeLoot) Evaluate(it d2.Item) Verdict { return VerdictKeep }

type fakeHealth struct {
	mu sync.Mutex
	ev launch.Event
}

func (h *fakeHealth) Step(time.Time) launch.Event {
	h.mu.Lock()
	defer h.mu.Unlock()
	e := h.ev
	h.ev = launch.EventNone
	return e
}

// ---- rig ----

type rig struct {
	t      *testing.T
	w      *world
	e      *env.Env
	r      *Runner
	mover  *fakeMover
	fight  *fakeFighter
	town   *fakeTown
	health *fakeHealth
	in     *input.Fake
	clock  *vclock
	cancel context.CancelFunc
	done   chan error
	evs    []goals.Event
	evmu   sync.Mutex
}

func newRig(t *testing.T, mutate ...func(*Config, *Deps, *env.Tuning)) *rig {
	t.Helper()
	w := newWorld()
	clk := newClock()
	tune := env.DefaultTuning()
	tune.Campaign.DenMinLevelCaster, tune.Campaign.DenMinLevelMelee = 1, 1 // the Den flow tests start a clvl 2 Sorceress; the gate has its own tests
	tune.Resource.EarlyGearMax = 0                                         // the gate tests pin the full rules; early-gear tests set it back
	in := input.NewFake(input.Point{})
	e := &env.Env{ID: "t1", Log: slog.New(slog.NewTextHandler(io.Discard, nil)), In: nil, Tune: &tune, Clock: clk}
	e.In = &spyIn{Fake: in, w: w}
	rg := &rig{t: t, w: w, e: e, in: in, clock: clk, mover: &fakeMover{w: w}, fight: &fakeFighter{w: w}, town: &fakeTown{w: w}, health: &fakeHealth{}}
	cfg := Config{StuckAfterSec: 20000, SweepTimeoutSec: 100000, TravelTimeoutSec: 100000, TownTimeoutSec: 100000, RecoverTimeoutSec: 100000, ErrorRetrySec: 100000, MaxStepFailures: 3, GlobalStuckSec: 1000000, StateStuckSec: map[string]int{}}
	d := Deps{Mover: rg.mover, Fighter: rg.fight, Town: rg.town, Loot: fakeLoot{w: w}, Catalog: fakeCatalog{}, Health: rg.health, Snapshot: w.snapshot}
	for _, m := range mutate {
		m(&cfg, &d, &tune)
	}
	r, err := New(e, d, cfg)
	if err != nil {
		t.Fatal(err)
	}
	rg.r = r
	w.belt(101, 102, 103, 104)
	return rg
}

// run starts the Run goroutine and collects events; stop it with finish.
func (g *rig) run() {
	ctx, cancel := context.WithCancel(context.Background())
	g.cancel = cancel
	g.done = make(chan error, 1)
	go func() { g.done <- g.r.Run(ctx) }()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-g.r.Events():
				g.evmu.Lock()
				g.evs = append(g.evs, ev)
				g.evmu.Unlock()
			}
		}
	}()
}

func (g *rig) finish() error {
	g.cancel()
	select {
	case err := <-g.done:
		return err
	case <-time.After(20 * time.Second):
		g.t.Fatal("Run did not return")
	}
	return nil
}

func (g *rig) events(typ goals.EventType) []goals.Event {
	g.evmu.Lock()
	defer g.evmu.Unlock()
	var out []goals.Event
	for _, e := range g.evs {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

func (g *rig) waitFor(what string, cond func() bool) {
	g.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	g.t.Fatalf("timeout waiting for %s\nstate=%s status=%+v\ncalls=%s", what, g.r.State(), g.r.Status(), g.w.log())
}

func farm(area string) goals.Goal {
	return goals.Goal{ID: "g1", Type: goals.FarmArea, Params: goals.Params{Area: area}}
}

var errBoom = errors.New("boom")

// spyIn records the controller calls the bot makes itself into the world log (input.Fake's own log is not synchronized for readers).
type spyIn struct {
	*input.Fake
	w *world
}

func (s *spyIn) Key(k string) error { s.w.call("key %s", k); return s.Fake.Key(k) }
func (s *spyIn) CloseAll() error    { s.w.call("closeall"); return s.Fake.CloseAll() }
func (s *spyIn) ClickWorld(p input.Point) error {
	s.w.call("clickworld %v", p)
	return s.Fake.ClickWorld(p)
}

func (t *fakeTown) Rebuy(ctx context.Context, budget int) (RebuyReport, error) {
	t.w.call("rebuy %d", budget)
	return RebuyReport{}, nil
}
func (t *fakeTown) NeedsTrip(*d2.Snapshot) (bool, string) { return t.needs, t.needsWhy }
