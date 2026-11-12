package env

// LootTuning holds the loot knobs. Owner: the loot agent; add fields, defaults and Knob entries here only.
type LootTuning struct {
	PickRadius      int // subtiles around the player in which ground items are considered
	PickStrictness  int // 0 loose (also sellable items), 1 normal, 2 strict (keepers only), 3 gold/runes/gems/uniques only
	PickTries       int // click attempts per item
	WalkDist        int // walk to within this many subtiles of an item before clicking it
	ConfirmMs       int // after a click, wait this long for the item to leave the ground
	GapMs           int // pause between two pickups
	GoldMinStack    int // ignore gold piles smaller than this (a ceiling: GoldPileShare lowers it in early areas)
	GoldPileShare   int // percent of an area's piles the min-pile bar must let through (model: data.PileMinWorth); 0 = the flat GoldMinStack only
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
	MemEnabled      int // 1: remember wanted drops beyond the pickup radius and walk back for them (docs/lootmemory.md)
	MemMinGold      int // smallest remembered gold pile worth a walk-back at clvl 1; it grows by 10 percent per clvl
	MemMaxDist      int // remembered drops farther than this (subtiles, straight line) are left
	MemMaxTrips     int // walk-back trips per visit of an area
	MemCap          int // remembered drops kept at most (lowest value evicted)
	GoldPriority    int // 1: gold piles are fetched right below survival (mid-fight when safe, before roaming on, before leaving an area; docs/goldprio.md)
	GoldGrabDist    int // subtiles: a pile this close is taken in the middle of a fight when the fight allows
	GoldSafeDist    int // subtiles: no hostile closer than this to the hero for a mid-fight grab
	GoldMaxDanger   int // highest pack danger (combat.Danger*) at which a mid-fight grab is still allowed
	GoldMinLifePct  int // life percent under which gold is ignored (survival first)
	GoldRetries     int // failed pickups of one pile (over all calls) before it is given up
}

func DefaultLootTuning() LootTuning {
	return LootTuning{
		PickRadius: 30, PickStrictness: 1, PickTries: 3, WalkDist: 4, ConfirmMs: 1500, GapMs: 150,
		GoldMinStack: 10, GoldPileShare: 50, FreeCellsKeep: 2, PotionBuffer: 2, RejuvSpare: 3, LowGoldBelow: 2000, MinValuePerCell: 100, ValuableMinGold: 1500, ValuableMinPrio: 70, CarryValuables: 0, MaxPerCall: 30,
		PoorGoldBelow: 300, PoorMinSell: 20,
		MemEnabled: 1, MemMinGold: 20, MemMaxDist: 120, MemMaxTrips: 6, MemCap: 64,
		GoldPriority: 1, GoldGrabDist: 20, GoldSafeDist: 8, GoldMaxDanger: 0, GoldMinLifePct: 60, GoldRetries: 4,
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
		lootK("goldMinStack", "Min gold pile", "gold", "Ignore gold piles smaller than this (a ceiling: see goldPileShare).", d.GoldMinStack, 0, 2000, false),
		lootK("goldPileShare", "Gold pile share", "percent", "The min-pile bar never skips more than this share of the area's piles: pile size is ilvl to 6 ilvl - 1 coins (treasure-trace section 4), so Blood Moor piles (1..5) would all fall under 10. 0 keeps the flat bar.", d.GoldPileShare, 0, 100, false),
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
		lootK("memEnabled", "Loot memory", "0/1", "1: remember wanted drops beyond the pickup radius and walk back for them at quiet moments.", d.MemEnabled, 0, 1, true),
		lootK("memMinGold", "Loot memory: min gold", "gold", "Smallest remembered gold pile worth a walk-back at clvl 1; the bar grows by 10 percent per character level.", d.MemMinGold, 0, 5000, false),
		lootK("memMaxDist", "Loot memory: max distance", "subtiles", "Remembered drops farther away than this are not fetched.", d.MemMaxDist, 20, 400, true),
		lootK("memMaxTrips", "Loot memory: trips per area", "count", "Walk-back trips per visit of an area.", d.MemMaxTrips, 0, 30, true),
		lootK("goldPriority", "Gold priority", "0/1", "1: gold piles are fetched right below survival: mid-fight when safe, before the next roam leg, and before an area is left (walk-back also for small piles, no trip cap). 0: old behaviour.", d.GoldPriority, 0, 1, true),
		lootK("goldGrabDist", "Gold: mid-fight grab distance", "subtiles", "A pile this close is picked up during a fight when no hostile is near.", d.GoldGrabDist, 5, 40, true),
		lootK("goldSafeDist", "Gold: hostile-free distance", "subtiles", "A mid-fight grab needs every hostile at least this far from the hero.", d.GoldSafeDist, 2, 30, true),
		lootK("goldMaxDanger", "Gold: max pack danger", "level", "Highest pack danger (0 none, 1 champions or shamans, 2 elite, 3 deadly) at which a mid-fight grab is still made.", d.GoldMaxDanger, 0, 3, true),
		lootK("goldMinLifePct", "Gold: min life", "percent", "Below this life percent gold is ignored (survival comes first).", d.GoldMinLifePct, 0, 100, false),
		lootK("goldRetries", "Gold: pickup retries", "count", "Failed pickups of one pile, over all calls, before it is given up.", d.GoldRetries, 1, 10, true),
		lootK("memCap", "Loot memory: size", "count", "Remembered drops kept at most; the lowest value is evicted.", d.MemCap, 4, 512, true),
	}
}
