package bot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/questspec"
	"d2lod-bot/internal/task"
)

const scrollClass = 900

// scrollCatalog knows the Scroll of Inifuss (base code bks) on top of the shared fake.
type scrollCatalog struct{ fakeCatalog }

func (c scrollCatalog) Item(class int) (ItemMeta, bool) {
	if class == scrollClass {
		return ItemMeta{Name: "Scroll of Inifuss", Code: "bks", W: 1, H: 2}, true
	}
	return c.fakeCatalog.Item(class)
}

// packLoot puts the scroll in the pack on its n-th call (the drop lands a moment after the click); never when n < 0.
type packLoot struct {
	fakeLoot
	n, calls int
}

func (l *packLoot) PickUp(ctx context.Context, o PickOpts) (PickReport, error) {
	l.w.call("pickup")
	l.calls++
	if l.n >= 0 && l.calls == l.n {
		l.w.set(func(w *world) {
			w.items = append(w.items, d2.Unit{ID: 77, Type: d2.TypeItem, Class: scrollClass, Item: &d2.Item{Where: d2.ContInventory}})
		})
		return PickReport{Picked: 1}, nil
	}
	return PickReport{}, nil
}

func execRig(t *testing.T, class d2.Class, dropAt int) (taskActs, *packLoot, *fakeObjects) {
	g := newRig(t)
	g.w.class = class
	g.r.d.Catalog = scrollCatalog{}
	g.r.e.Tune.Quests.StepExec = 1
	l := &packLoot{fakeLoot: fakeLoot{w: g.w}, n: dropAt}
	objs := &fakeObjects{}
	g.r.d.Task = &TaskDeps{Objects: objs}
	g.r.d.Loot = l
	return taskActs{r: g.r}, l, objs
}

var execClasses = []d2.Class{d2.Sorceress, d2.Paladin}

// The executor clicks the tree once, picks up until the scroll is in the PACK (not until the looter says so), and the
// pseudo item of the click is set.
func TestStepExecTakesTheScrollOnTwoClasses(t *testing.T) {
	for _, cl := range execClasses {
		a, l, objs := execRig(t, cl, 3)
		if err := a.Interact(context.Background(), TreeObject); err != nil {
			t.Fatalf("class %v: %v", cl, err)
		}
		if len(objs.clicked) != 1 || objs.clicked[0] != TreeObject || l.calls != 3 {
			t.Fatalf("class %v: clicked %v pickups %d", cl, objs.clicked, l.calls)
		}
	}
}

// Must fail: a looter that says "picked" while the pack never gets the scroll is not Done; the post is the pack.
func TestStepExecWaitsForThePackNotForTheLooter(t *testing.T) {
	for _, cl := range execClasses {
		a, _, objs := execRig(t, cl, -1)
		a.r.d.Loot = liar{fakeLoot{w: a.r.d.Loot.(*packLoot).w}}
		err := a.Interact(context.Background(), TreeObject)
		if task.Classify(err) != task.KindNotFound || err == nil || !strings.Contains(err.Error(), "not observed") {
			t.Fatalf("class %v: got %v", cl, err)
		}
		if len(objs.clicked) != 3 { // tries 3 of the data row
			t.Fatalf("class %v: clicks %d, want the row's 3 tries", cl, len(objs.clicked))
		}
	}
}

type liar struct{ fakeLoot }

func (l liar) PickUp(context.Context, PickOpts) (PickReport, error) {
	return PickReport{Picked: 1}, nil
}

// With the knob at 0 the old handler runs (its own pick-up loop of treeDropPolls looks), the executor is not used.
func TestStepExecOffKeepsTheOldHandler(t *testing.T) {
	a, l, _ := execRig(t, d2.Sorceress, -1)
	a.r.e.Tune.Quests.StepExec = 0
	if err := a.Interact(context.Background(), TreeObject); task.Classify(err) != task.KindNotFound {
		t.Fatalf("got %v", err)
	}
	if l.calls != treeDropPolls {
		t.Fatalf("polls %d, want the old bound %d", l.calls, treeDropPolls)
	}
}

const deciphered = 901

// bkdCatalog knows both the scroll (bks) and what Akara hands back after reading it (bkd).
type bkdCatalog struct{ scrollCatalog }

func (c bkdCatalog) Item(class int) (ItemMeta, bool) {
	if class == deciphered {
		return ItemMeta{Name: "Key to the Cairn Stones", Code: "bkd", W: 1, H: 2}, true
	}
	return c.scrollCatalog.Item(class)
}

// akaraReads swaps the scroll for the key in the pack on its n-th reading (never when n < 0); liar: says yes, changes nothing.
type akaraReads struct {
	g        *rig
	n, calls int
	picks    []string
}

func (d *akaraReads) Talk(_ context.Context, npc, pick string) error {
	d.picks = append(d.picks, npc+"/"+pick)
	d.calls++
	if d.n >= 0 && d.calls == d.n {
		d.g.w.set(func(w *world) {
			for i := range w.items {
				if w.items[i].Class == scrollClass {
					w.items[i].Class = deciphered
				}
			}
		})
	}
	return nil
}

func talkRig(t *testing.T, class d2.Class, readAt int) (taskActs, *akaraReads) {
	g := newRig(t)
	g.w.class = class
	g.w.set(func(w *world) {
		w.items = append(w.items, d2.Unit{ID: 77, Type: d2.TypeItem, Class: scrollClass, Item: &d2.Item{Where: d2.ContInventory}})
	})
	g.r.d.Catalog = bkdCatalog{}
	g.r.e.Tune.Quests.StepExec = 1
	d := &akaraReads{g: g, n: readAt}
	g.r.d.Task = &TaskDeps{Dialog: d}
	return taskActs{r: g.r}, d
}

// The bark reading is a talk step of the data (act1 chain read_scroll): the executor talks to Akara with the row's pick and
// waits for the key (bkd) in the PACK; a second try happens when the first talk changed nothing. Two classes.
func TestStepExecTalkReadsTheBark(t *testing.T) {
	for _, cl := range execClasses {
		a, d := talkRig(t, cl, 2)
		if err := a.Talk(context.Background(), task.NPCAkara, "bark"); err != nil {
			t.Fatalf("class %v: %v", cl, err)
		}
		if len(d.picks) != 2 || d.picks[0] != "Akara/bark" {
			t.Fatalf("class %v: talks %v", cl, d.picks)
		}
	}
}

// Must fail: a dialog that says yes while the pack keeps the scroll is not Done (the post is the pack), and the row's three
// tries are all used.
func TestStepExecTalkWaitsForThePack(t *testing.T) {
	for _, cl := range execClasses {
		a, d := talkRig(t, cl, -1)
		err := a.Talk(context.Background(), task.NPCAkara, "bark")
		if err == nil || !strings.Contains(err.Error(), "not observed") || len(d.picks) != 3 {
			t.Fatalf("class %v: err %v talks %v", cl, err, d.picks)
		}
	}
}

// Another NPC or another line, or the knob at 0, keeps the old path (the dialog is called once, as the task layer asks).
func TestStepExecTalkOthersKeepTheOldPath(t *testing.T) {
	for _, tc := range []struct {
		npc, pick string
		knob      int
	}{{task.NPCAkara, "reward", 1}, {task.NPCKashya, "bark", 1}, {task.NPCAkara, "bark", 0}} {
		a, d := talkRig(t, d2.Sorceress, -1)
		a.r.e.Tune.Quests.StepExec = tc.knob
		if err := a.Talk(context.Background(), tc.npc, tc.pick); err != nil || len(d.picks) != 1 {
			t.Fatalf("%+v: err %v talks %v", tc, err, d.picks)
		}
	}
}

// The USE adapter right-clicks the named pack item through the object adapter; an unknown code is a fatal row error.
func TestStepExecUseAdapter(t *testing.T) {
	for _, cl := range execClasses {
		a, _, objs := execRig(t, cl, -1)
		be := &execBackend{a: a}
		if err := be.Act(context.Background(), "use", questspec.Target{Kind: "item", Code: "bks"}, questspec.ActArgs{}); err != nil {
			t.Fatalf("class %v: %v", cl, err)
		}
		if len(objs.clicked) != 1 || objs.clicked[0] != "item:"+task.ItemBark {
			t.Fatalf("class %v: clicked %v", cl, objs.clicked)
		}
		var fatal *questspec.FatalError
		if err := be.Act(context.Background(), "use", questspec.Target{Kind: "item", Code: "zzz"}, questspec.ActArgs{}); !errors.As(err, &fatal) {
			t.Fatalf("class %v: unknown code: %v", cl, err)
		}
		if err := be.Act(context.Background(), "talk", questspec.Target{Kind: "npc", Class: 1}, questspec.ActArgs{}); !errors.As(err, &fatal) {
			t.Fatalf("class %v: unknown npc: %v", cl, err)
		}
	}
}
