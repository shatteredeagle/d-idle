package merc

import "fmt"

// Gear policy. Weights are U judgments (docs/merc.md): defense and life first, then damage, resists in Hell only.

// Score is the merc-specific value of an item. Life counts 3 per point (it is rare on merc-wearable gear and the merc
// has no mana), damage 2 per average point, defense 1, resists 2 per point in Hell only, an empty socket 5, and an
// ethereal item 5 % more (no repair need for a merc's gear at this stage; its bonus defense is already in Defense).
func Score(it Item, d Difficulty) float64 {
	v := float64(it.Defense) + 3*float64(it.Life) + float64(it.DmgMin+it.DmgMax) + 5*float64(it.Sockets)
	if d == Hell {
		v += 2 * float64(it.Res)
	}
	if it.Ethereal {
		v *= 1.05
	}
	return v
}

// Allowed reports whether the hireling class may wear or wield it in slot (V d2decomped re/hireling.md, S for us): helm and armor are free for
// all; Rogue bow, Desert spear or polearm, Iron Wolf shield or one-handed sword, Barbarian sword (class 561).
func Allowed(class int, slot Slot, it Item) bool {
	switch slot {
	case SlotHelm:
		return it.Kind == KindHelm
	case SlotArmor:
		return it.Kind == KindArmor
	case SlotWeapon:
		switch class {
		case ClassRogue:
			return it.Kind == KindBow
		case ClassDesert:
			return it.Kind == KindSpear
		case ClassIronWolf, ClassBarb:
			return it.Kind == KindSword1H
		}
	case SlotShield:
		return class == ClassIronWolf && it.Kind == KindShield
	}
	return false
}

// ReqResult is the outcome of the requirement check, logged as `merc: reqcheck`.
type ReqResult struct {
	OK     bool
	Reason string // "ok", "level", "str", "dex", "stats unknown", "unidentified", "broken"
}

// MeetsReq checks identified, unbroken and the requirements against the MERC'S OWN level and, when strDex is on,
// his own Strength and Dexterity (d2decomped re/hireling.md: S; UNVERIFIED live, probe docs/merc.md). With strDex
// off only the level counts. A merc whose stats were not read fails closed under strDex.
func MeetsReq(it Item, m MercState, strDex bool) ReqResult {
	switch {
	case !it.Identified:
		return ReqResult{false, "unidentified"}
	case it.Broken:
		return ReqResult{false, "broken"}
	case m.Level < it.ReqLevel:
		return ReqResult{false, "level"}
	}
	if strDex {
		switch {
		case !m.StatsKnown:
			return ReqResult{false, "stats unknown"}
		case m.Str < it.ReqStr:
			return ReqResult{false, "str"}
		case m.Dex < it.ReqDex:
			return ReqResult{false, "dex"}
		}
	}
	return ReqResult{true, "ok"}
}

// EquipChoice is the verdict of EquipIfBetter.
type EquipChoice struct {
	Do      bool
	Slot    Slot
	Reason  string // "empty slot", "better by 23%", or why not ("not allowed", "level", "gain 4% < 10%", ...)
	GainPct int
	Req     ReqResult
}

// EquipIfBetter decides whether cand should replace the item in slot. An empty slot takes any allowed item that
// meets the requirements and scores above 0; an occupied one needs a gain of knobs.SwapGainPct or more.
func EquipIfBetter(m MercState, slot Slot, cand Item, d Difficulty, k Knobs) EquipChoice {
	ch := EquipChoice{Slot: slot}
	if !m.Known || !m.Owned {
		ch.Reason = ReasonUnknown
		return ch
	}
	if !Allowed(m.Class, slot, cand) {
		ch.Reason = "not allowed"
		return ch
	}
	if ch.Req = MeetsReq(cand, m, k.ReqStrDex); !ch.Req.OK {
		ch.Reason = ch.Req.Reason
		return ch
	}
	cs := Score(cand, d)
	cur := m.Gear[slot]
	if cur == nil {
		if cs <= 0 {
			ch.Reason = "worthless"
			return ch
		}
		ch.Do, ch.Reason = true, "empty slot"
		return ch
	}
	os := Score(*cur, d)
	if os <= 0 {
		ch.Do, ch.Reason, ch.GainPct = cs > 0, "replaces worthless", 100
		return ch
	}
	ch.GainPct = int((cs - os) * 100 / os)
	if ch.GainPct >= k.SwapGainPct && cs > os {
		ch.Do, ch.Reason = true, fmt.Sprintf("better by %d%%", ch.GainPct)
		return ch
	}
	ch.Reason = fmt.Sprintf("gain %d%% < %d%%", ch.GainPct, k.SwapGainPct)
	return ch
}

// StripStep is one item to take off.
type StripStep struct {
	Slot Slot
	Item Item
}

// StripPlan is what to do before a replacing hire destroys the merc's gear.
type StripPlan struct {
	Steps []StripStep
	// CellsNeeded is the pack space the taken-off items occupy (about 18 for helm 2x2, armor 2x3, polearm 2x4).
	CellsNeeded int
	// StashCells is how much of that must be stashed first because FreeCells is short (0 = none, -1 when FreeCells unknown: stash everything it needs).
	StashCells int
}

// PlanStrip lists the worn items, weapon first, and the space they need. freeCells < 0 means unknown, in which case
// the whole footprint is requested as stash space (the safe reading).
func PlanStrip(m MercState, freeCells int) StripPlan {
	var p StripPlan
	for _, s := range []Slot{SlotWeapon, SlotShield, SlotArmor, SlotHelm} {
		if it := m.Gear[s]; it != nil {
			p.Steps = append(p.Steps, StripStep{s, *it})
			p.CellsNeeded += it.Cells()
		}
	}
	switch {
	case p.CellsNeeded == 0:
	case freeCells < 0:
		p.StashCells = p.CellsNeeded
	case freeCells < p.CellsNeeded:
		p.StashCells = p.CellsNeeded - freeCells
	}
	return p
}

// PotionCtx is the fight situation for a merc potion.
type PotionCtx struct {
	FightTrivial   bool
	HealingPotions int // in the belt
}

// ShouldGiveHealing: only healing, only below knobs.PotionLifePct, only in a non-trivial fight, only from potions the
// player carries anyway. Rejuvenation is NOT given: d2decomped lists only healing, antidote and thawing for the
// merc (conflict with docs/sme/mercenary.md, UNVERIFIED, probe docs/merc.md).
func ShouldGiveHealing(m MercState, c PotionCtx, k Knobs) bool {
	return m.Known && m.Owned && m.Alive && m.LifePct > 0 && m.LifePct < k.PotionLifePct && !c.FightTrivial && c.HealingPotions > 0
}

// BuyPotionsForMerc is always false: the merc never gets potions bought for him (plan-v2 section 3).
func BuyPotionsForMerc() bool { return false }
