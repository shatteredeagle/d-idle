package main

import (
	"context"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/drill"
)

// Retreat is the RT group's instance side (docs/retreatdrill.md). All cheats go through r.Cheat before the Act; after it
// the lab only steps and reads.
func (l *lab) Retreat(ctx context.Context, r *drill.Run, s drill.RetreatSpec) (ri drill.RetreatInfo) {
	ri.TicksToEffect = -1
	if s.Variant == "leave" && l.rig.Fighter.Leave == nil {
		// without the hook Act(Chicken) sends the unverified esc-and-click placeholder (combat/fighter.go leave)
		ri.Err = "leave hook not wired in d2drill (Fighter.Leave is nil): the chicken path cannot be run yet"
		return
	}
	const maxLife = 300
	if err := l.d11Setup(ctx, r, 2, maxLife, 8); err != nil {
		ri.Err = err.Error()
		return
	}
	defer l.d11Leave(ctx, r)
	if s.Belt {
		if err := l.ClearPack(ctx, r); err != nil {
			ri.Err = "pack: " + err.Error()
			return
		}
	} else if s.Scroll {
		if err := l.d11OneScroll(ctx, r); err != nil {
			ri.Err = "scroll: " + err.Error()
			return
		}
	} else {
		if _, err := r.Cheat(ctx, "debug belt clear", "no potion in the belt"); err != nil {
			ri.Err = err.Error()
			return
		}
		if err := l.ClearPack(ctx, r); err != nil {
			ri.Err = "pack: " + err.Error()
			return
		}
	}
	if got := l.d11Spawn(ctx, r, 1, s.Adjacent, 3); got < 0 {
		ri.Err = "spawn failed"
		return
	}
	if err := l.SetLife(ctx, r, maxLife*s.LifePct/100); err != nil {
		ri.Err = "life: " + err.Error()
		return
	}
	// from here no cheat: the bot decides and acts
	sn, err := l.be.Env.Snapshot()
	if err != nil || sn.Player == nil {
		ri.Err = "snapshot"
		return
	}
	ri.Adjacent, _, ri.Life, ri.MaxLife = l.d11Within(ctx, 3)
	ri.Life = sn.Player.Life
	ri.BeltBefore, _ = l.beltKinds(sn)
	if v, e := l.PView(ctx); e == nil {
		ri.ScrollsBefore = v.Scrolls
	}
	if s.Belt && ri.BeltBefore == 0 {
		ri.Err = "setup: no healing potion in the belt"
		return
	}
	a := l.rig.Fighter.Safety(sn)
	ri.Decision = a.String()
	ri.TicksToAct = 0
	ri.LifeFloor = sn.Player.Life
	if a == combat.SafetyNone {
		return
	}
	ri.Acted = true
	if err := l.rig.Fighter.Act(ctx, a); err != nil {
		ri.ActErr = err.Error()
		return
	}
	for t := 1; t <= s.MaxTicks; t++ {
		if err := l.Tick(ctx, 1); err != nil {
			ri.ActErr = "tick: " + err.Error()
			return
		}
		s2, err := l.be.Env.Snapshot()
		if err != nil || s2.State != d2.StateInGame || s2.Player == nil {
			ri.LeftGame = true
			ri.TicksToEffect = t
			return
		}
		ri.LifeEnd = s2.Player.Life
		ri.LifeFloor = min(ri.LifeFloor, s2.Player.Life)
		ri.Dead = s2.Player.Life <= 0
		ri.BeltAfter, _ = l.beltKinds(s2)
		if v, e := l.PView(ctx); e == nil {
			ri.ScrollsAfter, ri.PortalOpen = v.Scrolls, v.Near > 0
		}
		done := false
		switch s.Variant {
		case "drink":
			done = ri.BeltAfter < ri.BeltBefore && ri.LifeEnd > ri.Life
		case "portal":
			done = ri.PortalOpen && ri.ScrollsAfter < ri.ScrollsBefore
		}
		if done {
			ri.TicksToEffect = t
			return
		}
	}
	return
}

var _ drill.RetreatLab = (*lab)(nil)
