package party

// Scaling is the effect of the game's player count on monsters (docs/classes/party.md section 1). The count
// is game-wide: idle bots in town scale the monsters too (U for 1.14d). Used for planning and dashboards.
type Scaling struct {
	Players      int
	LifeFactor   float64 // monster life = base * (n+1)/2 (V Arreat); 1.5x at 2, 2.5x at 4
	XPFactor     float64 // same factor for monster XP (V)
	DamagePct    float64 // +6.25% per player above 1 (V for D2R, S for 1.14d)
	PartyXPBonus float64 // total XP to a party in one area rises ~34% per extra member in range (V)
	// XPPerKillRel is each of n equal-level members' share of one kill relative to a solo kill:
	// factor * (1 + 0.34*(n-1)) / n. Arithmetic is ours (U), formulas V.
	XPPerKillRel float64
	Notes        []string
}

// ScalingFor returns the effect for n players (1..8).
func ScalingFor(n int) Scaling {
	if n < 1 {
		n = 1
	}
	f := float64(n+1) / 2
	s := Scaling{Players: n, LifeFactor: f, XPFactor: f, DamagePct: 6.25 * float64(n-1), PartyXPBonus: 0.34 * float64(n-1)}
	s.XPPerKillRel = f * (1 + s.PartyXPBonus) / float64(n)
	if n > 1 {
		s.Notes = []string{
			"kills take longer (life) but pay more (XP); followers must stay within ~2 screens or lose the share",
			"keep levels within 5 of each other for full XP shares; a mule gets 5% at a 10 level gap",
			"Corpse Explosion and Death Sentry do not scale with players (weaker in bigger games); Crushing Blow is divided by 0.5+0.5*n",
			"quests are shared in the same act; a member that dies should stay down, the quest still counts",
		}
	}
	return s
}
