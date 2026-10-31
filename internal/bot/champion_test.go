package bot

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// A champion kills as an elite: autopsy label set, killerBoss false; a named unique/superunique is a boss. Neither
// the champion nor its kind writes into the danger table (that is for spirals, see deathPolicy).
func TestDeathLabelsChampionNotBoss(t *testing.T) {
	cases := []struct {
		name               string
		flags              uint8
		mname              string
		wantBoss, wantChmp bool
	}{
		{"champion", 1 << 2, "", false, false},
		{"named unique", 1 << 3, "Bishibosh", true, false},
		{"superunique unnamed flag", 1 << 3, "", true, false},
		{"plain", 1 << 1, "", false, false},
	}
	cases[0].wantChmp = true
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			g := newRig(t)
			now := g.r.e.Clock.Now()
			p := &d2.Player{Level: 5, Class: d2.Sorceress, Life: 50, MaxLife: 100, Area: 2, X: 100, Y: 100}
			u := d2.Unit{ID: 9, Type: d2.TypeMonster, Class: 10, Mode: d2.MonModeStand, X: 103, Y: 100,
				Monster: &d2.MonsterInfo{LifePct: 100, Flags: c.flags, Name: c.mname}}
			s := &d2.Snapshot{State: d2.StateInGame, Player: p, Monsters: []d2.Unit{u}}
			g.r.obMu.Lock()
			g.r.ob.lastGood.area = 2
			g.r.trackAlive(s, now)
			g.r.recordDeath(&d2.Player{Level: 5, Life: 0, MaxLife: 100, Area: 2}, now.Add(time.Second))
			rec := g.r.dlog.last()
			g.r.obMu.Unlock()
			if rec.KillerBoss != c.wantBoss || rec.KillerChampion != c.wantChmp {
				t.Fatalf("boss=%v champion=%v, want %v %v", rec.KillerBoss, rec.KillerChampion, c.wantBoss, c.wantChmp)
			}
		})
	}
}
