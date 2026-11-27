package town

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

// The legacy seam (townLegacy, bankFirst, withdrawShop) must decide exactly as the inline expressions it replaced, at
// town.plan 0 to 3 and every knob value, so deleting the knobs later is a one-line change. Two builds, no game tables.
func TestTownLegacySeamMatchesTheOldExpressions(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Barbarian} {
		for stage := 0; stage <= 3; stage++ {
			for bf := 0; bf <= 1; bf++ {
				for ws := 0; ws <= 1; ws++ {
					tune := env.DefaultTuning()
					tune.Town.Plan, tune.Town.BankFirst, tune.Town.WithdrawShop = stage, bf, ws
					tn := &Town{e: &env.Env{Tune: &tune}}
					tu := tn.tune()
					if got, want := tn.bankFirst(), tu.BankFirst != 0 || stage >= 3; got != want {
						t.Errorf("%v stage %d bf %d: bankFirst %v, old %v", class, stage, bf, got, want)
					}
					if got, want := tn.withdrawShop(), !(tu.WithdrawShop == 0 && stage < 3); got != want {
						t.Errorf("%v stage %d ws %d: withdrawShop %v, old %v", class, stage, ws, got, want)
					}
					if got, want := tn.townLegacy(), stage < 3; got != want {
						t.Errorf("%v stage %d: townLegacy %v, want %v", class, stage, got, want)
					}
					old := tu.StashGoldBase + 10*tu.StashGoldPerLvl
					if tu.BankFirst != 0 || stage >= 3 {
						old = max(old, tn.kitGold())
					}
					if got := tn.goldKeep(10); got != old {
						t.Errorf("%v stage %d bf %d: goldKeep %d, old %d", class, stage, bf, got, old)
					}
				}
			}
		}
	}
}
