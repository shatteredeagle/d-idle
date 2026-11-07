package bot

import (
	"testing"

	"d2lod-bot/internal/d2"
)

// tpTown is the fake Townsman with the town code's scroll count (bot.TPDiag half).
type tpTown struct {
	*fakeTown
	n int
}

func (t *tpTown) TPCount(*d2.Snapshot) int { return t.n }

// g2g S1: a hero in town with gold for both a scroll and gear and fewer scrolls than town.tpReserve restocks first,
// then shops; with the reserve held the gear pass is unchanged. Run for the default Sorceress and a Paladin.
func TestScrollIsBoughtBeforeGear(t *testing.T) {
	for _, tc := range []struct {
		name      string
		class     d2.Class
		scrolls   int
		wantFirst bool
	}{
		{"sorceress 0 scrolls", d2.Sorceress, 0, true},
		{"sorceress 1 scroll", d2.Sorceress, 1, true},
		{"paladin 0 scrolls", d2.Paladin, 0, true},
		{"paladin reserve held", d2.Paladin, 2, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := newRig(t)
			g.r.d.Town = &tpTown{fakeTown: g.town, n: tc.scrolls}
			gr := &fakeGear{w: g.w}
			gr.swaps.Store(1)
			g.r.d.Gear = gr
			g.mover.exploreFn = blockExplore
			g.w.gold(1000)
			g.w.set(func(w *world) { w.class = tc.class })
			g.run()
			defer g.finish()
			g.r.Start(reach(10))
			g.waitFor("shop", func() bool { return gr.shopped.Load() > 0 })
			g.w.mu.Lock()
			calls := append([]string(nil), g.w.calls...)
			g.w.mu.Unlock()
			first, shop := -1, -1
			for i, c := range calls {
				if c == "chores scroll before gear" && first < 0 {
					first = i
				}
				if len(c) >= 4 && c[:4] == "shop" && shop < 0 {
					shop = i
				}
			}
			if tc.wantFirst && (first < 0 || first > shop) {
				t.Fatalf("scroll restock must come before the gear shop: %v", calls)
			}
			if !tc.wantFirst && first >= 0 {
				t.Fatalf("reserve held, no restock wanted: %v", calls)
			}
		})
	}
}
