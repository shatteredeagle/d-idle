package task_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"d2lod-bot/internal/task"
	"d2lod-bot/internal/task/sim"
)

// chain stitches the per-act fakes (sim.Game for act 1, act2Game, act3Game, act45Game) into one world so a whole
// campaign runs end to end. The hand-offs (Warriv, Meshif, the red portal, Tyrael) are the fakes' own rules; chain only
// carries the character across them: level, gold, potions, items, kills, waypoints and quest words. Every rule inside a
// fake is U (see the fakes); what this tests is the campaign's ordering and the handlers' prerequisites.
type chain struct {
	reader bool
	diff   string
	cur    interface {
		task.Env
		task.Actions
	}
	act  int
	Log  []string
	Viol []string
}

func newChain(reader bool, diff string, level int) *chain {
	c := &chain{reader: reader, diff: diff}
	g := sim.New(reader)
	g.SetLevel(level)
	g.W.Difficulty = diff
	c.cur, c.act = g, 1
	return c
}

// raw returns the world of the current fake including its quest words (World copies, so it is safe to mutate).
func (c *chain) World(ctx context.Context) (task.World, error) { return c.cur.World(ctx) }

func (c *chain) viol(f string, a ...any) { c.Viol = append(c.Viol, fmt.Sprintf(f, a...)) }

// advance builds the next act's fake when the current one moved the character to a new act.
func (c *chain) advance() {
	w, _ := c.cur.World(context.Background())
	native := c.act
	if native == 5 {
		native = 4 // act45Game covers both
	}
	if w.Act == native || (c.act >= 4 && w.Act >= 4) {
		if c.act == 4 && w.Act == 5 {
			c.act = 5
		}
		return
	}
	if w.Act != c.act+1 {
		c.viol("jumped from act %d to act %d", c.act, w.Act)
		return
	}
	stage := map[string]task.Stage{}
	for q, s := range w.Quests {
		stage[q] = s.Stage
	}
	var nw *task.World
	switch w.Act {
	case 2:
		g := newAct2(c.reader, w.Level)
		g.stage, nw = stage, &g.W
		c.cur = g
	case 3:
		g := newAct3(c.reader, w.Level)
		g.stage, nw = stage, &g.W
		c.cur = g
	case 4:
		g := newAct45(c.reader, 4, w.Level)
		g.stage, nw = stage, &g.W
		c.cur = g
	}
	c.act = w.Act
	nw.Difficulty, nw.Gold, nw.Potions = c.diff, max(nw.Gold, w.Gold), w.Potions
	for a := range w.WPs {
		nw.WPs[a] = true
	}
	for k, v := range w.Killed {
		nw.Killed[k] = v
	}
	for k, v := range w.Items {
		if v > 0 && nw.Items[k] == 0 {
			nw.Items[k] = v
		}
	}
	for a, v := range w.Cleared {
		nw.Cleared[a] = v
	}
	if w.Act == 4 {
		delete(nw.Items, task.ItemSoulstone) // the act 4 fake hands the Hellforge optional a stone; the campaign skips it
	}
}

func (c *chain) do(name string, err error) error {
	c.Log = append(c.Log, name)
	c.advance()
	return err
}

func (c *chain) Travel(ctx context.Context, a task.Area) error {
	return c.do(fmt.Sprintf("travel:%d", a), c.cur.Travel(ctx, a))
}
func (c *chain) ToTown(ctx context.Context) error { return c.do("town", c.cur.ToTown(ctx)) }
func (c *chain) Clear(ctx context.Context, a task.Area) error {
	return c.do(fmt.Sprintf("clear:%d", a), c.cur.Clear(ctx, a))
}
func (c *chain) KillBoss(ctx context.Context, n string) error {
	return c.do("kill:"+n, c.cur.KillBoss(ctx, n))
}
func (c *chain) Talk(ctx context.Context, npc, pick string) error {
	return c.do("talk:"+npc+":"+pick, c.cur.Talk(ctx, npc, pick))
}
func (c *chain) Interact(ctx context.Context, o string) error {
	return c.do("interact:"+o, c.cur.Interact(ctx, o))
}
func (c *chain) PickUp(ctx context.Context, i string) error {
	return c.do("pickup:"+i, c.cur.PickUp(ctx, i))
}
func (c *chain) ActivateWaypoint(ctx context.Context, a task.Area) error {
	return c.do(fmt.Sprintf("wp:%d", a), c.cur.ActivateWaypoint(ctx, a))
}
func (c *chain) Chores(ctx context.Context) error { return c.do("chores", c.cur.Chores(ctx)) }
func (c *chain) LevelTo(ctx context.Context, n int) error {
	return c.do("level", c.cur.LevelTo(ctx, n))
}

func (c *chain) Transmute(ctx context.Context, in []string) error {
	if t, ok := c.cur.(task.Transmuter); ok {
		return c.do("transmute:"+strings.Join(in, "+"), t.Transmute(ctx, in))
	}
	return task.Errorf(task.KindPrereq, "act %d has no cube", c.act)
}
func (c *chain) HireMerc(ctx context.Context, act int, aura string) error {
	if t, ok := c.cur.(task.MercHirer); ok {
		return c.do(fmt.Sprintf("hire:%d", act), t.HireMerc(ctx, act, aura))
	}
	return task.Errorf(task.KindPrereq, "act %d cannot hire", c.act)
}
func (c *chain) SmashOrb(ctx context.Context, w string) error {
	if t, ok := c.cur.(task.OrbSmasher); ok {
		return c.do("smash:"+w, t.SmashOrb(ctx, w))
	}
	return task.Errorf(task.KindPrereq, "no orb in act %d", c.act)
}
func (c *chain) UseForge(ctx context.Context, s, h string) error {
	if t, ok := c.cur.(task.ForgeUser); ok {
		return c.do("forge", t.UseForge(ctx, s, h))
	}
	return task.Errorf(task.KindPrereq, "no forge in act %d", c.act)
}
func (c *chain) UseItem(ctx context.Context, i string) error {
	if t, ok := c.cur.(task.ItemUser); ok {
		return c.do("use:"+i, t.UseItem(ctx, i))
	}
	return task.Errorf(task.KindPrereq, "act %d cannot use items", c.act)
}

// violations collects the rule breaks every fake noticed.
func (c *chain) violations() []string {
	v := append([]string(nil), c.Viol...)
	switch g := c.cur.(type) {
	case *sim.Game:
		v = append(v, g.Violation...)
	case *act2Game:
		v = append(v, g.Viol...)
	case *act3Game:
		v = append(v, g.Viol...)
	case *act45Game:
		v = append(v, g.Viol...)
	}
	return v
}

// run drives the campaign to the end. rebuild creates a fresh Executor every tick (a bot restart at every node).
// The floor check: a quest group's step never starts below its entry floor (hop and turn-in steps excluded: they
// happen wherever the character stands).
func (c *chain) run(t *testing.T, camp *task.Campaign, rebuild bool, maxTicks int) (order []string) {
	t.Helper()
	floors := map[string]int{}
	for _, e := range camp.Entries {
		floors[e.Quest] = e.MinLevel
	}
	ex := task.NewExecutor(c, c)
	seen := map[string]bool{}
	for i := 0; i < maxTicks; i++ {
		if rebuild {
			ex = task.NewExecutor(c, c)
		}
		before, _ := c.World(context.Background())
		r, err := ex.Tick(context.Background(), camp)
		if err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
		if r.Outcome == task.OutDone {
			return order
		}
		if r.Stuck {
			t.Fatalf("stuck at tick %d (act %d): %s; log %v", i, c.act, r.Attention, c.Log)
		}
		if r.Group != "" && r.Outcome == task.OutRan {
			if !seen[r.Group] {
				seen[r.Group] = true
				order = append(order, r.Group)
			}
			if f := floors[r.Group]; before.Level < f && strings.Contains(r.StepID, "/") &&
				!strings.HasSuffix(r.StepID, "/home") && !strings.HasPrefix(r.StepID, "chores") {
				c.viol("step %s ran at clvl %d below the entry floor %d", r.StepID, before.Level, f)
			}
		}
	}
	t.Fatalf("no end after %d ticks (act %d); tail %v", maxTicks, c.act, c.Log[max(0, len(c.Log)-15):])
	return nil
}

func normalAll() *task.Campaign { return task.Template("normal_all", false) }

func checkNormalEnd(t *testing.T, c *chain) {
	t.Helper()
	if v := c.violations(); len(v) > 0 {
		t.Fatalf("violations: %v", v)
	}
	g, ok := c.cur.(*act45Game)
	if !ok || !g.W.Killed[task.BossBaal] || g.W.Act != 5 {
		t.Fatalf("campaign ended in act %d without Baal; tail %v", c.act, c.Log[max(0, len(c.Log)-10):])
	}
}

func TestCampaignNormalAllWithReader(t *testing.T) {
	c := newChain(true, "normal", 1)
	c.run(t, normalAll(), false, 3000)
	checkNormalEnd(t, c)
}

func TestCampaignNormalAllProxiesOnly(t *testing.T) {
	c := newChain(false, "normal", 1)
	c.run(t, normalAll(), false, 3000)
	checkNormalEnd(t, c)
}

func TestCampaignNormalAllRestartAtEveryNode(t *testing.T) {
	c := newChain(true, "normal", 1)
	c.run(t, normalAll(), true, 3000)
	checkNormalEnd(t, c)
}

// newGame starts a fresh game of diff after the previous difficulty's Baal: the character keeps level, gold and the
// Cube, everything else (quest words, kills, pseudo items, waypoints) is per game or per difficulty.
func (c *chain) newGame(diff string, level int) *chain {
	n := newChain(c.reader, diff, level)
	w, _ := c.cur.World(context.Background())
	g := n.cur.(*sim.Game)
	g.W.Gold, g.W.Potions = w.Gold, w.Potions
	if w.Items[task.ItemCube] > 0 {
		g.W.Items[task.ItemCube] = 1
	}
	return n
}

// Normal, then Nightmare, then Hell, each a full Acts 1-5 run in a new game of the next difficulty, gated by
// NextDifficulty and ReadyForNext.
func TestCampaignDifficultyChain(t *testing.T) {
	for _, reader := range []bool{true, false} {
		c := newChain(reader, "normal", 1)
		c.run(t, normalAll(), false, 3000)
		checkNormalEnd(t, c)
		for _, step := range []struct{ cur, next, camp string }{{"normal", "nightmare", "nightmare_all"}, {"nightmare", "hell", "hell_all"}} {
			w, _ := c.World(context.Background())
			if got := task.NextDifficulty(step.cur, w.Killed[task.BossBaal]); got != step.next {
				t.Fatalf("reader=%v NextDifficulty(%s) = %q", reader, step.cur, got)
			}
			lvl := w.Level
			if ok, _ := task.ReadyForNext(step.next, lvl, nil); !ok { // the bot farms the old difficulty until it is
				lvl = task.EntryLevel(step.next)
			}
			c = c.newGame(step.next, lvl)
			camp := task.Template(step.camp, false)
			// the campaign refuses the old difficulty and accepts the new game's
			if p := camp.Plan(task.World{Act: 1, Difficulty: step.cur, Level: 99}, nil); !p.Stuck {
				t.Fatalf("%s accepted a %s world", step.camp, step.cur)
			}
			c.run(t, camp, false, 4000)
			checkNormalEnd(t, c)
			if got := c.violations(); len(got) > 0 {
				t.Fatalf("%v", got)
			}
		}
	}
}

// wipeGame is what a game restart forgets: the kills of this game and the pseudo items the adapter derives from clicks
// (a new game clears them, internal/bot taskEnv). Real items, quest words and waypoints stay.
func (c *chain) wipeGame() {
	var w *task.World
	switch g := c.cur.(type) {
	case *sim.Game:
		w = &g.W
	case *act2Game:
		w = &g.W
	case *act3Game:
		w = &g.W
	case *act45Game:
		w = &g.W
	}
	// act45Game gates the Throne on Killed[Ancients]; in the game it is the quest, which a restart keeps.
	w.Killed = map[string]bool{task.BossAncients: w.Killed[task.BossAncients]}
	w.Cleared = map[task.Area]bool{}
	for _, it := range []string{task.ItemStaffPlaced, task.ItemBookRead, task.ItemOrbSmashed, task.ItemPortal, task.ItemScrollRead,
		task.ItemAnyaTouched, task.ItemAnyaThawed, task.ItemForgeUsed} {
		delete(w.Items, it)
	}
}

// restartRun runs a normal_all campaign with the game restarted once after action number at (0 = never) and checks it
// still ends at Baal with no rule broken (it may repeat work the quest words do not prove done).
func restartRun(t *testing.T, camp *task.Campaign, at int) {
	t.Helper()
	wiped := false
	c := newChain(true, "normal", 1)
	ex := task.NewExecutor(c, c)
	done := false
	for i := 0; i < 3000 && !done; i++ {
		if !wiped && at > 0 && len(c.Log) >= at {
			c.wipeGame()
			ex = task.NewExecutor(c, c)
			wiped = true
		}
		r, err := ex.Tick(context.Background(), camp)
		if err != nil {
			t.Fatal(err)
		}
		if r.Stuck {
			t.Fatalf("restart after action %d: stuck in act %d: %s; tail %v", at, c.act, r.Attention, c.Log[max(0, len(c.Log)-8):])
		}
		done = r.Outcome == task.OutDone
	}
	if !done {
		t.Fatalf("restart after action %d did not end; act %d tail %v", at, c.act, c.Log[max(0, len(c.Log)-8):])
	}
	checkNormalEnd(t, c)
}

// A restart of the whole game at every action of a normal_all run.
func TestCampaignGameRestartAtEveryTick(t *testing.T) {
	base := newChain(true, "normal", 1)
	camp := normalAll()
	base.run(t, camp, false, 3000)
	for at := 1; at < len(base.Log); at++ {
		restartRun(t, camp, at)
	}
}

// Regression: a restart after the Horadric Staff is placed (the adapter's pseudo item is gone, the quest word stays)
// made the planner skip Duriel and report Done in Act 2.
func TestCampaignRestartAfterStaffPlaced(t *testing.T) {
	base := newChain(true, "normal", 1)
	camp := normalAll()
	base.run(t, camp, false, 3000)
	n := 0
	for i, a := range base.Log {
		if a == "interact:tal-orifice" {
			n = i + 1
		}
	}
	if n == 0 {
		t.Fatal("no orifice action in the log")
	}
	for at := n; at <= n+3; at++ {
		restartRun(t, camp, at)
	}
}
