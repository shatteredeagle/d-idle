package bot

import (
	"context"
	"strings"
	"sync"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// Item classes of the fake catalog for the Horadric Staff parts (codes as the data rows name them).
const (
	cubeClass = 910 + iota
	shaftClass
	topClass
	staffClass
)

type staffCatalog struct{ fakeCatalog }

func (c staffCatalog) Item(class int) (ItemMeta, bool) {
	code := map[int]string{cubeClass: "box", shaftClass: "msf", topClass: "vip", staffClass: "hst"}[class]
	if code == "" {
		return c.fakeCatalog.Item(class)
	}
	// a display name the steps never ask for: the code must carry the identity (rule 5)
	return ItemMeta{Name: "item " + code, Code: code, W: 1, H: 1}, true
}

func packItem(class int) d2.Unit {
	return d2.Unit{ID: uint32(class), Type: d2.TypeItem, Class: class, Item: &d2.Item{Where: d2.ContInventory}}
}

// staffWorld: the Cube and the orifice act on the fake world on their n-th call (never when n < 0); each reports success
// every time, so only the world can say whether the step happened. word is what the quest reader says of the Staff.
type staffWorld struct {
	g                    *rig
	cubeAt, placeAt      int
	cubes, places, talks int
	got                  []string
	mu                   sync.Mutex
	word                 task.Stage
}

func (s *staffWorld) cube(_ context.Context, in []string) error {
	s.cubes++
	s.got = in
	if s.cubes == s.cubeAt {
		s.g.w.set(func(w *world) { w.items = append(dropClasses(w.items, shaftClass, topClass), packItem(staffClass)) })
	}
	return nil
}

func (s *staffWorld) Interact(_ context.Context, name string) error {
	s.places++
	s.got = []string{name}
	if s.places == s.placeAt {
		s.g.w.set(func(w *world) { w.items = dropClasses(w.items, staffClass) })
	}
	return nil
}

func (s *staffWorld) Talk(_ context.Context, npc, pick string) error {
	s.talks++
	s.got = []string{npc + "/" + pick}
	return nil
}

func (s *staffWorld) Read(int) (QuestView, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return QuestView{Quests: map[string]task.QuestState{task.QStaff: {Stage: s.word, Ev: task.EvMemory}}}, true
}

func dropClasses(items []d2.Unit, classes ...int) []d2.Unit {
	var keep []d2.Unit
	for _, u := range items {
		drop := false
		for _, c := range classes {
			drop = drop || u.Class == c
		}
		if !drop {
			keep = append(keep, u)
		}
	}
	return keep
}

func staffRig(t *testing.T, class d2.Class, pack ...int) (taskActs, *staffWorld) {
	g := newRig(t)
	g.w.class = class
	g.w.set(func(w *world) {
		for _, c := range pack {
			w.items = append(w.items, packItem(c))
		}
	})
	g.r.d.Catalog = staffCatalog{}
	g.r.e.Tune.Quests.StepExec = 1
	sw := &staffWorld{g: g, cubeAt: -1, placeAt: -1}
	g.r.d.Task = &TaskDeps{Cube: sw.cube, Objects: sw, Dialog: sw, Quests: sw}
	return taskActs{r: g.r}, sw
}

// The Staff assembly is the transmute row of the data (horadric_staff/transmute): the Cube gets the row's inputs (by
// code, mapped to the adapter's names) and the step is Done only when the Staff is in the PACK. Two classes.
func TestStepExecTransmutesTheStaff(t *testing.T) {
	for _, cl := range execClasses {
		a, sw := staffRig(t, cl, cubeClass, shaftClass, topClass)
		sw.cubeAt = 1
		if err := a.Transmute(context.Background(), []string{task.ItemStaffTop, task.ItemShaft}); err != nil {
			t.Fatalf("class %v: %v", cl, err)
		}
		if sw.cubes != 1 || cubeKey(sw.got) != cubeKey([]string{task.ItemShaft, task.ItemStaffTop}) {
			t.Fatalf("class %v: cubes %d inputs %v", cl, sw.cubes, sw.got)
		}
	}
}

// Must fail: a Cube that says yes while the pack never holds the Staff uses the row's two tries and is not Done.
func TestStepExecTransmuteWaitsForThePack(t *testing.T) {
	for _, cl := range execClasses {
		a, sw := staffRig(t, cl, cubeClass, shaftClass, topClass)
		err := a.Transmute(context.Background(), []string{task.ItemShaft, task.ItemStaffTop})
		if err == nil || !strings.Contains(err.Error(), "not observed") || sw.cubes != 2 {
			t.Fatalf("class %v: err %v cubes %d", cl, err, sw.cubes)
		}
	}
}

// The orifice is the interact row seven_tombs/place_staff: Done when the Staff has left the pack, and the task layer's
// pseudo item is set after it; must fail when the adapter says yes and the Staff stays. Two classes.
func TestStepExecPlacesTheStaff(t *testing.T) {
	for _, cl := range execClasses {
		a, sw := staffRig(t, cl, cubeClass, staffClass)
		sw.placeAt = 2
		if err := a.Interact(context.Background(), OrificeObject); err != nil || sw.places != 2 {
			t.Fatalf("class %v: err %v places %d", cl, err, sw.places)
		}
		if a.r.loop.task.pseudo[task.ItemStaffPlaced] != 1 {
			t.Fatalf("class %v: pseudo item not set", cl)
		}
		a, sw = staffRig(t, cl, cubeClass, staffClass)
		err := a.Interact(context.Background(), OrificeObject)
		if err == nil || !strings.Contains(err.Error(), "not observed") || sw.places != 2 {
			t.Fatalf("class %v: must fail: err %v places %d", cl, err, sw.places)
		}
		if a.r.loop.task.pseudo[task.ItemStaffPlaced] != 0 {
			t.Fatalf("class %v: pseudo item set on a failed placement", cl)
		}
	}
}

// quest_state is read from the quest READER: with the Staff word Done (placed in an earlier game) the transmute row is
// Done before any Cube call (resumption). The Cain hand-in has no exec row (no game fact proves it yet), so its talk keeps
// the old one-call path with the knob on.
func TestStepExecQuestWordAndCainTalk(t *testing.T) {
	for _, cl := range execClasses {
		a, sw := staffRig(t, cl, cubeClass, shaftClass, topClass)
		sw.word = task.StageDone
		if err := a.Transmute(context.Background(), []string{task.ItemShaft, task.ItemStaffTop}); err != nil || sw.cubes != 0 {
			t.Fatalf("class %v: err %v cubes %d", cl, err, sw.cubes)
		}
		sw.word = task.StageCleared
		if err := a.Transmute(context.Background(), []string{task.ItemShaft, task.ItemStaffTop}); err == nil || sw.cubes != 2 {
			t.Fatalf("class %v: must fail below Done: err %v cubes %d", cl, err, sw.cubes)
		}
		if err := a.Talk(context.Background(), task.NPCCain, "horadric staff"); err != nil || sw.talks != 1 {
			t.Fatalf("class %v: cain: err %v talks %d", cl, err, sw.talks)
		}
	}
}

// With the knob at 0 the old one-call handlers run: one Cube call, one click, one talk, whatever the world says.
func TestStepExecAct2OffKeepsTheOldPath(t *testing.T) {
	a, sw := staffRig(t, d2.Paladin, cubeClass, shaftClass, topClass, staffClass)
	a.r.e.Tune.Quests.StepExec = 0
	ctx := context.Background()
	if err := a.Transmute(ctx, []string{task.ItemShaft, task.ItemStaffTop}); err != nil || sw.cubes != 1 {
		t.Fatalf("cube: %v %d", err, sw.cubes)
	}
	if err := a.Interact(ctx, OrificeObject); err != nil || sw.places != 1 {
		t.Fatalf("orifice: %v %d", err, sw.places)
	}
	if err := a.Talk(ctx, task.NPCCain, "horadric staff"); err != nil || sw.talks != 1 {
		t.Fatalf("talk: %v %d", err, sw.talks)
	}
}
