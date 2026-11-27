package town_test

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// useCounter offers the use-item verb (input layer I3) and counts the bot's right-clicks. effect false is a build that
// takes the verb and opens nothing.
type useCounter struct {
	input.Controller
	r      *rig
	effect bool
	fail   error
	used   int
	rights int
}

func (c *useCounter) Click(b input.Button, x, y int, mods ...input.Mod) error {
	if b == input.Right {
		c.rights++
	}
	return c.Controller.Click(b, x, y, mods...)
}

func (c *useCounter) UseItem(id uint32) error {
	c.used++
	if c.fail != nil {
		return c.fail
	}
	if !c.effect {
		return nil
	}
	w := c.r.W
	for i, u := range w.Items {
		if u.ID == id {
			w.Items = append(w.Items[:i], w.Items[i+1:]...)
			w.AddObject(59, w.Player.X+2, w.Player.Y)
			return nil
		}
	}
	return errors.New("no such item")
}

func useRig(t *testing.T, cls d2.Class, on bool, caps ...string) (*rig, *useCounter) {
	t.Helper()
	r := earlyRig(t, cls, 14)
	r.W.Player.Area, r.W.PortalLevel = 2, 1
	r.W.Add("tsc", d2.ContInventory, 0, 0)
	c := &useCounter{Controller: r.Env.In, r: r, effect: true}
	r.Env.In = c
	r.Env.Planner = &input.Planner{Enabled: func() bool { return on }, Caps: &input.BackendCaps{Verbs: caps}}
	return r, c
}

func TestPortalUseVerbReplacesThePackClick(t *testing.T) {
	for _, cls := range []d2.Class{d2.Sorceress, d2.Paladin} {
		r, c := useRig(t, cls, true, input.CapUseItem)
		if err := r.T.ToTown(context.Background()); err != nil {
			t.Fatalf("%v: %v", cls, err)
		}
		if c.used != 1 || c.rights != 0 || r.W.Player.Area != 1 || r.W.UI[d2.UIInventory] != 0 {
			t.Fatalf("%v: used %d rights %d area %d", cls, c.used, c.rights, r.W.Player.Area)
		}
		if v, _, _, _ := r.Env.Planner.Counts(); v != 1 {
			t.Fatalf("%v: verb not counted", cls)
		}
	}
}

func TestPortalUseVerbOffOrMissingKeepsTheClick(t *testing.T) {
	for _, tc := range []struct {
		on   bool
		caps []string
		fail error
	}{{false, []string{input.CapUseItem}, nil}, {true, nil, nil}, {true, []string{input.CapUseItem}, errors.New("d2d: unknown cmd 'use'")}} {
		r, c := useRig(t, d2.Necromancer, tc.on, tc.caps...)
		c.fail = tc.fail
		if err := r.T.ToTown(context.Background()); err != nil {
			t.Fatalf("%+v: %v", tc, err)
		}
		if c.rights != 1 || r.W.Player.Area != 1 {
			t.Fatalf("%+v: used %d rights %d area %d", tc, c.used, c.rights, r.W.Player.Area)
		}
	}
}

func TestPortalUseVerbNoEffectIsNotClicked(t *testing.T) {
	r, c := useRig(t, d2.Barbarian, true, input.CapUseItem)
	c.effect = false
	if err := r.T.OpenPortalForTest(context.Background()); err == nil {
		t.Fatal("want an error")
	}
	if c.used != 1 || c.rights != 0 {
		t.Fatalf("used %d rights %d", c.used, c.rights)
	}
}
