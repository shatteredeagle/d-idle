package bot

import (
	"context"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// Item classes of the fake catalog for the Act 2 quest drops (codes as the data rows name them; no display name is
// matched: the code carries the identity).
const (
	bookClass = 930 + iota
	scrollTr1Class
	amuletClass
)

type chunkCatalog struct{ staffCatalog }

func (c chunkCatalog) Item(class int) (ItemMeta, bool) {
	code := map[int]string{bookClass: "ass", scrollTr1Class: "tr1", amuletClass: "vip"}[class]
	if code == "" {
		return c.staffCatalog.Item(class)
	}
	return ItemMeta{Name: "item " + code, Code: code, W: 1, H: 1}, true
}

// chunkWorld is the Act 2 fake: chests and altars drop an item on the ground when clicked (never when drops is empty),
// the looter moves ground items into the pack, a book is read away by its click, talks move quest words. Each adapter
// reports success every time, so only the world can say whether the step happened.
type chunkWorld struct {
	g       *rig
	mu      sync.Mutex
	drops   map[string]int // object name -> class that lands on the ground
	ground  []int
	clicks  []string
	pickups int
	talks   []string
	words   map[string]task.QuestState
	onTalk  map[string]func(*chunkWorld) // "npc/pick" -> effect
}

func (c *chunkWorld) Interact(_ context.Context, name string) error {
	c.clicks = append(c.clicks, name)
	if name == "item:"+task.ItemBook {
		c.g.w.set(func(w *world) { w.items = dropClasses(w.items, bookClass) })
		return nil
	}
	if cl, ok := c.drops[name]; ok {
		c.ground = append(c.ground, cl)
	}
	return nil
}

func (c *chunkWorld) PickUp(_ context.Context, o PickOpts) (PickReport, error) {
	c.pickups++
	var keep []int
	n := 0
	for _, cl := range c.ground {
		m, _ := chunkCatalog{}.Item(cl)
		take := false
		for _, want := range o.Names {
			take = take || questItemCodes[m.Code] == want
		}
		if take {
			c.g.w.set(func(w *world) { w.items = append(w.items, packItem(cl)) })
			n++
		} else {
			keep = append(keep, cl)
		}
	}
	c.ground = keep
	return PickReport{Picked: n}, nil
}

func (c *chunkWorld) Evaluate(d2.Item) Verdict { return VerdictKeep }

func (c *chunkWorld) Talk(_ context.Context, npc, pick string) error {
	c.talks = append(c.talks, npc+"/"+pick)
	if f := c.onTalk[npc+"/"+pick]; f != nil {
		f(c)
	}
	return nil
}

func (c *chunkWorld) Read(int) (QuestView, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return QuestView{Quests: c.words}, true
}

func (c *chunkWorld) setWord(q string, stage task.Stage, raw uint16) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.words[q] = task.QuestState{Stage: stage, Raw: raw, Ev: task.EvMemory}
}

func chunkRig(t *testing.T, class d2.Class, pack ...int) (taskActs, *chunkWorld) {
	g := newRig(t)
	g.w.class = class
	g.w.set(func(w *world) {
		for _, c := range pack {
			w.items = append(w.items, packItem(c))
		}
	})
	g.r.d.Catalog = chunkCatalog{}
	g.r.e.Tune.Quests.StepExec = 1
	cw := &chunkWorld{g: g, drops: map[string]int{}, words: map[string]task.QuestState{}, onTalk: map[string]func(*chunkWorld){}}
	g.r.d.Task = &TaskDeps{Objects: cw, Dialog: cw, Quests: cw}
	g.r.d.Loot = cw
	return taskActs{r: g.r}, cw
}

// Atma's two lines are talk rows: the quest line is Done when the word moves (bit 0x0004, V lab), the reward line when
// the stage reaches Done. Must fail: a talk that moves nothing uses the row's two tries. Two classes.
func TestRadamentTalkRows(t *testing.T) {
	for _, cl := range execClasses {
		a, cw := chunkRig(t, cl)
		cw.onTalk["Atma/quest"] = func(c *chunkWorld) { c.setWord(task.QRadament, task.StageStarted, 0x0004) }
		cw.onTalk["Atma/reward"] = func(c *chunkWorld) { c.setWord(task.QRadament, task.StageDone, 0x0004|0x0100) }
		if err := a.Talk(context.Background(), task.NPCAtma, "quest"); err != nil || len(cw.talks) != 1 {
			t.Fatalf("class %v: quest line: err %v talks %v", cl, err, cw.talks)
		}
		if err := a.Talk(context.Background(), task.NPCAtma, "reward"); err != nil || len(cw.talks) != 2 {
			t.Fatalf("class %v: reward line: err %v talks %v", cl, err, cw.talks)
		}
		a, cw = chunkRig(t, cl)
		err := a.Talk(context.Background(), task.NPCAtma, "reward")
		if err == nil || !strings.Contains(err.Error(), "not observed") || len(cw.talks) != 2 {
			t.Fatalf("class %v: must fail: err %v talks %v", cl, err, cw.talks)
		}
	}
}

// The Book of Skill: the use row right-clicks the book (the task layer's own pick-up node has put it in the pack); Done
// when the book has left the pack, and the pseudo item of the click is set. Must fail when the click leaves the book.
func TestRadamentBookIsRead(t *testing.T) {
	for _, cl := range execClasses {
		a, cw := chunkRig(t, cl, bookClass)
		if err := a.Interact(context.Background(), "item:"+task.ItemBook); err != nil {
			t.Fatalf("class %v: %v", cl, err)
		}
		if len(cw.clicks) != 1 || a.r.loop.task.pseudo[task.ItemBookRead] != 1 {
			t.Fatalf("class %v: clicks %v pseudo %v", cl, cw.clicks, a.r.loop.task.pseudo)
		}
		a, _ = chunkRig(t, cl, bookClass)
		a.r.d.Task.Objects = stuckBook{}
		err := a.Interact(context.Background(), "item:"+task.ItemBook)
		if err == nil || !strings.Contains(err.Error(), "not observed") || a.r.loop.task.pseudo[task.ItemBookRead] != 0 {
			t.Fatalf("class %v: must fail: err %v", cl, err)
		}
	}
}

// stuckBook says yes to the click and keeps the book in the pack.
type stuckBook struct{}

func (stuckBook) Interact(context.Context, string) error { return nil }

// The chests and the altar are interact rows with a pick-up behind them: Done when the item is in the PACK, and the task
// node's own pick-up afterwards is a no-op instead of "not on the ground" (taskActs.PickUp with the knob on).
func TestChestAndAltarRowsTakeTheItem(t *testing.T) {
	for _, cl := range execClasses {
		for _, c := range []struct {
			object, item string
			class        int
		}{
			{"Horadric Cube chest", task.ItemCube, cubeClass},
			{"Staff of Kings chest", task.ItemShaft, shaftClass},
			{"Horadric Scroll chest", task.ItemScroll, scrollTr1Class},
			{"taintedsunaltar", task.ItemStaffTop, amuletClass},
		} {
			a, cw := chunkRig(t, cl)
			cw.drops[c.object] = c.class
			if err := a.Interact(context.Background(), c.object); err != nil || len(cw.clicks) != 1 {
				t.Fatalf("class %v %s: err %v clicks %v", cl, c.object, err, cw.clicks)
			}
			before := cw.pickups
			if err := a.PickUp(context.Background(), c.item); err != nil || cw.pickups != before {
				t.Fatalf("class %v %s: node pick-up: err %v looter calls %d", cl, c.object, err, cw.pickups-before)
			}
			// must fail: a chest that drops nothing is clicked on each of the row's tries and is not Done
			a, cw = chunkRig(t, cl)
			err := a.Interact(context.Background(), c.object)
			if err == nil || !strings.Contains(err.Error(), "not observed") || len(cw.clicks) < 2 {
				t.Fatalf("class %v %s: must fail: err %v clicks %v", cl, c.object, err, cw.clicks)
			}
		}
	}
}

// The Summoner's journal is an interact row Done by the quest word; the Summoner kill step reads "The Summoner" from the
// data row (killed or stage Cleared) although the row names him "summoner".
func TestSummonerRows(t *testing.T) {
	for _, cl := range execClasses {
		a, cw := chunkRig(t, cl)
		if a.bossDead(task.BossSummoner) {
			t.Fatalf("class %v: alive Summoner read as dead", cl)
		}
		a.r.noteKill(task.BossSummoner)
		if !a.bossDead(task.BossSummoner) {
			t.Fatalf("class %v: a seen kill is dead", cl)
		}
		a, cw = chunkRig(t, cl)
		cw.setWord(task.QSummoner, task.StageCleared, 0)
		if !a.bossDead(task.BossSummoner) {
			t.Fatalf("class %v: stage Cleared is dead", cl)
		}
		// touching the journal: Done when the word reaches Done; a click that moves nothing fails
		if err := a.Interact(context.Background(), "Summoner's journal"); err == nil || len(cw.clicks) != 2 {
			t.Fatalf("class %v: must fail: err %v clicks %v", cl, err, cw.clicks)
		}
		cw.setWord(task.QSummoner, task.StageDone, 0)
		if err := a.Interact(context.Background(), "Summoner's journal"); err != nil {
			t.Fatalf("class %v: %v", cl, err)
		}
	}
}

// With the knob at 0 every one of these keeps its old one-call handler.
func TestAct2ChunksOffKeepOldPath(t *testing.T) {
	a, cw := chunkRig(t, d2.Paladin)
	a.r.e.Tune.Quests.StepExec = 0
	ctx := context.Background()
	if err := a.Interact(ctx, "Horadric Scroll chest"); err != nil || len(cw.clicks) != 1 || cw.pickups != 0 {
		t.Fatalf("chest: %v %v %d", err, cw.clicks, cw.pickups)
	}
	if err := a.Talk(ctx, task.NPCAtma, "reward"); err != nil || len(cw.talks) != 1 {
		t.Fatalf("talk: %v %v", err, cw.talks)
	}
	if err := a.Interact(ctx, "item:"+task.ItemBook); err != nil || len(cw.clicks) != 2 {
		t.Fatalf("book: %v %v", err, cw.clicks)
	}
}

// Tyrael after the Duriel kill (lab night 2026-11-28, drill.act2.duriel.stepexec 0 of 3): he stands out of sight in the
// lair, the dialog adapter walks only to an NPC the snapshot lists and the chunk parked. The talk row walks the level
// until the snapshot lists him, then talks. Must fail without that: no explore call. In a town nobody is looked for.
func TestTalkRowLooksForAFieldNPC(t *testing.T) {
	const lair, tyrael = 73, 251
	for _, cl := range execClasses {
		a, cw := chunkRig(t, cl)
		cw.setWord(task.QDuriel, task.StageCleared, 0x0020)
		cw.onTalk["Tyrael/reward"] = func(c *chunkWorld) { c.setWord(task.QDuriel, task.StageCleared, 0x0020|0x2000) }
		cw.g.w.set(func(w *world) { w.area = lair })
		explores := 0
		cw.g.mover.exploreFn = func(_ context.Context, done func() bool) error {
			explores++
			if done() {
				t.Errorf("class %v: Tyrael read as seen before he is in the snapshot", cl)
			}
			cw.g.w.set(func(w *world) {
				w.mons = append(w.mons, d2.Unit{Type: d2.TypeMonster, ID: 77, Class: tyrael, X: 150, Y: 150})
			})
			if !done() {
				t.Errorf("class %v: Tyrael in the snapshot is not seen", cl)
			}
			return nil
		}
		if err := a.Talk(context.Background(), task.NPCTyrael, "reward"); err != nil || explores != 1 || len(cw.talks) != 1 {
			t.Fatalf("class %v: err %v explores %d talks %v", cl, err, explores, cw.talks)
		}
		// in sight already: no walk
		cw.setWord(task.QDuriel, task.StageCleared, 0x0020)
		if err := a.Talk(context.Background(), task.NPCTyrael, "reward"); err != nil || explores != 1 {
			t.Fatalf("class %v: second talk: err %v explores %d", cl, err, explores)
		}
		// a town NPC out of sight is the town adapter's walk
		a, cw = chunkRig(t, cl)
		cw.g.mover.exploreFn = func(context.Context, func() bool) error { explores++; return nil }
		cw.onTalk["Atma/quest"] = func(c *chunkWorld) { c.setWord(task.QRadament, task.StageStarted, 0x0004) }
		if err := a.Talk(context.Background(), task.NPCAtma, "quest"); err != nil || explores != 1 {
			t.Fatalf("class %v: town talk: err %v explores %d", cl, err, explores)
		}
	}
}
