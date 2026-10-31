package resource

import "d2lod-bot/internal/d2"

// Potion kinds the policy talks about.
type Potion int

const (
	PotionNone Potion = iota
	PotionLife
	PotionMana
	PotionRejuv
)

func (p Potion) String() string { return [...]string{"none", "life", "mana", "rejuv"}[p] }

// PotionDecision is what to drink now.
type PotionDecision struct {
	Drink  Potion
	Best   bool // take the highest tier of that kind (emergency); false: the lowest, do not waste a super potion
	Reason string
}

// Chicken returns the leave-game life percent for the difficulty.
func (t Tuning) Chicken(d d2.Difficulty) int {
	if d == d2.Hell {
		return t.ChickenHell
	}
	return t.ChickenPct
}

// Poor reports whether the character is short of gold: life potions first, mana potions only when needed.
func Poor(in Input, t Tuning) bool { return in.Gold < t.PoorGold }

// bossRejuv is the earlier rejuvenation line against a champion/unique/boss pack once committed (SME advice, EXP): a
// rejuvenation is instant and also restores mana, healing potions are for trash.
func bossRejuv(in Input, t Tuning, lp int) bool {
	return in.Hostiles > 0 && in.PackDanger >= 1 && in.Belt.RV > 0 && t.DrinkRejuvBossPct > 0 && lp < t.DrinkRejuvBossPct
}

// PotionPolicy decides at most one drink. Gap timing (PotionGapMs) is the caller's.
//
//	life <= chicken or < rejuv%: rejuvenation if held (best tier), else healing
//	life <  drink%:             healing (lowest tier that exists in a fight), out of a fight only below the engage line
//	mana <  mana%:              mana potion, with the poor-character rules below
//
// Poor characters (gold < PoorGold) spend mana potions only when there is no fallback attack (melee/free) or the pack is
// dangerous: a healing potion is what keeps them alive, mana is recoverable by resting and melee.
func PotionPolicy(in Input, t Tuning) PotionDecision {
	if in.InTown || in.MaxLife <= 0 || in.Life <= 0 {
		return PotionDecision{Reason: "town or dead"}
	}
	lp := in.LifePct()
	b := in.Belt
	fight := in.Hostiles > 0
	switch {
	case lp <= t.Chicken(in.Diff) || lp < t.DrinkRejuvPct || bossRejuv(in, t, lp):
		if b.RV > 0 {
			return PotionDecision{Drink: PotionRejuv, Best: true, Reason: "life critical, rejuvenation"}
		}
		if b.HP > 0 {
			return PotionDecision{Drink: PotionLife, Best: true, Reason: "life critical, healing"}
		}
	case lp < t.DrinkLifePct:
		if !fight && lp >= t.MinEngageLifePct {
			break // out of a fight PreEngage/TownTripTrigger decide
		}
		if thinBeltRetreat(in, t) {
			return PotionDecision{Reason: "belt thin: leave the fight instead of drinking"}
		}
		if b.HP > 0 {
			return PotionDecision{Drink: PotionLife, Reason: "life low"}
		}
		if b.RV > 0 {
			return PotionDecision{Drink: PotionRejuv, Best: true, Reason: "life low, no healing potion"}
		}
	}
	if fight && in.manaNeed() > 0 && in.ManaPct() < manaDrinkPct(in, t) && in.MaxMana > 0 {
		cheap := in.cheapestCast()
		if cheap > 0 && t.ManaPaysCasts > 0 && in.Mana >= cheap*t.ManaPaysCasts {
			return PotionDecision{Reason: "mana still pays for casts"}
		}
		if Poor(in, t) && (in.hasMelee() || in.hasFree()) && in.PackDanger < 2 {
			return PotionDecision{Reason: "poor: melee fallback instead of a mana potion"}
		}
		if b.MP > 0 {
			return PotionDecision{Drink: PotionMana, Best: in.PackDanger >= 2, Reason: "mana low"}
		}
	}
	return PotionDecision{Reason: "nothing to drink"}
}

// BeltPlan is the column layout for a class at a level.
type BeltPlan struct {
	Cols     [4]Potion // front-to-back column kinds, hotkeys 1..4
	MaxTier  int       // highest tier worth carrying (the town band cap)
	PrefTier int       // tier to buy when gold allows
}

// PlanFor returns the belt plan. It differs from internal/town/belt/plan.json by giving the mana-hungry classes two
// mana columns before clvl 31 (the Combi starvation, docs/combat.md) and Barbarians none (R, tag U for the exact split).
func PlanFor(c d2.Class, level int) BeltPlan {
	var p BeltPlan
	switch {
	case level <= 12:
		p.MaxTier = 2
	case level <= 30:
		p.MaxTier = 4
	default:
		p.MaxTier = 5
	}
	switch {
	case level <= 7:
		p.PrefTier = 1
	case level <= 15:
		p.PrefTier = 2
	case level <= 27:
		p.PrefTier = 3
	case level <= 44:
		p.PrefTier = 4
	default:
		p.PrefTier = 5
	}
	if p.PrefTier > p.MaxTier {
		p.PrefTier = p.MaxTier
	}
	hp, mp, rv := PotionLife, PotionMana, PotionRejuv
	switch ManaNeed(c) {
	case 2:
		p.Cols = [4]Potion{hp, hp, mp, mp}
		if level >= 31 {
			p.Cols = [4]Potion{hp, hp, mp, rv}
		}
		if level >= 61 {
			p.Cols = [4]Potion{hp, rv, mp, rv}
		}
	case 1:
		p.Cols = [4]Potion{hp, hp, hp, mp}
		if level >= 31 {
			p.Cols = [4]Potion{hp, hp, mp, rv}
		}
		if level >= 61 {
			p.Cols = [4]Potion{hp, rv, mp, rv}
		}
	default:
		p.Cols = [4]Potion{hp, hp, hp, hp}
		if level >= 31 {
			p.Cols = [4]Potion{hp, hp, hp, rv}
		}
		if level >= 61 {
			p.Cols = [4]Potion{hp, rv, hp, rv}
		}
	}
	return p
}

// Prices are vendor prices per tier (index 0 = tier 1). U: defaults are placeholders; callers pass the stock prices.
type Prices struct{ HP, MP, RV [5]int }

func DefaultPrices() Prices {
	return Prices{HP: [5]int{30, 60, 150, 300, 600}, MP: [5]int{30, 60, 120, 240, 480}, RV: [5]int{150, 450, 0, 0, 0}}
}

// Buys is the shopping list.
type Buys struct {
	HP, MP, RV             int
	TierHP, TierMP, TierRV int
	Cost                   int
	Spendable              int
}

// capacity of the columns of one kind.
func (p BeltPlan) capacity(k Potion, rows int) int {
	if rows < 1 {
		rows = 1
	}
	n := 0
	for _, c := range p.Cols {
		if c == k {
			n += rows
		}
	}
	return n
}

// PlanBuys builds the potion shopping list within the gold budget.
//
// Spendable = (gold - ReserveGold - 100 when no town portal) and, unless poor, at most PotionGoldPct of it. Priority: first
// HPFirst healing potions, then MPFirst mana potions (casters), then fill healing, fill mana, then rejuvenation. The tier is the highest
// not above PrefTier whose "first" purchases fit; a poor character therefore buys minor potions.
func PlanBuys(in Input, t Tuning, pr Prices) Buys {
	plan := PlanForBuild(in.Class, in.Level, in.Diff, in.Build)
	rows := in.Belt.Rows
	spend := in.Gold - t.ReserveGold
	if !in.HasTP {
		spend -= t.KitTPGold // a portal scroll comes first (town chore)
	}
	if spend < 0 {
		spend = 0
	}
	if !Poor(in, t) {
		// The share never cuts into the survival kit the gear shop reserves (KitShortfall, minus the portal bought above).
		kit := KitShortfall(true, in.Belt.HP+in.Belt.RV, in.Belt.MP, t)
		spend = max(spend*t.PotionGoldPct/100, min(spend, kit))
	}
	b := Buys{Spendable: spend}
	needHP := plan.capacity(PotionLife, rows) - in.Belt.HP
	needMP := plan.capacity(PotionMana, rows) - in.Belt.MP
	needRV := plan.capacity(PotionRejuv, rows) - in.Belt.RV
	if needHP < 0 {
		needHP = 0
	}
	if needMP < 0 {
		needMP = 0
	}
	if needRV < 0 {
		needRV = 0
	}
	firstHP := min(t.HPFirst, needHP)
	firstMP := 0
	if in.manaNeed() >= 1 {
		firstMP = min(t.MPFirst, needMP)
	}
	tier := plan.PrefTier
	for tier > 1 && firstHP*pr.HP[tier-1]+firstMP*pr.MP[min(tier, 5)-1] > spend {
		tier--
	}
	hpP, mpP := pr.HP[tier-1], pr.MP[tier-1]
	buy := func(n *int, want, price int) {
		for *n < want && price > 0 && b.Cost+price <= spend {
			*n++
			b.Cost += price
		}
	}
	buy(&b.HP, firstHP, hpP)
	buy(&b.MP, firstMP, mpP)
	buy(&b.HP, needHP, hpP)
	if in.manaNeed() >= 1 {
		buy(&b.MP, needMP, mpP)
	}
	rvTier := 1
	if plan.PrefTier >= 4 {
		rvTier = 2
	}
	buy(&b.RV, needRV, pr.RV[rvTier-1])
	b.TierHP, b.TierMP, b.TierRV = tier, tier, rvTier
	if b.HP == 0 {
		b.TierHP = 0
	}
	if b.MP == 0 {
		b.TierMP = 0
	}
	if b.RV == 0 {
		b.TierRV = 0
	}
	return b
}

func manaDrinkPct(in Input, t Tuning) int {
	if in.manaHeavy() {
		return max(t.DrinkManaPct, ManaHeavyDrinkPct)
	}
	return t.DrinkManaPct
}
