package main

// Verbatim copies of the bot's d2d portal and town-entry code (cmd/d2bot/d2dwire.go and adapters.go): package main
// cannot be imported, and the drills must call the code the bot calls. TestPortalCopyInStep fails when the originals
// change, so the copy is refreshed on purpose (docs/probes/drills/README.md).

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town"
)

const (
	townPortalObject = 59
	portalIdleWait   = 2 * time.Second // how long a busy player is waited for before the click
	portalIdlePoll   = 100 * time.Millisecond
)

var errPortalBusy = errors.New("d2d portal: player busy (hit recovery, attack, cast or block), the click would be dropped")

// busyPMode says whether d2d drops a right click in this player mode (world.cpp pmode >= 0 outside walk and run, V-d2d
// per docs/research/town-portal-trace.md): get hit, attacks, block, cast, throw, kick, skills, sequence, knockback.
// Modes use PlrMode.txt order (d2dsrc plrModes); -1 (unknown) counts as idle.
func busyPMode(m int) bool {
	switch m {
	case 4, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 18, 19:
		return true
	}
	return false
}

func portalDead(ctx context.Context) error {
	return fmt.Errorf("d2d portal: cancelled, nothing clicked (%v)", context.Cause(ctx))
}

// portalCut is the error of a cast whose context died after the click: d2d may have spent the charge (Gate 1 20261102
// BELT window: two casts in one second, a safety drink cancelling each, the last two charges gone for no portal).
func portalCut(ctx context.Context) error {
	return fmt.Errorf("d2d portal: cast cut after the click, source maybe spent (%v)", context.Cause(ctx))
}

const (
	portalCutMemory = 8 * time.Second // a cut cast's portal may still open this long after the click
	portalCutWait   = 3 * time.Second
)

func d2dPortalWith(e *env.Env, itemAt func(context.Context, int) (int, int, error), cat combat.Catalog) func(context.Context) error {
	stats := func() combat.Stats {
		if e.Read != nil {
			return e.Read.UnitStats
		}
		return nil
	}
	// the same pick as combat's (combat.PortalSource): a scroll or a tome WITH charges, never an empty tome (F4)
	pick := func(s *d2.Snapshot) *d2.Unit { return combat.PortalSource(s, cat, stats()) }
	portals := func(ns *d2.Snapshot) (near, any int) {
		for _, o := range ns.Objects {
			if o.Class == townPortalObject {
				any++
				if math.Hypot(float64(o.X-ns.Player.X), float64(o.Y-ns.Player.Y)) < 25 {
					near++
				}
			}
		}
		return
	}
	// idle waits (bounded) until the player is out of every mode that drops the click: nil, errPortalBusy or the dead ctx.
	idle := func(ctx context.Context) error {
		for waited := time.Duration(0); ; waited += portalIdlePoll {
			if ctx.Err() != nil {
				return portalDead(ctx)
			}
			if s, err := e.Snapshot(); err == nil && s.Player != nil && !busyPMode(s.Player.Mode) {
				return nil
			}
			if waited >= portalIdleWait {
				return errPortalBusy
			}
			e.Clock.Sleep(portalIdlePoll)
		}
	}
	var cutMu sync.Mutex
	var cutAt time.Time
	// afterCut looks for the portal of the last cut cast before another charge is read: it reports true when it is open.
	afterCut := func(ctx context.Context) bool {
		cutMu.Lock()
		at := cutAt
		cutMu.Unlock()
		if at.IsZero() || e.Clock.Now().Sub(at) > portalCutMemory {
			return false
		}
		if ctx.Err() != nil {
			return false // a dead context never polled: keep the memory for the next call (codefix F8)
		}
		for waited := time.Duration(0); waited <= portalCutWait && ctx.Err() == nil; waited += portalIdlePoll {
			if ns, err := e.Snapshot(); err == nil && ns.Player != nil {
				if n, _ := portals(ns); n > 0 {
					e.Log.Info("d2d portal: the cut cast opened its portal, not casting again")
					return true
				}
			}
			e.Clock.Sleep(portalIdlePoll)
		}
		cutMu.Lock()
		cutAt = time.Time{}
		cutMu.Unlock()
		return false
	}
	return func(ctx context.Context) error {
		var last error
		if afterCut(ctx) {
			return nil
		}
		for try := 0; try < 3; try++ {
			if ctx.Err() != nil { // the action's context died (guard, step deadline, stop): the tries cannot work
				e.Log.Info("d2d portal: context dead, not clicking", "try", try+1, "cause", context.Cause(ctx))
				return portalDead(ctx)
			}
			s, err := e.Snapshot()
			if err != nil || s.Player == nil {
				return errors.New("d2d portal: no snapshot")
			}
			it := pick(s) // re-read every try: a cut cast spent a charge, an empty source is never retried
			if it == nil {
				e.Log.Info("d2d town portal: no portal source (empty)", "try", try+1)
				if try > 0 {
					return fmt.Errorf("%w (empty after the last cast)", combat.ErrNoPortal)
				}
				return combat.ErrNoPortal
			}
			before := combat.PortalQty(it, cat, stats())
			last = func() error {
				if err := idle(ctx); err != nil {
					return err
				}
				if err := e.In.Key("i"); err != nil {
					return err
				}
				defer e.In.Key("i")
				e.Clock.Sleep(700 * time.Millisecond)
				if ctx.Err() != nil {
					return portalDead(ctx)
				}
				x, y, err := itemAt(ctx, int(it.ID))
				if err != nil {
					return err
				}
				if err := e.In.MoveMouse(x, y); err != nil {
					return err
				}
				e.Clock.Sleep(150 * time.Millisecond)
				if err := idle(ctx); err != nil { // hit while the panel opened: the click would be dropped
					return err
				}
				if err := e.In.Click(input.Right, x, y); err != nil {
					return err
				}
				for i := 0; i < 8; i++ {
					e.Clock.Sleep(200 * time.Millisecond)
					ns, err := e.Snapshot()
					if err != nil || ns.Player == nil {
						continue
					}
					if n, _ := portals(ns); n > 0 {
						return nil // only the object counts: a scroll that left the pack may be a cut cast
					}
					if ctx.Err() != nil {
						cutMu.Lock()
						cutAt = e.Clock.Now()
						cutMu.Unlock()
						return portalCut(ctx)
					}
				}
				// diagnostic for the unexplained failures (docs/fixportal.md): what was clicked and what the world showed
				nearN, anyN, spent := 0, 0, false
				if ns, nerr := e.Snapshot(); nerr == nil && ns.Player != nil {
					nearN, anyN = portals(ns)
					spent = true // gone from the pack, unless it is still there with as many casts as before
					if ns.Inventory != nil {
						for i := range ns.Inventory.Items {
							if u := &ns.Inventory.Items[i]; u.ID == it.ID {
								spent = combat.PortalQty(u, cat, stats()) < before
							}
						}
					}
				}
				e.Log.Info("d2d portal: click without a portal", "item", it.ID, "class", it.Class, "portalObjNear", nearN, "portalObjAny", anyN, "sourceSpent", spent)
				if spent {
					return errors.New("d2d portal: no object after the click, the cast was cut (source spent)")
				}
				return errors.New("d2d portal: no object after the click, the click was dropped (source unchanged)")
			}()
			if last == nil {
				return nil
			}
			e.Log.Info("d2d town portal attempt failed", "try", try+1, "err", last)
			if ctx.Err() != nil {
				return last
			}
			e.Clock.Sleep(300 * time.Millisecond)
		}
		return last
	}
}

type d2dTown struct {
	townAdapter
	portal func(context.Context) error
}

func (t d2dTown) ToTown(ctx context.Context) error {
	if s, err := t.e.Snapshot(); err == nil && s.Player != nil && !s.Player.InTown() && t.portal != nil {
		open := false
		for _, o := range s.Objects {
			open = open || (o.Class == town.ObjPortal && max(abs(o.X-s.Player.X), abs(o.Y-s.Player.Y)) <= 25)
		}
		if !open {
			if err := t.portal(ctx); err != nil && !errors.Is(err, combat.ErrNoPortal) {
				t.e.Log.Info("d2d town: no portal opened, falling back", "err", err)
			}
		}
	}
	return t.townAdapter.ToTown(ctx)
}

type townAdapter struct {
	t  *town.Town
	e  *env.Env
	mv interface {
		GoToUnitOf(ctx context.Context, typ int, id uint32) error
	}
}

// ToTown reuses a portal the fighter already opened next to the player, else lets the Townsman open one.
func (a townAdapter) ToTown(ctx context.Context) error {
	s, err := a.e.Snapshot()
	if err == nil && s.Player != nil && !s.Player.InTown() {
		for _, o := range s.Objects {
			if o.Class != town.ObjPortal || max(abs(o.X-s.Player.X), abs(o.Y-s.Player.Y)) > 25 {
				continue
			}
			if err := a.e.In.Interact(input.Target{Type: town.TypeObject, ID: o.ID, Pos: input.Point{X: o.X, Y: o.Y}}); err != nil {
				break
			}
			if a.waitTown(ctx, 15*time.Second) == nil {
				return nil
			}
			break
		}
	}
	return a.t.ToTown(ctx)
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

func (a townAdapter) waitTown(ctx context.Context, d time.Duration) error {
	deadline := a.e.Clock.Now().Add(d)
	for a.e.Clock.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if s, err := a.e.Snapshot(); err == nil && s.State == d2.StateInGame && s.Player != nil && s.Player.InTown() {
			return nil
		}
		a.e.Clock.Sleep(150 * time.Millisecond)
	}
	return errors.New("portal did not lead to town")
}

func (a townAdapter) BackToField(ctx context.Context, level int) error {
	err := a.t.BackToField(ctx, level)
	if errors.Is(err, town.ErrWrongLevel) {
		return fmt.Errorf("%w: %v", bot.ErrPortalWrongLevel, err)
	}
	return err
}
