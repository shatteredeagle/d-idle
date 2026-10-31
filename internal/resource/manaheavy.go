package resource

import "d2lod-bot/internal/d2"

// Mana-heavy builds (docs/builds-endgame.md W7, docs/plan/data/potions-belt.json "mana_heavy_builds", U design, needs
// a live Hell check): War Cry and Blizzard packs drain the single mana column, so from Nightmare on these builds
// carry two mana columns and drink at 35% instead of the normal threshold. Normal keeps the default plan.

// ManaHeavyDrinkPct is the mana % below which a mana-heavy build drinks (json drink_mana_below_pct).
const ManaHeavyDrinkPct = 35

// manaHeavyBuilds is the json "builds" list (a test compares both).
var manaHeavyBuilds = map[string]bool{
	"barbarian_warcry_endgame90":     true,
	"sorceress_blizzard_endgame90":   true,
	"druid_tornado_endgame90":        true,
	"necromancer_summoner_endgame90": true,
}

// ManaHeavyBuild reports whether the leveler table name is a mana-heavy build.
func ManaHeavyBuild(build string) bool { return manaHeavyBuilds[build] }

// manaHeavyCols is the json by_clvl block (hp, mp, mp, hp / hp, mp, mp, rejuv / rejuv, mp, mp, rejuv).
func manaHeavyCols(level int) [4]Potion {
	hp, mp, rv := PotionLife, PotionMana, PotionRejuv
	switch {
	case level >= 61:
		return [4]Potion{rv, mp, mp, rv}
	case level >= 31:
		return [4]Potion{hp, mp, mp, rv}
	}
	return [4]Potion{hp, mp, mp, hp}
}

func (in Input) manaHeavy() bool { return in.Diff != d2.Normal && ManaHeavyBuild(in.Build) }

// manaNeed is ManaNeed(class), raised to 2 for a mana-heavy build (a Barbarian with War Cry included).
func (in Input) manaNeed() int {
	if in.manaHeavy() {
		return 2
	}
	return ManaNeed(in.Class)
}

// PlanForBuild is PlanForDiff with the mana-heavy columns for a mana-heavy build in Nightmare and Hell. The Hell
// rejuvenation columns of PlanForDiff do not apply to it: its own rows already carry them from clvl 31.
func PlanForBuild(c d2.Class, level int, d d2.Difficulty, build string) BeltPlan {
	p := PlanForDiff(c, level, d)
	if d != d2.Normal && ManaHeavyBuild(build) {
		p.Cols = manaHeavyCols(level)
	}
	return p
}
