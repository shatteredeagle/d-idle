package invdesk

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/status"
)

// A small gworld the fake executors change like the game would: worn slots, backpack and stash. The desk reads it
// through Inventory and Snap, so an apply is tested end to end through RunPending.

type gitem struct {
	id    uint32
	code  string
	name  string
	slot  string // the slot kind it fits: rhand, lhand, head, ring ...
	two   bool
	w, h  int
	class map[d2.Class]bool
	level int
	unid  bool // not identified yet
}

type gworld struct {
	t      *testing.T
	class  d2.Class
	worn   map[string]*gitem
	pack   []*gitem
	stash  []*gitem
	inTown bool
	log    []string
	failEq map[uint32]error
}

var codes = map[string]int{"sword": 10, "shield": 11, "staff": 12, "orb": 13, "helm": 14, "ring": 15, "club": 16, "armor": 17}

func (w *gworld) item(code string, id uint32, name string, two bool, fits string, classes ...d2.Class) *gitem {
	g := &gitem{id: id, code: code, name: name, slot: fits, two: two, w: 1, h: 2, class: map[d2.Class]bool{}, level: 10}
	for _, c := range classes {
		g.class[c] = true
	}
	return g
}

func (w *gworld) find(id uint32) (it *gitem, place, slot string) {
	for k, g := range w.worn {
		if g.id == id {
			return g, "worn", k
		}
	}
	for _, g := range w.pack {
		if g.id == id {
			return g, "pack", ""
		}
	}
	for _, g := range w.stash {
		if g.id == id {
			return g, "stash", ""
		}
	}
	return nil, "", ""
}

func (w *gworld) remove(id uint32) {
	for k, g := range w.worn {
		if g.id == id {
			delete(w.worn, k)
		}
	}
	keep := func(l []*gitem) []*gitem {
		var o []*gitem
		for _, g := range l {
			if g.id != id {
				o = append(o, g)
			}
		}
		return o
	}
	w.pack, w.stash = keep(w.pack), keep(w.stash)
}

func (g *gitem) status(x int) status.Item {
	return status.Item{ID: g.id, Code: g.code, Name: g.name, Quality: "magic", Level: g.level, Identified: !g.unid, X: x % PackCols, Y: x / PackCols * 2, W: g.w, H: g.h}
}

func (w *gworld) data() status.InventoryData {
	d := status.InventoryData{Equipment: map[string]*status.Item{}}
	for k, g := range w.worn {
		s := g.status(0)
		d.Equipment[k] = &s
	}
	for i, g := range w.pack {
		d.Inventory = append(d.Inventory, g.status(i))
	}
	for i, g := range w.stash {
		d.Stash = append(d.Stash, g.status(i))
	}
	return d
}

func (w *gworld) snap() *d2.Snapshot {
	area := 2
	if w.inTown {
		area = 1
	}
	s := &d2.Snapshot{Player: &d2.Player{Class: w.class, Area: area, Level: 20}, Inventory: &d2.Inventory{}}
	add := func(g *gitem, where d2.Container, body int) {
		s.Inventory.Items = append(s.Inventory.Items, d2.Unit{ID: g.id, Class: codes[g.code], Item: &d2.Item{Where: where, Body: body, Quality: 4, ILvl: g.level}})
	}
	for k, g := range w.worn {
		add(g, d2.ContEquipment, SlotBody[k])
	}
	for _, g := range w.pack {
		add(g, d2.ContInventory, 0)
	}
	for _, g := range w.stash {
		add(g, d2.ContStash, 0)
	}
	return s
}

func fits(g *gitem, slot string) bool {
	if g.slot == "ring" {
		return slot == "rring" || slot == "lring"
	}
	return g.slot == slot
}

// CheckPlayerEquip / PlayerEquip: the leveler's rules in small (class, slot, the two-hander and shield rule).
func (w *gworld) CheckPlayerEquip(s *d2.Snapshot, id uint32, body int) (string, error) {
	g, place, _ := w.find(id)
	if g == nil || place != "pack" {
		return "", errors.New("the item is not in the backpack")
	}
	slot := ""
	for k, b := range SlotBody {
		if b == body {
			slot = k
		}
	}
	if !g.class[w.class] {
		return g.name, errors.New(g.name + ": restricted to another class")
	}
	if !fits(g, slot) {
		return g.name, errors.New(g.name + ": the item does not fit that slot")
	}
	other := map[string]string{"rhand": "lhand", "lhand": "rhand"}[slot]
	if o := w.worn[other]; o != nil && (g.two || o.two) {
		return g.name, errors.New("a two-handed weapon and a shield cannot be worn together; take the other one off first")
	}
	return g.name, nil
}

func (w *gworld) PlayerEquip(ctx context.Context, id uint32, body int) error {
	if err := w.failEq[id]; err != nil {
		return err
	}
	if _, err := w.CheckPlayerEquip(w.snap(), id, body); err != nil {
		return err
	}
	g, _, _ := w.find(id)
	slot := ""
	for k, b := range SlotBody {
		if b == body {
			slot = k
		}
	}
	w.remove(id)
	if old := w.worn[slot]; old != nil {
		w.pack = append(w.pack, old)
	}
	w.worn[slot] = g
	w.log = append(w.log, "equip "+g.name+" "+slot)
	return nil
}

func (w *gworld) PlayerUnequip(ctx context.Context, id uint32) error {
	g, place, _ := w.find(id)
	if g == nil || place != "worn" {
		return errors.New("not worn")
	}
	w.remove(id)
	w.pack = append(w.pack, g)
	w.log = append(w.log, "unequip "+g.name)
	return nil
}

func (w *gworld) IsTwoHanded(s *d2.Snapshot, id uint32) bool {
	g, _, _ := w.find(id)
	return g != nil && g.two
}

type worldStash struct {
	*fakeSv
	w *gworld
}

func (f *worldStash) CheckStashMove(s *d2.Snapshot, id uint32, toStash bool) (string, error) {
	return "thing", nil
}
func (f *worldStash) CheckStashIdentify(s *d2.Snapshot, id uint32) (string, error) { return "", nil }
func (f *worldStash) StashVisit(ctx context.Context, p StashPlan) StashResult {
	r := StashResult{Errs: map[uint32]error{}}
	for _, id := range p.Take {
		r.Errs[id] = nil
		g, place, _ := f.w.find(id)
		if g == nil || place != "stash" {
			r.Errs[id] = errors.New("not in the stash")
			continue
		}
		f.w.remove(id)
		f.w.pack = append(f.w.pack, g)
		f.w.log = append(f.w.log, "take "+g.name)
	}
	return r
}

func newGWorld(t *testing.T, class d2.Class) (*Desk, *gworld) {
	t.Helper()
	w := &gworld{t: t, class: class, worn: map[string]*gitem{}, failEq: map[uint32]error{}, inTown: true}
	now := time.Unix(1_800_000_000, 0)
	d, err := New(Config{
		Path:      filepath.Join(t.TempDir(), StateFile),
		Snap:      w.snap,
		Inventory: w.data,
		Now:       func() time.Time { now = now.Add(time.Second); return now },
	})
	if err != nil {
		t.Fatal(err)
	}
	d.SetEquipper(w)
	d.SetUnequipper(w)
	d.SetHands(w)
	d.SetServices(&worldStash{fakeSv: &fakeSv{tome: 1}, w: w})
	return d, w
}

// kit gives the class its things: a one-hander with a shield, a two-hander, a helm and two rings.
func kit(w *gworld, class d2.Class) {
	cl := []d2.Class{class}
	w.worn["rhand"] = w.item("sword", 1, "Short Sword", false, "rhand", cl...)
	w.worn["lhand"] = w.item("shield", 2, "Buckler", false, "lhand", cl...)
	w.pack = []*gitem{
		w.item("staff", 3, "War Staff", true, "rhand", cl...),
		w.item("helm", 4, "Cap", false, "head", cl...),
		w.item("ring", 5, "Ring A", false, "ring", cl...),
		w.item("ring", 6, "Ring B", false, "ring", cl...),
	}
	w.stash = []*gitem{w.item("club", 7, "Club", false, "rhand", cl...)}
}

func run(t *testing.T, d *Desk) []string {
	t.Helper()
	var out []string
	for i := 0; i < 10 && d.Pending() > 0; i++ {
		out = append(out, d.RunPending(context.Background())...)
	}
	return out
}

func TestSetSaveRenameDeleteAndLimits(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			d, w := newGWorld(t, class)
			kit(w, class)
			d.c.MaxSets = func() int { return 2 }
			a, err := d.SaveSet("  Melee   set ", "")
			if err != nil || a.Name != "Melee set" || len(a.Slots) != 2 || a.Target != TargetHero {
				t.Fatalf("save: %+v %v", a, err)
			}
			if _, err := d.SaveSet("melee SET", ""); code(err) != "name" {
				t.Errorf("duplicate name: %v", err)
			}
			if _, err := d.SaveSet("", ""); code(err) != "name" {
				t.Errorf("empty name: %v", err)
			}
			if _, err := d.SaveSet(strings.Repeat("x", 25), ""); code(err) != "name" {
				t.Errorf("long name: %v", err)
			}
			if _, err := d.SaveSet("b", ""); err != nil {
				t.Fatal(err)
			}
			if _, err := d.SaveSet("c", ""); code(err) != "full" {
				t.Errorf("limit: %v", err)
			}
			if err := d.RenameSet(a.ID, "Tank"); err != nil {
				t.Fatal(err)
			}
			if err := d.RenameSet(a.ID, "b"); code(err) != "name" {
				t.Errorf("rename onto another: %v", err)
			}
			if err := d.DeleteSet(a.ID); err != nil {
				t.Fatal(err)
			}
			if err := d.DeleteSet(a.ID); code(err) != "set" {
				t.Errorf("delete twice: %v", err)
			}
			// persisted, and an old file without sets still loads
			d2_, err := New(Config{Path: d.c.Path})
			if err != nil || len(d2_.st.Sets) != 1 || d2_.st.SetsSchema != SetsSchema {
				t.Fatalf("reload: %v %+v", err, d2_.st)
			}
			w.worn = map[string]*gitem{}
			if _, err := d.SaveSet("empty", ""); code(err) != "empty" {
				t.Errorf("nothing worn: %v", err)
			}
		})
	}
}

// A two-hander to one-hander and shield swap, both ways: the shield comes off before the two-hander goes on, and the
// weapon hand is dressed before the shield goes on.
func TestSetApplyTwoHanderAndShieldSwap(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			d, w := newGWorld(t, class)
			kit(w, class)
			melee, err := d.SaveSet("Sword and board", "")
			if err != nil {
				t.Fatal(err)
			}
			// wear the staff instead (the shield comes off by hand), save that as the caster set
			if _, err := d.RequestUnequip("lhand"); err != nil {
				t.Fatal(err)
			}
			run(t, d)
			if _, err := d.RequestEquip(3, "rhand"); err != nil {
				t.Fatal(err)
			}
			run(t, d)
			caster, err := d.SaveSet("Two hander", "")
			if err != nil {
				t.Fatal(err)
			}
			w.log = nil

			pv, err := d.PreviewSet(melee.ID)
			if err != nil || !pv.Ready {
				t.Fatalf("preview: %+v %v", pv, err)
			}
			if _, err := d.ApplySet(melee.ID); err != nil {
				t.Fatal(err)
			}
			if _, err := d.ApplySet(caster.ID); code(err) != "queued" {
				t.Errorf("second apply while the first waits: %v", err)
			}
			run(t, d)
			if w.worn["rhand"].name != "Short Sword" || w.worn["lhand"].name != "Buckler" {
				t.Fatalf("sword and board: %v log %v", w.worn, w.log)
			}

			w.log = nil
			if _, err := d.ApplySet(caster.ID); err != nil {
				t.Fatal(err)
			}
			run(t, d)
			if w.worn["rhand"].name != "War Staff" || w.worn["lhand"] != nil {
				t.Fatalf("two hander: %v log %v", w.worn, w.log)
			}
			if len(w.log) != 2 || !strings.HasPrefix(w.log[0], "unequip Buckler") || !strings.HasPrefix(w.log[1], "equip War Staff") {
				t.Errorf("order: %v", w.log)
			}
			// applying a set that is worn already has nothing to do
			if pv, _ := d.PreviewSet(caster.ID); pv.Ready || pv.Why != "the set is already worn" {
				t.Errorf("worn already: %+v", pv)
			}
		})
	}
}

func TestSetApplyMissingItemIsSkippedNeverSubstituted(t *testing.T) {
	d, w := newGWorld(t, d2.Paladin)
	kit(w, d2.Paladin)
	w.worn["head"] = w.item("helm", 8, "Great Helm", false, "head", d2.Paladin)
	gs, err := d.SaveSet("Full", "")
	if err != nil {
		t.Fatal(err)
	}
	// the great helm is sold, the cap in the pack must NOT take its place; the sword goes away from the hand
	w.remove(8)
	w.remove(1)
	pv, err := d.PreviewSet(gs.ID)
	if err != nil || pv.Missing != 2 {
		t.Fatalf("preview: %+v %v", pv, err)
	}
	for _, s := range pv.Slots {
		if s.Slot == "head" && s.Status != "missing" || s.Slot == "rhand" && s.Status != "missing" {
			t.Errorf("slot %s: %+v", s.Slot, s)
		}
	}
	if pv.Ready || pv.Why != "the set is already worn" && !strings.Contains(pv.Why, "nothing") {
		// the shield is still worn: nothing is left to do
		t.Errorf("only the shield is left and it is worn: %+v", pv)
	}
	if len(w.pack) == 0 || w.pack[1].name != "Cap" {
		t.Fatal("the cap should be untouched")
	}
}

func TestSetApplyItemFoundAfterRestartAndIdentify(t *testing.T) {
	d, w := newGWorld(t, d2.Sorceress)
	kit(w, d2.Sorceress)
	w.worn["lhand"].unid, w.worn["rhand"].unid = true, true // saved unidentified
	gs, _ := d.SaveSet("Mine", "")
	// the game restarts: every id is new and the names changed (identified since); the shield is in the pack now
	w.remove(2)
	sh := w.item("shield", 900, "Buckler of Warding", false, "lhand", d2.Sorceress)
	w.pack = append(w.pack, sh)
	w.worn["rhand"].id, w.worn["rhand"].name, w.worn["rhand"].unid = 901, "Sharp Short Staff", false
	w.pack[0].id = 902 // the war staff that was in the pack
	pv, err := d.PreviewSet(gs.ID)
	if err != nil {
		t.Fatal(err)
	}
	var lh SetSlotPreview
	for _, s := range pv.Slots {
		if s.Slot == "lhand" {
			lh = s
		}
	}
	if lh.Status != "pack" || lh.ID != 900 {
		t.Fatalf("the shield is found by its base: %+v", lh)
	}
	if _, err := d.ApplySet(gs.ID); err != nil {
		t.Fatal(err)
	}
	run(t, d)
	if w.worn["lhand"] == nil || w.worn["lhand"].id != 900 {
		t.Fatalf("worn: %v", w.worn)
	}
}

func TestSetApplyFromTheStashNeedsTheTownThenOrders(t *testing.T) {
	d, w := newGWorld(t, d2.Paladin)
	kit(w, d2.Paladin)
	gs, _ := d.SaveSet("Before", "")
	// the club replaces the sword in the set: the club lies in the stash
	gs2, _ := d.SaveSet("Club", "")
	d.mu.Lock()
	for i := range d.st.Sets {
		if d.st.Sets[i].ID == gs2.ID {
			d.st.Sets[i].Slots["rhand"] = SetItem{ID: 7, Code: "club", Quality: "magic", Level: 10, Name: "Club"}
		}
	}
	d.mu.Unlock()
	_ = gs
	w.inTown = false // in the field: the take is a town errand, the equip waits for it
	pv, _ := d.PreviewSet(gs2.ID)
	if !pv.Ready || len(pv.Steps) != 2 || pv.Steps[0].Op != KindTake || pv.Steps[1].Op != KindEquip {
		t.Fatalf("plan: %+v", pv)
	}
	if _, err := d.ApplySet(gs2.ID); err != nil {
		t.Fatal(err)
	}
	if got := d.RunPending(context.Background()); len(got) != 0 {
		t.Fatalf("nothing may run in the field: %v", got)
	}
	if d.ErrandsWaiting() != 1 {
		t.Errorf("the take is an errand: %d", d.ErrandsWaiting())
	}
	w.inTown = true
	run(t, d)
	if w.worn["rhand"].name != "Club" || len(w.stash) != 0 {
		t.Fatalf("club worn: %v log %v", w.worn, w.log)
	}
	if w.log[0] != "take Club" {
		t.Errorf("order: %v", w.log)
	}
}

func TestSetApplyStopsWhenAStepFailsAndCancels(t *testing.T) {
	d, w := newGWorld(t, d2.Paladin)
	kit(w, d2.Paladin)
	gs, _ := d.SaveSet("Mine", "")
	// both rings and the helm go on
	d.mu.Lock()
	for i := range d.st.Sets {
		d.st.Sets[i].Slots = map[string]SetItem{
			"head":  {ID: 4, Code: "helm", Quality: "magic", Level: 10, Name: "Cap"},
			"rring": {ID: 5, Code: "ring", Quality: "magic", Level: 10, Name: "Ring A"},
			"lring": {ID: 6, Code: "ring", Quality: "magic", Level: 10, Name: "Ring B"},
		}
	}
	d.mu.Unlock()
	w.failEq[5] = errors.New("it slipped")
	if _, err := d.ApplySet(gs.ID); err != nil {
		t.Fatal(err)
	}
	if n := len(d.View().Queue); n != 3 {
		t.Fatalf("steps: %d", n)
	}
	lines := run(t, d)
	v := d.View()
	if len(v.Queue) != 0 || w.worn["head"] == nil || w.worn["rring"] != nil || w.worn["lring"] != nil {
		t.Fatalf("after: queue %v worn %v lines %v", v.Queue, w.worn, lines)
	}
	failed, cancelled := 0, 0
	for _, r := range v.Recent {
		switch r.State {
		case StateFailed:
			failed++
		case StateCancelled:
			cancelled++
			if r.Group == "" || !strings.Contains(r.Reason, "stopped") {
				t.Errorf("cancel reason: %+v", r)
			}
		}
	}
	if failed != 1 || cancelled != 1 {
		t.Errorf("recent: %+v", v.Recent)
	}

	// cancelling one queued step cancels the rest of the group, and the group can be cancelled whole
	w.failEq = map[uint32]error{}
	w.worn = map[string]*gitem{}
	if _, err := d.ApplySet(gs.ID); err != nil {
		t.Fatal(err)
	}
	q := d.View().Queue
	if err := d.CancelApply(q[0].Group); err != nil {
		t.Fatal(err)
	}
	if len(d.View().Queue) != 0 || d.Pending() != 0 {
		t.Errorf("cancel apply: %+v", d.View().Queue)
	}
	if err := d.CancelApply(q[0].Group); code(err) != "item" {
		t.Errorf("cancel twice: %v", err)
	}
}

func TestSetApplyInAFightWaitsForTheBot(t *testing.T) {
	d, w := newGWorld(t, d2.Paladin)
	kit(w, d2.Paladin)
	gs, _ := d.SaveSet("Mine", "")
	w.worn = map[string]*gitem{}
	w.failEq[1] = errors.New("leveler: a fight is on")
	w.pack = append(w.pack, w.item("sword", 1, "Short Sword", false, "rhand", d2.Paladin))
	if _, err := d.ApplySet(gs.ID); err != nil {
		t.Fatal(err)
	}
	if out := d.RunPending(context.Background()); len(out) == 0 || !strings.Contains(out[0], "postponed") {
		t.Fatalf("out: %v", out)
	}
	for _, r := range d.View().Queue {
		if r.State != StateQueued {
			t.Errorf("postponed step must be queued again: %+v", r)
		}
	}
	delete(w.failEq, 1)
	run(t, d)
	if w.worn["rhand"] == nil {
		t.Fatalf("after the fight: %v", w.worn)
	}
}

func TestSetsLockedAndTheBotNeverApplies(t *testing.T) {
	d, w := newGWorld(t, d2.Paladin)
	kit(w, d2.Paladin)
	gs, _ := d.SaveSet("Mine", "")
	d.c.SetsGate = func() (bool, string) { return false, "Reach character level 15." }
	if _, err := d.SaveSet("x", ""); code(err) != "locked" {
		t.Errorf("save: %v", err)
	}
	if _, err := d.ApplySet(gs.ID); code(err) != "locked" {
		t.Errorf("apply: %v", err)
	}
	if v := d.View(); v.Sets == nil || v.Sets.Open || v.Sets.Why != "Reach character level 15." || len(v.Sets.Sets) != 0 {
		t.Errorf("locked view: %+v", v.Sets)
	}
	// nothing on the desk's bot-facing side queues or runs anything of a set by itself
	if d.Pending() != 0 || d.NextKind() != "" {
		t.Error("the desk queued something by itself")
	}
}

func TestMercSetsSaveAndPreviewButNoApply(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			d, w := newGWorld(t, class)
			kit(w, class)
			base := d.c.Inventory
			d.c.Inventory = func() status.InventoryData {
				v := base()
				v.Merc = &status.MercInv{Hired: true, Alive: true, GearKnown: true, Class: 338, Equipment: map[string]*status.Item{
					"mhead": {ID: 1001, Code: "helm", Name: "Merc Helm", Quality: "magic", Level: 9}}}
				return v
			}
			gs, err := d.SaveSet("Merc kit", TargetMerc)
			if err != nil || gs.Target != TargetMerc || len(gs.Slots) != 1 {
				t.Fatalf("merc save: %+v %v", gs, err)
			}
			if _, err := d.SaveSet("Hero kit", ""); err != nil {
				t.Fatal("hero and merc sets are separate: ", err)
			}
			pv, err := d.PreviewSet(gs.ID)
			if err != nil || pv.Ready || !strings.Contains(pv.Why, "mercenary") || pv.Slots[0].Status != "worn" {
				t.Fatalf("merc preview: %+v %v", pv, err)
			}
			if _, err := d.ApplySet(gs.ID); code(err) != "merc" {
				t.Errorf("merc apply: %v", err)
			}
			if _, err := d.RequestEquipFor(4, "mhead", TargetMerc); code(err) != "merc" {
				t.Errorf("merc equip: %v", err)
			}
			if _, err := d.RequestUnequipFor("mhead", TargetMerc); code(err) != "merc" {
				t.Errorf("merc unequip: %v", err)
			}
			if d.Pending() != 0 {
				t.Error("a merc refusal queued something")
			}
		})
	}
}

func TestUnequipRequestRefusalsAndRun(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			d, w := newGWorld(t, class)
			kit(w, class)
			if _, err := d.RequestUnequip("head"); code(err) != "item" {
				t.Errorf("empty slot: %v", err)
			}
			if _, err := d.RequestUnequip("pocket"); code(err) != "slot" {
				t.Errorf("unknown slot: %v", err)
			}
			if _, err := d.RequestUnequip("lhand"); err != nil {
				t.Fatal(err)
			}
			if _, err := d.RequestUnequip("lhand"); code(err) != "queued" {
				t.Errorf("twice: %v", err)
			}
			run(t, d)
			if w.worn["lhand"] != nil || len(w.pack) != 5 {
				t.Errorf("after: %v %d", w.worn, len(w.pack))
			}
			// a full backpack has no room for a 2 high item
			w.pack = nil
			for i := 0; i < PackCols*PackRows/2; i++ {
				g := w.item("ring", uint32(100+i), "Filler", false, "ring", class)
				g.h = 2
				w.pack = append(w.pack, g)
			}
			for i := range w.pack { // lay the fillers on a 10 x 4 grid, 1 x 2 each
				w.pack[i].w, w.pack[i].h = 1, 2
			}
			v := d.View()
			if why := v.Equipment["rhand"].NoUnequip; why == "" {
				t.Errorf("a full pack must refuse: %+v", v.Equipment["rhand"])
			}
		})
	}
}

func TestViewGivesPerSlotWearReasons(t *testing.T) {
	d, w := newGWorld(t, d2.Paladin)
	kit(w, d2.Paladin)
	var staff ItemView
	for _, it := range d.View().Backpack {
		if it.ID == 3 {
			staff = it
		}
	}
	// the war staff: no slot but the right hand, and the shield is worn so even that is refused
	if len(staff.CanWear) != 0 || staff.NoWearSlot["head"] == "" || !strings.Contains(staff.NoWearSlot["rhand"], "two-handed") {
		t.Errorf("staff: %+v", staff)
	}
	w.remove(2)
	for _, it := range d.View().Backpack {
		if it.ID == 3 {
			staff = it
		}
	}
	if len(staff.CanWear) != 1 || staff.CanWear[0] != "rhand" || staff.NoWearSlot["rhand"] != "" {
		t.Errorf("staff with the shield off: %+v", staff)
	}
}

// The state file written before sets and the view of an old reader: nothing moved.
func TestOldStateFileHasNoSetsAndSetsRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), StateFile)
	if err := os.WriteFile(p, []byte(`{"marks":[],"pins":[],"queue":[{"id":3,"slot":"rhand","name":"x","state":"queued","at":1}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	d, err := New(Config{Path: p})
	if err != nil || len(d.st.Sets) != 0 || len(d.st.Queue) != 1 || d.st.Queue[0].Group != "" {
		t.Fatalf("%v %+v", err, d.st)
	}
	b, _ := json.Marshal(d.st)
	if strings.Contains(string(b), "sets") {
		t.Errorf("a state without sets must not write the keys: %s", b)
	}
}

func TestSetsViewGolden(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, w := newGWorld(t, class)
		kit(w, class)
		if _, err := d.SaveSet("Sword and board", ""); err != nil {
			t.Fatal(err)
		}
		gs, _ := d.SaveSet("Spare", "")
		// the hero now holds the war staff (shield and sword in the pack), and the set names a club that is gone
		w.pack = append(w.pack, w.worn["rhand"], w.worn["lhand"])
		delete(w.worn, "lhand")
		w.worn["rhand"] = w.pack[0]
		w.pack = w.pack[1:]
		d.mu.Lock()
		d.st.Sets[1].Slots["head"] = SetItem{ID: 99, Code: "helm", Quality: "magic", Level: 10, Name: "Great Helm", Ident: true}
		d.mu.Unlock()
		pv, err := d.PreviewSet(gs.ID)
		if err != nil {
			t.Fatal(err)
		}
		out := struct {
			Sets    *SetsView  `json:"sets"`
			Preview SetPreview `json:"preview"`
		}{d.View().Sets, pv}
		b, _ := json.MarshalIndent(out, "", " ")
		b = append(b, '\n')
		path := filepath.Join("testdata", "sets_schema7_"+class.String()+".json")
		if os.Getenv("UPDATE") != "" {
			if err := os.WriteFile(path, b, 0o644); err != nil {
				t.Fatal(err)
			}
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if string(want) != string(b) {
			t.Errorf("%v: differs from %s\n got: %s\nwant: %s", class, path, b, want)
		}
	}
}
