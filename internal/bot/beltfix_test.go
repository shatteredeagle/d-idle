package bot

import (
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/task"
)

// Replay of the Gate 1 20261102 BELT window (virtual minutes 446 to 457, clvl 12, Blood Moor, 37 gold, no healing
// potion, no portal source). The kit trip was priced at the scroll (100) and never fired; one potion (30) was in reach.
func TestKitTripNoPortalPotionPrice(t *testing.T) {
	for _, tc := range []struct {
		name string
		area int
		gold int
		tp   bool
		hp   int
		want bool
	}{
		{"window: Blood Moor, 37 gold, nothing", 2, 37, false, 0, true},
		{"window: Blood Moor, 29 gold", 2, 29, false, 0, false},
		{"Stony Field: the level chain leads home", 4, 37, false, 0, true},
		{"Underground Passage 1 has no portal and no waypoint, 0 gold", 10, 20, false, 0, false},
		{"portal source, 37 gold", 4, 37, true, 0, true},
		{"enough potions, scroll missing needs the scroll price", 2, 37, false, 4, false},
		{"enough potions, scroll affordable", 2, 120, false, 4, true},
	} {
		g, s := kitField(t, 12, tc.hp, tc.tp, tc.gold)
		g.w.set(func(w *world) { w.area = tc.area })
		s, _ = g.w.snapshot()
		plan, emerg, ok := g.r.choresDue(s)
		got := ok && strings.Contains(plan.Reason, "kit:")
		if got != tc.want {
			t.Errorf("%s: kit trip %v, want %v (%+v)", tc.name, got, tc.want, plan)
		}
		if got && emerg != (tc.hp == 0) {
			t.Errorf("%s: emergency %v with %d potions", tc.name, emerg, tc.hp)
		}
	}
}

// An empty kit trip waits the emergency cooldown (a quarter of ResupplyCooldownSec), not the full one.
func TestKitTripNoPotionEmergencyCooldown(t *testing.T) {
	g, s := kitField(t, 12, 0, false, 37)
	g.w.set(func(w *world) { w.area = 2 })
	s, _ = g.w.snapshot()
	g.r.setLastTrip(g.e.Clock.Now().Add(-90 * time.Second))
	if !g.r.tripDue(s) {
		t.Error("no healing potion anywhere: the trip is due after the emergency gap")
	}
	g, s = kitField(t, 12, 2, true, 37)
	g.r.setLastTrip(g.e.Clock.Now().Add(-90 * time.Second))
	if g.r.tripDue(s) {
		t.Error("two potions and a portal: the ordinary cooldown applies")
	}
}

// Stony Field (lines 446 to 448 of the window) has no walk home, but its waypoint was used (log "wp: used 1 -> 4"):
// with the waypoint activated the potion trip is allowed, without it it stays quiet.
func TestKitTripWaypointHome(t *testing.T) {
	for _, on := range []bool{false, true} {
		g, _ := kitField(t, 12, 0, false, 30)
		g.w.set(func(w *world) { w.area = 7 }) // Tamoe Highland: no walk home known
		q := &fakeQuests{}
		q.v.Quests = map[string]task.QuestState{}
		q.v.WPs = map[int]bool{7: on}
		g.r.d.Task = &TaskDeps{Quests: q}
		s, _ := g.w.snapshot()
		_, ok := g.r.kitTrip(s)
		if ok != on {
			t.Errorf("waypoint active %v: kit trip %v", on, ok)
		}
	}
}
