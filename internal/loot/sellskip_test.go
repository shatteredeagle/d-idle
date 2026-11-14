package loot_test

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/loot"
)

// sellskip: the early-gear "wear what is better" keep is only valid when the bot wears it. With the player deciding
// gear (the default) a poor clvl 14 character's plain helm and belt are junk to sell, not "kept for the empty slot"
// (they sat unworn in the pack for 6.6 h and the sell chore skipped every visit).
func TestEarlyGearKeepNeedsTheBotToWearIt(t *testing.T) {
	for _, cls := range []d2.Class{d2.Paladin, d2.Sorceress} {
		l, r := newLooter(t)
		w := r.W
		dress(w)
		w.Player.Class, w.Player.Level, w.Player.Gold = cls, 14, 40
		w.Player.Strength, w.Player.Dexterity = 30, 30
		r.Env.Tune.Resource.EarlyGearMax = 15
		for _, code := range []string{"cap", "lbt"} {
			u := w.NewItem(code, 1)
			r.Env.Tune.Resource.GearDecisions = 0
			s, _ := w.Snapshot()
			if v := l.EvaluatorFor(s)(u); v.Rule == "empty slot" {
				t.Errorf("%v %s: kept for an empty slot while the player decides gear: %v", cls, code, v)
			}
			r.Env.Tune.Resource.GearDecisions = 1
			s, _ = w.Snapshot()
			if v := l.EvaluatorFor(s)(u); v.Action != loot.Keep {
				t.Errorf("%v %s: the bot-gear policy lost its keep: %v", cls, code, v)
			}
		}
	}
}
