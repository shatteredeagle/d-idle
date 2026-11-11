package invdesk

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// fakeTr drops pack items: the item leaves the snapshot, like the real path (pick up, ground click, prove).
type fakeTr struct {
	snap    *d2.Snapshot
	dropped []uint32
	err     error
	during  func() // runs while the drop is "in progress" (the request shows doing)
}

func (f *fakeTr) CheckPlayerTrash(s *d2.Snapshot, id uint32) (string, error) {
	for _, u := range s.Inventory.Items {
		if u.ID == id {
			return "thing", nil
		}
	}
	return "", errors.New("not in the backpack")
}

func (f *fakeTr) PlayerTrash(ctx context.Context, id uint32) error {
	if f.during != nil {
		f.during()
	}
	if f.err != nil {
		return f.err
	}
	f.dropped = append(f.dropped, id)
	return nil
}

func trashRig(t *testing.T, class d2.Class, area int) (*Desk, *fakeTr, *d2.Snapshot) {
	t.Helper()
	d, _, snap := rig(t, class, area)
	tr := &fakeTr{snap: snap}
	d.SetTrasher(tr)
	d.SetTrashBlock(func(u d2.Unit) string {
		if u.Class == 99 {
			return "quest item"
		}
		return ""
	})
	return d, tr, snap
}

func code(err error) string {
	var r *Refusal
	if errors.As(err, &r) {
		return r.Kind()
	}
	if err != nil {
		return "other:" + err.Error()
	}
	return ""
}

func TestTrashRefusals(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			d, _, _ := trashRig(t, class, field)
			for _, c := range []struct {
				id      uint32
				confirm bool
				want    string
			}{
				{1, false, "confirm"}, // the page asks twice
				{3, true, "place"},    // stash
				{4, true, "never"},    // quest item
				{5, true, "equipped"}, // worn
				{77, true, "item"},    // gone
			} {
				if _, err := d.RequestTrash(c.id, c.confirm); code(err) != c.want {
					t.Errorf("item %d: %v, want %s", c.id, err, c.want)
				}
			}
			if err := d.SetMark(1, MarkKeep); err != nil {
				t.Fatal(err)
			}
			if _, err := d.RequestTrash(1, true); code(err) != "keep" {
				t.Errorf("keep mark: %v", err)
			}
			if d.Pending() != 0 {
				t.Error("a refusal queued something")
			}
		})
	}
}

func TestTrashRunsDropsAndIgnores(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		t.Run(class.String(), func(t *testing.T) {
			d, tr, snap := trashRig(t, class, field)
			snap.Player.X, snap.Player.Y = 100, 200
			if _, err := d.RequestTrash(2, true); err != nil {
				t.Fatal(err)
			}
			var seen string
			tr.during = func() { seen = d.View().Queue[0].State }
			lines := d.RunPending(context.Background())
			if len(lines) != 1 || len(tr.dropped) != 1 || tr.dropped[0] != 2 || seen != StateDoing {
				t.Fatalf("lines %v dropped %v state while running %q", lines, tr.dropped, seen)
			}
			v := d.View()
			if len(v.Queue) != 0 || v.Recent[0].State != StateDone || v.Recent[0].Kind != KindTrash {
				t.Fatalf("view %+v", v)
			}
			on := d2.Unit{ID: 2, Class: 11, X: 101, Y: 201, Item: &d2.Item{Quality: 2}}
			if !d.Ignored(on) {
				t.Error("the trashed item is not ignored by id")
			}
			moved := on
			moved.ID = 900 // a new id after the drop: the fingerprint near the drop point still matches
			if !d.Ignored(moved) {
				t.Error("not ignored by fingerprint near the drop point")
			}
			far := moved
			far.X = 400
			other := on
			other.Class = 10
			if d.Ignored(far) || d.Ignored(other) {
				t.Error("ignore is too wide")
			}
			// the list survives a restart and ends after the TTL
			d2x, err := New(Config{Path: d.c.Path, Now: func() time.Time { return time.Now().Add(TrashTTL - time.Minute) }})
			if err != nil || !d2x.Ignored(on) {
				t.Fatalf("reload: %v", err)
			}
			late, _ := New(Config{Path: d.c.Path, Now: func() time.Time { return time.Now().Add(TrashTTL + time.Minute) }})
			if late.Ignored(on) {
				t.Error("ignore outlived its lifetime")
			}
		})
	}
}

func TestTrashFailureAndKeepAfterQueueing(t *testing.T) {
	d, tr, _ := trashRig(t, d2.Paladin, town1)
	if _, err := d.RequestTrash(1, true); err != nil {
		t.Fatal(err)
	}
	if err := d.SetMark(1, MarkKeep); err != nil { // marked keep while it waited
		t.Fatal(err)
	}
	d.RunPending(context.Background())
	v := d.View()
	if len(tr.dropped) != 0 || v.Recent[0].State != StateFailed || v.Recent[0].Reason == "" {
		t.Fatalf("keep mark did not stop the trash: %+v", v.Recent)
	}
	if err := d.SetMark(1, ""); err != nil {
		t.Fatal(err)
	}
	if _, err := d.RequestTrash(1, true); err != nil {
		t.Fatal(err)
	}
	tr.err = errors.New("no free spot")
	d.RunPending(context.Background())
	if r := d.View().Recent[0]; r.State != StateFailed || r.Reason != "no free spot" || d.Ignored(d2.Unit{ID: 1, Class: 10, Item: &d2.Item{Quality: 4, ILvl: 5}}) {
		t.Fatalf("failed trash: %+v", r)
	}
}

func TestCancelBoundAndPersistence(t *testing.T) {
	d, _, _ := trashRig(t, d2.Paladin, field)
	if _, err := d.RequestEquip(1, "rhand"); err != nil {
		t.Fatal(err)
	}
	if _, err := d.RequestTrash(1, true); err != nil || d.Pending() != 1 { // the trash replaces the queued equip
		t.Fatalf("trash over equip: %v pending %d", err, d.Pending())
	}
	if _, err := d.RequestEquip(1, "rhand"); code(err) != "trash" {
		t.Errorf("equip of a trash-queued item: %v", err)
	}
	// reload: the queue is persisted
	again, err := New(Config{Path: d.c.Path})
	if err != nil || again.Pending() != 1 {
		t.Fatalf("queue lost on reload: %v", err)
	}
	if err := d.Cancel(1, KindTrash); err != nil || d.Pending() != 0 {
		t.Fatalf("cancel: %v", err)
	}
	if r := d.View().Recent[0]; r.State != StateCancelled {
		t.Errorf("recent %+v", r)
	}
	if code(d.Cancel(1, KindTrash)) != "item" {
		t.Error("second cancel")
	}
	// the bound: MaxQueue pending, then "full"
	for i := 0; i < MaxQueue; i++ {
		d.st.Queue = append(d.st.Queue, Request{ID: uint32(100 + i), Kind: KindTrash, State: StateQueued})
	}
	if _, err := d.RequestTrash(2, true); code(err) != "full" {
		t.Errorf("bound: %v", err)
	}
}

func TestDoingCannotBeCancelledAndRestartFailsIt(t *testing.T) {
	d, tr, _ := trashRig(t, d2.Sorceress, field)
	if _, err := d.RequestTrash(2, true); err != nil {
		t.Fatal(err)
	}
	var cancel string
	tr.during = func() {
		cancel = code(d.Cancel(2, KindTrash))
		// the bot dies here: the state file says doing
		d2x, err := New(Config{Path: d.c.Path})
		if err != nil {
			t.Error(err)
			return
		}
		v := d2x.View()
		if len(v.Queue) != 0 || len(v.Recent) == 0 || v.Recent[0].State != StateFailed {
			t.Errorf("restart left %+v", v)
		}
	}
	d.RunPending(context.Background())
	if cancel != "busy" {
		t.Errorf("cancel while doing: %q", cancel)
	}
}

func TestAutoSlotRings(t *testing.T) {
	d, eq, snap := rig(t, d2.Paladin, town1)
	eq.allowed[d2.Paladin][50] = true
	snap.Inventory.Items = append(snap.Inventory.Items, d2.Unit{ID: 50, Class: 50, Item: &d2.Item{Where: d2.ContInventory}})
	ring := &ringEq{fakeEq: eq}
	d.SetEquipper(ring)
	r, err := d.RequestEquip(50, "")
	if err != nil || r.Slot != "rring" {
		t.Fatalf("both free: %v %+v", err, r)
	}
	snap.Inventory.Items = append(snap.Inventory.Items, d2.Unit{ID: 51, Class: 50, Item: &d2.Item{Where: d2.ContEquipment, Body: d2.BodyRRing}})
	if r, err = d.RequestEquip(50, ""); err != nil || r.Slot != "lring" {
		t.Fatalf("right worn: %v %+v", err, r)
	}
	snap.Inventory.Items = append(snap.Inventory.Items, d2.Unit{ID: 52, Class: 50, Item: &d2.Item{Where: d2.ContEquipment, Body: d2.BodyLRing}})
	if _, err = d.RequestEquip(50, ""); code(err) != "choose" {
		t.Fatalf("both worn: %v", err)
	}
	if r, err = d.RequestEquip(50, "lring"); err != nil || r.Slot != "lring" {
		t.Fatalf("named: %v", err)
	}
	if _, err = d.RequestEquip(2, ""); code(err) != "requirements" {
		t.Fatalf("no slot at all: %v", err)
	}
}

// ringEq lets class-50 items go on either ring finger.
type ringEq struct{ *fakeEq }

func (r *ringEq) CheckPlayerEquip(s *d2.Snapshot, id uint32, body int) (string, error) {
	for _, u := range s.Inventory.Items {
		if u.ID == id && u.Class == 50 {
			if body == d2.BodyRRing || body == d2.BodyLRing {
				return "ring", nil
			}
			return "ring", errors.New("does not fit that slot")
		}
	}
	return r.fakeEq.CheckPlayerEquip(s, id, body)
}

func TestViewExplainsMissingActions(t *testing.T) {
	d, _, _ := trashRig(t, d2.Paladin, town1)
	v := d.View()
	if v.Backpack[2].NoTrash == "" || v.Backpack[2].NoSell == "" {
		t.Errorf("quest-like item must say why: %+v", v.Backpack[2])
	}
	if v.Backpack[0].NoTrash != "" || v.Backpack[0].NoSell != "" || v.Backpack[0].NoWear != "" {
		t.Errorf("sword offers everything: %+v", v.Backpack[0])
	}
	if v.Backpack[1].NoWear == "" {
		t.Errorf("a Paladin cannot wear item 2: %+v", v.Backpack[1])
	}
	_ = filepath.Join
}
