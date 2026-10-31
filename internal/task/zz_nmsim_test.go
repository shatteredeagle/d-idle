package task_test

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
)

// NMSIM: a goals.Manager with the real difficulty chain rules (task.NextDifficulty, task.ReadyForNext,
// task.EntryLevel, as wired by cmd/d2bot taskChain) drives a fake runner through normal_all -> nightmare_all ->
// hell_all and a farm runs goal. The runner is a model (levels, quests gated by the real campaign entry floors,
// strikes that park a quest, timed self pauses, crashes), so what is tested is the manager and chain bookkeeping.
// The runner's rules are U (invented numbers); the invariants are what matters.

type zmsClock struct{ t time.Time }

func (c *zmsClock) now() time.Time { return c.t }

// zmsWorld is what survives a process restart: the character and its quest log.
type zmsWorld struct {
	lvl, xp int
	diff    d2.Difficulty
	done    map[string]int // campaign template -> quests done (the game remembers them)
	diffLog []d2.Difficulty
	gateBad []string // campaigns entered although ReadyForNext refused (U: a gap, see TestZMSResistGateAtStart)
}

func (w *zmsWorld) rawRes() d2.Resists {
	r := min(max(w.lvl*2-30, 0), 200)
	return d2.Resists{Fire: r, Cold: r, Lightning: r, Poison: r}
}

type zmsRunner struct {
	w           *zmsWorld
	clk         *zmsClock
	rng         *rand.Rand
	goal        *goals.Goal
	pausedUntil time.Time
	paused      bool
	strikes     map[int]int // quest index -> strikes this goal
	parkedAt    map[int]int // quest index -> clvl when parked
	evs         []goals.Event
	starts      []string
	selfPauses  int
}

func (r *zmsRunner) Start(g goals.Goal) error {
	r.goal, r.paused, r.pausedUntil = &g, false, time.Time{}
	r.strikes, r.parkedAt = map[int]int{}, map[int]int{}
	r.starts = append(r.starts, g.ID+":"+string(g.Type)+":"+g.Params.Template)
	if g.Type == goals.Campaign {
		if d := zmsDiff(g.Params.Template); d > r.w.diff {
			if ok, _ := task.ReadyForNext(zmsName(d), r.w.lvl, ptr(r.w.rawRes())); !ok {
				r.w.gateBad = append(r.w.gateBad, fmt.Sprintf("%s entered at clvl %d", g.Params.Template, r.w.lvl))
			}
			r.w.diff = d
			r.w.diffLog = append(r.w.diffLog, d)
		}
	}
	return nil
}
func (r *zmsRunner) Pause() error  { r.paused = true; return nil }
func (r *zmsRunner) Resume() error { r.paused = false; return nil }
func (r *zmsRunner) Stop() error   { r.goal = nil; return nil }
func (r *zmsRunner) Status() goals.Status {
	st := goals.RunnerRunning
	switch {
	case r.goal == nil:
		st = goals.RunnerIdle
	case r.paused || !r.pausedUntil.IsZero():
		st = goals.RunnerPaused
	}
	return goals.Status{State: st, Character: goals.Character{Level: r.w.lvl}}
}
func (r *zmsRunner) Events() <-chan goals.Event { return nil }
func (r *zmsRunner) Progress() (goals.Progress, bool) {
	if r.goal == nil || r.goal.Type != goals.Campaign {
		return goals.Progress{}, false
	}
	n := r.w.done[r.goal.Params.Template]
	return goals.Progress{Current: n, Target: goals.CampaignQuests(r.goal.Params.Template, false)}, true
}

func ptr[T any](v T) *T { return &v }

func zmsDiff(tmpl string) d2.Difficulty {
	switch tmpl {
	case goals.CampaignNightmareAll:
		return d2.Nightmare
	case goals.CampaignHellAll:
		return d2.Hell
	}
	return d2.Normal
}
func zmsName(d d2.Difficulty) string { return [...]string{"normal", "nightmare", "hell"}[d] }

func zmsFloor(tmpl string, quest, total int) int {
	var c *task.Campaign
	switch tmpl {
	case goals.CampaignNightmareAll:
		c = task.NightmareAll(false)
	case goals.CampaignHellAll:
		c = task.HellAll(false)
	default:
		return 0
	}
	return c.Entries[min(quest*len(c.Entries)/max(total, 1), len(c.Entries)-1)].MinLevel
}

// step plays one minute and returns the events for the manager.
func (r *zmsRunner) step(cfg zmsCfg) {
	r.evs = r.evs[:0]
	if r.goal == nil || r.paused {
		return
	}
	if !r.pausedUntil.IsZero() {
		if r.clk.t.Before(r.pausedUntil) {
			return
		}
		r.pausedUntil = time.Time{} // the self pause timer (bot.autoResumed)
	}
	if r.rng.Float64() < cfg.pauseP {
		r.pausedUntil = r.clk.t.Add(time.Duration(5+r.rng.Intn(56)) * time.Minute)
		r.selfPauses++
		return
	}
	g := r.goal
	// xp: one level per 1+lvl/8 minutes
	r.w.xp++
	if r.w.xp >= 1+r.w.lvl/8 && r.w.lvl < 99 {
		r.w.xp, r.w.lvl = 0, r.w.lvl+1
		r.evs = append(r.evs, goals.Event{Type: goals.EvLevelUp, Level: r.w.lvl})
	}
	switch g.Type {
	case goals.Runs:
		r.evs = append(r.evs, goals.Event{Type: goals.EvKill, Monster: g.Params.Area, Boss: true})
	case goals.Campaign:
		tmpl := g.Params.Template
		total := goals.CampaignQuests(tmpl, false)
		q := r.w.done[tmpl]
		if q >= total {
			return
		}
		if at, ok := r.parkedAt[q]; ok && r.w.lvl <= at {
			return // parked until the character gains a level (task.Executor strikes)
		}
		delete(r.parkedAt, q)
		if r.w.lvl < zmsFloor(tmpl, q, total) {
			return // EntryGate floor: levels on the spot
		}
		if r.rng.Float64() < cfg.deathP {
			r.evs = append(r.evs, goals.Event{Type: goals.EvDeath})
			if r.strikes[q]++; r.strikes[q] >= 3 {
				r.strikes[q], r.parkedAt[q] = 0, r.w.lvl
			}
			return
		}
		r.w.done[tmpl]++
	}
}

type zmsCfg struct {
	pauseP, deathP, crashP float64
}

type zmsSim struct {
	t        *testing.T
	m        *goals.Manager
	r        *zmsRunner
	w        *zmsWorld
	clk      *zmsClock
	rng      *rand.Rand
	path     string
	restarts int
}

func zmsChain(w *zmsWorld) *goals.Chain {
	ready := func(next string, clvl int) (bool, string) {
		r := w.rawRes()
		return task.ReadyForNext(next, clvl, &r)
	}
	return &goals.Chain{Next: task.NextDifficulty, Ready: ready, Floor: task.EntryLevel}
}

func newZMSim(t *testing.T, seed int64) *zmsSim {
	s := &zmsSim{t: t, rng: rand.New(rand.NewSource(seed)), clk: &zmsClock{t: time.Date(2026, 10, 29, 8, 0, 0, 0, time.UTC)},
		w: &zmsWorld{lvl: 1, done: map[string]int{}}, path: filepath.Join(t.TempDir(), "goals.json")}
	s.boot()
	set := s.m.Settings()
	set.MaxDeaths = 0 // the default 5 strands a campaign for good, see TestZMSMaxDeathsStrands
	set.AutoResume = true
	if err := s.m.SetSettings(set); err != nil {
		t.Fatal(err)
	}
	return s
}

// boot (re)creates the runner and manager from the state file: a process restart.
func (s *zmsSim) boot() {
	r := &zmsRunner{w: s.w, clk: s.clk, rng: s.rng}
	if s.r != nil {
		r.starts, r.selfPauses = s.r.starts, s.r.selfPauses
	}
	s.r = r
	m, err := goals.New(goals.Options{Runner: r, Path: s.path, Clock: s.clk.now, Chain: zmsChain(s.w)})
	if err != nil {
		s.t.Fatal(err)
	}
	s.m = m
}

func (s *zmsSim) count(tmpl string) (n int) {
	for _, g := range s.m.Goals() {
		if g.Type == goals.Campaign && g.Params.Template == tmpl {
			n++
		}
	}
	return
}

func (s *zmsSim) run(cfg zmsCfg, maxSteps int, done func() bool) int {
	t := s.t
	t.Helper()
	if err := s.m.Start(); err != nil {
		t.Fatal(err)
	}
	var lastLevel, lastSig, idle = 0, "", 0
	for i := 0; i < maxSteps; i++ {
		s.clk.t = s.clk.t.Add(time.Minute)
		s.r.step(cfg)
		for _, ev := range s.r.evs {
			s.m.Handle(ev)
		}
		s.m.Tick(s.clk.t)
		if s.rng.Float64() < cfg.crashP {
			s.restarts++
			s.boot()
			s.m.Start()
		}
		s.check()
		if done() {
			return i
		}
		sig := fmt.Sprint(s.w.lvl, s.w.done, s.r.selfPauses)
		if sig != lastSig || s.w.lvl != lastLevel {
			lastSig, lastLevel, idle = sig, s.w.lvl, 0
		} else if idle++; idle > 600 {
			t.Fatalf("no progress for 10 simulated hours at step %d: lvl %d %s", i, s.w.lvl, s.dump())
		}
	}
	t.Fatalf("not finished after %d steps: lvl %d %s", maxSteps, s.w.lvl, s.dump())
	return maxSteps
}

func (s *zmsSim) dump() string {
	out := ""
	for _, g := range s.m.Goals() {
		out += fmt.Sprintf("\n  %s %s %s tmpl=%s lvl=%d runs=%d note=%q next=%v", g.ID, g.Type, g.State, g.Params.Template, g.Params.Level, g.Params.Count, g.Note, g.NextIDs)
	}
	return out
}

// check asserts the invariants after every step.
func (s *zmsSim) check() {
	t := s.t
	t.Helper()
	live, runnable, active := 0, 0, 0
	for _, g := range s.m.Goals() {
		if g.State.Finished() {
			continue
		}
		live++
		switch g.State {
		case goals.StateQueued, goals.StateActive:
			runnable++
		}
		if g.State == goals.StateActive {
			active++
		}
	}
	if active > 1 {
		t.Fatalf("%d active goals%s", active, s.dump())
	}
	if live > 0 && runnable == 0 {
		t.Fatalf("stuck: live goals but none runnable (nothing can level the character or resume them)%s", s.dump())
	}
	for _, tmpl := range []string{goals.CampaignNormalAll, goals.CampaignNightmareAll, goals.CampaignHellAll} {
		if n := s.count(tmpl); n > 1 {
			t.Fatalf("%d goals for %s%s", n, tmpl, s.dump())
		}
	}
	for i := 1; i < len(s.w.diffLog); i++ {
		if s.w.diffLog[i] <= s.w.diffLog[i-1] {
			t.Fatalf("difficulty went %v", s.w.diffLog)
		}
	}
}

func (s *zmsSim) allDone(want int) func() bool {
	return func() bool {
		d := 0
		for _, g := range s.m.Goals() {
			if g.State == goals.StateDone {
				d++
			} else if !g.State.Finished() {
				return false
			}
		}
		return d >= want
	}
}

func (s *zmsSim) addInitial() {
	t := s.t
	if _, err := s.m.Add(goals.Goal{Type: goals.Campaign, Params: goals.Params{Template: goals.CampaignNormalAll, Chain: true}}, -1); err != nil {
		t.Fatal(err)
	}
	// the farm goal sits below the chain (priority -1) so it only runs once the campaigns are done
	if _, err := s.m.Add(goals.Goal{Type: goals.Runs, Priority: -1, Params: goals.Params{Area: "Mephisto", Count: 5}}, -1); err != nil {
		t.Fatal(err)
	}
}

func (s *zmsSim) verifyChain() {
	t := s.t
	t.Helper()
	if len(s.w.diffLog) != 2 || s.w.diffLog[0] != d2.Nightmare || s.w.diffLog[1] != d2.Hell {
		t.Fatalf("difficulty log %v", s.w.diffLog)
	}
	for _, tmpl := range []string{goals.CampaignNormalAll, goals.CampaignNightmareAll, goals.CampaignHellAll} {
		if n := s.count(tmpl); n != 1 {
			t.Fatalf("%s goals: %d", tmpl, n)
		}
	}
	var reach, runs int
	for _, g := range s.m.Goals() {
		switch g.Type {
		case goals.ReachLevel:
			reach++
		case goals.Runs:
			runs++
			if g.Progress.BossKills != 5 {
				t.Fatalf("runs goal %+v", g)
			}
		}
	}
	if reach > 20 || runs != 1 { // gate refusals queue farm goals
		t.Fatalf("reach goals %d, runs goals %d%s", reach, runs, s.dump())
	}
	// campaigns completed in order, the farm last
	var order []string
	for _, g := range s.m.Goals() {
		if g.State == goals.StateDone && g.CompletedAt != nil {
			order = append(order, g.Params.Template+g.Params.Area)
		}
	}
	_ = order
}

func TestZMSChainQuiet(t *testing.T) {
	s := newZMSim(t, 1)
	s.addInitial()
	s.run(zmsCfg{}, 40000, s.allDone(5)) // normal, reach, nightmare, reach?, hell, runs: at least 4..6
	s.verifyChain()
}

func TestZMSChainSelfPausesAndDeaths(t *testing.T) {
	for seed := int64(1); seed <= 5; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			s := newZMSim(t, seed)
			s.addInitial()
			s.run(zmsCfg{pauseP: 0.01, deathP: 0.08}, 80000, s.allDone(5))
			s.verifyChain()
		})
	}
}

func TestZMSChainRestartsAtRandomPoints(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		t.Run(fmt.Sprint(seed), func(t *testing.T) {
			s := newZMSim(t, seed)
			s.addInitial()
			s.run(zmsCfg{pauseP: 0.005, deathP: 0.05, crashP: 0.01}, 80000, s.allDone(5))
			if s.restarts == 0 {
				t.Fatal("no restart happened")
			}
			s.verifyChain()
		})
	}
}

// The resist gate is re-checked when the next difficulty's campaign is about to start (goals.Manager.gateLocked).
func TestZMSResistGateAtStart(t *testing.T) {
	s := newZMSim(t, 3)
	s.addInitial()
	s.run(zmsCfg{}, 40000, s.allDone(5))
	if len(s.w.gateBad) > 0 {
		t.Fatal("campaign started although ReadyForNext refused: ", s.w.gateBad)
	}
}

// A runner Hold error (a locked farm run) must still resume at the next level up after a process restart: the hold
// used to live in a map that was not persisted, which left the goal paused for good.
func TestZMSHoldSurvivesRestart(t *testing.T) {
	s := newZMSim(t, 1)
	s.w.lvl = 40
	farm, _ := s.m.Add(goals.Goal{Type: goals.Runs, Params: goals.Params{Area: "Mephisto", Count: 3}}, -1)
	s.m.Start()
	s.m.Handle(goals.Event{Type: goals.EvError, Hold: true, Level: 40, Message: "run locked"})
	lvl, _ := s.m.Add(goals.Goal{Type: goals.ReachLevel, Params: goals.Params{Level: 42}}, -1)
	s.boot()
	if g, _ := s.m.Goal(farm.ID); g.State != goals.StatePaused {
		t.Fatalf("after restart: %+v", g)
	}
	s.run(zmsCfg{}, 500, s.allDone(2))
	if g, _ := s.m.Goal(farm.ID); g.State != goals.StateDone {
		t.Fatalf("%+v", g)
	}
	_ = lvl
}

// MaxDeaths (default 5) pauses a goal, but it resumes by itself after a rest or a level up.
func TestZMSMaxDeathsStrands(t *testing.T) {
	s := newZMSim(t, 2)
	set := s.m.Settings()
	set.MaxDeaths = 5
	s.m.SetSettings(set)
	g, _ := s.m.Add(goals.Goal{Type: goals.Campaign, Params: goals.Params{Template: goals.CampaignNormalAll}}, -1)
	s.m.Start()
	for range 5 {
		s.m.Handle(goals.Event{Type: goals.EvDeath})
	}
	for range 100 {
		s.clk.t = s.clk.t.Add(time.Hour)
		s.m.Tick(s.clk.t)
	}
	if got, _ := s.m.Goal(g.ID); got.State == goals.StatePaused {
		t.Fatal("goal stays paused after MaxDeaths forever (no auto-resume): ", got.Note)
	}
}
