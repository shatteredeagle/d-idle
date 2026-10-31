package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2map"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/quest"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/town"
	"d2lod-bot/internal/town/hand"
)

// objectAdapter is bot.Objects for the Act 2 and 3 steps: the task layer names an object, this resolves the name to an
// object class and clicks it. Classes are V from objects.txt (docs/act2.md, docs/act3.md); which chest holds what and
// whether a plain click suffices is per entry. The Act 1 names (Inifuss tree, cairn stones, Tristram portal, Cain's
// cage) are in cain.go. Anything not listed fails with a Prereq error instead of a guessed click.
type objectAdapter struct {
	e  *env.Env
	nv *nav.Navigator
	// clicked is the objects of objOnce specs already clicked in this area (the pointer keeps the value-type adapter
	// shareable); nil in tests that do not click.
	clicked *onceSet
}

type objKind int

const (
	objClick objKind = iota // walk next to it and click once (chest, altar)
	objLeave                // a portal or door: the click must change the area
	objOrb                  // click until the object's mode changes (the Compelling Orb)
	objPlace                // put the quest item (spec.item, an items.txt code) on the object: the Staff into the orifice, the Soulstone on the forge
	objOnce                 // click the nearest one not clicked yet in this area (the three Frigid Highlands cages, Cain's cage)
	objSmash                // click it several times (the Hellforge with the Hammer in hand); the result shows in the quest word, not in a mode (U)
)

type objSpec struct {
	class int
	kind  objKind
	item  string // objPlace only
}

// objSpecs maps the names the task nodes pass to Interact. Tags: class V (objects.txt), the role of the relic chests
// 405..407 is S (Kolbot: A Heart, B Brain, C Eye) and U live; 342 is S (Kolbot RedPortalToAct4).
var objSpecs = map[string]objSpec{
	"Horadric Scroll chest":   {quest.ObjScrollChest, objClick, ""},
	"Horadric Cube chest":     {quest.ObjCubeChest, objClick, ""},
	"Staff of Kings chest":    {quest.ObjStaffChest, objClick, ""},
	"taintedsunaltar":         {quest.ObjTaintedSun, objClick, ""},
	"tal-orifice":             {quest.ObjOrifice, objPlace, quest.ItemHoradricStaff},
	"duriel-door":             {quest.ObjDurielDoor, objLeave, ""},
	"Arcane Sanctuary portal": {quest.ObjArcanePortal, objLeave, ""},
	"Summoner's journal":      {quest.ObjTomeJournal, objClick, ""},
	"Khalim's Heart chest":    {quest.ObjRelicChestA, objClick, ""},
	"Khalim's Brain chest":    {quest.ObjRelicChestB, objClick, ""},
	"Khalim's Eye chest":      {quest.ObjRelicChestC, objClick, ""},
	"gidbinn altar":           {quest.ObjGidbinnAltar, objClick, ""},
	"Lam Esen altar":          {quest.ObjLamTomeAltar, objClick, ""},
	"red portal":              {quest.ObjRedPortalA3, objLeave, ""},
	"Compelling Orb":          {quest.ObjCompellingOrb, objOrb, ""},
	// Acts 4 and 5. Classes V (objects.txt, docs/act4.md and act5.md), roles S (Kolbot), every click U live.
	ForgePlaceObject: {quest.ObjHellforge, objPlace, quest.ItemSoulstone},
	ForgeSmashObject: {quest.ObjHellforge, objSmash, ""},
	anyaObject:       {quest.ObjFrozenAnya, objClick, ""},
	"Anya's portal":  {quest.ObjRedPortal, objLeave, ""}, // S Kolbot: class 60 in Harrogath (not the Act 3 portal, 342)
	// Act 1 (docs/quest.md check 7): classes 30, 26 are U (objects.txt names not re-read), 60 is S (Kolbot RedPortal), 473 V.
	"Tree of Inifuss": {quest.ObjTreeInifuss, objClick, ""},
	"Tristram portal": {quest.ObjRedPortal, objLeave, ""},
	"Cain's Cage":     {quest.ObjCainGibbet, objOnce, ""},
	"cage":            {quest.ObjCage, objOnce, ""}, // Rescue on Mount Arreat: the step calls this three times
}

// Object names the Hellforge step uses (bot.ForgePlaceObject and bot.ForgeSmashObject, shared with internal/bot).
const (
	ForgePlaceObject = bot.ForgePlaceObject
	ForgeSmashObject = bot.ForgeSmashObject
	anyaObject       = bot.AnyaObject
)

func (o objectAdapter) Interact(ctx context.Context, name string) error {
	if item, ok := strings.CutPrefix(name, "item:"); ok {
		return o.useItem(ctx, item)
	}
	if name == CairnStones {
		return o.cairnStones(ctx)
	}
	spec, ok := objSpecs[name]
	if !ok {
		return task.Errorf(task.KindPrereq, "object %q is not implemented in the d2bot adapter", name)
	}
	if spec.kind == objPlace {
		return o.placeItem(ctx, spec.class, spec.item)
	}
	s0, err := o.e.Snapshot()
	if err != nil {
		return err
	}
	area := s0.Player.Area
	tries := 1
	switch spec.kind {
	case objOrb:
		tries = 8
	case objSmash:
		tries = 4
	}
	for try := 0; try < tries; try++ {
		var u *d2.Unit
		var err error
		if spec.kind == objOnce {
			u, err = o.findUnclicked(ctx, spec.class, area)
		} else {
			u, err = o.find(ctx, spec.class)
		}
		if err != nil {
			return task.Wrap(task.KindNotFound, err)
		}
		if err := o.click(ctx, u); err != nil {
			return err
		}
		if spec.kind == objOnce {
			o.clicked.add(area, u.ID)
		}
		switch spec.kind {
		case objLeave:
			if o.waitArea(ctx, area, 8*time.Second) {
				return nil
			}
			return task.Errorf(task.KindNotFound, "%s: the click did not change the area", name)
		case objOrb:
			if o.waitMode(ctx, u.ID, u.Mode, 2500*time.Millisecond) {
				return nil
			}
		case objSmash:
			o.e.Clock.Sleep(1500 * time.Millisecond)
		default:
			o.e.Clock.Sleep(1200 * time.Millisecond) // the drop lands on the ground
			return nil
		}
	}
	if spec.kind == objSmash {
		return nil
	}
	return task.Errorf(task.KindTimeout, "%s: the object never changed after %d clicks", name, tries)
}

func (o objectAdapter) find(ctx context.Context, class int) (*d2.Unit, error) {
	return o.findNear(ctx, class, nil)
}

// findNear is the object of the class nearest to at (the character when at is nil), looking again after walking to the
// level preset when nothing streamed in yet.
func (o objectAdapter) findNear(ctx context.Context, class int, at *input.Point) (*d2.Unit, error) {
	return o.findNearSkip(ctx, class, at, nil)
}

// findNearSkip is findNear that ignores the units skip accepts.
func (o objectAdapter) findNearSkip(ctx context.Context, class int, at *input.Point, skip func(*d2.Unit) bool) (*d2.Unit, error) {
	for try := 0; try < 3; try++ {
		s, err := o.e.Snapshot()
		if err != nil {
			return nil, err
		}
		ref := input.Point{X: s.Player.X, Y: s.Player.Y}
		if at != nil {
			ref = *at
		}
		d := func(u *d2.Unit) int { return (u.X-ref.X)*(u.X-ref.X) + (u.Y-ref.Y)*(u.Y-ref.Y) }
		var best *d2.Unit
		for i := range s.Objects {
			if u := &s.Objects[i]; u.Class == class && (skip == nil || !skip(u)) && (best == nil || d(u) < d(best)) {
				best = u
			}
		}
		if best != nil {
			return best, nil
		}
		// quest chests stream in late (V act2 F1): walk to the preset, then look again
		if o.e.World != nil {
			_, _ = o.e.World.Update()
			if lm := o.e.World.Current(); lm != nil {
				for _, p := range lm.Presets {
					if p.Kind == d2map.PresetObject && p.ID == class {
						_ = o.nv.GoTo(ctx, input.Point{X: p.Pos.X, Y: p.Pos.Y})
						break
					}
				}
			}
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	return nil, fmt.Errorf("object class %d not found", class)
}

// click walks next to the object and clicks it. The hover check of Interact fails for some objects (V act2 S1, W3);
// then a plain click on its world position is the fallback.
func (o objectAdapter) click(ctx context.Context, u *d2.Unit) error {
	if err := o.nv.GoToUnitOf(ctx, d2.TypeObject, u.ID); err != nil {
		return task.Wrap(task.KindNotFound, err)
	}
	_ = o.e.In.CloseAll()
	t := input.Target{Type: town.TypeObject, ID: u.ID, Pos: input.Point{X: u.X, Y: u.Y}}
	if err := o.e.In.Interact(t); err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err := o.e.In.ClickWorld(t.Pos); err != nil {
			return err
		}
	}
	return nil
}

func (o objectAdapter) waitArea(ctx context.Context, from int, d time.Duration) bool {
	deadline := o.e.Clock.Now().Add(d)
	for o.e.Clock.Now().Before(deadline) && ctx.Err() == nil {
		if s, err := o.e.Snapshot(); err == nil && s.Player != nil && s.Player.Area != from {
			return true
		}
		o.e.Clock.Sleep(250 * time.Millisecond)
	}
	return false
}

func (o objectAdapter) waitMode(ctx context.Context, id uint32, mode int, d time.Duration) bool {
	deadline := o.e.Clock.Now().Add(d)
	for o.e.Clock.Now().Before(deadline) && ctx.Err() == nil {
		if s, err := o.e.Snapshot(); err == nil {
			for i := range s.Objects {
				if s.Objects[i].ID == id {
					if s.Objects[i].Mode != mode {
						return true
					}
					break
				}
			}
		}
		o.e.Clock.Sleep(250 * time.Millisecond)
	}
	return false
}

// useItem right-clicks the pack item with the display name (the Book of Skill: +1 skill point).
func (o objectAdapter) useItem(ctx context.Context, display string) error {
	h := hand.New(o.e)
	sn, err := h.OpenInventory(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = h.CloseAll(ctx) }()
	lay, err := h.Layout(sn)
	if err != nil {
		return err
	}
	for _, it := range sn.Inventory.Container(d2.ContInventory) {
		if b := h.Base(it); b != nil && strings.EqualFold(b.Display, display) {
			if err := h.RightClick(it, lay.Inventory); err != nil {
				return err
			}
			o.e.Clock.Sleep(600 * time.Millisecond)
			return nil
		}
	}
	return task.Errorf(task.KindNotFound, "no %s in the pack", display)
}

// UseObject and ObjectMode are boss.ObjectOps (seals, the Ancients' altar, Baal's portal): the object of the class nearest
// to at. Mode 0 is untouched (S for seals); an object that is not streamed in reads as an error, which the scripts treat
// as "not open yet".
func (o objectAdapter) UseObject(ctx context.Context, class int, at boss.Pt) error {
	p := input.Point{X: at.X, Y: at.Y}
	u, err := o.findNear(ctx, class, &p)
	if err != nil {
		return task.Wrap(task.KindNotFound, err)
	}
	return o.click(ctx, u)
}

func (o objectAdapter) ObjectMode(ctx context.Context, class int, at boss.Pt) (int, error) {
	s, err := o.e.Snapshot()
	if err != nil {
		return 0, err
	}
	var best *d2.Unit
	for i := range s.Objects {
		u := &s.Objects[i]
		if u.Class != class {
			continue
		}
		if best == nil || sq(u.X-at.X)+sq(u.Y-at.Y) < sq(best.X-at.X)+sq(best.Y-at.Y) {
			best = u
		}
	}
	if best == nil {
		return 0, fmt.Errorf("object class %d not in view", class)
	}
	return best.Mode, nil
}

func sq(n int) int { return n * n }

var _ boss.ObjectOps = objectAdapter{}

// placeItem puts the quest item (an items.txt code) on the object. Ported from tools/rigs/act2live a2env.PlaceItem (V live up to taking the
// Staff to the cursor; the orifice accepting it was never reached, U). With an item on the cursor the game reports no
// hover, so the click point is found first (scan for the object being hovered), then the item is taken and that point
// clicked.
func (o objectAdapter) placeItem(ctx context.Context, class int, code string) error {
	err := o.placeItemOnce(ctx, class, code)
	// Never leave the quest item on the cursor: the next walk click would drop it on the floor (review M6). CloseAll
	// returns a held item to the pack; on success it only confirms the cursor is empty.
	if s, serr := o.e.Snapshot(); serr == nil && s.Inventory != nil && s.Inventory.Cursor != nil {
		if cerr := hand.New(o.e).CloseAll(context.WithoutCancel(ctx)); cerr != nil && err == nil {
			err = task.Wrap(task.KindUI, fmt.Errorf("item stuck on the cursor: %w", cerr))
		}
	}
	return err
}

func (o objectAdapter) placeItemOnce(ctx context.Context, class int, code string) error {
	g, ok := o.e.In.(*input.Game)
	if !ok {
		return task.Errorf(task.KindPrereq, "placing an item on an object needs the real game input (hover scan)")
	}
	u, err := o.find(ctx, class)
	if err != nil {
		return task.Wrap(task.KindNotFound, err)
	}
	if err := o.nv.GoToUnitOf(ctx, d2.TypeObject, u.ID); err != nil {
		return task.Wrap(task.KindNotFound, err)
	}
	_ = g.CloseAll()
	o.e.Clock.Sleep(400 * time.Millisecond)
	px, py, ok := findHover(g, u, o.e.Clock)
	if !ok {
		return task.Errorf(task.KindUI, "the orifice is never hovered (monsters over it?)")
	}
	h := hand.New(o.e)
	sn, err := h.OpenInventory(ctx)
	if err != nil {
		return err
	}
	lay, err := h.Layout(sn)
	if err != nil {
		return err
	}
	taken := false
	for _, it := range sn.Inventory.Container(d2.ContInventory) {
		if b := h.Base(it); b != nil && b.Code == code {
			if err := h.Take(ctx, it, lay.Inventory); err != nil {
				return err
			}
			taken = true
			break
		}
	}
	if !taken {
		_ = h.CloseAll(ctx)
		return task.Errorf(task.KindPrereq, "no %s in the pack", code)
	}
	if err := g.CloseAll(); err != nil {
		return err
	}
	o.e.Clock.Sleep(600 * time.Millisecond)
	for _, off := range [][2]int{{0, 0}, {0, -6}, {0, 6}, {-6, 0}, {6, 0}} {
		x, y := px+off[0], py+off[1]
		_ = g.MoveMouse(x, y)
		o.e.Clock.Sleep(300 * time.Millisecond)
		_ = g.Click(input.Left, x, y)
		o.e.Clock.Sleep(2500 * time.Millisecond)
		s, err := o.e.Snapshot()
		if err != nil {
			return err
		}
		for i := range s.Objects {
			if s.Objects[i].ID == u.ID && s.Objects[i].Mode != u.Mode {
				return nil
			}
		}
		if s.Inventory != nil && s.Inventory.Cursor == nil {
			return nil // the cursor is empty: the orifice took it
		}
	}
	return errors.New("the object did not take the item")
}

func findHover(g *input.Game, o *d2.Unit, clk input.Clock) (int, int, bool) {
	if g.Hover == nil {
		return 0, 0, false
	}
	vx, vy, err := g.R.Viewport()
	if err != nil {
		return 0, 0, false
	}
	sx, sy := g.X.WorldToScreenVP(vx, vy, float64(o.X)+0.5, float64(o.Y)+0.5)
	cx, cy := int(sx), int(sy)
	var mx, my, n int
	for dy := -70; dy <= 30; dy += 10 {
		for dx := -60; dx <= 60; dx += 10 {
			x, y := cx+dx, cy+dy
			if !g.X.InView(input.Screen{X: x, Y: y}) {
				continue
			}
			_ = g.MoveMouse(x, y)
			clk.Sleep(90 * time.Millisecond)
			if _, id, err := g.Hover(); err == nil && id == o.ID {
				mx, my, n = mx+x, my+y, n+1
			}
		}
	}
	if n == 0 {
		return 0, 0, false
	}
	return mx / n, my / n, true
}
