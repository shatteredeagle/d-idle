package town

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// waitArea waits until the player stands in level (any level when level < 0 and the area is a town).
// Loading screens read as "not in game" and are simply waited through.
func (t *Town) waitArea(ctx context.Context, level int, timeout time.Duration) (*d2.Snapshot, error) {
	deadline := t.e.Clock.Now().Add(timeout)
	for {
		s, err := t.h.Snap()
		if err != nil {
			return nil, err
		}
		if s.State == d2.StateInGame && s.Player != nil {
			if level >= 0 && s.Player.Area == level || level < 0 && data.IsTown(s.Player.Area) {
				return s, nil
			}
		}
		if !t.e.Clock.Now().Before(deadline) {
			return s, fmt.Errorf("town: area %d not reached within %v", level, timeout)
		}
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		t.e.Clock.Sleep(100 * time.Millisecond)
	}
}

// tpTool finds a town portal scroll or a tome with scrolls in the pack.
func (t *Town) tpTool(s *d2.Snapshot) (d2.Unit, bool) {
	var tome d2.Unit
	have := false
	for _, u := range s.Inventory.Container(d2.ContInventory) {
		switch b := t.base(u); {
		case b == nil:
		case b.Code == "tsc":
			return u, true
		case b.Code == "tbk" && t.qty(u) > 0:
			tome, have = u, true
		}
	}
	return tome, have
}

// ToTown returns to the act's town: a town portal scroll (right-click it in the open pack, then click the
// portal that appears), unless already in town. UNVERIFIED live: needs a field area, outside this agent's
// live scope (docs/town.md). The Mover is the fallback only for walking to a portal that is not on screen.
func (t *Town) ToTown(ctx context.Context) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	if data.IsTown(s.Player.Area) {
		return nil
	}
	var last error = ErrNoPortal
	// A portal that is already open nearby (the one we came out of) is free: use it before spending a scroll.
	if portal, ok := findObject(s, ObjPortal); ok && dist(portal.X, portal.Y, s.Player.X, s.Player.Y) <= float64(t.tune().PortalReuseDist) {
		if err := t.enterPortal(ctx, portal); err == nil {
			if cur, werr := t.waitArea(ctx, -1, 15*time.Second); werr == nil {
				t.arrived = t.e.Clock.Now()
				t.e.Log.Info("arrived in town through an open portal", "area", cur.Player.Area)
				return nil
			}
		}
	}
	for try := 0; try < 2; try++ {
		portal, err := t.openPortal(ctx)
		if err != nil {
			if errors.Is(err, ErrNoPortal) {
				return t.walkToTown(ctx, s, err)
			}
			last = err
			continue
		}
		if err := t.enterPortal(ctx, portal); err != nil {
			last = err
			continue
		}
		cur, werr := t.waitArea(ctx, -1, 15*time.Second) // never overwrite s: a failed wait returns a nil snapshot and walkToTown needs the field one
		if werr != nil {
			last = werr
			continue
		}
		t.arrived = t.e.Clock.Now()
		t.e.Log.Info("arrived in town", "area", cur.Player.Area)
		return nil
	}
	return last
}

// townNeighbour maps the first field area of an act to its town: a character there with no scroll can simply walk
// back (Blood Moor to the Rogue Encampment, VERIFIED LIVE for the opposite direction by BackToField's exit fallback).
// Anywhere farther the walk is not attempted: nav would explore for exits.
var townNeighbour = map[int]int{2: 1}

// hopToNeighbour is the walk chain for fields that border Blood Moor (nav/route.go: Den of Evil is a dead end off
// Blood Moor, Cold Plains is its other neighbour). Used when there is no scroll and no usable waypoint (the Den has
// none; found live: the Den claim trip failed 5 times with ErrNoPortal).
var hopToNeighbour = map[int]int{8: 2, 3: 2, 17: 3, 9: 3}

// walkToTown is the no-scroll fallback of ToTown (found live on i4, 2026-10-28: a level 3 Booti with no potions and
// no scroll abandoned every emergency trip and fought on at 0 potions). It returns noPortal when it does not apply.
func (t *Town) walkToTown(ctx context.Context, s *d2.Snapshot, noPortal error) error {
	town, ok := townNeighbour[s.Player.Area]
	if !ok {
		err := t.waypointToTown(ctx, s, noPortal)
		hop, hok := hopToNeighbour[s.Player.Area]
		if err == nil || !hok {
			return err
		}
		t.e.Log.Info("no town portal and no waypoint: walking back via the neighbour", "from", s.Player.Area, "via", hop)
		if err := t.mv.ExitTo(ctx, hop); err != nil {
			return fmt.Errorf("%w (walk to town failed: %v)", noPortal, err)
		}
		cur, err := t.waitArea(ctx, hop, 15*time.Second)
		if err != nil {
			return fmt.Errorf("%w (walk to town failed: %v)", noPortal, err)
		}
		return t.walkToTown(ctx, cur, noPortal)
	}
	t.e.Log.Info("no town portal: walking back to town", "from", s.Player.Area, "to", town)
	if err := t.mv.ExitTo(ctx, town); err != nil {
		return fmt.Errorf("%w (walk to town failed: %v)", noPortal, err)
	}
	if _, err := t.waitArea(ctx, town, 15*time.Second); err != nil {
		return fmt.Errorf("%w (walk to town failed: %v)", noPortal, err)
	}
	t.arrived = t.e.Clock.Now()
	return nil
}

// waypointToTown is the second step of walkToTown for fields farther than one hop (Stony Field, Cold Plains): take
// the waypoint of the current level to the act's town. It needs both waypoints activated (U: not exercised live;
// TakeWaypoint reports ErrNoWaypoint when the town entry does not respond) and a level with a waypoint.
func (t *Town) waypointToTown(ctx context.Context, s *d2.Snapshot, noPortal error) error {
	lvl := t.e.Data.AreaByID(s.Player.Area)
	if lvl == nil || lvl.Waypoint == 255 || lvl.Waypoint == 0 || s.Player.Area >= data.Town(1) || !t.e.Tune.Nav.UseWaypoints {
		return noPortal
	}
	town := data.Town(0) // act 1 only: later acts keep ErrNoPortal until their routes are verified
	t.e.Log.Info("no town portal: taking the waypoint to town", "from", s.Player.Area)
	if err := t.mv.TakeWaypoint(ctx, town); err != nil {
		return fmt.Errorf("%w (waypoint to town failed: %v)", noPortal, err)
	}
	if _, err := t.waitArea(ctx, town, 15*time.Second); err != nil {
		return fmt.Errorf("%w (waypoint to town failed: %v)", noPortal, err)
	}
	t.arrived = t.e.Clock.Now()
	return nil
}

// openPortal reads a town portal scroll (or the tome) from the open pack and waits for the portal object to
// appear next to the player.
func (t *Town) openPortal(ctx context.Context) (d2.Unit, error) {
	s, err := t.current()
	if err != nil {
		return d2.Unit{}, err
	}
	tool, ok := t.tpTool(s)
	if !ok {
		return d2.Unit{}, ErrNoPortal
	}
	if s, err = t.h.OpenInventory(ctx); err != nil {
		return d2.Unit{}, err
	}
	lay, err := t.layout(s)
	if err != nil {
		return d2.Unit{}, err
	}
	if err := t.h.RightClick(tool, lay.Inventory); err != nil {
		return d2.Unit{}, err
	}
	if err := t.h.CloseAll(ctx); err != nil {
		return d2.Unit{}, err
	}
	var portal d2.Unit
	_, found, err := t.h.Wait(ctx, 3*time.Second, func(s *d2.Snapshot) bool {
		portal, ok = findObject(s, ObjPortal)
		return ok && dist(portal.X, portal.Y, s.Player.X, s.Player.Y) < 25
	})
	if err != nil {
		return d2.Unit{}, err
	}
	if !found {
		if s, err := t.current(); err == nil && s.Inventory != nil {
			t.e.Log.Warn("town portal did not appear", "tool", t.base(tool).Code, "toolQty", t.qty(tool), "gx", tool.Item.GX, "gy", tool.Item.GY,
				"packItems", len(s.Inventory.Container(d2.ContInventory)), "area", s.Player.Area, "mana", s.Player.Mana, "ui", fmt.Sprint(s.UI))
		}
		return d2.Unit{}, fmt.Errorf("town: portal did not appear")
	}
	return portal, nil
}

// OpenPortalForTest opens a portal without entering it.
func (t *Town) OpenPortalForTest(ctx context.Context) error { _, err := t.openPortal(ctx); return err }

func (t *Town) enterPortal(ctx context.Context, portal d2.Unit) error {
	if s, err := t.current(); err == nil && dist(portal.X, portal.Y, s.Player.X, s.Player.Y) > float64(t.tune().NPCStopDist) {
		if err := t.mv.GoTo(ctx, input.Point{X: portal.X, Y: portal.Y}); err != nil {
			return err
		}
	}
	err := t.e.In.Interact(input.Target{Type: TypeObject, ID: portal.ID, Pos: input.Point{X: portal.X, Y: portal.Y}})
	if errors.Is(err, input.ErrNotHovered) || errors.Is(err, input.ErrNoHover) {
		// Portals are not selectable (objects.txt) and a monster standing on top can steal the hover: walking onto
		// the portal works as well.
		return t.e.In.ClickWorld(input.Point{X: portal.X, Y: portal.Y})
	}
	return err
}

// ErrWrongLevel: the town portal led to another field level than the wanted one (it stands where the last town
// portal was opened, not where the character died: LIVE G1 attempt 12, portal to Cold Plains, corpse in Monastery
// Gate). The character stands in that level; the caller walks on with its route logic (waypoints, multi-hop).
var ErrWrongLevel = errors.New("town: portal led to another level")

func (t *Town) wrongLevel(levelNo int) error {
	if s, err := t.current(); err == nil && s.Player.Area != levelNo && !data.IsTown(s.Player.Area) {
		return fmt.Errorf("%w: wanted %d, in %d", ErrWrongLevel, levelNo, s.Player.Area)
	}
	return nil
}

// BackToField returns to levelNo: through the town portal that stands in town, else the waypoint, else the
// level border exit. It waits out the portal settle time after arriving in town.
func (t *Town) BackToField(ctx context.Context, levelNo int) error {
	s, err := t.current()
	if err != nil {
		return err
	}
	if s.Player.Area == levelNo {
		return nil
	}
	if !data.IsTown(s.Player.Area) {
		return fmt.Errorf("town: BackToField called outside town (area %d)", s.Player.Area)
	}
	if wait := time.Duration(t.tune().PortalSettle)*time.Millisecond - t.e.Clock.Now().Sub(t.arrived); !t.arrived.IsZero() && wait > 0 {
		t.e.Clock.Sleep(wait)
	}
	if portal, ok := findObject(s, ObjPortal); ok {
		t.e.Log.Info("back to field: portal in town", "id", portal.ID, "x", portal.X, "y", portal.Y, "player_x", s.Player.X, "player_y", s.Player.Y)
		if err := t.enterPortal(ctx, portal); err != nil {
			t.e.Log.Warn("back to field: portal click failed", "err", err)
			if err := t.wrongLevel(levelNo); err != nil {
				return err
			}
		} else if _, err := t.waitArea(ctx, levelNo, 15*time.Second); err == nil {
			return nil
		} else {
			t.e.Log.Warn("back to field: portal did not lead to the level", "err", err)
			if err := t.wrongLevel(levelNo); err != nil {
				return err
			}
		}
	} else {
		t.e.Log.Info("back to field: no portal in the unit table")
	}
	if lvl := t.e.Data.AreaByID(levelNo); lvl != nil && lvl.Waypoint != 255 && lvl.Waypoint != 0 && t.e.Tune.Nav.UseWaypoints {
		if err := t.mv.TakeWaypoint(ctx, levelNo); err == nil {
			if _, err := t.waitArea(ctx, levelNo, 15*time.Second); err == nil {
				return nil
			}
		}
	}
	if err := t.mv.ExitTo(ctx, levelNo); err != nil {
		// Not a neighbour of the town (Cold Plains from the Rogue Encampment, waypoint not active yet): walk the chain of
		// levels. Without this the trip home ended in "no way back" over and over, 17 % of a 6 h run (D2DLOOP r20, V live).
		ta, ok := t.mv.(interface {
			ToArea(context.Context, int) error
		})
		if !ok {
			return fmt.Errorf("town: no way back to level %d: %w", levelNo, err)
		}
		if err2 := ta.ToArea(ctx, levelNo); err2 != nil {
			return fmt.Errorf("town: no way back to level %d: %w (route: %v)", levelNo, err, err2)
		}
	}
	_, err = t.waitArea(ctx, levelNo, 15*time.Second)
	return err
}

// ClearCursor puts a held item into the pack. nav.Navigator.ClearCursor takes it: the waypoint panel does not open
// with an item on the cursor (D2D-V).
func (t *Town) ClearCursor(ctx context.Context) error { return t.h.ClearCursor(ctx) }
