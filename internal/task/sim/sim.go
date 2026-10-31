// Package sim is a fake Act 1 world for the task layer: it implements task.Env and task.Actions with simple rules
// (levels, quest stages, bosses, items, a town) so whole campaigns run offline. It is a model of the rules the
// design assumes (docs/task.md), not of the game: every rule here is U until verified live.
package sim

import (
	"context"
	"fmt"
	"sync"

	"d2lod-bot/internal/env"
	"d2lod-bot/internal/task"
)

// Game is the fake world. Zero value is not usable; call New.
type Game struct {
	mu sync.Mutex

	W         task.World
	Reader    bool                  // true: quest stages are visible (the T2 reader exists)
	stage     map[string]task.Stage // true quest bits
	ground    map[string]bool       // items lying around (dropped by bosses)
	Log       []string              // every action, in order
	Fail      map[string][]error    // "kill:Andariel" -> errors to return, one per call
	DieOn     map[string]int        // action key -> times to "die" (returns ErrDied after teleporting to town)
	Calls     map[string]int
	Violation []string // rule breaks the fake noticed (level gate ignored, talk in the field ...)
	XPStep    int      // levels gained per LevelTo call (default 3)
}

// ErrDied is what an action returns when the fake character dies; adapters would wrap it with task.KindDeath.
var ErrDied = task.Errorf(task.KindDeath, "character died")

// New returns a fresh clvl 1 character in the Rogue Encampment, Normal Act 1, no waypoints but town.
func New(reader bool) *Game {
	g := &Game{Reader: reader, stage: map[string]task.Stage{}, ground: map[string]bool{},
		Fail: map[string][]error{}, DieOn: map[string]int{}, Calls: map[string]int{}, XPStep: 3}
	g.W = task.World{Act: 1, Difficulty: "normal", Area: task.RogueEncampment, InTown: true, Level: 1, Gold: 500, Potions: 6, Antidotes: 2, HasTP: true,
		WPs: map[task.Area]bool{task.RogueEncampment: true}, Killed: map[string]bool{}, Items: map[string]int{}, Cleared: map[task.Area]bool{}}
	return g
}

// SetLevel is for tests that start mid-way.
func (g *Game) SetLevel(n int) { g.W.Level = n }

// World implements task.Env.
func (g *Game) World(context.Context) (task.World, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	w := g.W
	w.WPs, w.Killed, w.Items, w.Cleared = cpB(g.W.WPs), cpS(g.W.Killed), cpI(g.W.Items), cpA(g.W.Cleared)
	w.Quests = nil
	if g.Reader {
		w.Quests = map[string]task.QuestState{}
		for q, s := range g.stage {
			w.Quests[q] = task.QuestState{Stage: s, Ev: task.EvMemory}
		}
		if len(w.Quests) == 0 {
			w.Quests[task.QDen] = task.QuestState{Ev: task.EvMemory}
		}
	}
	return w, nil
}

func cpB(m map[task.Area]bool) map[task.Area]bool {
	o := map[task.Area]bool{}
	for k, v := range m {
		o[k] = v
	}
	return o
}
func cpS(m map[string]bool) map[string]bool {
	o := map[string]bool{}
	for k, v := range m {
		o[k] = v
	}
	return o
}
func cpI(m map[string]int) map[string]int {
	o := map[string]int{}
	for k, v := range m {
		o[k] = v
	}
	return o
}
func cpA(m map[task.Area]bool) map[task.Area]bool { return cpB(m) }

// begin records the call, applies injected failures and deaths. Returns a non-nil error to abort the action.
func (g *Game) begin(key string) error {
	g.Log = append(g.Log, key)
	g.Calls[key]++
	if n := g.DieOn[key]; n > 0 {
		g.DieOn[key]--
		g.W.Area, g.W.InTown = task.RogueEncampment, true
		return ErrDied
	}
	if errs := g.Fail[key]; len(errs) > 0 {
		g.Fail[key] = errs[1:]
		return errs[0]
	}
	return nil
}

func (g *Game) moveTo(a task.Area) {
	g.W.Area = a
	g.W.InTown = a == task.RogueEncampment || a == task.LutGholein
	if g.W.InTown {
		g.W.WPs[a] = true
	}
}

func (g *Game) set(q string, s task.Stage) {
	if g.stage[q] < s {
		g.stage[q] = s
	}
}

// Travel implements task.Actions.
func (g *Game) Travel(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin(fmt.Sprintf("travel:%d", a)); err != nil {
		return err
	}
	if a == task.Tristram && g.W.Items[task.ItemPortal] == 0 {
		return task.Errorf(task.KindNotFound, "no portal to Tristram")
	}
	g.moveTo(a)
	return nil
}

// ToTown implements task.Actions.
func (g *Game) ToTown(context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin("town"); err != nil {
		return err
	}
	g.moveTo(task.RogueEncampment)
	return nil
}

// Clear implements task.Actions.
func (g *Game) Clear(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin(fmt.Sprintf("clear:%d", a)); err != nil {
		return err
	}
	if g.W.Area != a {
		g.Violation = append(g.Violation, fmt.Sprintf("clear %d while in %d", a, g.W.Area))
		return task.Errorf(task.KindPrereq, "not in area")
	}
	g.W.Cleared[a] = true
	if a == task.DenOfEvil {
		g.set(task.QDen, task.StageCleared)
	}
	return nil
}

// KillBoss implements task.Actions.
func (g *Game) KillBoss(_ context.Context, name string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin("kill:" + name); err != nil {
		return err
	}
	need := map[string]struct {
		area task.Area
		lvl  int
	}{task.BossBR: {task.BurialGrounds, 6}, task.BossSmith: {task.Barracks, 8}, task.BossCount: {task.TowerCellar5, 14}, task.BossAndy: {task.Catacombs4, env.DefaultAndarielMinLevel}}
	n, ok := need[name]
	if !ok || g.W.Area != n.area {
		return task.Errorf(task.KindNotFound, "%s not in this area", name)
	}
	if g.W.Level < n.lvl {
		g.Violation = append(g.Violation, fmt.Sprintf("killed %s at clvl %d (floor %d)", name, g.W.Level, n.lvl))
	}
	g.W.Killed[name] = true
	switch name {
	case task.BossBR:
		g.set(task.QBurial, task.StageCleared)
	case task.BossSmith:
		g.ground[task.ItemMalus] = true
		g.set(task.QTools, task.StageStarted)
	case task.BossCount:
		g.set(task.QTower, task.StageDone)
	case task.BossAndy:
		g.set(task.QAndariel, task.StageCleared)
	}
	g.W.Level++ // bosses give a level's worth of fun in the fake
	return nil
}

// Talk implements task.Actions.
func (g *Game) Talk(_ context.Context, npc, pick string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin("talk:" + npc + ":" + pick); err != nil {
		return err
	}
	if !g.W.InTown {
		g.Violation = append(g.Violation, "talk to "+npc+" in the field")
		return task.Errorf(task.KindPrereq, "not in town")
	}
	switch {
	case npc == task.NPCAkara && pick == "quest":
		g.set(task.QDen, task.StageStarted)
	case npc == task.NPCAkara && pick == "reward" && g.stage[task.QDen] == task.StageCleared:
		g.stage[task.QDen] = task.StageDone
		g.W.Level++ // skill point stands in for a reward
	case npc == task.NPCKashya && pick == "reward" && g.stage[task.QBurial] == task.StageCleared:
		g.stage[task.QBurial] = task.StageDone
	case npc == task.NPCCharsi && pick == "quest":
		g.set(task.QTools, task.StageStarted)
	case npc == task.NPCCharsi && g.W.Items[task.ItemMalus] > 0:
		g.W.Items[task.ItemMalus] = 0
		g.stage[task.QTools] = task.StageDone
	case npc == task.NPCWarriv && g.stage[task.QAndariel] == task.StageCleared:
		g.stage[task.QAndariel], g.stage[task.QAct1Done] = task.StageDone, task.StageDone
		g.W.Act = 2
		g.W.Area, g.W.InTown = task.LutGholein, true
	case npc == task.NPCAkara && pick == "bark" && g.W.Items[task.ItemBark] > 0:
		g.W.Items[task.ItemBark] = 0
		g.W.Items[task.ItemCainSc] = 1
		g.set(task.QCain, task.StageStarted)
	}
	return nil
}

// Interact implements task.Actions.
func (g *Game) Interact(_ context.Context, object string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin("interact:" + object); err != nil {
		return err
	}
	switch object {
	case "Tree of Inifuss":
		g.W.Items[task.ItemBark] = 1
	case "cairn-stones":
		g.W.Items[task.ItemPortal] = 1
	case "Cain's Cage":
		g.stage[task.QCain] = task.StageDone
	}
	return nil
}

// PickUp implements task.Actions.
func (g *Game) PickUp(_ context.Context, item string) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin("pickup:" + item); err != nil {
		return err
	}
	if !g.ground[item] {
		return task.Errorf(task.KindNotFound, "no %s on the ground", item)
	}
	delete(g.ground, item)
	g.W.Items[item]++
	if item == task.ItemMalus {
		g.set(task.QTools, task.StageCleared)
	}
	return nil
}

// ActivateWaypoint implements task.Actions.
func (g *Game) ActivateWaypoint(_ context.Context, a task.Area) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin(fmt.Sprintf("wp:%d", a)); err != nil {
		return err
	}
	g.W.WPs[a] = true
	return nil
}

// Chores implements task.Actions.
func (g *Game) Chores(context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin("chores"); err != nil {
		return err
	}
	g.W.Potions = 10
	// A fresh character (EARLYGAME tests) arrives with no mana or antidote potions; the real chores buy them with gold,
	// so the fake does too (U: the thresholds are judgment). The default world has gold 500 and 2 antidotes.
	if g.W.Caster && g.W.Gold >= choreManaGold {
		g.W.ManaPotions = max(g.W.ManaPotions, 6)
	}
	if g.W.Gold >= choreAntidoteGold {
		g.W.Antidotes = max(g.W.Antidotes, 2)
	}
	return nil
}

const (
	choreManaGold     = 100
	choreAntidoteGold = 150
	killGold          = 30 // gold a LevelTo slice drops (U)
)

// LevelTo implements task.Actions.
func (g *Game) LevelTo(_ context.Context, n int) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if err := g.begin("level"); err != nil {
		return err
	}
	g.W.Level += g.XPStep
	if g.W.Level > n {
		g.W.Level = n
	}
	g.W.Gold += killGold
	g.W.Potions-- // fights cost potions
	if g.W.Potions < 0 {
		g.W.Potions = 0
	}
	return nil
}

// Count returns how many actions with the given key prefix ran.
func (g *Game) Count(prefix string) int {
	n := 0
	for _, l := range g.Log {
		if len(l) >= len(prefix) && l[:len(prefix)] == prefix {
			n++
		}
	}
	return n
}

// MarkDone sets a quest's true bit to Done (for tests that start with finished quests).
func (g *Game) MarkDone(q string) { g.stage[q] = task.StageDone }
