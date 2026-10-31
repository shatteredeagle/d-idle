package combat

import (
	"fmt"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Resist bonuses of the monster modifiers (V: Arreat bonus.shtml, docs/difficulty.md section 4): Magic Resistant
// +40 fire/cold/lightning, the three Enchanted +75 to their own element, Mana Burn +20 magic. A bonus is not applied
// when it would create a third immunity (Arreat: "only if not already a third immunity").
const (
	magicResistantBonus = 40
	enchantedBonus      = 75
	manaBurnMagicBonus  = 20
)

// AffixResists adds the modifier resist bonuses to monstats base resistances. Use it only for the monstats fallback:
// a live stat list already contains them. The immune count looks at the four elements plus magic.
func AffixResists(r [6]int, a Affix) [6]int {
	add := func(t data.DamageType, n int) {
		if r[t] >= 100 {
			return
		}
		nr := r
		nr[t] += n
		if nr[t] >= 100 && immuneCount(r) >= 2 {
			return // would be a third immunity
		}
		r = nr
	}
	if a&AffixMagicResistant != 0 {
		add(data.Fire, magicResistantBonus)
		add(data.Cold, magicResistantBonus)
		add(data.Lightning, magicResistantBonus)
	}
	if a&AffixFireEnchanted != 0 {
		add(data.Fire, enchantedBonus)
	}
	if a&AffixColdEnchanted != 0 {
		add(data.Cold, enchantedBonus)
	}
	if a&AffixLightningEnchanted != 0 {
		add(data.Lightning, enchantedBonus)
	}
	if a&AffixManaBurn != 0 {
		add(data.Magic, manaBurnMagicBonus)
	}
	return r
}

func immuneCount(r [6]int) int {
	n := 0
	for _, t := range []data.DamageType{data.Magic, data.Fire, data.Lightning, data.Cold, data.Poison} {
		if r[t] >= 100 {
			n++
		}
	}
	return n
}

// Pack danger levels (resource.Input.PackDanger scale).
const (
	DangerNone   = 0
	DangerShaman = 1 // champions, ranged shamans, mild modifiers
	DangerElite  = 2 // boss, deadly modifier, death-weight avoid
	DangerDeadly = 3 // Hell uniques the character is too low for
	// deadlyLevel is the character level under which a Hell unique with a deadly modifier is rated 3. U: the same 75
	// as avoidHellLevel (docs/difficulty.md P1 item 15), a planning number, not a measurement.
	deadlyLevel     = avoidHellLevel
	deadlyMinions   = 5
	minionDangerMin = 3 // in Hell a pack of this many unique minions is already DangerElite
)

// TargetDanger rates one hostile (rules of docs/difficulty.md section 4 and P1 items 9-15; the numbers are V from
// Arreat, the thresholds R). Fire Enchanted explodes on death only in Normal/Nightmare (Hell 0.75-1.25% of life,
// negligible), so it counts there only.
func TargetDanger(t Target, diff d2.Difficulty, clvl int) int {
	d := DangerNone
	switch {
	case t.Boss || t.Danger >= 1:
		d = DangerElite
	case t.Elite:
		d = DangerShaman
	}
	a := t.Affixes
	hard := diff != d2.Normal
	if hard && a&(AffixExtraFast|AffixTeleportation|AffixExtraStrong|AffixManaBurn|AffixCursed) != 0 {
		d = max(d, DangerElite)
	}
	if a&(AffixExtraFast|AffixExtraStrong|AffixTeleportation|AffixManaBurn) != 0 {
		d = max(d, DangerShaman)
	}
	if a&(AffixStoneSkin|AffixMagicResistant) != 0 {
		d = max(d, DangerShaman) // slow to kill, not lethal
	}
	if a&AffixAura != 0 {
		d = max(d, DangerElite)
	}
	if a&AffixFireEnchanted != 0 && diff != d2.Hell {
		d = max(d, DangerShaman)
	}
	if diff == d2.Hell && clvl < deadlyLevel && !t.Boss {
		if a&avoidHell != 0 || a&(AffixExtraStrong|AffixExtraFast) == AffixExtraStrong|AffixExtraFast {
			d = DangerDeadly
		}
	}
	return d
}

// PackDanger is the highest TargetDanger in the pack, raised to DangerDeadly in Hell by a crowd of unique minions.
func PackDanger(ts []Target, diff d2.Difficulty, clvl int) int {
	d, minions := DangerNone, 0
	for _, t := range ts {
		d = max(d, TargetDanger(t, diff, clvl))
		if t.Unit.Monster != nil && t.Unit.Monster.Kind() == d2.MonKindMinion {
			minions++
		}
	}
	if diff == d2.Hell && minions >= deadlyMinions && clvl < deadlyLevel {
		d = DangerDeadly
	} else if diff != d2.Normal && minions >= minionDangerMin {
		d = max(d, DangerElite)
	}
	return d
}

// attackBlocked reports whether no known attack (nor the Static Field fallback) can hurt the target. Mana and
// cooldowns are ignored: this is about immunity only.
func (f *Fighter) attackBlocked(p *d2.Player, tg *Target) bool {
	if !tg.HasRes {
		return false
	}
	if _, why := PickAttack(f.attackSkills(p), tg, 1<<20, nil); why != PickImmune {
		return false
	}
	return !f.hasFallback(p, tg)
}

func (f *Fighter) hasFallback(p *d2.Player, tg *Target) bool {
	sf, ok := KnownSkills(p, f.cat, f.keys)["Static Field"]
	return ok && !tg.Immune(sf.Elem)
}

// packSkips are the immune pack members to leave alone before approaching them (P1 item 5): a unique/champion/boss
// that none of our attacks can hurt, or any immune member when at least three are in view. A single immune trash
// monster is still tried (the old behaviour: the merc or a fallback may handle it and the loop ignores it on its own).
func (f *Fighter) packSkips(p *d2.Player, hs []Target) []Target {
	var blocked []Target
	for _, h := range hs {
		if !h.Avoid && f.attackBlocked(p, &h) {
			blocked = append(blocked, h)
		}
	}
	var out []Target
	for _, h := range blocked {
		if h.Elite || h.Boss || len(blocked) >= 3 {
			out = append(out, h)
		}
	}
	return out
}

func immuneDetail(t Target) string { return fmt.Sprintf("pack immune, res %v", t.Res) }
