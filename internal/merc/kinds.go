package merc

// Kind names for the four hireling classes and the equipment slots each can use, for display and for the later
// equip legality check. V: the slot sets from the d2d source (merc_slot, docs/research/merc-gear.md section 2.3) and the
// 1.14d decompilation summary; S: SME advice (docs/sme/mercenary.md); the Barbarian's second weapon rule and
// javelins on the Desert Guard are U.

// MercSlotKeys are the keys of a merc's worn items as the inventory page names them (the hero's keys with an m).
var MercSlotKeys = []string{"mhead", "mtorso", "mrhand", "mlhand"}

// KindName is the short name of a hireling class ("rogue", "desert", "ironwolf", "barbarian"), "" when unknown.
func KindName(class int) string {
	switch class {
	case ClassRogue:
		return "rogue"
	case ClassDesert:
		return "desert"
	case ClassIronWolf:
		return "ironwolf"
	case ClassBarb:
		return "barbarian"
	}
	return ""
}

// WearableKeys lists the slot keys a merc of that class can use at all: no belt, gloves, boots, amulet or ring for any
// of them; only the Iron Wolf takes a shield. nil for an unknown class.
func WearableKeys(class int) []string {
	switch class {
	case ClassRogue, ClassDesert, ClassBarb:
		return []string{"mhead", "mtorso", "mrhand"}
	case ClassIronWolf:
		return []string{"mhead", "mtorso", "mrhand", "mlhand"}
	}
	return nil
}
