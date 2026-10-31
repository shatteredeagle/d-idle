package resource

import "d2lod-bot/internal/d2"

// Merc policy (agent MERCREAD). Tags: V verified, S secondary, U unverified.

// MercReviveCost is the price of reviving a merc of the given level: min(int(lvl*lvl/2)*15, 50000). S: Maxroll page
// (D2R wording; the formula is plausibly unchanged in 1.14d), U live. Level 0 (unknown) returns 0.
func MercReviveCost(level int) int {
	if level <= 0 {
		return 0
	}
	return min((level*level/2)*15, 50000)
}

// SetMerc fills the merc fields from a memory read. known is the caller's sticky "a merc was hired or seen this run":
// a merc that died may vanish from the unit table (U), and a vanished known merc must read as dead, not as no merc.
// An absent unit with known=false leaves the fields zero (no merc, nothing to revive).
func (in *Input) SetMerc(m d2.Merc, known bool) {
	in.MercKnown = known || m.Present
	in.MercLifePct, in.MercLevel = 0, m.Level
	if m.Alive {
		// a live unit that reads 0% (unverified life stat) still counts as alive, so floor at 1
		in.MercLifePct = max(m.LifePct, 1)
	}
}

// MercReviveWanted reports whether a revive trip is due: a known merc is dead and the gold covers the revive price plus
// reserve. An unknown merc level (0) prices at 0, so the caller should read the quote from the dialog and stop when
// gold fell short; reserve keeps potion money.
func (in Input) MercReviveWanted(reserve int) bool {
	return in.MercDead() && in.Gold >= MercReviveCost(in.MercLevel)+reserve
}
