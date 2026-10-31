package task_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/task/sim"
)

// drive ticks until the planner is done or a limit is hit. rebuild, when set, replaces the executor before every tick
// (a bot restart at every node: there is no cursor, so nothing may be lost).
func drive(t *testing.T, g *sim.Game, c *task.Campaign, rebuild bool, max int) (*task.Executor, []task.Result) {
	t.Helper()
	ex := task.NewExecutor(g, g)
	var out []task.Result
	for i := 0; i < max; i++ {
		if rebuild {
			ex = task.NewExecutor(g, g)
		}
		r, err := ex.Tick(context.Background(), c)
		if err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
		out = append(out, r)
		if r.Outcome == task.OutDone || r.Stuck {
			return ex, out
		}
	}
	t.Fatalf("no end after %d ticks; last log: %v", max, tail(g.Log))
	return ex, out
}

func tail(l []string) []string {
	if len(l) > 12 {
		return l[len(l)-12:]
	}
	return l
}

func idx(log []string, s string) int {
	for i, l := range log {
		if l == s {
			return i
		}
	}
	return -1
}

func TestFullAct1Campaign(t *testing.T) {
	g := sim.New(true)
	c := task.NormalAct1(task.Act1Options{})
	_, res := drive(t, g, c, false, 200)
	if len(g.Violation) > 0 {
		t.Fatalf("rule violations: %v", g.Violation)
	}
	if g.W.Act != 2 {
		t.Fatalf("not in act 2: act %d, log %v", g.W.Act, tail(g.Log))
	}
	w, _ := g.World(context.Background())
	for _, q := range []string{task.QDen, task.QBurial, task.QTools, task.QTower, task.QAndariel} {
		if !task.QuestDone(w, q) {
			t.Errorf("quest %s not done: %+v", q, w.Quest(q))
		}
	}
	// order: Den before Blood Raven before Smith before Countess before Andariel
	order := []string{"kill:" + task.BossBR, "kill:" + task.BossSmith, "kill:" + task.BossCount, "kill:" + task.BossAndy}
	last := idx(g.Log, "talk:Akara:reward")
	if last < 0 {
		t.Fatal("no Den reward")
	}
	for _, k := range order {
		i := idx(g.Log, k)
		if i < last {
			t.Errorf("%s at %d before previous step at %d", k, i, last)
		}
		last = i
	}
	if n := g.Count("kill:"); n != 4 {
		t.Errorf("kills = %d, want 4 (no repeats): %v", n, g.Log)
	}
	if res[len(res)-1].Outcome != task.OutDone {
		t.Errorf("last outcome %v", res[len(res)-1].Outcome)
	}
}

// Crash/restart before every tick must reach the same end state with the same set of kills.
func TestRestartAtEveryNode(t *testing.T) {
	a, b := sim.New(true), sim.New(true)
	c := task.NormalAct1(task.Act1Options{})
	drive(t, a, c, false, 200)
	drive(t, b, c, true, 200)
	if b.W.Act != 2 || b.W.Level != a.W.Level {
		t.Fatalf("restarted run differs: act %d level %d vs %d", b.W.Act, b.W.Level, a.W.Level)
	}
	if len(b.Log) != len(a.Log) {
		t.Errorf("action count differs: %d vs %d", len(b.Log), len(a.Log))
	}
}

func TestAlreadyDoneZeroActions(t *testing.T) {
	g := sim.New(true)
	g.W.Act, g.W.Level = 2, 30
	for _, q := range []string{task.QDen, task.QBurial, task.QTools, task.QTower, task.QAndariel, task.QAct1Done} {
		g.Reader = true
		g.MarkDone(q)
	}
	ex := task.NewExecutor(g, g)
	r, err := ex.Tick(context.Background(), task.NormalAct1(task.Act1Options{}))
	if err != nil || r.Outcome != task.OutDone || len(g.Log) != 0 {
		t.Fatalf("outcome %v err %v log %v", r.Outcome, err, g.Log)
	}
}

// Without a quest reader the campaign still finishes: rewards are assumed after a clean talk.
func TestNoReaderUsesProxies(t *testing.T) {
	g := sim.New(false)
	c := task.NormalAct1(task.Act1Options{})
	ex, _ := drive(t, g, c, false, 300)
	if g.W.Act != 2 {
		t.Fatalf("act %d, log %v", g.W.Act, tail(g.Log))
	}
	if !ex.Assumed("quest.den_of_evil/reward") {
		t.Error("den reward should be assumed without a reader")
	}
	if p := c.Report(mustWorld(t, g), ex); p.Verified {
		t.Errorf("progress must not claim verified without a reader: %+v", p)
	}
}

func mustWorld(t *testing.T, g *sim.Game) task.World {
	w, err := g.World(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestLevelGatesInsertLevelSteps(t *testing.T) {
	g := sim.New(true)
	g.XPStep = 2
	drive(t, g, task.NormalAct1(task.Act1Options{}), false, 300)
	if g.Count("level") == 0 {
		t.Error("expected level steps")
	}
	if len(g.Violation) > 0 {
		t.Errorf("gate broken: %v", g.Violation)
	}
}

func TestLowPotionsInsertsChores(t *testing.T) {
	g := sim.New(true)
	g.SetLevel(22)
	g.W.Potions = 1
	g.W.Area, g.W.InTown = task.ColdPlains, false
	c := task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true})
	drive(t, g, c, false, 300)
	ic, ik := idx(g.Log, "chores"), idx(g.Log, "kill:"+task.BossAndy)
	if ic < 0 || ik < 0 || ic > ik {
		t.Fatalf("chores must come before Andariel: %v", g.Log)
	}
}

func TestDeathResumesWithoutCount(t *testing.T) {
	g := sim.New(true)
	g.DieOn["kill:"+task.BossBR] = 2
	_, res := drive(t, g, task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true}), false, 300)
	deaths := 0
	for _, r := range res {
		if r.Outcome == task.OutDeath {
			deaths++
		}
	}
	if deaths != 2 || g.W.Act != 2 {
		t.Fatalf("deaths %d act %d", deaths, g.W.Act)
	}
}

func TestThreeDeathsBlockTheQuestAndPlannerMovesOn(t *testing.T) {
	g := sim.New(true)
	g.DieOn["kill:"+task.BossBR] = 99
	c := task.NormalAct1(task.Act1Options{SkipTools: true})
	ex, res := drive(t, g, c, false, 400)
	last := res[len(res)-1]
	if last.Outcome != task.OutBlocked || !strings.Contains(last.Attention, "sisters_burial") {
		t.Fatalf("last = %+v", last)
	}
	if _, b := ex.Blocked(task.QBurial); !b {
		t.Error("burial group should be blocked")
	}
	if g.Count("kill:"+task.BossCount) == 0 {
		t.Error("planner should have moved on to the Countess")
	}
}

func TestNotFoundOptionalSkippedRequiredBlocks(t *testing.T) {
	g := sim.New(true)
	nf := task.Errorf(task.KindNotFound, "countess not found")
	g.Fail["kill:"+task.BossCount] = []error{nf, nf, nf, nf}
	c := task.NormalAct1(task.Act1Options{SkipTools: true})
	ex, _ := drive(t, g, c, false, 400)
	if _, b := ex.Blocked(task.QTower); !b {
		t.Fatal("tower should be blocked after 3 NotFound")
	}
	// Andariel was still reachable and the stuck report names the blocked quest
	if g.W.Act != 2 {
		t.Errorf("act %d; the rest of the campaign should continue", g.W.Act)
	}
}

func TestTimeoutRetriesOnceThenEscapes(t *testing.T) {
	g := sim.New(true)
	to := task.Wrap(task.KindTimeout, context.DeadlineExceeded)
	g.Fail["clear:8"] = []error{to, to}
	ex, res := drive(t, g, task.NormalAct1(task.Act1Options{SkipTools: true, SkipTower: true}), false, 400)
	if _, b := ex.Blocked(task.QDen); !b {
		t.Fatalf("den should block after a second timeout: %v", res[:6])
	}
	if g.Count("clear:8") != 2 {
		t.Errorf("clear ran %d times, want 2", g.Count("clear:8"))
	}
}

func TestInterruptedIsNotAFailure(t *testing.T) {
	g := sim.New(true)
	ex := task.NewExecutor(g, g)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r, _ := ex.Tick(ctx, task.NormalAct1(task.Act1Options{}))
	if r.Outcome != task.OutInterrupted {
		t.Fatalf("outcome %v", r.Outcome)
	}
}

func TestConstraintHook(t *testing.T) {
	g := sim.New(true)
	var started []string
	ex := task.NewExecutor(g, g)
	ex.Hooks.OnStart = func(s task.Step) {
		if s.Constraints.NoTownPortal {
			started = append(started, s.ID)
		}
	}
	p := fixedPlanner{task.Plan{Group: "x", Nodes: []task.Node{{Step: task.Step{
		ID: "ancients", Constraints: task.Constraints{NoTownPortal: true}, Budget: time.Second,
		Need: func(w task.World) (bool, task.Evidence) { return !w.Killed["done"], task.EvMemory },
		Do:   func(ctx context.Context, a task.Actions) error { g.W.Killed["done"] = true; return nil },
	}}}}}
	if _, err := ex.Tick(context.Background(), p); err != nil || len(started) != 1 {
		t.Fatalf("hook: %v %v", started, err)
	}
}

type fixedPlanner struct{ p task.Plan }

func (f fixedPlanner) Plan(task.World, task.Session) task.Plan { return f.p }

func TestBudgetCancelsStep(t *testing.T) {
	g := sim.New(true)
	ex := task.NewExecutor(g, g)
	p := fixedPlanner{task.Plan{Group: "x", Nodes: []task.Node{{Step: task.Step{
		ID: "slow", Budget: 20 * time.Millisecond,
		Need: func(w task.World) (bool, task.Evidence) { return true, task.EvMemory },
		Do:   func(ctx context.Context, a task.Actions) error { <-ctx.Done(); return ctx.Err() },
	}}}}}
	r, _ := ex.Tick(context.Background(), p)
	if r.Outcome != task.OutRetry || r.Kind != task.KindTimeout {
		t.Fatalf("%+v", r)
	}
}

func TestClassify(t *testing.T) {
	cases := []struct {
		err  error
		want task.Kind
	}{
		{nil, task.KindNone},
		{task.Errorf(task.KindDanger, "x"), task.KindDanger},
		{context.DeadlineExceeded, task.KindTimeout},
		{errors.New("boom"), task.KindUI},
		{&task.Fault{Kind: task.KindResource}, task.KindResource},
	}
	for _, c := range cases {
		if got := task.Classify(c.err); got != c.want {
			t.Errorf("Classify(%v) = %v, want %v", c.err, got, c.want)
		}
	}
}

// Need/Ready are pure: table over fake worlds.
func TestNeedTable(t *testing.T) {
	base := func() task.World {
		return task.World{Act: 1, Area: task.RogueEncampment, InTown: true, Level: 10, Potions: 5, HasTP: true,
			WPs: map[task.Area]bool{}, Killed: map[string]bool{}, Items: map[string]int{}, Cleared: map[task.Area]bool{}}
	}
	cleared := base()
	cleared.Cleared[task.DenOfEvil] = true
	killedBR := base()
	killedBR.Killed[task.BossBR] = true
	cases := []struct {
		name string
		w    task.World
		step string
		need bool
	}{
		{"den clear needed in den", func() task.World { w := base(); w.Area = task.DenOfEvil; w.InTown = false; return w }(), "quest.den_of_evil/clear", true},
		{"den cleared: reward needed", cleared, "quest.den_of_evil/reward", true},
		{"den cleared: travel to den not needed", cleared, "quest.den_of_evil/to-den", false},
		{"fresh: reward not needed", base(), "quest.den_of_evil/reward", false},
		{"BR dead: kashya reward", killedBR, "quest.sisters_burial/reward", true},
	}
	for _, c := range cases {
		var got *bool
		for _, q := range []string{task.QDen, task.QBurial} {
			for _, n := range task.Handler(q) {
				if n.Step.ID == c.step {
					v, _ := n.Step.Need(c.w)
					got = &v
				}
			}
		}
		if got == nil || *got != c.need {
			t.Errorf("%s: need = %v, want %v", c.name, got, c.need)
		}
	}
	// Ready is pure: Andariel gate and potion insert.
	for _, n := range task.Handler(task.QAndariel) {
		if n.Step.ID != "quest.sisters_to_slaughter/to-catacombs" {
			continue
		}
		w := base()
		w.Level = 10
		if f := n.Step.Ready(w); f == nil || f.Kind != task.KindDanger {
			t.Errorf("clvl 10: fault %v", f)
		}
		w.Level, w.Potions = 22, 0
		if f := n.Step.Ready(w); f == nil || f.Kind != task.KindResource || len(f.Insert) != 1 {
			t.Errorf("no potions: fault %v", f)
		}
		w.Potions = 6
		if f := n.Step.Ready(w); f != nil {
			t.Errorf("ready world faulted: %v", f)
		}
	}
}

func TestBossTravelGatesLevelAndPotions(t *testing.T) {
	cases := []struct {
		quest, step string
		lvl, pots   int
	}{
		{task.QTools, "to-barracks", 8, 4},
		{task.QTower, "to-cellar", 14, 4},
	}
	for _, c := range cases {
		found := false
		for _, n := range task.Handler(c.quest) {
			if n.Step.ID != "quest."+c.quest+"/"+c.step {
				continue
			}
			found = true
			w := task.World{Level: c.lvl - 1, Potions: 9, HasTP: true}
			if f := n.Step.Ready(w); f == nil || f.Kind != task.KindDanger {
				t.Errorf("%s low level: %v", c.step, f)
			}
			w.Level, w.Potions = c.lvl, c.pots-1
			if f := n.Step.Ready(w); f == nil || f.Kind != task.KindResource || len(f.Insert) != 1 {
				t.Errorf("%s low potions: %v", c.step, f)
			}
			w.Potions = c.pots
			if f := n.Step.Ready(w); f != nil {
				t.Errorf("%s ready world faulted: %v", c.step, f)
			}
		}
		if !found {
			t.Errorf("no step %s", c.step)
		}
	}
}

func TestProgressReport(t *testing.T) {
	g := sim.New(true)
	c := task.NormalAct1(task.Act1Options{})
	ex := task.NewExecutor(g, g)
	for i := 0; i < 6; i++ {
		ex.Tick(context.Background(), c)
	}
	p := c.Report(mustWorld(t, g), ex)
	if p.Target != len(c.Entries) || len(p.Steps) < len(c.Entries) || p.Phase == "" || p.ETASeconds <= 0 {
		t.Fatalf("report: %+v", p)
	}
	if !p.Verified || p.Source != "memory" {
		t.Errorf("reader present: source %q verified %v", p.Source, p.Verified)
	}
	if _, err := json.Marshal(p); err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, s := range p.Steps {
		if s.State == goals.StateActive {
			active++
		}
	}
	if active != 1 {
		t.Errorf("want one active step, got %d: %+v", active, p.Steps)
	}
	// finished campaign: 100 %
	drive(t, g, c, false, 300)
	p = c.Report(mustWorld(t, g), ex)
	if p.Percent != 100 || p.Current != p.Target {
		t.Errorf("done report: %+v", p)
	}
}

// Golden plan node ids; run with TASK_UPDATE=1 to rewrite after a deliberate change.
func TestGoldenPlans(t *testing.T) {
	var b strings.Builder
	for _, q := range []string{task.QDen, task.QBurial, task.QCain, task.QTools, task.QTower, task.QAndariel, task.QAct1Done} {
		b.WriteString("# " + q + "\n")
		for _, n := range task.Handler(q) {
			b.WriteString(n.Step.ID)
			if len(n.Requires) > 0 {
				b.WriteString(" <- " + strings.Join(n.Requires, ","))
			}
			b.WriteString("\n")
		}
	}
	const path = "testdata/plans/act1.txt"
	if os.Getenv("TASK_UPDATE") != "" {
		os.MkdirAll("testdata/plans", 0o755)
		os.WriteFile(path, []byte(b.String()), 0o644)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil || string(want) != b.String() {
		t.Fatalf("plans differ from %s (TASK_UPDATE=1 to accept):\n%s", path, b.String())
	}
}

func TestCainOption(t *testing.T) {
	g := sim.New(true)
	_, _ = drive(t, g, task.NormalAct1(task.Act1Options{Cain: true}), false, 400)
	w := mustWorld(t, g)
	if g.W.Act != 2 || !task.QuestDone(w, task.QCain) || len(g.Violation) > 0 {
		t.Fatalf("act %d cain %+v viol %v log %v", g.W.Act, w.Quest(task.QCain), g.Violation, g.Log)
	}
}

// A spliced chores step that can never succeed (no gold) must not starve the level step once its group is blocked.
type choreEnv struct{}

func (choreEnv) World(context.Context) (task.World, error) {
	return task.World{InTown: true, Level: 12}, nil
}

type choreActs struct {
	task.Actions
	chores, level int
}

func (a *choreActs) Chores(context.Context) error       { a.chores++; return nil }
func (a *choreActs) LevelTo(context.Context, int) error { a.level++; return nil }

type blockPlanner struct{}

func (blockPlanner) Plan(w task.World, s task.Session) task.Plan {
	if _, b := s.Blocked("q"); b {
		return task.Plan{ID: "level.14", Nodes: []task.Node{{Step: task.StepLevel(14)}}}
	}
	q := task.Step{ID: "q/go", Title: "go", Phase: "travel",
		Need: func(task.World) (bool, task.Evidence) { return true, task.EvMemory },
		Ready: func(task.World) *task.Fault {
			return &task.Fault{Kind: task.KindResource, Why: "potions", Insert: []task.Step{chores()}}
		},
		Do: func(context.Context, task.Actions) error { return nil }}
	return task.Plan{ID: "quest.q", Group: "q", Nodes: []task.Node{{Step: q}}}
}

func chores() task.Step {
	s := task.StepChores("q/chores")
	s.Need = func(task.World) (bool, task.Evidence) { return true, task.EvMemory }
	return s
}

func TestBlockedInsertedStepDoesNotStarveLevelStep(t *testing.T) {
	acts := &choreActs{}
	ex := task.NewExecutor(choreEnv{}, acts)
	for i := 0; i < 12; i++ {
		if _, err := ex.Tick(context.Background(), blockPlanner{}); err != nil {
			t.Fatal(err)
		}
	}
	if acts.chores > 3 || acts.level == 0 {
		t.Fatalf("chores ran %d times, level %d", acts.chores, acts.level)
	}
}

func TestForgetFailsLetsAGivenUpPrerequisiteRunAgain(t *testing.T) {
	acts := &choreActs{}
	ex := task.NewExecutor(choreEnv{}, acts)
	for i := 0; i < 12; i++ {
		if _, err := ex.Tick(context.Background(), blockPlanner{}); err != nil {
			t.Fatal(err)
		}
	}
	before := acts.chores
	ex.ForgetFails()
	for i := 0; i < 6; i++ {
		if _, err := ex.Tick(context.Background(), blockPlanner{}); err != nil {
			t.Fatal(err)
		}
	}
	if acts.chores <= before {
		t.Fatalf("chores still given up after ForgetFails: %d -> %d", before, acts.chores)
	}
}
