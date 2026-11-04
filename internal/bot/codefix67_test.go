package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// codefix67 F7: a town visit that ended with no healing potion and no more gold (stash included) starts no further
// in-town emergency round until gold grows, a potion shows up, or the character leaves town. Two profiles.
func TestInTownFutilityLatch(t *testing.T) {
	for _, tc := range []struct {
		name  string
		class d2.Class
		level int
		gold  int
		stash int
	}{{"sorceress", d2.Sorceress, 6, 185, 0}, {"barbarian", d2.Barbarian, 12, 20, 1400}} {
		g := newRig(t)
		g.w.set(func(w *world) { w.level, w.area, w.gold0, w.stash0 = tc.level, 1, tc.gold, tc.stash })
		g.w.belt()
		g.r.loop.broke.on = tc.stash == 0
		g.r.setLastTrip(g.e.Clock.Now().Add(-10 * time.Second))
		s, _ := g.w.snapshot()
		s.Player.Class = tc.class
		if _, emerg, ok := g.r.needChores(s); !ok || !emerg {
			t.Fatalf("%s: want an emergency restock before the visit, got ok=%v emergency=%v", tc.name, ok, emerg)
		}
		basis := g.r.resInput(s).Gold
		g.r.noteTownFutile(&trip{startBasis: basis}, s) // the visit bought nothing
		if _, _, ok := g.r.needChores(s); ok {
			t.Errorf("%s: a futile visit must not start another in-town round", tc.name)
		}
		g.w.set(func(w *world) { w.gold0 += 50 })
		s2, _ := g.w.snapshot()
		s2.Player.Class = tc.class
		if _, _, ok := g.r.needChores(s2); !ok {
			t.Errorf("%s: gold growth must lift the latch", tc.name)
		}
	}
}
