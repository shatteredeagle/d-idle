package boss

import (
	"context"
	"errors"
	"testing"
	"time"
)

func mustTable(t *testing.T) FarmTable {
	t.Helper()
	tab, err := DefaultFarmTable()
	if err != nil {
		t.Fatal(err)
	}
	return tab
}

func TestFarmTableCoversRunsJSONAndKnownBosses(t *testing.T) {
	tab := mustTable(t)
	known := AllActs()
	for k, v := range FarmScripts() {
		known[k] = v
	}
	for _, r := range tab {
		if r.Act == 0 {
			t.Errorf("%s: not in runs.json (no act)", r.Name)
		}
		for _, s := range r.Steps {
			if _, ok := known[s]; !ok {
				t.Errorf("%s: step %q has no script", r.Name, s)
			}
		}
	}
	for _, n := range []string{"Mephisto", "Andariel", "Countess", "Pindleskin", "Eldritch+Shenk", "Baal"} {
		if _, ok := tab.Find(n); !ok {
			t.Errorf("missing run %s", n)
		}
	}
	if r, ok := tab.Find("eldritch shenk"); !ok || r.Name != "Eldritch+Shenk" {
		t.Errorf("Find with spaces: %v %v", r.Name, ok)
	}
	if r, _ := tab.Find("Baal"); r.Floor(2) != 82 {
		t.Errorf("Baal Hell floor from runs.json = %d", r.Floor(2))
	}
	if r, _ := tab.Find("Pindleskin"); r.Floor(2) != 70 {
		t.Errorf("Pindleskin override floor = %d", r.Floor(2))
	}
}

func TestExpectedXPMatchesDocTable(t *testing.T) {
	tab := mustTable(t)
	baal, _ := tab.Find("Baal")
	for _, c := range []struct {
		d, lvl int
		want   float64
	}{{2, 75, 3260000}, {2, 70, 4320000}, {2, 72, 3896000}, {1, 55, 1190000}, {2, 99, 1130000}, {2, 60, 0}} {
		if got := baal.ExpectedXP(c.d, c.lvl); got < c.want-1 || got > c.want+1 {
			t.Errorf("Baal d%d clvl %d: %.0f, want %.0f", c.d, c.lvl, got, c.want)
		}
	}
}

func rankNames(t *testing.T, c FarmChar, goal string) []string {
	var out []string
	for _, ch := range mustTable(t).Rank(c, goal, nil) {
		out = append(out, ch.Run.Name)
	}
	return out
}

func TestSchedulerPicksByLevelClassAndGoal(t *testing.T) {
	tab := mustTable(t)
	// Hell clvl 75 is under Baal's floor 82: Mephisto is the XP pick; at 85 Baal wins (docs/merc-farm.md 2.2).
	c := FarmChar{Level: 75, Class: "Sorceress", Difficulty: 2, Teleport: true, Has: map[string]bool{"anya_portal": true}}
	if r, ok := tab.Pick(c, GoalXP, nil); !ok || r.Name != "Mephisto" {
		t.Fatalf("hell 75 xp: %v %v", r.Name, ok)
	}
	c85 := c
	c85.Level = 85
	if r, ok := tab.Pick(c85, GoalXP, nil); !ok || r.Name != "Baal" {
		t.Fatalf("hell 85 xp: %v %v", r.Name, ok)
	}
	// Runes: Countess.
	if r, _ := tab.Pick(c, GoalRunes, nil); r.Name != "Countess" {
		t.Fatalf("runes: %v", r.Name)
	}
	// Uniques: Pindleskin (short, high weight) when Anya's portal is open, not offered without it.
	if r, _ := tab.Pick(c, GoalUniques, nil); r.Name != "Pindleskin" {
		t.Fatalf("uniques: %v", r.Name)
	}
	c.Has = nil
	if r, _ := tab.Pick(c, GoalUniques, nil); r.Name == "Pindleskin" {
		t.Fatal("Pindleskin offered without anya_portal")
	}
	// Below every floor: nothing.
	if _, ok := tab.Pick(FarmChar{Level: 10, Difficulty: 0}, GoalXP, nil); ok {
		t.Fatal("clvl 10 must have no run")
	}
	// Nightmare clvl 50: Baal NM floor 58 blocks it, Mephisto NM (floor 55) is out too, Andariel NM is the XP pick.
	if got := rankNames(t, FarmChar{Level: 50, Difficulty: 1, Teleport: true}, GoalXP); len(got) == 0 || got[0] != "Andariel" {
		t.Fatalf("nm 50: %v", got)
	}
	// No teleport slows Baal and Mephisto: at clvl 85 Hell Baal stays ahead anyway, and the skip list drops a run.
	c = FarmChar{Level: 85, Difficulty: 2}
	rk := tab.Rank(c, GoalXP, map[string]bool{"Baal": true})
	if len(rk) == 0 || rk[0].Run.Name == "Baal" {
		t.Fatalf("skip ignored: %+v", rk)
	}
	// Class multiplier: a Necromancer is ranked lower on Hell Andariel than a Sorceress.
	a, _ := tab.Find("Andariel")
	score := func(class string) float64 {
		for _, ch := range tab.Rank(FarmChar{Level: 75, Class: class, Difficulty: 2, Teleport: true}, GoalXP, nil) {
			if ch.Run.Name == a.Name {
				return ch.Score
			}
		}
		return 0
	}
	if score("Necromancer") >= score("Sorceress") {
		t.Fatal("necromancer should score lower on hell Andariel")
	}
}

func TestPindleskinThroughRedPortal(t *testing.T) {
	o := newObjFake()
	f := o.fake
	f.level, f.potions, f.area, f.diff = 75, 5, 1, 2
	f.presets = []Preset{{Kind: PresetObject, ID: objRedPortal, Pos: Pt{500, 500}}}
	f.addBoss(Mon{Class: 800, UniqueNo: uniquePindleskin, Name: NamePindleskin, Pos: Pt{200, 200}}, AreaNihlathakTemple, 60)
	o.onUse = func(o *objFake, class int) { o.fake.arrive(AreaNihlathakTemple) }
	res, err := RunPindleskin(context.Background(), f, f, o, Opts{})
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v log %v", res, err, f.log)
	}
	if f.log[0] != "travel 109" || o.uses[0] != objRedPortal {
		t.Fatalf("log %v uses %v", f.log, o.uses)
	}
	if f.did("town") != 1 {
		t.Fatalf("Pindleskin exits to town: %v", f.log)
	}
	// Without the object capability nothing is walked.
	f2 := newFake()
	f2.level, f2.area = 75, 1
	if _, err := RunPindleskin(context.Background(), f2, f2, nil, Opts{}); !errors.Is(err, ErrAborted) || len(f2.log) != 0 {
		t.Fatalf("no ops: %v %v", err, f2.log)
	}
	// Too weak for the difficulty.
	f3 := newFake()
	f3.level, f3.diff = 40, 2
	if _, err := RunPindleskin(context.Background(), f3, f3, nil, Opts{}); !errors.Is(err, ErrTooWeak) {
		t.Fatalf("weak: %v", err)
	}
}

func TestEldritchFromTownViaWaypoint(t *testing.T) {
	f := newFake()
	f.level, f.potions, f.area = 30, 4, 109
	f.addBoss(Mon{Class: 700, UniqueNo: uniqueEldritch, Name: NameEldritch, Pos: Pt{300, 300}}, AreaFrigidHighlands, 60)
	r := &FarmRunner{Act45Runner: Act45Runner{Runner: Runner{W: f, A: f}}}
	if err := r.KillBoss(context.Background(), NameEldritch); err != nil || !r.LastResult().Killed {
		t.Fatalf("err %v res %+v", err, r.Last)
	}
	if f.log[0] != "travel 111" {
		t.Fatalf("log %v", f.log)
	}
	if err := r.KillBoss(context.Background(), "Nobody"); err == nil {
		t.Fatal("unknown boss must fail")
	}
}

// loop fakes

type fakeKiller struct {
	w     *fake
	last  Result
	calls []string
	do    func(name string, n int) (loot int, err error)
}

func (k *fakeKiller) KillBoss(ctx context.Context, name string) error {
	k.calls = append(k.calls, name)
	k.w.now = k.w.now.Add(2 * time.Minute)
	n, err := 0, error(nil)
	if k.do != nil {
		n, err = k.do(name, len(k.calls))
	}
	k.last = Result{Boss: name, Loot: n, Killed: err == nil}
	return err
}
func (k *fakeKiller) LastResult() Result { return k.last }

type fakeOps struct {
	c        FarmChar
	games    int
	chores   int
	choreErr error
}

func (o *fakeOps) Char(ctx context.Context) (FarmChar, error) { return o.c, nil }
func (o *fakeOps) NewGame(ctx context.Context) error          { o.games++; return nil }
func (o *fakeOps) Chores(ctx context.Context) error           { o.chores++; return o.choreErr }

func hell85() FarmChar { return FarmChar{Level: 85, Difficulty: 2, Teleport: true} }

func TestFarmLoopRunCountAndPerRunOrder(t *testing.T) {
	f := newFake()
	k := &fakeKiller{w: f, do: func(string, int) (int, error) { return 3, nil }}
	ops := &fakeOps{c: hell85()}
	rep, err := FarmLoop(context.Background(), f, ops, k, mustTable(t), FarmOpts{Fixed: "Eldritch+Shenk", Limits: Limits{Runs: 3}, FirstHere: true})
	if err != nil || rep.Stop != "runs" || rep.Runs != 3 || rep.Kills != 3 {
		t.Fatalf("%+v %v", rep, err)
	}
	if ops.games != 2 || ops.chores != 3 {
		t.Fatalf("games %d chores %d (first run stays in the current game)", ops.games, ops.chores)
	}
	if len(k.calls) != 6 || k.calls[0] != NameEldritch || k.calls[1] != NameShenk {
		t.Fatalf("calls %v", k.calls)
	}
	if rep.Loot != 18 || rep.ByRun["Eldritch+Shenk"].Kills != 3 {
		t.Fatalf("tally %+v", rep.ByRun["Eldritch+Shenk"])
	}
}

func TestFarmLoopStopsOnTimeAndDeaths(t *testing.T) {
	f := newFake()
	k := &fakeKiller{w: f}
	rep, _ := FarmLoop(context.Background(), f, &fakeOps{c: hell85()}, k, mustTable(t),
		FarmOpts{Fixed: "Andariel", Limits: Limits{Duration: 7 * time.Minute}})
	if rep.Stop != "time" || rep.Runs != 4 {
		t.Fatalf("time stop: %+v", rep)
	}
	f = newFake()
	k = &fakeKiller{w: f, do: func(string, int) (int, error) { return 0, ErrDead }}
	rep, _ = FarmLoop(context.Background(), f, &fakeOps{c: hell85()}, k, mustTable(t),
		FarmOpts{Fixed: "Andariel", Limits: Limits{Runs: 10, Deaths: 2}})
	if rep.Stop != "deaths" || rep.Deaths != 2 || rep.Runs != 2 {
		t.Fatalf("death stop: %+v", rep)
	}
}

func TestFarmLoopSchedulerSwitchesAwayFromFailingRun(t *testing.T) {
	f := newFake()
	k := &fakeKiller{w: f, do: func(name string, _ int) (int, error) {
		if name == "Baal" {
			return 0, ErrBossNotFound
		}
		return 1, nil
	}}
	rep, _ := FarmLoop(context.Background(), f, &fakeOps{c: hell85()}, k, mustTable(t), FarmOpts{Limits: Limits{Runs: 4}})
	if k.calls[0] != "Baal" || k.calls[1] != "Baal" {
		t.Fatalf("calls %v", k.calls)
	}
	if k.calls[2] == "Baal" || rep.Kills != 2 || rep.ByRun["Baal"].Fails != 2 {
		t.Fatalf("no switch: %v %+v", k.calls, rep)
	}
}

func TestFarmLoopTooWeakSkipsWithoutCounting(t *testing.T) {
	f := newFake()
	k := &fakeKiller{w: f, do: func(name string, _ int) (int, error) {
		if name == "Baal" {
			return 0, ErrTooWeak
		}
		return 0, nil
	}}
	rep, _ := FarmLoop(context.Background(), f, &fakeOps{c: hell85()}, k, mustTable(t), FarmOpts{Limits: Limits{Runs: 2}})
	if rep.Runs != 2 || rep.ByRun["Baal"].Runs != 0 {
		t.Fatalf("%+v", rep)
	}
}

func TestFarmLoopConsecutiveFailsAndNoRunAndCtx(t *testing.T) {
	f := newFake()
	k := &fakeKiller{w: f, do: func(string, int) (int, error) { return 0, ErrTimeout }}
	rep, _ := FarmLoop(context.Background(), f, &fakeOps{c: hell85()}, k, mustTable(t),
		FarmOpts{Fixed: "Mephisto", Limits: Limits{Runs: 50, ConsecFails: 3}})
	if rep.Stop != "fails" || rep.Runs != 3 {
		t.Fatalf("%+v", rep)
	}
	rep, _ = FarmLoop(context.Background(), f, &fakeOps{c: FarmChar{Level: 5}}, k, mustTable(t), FarmOpts{})
	if rep.Stop != "none" || rep.Runs != 0 {
		t.Fatalf("none: %+v", rep)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if rep, err := FarmLoop(ctx, f, &fakeOps{c: hell85()}, k, mustTable(t), FarmOpts{}); rep.Stop != "ctx" || err == nil {
		t.Fatalf("ctx: %+v %v", rep, err)
	}
}

func TestFarmLoopChoreFailureDoesNotStopTheRun(t *testing.T) {
	f := newFake()
	k := &fakeKiller{w: f}
	ops := &fakeOps{c: hell85(), choreErr: errors.New("no gold")}
	rep, _ := FarmLoop(context.Background(), f, ops, k, mustTable(t), FarmOpts{Fixed: "Andariel", Limits: Limits{Runs: 1}})
	if rep.Kills != 1 {
		t.Fatalf("%+v", rep)
	}
}

func TestFarmLoopFixedRunLocked(t *testing.T) {
	f := newFake()
	k := &fakeKiller{w: f}
	// Pindleskin needs anya_portal: without it the fixed run must not start or retry
	ops := &fakeOps{c: hell85()}
	rep, err := FarmLoop(context.Background(), f, ops, k, mustTable(t), FarmOpts{Fixed: "Pindleskin", Limits: Limits{Runs: 5}, FirstHere: true})
	if !errors.Is(err, ErrRunLocked) || rep.Stop != "locked" || len(k.calls) != 0 || ops.games != 0 {
		t.Fatalf("%+v %v calls %v", rep, err, k.calls)
	}
	// clvl floor applies to a fixed run as well
	low := &fakeOps{c: FarmChar{Level: 1, Difficulty: 2, Teleport: true}}
	if _, err = FarmLoop(context.Background(), f, low, k, mustTable(t), FarmOpts{Fixed: "Mephisto", Limits: Limits{Runs: 1}, FirstHere: true}); !errors.Is(err, ErrRunLocked) {
		t.Fatalf("floor: %v", err)
	}
	// unlocked: runs
	ok := &fakeOps{c: FarmChar{Level: 85, Difficulty: 2, Has: map[string]bool{"anya_portal": true}}}
	rep, err = FarmLoop(context.Background(), f, ok, k, mustTable(t), FarmOpts{Fixed: "Pindleskin", Limits: Limits{Runs: 1}, FirstHere: true})
	if err != nil || rep.Runs != 1 {
		t.Fatalf("%+v %v", rep, err)
	}
	if _, err = FarmLoop(context.Background(), f, ok, k, mustTable(t), FarmOpts{Fixed: "Nobody", Limits: Limits{Runs: 1}}); !errors.Is(err, ErrRunLocked) {
		t.Fatalf("unknown name: %v", err)
	}
}
