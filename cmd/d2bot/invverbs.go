package main

import (
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/invdesk"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/town"
)

// wireItemVerbs puts the verb-first executors in front of the click paths on the d2d backend
// (docs/design/item-verbs.md). Every executor asks the knob invdesk.verbs (default 0) at call time and the backend's
// caps, and falls back to the click path it wraps, so with the knob off nothing changes. Stash verbs are only asked by
// the town while the stash is open at the chest. Not used on Game.exe (no client).
func wireItemVerbs(ref *runnerRef, tw *town.Town, lv *leveler.Leveler, c *d2dctl.Client, e *env.Env) {
	if ref == nil || ref.inv == nil || c == nil || e == nil {
		return
	}
	on := func() bool { return e.Tune != nil && e.Tune.Invdesk.Verbs != 0 }
	verbs := d2dctl.ItemVerbs{C: c}
	mover := invdesk.NewVerbMover(verbs, on)
	mover.Fresh = e.Snapshot
	if tw != nil {
		tw.SetMover(mover)
	}
	ref.inv.SetHowSource(mover)
	if lv != nil {
		ve := invdesk.NewVerbEquipper(lv, verbs, on)
		ve.Verify = mover.Equipped
		ref.inv.SetEquipper(ve)
		ref.inv.SetTrasher(invdesk.VerbTrasher{Click: lv, Mover: mover})
	}
}
