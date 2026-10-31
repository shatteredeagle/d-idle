package task_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/task"
)

// act2Game is a scripted Normal Act 2 world with the rules the design assumes (every one is U until the live checks of
// docs/act2.md): it implements Env, Actions, Transmuter and MercHirer.
type act2Game struct {
	mu     sync.Mutex
	W      task.World
	Reader bool
	stage  map[string]task.Stage
	ground map[string]bool
	Log    []string
	Viol   []string
	Placed int
	Hired  int
}

func newAct2(reader bool, level int) *act2Game {
	g := &act2Game{Reader: reader, stage: map[string]task.Stage{}, ground: map[string]bool{}}
	g.W = task.World{Act: 2, Difficulty: "normal", Area: task.LutGholein, InTown: true, Level: level, Gold: 5000, Potions: 6, HasTP: true,
		WPs: map[task.Area]bool{task.LutGholein: true}, Killed: map[string]bool{}, Items: map[string]int{}, Cleared: map[task.Area]bool{}}
	return g
}

func (g *act2Game) World(context.Context) (task.World, error) {
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
		w.Quests = map[string]task.QuestState{task.QRadament: {Ev: task.EvMemory}}
		for q, s := range g.stage {
			w.Quests[q] = task.QuestState{Stage: s, Ev: task.EvMemory}
		}
	}
	return w, nil
}

func (g *act2Game) rec(k string) { g.Log = append(g.Log, k) }
func (g *act2Game) set(q string, s task.Stage) {
	if g.stage[q] < s {
		g.stage[q] = s
	}
}
func (g *act2Game) at(a task.Area) {
	g.W.Area, g.W.InTown = a, a == task.LutGholein || a == task.KurastDocks
}

func (g *act2Game) Travel(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("travel:%d", a))
	g.at(a)
	if g.W.InTown {
		g.W.WPs[a] = true
	}
	return nil
}
func (g *act2Game) ToTown(context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("town")
	g.at(task.LutGholein)
	return nil
}
func (g *act2Game) Clear(context.Context, task.Area) error { return nil }
func (g *act2Game) KillBoss(_ context.Context, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("kill:" + name)
	rule := map[string]struct {
		a   task.Area
		lvl int
	}{task.BossRadament: {task.Sewers3, 16}, task.BossFangskin: {task.ClawViper2, 18}, task.BossSummoner: {task.ArcaneSanctuary, 24}, task.BossDuriel: {task.DurielsLair, 24}}
	r, ok := rule[name]
	if !ok || g.W.Area != r.a {
		return task.Errorf(task.KindNotFound, "%s is not here", name)
	}
	if g.W.Level < r.lvl {
		g.Viol = append(g.Viol, fmt.Sprintf("killed %s at clvl %d (floor %d)", name, g.W.Level, r.lvl))
	}
	if name == task.BossDuriel && g.W.Items[task.ItemStaffPlaced] == 0 {
		g.Viol = append(g.Viol, "Duriel fought without the staff placed")
	}
	g.W.Killed[name] = true
	switch name {
	case task.BossRadament:
		g.ground[task.ItemBook] = true
		g.set(task.QRadament, task.StageCleared)
	case task.BossDuriel:
		g.set(task.QDuriel, task.StageCleared)
	case task.BossSummoner:
		g.set(task.QSummoner, task.StageCleared)
	}
	g.W.Level++
	return nil
}
func (g *act2Game) Talk(_ context.Context, npc, pick string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("talk:" + npc + ":" + pick)
	if !g.W.InTown && npc != task.NPCTyrael {
		g.Viol = append(g.Viol, "talk to "+npc+" in the field")
		return task.Errorf(task.KindPrereq, "not in town")
	}
	switch {
	case npc == task.NPCAtma && pick == "quest":
		g.set(task.QRadament, task.StageStarted)
	case npc == task.NPCAtma && pick == "reward" && g.stage[task.QRadament] >= task.StageCleared:
		g.stage[task.QRadament] = task.StageDone
		g.W.Level++ // the skill point
	case npc == task.NPCCain && g.W.Items[task.ItemScroll] > 0:
		g.W.Items[task.ItemScroll] = 0
		g.set(task.QStaff, task.StageStarted)
	case npc == task.NPCTyrael && g.stage[task.QDuriel] >= task.StageCleared:
		g.stage[task.QDuriel] = task.StageDone
	case (npc == task.NPCDrognan || npc == task.NPCJerhyn) && pick == "arcane":
		g.set(task.QArcane, task.StageStarted)
	case npc == task.NPCJerhyn && pick == "act2":
		g.stage[task.QAct2Word] = task.StageDone
	case npc == task.NPCMeshif && pick == "sail east":
		if g.stage[task.QDuriel] < task.StageDone {
			g.Viol = append(g.Viol, "sailed before Tyrael")
			return task.Errorf(task.KindNotFound, "no sail row")
		}
		g.W.Act = 3
		g.at(task.KurastDocks)
	}
	return nil
}
func (g *act2Game) Interact(_ context.Context, object string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("interact:" + object)
	chest := map[string]struct {
		a    task.Area
		item string
	}{
		"Horadric Scroll chest": {task.Sewers3, task.ItemScroll}, "Horadric Cube chest": {task.HallsOfDead3, task.ItemCube},
		"taintedsunaltar": {task.ClawViper2, task.ItemStaffTop}, "Staff of Kings chest": {task.MaggotLair3, task.ItemShaft},
	}
	if c, ok := chest[object]; ok {
		if g.W.Area != c.a {
			return task.Errorf(task.KindNotFound, "%s not here", object)
		}
		g.ground[c.item] = true
		return nil
	}
	switch object {
	case "item:" + task.ItemBook:
		g.W.Items[task.ItemBook]--
		g.W.Items[task.ItemBookRead] = 1
		g.W.Level++
	case "tal-orifice":
		if g.W.Area != task.CanyonOfMagi || g.W.Items[task.ItemStaff] == 0 {
			return task.Errorf(task.KindNotFound, "no orifice")
		}
		g.W.Items[task.ItemStaff] = 0
		g.W.Items[task.ItemStaffPlaced] = 1
		g.set(task.QStaff, task.StageDone)
		g.Placed++
	case "duriel-door":
		// the door is the game's quest state, not the adapter's per-game pseudo item
		if g.W.Items[task.ItemStaffPlaced] == 0 && g.stage[task.QStaff] < task.StageDone {
			return task.Errorf(task.KindNotFound, "door shut")
		}
		g.at(task.DurielsLair)
	case "Arcane Sanctuary portal":
		g.at(task.ArcaneSanctuary)
	case "Summoner's journal":
		g.set(task.QSummoner, task.StageDone)
	}
	return nil
}
func (g *act2Game) PickUp(_ context.Context, item string) error {
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
func (g *act2Game) ActivateWaypoint(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("wp:%d", a))
	g.W.WPs[a] = true
	return nil
}
func (g *act2Game) Chores(context.Context) error { g.W.Potions = 10; return nil }
func (g *act2Game) LevelTo(_ context.Context, n int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("level")
	if g.W.Level += 3; g.W.Level > n {
		g.W.Level = n
	}
	return nil
}
func (g *act2Game) Transmute(_ context.Context, in []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("transmute:" + strings.Join(in, "+"))
	if g.W.Items[task.ItemCube] == 0 || g.W.Items[task.ItemShaft] == 0 || g.W.Items[task.ItemStaffTop] == 0 {
		return task.Errorf(task.KindPrereq, "cube or inputs missing")
	}
	g.W.Items[task.ItemShaft], g.W.Items[task.ItemStaffTop] = 0, 0
	g.W.Items[task.ItemStaff] = 1
	return nil
}
func (g *act2Game) HireMerc(_ context.Context, act int, aura string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("hire:%d:%s", act, aura))
	g.W.Items[task.ItemMercenary]++
	g.W.Gold -= 360
	g.Hired++
	return nil
}

func driveAct2(t *testing.T, g *act2Game, c *task.Campaign, rebuild bool) {
	t.Helper()
	ex := task.NewExecutor(g, g)
	for i := 0; i < 400; i++ {
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
	t.Fatalf("no end; log %v", g.Log[max(0, len(g.Log)-12):])
}

func pos(log []string, s string) int {
	for i, l := range log {
		if l == s {
			return i
		}
	}
	return -1
}

func checkAct2Run(t *testing.T, g *act2Game, wantSummoner bool) {
	t.Helper()
	if len(g.Viol) > 0 {
		t.Fatalf("violations: %v", g.Viol)
	}
	if g.W.Act != 3 || g.W.Area != task.KurastDocks {
		t.Fatalf("ended in act %d area %d; log tail %v", g.W.Act, g.W.Area, g.Log[max(0, len(g.Log)-8):])
	}
	if g.Placed != 1 || g.Hired != 1 {
		t.Fatalf("placed %d hired %d", g.Placed, g.Hired)
	}
	tr, or, du := pos(g.Log, "transmute:"+task.ItemShaft+"+"+task.ItemStaffTop), pos(g.Log, "interact:tal-orifice"), pos(g.Log, "kill:"+task.BossDuriel)
	if tr < 0 || !(tr < or && or < du) {
		t.Fatalf("order transmute %d orifice %d duriel %d", tr, or, du)
	}
	if rd := pos(g.Log, "kill:"+task.BossRadament); rd < 0 || rd > tr {
		t.Fatalf("Radament (%d) must come before the assembly (%d)", rd, tr)
	}
	if got := pos(g.Log, "kill:"+task.BossSummoner) >= 0; got != wantSummoner {
		t.Fatalf("summoner killed = %v, want %v", got, wantSummoner)
	}
	if pos(g.Log, "interact:item:"+task.ItemBook) < 0 {
		t.Fatal("Book of Skill never read")
	}
}

func TestAct2CampaignWithReader(t *testing.T) {
	g := newAct2(true, 1)
	driveAct2(t, g, task.NormalAct2(task.Act2Options{Merc: true, Arcane: true}), false)
	checkAct2Run(t, g, true)
}

func TestAct2CampaignProxiesOnly(t *testing.T) {
	g := newAct2(false, 10)
	driveAct2(t, g, task.NormalAct2(task.Act2Options{Merc: true}), false)
	checkAct2Run(t, g, false)
}

func TestAct2RestartAtEveryNode(t *testing.T) {
	g := newAct2(true, 12)
	driveAct2(t, g, task.NormalAct2(task.Act2Options{Merc: true}), true)
	checkAct2Run(t, g, false)
}

func TestAct2HandlersRegistered(t *testing.T) {
	seen := map[string]bool{}
	for _, q := range []string{task.QRadament, task.QCube, task.QViper, task.QShaft, task.QStaff, task.QArcane, task.QSummoner, task.QDuriel, task.QAct2Done, task.QMercAct2} {
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

func TestAct2PlanOrderAndLevelFloors(t *testing.T) {
	c := task.NormalAct2(task.Act2Options{})
	w, _ := newAct2(true, 10).World(context.Background())
	if p := c.Plan(w, nil); p.ID != "level.16" {
		t.Fatalf("clvl 10 plan %q", p.ID)
	}
	w, _ = newAct2(true, 30).World(context.Background())
	if p := c.Plan(w, nil); p.Group != task.QRadament {
		t.Fatalf("first quest %q", p.Group)
	}
	w.Quests[task.QRadament] = task.QuestState{Stage: task.StageDone, Ev: task.EvMemory}
	if p := c.Plan(w, nil); p.Group != task.QCube {
		t.Fatalf("after Radament: %q", p.Group)
	}
	// With Act 1 prepended nothing in Act 2 plans before Act 1 is complete.
	all := task.NormalActs12(task.Act1Options{}, task.Act2Options{})
	w1 := w
	w1.Act = 1
	w1.Quests = nil
	if p := all.Plan(w1, nil); !strings.HasPrefix(p.ID, "quest.den_of_evil") {
		t.Fatalf("act 1 first: %q", p.ID)
	}
}

func TestAct2StaffAssemblyNeedsCubeInPack(t *testing.T) {
	g := newAct2(true, 30)
	g.W.Items[task.ItemShaft], g.W.Items[task.ItemStaffTop] = 1, 1 // no Cube
	ex := task.NewExecutor(g, g)
	plan := task.Plan{ID: "quest." + task.QStaff, Group: task.QStaff, Nodes: task.Handler(task.QStaff)}
	r, err := ex.Tick(context.Background(), fixed{plan})
	if err != nil {
		t.Fatal(err)
	}
	if r.Outcome == task.OutRan || g.Placed != 0 || pos(g.Log, "transmute:"+task.ItemShaft+"+"+task.ItemStaffTop) >= 0 {
		t.Fatalf("assembled without a cube: %+v log %v", r, g.Log)
	}
}

type fixed struct{ p task.Plan }

func (f fixed) Plan(task.World, task.Session) task.Plan { return f.p }
