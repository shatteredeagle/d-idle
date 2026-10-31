package task_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/task"
)

// act3Game is a scripted Normal Act 3 world. Every rule is U until docs/act3.md live check A3-4 ran.
type act3Game struct {
	mu     sync.Mutex
	W      task.World
	Reader bool
	stage  map[string]task.Stage
	ground map[string]bool
	Log    []string
	Viol   []string
	Hired  int
	Smash  int
}

func newAct3(reader bool, level int) *act3Game {
	g := &act3Game{Reader: reader, stage: map[string]task.Stage{}, ground: map[string]bool{}}
	g.W = task.World{Act: 3, Difficulty: "normal", Area: task.KurastDocks, InTown: true, Level: level, Gold: 9000, Potions: 6, HasTP: true,
		WPs: map[task.Area]bool{task.KurastDocks: true}, Killed: map[string]bool{}, Items: map[string]int{task.ItemCube: 1}, Cleared: map[task.Area]bool{}}
	return g
}

func (g *act3Game) World(context.Context) (task.World, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	w := g.W
	w.WPs, w.Killed, w.Items, w.Cleared = map[task.Area]bool{}, map[string]bool{}, map[string]int{}, map[task.Area]bool{}
	for k, v := range g.W.WPs {
		w.WPs[k] = v
	}
	for k, v := range g.W.Killed {
		w.Killed[k] = v
	}
	for k, v := range g.W.Items {
		w.Items[k] = v
	}
	w.Quests = nil
	if g.Reader {
		w.Quests = map[string]task.QuestState{task.QEye: {Ev: task.EvMemory}}
		for q, s := range g.stage {
			w.Quests[q] = task.QuestState{Stage: s, Ev: task.EvMemory}
		}
	}
	return w, nil
}

func (g *act3Game) rec(k string) { g.Log = append(g.Log, k) }
func (g *act3Game) at(a task.Area) {
	g.W.Area = a
	g.W.InTown = a == task.KurastDocks || a == task.Fortress
}
func (g *act3Game) viol(f string, a ...any) { g.Viol = append(g.Viol, fmt.Sprintf(f, a...)) }

func (g *act3Game) Travel(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("travel:%d", a))
	if a == task.Durance3 && g.W.Items[task.ItemOrbSmashed] == 0 {
		g.viol("walked into Durance before the orb")
	}
	g.at(a)
	return nil
}
func (g *act3Game) ToTown(context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("town")
	g.at(task.KurastDocks)
	return nil
}
func (g *act3Game) Clear(context.Context, task.Area) error { return nil }
func (g *act3Game) KillBoss(_ context.Context, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("kill:" + name)
	rule := map[string]struct {
		a   task.Area
		lvl int
	}{task.BossCouncil: {task.Travincal, 24}, task.BossMephisto: {task.Durance3, 28}}
	r, ok := rule[name]
	if !ok || g.W.Area != r.a {
		return task.Errorf(task.KindNotFound, "%s is not here", name)
	}
	if g.W.Level < r.lvl {
		g.viol("killed %s at clvl %d (floor %d)", name, g.W.Level, r.lvl)
	}
	g.W.Killed[name] = true
	if name == task.BossCouncil {
		g.ground[task.ItemFlail] = true
	}
	g.W.Level++
	return nil
}
func (g *act3Game) Talk(_ context.Context, npc, pick string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("talk:" + npc + ":" + pick)
	if !g.W.InTown {
		g.viol("talk to %s in the field", npc)
		return task.Errorf(task.KindPrereq, "not in town")
	}
	switch {
	case npc == task.NPCMeshif && g.W.Items[task.ItemJade] > 0:
		g.W.Items[task.ItemJade] = 0
		g.W.Items[task.ItemGolden] = 1
	case npc == task.NPCAlkor && g.W.Items[task.ItemGolden] > 0:
		g.W.Items[task.ItemGolden] = 0
		g.stage[task.QGolden] = task.StageDone
	case npc == task.NPCAlkor && g.W.Items[task.ItemLamTome] > 0:
		g.W.Items[task.ItemLamTome] = 0
		g.stage[task.QLamEsen] = task.StageDone
	case npc == task.NPCHratli && g.W.Items[task.ItemGidbinnRaw] > 0:
		g.W.Items[task.ItemGidbinnRaw] = 0
		g.stage[task.QBlade] = task.StageDone
	}
	return nil
}
func (g *act3Game) Interact(_ context.Context, object string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("interact:" + object)
	chest := map[string]struct {
		a    task.Area
		item string
	}{
		"Khalim's Eye chest": {task.SpiderCavern, task.ItemEye}, "Khalim's Brain chest": {task.FlayerDungeon3, task.ItemBrain},
		"Khalim's Heart chest": {task.KurastSewers2, task.ItemHeart}, "gidbinn altar": {task.FlayerJungle, task.ItemGidbinnRaw},
		"Lam Esen altar": {task.RuinedTemple, task.ItemLamTome},
	}
	if c, ok := chest[object]; ok {
		if g.W.Area != c.a {
			return task.Errorf(task.KindNotFound, "%s not here", object)
		}
		g.ground[c.item] = true
		return nil
	}
	if object == "red portal" {
		if !g.W.Killed[task.BossMephisto] || g.W.Area != task.Durance3 {
			return task.Errorf(task.KindNotFound, "no portal")
		}
		g.W.Act = 4
		g.at(task.Fortress)
	}
	return nil
}
func (g *act3Game) PickUp(_ context.Context, item string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("pickup:" + item)
	if !g.ground[item] {
		return task.Errorf(task.KindNotFound, "no %s", item)
	}
	delete(g.ground, item)
	g.W.Items[item]++
	return nil
}
func (g *act3Game) ActivateWaypoint(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("wp:%d", a))
	g.W.WPs[a] = true
	return nil
}
func (g *act3Game) Chores(context.Context) error { g.W.Potions = 10; return nil }
func (g *act3Game) LevelTo(_ context.Context, n int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("level")
	if g.W.Level += 4; g.W.Level > n {
		g.W.Level = n
	}
	return nil
}
func (g *act3Game) Transmute(_ context.Context, in []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("transmute:" + strings.Join(in, "+"))
	if g.W.Items[task.ItemCube] == 0 {
		return task.Errorf(task.KindPrereq, "no cube")
	}
	for _, it := range in {
		if g.W.Items[it] == 0 {
			return task.Errorf(task.KindPrereq, "missing %s", it)
		}
		g.W.Items[it]--
	}
	g.W.Items[task.ItemWill] = 1
	return nil
}
func (g *act3Game) SmashOrb(_ context.Context, weapon string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("smash:" + weapon)
	if g.W.Area != task.Travincal || g.W.Items[weapon] == 0 {
		return task.Errorf(task.KindPrereq, "no orb or no weapon")
	}
	g.W.Items[task.ItemOrbSmashed] = 1
	g.Smash++
	return nil
}
func (g *act3Game) HireMerc(_ context.Context, act int, aura string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("hire:%d:%s", act, aura))
	g.W.Items[task.ItemMercenary]++
	g.Hired++
	return nil
}

func driveAct3(t *testing.T, g *act3Game, c *task.Campaign, rebuild bool) {
	t.Helper()
	ex := task.NewExecutor(g, g)
	for i := 0; i < 500; i++ {
		if rebuild {
			ex = task.NewExecutor(g, g)
		}
		r, err := ex.Tick(context.Background(), c)
		if err != nil {
			t.Fatalf("tick %d: %v", i, err)
		}
		if r.Outcome == task.OutDone {
			return
		}
		if r.Stuck {
			t.Fatalf("stuck at tick %d: %s; log %v", i, r.Attention, g.Log[max(0, len(g.Log)-10):])
		}
	}
	t.Fatalf("no end; log %v", g.Log[:min(40, len(g.Log))])
}

func checkAct3Run(t *testing.T, g *act3Game) {
	t.Helper()
	if len(g.Viol) > 0 {
		t.Fatalf("violations: %v", g.Viol)
	}
	if g.W.Act != 4 || g.W.Area != task.Fortress {
		t.Fatalf("ended in act %d area %d; tail %v", g.W.Act, g.W.Area, g.Log[max(0, len(g.Log)-8):])
	}
	tr := pos(g.Log, "transmute:"+strings.Join([]string{task.ItemFlail, task.ItemHeart, task.ItemEye, task.ItemBrain}, "+"))
	co, sm, me := pos(g.Log, "kill:"+task.BossCouncil), pos(g.Log, "smash:"+task.ItemWill), pos(g.Log, "kill:"+task.BossMephisto)
	if !(co >= 0 && co < tr && tr < sm && sm < me) {
		t.Fatalf("order council %d transmute %d smash %d mephisto %d", co, tr, sm, me)
	}
	for _, c := range []string{"Eye", "Brain", "Heart"} {
		if p := pos(g.Log, "interact:Khalim's "+c+" chest"); p < 0 || p > tr {
			t.Fatalf("%s chest at %d, transmute at %d", c, p, tr)
		}
	}
	if g.Smash != 1 {
		t.Fatalf("orb smashed %d times", g.Smash)
	}
}

func TestAct3CampaignWithReader(t *testing.T) {
	g := newAct3(true, 5)
	driveAct3(t, g, task.NormalAct3(task.Act3Options{Merc: true}), false)
	checkAct3Run(t, g)
	if g.Hired != 1 {
		t.Fatalf("hired %d", g.Hired)
	}
}

func TestAct3CampaignProxiesOnly(t *testing.T) {
	g := newAct3(false, 12)
	driveAct3(t, g, task.NormalAct3(task.Act3Options{}), false)
	checkAct3Run(t, g)
}

func TestAct3RestartAtEveryNode(t *testing.T) {
	g := newAct3(true, 20)
	driveAct3(t, g, task.NormalAct3(task.Act3Options{Merc: true}), true)
	checkAct3Run(t, g)
}

func TestAct3OptionalQuests(t *testing.T) {
	g := newAct3(true, 30)
	g.W.Items[task.ItemJade] = 1
	driveAct3(t, g, task.NormalAct3(task.Act3Options{GoldenBird: true, LamEsen: true, Blade: true}), false)
	checkAct3Run(t, g)
	for _, l := range []string{"talk:Alkor:golden bird", "talk:Alkor:lam esen", "talk:Hratli:blade"} {
		if pos(g.Log, l) < 0 {
			t.Errorf("missing %s", l)
		}
	}
}

func TestAct3MephistoAgainAfterRestartInAct3(t *testing.T) {
	// Orb smashed in an earlier game, quest word says Mephisto is done, but the portal is gone: the hop group fights again.
	g := newAct3(true, 30)
	g.stage[task.QGuardian] = task.StageDone
	g.stage[task.QKhalim] = task.StageDone
	g.W.Items[task.ItemOrbSmashed] = 1
	driveAct3(t, g, task.NormalAct3(task.Act3Options{}), false)
	if g.W.Act != 4 || pos(g.Log, "kill:"+task.BossMephisto) < 0 {
		t.Fatalf("act %d log %v", g.W.Act, g.Log)
	}
}

func TestAct3HandlersRegistered(t *testing.T) {
	seen := map[string]bool{}
	for _, q := range []string{task.QEye, task.QBrain, task.QHeart, task.QCouncil, task.QKhalim, task.QGuardian, task.QAct3Done, task.QGolden, task.QBlade, task.QLamEsen, task.QMercAct3} {
		ns := task.Handler(q)
		if len(ns) == 0 {
			t.Fatalf("no nodes for %s", q)
		}
		for _, n := range ns {
			if seen[n.Step.ID] {
				t.Errorf("duplicate step id %s", n.Step.ID)
			}
			seen[n.Step.ID] = true
			if n.Step.Need == nil || n.Step.Do == nil {
				t.Errorf("%s lacks Need or Do", n.Step.ID)
			}
			if !strings.HasPrefix(n.Step.ID, "quest."+q+"/") {
				t.Errorf("step id %q does not start with the quest group", n.Step.ID)
			}
		}
	}
}

func TestAct3FloorsAndChain(t *testing.T) {
	c := task.NormalAct3(task.Act3Options{})
	w, _ := newAct3(true, 10).World(context.Background())
	if p := c.Plan(w, nil); p.ID != "level.20" {
		t.Fatalf("clvl 10 plan %q", p.ID)
	}
	all := task.NormalActs123(task.Act1Options{}, task.Act2Options{}, task.Act3Options{})
	w.Act = 1
	w.Quests = nil
	if p := all.Plan(w, nil); !strings.HasPrefix(p.ID, "quest.den_of_evil") && !strings.HasPrefix(p.ID, "level.") {
		t.Fatalf("act 1 first: %q", p.ID)
	}
}

func TestAct3WillNeedsAllFourRelics(t *testing.T) {
	g := newAct3(true, 30)
	g.W.Items[task.ItemFlail], g.W.Items[task.ItemHeart], g.W.Items[task.ItemEye] = 1, 1, 1 // no Brain
	ex := task.NewExecutor(g, g)
	plan := task.Plan{ID: "quest." + task.QKhalim, Group: task.QKhalim, Nodes: task.Handler(task.QKhalim)}
	if _, err := ex.Tick(context.Background(), fixed{plan}); err != nil {
		t.Fatal(err)
	}
	if pos(g.Log, "transmute:"+task.ItemFlail+"+"+task.ItemHeart+"+"+task.ItemEye+"+"+task.ItemBrain) >= 0 {
		t.Fatalf("cubed without the Brain: %v", g.Log)
	}
}
