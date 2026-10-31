package bot

import (
	"context"
	"errors"
	"strings"
	"testing"

	"d2lod-bot/internal/task"
)

type fakeObjects struct{ clicked []string }

func (o *fakeObjects) Interact(_ context.Context, name string) error {
	o.clicked = append(o.clicked, name)
	return nil
}

func TestTaskActsCubeAndMercNeedTheirCapability(t *testing.T) {
	g := newRig(t)
	acts := taskActs{r: g.r}
	g.r.d.Task = &TaskDeps{}
	if err := acts.Transmute(context.Background(), []string{"a"}); task.Classify(err) != task.KindPrereq {
		t.Fatalf("cube without capability: %v", err)
	}
	if err := acts.HireMerc(context.Background(), 3, "Cold"); task.Classify(err) != task.KindPrereq {
		t.Fatalf("merc without capability: %v", err)
	}
	var gotIn []string
	var gotAct int
	var gotAura string
	g.r.d.Task = &TaskDeps{
		Cube: func(_ context.Context, in []string) error { gotIn = in; return nil },
		HireMerc: func(_ context.Context, act int, aura string) error {
			gotAct, gotAura = act, aura
			return errors.New("no gold")
		},
	}
	if err := acts.Transmute(context.Background(), []string{task.ItemFlail, task.ItemEye}); err != nil || len(gotIn) != 2 {
		t.Fatalf("cube: %v %v", err, gotIn)
	}
	if err := acts.HireMerc(context.Background(), 3, "Cold"); err == nil || gotAct != 3 || gotAura != "Cold" {
		t.Fatalf("merc: %v %d %q", err, gotAct, gotAura)
	}
}

// An object click that proves a pseudo item sets it for the World; a new game forgets it.
func TestInteractSetsPseudoItems(t *testing.T) {
	g := newRig(t)
	objs := &fakeObjects{}
	g.r.d.Task = &TaskDeps{Objects: objs}
	acts := taskActs{r: g.r}
	if _, err := (taskEnv{g.r}).World(context.Background()); err != nil { // the first poll counts the game
		t.Fatal(err)
	}
	if err := acts.Interact(context.Background(), "tal-orifice"); err != nil {
		t.Fatal(err)
	}
	w, err := taskEnv{g.r}.World(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.Items[task.ItemStaffPlaced] != 1 {
		t.Fatalf("staff placed pseudo item missing: %v", w.Items)
	}
	_ = acts.Interact(context.Background(), "Horadric Cube chest")
	if w2, _ := (taskEnv{g.r}).World(context.Background()); w2.Items[task.ItemBookRead] != 0 {
		t.Fatal("an unrelated click must not set a pseudo item")
	}
	g.r.obMu.Lock()
	g.r.ob.newGames++
	g.r.obMu.Unlock()
	if w3, _ := (taskEnv{g.r}).World(context.Background()); w3.Items[task.ItemStaffPlaced] != 0 {
		t.Fatal("a new game must forget the pseudo item")
	}
}

// SmashOrb without a town that can swap weapons is a prereq fault and sets nothing; the orb click goes through Interact.
func TestSmashOrbNeedsWeaponSwap(t *testing.T) {
	g := newRig(t)
	objs := &fakeObjects{}
	g.r.d.Task = &TaskDeps{Objects: objs}
	err := taskActs{r: g.r}.SmashOrb(context.Background(), task.ItemWill)
	if task.Classify(err) != task.KindPrereq || len(objs.clicked) != 0 {
		t.Fatalf("got %v clicked %v", err, objs.clicked)
	}
	if w, _ := (taskEnv{g.r}).World(context.Background()); w.Items[task.ItemOrbSmashed] != 0 {
		t.Fatal("a failed smash must not set the pseudo item")
	}
}

func TestWorldReportsMercenaryPseudoItem(t *testing.T) {
	g := newRig(t)
	g.r.d.Task = &TaskDeps{}
	w, err := taskEnv{g.r}.World(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if w.HasMerc != (w.Items[task.ItemMercenary] == 1) {
		t.Fatalf("HasMerc %v but Items %v", w.HasMerc, w.Items)
	}
}

// Reading the scroll and touching Anya prove their pseudo items; the forge needs a town that swaps weapons.
func TestAct5PseudoItemsAndForge(t *testing.T) {
	g := newRig(t)
	objs := &fakeObjects{}
	g.r.d.Task = &TaskDeps{Objects: objs}
	acts := taskActs{r: g.r}
	ctx := context.Background()
	if _, err := (taskEnv{g.r}).World(ctx); err != nil {
		t.Fatal(err)
	}
	if err := acts.UseItem(ctx, task.ItemResScroll); err != nil {
		t.Fatal(err)
	}
	if objs.clicked[0] != "item:"+task.ItemResScroll {
		t.Fatalf("clicked %v", objs.clicked)
	}
	if err := acts.Interact(ctx, AnyaObject); err != nil { // no potion in the rig's pack: a touch
		t.Fatal(err)
	}
	w, _ := taskEnv{g.r}.World(ctx)
	if w.Items[task.ItemScrollRead] != 1 || w.Items[task.ItemAnyaTouched] != 1 || w.Items[task.ItemAnyaThawed] != 0 {
		t.Fatalf("pseudo items %v", w.Items)
	}
	objs.clicked = nil
	err := acts.UseForge(ctx, task.ItemSoulstone, task.ItemHammer)
	if task.Classify(err) != task.KindPrereq || len(objs.clicked) != 1 || objs.clicked[0] != ForgePlaceObject {
		t.Fatalf("got %v clicked %v", err, objs.clicked)
	}
	if w, _ := (taskEnv{g.r}).World(ctx); w.Items[task.ItemForgeUsed] != 0 {
		t.Fatal("a failed forge must not set the pseudo item")
	}
}

// The Inifuss tree click only drops the scroll: the adapter picks it up, and the cairn stones prove the portal pseudo item.
func TestTreePicksUpTheScrollAndStonesSetThePortal(t *testing.T) {
	g := newRig(t)
	objs := &fakeObjects{}
	g.r.d.Task = &TaskDeps{Objects: objs}
	g.r.d.Loot = fakeLoot{w: g.w}
	acts := taskActs{r: g.r}
	ctx := context.Background()
	if _, err := (taskEnv{g.r}).World(ctx); err != nil {
		t.Fatal(err)
	}
	if err := acts.Interact(ctx, TreeObject); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(g.w.log(), "pickup") {
		t.Fatal("no pick-up after the tree")
	}
	if err := acts.Interact(ctx, "cairn-stones"); err != nil {
		t.Fatal(err)
	}
	if w, _ := (taskEnv{g.r}).World(ctx); w.Items[task.ItemPortal] != 1 {
		t.Fatalf("portal pseudo item missing: %v", w.Items)
	}
}
