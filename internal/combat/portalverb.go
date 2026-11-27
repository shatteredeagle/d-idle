package combat

import (
	"context"
	"errors"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// errPortalVerbNoEffect is a use-item verb that was sent but opened no portal within the wait: the attempt fails and the
// caller's next try sends the verb again, never the pack click (07-input-layer.md: the dropped click cast is the incident).
var errPortalVerbNoEffect = errors.New("combat: no portal appeared after the use-item verb")

// portalByVerb casts Town Portal from scroll or tome it with the use-item verb, the message the pack right-click sends
// (input layer I3, docs/design/input-i3.md): no panel, no pixel, no 700 ms wait. It runs only with the knob input.planner
// on and the connected build listing input.use-item; handled false leaves the click path to the caller (knob off, no
// capability, a controller without the verb, a transport error on the send). Success is read from state with the
// portal object, the scroll gone, or the tome's quantity stat lower than before.
func (f *Fighter) portalByVerb(ctx context.Context, it *d2.Unit) (done, handled bool, err error) {
	if !f.e.Planner.On() || !f.e.Planner.HasCap(input.CapUseItem) {
		return false, false, nil
	}
	u, ok := input.AsItemUser(f.e.In)
	if !ok {
		return false, false, nil
	}
	qty0 := PortalQty(it, f.cat, f.stats())
	if verr := u.UseItem(it.ID); verr != nil {
		f.info("portal: use-item verb unavailable, clicking", "item", it.ID, "err", verr)
		return false, false, nil
	}
	for i := 0; i < 6; i++ {
		if err := f.sleep(ctx, 150*time.Millisecond); err != nil {
			return false, true, err
		}
		s, err := f.snapFn()
		if err != nil || s == nil || s.Player == nil {
			continue
		}
		if portalCast(s, it) || f.tomeSpent(s, it, qty0) {
			f.e.Planner.Count(input.PathVerb)
			f.info("town portal cast", "item", it.ID, "path", string(input.PathVerb))
			return true, true, nil
		}
	}
	return false, true, errPortalVerbNoEffect
}

// tomeSpent reports whether tome it now holds fewer charges than qty0 (the quantity stat, stat 70).
func (f *Fighter) tomeSpent(s *d2.Snapshot, it *d2.Unit, qty0 int) bool {
	if !f.cat.Item(it.Class).Tome || s.Inventory == nil {
		return false
	}
	for i := range s.Inventory.Items {
		if u := &s.Inventory.Items[i]; u.ID == it.ID {
			return PortalQty(u, f.cat, f.stats()) < qty0
		}
	}
	return false
}
