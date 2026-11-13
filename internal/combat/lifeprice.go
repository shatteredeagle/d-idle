package combat

import (
	"slices"

	"d2lod-bot/internal/d2"
)

// Sacrifice and the other life-price attacks (Profile.LifePrice). Sacrifice takes 8 percent of the damage it deals
// from the caster (V read, docs/research/pal-skill-matrix.md section 3.1; whether the cap rule uses the life before
// or after the blow is U), so one swing costs a few points while the character has 80 to 200. The bot cannot drain
// itself into a chicken loop because the skill is offered only while all of these hold, and plain Attack (the next
// entry of every band) takes over the moment one fails:
//   - combat.sacrifice is on;
//   - life is at or above combat.sacrificeMinLifePct; once below it stays off until life is 10 points above (hysteresis,
//     so the left skill does not flip every swing);
//   - the belt holds combat.sacrificeMinPotions healing or rejuvenation potions;
//   - the merc, when the merc wiring reports one, is not dead.
//
// The state is noted once per fight tick from the snapshot (noteLifePrice), because attackSkills has no snapshot.

const lifePriceHysteresis = 10 // percentage points above the floor before a stopped life-price skill returns

// lifePriceState is what noteLifePrice remembered.
type lifePriceState struct {
	noted   bool
	lifePct int
	potions int
	mercBad bool
	stopped bool // below the floor: wait for floor + lifePriceHysteresis
}

// noteLifePrice records life, potions and the merc state for lifePriceOK.
func (f *Fighter) noteLifePrice(snap *d2.Snapshot) {
	if snap == nil || snap.Player == nil {
		return
	}
	p, t := snap.Player, f.tune()
	b := BeltOf(snap.Inventory, f.cat)
	s := &f.lifePrice
	s.noted, s.lifePct = true, pct(p.Life, p.MaxLife)
	s.potions = b.Count(PotionHealing) + b.Count(PotionRejuv)
	st, ok := f.mercSnapshot()
	s.mercBad = ok && st.Owned && !st.Alive
	switch {
	case s.lifePct < t.SacrificeMinLifePct:
		s.stopped = true
	case s.lifePct >= t.SacrificeMinLifePct+lifePriceHysteresis:
		s.stopped = false
	}
}

// lifePriceOK says whether a life-price skill may be used now.
func (f *Fighter) lifePriceOK() bool {
	t, s := f.tune(), &f.lifePrice
	return t.Sacrifice != 0 && s.noted && !s.stopped && s.potions >= t.SacrificeMinPotions && !s.mercBad
}

// costsLife reports whether the profile marks the skill as a life-price skill.
func (pr *Profile) costsLife(name string) bool { return slices.Contains(pr.LifePrice, name) }
