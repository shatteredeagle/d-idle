package invdesk

import (
	"context"
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

type fakeMoverVerbs struct {
	caps    []string
	err     error
	calls   []string
	applied func(kind string, id uint32) // mutates the fake world, like the d2d tick
}

func (f *fakeMoverVerbs) Caps(context.Context) ([]string, error) { return f.caps, nil }
func (f *fakeMoverVerbs) ItemMove(_ context.Context, id uint32, panel string, _, _ int) (bool, error) {
	f.calls = append(f.calls, "move "+panel)
	if f.err == nil && f.applied != nil {
		f.applied("move:"+panel, id)
	}
	return false, f.err
}
func (f *fakeMoverVerbs) ItemIdentify(_ context.Context, id uint32) (bool, error) {
	f.calls = append(f.calls, "identify")
	if f.err == nil && f.applied != nil {
		f.applied("identify", id)
	}
	return false, f.err
}
func (f *fakeMoverVerbs) ItemDrop(_ context.Context, id uint32) error {
	f.calls = append(f.calls, "drop")
	if f.err == nil && f.applied != nil {
		f.applied("drop", id)
	}
	return f.err
}
func (f *fakeMoverVerbs) GoldStash(_ context.Context, amount int, deposit bool) (int, int, error) {
	f.calls = append(f.calls, "gold")
	if f.err != nil {
		return 0, 0, f.err
	}
	return 600, 400, nil
}

// world is the snapshot the mover reads back.
type world struct {
	where      d2.Container
	identified bool
	gone       bool
	gold, bank int
}

func (w *world) snap() (*d2.Snapshot, error) {
	it := &d2.Item{Where: w.where}
	if w.identified {
		it.Flags = d2.FlagIdentified
	}
	items := []d2.Unit{}
	if !w.gone {
		items = append(items, d2.Unit{ID: 7, Item: it})
	}
	return &d2.Snapshot{Player: &d2.Player{Gold: w.gold, StashGold: w.bank}, Inventory: &d2.Inventory{Items: items}}, nil
}

func newMover(fv *fakeMoverVerbs, w *world, on bool) *VerbMover {
	m := NewVerbMover(fv, func() bool { return on })
	m.Fresh = w.snap
	m.Pause = func(context.Context, time.Duration) error { return nil }
	m.Settles = 2
	return m
}

func TestMoverVerbFirst(t *testing.T) {
	ctx := context.Background()
	all := []string{CapItemMove, CapItemIdentify, CapItemDrop, CapGoldStash}
	w := &world{where: d2.ContInventory}
	fv := &fakeMoverVerbs{caps: all}
	fv.applied = func(kind string, _ uint32) {
		switch kind {
		case "move:stash":
			w.where = d2.ContStash
		case "identify":
			w.identified = true
		case "drop":
			w.gone = true
		}
	}
	m := newMover(fv, w, true)
	if done, err := m.Move(ctx, 7, true); !done || err != nil || m.HowOf("stash", 7) != HowVerb || m.HowOf("stash", 7) != "" {
		t.Fatalf("move: %v %v", done, err)
	}
	if done, err := m.Identify(ctx, 7); !done || err != nil || m.HowOf("identify", 7) != HowVerb {
		t.Fatalf("identify: %v %v", done, err)
	}
	w.gold, w.bank = 400, 600
	if s, done, err := m.Gold(ctx, 600, true); !done || err != nil || s != "deposited 600 gold" || m.HowOf("gold", 0) != HowVerb {
		t.Fatalf("gold: %q %v %v", s, done, err)
	}
	if done, err := m.Drop(ctx, 7); !done || err != nil || m.HowOf("trash", 7) != HowVerb {
		t.Fatalf("drop: %v %v", done, err)
	}
	// An ok reply the game does not show is an error, never a silent click retry.
	w2 := &world{where: d2.ContInventory}
	if done, err := newMover(&fakeMoverVerbs{caps: all}, w2, true).Move(ctx, 7, true); !done || err == nil {
		t.Fatalf("unconfirmed: %v %v", done, err)
	}
}

func TestMoverFallbacks(t *testing.T) {
	ctx := context.Background()
	all := []string{CapItemMove, CapItemIdentify, CapItemDrop, CapGoldStash}
	w := &world{where: d2.ContInventory}
	// knob off / no capability: not handled, the verbs are not even called.
	for _, c := range []struct {
		on   bool
		caps []string
	}{{false, all}, {true, nil}} {
		fv := &fakeMoverVerbs{caps: c.caps}
		m := newMover(fv, w, c.on)
		if done, _ := m.Move(ctx, 7, true); done || len(fv.calls) != 0 {
			t.Fatalf("on=%v caps=%v: handled %v calls %v", c.on, c.caps, done, fv.calls)
		}
		if _, done, _ := m.Gold(ctx, 5, true); done {
			t.Fatal("gold handled")
		}
	}
	// a rule refusal is final, with how=verb
	fv := &fakeMoverVerbs{caps: all, err: refusal{"far"}}
	m := newMover(fv, w, true)
	done, err := m.Move(ctx, 7, true)
	if !done || err == nil || m.HowOf("stash", 7) != HowVerb {
		t.Fatalf("refusal: %v %v", done, err)
	}
	// a transport error falls back, noted as click
	fv = &fakeMoverVerbs{caps: all, err: errors.New("reset")}
	m = newMover(fv, w, true)
	if done, err := m.Identify(ctx, 7); done || err != nil || m.HowOf("identify", 7) != HowClick {
		t.Fatalf("fallback: %v %v", done, err)
	}
}

type clickTrasher struct{ dropped []uint32 }

func (c *clickTrasher) CheckPlayerTrash(*d2.Snapshot, uint32) (string, error) { return "x", nil }
func (c *clickTrasher) PlayerTrash(_ context.Context, id uint32) error {
	c.dropped = append(c.dropped, id)
	return nil
}

func TestVerbTrasherAndDeskHow(t *testing.T) {
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		d, _, snap := rig(t, class, town1)
		w := &world{where: d2.ContInventory}
		fv := &fakeMoverVerbs{caps: []string{CapItemDrop}}
		fv.applied = func(string, uint32) { w.gone = true }
		m := newMover(fv, w, true)
		click := &clickTrasher{}
		d.SetTrasher(VerbTrasher{Click: click, Mover: m})
		d.SetHowSource(m)
		_ = snap
		// the mover reads item 7; trash item 1 of the rig by pointing the fake world at id 1
		w2 := &worldID{world: w, id: 1}
		m.Fresh = w2.snap
		if _, err := d.RequestTrash(1, true); err != nil {
			t.Fatal(err)
		}
		lines := d.RunPending(context.Background())
		if len(click.dropped) != 0 || len(lines) != 1 {
			t.Fatalf("%s: click %v lines %v", class, click.dropped, lines)
		}
		if recent := d.st.Recent; len(recent) == 0 || recent[0].How != HowVerb {
			t.Fatalf("%s: how not recorded: %+v", class, d.st.Recent)
		}
		// knob off: the click path
		d3, _, _ := rig(t, class, town1)
		m3 := newMover(&fakeMoverVerbs{caps: []string{CapItemDrop}}, w, false)
		click3 := &clickTrasher{}
		d3.SetTrasher(VerbTrasher{Click: click3, Mover: m3})
		d3.RequestTrash(1, true)
		d3.RunPending(context.Background())
		if len(click3.dropped) != 1 {
			t.Fatalf("%s: knob off did not click", class)
		}
	}
}

type worldID struct {
	*world
	id uint32
}

func (w *worldID) snap() (*d2.Snapshot, error) {
	s, _ := w.world.snap()
	for i := range s.Inventory.Items {
		s.Inventory.Items[i].ID = w.id
	}
	return s, nil
}
