package town_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/sim"
)

// Data-free tests of the use-item portal cast (input layer I3); portalverb_test.go runs the same through the sim rig
// where the game tables exist.

type useStub struct {
	input.Controller
	use  func(id uint32) error
	used []uint32
}

func (u *useStub) UseItem(id uint32) error { u.used = append(u.used, id); return u.use(id) }

var tpScroll = d2.Unit{ID: 0x90002, Type: d2.TypeItem, Item: &d2.Item{Where: d2.ContInventory}}

// useReader has no stat lists: a scroll counts as one cast.
type useReader struct{ stubReader }

func (useReader) UnitStats(uint32) (d2.Stats, error) { return nil, errors.New("no stat list") }

type useCase struct {
	t    *town.Town
	snap *d2.Snapshot
	pl   *input.Planner
}

func useTown(cls d2.Class, on bool, caps ...string) (*useCase, *useStub) {
	snap := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: cls, Level: 12, Area: 2, X: 100, Y: 100}, Inventory: &d2.Inventory{}}
	u := &useStub{}
	tune := env.DefaultTuning()
	e := &env.Env{ID: "t", Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Tune: &tune, In: u, Read: useReader{stubReader{snap: snap}},
		Clock: &sim.Clock{T: time.Unix(1000, 0)}}
	e.Planner = &input.Planner{Enabled: func() bool { return on }, Caps: &input.BackendCaps{Verbs: caps}}
	return &useCase{t: town.New(e, nil), snap: snap, pl: e.Planner}, u
}

func TestPortalByUseVerb(t *testing.T) {
	ctx := context.Background()
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		c, u := useTown(cls, true, input.CapUseItem)
		u.use = func(uint32) error {
			c.snap.Objects = append(c.snap.Objects, d2.Unit{ID: 5, Type: d2.TypeObject, Class: 59, X: 102, Y: 100})
			return nil
		}
		p, handled, err := c.t.PortalByVerbForTest(ctx, tpScroll)
		if !handled || err != nil || p.ID != 5 || len(u.used) != 1 || u.used[0] != tpScroll.ID {
			t.Fatalf("%v: portal %v handled %v err %v used %v", cls, p.ID, handled, err, u.used)
		}
		if v, k, cl, _ := c.pl.Counts(); v != 1 || k+cl != 0 {
			t.Fatalf("%v: counts %d/%d/%d", cls, v, k, cl)
		}

		// knob off, no capability: the verb is not asked and the click path stays the caller's
		for name, on := range map[string]bool{"knob off": false, "no cap": true} {
			caps := []string{input.CapUseItem}
			if name == "no cap" {
				caps = nil
			}
			c, u := useTown(cls, on, caps...)
			u.use = func(uint32) error { t.Fatalf("%v %s: the verb was asked", cls, name); return nil }
			if _, handled, err := c.t.PortalByVerbForTest(ctx, tpScroll); handled || err != nil {
				t.Fatalf("%v %s: handled %v err %v", cls, name, handled, err)
			}
		}

		// a send error (older build) hands back to the click; a sent verb with no portal is an error, not a click
		c, u = useTown(cls, true, input.CapUseItem)
		u.use = func(uint32) error { return errors.New("d2d: unknown cmd 'use'") }
		if _, handled, err := c.t.PortalByVerbForTest(ctx, tpScroll); handled || err != nil {
			t.Fatalf("%v: send error: handled %v err %v", cls, handled, err)
		}
		c, u = useTown(cls, true, input.CapUseItem)
		u.use = func(uint32) error { return nil }
		if _, handled, err := c.t.PortalByVerbForTest(ctx, tpScroll); !handled || err == nil {
			t.Fatalf("%v: no effect: handled %v err %v", cls, handled, err)
		}
	}
}
