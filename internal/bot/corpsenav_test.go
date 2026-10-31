package bot

import (
	"context"
	"testing"

	"d2lod-bot/internal/input"
)

// LIVE G1 attempt 12: the town portal led to Cold Plains, the corpse lay in Black Marsh, BackToField failed with
// "no exit to that level". The retrieval must walk the rest of the route itself.
func TestRetrieveCorpseTravelsWhenPortalLandsElsewhere(t *testing.T) {
	g := newRig(t)
	g.w.set(func(w *world) { w.area = 1 })
	g.r.poll()
	g.town.armed.Store(true)
	g.town.backErr = func(int) { g.w.move(3); g.town.backReturn = ErrPortalWrongLevel } // the portal dropped us in Cold Plains
	g.r.poll()
	if err := g.r.retrieveCorpse(context.Background(), 6, input.Point{X: 15141, Y: 5080}); err != nil {
		t.Fatal(err)
	}
	if g.w.count("wp 6")+g.w.count("exit 6") != 1 || g.w.count("goto 15141,5080") != 1 {
		t.Fatalf("calls: %v", g.w.log())
	}
	if s := g.r.poll(); s.Player.Area != 6 {
		t.Fatalf("area %d", s.Player.Area)
	}
}
