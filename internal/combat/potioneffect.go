package combat

import (
	"time"

	"d2lod-bot/internal/d2"
)

// The game's healing potions heal over time (docs/probes/fid/fid-potions.md Q2, V read of FUN_005be3f0 and misc.txt):
// a drink adds `calc1` life (times the class factor) spread over `len` ticks of 40 ms. A further drink does not heal
// faster: the new amount joins what is left, rate = (old*left + amount) / (left + len), lasting left + len (Q5, V read).
// Only a higher tier raises the rate, and then only by the average.

const gameTick = 40 * time.Millisecond

// healPotions is misc.txt hp1..hp5 (D): life per drink and duration in ticks, indexed by tier-1.
var healPotions = [...]struct{ Life, Ticks int }{{30, 192}, {60, 160}, {100, 171}, {180, 192}, {320, 256}}

// lifeFactorPct is the class factor on a healing potion's amount in percent (V run, fid-potions.md Q3); the
// double-fill roll from vitality is not modelled (expected +(vit-1)/400, U). A class missing here has an effect with
// a duration but no rate: the decision uses only tier and end time, so it never guesses a factor.
var lifeFactorPct = map[d2.Class]int{
	d2.Amazon: 150, d2.Paladin: 150, d2.Assassin: 150,
	d2.Sorceress: 100, d2.Necromancer: 100, d2.Druid: 100,
	d2.Barbarian: 200,
}

// healEffect is the healing potion the character is still absorbing.
type healEffect struct {
	Tier int       // highest tier that went into the effect
	End  time.Time // when the recovery stops
	Rate float64   // life per second now; 0 when the class factor is unknown
}

func (e healEffect) active(now time.Time) bool { return e.Tier > 0 && now.Before(e.End) }

// Pending is the life the effect still has to give.
func (e healEffect) Pending(now time.Time) float64 {
	if !e.active(now) {
		return 0
	}
	return e.Rate * e.End.Sub(now).Seconds()
}

// drink returns the effect after drinking a potion of the tier at now (the old effect joins by the game's rule).
func (e healEffect) drink(now time.Time, tier int, class d2.Class) healEffect {
	if tier < 1 || tier > len(healPotions) {
		return e
	}
	hp := healPotions[tier-1]
	length := time.Duration(hp.Ticks) * gameTick
	var amount float64
	if f, ok := lifeFactorPct[class]; ok {
		amount = float64(hp.Life*f) / 100
	}
	left := time.Duration(0)
	var oldTotal float64
	if e.active(now) {
		left = e.End.Sub(now)
		oldTotal = e.Rate * left.Seconds()
	}
	out := healEffect{Tier: max(tier, e.Tier), End: now.Add(left + length)}
	if !e.active(now) {
		out.Tier = tier
	}
	out.Rate = (oldTotal + amount) / (left + length).Seconds()
	return out
}

// healAllowed reports whether another healing potion may be drunk: nothing is working, or life is below the
// emergency line, or the belt holds a higher tier than the one working (combat.potionEmergencyPct, docs/potions.md).
func (f *Fighter) healAllowed(now time.Time, lifePct int, belt Belt) bool {
	if !f.heal.active(now) || lifePct < f.tune().PotionEmergencyPct {
		return true
	}
	_, ok := belt.ColumnAbove(PotionHealing, f.heal.Tier)
	return ok
}

// ColumnAbove is the column of the lowest tier above tier for the kind (the cheapest potion that raises the rate).
func (b Belt) ColumnAbove(k PotionKind, tier int) (int, bool) {
	idx := -1
	for i, c := range b.Cols {
		if c.Kind != k || c.Count == 0 || c.Tier <= tier {
			continue
		}
		if idx < 0 || c.Tier < b.Cols[idx].Tier {
			idx = i
		}
	}
	return idx, idx >= 0
}
