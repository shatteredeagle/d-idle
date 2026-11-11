package invdesk

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/status"
)

// fakeEq is a class table: which item classes (unit Class numbers) each hero class may wear, and which slot they fit.
type fakeEq struct {
	class   d2.Class
	allowed map[d2.Class]map[int]bool
	slot    map[int]int
	worn    []uint32
	busy    bool
}

func (f *fakeEq) CheckPlayerEquip(s *d2.Snapshot, id uint32, body int) (string, error) {
	for _, u := range s.Inventory.Items {
		if u.ID != id {
			continue
		}
		if !f.allowed[s.Player.Class][u.Class] {
			return "thing", errors.New("restricted to another class")
		}
		if f.slot[u.Class] != body {
			return "thing", errors.New("does not fit that slot")
		}
		return "thing", nil
	}
	return "", errors.New("not in the backpack")
}

func (f *fakeEq) PlayerEquip(ctx context.Context, id uint32, body int) error {
	if f.busy {
		return errors.New("leveler: a fight is on")
	}
	f.worn = append(f.worn, id)
	return nil
}

func rig(t *testing.T, class d2.Class, area int) (*Desk, *fakeEq, *d2.Snapshot) {
	t.Helper()
	snap := &d2.Snapshot{Player: &d2.Player{Class: class, Area: area}, Inventory: &d2.Inventory{Items: []d2.Unit{
		{ID: 1, Class: 10, Item: &d2.Item{Where: d2.ContInventory, Quality: 4, ILvl: 5}}, // a sword
		{ID: 2, Class: 11, Item: &d2.Item{Where: d2.ContInventory, Quality: 2}},          // a wand-like item for casters
		{ID: 3, Class: 12, Item: &d2.Item{Where: d2.ContStash}},                          // stash item
		{ID: 4, Class: 99, Item: &d2.Item{Where: d2.ContInventory}},                      // quest-like item
		{ID: 5, Class: 13, Item: &d2.Item{Where: d2.ContEquipment, Body: d2.BodyTorso}},  // worn armor
	}}}
	eq := &fakeEq{allowed: map[d2.Class]map[int]bool{
		d2.Paladin:   {10: true},
		d2.Sorceress: {11: true},
	}, slot: map[int]int{10: d2.BodyRHand, 11: d2.BodyRHand}}
	d, err := New(Config{
		Path: filepath.Join(t.TempDir(), StateFile),
		Snap: func() *d2.Snapshot { return snap },
		Inventory: func() status.InventoryData {
			return status.InventoryData{Inventory: []status.Item{{ID: 1, Code: "10"}, {ID: 2, Code: "11"}, {ID: 4}}}
		},
		SellBlock: func(u d2.Unit) string {
			if u.Class == 99 {
				return "quest item"
			}
			return ""
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	d.SetEquipper(eq)
	return d, eq, snap
}

const town1, field = 1, 2 // Rogue Encampment, Blood Moor

func TestMarksAndRefusals(t *testing.T) {
	d, _, _ := rig(t, d2.Paladin, town1)
	u1 := d2.Unit{ID: 1, Class: 10, Item: &d2.Item{Quality: 4, ILvl: 5}}
	if err := d.SetMark(1, MarkSell); err != nil {
		t.Fatal(err)
	}
	if got := d.MarkOf(u1); got != MarkSell {
		t.Errorf("mark %q", got)
	}
	// a reused id of another item does not inherit the mark
	if got := d.MarkOf(d2.Unit{ID: 1, Class: 77, Item: &d2.Item{}}); got != "" {
		t.Errorf("foreign item got mark %q", got)
	}
	for _, c := range []struct {
		id   uint32
		mark string
		kind string
	}{{3, MarkSell, "place"}, {4, MarkSell, "never"}, {1, "dump", "mark"}, {42, MarkKeep, "item"}} {
		err := d.SetMark(c.id, c.mark)
		var r *Refusal
		if !errors.As(err, &r) || r.Kind() != c.kind {
			t.Errorf("SetMark(%d,%q) = %v, want refusal %s", c.id, c.mark, err, c.kind)
		}
	}
	if err := d.SetMark(3, MarkKeep); err != nil { // keep works anywhere
		t.Error(err)
	}
	if err := d.SetMark(1, ""); err != nil || d.MarkOf(u1) != "" {
		t.Errorf("clear: %v", err)
	}
}

func TestEquipRequestPinsAndOnlyUnpinEnds(t *testing.T) {
	for _, c := range []struct {
		class     d2.Class
		good, bad uint32
	}{{d2.Paladin, 1, 2}, {d2.Sorceress, 2, 1}} {
		t.Run(c.class.String(), func(t *testing.T) {
			d, eq, _ := rig(t, c.class, town1)
			var r *Refusal
			if _, err := d.RequestEquip(c.bad, "rhand"); !errors.As(err, &r) || r.Kind() != "requirements" {
				t.Fatalf("other class's item: %v", err)
			}
			if _, err := d.RequestEquip(c.good, "head"); !errors.As(err, &r) || r.Kind() != "requirements" {
				t.Fatalf("wrong slot: %v", err)
			}
			if _, err := d.RequestEquip(c.good, "elbow"); !errors.As(err, &r) || r.Kind() != "slot" {
				t.Fatalf("unknown slot: %v", err)
			}
			if _, err := d.RequestEquip(3, "rhand"); !errors.As(err, &r) || r.Kind() != "place" {
				t.Fatalf("stash item: %v", err)
			}
			if _, err := d.RequestEquip(c.good, "rhand"); err != nil {
				t.Fatal(err)
			}
			if d.Pending() != 1 || d.Pinned(d2.BodyRHand) {
				t.Fatal("queued but not yet pinned")
			}
			if lines := d.RunPending(context.Background()); len(lines) != 1 || len(eq.worn) != 1 || eq.worn[0] != c.good {
				t.Fatalf("run: %v worn %v", lines, eq.worn)
			}
			if !d.Pinned(d2.BodyRHand) || d.Pending() != 0 {
				t.Fatal("slot not pinned after the equip")
			}
			// a pin never expires by itself: more queue runs, reloads and views leave it
			d.RunPending(context.Background())
			d2x, err := New(Config{Path: d.c.Path})
			if err != nil || !d2x.Pinned(d2.BodyRHand) {
				t.Fatalf("pin lost on reload: %v", err)
			}
			if err := d.Unpin("lhand"); !errors.As(err, &r) || r.Kind() != "pin" {
				t.Errorf("unpin of a free slot: %v", err)
			}
			if err := d.Unpin("rhand"); err != nil || d.Pinned(d2.BodyRHand) {
				t.Errorf("unpin: %v", err)
			}
		})
	}
}

func TestEquipQueuesInTheFieldAndFailureDoesNotPin(t *testing.T) {
	d, eq, _ := rig(t, d2.Paladin, field)
	if _, err := d.RequestEquip(1, "rhand"); err != nil { // the open opportunity is the bot's call, not a refusal
		t.Fatal(err)
	}
	eq.busy = true // a fight: the request waits, nothing pinned
	if lines := d.RunPending(context.Background()); len(lines) != 1 || d.Pending() != 1 || d.Pinned(d2.BodyRHand) {
		t.Fatalf("busy: %v pending %d", lines, d.Pending())
	}
	eq.busy = false
	if len(d.RunPending(context.Background())) != 1 || !d.Pinned(d2.BodyRHand) {
		t.Fatal("retry did not equip")
	}
}

func TestViewListsMarksPinsAndWearableSlots(t *testing.T) {
	d, _, _ := rig(t, d2.Sorceress, town1)
	if err := d.SetMark(2, MarkKeep); err != nil {
		t.Fatal(err)
	}
	v := d.View()
	if !v.InTown || v.Class != "Sorceress" || len(v.Backpack) != 3 {
		t.Fatalf("view %+v", v)
	}
	if v.Backpack[1].Mark != MarkKeep || len(v.Backpack[1].CanWear) != 1 || v.Backpack[1].CanWear[0] != "rhand" {
		t.Errorf("item 2: %+v", v.Backpack[1])
	}
	if len(v.Backpack[0].CanWear) != 0 {
		t.Errorf("a Sorceress may not wear item 1: %+v", v.Backpack[0].CanWear)
	}
}

// The tooltip fields of status.Item pass through the view untouched (schema 2, additive): an item that carries none of
// them marshals without them, one that does carries them for both classes' desks.
func TestViewCarriesTooltipFields(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		d, _, _ := rig(t, class, town1)
		d.c.Inventory = func() status.InventoryData {
			return status.InventoryData{Inventory: []status.Item{
				{ID: 1, Code: "10", Tip: []string{"Sword", "One-Hand Damage: 2 to 4"}, Sell: 20, Base: &status.ItemBase{Name: "Sword", MinDam: 2, MaxDam: 4}},
				{ID: 2, Code: "11"},
			}}
		}
		v := d.View()
		if v.Schema != ViewSchema {
			t.Errorf("%v: schema %d", class, v.Schema)
		}
		b, err := json.Marshal(v.Backpack)
		if err != nil {
			t.Fatal(err)
		}
		var items []map[string]any
		if err := json.Unmarshal(b, &items); err != nil {
			t.Fatal(err)
		}
		if _, ok := items[0]["tip"]; !ok || items[0]["sell"] != float64(20) || items[0]["base"] == nil {
			t.Errorf("%v: item 1 lost its tooltip fields: %v", class, items[0])
		}
		for _, k := range []string{"tip", "base", "sell"} {
			if _, ok := items[1][k]; ok {
				t.Errorf("%v: item 2 must not carry %s", class, k)
			}
		}
	}
}

// The reason source is bound once per View, not once per item (TASK-268: a per-item game read cost about 1 s).
func TestWhyForBuiltOncePerView(t *testing.T) {
	d, _, _ := rig(t, d2.Paladin, town1)
	built, judged := 0, 0
	d.SetWhyFor(func(s *d2.Snapshot) func(u d2.Unit) string {
		built++
		return func(u d2.Unit) string { judged++; return "why" }
	})
	v := d.View()
	if built != 1 || judged < 3 || v.Backpack[0].Why != "why" {
		t.Errorf("built %d judged %d why %q", built, judged, v.Backpack[0].Why)
	}
	d.View()
	if built != 2 {
		t.Errorf("built %d after two views", built)
	}
}

// The merc block (schema 3): its four states, the legal slots per kind, and that his items skip marks and pins by id
// (their own id space). Both hero classes, since the block must not depend on the player's class.
func TestViewMercBlock(t *testing.T) {
	cap := &status.Item{ID: 1, Code: "5", Name: "Cap", Quality: "normal", W: 2, H: 2} // same id as a pack item on purpose
	cases := []struct {
		name  string
		m     *status.MercInv
		state string
		slots int
		items int
	}{
		{"backend cannot read", nil, "", 0, 0},
		{"nobody hired", &status.MercInv{}, MercNone, 0, 0},
		{"desert ready", &status.MercInv{Hired: true, Alive: true, Class: merc.ClassDesert, Level: 17, GearKnown: true, Equipment: map[string]*status.Item{"mhead": cap}}, MercReady, 3, 1},
		{"ironwolf ready", &status.MercInv{Hired: true, Alive: true, Class: merc.ClassIronWolf, GearKnown: true}, MercReady, 4, 0},
		{"rogue dead", &status.MercInv{Hired: true, Class: merc.ClassRogue, GearKnown: true, Equipment: map[string]*status.Item{"mhead": cap}}, MercDead, 3, 1},
		{"gear unread", &status.MercInv{Hired: true, Alive: true, Class: merc.ClassRogue}, MercUnknown, 3, 0},
	}
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, c := range cases {
			d, _, _ := rig(t, class, town1)
			m := c.m
			d.c.Inventory = func() status.InventoryData {
				return status.InventoryData{Inventory: []status.Item{{ID: 1, Code: "10"}}, Merc: m}
			}
			v := d.View()
			if c.m == nil {
				if v.Merc != nil {
					t.Errorf("%v %s: want no merc block", class, c.name)
				}
				continue
			}
			if v.Merc == nil || v.Merc.State != c.state || len(v.Merc.Slots) != c.slots || len(v.Merc.Equipment) != c.items {
				t.Fatalf("%v %s: %+v", class, c.name, v.Merc)
			}
			if h := v.Merc.Equipment["mhead"]; h != nil && (h.Mark != "" || h.CanSell || len(h.CanWear) != 0 || h.Name != "Cap") {
				t.Errorf("%v %s: merc item decorated like a pack item: %+v", class, c.name, h)
			}
			if _, err := json.Marshal(v); err != nil {
				t.Fatal(err)
			}
		}
	}
}
