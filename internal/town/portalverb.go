package town

import (
	"context"
	"errors"
	"fmt"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/input"
)

// errPortalVerbNoEffect is a use-item verb that was sent but no portal stood by the player within the wait. The pack
// click is not tried after it (a dropped click cast is the incident the verb replaces); the caller's retry sends the
// verb again.
var errPortalVerbNoEffect = errors.New("town: portal did not appear after the use-item verb")

// portalByVerb opens a town portal from scroll or tome tool with the use-item verb, the message the pack right-click
// sends (input layer I3, docs/design/input-i3.md): no pack panel, no cell geometry. Only with the knob input.planner on
// and the connected build listing input.use-item; handled false leaves the click path to the caller. The tome's
// quantity stat is read before and after, so a failure says whether a charge was spent.
func (t *Town) portalByVerb(ctx context.Context, tool d2.Unit) (portal d2.Unit, handled bool, err error) {
	if !t.e.Planner.On() || !t.e.Planner.HasCap(input.CapUseItem) {
		return d2.Unit{}, false, nil
	}
	u, ok := input.AsItemUser(t.e.In)
	if !ok {
		return d2.Unit{}, false, nil
	}
	q0 := t.qty(tool)
	if verr := u.UseItem(tool.ID); verr != nil {
		t.e.Log.Info("town portal: use-item verb unavailable, clicking", "item", tool.ID, "err", verr)
		return d2.Unit{}, false, nil
	}
	_, found, err := t.h.Wait(ctx, 3*time.Second, func(s *d2.Snapshot) bool {
		var ok bool
		portal, ok = t.findPortal(s)
		return ok && dist(portal.X, portal.Y, s.Player.X, s.Player.Y) < 25
	})
	if err != nil {
		return d2.Unit{}, true, err
	}
	if !found {
		q1 := t.qty(tool)
		t.e.Log.Warn("town portal did not appear", "path", string(input.PathVerb), "tool", tool.ID, "qty", fmt.Sprintf("%d->%d", q0, q1))
		return d2.Unit{}, true, errPortalVerbNoEffect
	}
	t.e.Planner.Count(input.PathVerb)
	return portal, true, nil
}
