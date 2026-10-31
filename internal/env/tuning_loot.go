package env

// LootTuning holds the loot knobs. Owner: the loot agent; add fields, defaults and Knob entries here only.
type LootTuning struct {
	PickRadius      int // subtiles around the player in which ground items are considered
	PickStrictness  int // 0 loose (also sellable items), 1 normal, 2 strict (keepers only), 3 gold/runes/gems/uniques only
	PickTries       int // click attempts per item
	WalkDist        int // walk to within this many subtiles of an item before clicking it
	ConfirmMs       int // after a click, wait this long for the item to leave the ground
	GapMs           int // pause between two pickups
	GoldMinStack    int // ignore gold piles smaller than this
	FreeCellsKeep   int // inventory cells that must stay free after a pickup (the pickit may override for gold and runes)
	PotionBuffer    int // extra potions of a needed type to carry in the pack beyond the belt
	RejuvSpare      int // rejuvenation potions carried in total, belt plus pack (the rest is left or sold, Full preferred)
	LowGoldBelow    int // below this much gold, also pick items worth selling
	MinValuePerCell int // sell price per inventory cell an item needs to be worth picking in low-gold mode
	ValuableMinGold int // an unequipped item worth at least this (estimated sell value) counts as a valuable
	ValuableMinPrio int // ... or a pickit rule priority at least this (runes, gems, uniques sit at 70+)
	CarryValuables  int // valuables the bot may carry out of town; the rest are stashed before a dangerous trip
	MaxPerCall      int // pickups per PickUp call
	PoorGoldBelow   int // below this gold the looter takes every gold pile and any junk weapon or armor worth PoorMinSell (a naked, broke character earns its rebuy that way)
	PoorMinSell     int // estimated sell value an item needs to be picked in poor mode
}

func DefaultLootTuning() LootTuning {
	return LootTuning{
		PickRadius: 30, PickStrictness: 1, PickTries: 3, WalkDist: 4, ConfirmMs: 1500, GapMs: 150,
		GoldMinStack: 10, FreeCellsKeep: 2, PotionBuffer: 0, RejuvSpare: 3, LowGoldBelow: 2000, MinValuePerCell: 100, ValuableMinGold: 1500, ValuableMinPrio: 70, CarryValuables: 0, MaxPerCall: 30,
		PoorGoldBelow: 300, PoorMinSell: 6,
	}
}

func lootK(key, label, unit, desc string, def, min, max int, higher bool) Knob {
	return Knob{Key: "loot." + key, Label: label, Unit: unit, Desc: desc,
		Default: float64(def), Min: float64(min), Max: float64(max), HigherIsBetter: higher}
}

func lootKnobs() []Knob {
	d := DefaultLootTuning()
	return []Knob{
		lootK("pickRadius", "Pickup radius", "subtiles", "Ground items within this distance are considered.", d.PickRadius, 5, 60, true),
		lootK("pickStrictness", "Filter strictness", "level", "0 loose (also items worth selling), 1 normal, 2 keepers only, 3 gold/runes/gems/uniques only.", d.PickStrictness, 0, 3, false),
		lootK("pickTries", "Pickup attempts", "count", "Click attempts per item.", d.PickTries, 1, 6, true),
		lootK("walkDist", "Walk-to distance", "subtiles", "Walk to within this distance before clicking an item.", d.WalkDist, 2, 8, false),
		lootK("confirmMs", "Pickup confirm wait", "ms", "Wait after a click for the item to leave the ground.", d.ConfirmMs, 400, 4000, true),
		lootK("gapMs", "Pickup gap", "ms", "Pause between two pickups.", d.GapMs, 0, 1000, false),
		lootK("goldMinStack", "Min gold pile", "gold", "Ignore gold piles smaller than this.", d.GoldMinStack, 0, 2000, false),
		lootK("freeCellsKeep", "Free cells kept", "cells", "Inventory cells that must stay free after a pickup.", d.FreeCellsKeep, 0, 12, true),
		lootK("potionBuffer", "Potion buffer", "count", "Extra needed-type potions carried in the pack beyond the belt.", d.PotionBuffer, 0, 10, true),
		lootK("rejuvSpare", "Rejuvenation cap", "count", "Rejuvenation potions carried in total (belt plus pack); more are left on the ground or sold, Full Rejuvenation preferred over regular. Pack ones are moved to a free belt slot in town so the belt hotkey can drink them.", d.RejuvSpare, 0, 6, true),
		lootK("lowGoldBelow", "Low gold mode below", "gold", "Below this gold, also pick items worth selling.", d.LowGoldBelow, 0, 100000, false),
		lootK("minValuePerCell", "Min value per cell", "gold", "Sell value per cell an item needs to be picked in low-gold mode.", d.MinValuePerCell, 0, 5000, false),
		lootK("valuableMinGold", "Valuable: min value", "gold", "Unequipped items worth at least this are valuables (stash before dangerous trips).", d.ValuableMinGold, 100, 100000, false),
		lootK("valuableMinPrio", "Valuable: min rule prio", "prio", "Items whose pickit rule prio is at least this are valuables.", d.ValuableMinPrio, 1, 100, false),
		lootK("carryValuables", "Valuables carried out", "count", "Valuables allowed in the pack outside town (insurance: the rest stay in the stash).", d.CarryValuables, 0, 20, true),
		lootK("poorGoldBelow", "Poor mode below", "gold", "Below this gold every gold pile and junk weapon or armor worth selling is picked.", d.PoorGoldBelow, 0, 10000, false),
		lootK("poorMinSell", "Poor mode min sell value", "gold", "Estimated sell value a junk item needs to be picked in poor mode.", d.PoorMinSell, 0, 500, false),
		lootK("maxPerCall", "Pickups per call", "count", "Upper bound on pickups in one PickUp call.", d.MaxPerCall, 1, 100, true),
	}
}
