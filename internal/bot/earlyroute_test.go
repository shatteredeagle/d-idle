package bot

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/task"
)

// Lab night 2026-11-28, drill.act2.shaft.stepexec: the hero spent its only portal scroll to leave a stall in Maggot Lair Level 1,
// came back by waypoint and walked in again. Inside the lair the early road guard refused the next step for the missing portal,
// the chores trip it inserted could not reach town without one, and the chunk was parked: the hero idled in the lair. The kit is
// an entry rule: from Far Oasis the road stays closed, from inside the road (level 1 asking for level 3) it is open. A level or
// life floor still closes it. Two classes.
func TestEarlyRoadKitIsAnEntryRule(t *testing.T) {
	const farOasis, lair1, lair3 = 43, 62, 64
	for _, class := range []d2.Class{d2.Paladin, d2.Sorceress} {
		for _, tc := range []struct {
			area, level int
			want        string
		}{
			{farOasis, 30, "no town portal for the act2_maggot_lair road"},
			{lair1, 30, ""},
			{lair1, 12, "needed for the act2_maggot_lair road"},
		} {
			g := newRig(t, func(c *Config, d *Deps, tu *env.Tuning) { tu.Route.EarlyRoute = 1 })
			g.w.set(func(w *world) {
				w.class, w.level, w.area, w.maxL, w.life = class, tc.level, tc.area, 1400, 1400
				var keep []d2.Unit
				for _, u := range w.items {
					if u.Class != 1 && u.Class != 5 { // no portal scroll; the potions are set below
						keep = append(keep, u)
					}
				}
				for i := 0; i < 4; i++ {
					keep = append(keep, d2.Unit{ID: uint32(700 + i), Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContBelt}})
				}
				w.items = keep
			})
			g.r.poll()
			got := g.r.earlyRouteWhy(g.r.latest(), lair3)
			if tc.want == "" && got != "" || tc.want != "" && !strings.Contains(got, tc.want) {
				t.Errorf("%v clvl %d in area %d: %q, want %q", class, tc.level, tc.area, got, tc.want)
			}
			var w task.World
			g.r.fillEarly(&w)
			if _, blocked := w.EarlyBlocked[lair3]; blocked != (tc.want != "") {
				t.Errorf("%v clvl %d in area %d: task layer blocked=%v, want %v", class, tc.level, tc.area, blocked, tc.want != "")
			}
		}
	}
}
