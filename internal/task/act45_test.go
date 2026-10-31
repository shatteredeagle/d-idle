package task_test

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/task"
)

// act45Game is a scripted Normal Act 4 + 5 world. Every rule is U until the live checks of docs/act4.md and docs/act5.md ran.
type act45Game struct {
	mu     sync.Mutex
	W      task.World
	Reader bool
	stage  map[string]task.Stage
	ground map[string]bool
	Log    []string
	Viol   []string
	Forge  int
	cages  int
}

func newAct45(reader bool, act int, level int) *act45Game {
	g := &act45Game{Reader: reader, stage: map[string]task.Stage{}, ground: map[string]bool{}}
	g.W = task.World{Act: act, Difficulty: "normal", Level: level, Gold: 50000, Potions: 8, HasTP: true,
		WPs: map[task.Area]bool{}, Killed: map[string]bool{}, Items: map[string]int{}, Cleared: map[task.Area]bool{}}
	if act == 4 {
		g.at(task.Fortress)
		g.W.Items[task.ItemSoulstone] = 1
	} else {
		g.at(task.Harrogath)
	}
	return g
}

func (g *act45Game) at(a task.Area) {
	g.W.Area = a
	g.W.InTown = a == task.Fortress || a == task.Harrogath
}

func (g *act45Game) World(context.Context) (task.World, error) {
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
		w.Quests = map[string]task.QuestState{}
		for q, s := range g.stage {
			w.Quests[q] = task.QuestState{Stage: s, Ev: task.EvMemory}
		}
	}
	return w, nil
}

func (g *act45Game) rec(k string)                           { g.Log = append(g.Log, k) }
func (g *act45Game) viol(f string, a ...any)                { g.Viol = append(g.Viol, fmt.Sprintf(f, a...)) }
func (g *act45Game) Clear(context.Context, task.Area) error { return nil }
func (g *act45Game) Chores(context.Context) error           { g.W.Potions = 10; return nil }

func (g *act45Game) Travel(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("travel:%d", a))
	if a == task.HallsOfPain && !g.W.WPs[a] && g.W.Area != task.NihlathakTemple && g.W.Area != task.HallsOfPain {
		g.viol("walked to the Halls of Pain without the portal or the waypoint")
	}
	if a == task.ThroneOfDestruction && !g.W.Killed[task.BossAncients] {
		g.viol("walked to the Throne before the Ancients")
	}
	g.at(a)
	return nil
}

func (g *act45Game) ToTown(context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("town")
	if g.W.Act >= 5 {
		g.at(task.Harrogath)
	} else {
		g.at(task.Fortress)
	}
	return nil
}

func (g *act45Game) KillBoss(_ context.Context, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("kill:" + name)
	rule := map[string]struct {
		a   task.Area
		lvl int
	}{
		task.BossIzual: {task.PlainsOfDespair, 30}, task.BossHephasto: {task.RiverOfFlame, 30}, task.BossDiablo: {task.ChaosSanctuary, 30},
		task.BossShenk: {task.BloodyFoothills, 25}, task.BossNihlathak: {task.HallsOfVaught, 30}, task.BossAncients: {task.ArreatSummit, 36},
		task.BossBaal: {task.ThroneOfDestruction, 38},
	}
	r, ok := rule[name]
	if !ok || g.W.Area != r.a {
		return task.Errorf(task.KindNotFound, "%s is not here", name)
	}
	if g.W.Level < r.lvl {
		g.viol("killed %s at clvl %d (floor %d)", name, g.W.Level, r.lvl)
	}
	if name == task.BossAncients && g.W.NoTownPortal != true && !g.ancientsConstrained() {
		g.viol("the Ancients step ran without NoTownPortal")
	}
	g.W.Killed[name] = true
	switch name {
	case task.BossHephasto:
		g.ground[task.ItemHammer] = true
	case task.BossBaal:
		g.at(task.WorldstoneChamber)
	}
	g.W.Level++
	return nil
}

// ancientsConstrained looks the step up in the handler: the constraint is data, so the test reads it from there.
func (g *act45Game) ancientsConstrained() bool {
	for _, n := range task.Handler(task.QAncients) {
		if strings.HasSuffix(n.Step.ID, "/kill") {
			return n.Step.Constraints.NoTownPortal && n.Step.Constraints.NoLeaveArea
		}
	}
	return false
}

func (g *act45Game) Talk(_ context.Context, npc, pick string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("talk:" + npc + ":" + pick)
	if !g.W.InTown {
		g.viol("talk to %s in the field", npc)
		return task.Errorf(task.KindPrereq, "not in town")
	}
	switch {
	case npc == task.NPCTyrael && pick == "fallen angel":
		g.stage[task.QIzual] = task.StageDone
	case npc == task.NPCTyrael && pick == "diablo":
		g.stage[task.QDiablo] = task.StageDone
	case npc == task.NPCTyrael && pick == "harrogath":
		if !g.W.Killed[task.BossDiablo] && g.stage[task.QDiablo] < task.StageCleared {
			g.viol("hopped before Diablo")
		}
		g.W.Act = 5
		g.at(task.Harrogath)
	case npc == task.NPCLarzuk:
		g.stage[task.QSiege] = task.StageDone
	case npc == task.NPCMalah && pick == "potion" && g.W.Items[task.ItemAnyaTouched] > 0:
		g.W.Items[task.ItemMalahPotion] = 1
	case npc == task.NPCMalah && pick == "scroll" && g.W.Items[task.ItemAnyaThawed] > 0:
		g.W.Items[task.ItemResScroll] = 1
	case npc == task.NPCQualKehk && pick == "rite of passage":
		g.stage[task.QAncients] = task.StageDone
	case npc == task.NPCQualKehk && pick == "rescue":
		g.stage[task.QRescue] = task.StageDone
	case npc == task.NPCAnya:
		g.stage[task.QBetrayal] = task.StageDone
	}
	return nil
}

func (g *act45Game) Interact(_ context.Context, object string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("interact:" + object)
	switch object {
	case "Frozen Anya":
		if g.W.Area != task.FrozenRiver {
			return task.Errorf(task.KindNotFound, "no Anya here")
		}
		if g.W.Items[task.ItemMalahPotion] > 0 {
			g.W.Items[task.ItemMalahPotion] = 0
			g.W.Items[task.ItemAnyaThawed] = 1
		} else {
			g.W.Items[task.ItemAnyaTouched] = 1
		}
	case "Anya's portal":
		if !g.W.InTown || g.W.Items[task.ItemScrollRead] == 0 {
			return task.Errorf(task.KindNotFound, "no portal")
		}
		g.at(task.NihlathakTemple)
	case "cage":
		if g.W.Area != task.FrigidHighlands {
			return task.Errorf(task.KindNotFound, "no cage")
		}
		if g.cages++; g.cages >= 3 {
			g.stage[task.QRescue] = task.StageCleared
		}
	}
	return nil
}

func (g *act45Game) PickUp(_ context.Context, item string) error {
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

func (g *act45Game) ActivateWaypoint(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("wp:%d", a))
	g.W.WPs[a] = true
	return nil
}

func (g *act45Game) LevelTo(_ context.Context, n int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("level")
	if g.W.Level += 4; g.W.Level > n {
		g.W.Level = n
	}
	return nil
}

func (g *act45Game) UseForge(_ context.Context, stone, hammer string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("forge")
	if g.W.Area != task.RiverOfFlame || g.W.Items[stone] == 0 || g.W.Items[hammer] == 0 {
		return task.Errorf(task.KindPrereq, "forge needs the stone and the hammer")
	}
	g.W.Items[stone], g.W.Items[hammer] = 0, 0
	g.W.Items[task.ItemForgeUsed] = 1
	g.stage[task.QHellforge] = task.StageDone
	g.Forge++
	return nil
}

func (g *act45Game) UseItem(_ context.Context, item string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec("use:" + item)
	if g.W.Items[item] == 0 {
		return task.Errorf(task.KindNotFound, "no %s", item)
	}
	g.W.Items[item] = 0
	g.W.Items[task.ItemScrollRead] = 1
	return nil
}

func (g *act45Game) HireMerc(_ context.Context, act int, aura string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.rec(fmt.Sprintf("hire:%d:%s", act, aura))
	g.W.Items[task.ItemMercenary]++
	return nil
}

func driveAct45(t *testing.T, g *act45Game, c *task.Campaign, rebuild bool) {
	t.Helper()
	ex := task.NewExecutor(g, g)
	for i := 0; i < 600; i++ {
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
	t.Fatalf("no end; log %v", g.Log[:min(60, len(g.Log))])
}

func checkAct4(t *testing.T, g *act45Game) {
	t.Helper()
	if len(g.Viol) > 0 {
		t.Fatalf("violations: %v", g.Viol)
	}
	if g.W.Act != 5 || g.W.Area != task.Harrogath {
		t.Fatalf("ended in act %d area %d; tail %v", g.W.Act, g.W.Area, g.Log[max(0, len(g.Log)-8):])
	}
	iz, di := pos(g.Log, "kill:"+task.BossIzual), pos(g.Log, "kill:"+task.BossDiablo)
	if !(iz >= 0 && iz < di) {
		t.Fatalf("izual %d diablo %d", iz, di)
	}
}

func TestAct4CampaignWithReader(t *testing.T) {
	g := newAct45(true, 4, 5)
	driveAct45(t, g, task.NormalAct4(task.Act4Options{}), false)
	checkAct4(t, g)
	if pos(g.Log, "talk:Tyrael:fallen angel") < 0 {
		t.Fatal("Izual reward never claimed")
	}
}

func TestAct4CampaignProxiesOnly(t *testing.T) {
	g := newAct45(false, 4, 30)
	driveAct45(t, g, task.NormalAct4(task.Act4Options{}), false)
	checkAct4(t, g)
}

func TestAct4RestartAtEveryNode(t *testing.T) {
	g := newAct45(true, 4, 12)
	driveAct45(t, g, task.NormalAct4(task.Act4Options{Hellforge: true}), true)
	checkAct4(t, g)
}

func TestAct4HellforgeNeedsTheSoulstone(t *testing.T) {
	g := newAct45(true, 4, 30)
	g.W.Items[task.ItemSoulstone] = 0
	driveAct45(t, g, task.NormalAct4(task.Act4Options{Hellforge: true}), false)
	checkAct4(t, g)
	if pos(g.Log, "kill:"+task.BossHephasto) >= 0 || g.Forge != 0 {
		t.Fatalf("Hellforge ran without a Soulstone: %v", g.Log)
	}
}

func TestAct4HellforgeWithSoulstone(t *testing.T) {
	g := newAct45(true, 4, 30)
	driveAct45(t, g, task.NormalAct4(task.Act4Options{Hellforge: true, NoIzual: true}), false)
	if g.W.Act != 5 || len(g.Viol) > 0 {
		t.Fatalf("act %d viol %v", g.W.Act, g.Viol)
	}
	if g.Forge != 1 || pos(g.Log, "pickup:"+task.ItemHammer) < 0 {
		t.Fatalf("forge %d log %v", g.Forge, g.Log)
	}
}

func TestAct4NoTyraelHopBeforeDiablo(t *testing.T) {
	g := newAct45(true, 4, 30)
	g.stage[task.QIzual] = task.StageDone
	ex := task.NewExecutor(g, g)
	plan := task.Plan{ID: "quest." + task.QAct4Done, Group: task.QAct4Done, Nodes: task.Handler(task.QAct4Done)}
	if _, err := ex.Tick(context.Background(), fixed{plan}); err != nil {
		t.Fatal(err)
	}
	if g.W.Act != 4 || pos(g.Log, "talk:Tyrael:harrogath") >= 0 {
		t.Fatalf("hopped without Diablo: %v", g.Log)
	}
}

func checkAct5(t *testing.T, g *act45Game) {
	t.Helper()
	if len(g.Viol) > 0 {
		t.Fatalf("violations: %v", g.Viol)
	}
	an, ba := pos(g.Log, "kill:"+task.BossAncients), pos(g.Log, "kill:"+task.BossBaal)
	if !(an >= 0 && an < ba) {
		t.Fatalf("ancients %d baal %d; log %v", an, ba, g.Log)
	}
	if g.W.Area != task.WorldstoneChamber {
		t.Fatalf("ended in area %d", g.W.Area)
	}
}

func TestAct5CampaignWithReader(t *testing.T) {
	g := newAct45(true, 5, 10)
	driveAct45(t, g, task.NormalAct5(task.Act5Options{Merc: true}), false)
	checkAct5(t, g)
	if pos(g.Log, "kill:"+task.BossShenk) < 0 || pos(g.Log, "talk:Larzuk:sockets") < 0 {
		t.Fatalf("siege missing: %v", g.Log)
	}
	if pos(g.Log, "hire:5:Bash") < 0 {
		t.Fatal("no hire")
	}
}

func TestAct5CampaignProxiesOnly(t *testing.T) {
	g := newAct45(false, 5, 30)
	driveAct45(t, g, task.NormalAct5(task.Act5Options{}), false)
	checkAct5(t, g)
}

func TestAct5RestartAtEveryNode(t *testing.T) {
	g := newAct45(true, 5, 22)
	driveAct45(t, g, task.NormalAct5(task.Act5Options{Prison: true}), true)
	checkAct5(t, g)
}

func TestAct5PrisonOfIceFlow(t *testing.T) {
	g := newAct45(true, 5, 30)
	driveAct45(t, g, task.NormalAct5(task.Act5Options{Prison: true, Rescue: true}), false)
	checkAct5(t, g)
	want := []string{"interact:Frozen Anya", "talk:Malah:potion", "interact:Frozen Anya", "talk:Malah:scroll", "use:" + task.ItemResScroll}
	last := -1
	for _, w := range want {
		p := pos(g.Log[last+1:], w)
		if p < 0 {
			t.Fatalf("flow broke at %q: %v", w, g.Log)
		}
		last += 1 + p
	}
	if pos(g.Log, "interact:cage") < 0 {
		t.Fatal("rescue never opened a cage")
	}
}

func TestAct5NihlathakThroughAnyasPortal(t *testing.T) {
	g := newAct45(true, 5, 38)
	driveAct45(t, g, task.NormalAct5(task.Act5Options{Nihlathak: true}), false)
	checkAct5(t, g)
	pi, po, ni := pos(g.Log, "use:"+task.ItemResScroll), pos(g.Log, "interact:Anya's portal"), pos(g.Log, "kill:"+task.BossNihlathak)
	if !(pi >= 0 && pi < po && po < ni) {
		t.Fatalf("scroll %d portal %d nihlathak %d", pi, po, ni)
	}
}

func TestAct5AncientsStepCarriesTheConstraints(t *testing.T) {
	found := false
	for _, n := range task.Handler(task.QAncients) {
		if strings.HasSuffix(n.Step.ID, "/kill") {
			found = true
			c := n.Step.Constraints
			if !c.NoTownPortal || !c.NoLeaveArea {
				t.Fatalf("constraints %+v", c)
			}
		}
	}
	if !found {
		t.Fatal("no kill step")
	}
}

func TestAct5BaalKillSpansThroneAndChamber(t *testing.T) {
	g := newAct45(true, 5, 40)
	g.W.Killed[task.BossAncients] = true
	g.stage[task.QAncients] = task.StageDone
	g.at(task.WorldstoneChamber)
	ex := task.NewExecutor(g, g)
	plan := task.Plan{ID: "quest." + task.QBaal, Group: task.QBaal, Nodes: task.Handler(task.QBaal)}
	if _, err := ex.Tick(context.Background(), fixed{plan}); err != nil {
		t.Fatal(err)
	}
	for _, n := range plan.Nodes {
		if strings.HasSuffix(n.Step.ID, "/kill") {
			w, _ := g.World(context.Background())
			if need, _ := n.Step.Need(w); !need && !g.W.Killed[task.BossBaal] {
				t.Fatal("a restart inside the chamber must still need the kill")
			}
		}
	}
}

func TestAct45HandlersRegistered(t *testing.T) {
	seen := map[string]bool{}
	for _, q := range []string{task.QIzual, task.QHellforge, task.QDiablo, task.QAct4Done, task.QSiege, task.QRescue, task.QPrison,
		task.QBetrayal, task.QAncients, task.QBaal, task.QMercAct5} {
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

func TestAct45FloorsAndChain(t *testing.T) {
	w, _ := newAct45(true, 4, 10).World(context.Background())
	if p := task.NormalAct4(task.Act4Options{}).Plan(w, nil); p.ID != "level.30" {
		t.Fatalf("act 4 at clvl 10: %q", p.ID)
	}
	w5, _ := newAct45(true, 5, 10).World(context.Background())
	if p := task.NormalAct5(task.Act5Options{}).Plan(w5, nil); p.ID != "level.25" {
		t.Fatalf("act 5 at clvl 10: %q", p.ID)
	}
	all := task.NormalActs12345(task.Act1Options{}, task.Act2Options{}, task.Act3Options{}, task.Act4Options{}, task.Act5Options{})
	last := all.Entries[len(all.Entries)-1]
	if last.Quest != task.QBaal || all.Name != "normal_act1_5" {
		t.Fatalf("last %+v name %s", last, all.Name)
	}
	w.Act, w.Quests = 1, nil
	if p := all.Plan(w, nil); !strings.HasPrefix(p.ID, "quest.den_of_evil") && !strings.HasPrefix(p.ID, "level.") {
		t.Fatalf("act 1 first: %q", p.ID)
	}
	seen := map[string]int{}
	for _, e := range all.Entries {
		seen[e.Quest]++
		if seen[e.Quest] > 1 {
			t.Errorf("quest %s twice in the campaign", e.Quest)
		}
	}
}
