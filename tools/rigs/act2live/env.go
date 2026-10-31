//go:build rigs

package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/town/hand"
)

// rig is the live Act 2 driver: it implements task.StallRecoverer (ClearAround = combat.Clear, TeleportToward = the
// exitto logic) and, embedded in a2env, the parts of quest.Act2Env the tomb and boss legs need.
type rig struct {
	e     *env.Env
	f     *combat.Fighter
	nv    *nav.Navigator
	game  *input.Game
	probe *nav.FarProbe
}

var _ task.StallRecoverer = (*rig)(nil)

// exitTarget is where the character has to stand to cross into dest: an exit of the level (V live: tile exits and border
// gaps), else the middle of the Adjacent contact (V live nav3: teleporting there and calling ToArea crosses in seconds).
func exitTarget(lm *d2map.LevelMap, dest int) *d2map.Exit {
	if lm == nil {
		return nil
	}
	for i := range lm.Exits {
		if lm.Exits[i].Dest == dest {
			return &lm.Exits[i]
		}
	}
	for _, a := range lm.Adjacent {
		if a.LevelNo == dest {
			c := a.Contact
			return &d2map.Exit{Dest: dest, Pos: d2map.Point{X: c.X + c.W/2, Y: c.Y + c.H/2}}
		}
	}
	return nil
}

func (r *rig) ClearAround(ctx context.Context) error {
	_, err := r.f.Clear(ctx, combat.ClearOpts{Radius: 40, Timeout: 20 * time.Second})
	return err
}

func (r *rig) TeleportToward(ctx context.Context, a task.Area) error {
	_, _ = r.e.World.Update()
	ex := exitTarget(r.e.World.Current(), int(a))
	if ex == nil {
		return task.ErrNoTeleport
	}
	syncTele(r.e, r.nv)
	cc, cancel := context.WithTimeout(ctx, 40*time.Second) // the navigator hops (TeleportKey) along the way
	defer cancel()
	return r.nv.ToArea(cc, int(a))
}

// reach gets into area lv: clear, hop to the exit when known, else explore; bounded by tries.
func (r *rig) reach(ctx context.Context, lv int) error {
	for try := 0; try < 30 && ctx.Err() == nil; try++ {
		s, err := r.e.Snapshot()
		if err != nil {
			return err
		}
		if s.Player.Area == lv {
			return nil
		}
		_ = r.ClearAround(ctx)
		if err := r.TeleportToward(ctx, task.Area(lv)); err == nil {
			continue
		} else if !errors.Is(err, task.ErrNoTeleport) {
			fmt.Println("  hop:", err)
		}
		syncTele(r.e, r.nv)
		cc, cancel := context.WithTimeout(ctx, 12*time.Second)
		fmt.Println("  Explore:", r.nv.Explore(cc, func() bool { return false }))
		cancel()
	}
	return fmt.Errorf("could not reach area %d", lv)
}

// a2env: quest.Act2Env over the rig. Only the methods the live legs call are real; the embedded nil interface makes
// any other call panic loudly instead of pretending.
type a2env struct {
	quest.Act2Env
	*rig
}

func (a a2env) Area() int {
	s, err := a.e.Snapshot()
	if err != nil || s.Player == nil {
		return 0
	}
	return s.Player.Area
}

// StaffTombLevel reads ActMisc.dwStaffTombLevel (S offsets, U on 1.14d: compare with the tomb that holds object 152).
func (a a2env) StaffTombLevel(context.Context) (int, bool) {
	s, err := a.e.Snapshot()
	if err != nil || s.Player == nil {
		return 0, false
	}
	lv, err := a.e.Read.StaffTombLevel(s.Player.Addr)
	return lv, err == nil && lv != 0
}

// ProbeFar takes one step of the rig's far probe (kept across calls so visited points persist).
func (a a2env) ProbeFar(ctx context.Context) (bool, error) {
	if a.rig.probe == nil {
		a.rig.probe = a.rig.nv.NewFarProbe()
	}
	return a.rig.probe.Step(ctx)
}

func (a a2env) GoArea(ctx context.Context, area int) error { return a.rig.reach(ctx, area) }

// HasObject answers from presets plus the units in view (V live: the orifice class 152 is a preset object of its tomb).
func (a a2env) HasObject(_ context.Context, class int) bool {
	_, _ = a.e.World.Update()
	if lm := a.e.World.Current(); lm != nil {
		for _, p := range lm.Presets {
			if p.Kind == d2map.PresetObject && p.ID == class {
				return true
			}
		}
	}
	if s, err := a.e.Snapshot(); err == nil {
		for _, o := range s.Objects {
			if o.Class == class {
				return true
			}
		}
	}
	return false
}

// objectUnit finds the nearest live object of the class: a unit in view, else walks to the preset and looks again.
func (a a2env) objectUnit(ctx context.Context, class int) (*d2.Unit, error) {
	for try := 0; try < 3; try++ {
		s, err := a.e.Snapshot()
		if err != nil {
			return nil, err
		}
		var best *d2.Unit
		for i := range s.Objects {
			if o := &s.Objects[i]; o.Class == class && (best == nil || dist(s, o) < dist(s, best)) {
				best = o
			}
		}
		if best != nil {
			return best, nil
		}
		if lm := a.e.World.Current(); lm != nil {
			for _, p := range lm.Presets {
				if p.Kind == d2map.PresetObject && p.ID == class {
					fmt.Println("  walking to preset", p.Pos)
					_ = a.nv.GoTo(ctx, input.Point{X: p.Pos.X, Y: p.Pos.Y})
					break
				}
			}
		}
		_, _ = a.e.World.Update()
	}
	return nil, fmt.Errorf("object class %d not found", class)
}

// PlaceItem puts the pack item with the given code into the object (the Horadric Staff into the orifice). V live
// (ACT2LIVE6): a click with the item on the cursor needs the object's screen point, but the game reports no hover with
// an item on the cursor and a blind projected click dropped the Staff, so the point is found first (findHover), then
// the item is taken to the cursor and that point clicked. U: the orifice accepting the item.
func (a a2env) PlaceItem(ctx context.Context, code string, class int) error {
	o, err := a.objectUnit(ctx, class)
	if err != nil {
		return err
	}
	fmt.Println("GoToUnit:", a.nv.GoToUnitOf(ctx, d2.TypeObject, o.ID))
	_ = a.game.CloseAll() // V: ui flag 0xE (unknown) shows up next to the orifice
	time.Sleep(400 * time.Millisecond)
	px, py, ok := a.findHover(o)
	if !ok {
		return errors.New("the orifice is never hovered (monsters over it?)")
	}
	h := hand.New(a.e)
	sn, err := h.OpenInventory(ctx)
	if err != nil {
		_ = a.game.CloseAll()
		time.Sleep(400 * time.Millisecond)
		if sn, err = h.OpenInventory(ctx); err != nil {
			return err
		}
	}
	lay, err := h.Layout(sn)
	if err != nil {
		return err
	}
	taken := false
	for _, it := range sn.Inventory.Container(d2.ContInventory) {
		if baseCode(a.e, it.Class) == code {
			if err := h.Take(ctx, it, lay.Inventory); err != nil {
				return err
			}
			taken = true
			break
		}
	}
	if !taken {
		_ = h.CloseAll(ctx)
		return fmt.Errorf("no %s in the pack", code)
	}
	if err := a.game.CloseAll(); err != nil {
		return err
	}
	time.Sleep(600 * time.Millisecond)
	s, err := a.e.Snapshot()
	if err != nil {
		return err
	}
	fmt.Println("cursor after closing panels:", s.Inventory.Cursor != nil)
	mode := o.Mode
	for _, off := range [][2]int{{0, 0}, {0, -6}, {0, 6}, {-6, 0}, {6, 0}} {
		x, y := px+off[0], py+off[1]
		_ = a.game.MoveMouse(x, y)
		time.Sleep(300 * time.Millisecond)
		_ = a.game.Click(input.Left, x, y)
		time.Sleep(2500 * time.Millisecond)
		s2, err := a.e.Snapshot()
		if err != nil {
			return err
		}
		fmt.Println("click", x, y, "cursor held:", s2.Inventory.Cursor != nil)
		if u := findObj(s2, o.ID); u != nil && u.Mode != mode {
			fmt.Println("object mode changed", mode, "->", u.Mode)
			return nil
		}
		if s2.Inventory.Cursor == nil {
			fmt.Println("cursor empty after the click")
			return nil
		}
	}
	return errors.New("the orifice did not take the item")
}

// findHover scans screen points around the object's projected position for one where the game reports the object as
// hovered. V (ACT2LIVE6): with an item on the cursor the game reports no hover at all, so the point has to be found
// BEFORE the item is taken (the camera does not move while the player stands still).
func (a a2env) findHover(o *d2.Unit) (int, int, bool) {
	if a.game.Hover == nil {
		return 0, 0, false
	}
	vx, vy, err := a.game.R.Viewport()
	if err != nil {
		return 0, 0, false
	}
	sx, sy := a.game.X.WorldToScreenVP(vx, vy, float64(o.X)+0.5, float64(o.Y)+0.5)
	cx, cy := int(sx), int(sy)
	var hits [][2]int
	for dy := -70; dy <= 30; dy += 10 {
		for dx := -60; dx <= 60; dx += 10 {
			x, y := cx+dx, cy+dy
			if !a.game.X.InView(input.Screen{X: x, Y: y}) {
				continue
			}
			_ = a.game.MoveMouse(x, y)
			time.Sleep(90 * time.Millisecond)
			if _, id, err := a.game.Hover(); err == nil && id == o.ID {
				hits = append(hits, [2]int{x, y})
			}
		}
	}
	fmt.Println("hover scan: projected", cx, cy, "hits", len(hits))
	if len(hits) == 0 {
		return 0, 0, false
	}
	var mx, my int
	for _, h := range hits {
		mx += h[0]
		my += h[1]
	}
	return mx / len(hits), my / len(hits), true
}

func findObj(s *d2.Snapshot, id uint32) *d2.Unit {
	for i := range s.Objects {
		if s.Objects[i].ID == id {
			return &s.Objects[i]
		}
	}
	return nil
}
