package combat

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Scenario combat.buffs-stay-up, offline: 15 virtual minutes with the upkeep driver on (mode 2). The buff state is
// on for life seconds after each cast (the game's side, a fixture); the driver must keep it up 90 percent of the
// time and must not recast in a spam (casts bounded by the lifetime). Two builds: a Sorceress armor and a
// Barbarian shout, both read from the table row, none from Go.
func TestBuffsStayUp(t *testing.T) {
	const life, total = 120, 15 * 60
	fa := frozenArmorRow(true)
	fa.StateID = d2.StateFrozenArmor
	bo := &data.Skill{ID: 149, Key: "Battle Orders", Name: "Battle Orders", Mana: 5, ManaShift: 8, MinMana: 5, StateID: 32}
	for _, b := range []upkeepBuild{{d2.Sorceress, fa}, {d2.Barbarian, bo}} {
		h, w := upkeepSetup(t, b, 2)
		seen, up, expire := 0, 0, -1
		for s := 0; s < total; s++ {
			setState(w.snap.Player, b.row.StateID, s < expire)
			buffNow(t, h)
			if len(w.casts) > seen { // a cast refreshes the state for life seconds
				seen = len(w.casts)
				expire = s + life
				setState(w.snap.Player, b.row.StateID, true)
			}
			if s < expire {
				up++
			}
			h.s.Sleep(time.Second)
		}
		if up*100 < total*90 {
			t.Errorf("%v: state up %d of %d s, want 90 percent", b.class, up, total)
		}
		if max := total/life + 3; len(w.casts) > max {
			t.Errorf("%v: %d casts in %d s of a %d s buff, want at most %d (recast spam)", b.class, len(w.casts), total, life, max)
		}
	}
}
