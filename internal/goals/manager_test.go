package goals

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

type fakeRunner struct {
	mu      sync.Mutex
	started []Goal
	state   RunnerState
	level   int
	failFor map[Type]bool
	stops   int
	ch      chan Event
	cfg     *Settings
}

func newFake() *fakeRunner {
	return &fakeRunner{state: RunnerIdle, level: 1, ch: make(chan Event, 16)}
}

func (f *fakeRunner) Start(g Goal) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failFor[g.Type] {
		return errors.New("boom")
	}
	f.started = append(f.started, g)
	f.state = RunnerRunning
	return nil
}
func (f *fakeRunner) Pause() error  { f.mu.Lock(); f.state = RunnerPaused; f.mu.Unlock(); return nil }
func (f *fakeRunner) Resume() error { f.mu.Lock(); f.state = RunnerRunning; f.mu.Unlock(); return nil }
func (f *fakeRunner) Stop() error {
	f.mu.Lock()
	f.state = RunnerIdle
	f.stops++
	f.mu.Unlock()
	return nil
}
func (f *fakeRunner) Status() Status {
	f.mu.Lock()
	defer f.mu.Unlock()
	return Status{State: f.state, Character: Character{Level: f.level}}
}
func (f *fakeRunner) Events() <-chan Event { return f.ch }
func (f *fakeRunner) Configure(s Settings) { f.cfg = &s }
func (f *fakeRunner) lastStarted() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.started) == 0 {
		return ""
	}
	return f.started[len(f.started)-1].Title
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time          { return c.t }
func (c *clock) advance(d time.Duration) { c.t = c.t.Add(d) }

func newMgr(t *testing.T, path string) (*Manager, *fakeRunner, *clock) {
	t.Helper()
	f := newFake()
	c := &clock{t: time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)}
	m, err := New(Options{Runner: f, Path: path, Clock: c.now})
	if err != nil {
		t.Fatal(err)
	}
	return m, f, c
}

func mustAdd(t *testing.T, m *Manager, g Goal) Goal {
	t.Helper()
	out, err := m.Add(g, -1)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func titles(m *Manager) string {
	var s []string
	for _, g := range m.Goals() {
		s = append(s, fmt.Sprintf("%s:%s", g.Title, g.State))
	}
	return strings.Join(s, ", ")
}

func lvl(n int) Goal { return Goal{Type: ReachLevel, Params: Params{Level: n}} }
func boss(n string, c int) Goal {
	return Goal{Type: KillBoss, Params: Params{Boss: n, Count: c}}
}

func TestSchedulerOrderAndChaining(t *testing.T) {
	m, f, c := newMgr(t, "")
	mustAdd(t, m, lvl(10))
	mustAdd(t, m, boss("Mephisto", 2))
	hi := lvl(5)
	hi.Priority = 1
	mustAdd(t, m, hi)
	if got := titles(m); got != "Reach level 5:queued, Reach level 10:queued, Kill Mephisto x2:queued" {
		t.Fatalf("order: %s", got)
	}
	if f.lastStarted() != "" {
		t.Fatal("nothing should run while stopped")
	}
	if err := m.Start(); err != nil {
		t.Fatal(err)
	}
	if f.lastStarted() != "Reach level 5" {
		t.Fatalf("started %q", f.lastStarted())
	}
	f.level = 5
	c.advance(time.Second)
	m.Tick(c.now())
	if f.lastStarted() != "Reach level 10" {
		t.Fatalf("after completion started %q (%s)", f.lastStarted(), titles(m))
	}
	f.level = 10
	c.advance(time.Second)
	m.Tick(c.now())
	if f.lastStarted() != "Kill Mephisto x2" {
		t.Fatalf("got %q", f.lastStarted())
	}
	m.Handle(Event{Type: EvKill, Monster: "Mephisto", Boss: true})
	m.Handle(Event{Type: EvKill, Monster: "Andariel", Boss: true}) // other boss: no credit
	g, _ := m.Goal("g2")
	if g.Progress.BossKills != 1 || g.Progress.Percent != 50 {
		t.Fatalf("progress %+v", g.Progress)
	}
	m.Handle(Event{Type: EvKill, Monster: "mephisto"})
	if g, _ = m.Goal("g2"); g.State != StateDone {
		t.Fatalf("state %s", g.State)
	}
	if snap := m.Snapshot(); snap.ActiveGoalID != "" {
		t.Fatalf("active %q after queue drained", snap.ActiveGoalID)
	}
	// finished goals sort after live ones
	mustAdd(t, m, lvl(50))
	gs := m.Goals()
	if gs[0].State != StateActive {
		t.Fatalf("new goal should run: %s", titles(m))
	}
}

func TestFarmCompletionConditions(t *testing.T) {
	tests := []struct {
		name  string
		p     Params
		drive func(m *Manager, c *clock)
	}{
		{"kills", Params{Area: "A", UntilKills: 3}, func(m *Manager, c *clock) {
			for i := 0; i < 3; i++ {
				m.Handle(Event{Type: EvKill, Monster: "x"})
			}
		}},
		{"minutes", Params{Area: "A", UntilMins: 1}, func(m *Manager, c *clock) {
			for i := 0; i < 61; i++ {
				c.advance(time.Second)
				m.Tick(c.now())
			}
		}},
		{"matching drops", Params{Area: "A", UntilDrops: 2, Item: ItemFilter{Type: "rune"}}, func(m *Manager, c *clock) {
			m.Handle(Event{Type: EvDrop, Item: &Item{Name: "Tir Rune", Type: "rune"}})
			m.Handle(Event{Type: EvDrop, Item: &Item{Name: "Cap", Type: "helm"}})
			m.Handle(Event{Type: EvDrop, Item: &Item{Name: "Eld Rune", Type: "rune"}})
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			m, _, c := newMgr(t, "")
			mustAdd(t, m, Goal{Type: FarmArea, Params: tc.p})
			m.Start()
			tc.drive(m, c)
			if g, _ := m.Goal("g1"); g.State != StateDone || g.Progress.Percent != 100 {
				t.Fatalf("goal %s %+v", g.State, g.Progress)
			}
		})
	}
	t.Run("endless never completes", func(t *testing.T) {
		m, _, c := newMgr(t, "")
		mustAdd(t, m, Goal{Type: FarmArea, Params: Params{Area: "A"}})
		m.Start()
		for i := 0; i < 200; i++ {
			m.Handle(Event{Type: EvKill, Monster: "x"})
			c.advance(time.Minute)
			m.Tick(c.now())
		}
		if g, _ := m.Goal("g1"); g.State != StateActive || !g.Endless {
			t.Fatalf("%s endless=%v", g.State, g.Endless)
		}
	})
}

func TestCollectItem(t *testing.T) {
	m, _, _ := newMgr(t, "")
	mustAdd(t, m, Goal{Type: CollectItem, Params: Params{Count: 3, Item: ItemFilter{Type: "rune"}}})
	m.Start()
	m.Handle(Event{Type: EvDrop, Item: &Item{Name: "Tir Rune", Type: "rune", Qty: 2}})
	m.Handle(Event{Type: EvDrop, Item: &Item{Name: "Gem", Type: "gem"}})
	if g, _ := m.Goal("g1"); g.Progress.Current != 2 || g.State != StateActive {
		t.Fatalf("%+v", g.Progress)
	}
	m.Handle(Event{Type: EvDrop, Item: &Item{Name: "El Rune", Type: "rune"}})
	if g, _ := m.Goal("g1"); g.State != StateDone {
		t.Fatalf("%s", g.State)
	}
}

func TestIdleGoalTimedAndWindowed(t *testing.T) {
	m, _, c := newMgr(t, "")
	mustAdd(t, m, Goal{Type: Idle, Params: Params{Minutes: 1}})
	m.Start()
	for i := 0; i < 61; i++ {
		c.advance(time.Second)
		m.Tick(c.now())
	}
	if g, _ := m.Goal("g1"); g.State != StateDone {
		t.Fatalf("%s", g.State)
	}
}

func TestScheduleWindowPreemption(t *testing.T) {
	m, f, c := newMgr(t, "")
	c.t = time.Date(2026, 10, 28, 1, 59, 0, 0, time.UTC)
	rest := Goal{Type: Idle, Priority: 1, Schedule: []Window{{Start: "02:00", End: "06:00"}}}
	mustAdd(t, m, Goal{Type: FarmArea, Params: Params{Area: "Pit"}})
	mustAdd(t, m, rest)
	m.Start()
	if f.lastStarted() != "Farm Pit" {
		t.Fatalf("started %q", f.lastStarted())
	}
	c.t = time.Date(2026, 10, 28, 2, 0, 0, 0, time.UTC)
	m.Tick(c.now())
	if f.lastStarted() != "Rest during scheduled windows" {
		t.Fatalf("window did not preempt: %s", titles(m))
	}
	if g, _ := m.Goal("g1"); g.State != StateQueued {
		t.Fatalf("preempted goal should be queued, is %s", g.State)
	}
	c.t = time.Date(2026, 10, 28, 6, 0, 0, 0, time.UTC)
	m.Tick(c.now())
	if f.lastStarted() != "Farm Pit" || len(f.started) != 3 {
		t.Fatalf("farm should resume after the window: %v", f.started)
	}
	// the endless windowed goal stays queued for tomorrow, not finished
	if g, _ := m.Goal("g2"); g.State != StateQueued {
		t.Fatalf("%s", g.State)
	}
}

func TestPauseSkipResumeReset(t *testing.T) {
	m, f, _ := newMgr(t, "")
	mustAdd(t, m, lvl(10))
	mustAdd(t, m, lvl(20))
	m.Start()
	if err := m.PauseGoal("g1"); err != nil {
		t.Fatal(err)
	}
	if f.lastStarted() != "Reach level 20" {
		t.Fatalf("held goal must be skipped: %s", titles(m))
	}
	if err := m.PauseGoal("g1"); !errors.Is(err, ErrState) {
		t.Fatalf("double pause: %v", err)
	}
	if err := m.ResumeGoal("g1"); err != nil {
		t.Fatal(err)
	}
	// g1 is first in order again, so it preempts
	if f.lastStarted() != "Reach level 10" {
		t.Fatalf("%s", titles(m))
	}
	if err := m.Skip("g1"); err != nil {
		t.Fatal(err)
	}
	if g, _ := m.Goal("g1"); g.State != StateSkipped || f.lastStarted() != "Reach level 20" {
		t.Fatalf("%s / %s", g.State, titles(m))
	}
	if err := m.Reset("g1"); err != nil {
		t.Fatal(err)
	}
	if g, _ := m.Goal("g1"); g.State != StateActive {
		t.Fatalf("reset goal should run first: %s", titles(m))
	}
	if err := m.Skip("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatal(err)
	}
}

func TestControlModes(t *testing.T) {
	m, f, c := newMgr(t, "")
	mustAdd(t, m, Goal{Type: FarmArea, Params: Params{Area: "A"}})
	if err := m.Pause(); !errors.Is(err, ErrState) {
		t.Fatalf("pause while stopped: %v", err)
	}
	m.Start()
	m.Pause()
	if m.Snapshot().Mode != ModePaused || f.state != RunnerPaused {
		t.Fatal("not paused")
	}
	// no time credit and no scheduling while paused
	c.advance(10 * time.Second)
	m.Tick(c.now())
	if g, _ := m.Goal("g1"); g.Progress.ActiveSeconds != 0 {
		t.Fatalf("credited %d s while paused", g.Progress.ActiveSeconds)
	}
	m.Start()
	if f.state != RunnerRunning {
		t.Fatal("resume did not reach the runner")
	}
	c.advance(10 * time.Second)
	m.Tick(c.now())
	if g, _ := m.Goal("g1"); g.Progress.ActiveSeconds != 10 {
		t.Fatalf("credited %d", g.Progress.ActiveSeconds)
	}
	m.Stop()
	if g, _ := m.Goal("g1"); g.State != StateQueued || m.Snapshot().Mode != ModeStopped || f.state != RunnerIdle {
		t.Fatalf("stop: %s", titles(m))
	}
	// a long gap (suspend / clock jump) is not credited
	m.Start()
	c.advance(2 * time.Hour)
	m.Tick(c.now())
	if g, _ := m.Goal("g1"); g.Progress.ActiveSeconds != 10 {
		t.Fatalf("gap credited: %d", g.Progress.ActiveSeconds)
	}
}

func TestStartFailureAndFatalError(t *testing.T) {
	m, f, _ := newMgr(t, "")
	f.failFor = map[Type]bool{KillBoss: true}
	mustAdd(t, m, boss("Baal", 1))
	mustAdd(t, m, lvl(30))
	m.Start()
	g1, _ := m.Goal("g1")
	if g1.State != StateFailed || !strings.Contains(g1.Note, "boom") {
		t.Fatalf("%+v", g1)
	}
	if f.lastStarted() != "Reach level 30" {
		t.Fatal("should fall through to the next goal")
	}
	m.Handle(Event{Type: EvError, Message: "game window lost", Fatal: true})
	s := m.Snapshot()
	if s.Mode != ModeStopped || s.LastError != "game window lost" {
		t.Fatalf("%+v", s)
	}
	if g, _ := m.Goal("g2"); g.State != StateQueued {
		t.Fatalf("active goal should return to the queue, is %s", g.State)
	}
}

func TestMaxDeaths(t *testing.T) {
	m, _, _ := newMgr(t, "")
	s := m.Settings()
	s.MaxDeaths = 2
	if err := m.SetSettings(s); err != nil {
		t.Fatal(err)
	}
	mustAdd(t, m, Goal{Type: FarmArea, Params: Params{Area: "A"}})
	m.Start()
	m.Handle(Event{Type: EvDeath})
	if g, _ := m.Goal("g1"); g.State != StateActive {
		t.Fatal("one death is fine")
	}
	m.Handle(Event{Type: EvDeath})
	if g, _ := m.Goal("g1"); g.State != StatePaused || !strings.Contains(g.Note, "2 deaths") {
		t.Fatalf("%s %q", g.State, g.Note)
	}
}

func TestMoveAndReorderPriorityClamp(t *testing.T) {
	m, _, _ := newMgr(t, "")
	a := mustAdd(t, m, lvl(1+1))
	b := mustAdd(t, m, lvl(3))
	hi := lvl(4)
	hi.Priority = 1
	h := mustAdd(t, m, hi)
	// order: h, a, b. Move b above h: b adopts h's priority.
	if err := m.Move(b.ID, 0); err != nil {
		t.Fatal(err)
	}
	gs := m.Goals()
	if gs[0].ID != b.ID || gs[0].Priority != 1 || gs[1].ID != h.ID {
		t.Fatalf("%v", titles(m))
	}
	// Move h to the bottom: it adopts the lowest neighbour priority so the order holds.
	m.Move(h.ID, 99)
	gs = m.Goals()
	if gs[2].ID != h.ID || gs[2].Priority != 0 {
		t.Fatalf("%+v", gs[2])
	}
	if err := m.Reorder([]string{a.ID, b.ID, h.ID}); err != nil {
		t.Fatal(err)
	}
	if got := m.Goals(); got[0].ID != a.ID || got[0].Priority < got[1].Priority {
		t.Fatalf("reorder: %s", titles(m))
	}
	if err := m.Reorder([]string{a.ID}); !errors.Is(err, ErrInvalid) {
		t.Fatal("partial reorder must be rejected")
	}
	if err := m.Reorder([]string{a.ID, a.ID, b.ID}); !errors.Is(err, ErrInvalid) {
		t.Fatal("duplicate must be rejected")
	}
}

func TestUpdateRestartsActiveGoal(t *testing.T) {
	m, f, _ := newMgr(t, "")
	mustAdd(t, m, lvl(10))
	m.Start()
	n := len(f.started)
	g, err := m.Update("g1", func(g *Goal) { g.Params.Level = 20; g.Title = "" })
	if err != nil {
		t.Fatal(err)
	}
	if g.Title != "Reach level 20" || len(f.started) != n+1 || g.State != StateActive {
		t.Fatalf("%+v starts=%d", g, len(f.started))
	}
	if _, err := m.Update("g1", func(g *Goal) { g.Type = Idle }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("type change: %v", err)
	}
	if _, err := m.Update("g1", func(g *Goal) { g.Params.Level = 0 }); !errors.Is(err, ErrInvalid) {
		t.Fatalf("invalid edit: %v", err)
	}
}

func TestEventsAndSubscribe(t *testing.T) {
	m, _, _ := newMgr(t, "")
	ch, cancel := m.Subscribe()
	mustAdd(t, m, lvl(5))
	m.Start()
	m.Handle(Event{Type: EvKill, Monster: "Fallen", XP: 7})
	var types []EventType
	for len(ch) > 0 {
		types = append(types, (<-ch).Type)
	}
	want := []EventType{EvGoalAdded, EvControl, EvGoalStarted, EvKill}
	if fmt.Sprint(types) != fmt.Sprint(want) {
		t.Fatalf("got %v", types)
	}
	evs := m.Events(0, 0)
	if len(evs) != 4 || evs[3].Message != "Killed Fallen (+7 xp)" || evs[3].GoalID != "g1" {
		t.Fatalf("%+v", evs[len(evs)-1])
	}
	if got := m.Events(evs[1].Seq, 0); len(got) != 2 {
		t.Fatalf("since filter: %d", len(got))
	}
	if got := m.Events(0, 2); len(got) != 2 || got[1].Seq != evs[3].Seq {
		t.Fatal("limit keeps newest")
	}
	cancel()
	cancel() // idempotent
}

func TestEventLogIsBounded(t *testing.T) {
	f := newFake()
	m, _ := New(Options{Runner: f, EventLogSize: 50})
	for i := 0; i < 5000; i++ {
		m.Handle(Event{Type: EvKill, Monster: "x"})
	}
	if n := len(m.Events(0, 0)); n != 50 {
		t.Fatalf("log holds %d", n)
	}
	if m.StatsSummary().Totals.Kills != 5000 {
		t.Fatal("totals must not be bounded")
	}
}

func TestConfigureAndAutoResume(t *testing.T) {
	f := newFake()
	m, _ := New(Options{Runner: f})
	if f.cfg == nil || f.cfg.ChickenLifePct != 30 {
		t.Fatalf("runner not configured: %+v", f.cfg)
	}
	bad := m.Settings()
	bad.ChickenLifePct, bad.PotionLifePct = 60, 50
	if err := m.SetSettings(bad); !errors.Is(err, ErrInvalid) {
		t.Fatal(err)
	}

	path := filepath.Join(t.TempDir(), "s.json")
	m1, f1, _ := newMgr(t, path)
	s := m1.Settings()
	s.AutoResume = true
	m1.SetSettings(s)
	mustAdd(t, m1, lvl(9))
	m2, f2, _ := newMgr(t, path)
	if m2.Snapshot().Mode != ModeRunning || f2.lastStarted() != "Reach level 9" || f1 == f2 {
		t.Fatalf("autoresume: %v %q", m2.Snapshot().Mode, f2.lastStarted())
	}
}

func TestFinishedGoalsPruned(t *testing.T) {
	m, _, _ := newMgr(t, "")
	m.Start()
	for i := 0; i < maxFinished+20; i++ {
		g := mustAdd(t, m, Goal{Type: KillBoss, Params: Params{Boss: "X", Count: 1}})
		m.Handle(Event{Type: EvKill, Monster: "X"})
		if gg, _ := m.Goal(g.ID); gg.State != StateDone {
			t.Fatal("not done")
		}
	}
	if n := len(m.Goals()); n != maxFinished {
		t.Fatalf("kept %d finished goals", n)
	}
}

func TestHoldErrorPausesUntilLevelUp(t *testing.T) {
	m, f, _ := newMgr(t, "")
	a := mustAdd(t, m, boss("Mephisto", 1))
	m.Start()
	m.Handle(Event{Type: EvError, Hold: true, Level: 20, Message: "run locked: needs clvl 24"})
	if got := titles(m); got != "Kill Mephisto:paused" {
		t.Fatalf("after hold: %s", got)
	}
	m.Handle(Event{Type: EvLevelUp, Level: 20})
	if titles(m) != "Kill Mephisto:paused" {
		t.Fatalf("same level resumed: %s", titles(m))
	}
	m.Handle(Event{Type: EvLevelUp, Level: 21})
	if g := m.find(a.ID); g.State == StatePaused {
		t.Fatalf("not resumed: %s", titles(m))
	}
	_ = f
}

func holdForQuest(t *testing.T, m *Manager, diff int) Goal {
	t.Helper()
	a := mustAdd(t, m, boss("Mephisto", 1))
	m.Start()
	m.Handle(Event{Type: EvError, Hold: true, Level: 30, Unlock: "anya_portal", Diff: diff, Message: "run locked: needs anya_portal"})
	if m.find(a.ID).State != StatePaused {
		t.Fatalf("not held: %s", titles(m))
	}
	return a
}

func TestQuestDoneResumesHeldGoal(t *testing.T) {
	m, _, _ := newMgr(t, "")
	a := holdForQuest(t, m, 1)
	m.Handle(Event{Type: EvQuest, Quest: "prison_of_ice", Stage: 2, Diff: 1, Unlock: "anya_portal"}) // not done
	m.Handle(Event{Type: EvQuest, Quest: "prison_of_ice", Stage: 3, Done: true, Diff: 0, Unlock: "anya_portal"})
	if m.find(a.ID).State != StatePaused {
		t.Fatal("resumed by another difficulty or an unfinished quest")
	}
	m.Handle(Event{Type: EvQuest, Quest: "prison_of_ice", Stage: 3, Done: true, Diff: 1, Unlock: "anya_portal"})
	if m.find(a.ID).State == StatePaused {
		t.Fatalf("not resumed: %s", titles(m))
	}
}

func TestQuestEventIdempotentAndPersisted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	m, _, _ := newMgr(t, path)
	a := holdForQuest(t, m, 0)
	ev := Event{Type: EvQuest, Quest: "den", Stage: 3, Done: true}
	m.Handle(ev)
	m.Handle(ev)
	n := 0
	for _, e := range m.Events(0, 1000) {
		if e.Type == EvQuest {
			n++
		}
	}
	if n != 1 {
		t.Fatalf("repeat logged %d times", n)
	}
	m.mu.Lock()
	m.saveLocked(m.opt.Clock())
	m.mu.Unlock()

	// restart: the lock and the seen stage survive, and a replayed done event still releases the goal
	m2, _, _ := newMgr(t, path)
	if m2.questSeen["0/den"] != 3 {
		t.Fatalf("questSeen lost: %v", m2.questSeen)
	}
	if _, ok := m2.locked[a.ID]; !ok {
		t.Fatal("lock lost on restart")
	}
	m2.Handle(Event{Type: EvQuest, Quest: "prison_of_ice", Stage: 3, Done: true, Unlock: "anya_portal"})
	if m2.find(a.ID).State == StatePaused {
		t.Fatalf("not resumed after restart: %s", titles(m2))
	}
}

func hasLock(m *Manager, id string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.locked[id]
	return ok
}

func TestUserPauseAfterHoldIsNotOverriddenByLevelUp(t *testing.T) {
	m, _, _ := newMgr(t, "")
	a := mustAdd(t, m, boss("Mephisto", 1))
	m.Start()
	m.Handle(Event{Type: EvError, Hold: true, Level: 30, Message: "locked"})
	m.Handle(Event{Type: EvLevelUp, Level: 31})
	if hasLock(m, a.ID) {
		t.Fatal("lock left after the level-up resumed the goal")
	}
	if err := m.PauseGoal(a.ID); err != nil {
		t.Fatal(err)
	}
	m.Handle(Event{Type: EvLevelUp, Level: 32})
	if m.find(a.ID).State != StatePaused {
		t.Fatalf("user pause overridden: %s", titles(m))
	}
}

func TestUserPauseReplacesHold(t *testing.T) {
	m, _, _ := newMgr(t, "")
	a := holdForQuest(t, m, 0)
	if err := m.ResumeGoal(a.ID); err != nil || hasLock(m, a.ID) {
		t.Fatalf("resume left a lock: %v", err)
	}
	if err := m.PauseGoal(a.ID); err != nil {
		t.Fatal(err)
	}
	m.Handle(Event{Type: EvQuest, Quest: "den", Stage: 3, Done: true, Unlock: "anya_portal"})
	if m.find(a.ID).State != StatePaused {
		t.Fatal("quest event resumed a user pause")
	}
}

func TestRemoveClearsHoldForReusedID(t *testing.T) {
	m, _, _ := newMgr(t, "")
	a := mustAdd(t, m, boss("Mephisto", 1))
	m.Start()
	m.Handle(Event{Type: EvError, Hold: true, Level: 30, Message: "locked"})
	if err := m.Remove(a.ID); err != nil {
		t.Fatal(err)
	}
	if hasLock(m, a.ID) {
		t.Fatal("Remove left a lock")
	}
	m.mu.Lock()
	g := boss("Mephisto", 1)
	g.ID, g.State, g.Note = a.ID, StatePaused, userPauseNote
	m.goals = append(m.goals, &g)
	m.mu.Unlock()
	m.Handle(Event{Type: EvLevelUp, Level: 40})
	if m.find(a.ID).State != StatePaused {
		t.Fatal("a stale lock resumed a re-added goal")
	}
}

func TestRestoreDropsStaleHolds(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	m, _, _ := newMgr(t, path)
	a := mustAdd(t, m, boss("Mephisto", 1))
	b := mustAdd(t, m, boss("Andariel", 1))
	c := mustAdd(t, m, boss("Duriel", 1))
	m.mu.Lock()
	m.find(a.ID).State, m.find(a.ID).Note = StatePaused, userPauseNote // user pause, hold left by an old build
	m.find(b.ID).State, m.find(b.ID).Note = StatePaused, "run locked"
	m.find(b.ID).HeldAt = 30 // NMSIM-only file: HeldAt without Locks
	m.lockAtLocked(m.find(a.ID), 30)
	m.lockAtLocked(m.find(c.ID), 30) // queued goal with a lock
	m.locked["g999"] = lock{Level: 5}
	m.saveLocked(m.opt.Clock())
	m.mu.Unlock()

	m2, _, _ := newMgr(t, path)
	if hasLock(m2, a.ID) || hasLock(m2, c.ID) || hasLock(m2, "g999") {
		t.Fatalf("stale locks survived: %v", m2.locked)
	}
	if !hasLock(m2, b.ID) {
		t.Fatal("HeldAt not migrated into a lock")
	}
	m2.Handle(Event{Type: EvLevelUp, Level: 31})
	if m2.find(a.ID).State != StatePaused {
		t.Fatal("user-paused goal auto-resumed")
	}
	if m2.find(b.ID).State == StatePaused {
		t.Fatal("migrated hold did not resume on level-up")
	}
}

func TestPrunedSeedStaysRefusedAfterReload(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	m, _, _ := newMgr(t, path)
	m.Start()
	if _, ok, err := m.AddSeeded(boss("Mephisto", 1), "seedA"); err != nil || !ok {
		t.Fatalf("first add: %v %v", ok, err)
	}
	m.Handle(Event{Type: EvKill, Monster: "Mephisto"})
	for i := 0; i < maxFinished+1; i++ { // push seedA out of the kept finished goals
		mustAdd(t, m, boss("X", 1))
		m.Handle(Event{Type: EvKill, Monster: "X"})
	}
	for _, g := range m.Goals() {
		if g.Seed == "seedA" {
			t.Fatal("seedA goal not pruned")
		}
	}
	m.Close()
	m2, _, _ := newMgr(t, path)
	if _, ok, _ := m2.AddSeeded(boss("Mephisto", 1), "seedA"); ok {
		t.Fatal("pruned seeded goal queued again after reload")
	}
	if _, ok, _ := m2.AddSeeded(boss("Mephisto", 1), "seedB"); !ok {
		t.Fatal("a changed --goals entry must still add")
	}
}

func TestPrunedSeedsBounded(t *testing.T) {
	s := make([]string, maxPrunedSeeds+30)
	for i := range s {
		s[i] = strconv.Itoa(i)
	}
	if got := trimSeeds(s); len(got) != maxPrunedSeeds || got[0] != "30" {
		t.Fatalf("trim: %d %q", len(got), got[0])
	}
}

func TestPruneDropsLocksOfPrunedGoals(t *testing.T) {
	m, _, _ := newMgr(t, "")
	m.mu.Lock()
	for i := 0; i < maxFinished+3; i++ {
		c, _ := m.addLocked(boss("X", 1), m.opt.Clock())
		g := m.find(c.ID)
		g.State = StateDone
		now := m.opt.Clock().Add(time.Duration(i) * time.Second)
		g.CompletedAt = &now
		m.lockAtLocked(g, 5)
	}
	m.pruneLocked()
	n := len(m.locked)
	m.mu.Unlock()
	if n != maxFinished {
		t.Fatalf("%d locks left, want %d", n, maxFinished)
	}
}

func TestRepeatQuestEventsDoNotSave(t *testing.T) {
	path := filepath.Join(t.TempDir(), "goals.json")
	m, _, c := newMgr(t, path)
	m.Start()
	ev := Event{Type: EvQuest, Quest: "den", Stage: 3, Done: true}
	m.Handle(ev)
	c.t = c.t.Add(saveEvery)
	m.Tick(c.now()) // the new questSeen entry is written by the tick
	st1, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	m.mu.Lock()
	saved := m.lastSave
	m.mu.Unlock()
	for i := 0; i < 27; i++ {
		c.t = c.t.Add(10 * time.Millisecond)
		m.Handle(ev)
	}
	m.mu.Lock()
	again, dirty := m.lastSave, m.dirty
	m.mu.Unlock()
	if !again.Equal(saved) || dirty {
		t.Fatalf("repeat events saved or dirtied the state (lastSave %v -> %v, dirty %v)", saved, again, dirty)
	}
	if st2, _ := os.Stat(path); !st2.ModTime().Equal(st1.ModTime()) {
		t.Fatal("file rewritten")
	}
}

func TestStaleHoldLevelResumesAtOnce(t *testing.T) {
	m, f, _ := newMgr(t, "")
	a := mustAdd(t, m, boss("Mephisto", 1))
	m.Start()
	f.mu.Lock()
	f.level = 21 // levelled up after the runner read 20
	f.mu.Unlock()
	m.Handle(Event{Type: EvError, Hold: true, Level: 20, Message: "run locked: needs clvl 24"})
	if m.find(a.ID).State == StatePaused || hasLock(m, a.ID) {
		t.Fatalf("stale hold not re-evaluated: %s", titles(m))
	}
	// a current-level hold still holds
	m.Handle(Event{Type: EvError, Hold: true, Level: 21, Message: "run locked: needs clvl 24"})
	if m.find(a.ID).State != StatePaused {
		t.Fatalf("current hold ignored: %s", titles(m))
	}
}

func TestStaleHoldLevelFromLevelUpEvent(t *testing.T) {
	m, _, _ := newMgr(t, "")
	a := mustAdd(t, m, boss("Mephisto", 1))
	m.Start()
	m.Handle(Event{Type: EvLevelUp, Level: 22})
	m.Handle(Event{Type: EvError, Hold: true, Level: 20, Message: "x"})
	if m.find(a.ID).State == StatePaused {
		t.Fatalf("not resumed: %s", titles(m))
	}
}
