package combat

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Test catalog: the classes used by the hand-built snapshots.
const (
	clsFallen                        = 1 // melee grunt
	clsShaman                        = 2 // kill first
	clsTownNPC                       = 3 // Kashya-like
	clsColdImm                       = 4 // immune to cold
	clsAllImmun                      = 5 // immune to everything the build casts
	clsBoss                          = 6 // monstats boss
	clsArcher                        = 7 // ranged
	itHP1, itHP3, itMP1, itRV, itTSC = 587, 589, 592, 600, 529
)

func testCatalog() MapCatalog {
	mk := func(key string) *data.Monster {
		return &data.Monster{Key: key, Name: key, Killable: true, Enabled: true, Melee: true}
	}
	fallen, shaman := mk("fallen1"), mk("fallenshaman1")
	npc := &data.Monster{Key: "kashya", Name: "Kashya", NPC: true, Interact: true}
	cold := mk("coldimmune")
	cold.Res[0][data.Cold] = 100
	all := mk("everything")
	for i := range all.Res[0] {
		all.Res[0][i] = 100
	}
	all.Res[0][data.Physical] = 0
	boss := mk("andariel")
	boss.Boss = true
	arch := mk("archer")
	arch.Melee, arch.Ranged = false, true
	skill := func(id int, key, et string, mana, lvl, shift, delay int) *data.Skill {
		return &data.Skill{ID: id, Key: key, Name: key, EType: et, Mana: mana, LvlMana: lvl, ManaShift: shift, MinMana: 1, Delay: delay}
	}
	return MapCatalog{
		Monsters: map[int]*data.Monster{clsFallen: fallen, clsShaman: shaman, clsTownNPC: npc, clsColdImm: cold, clsAllImmun: all, clsBoss: boss, clsArcher: arch},
		Skills: map[int]*data.Skill{
			36: skill(36, "Fire Bolt", "fire", 5, 0, 7, 0),
			38: skill(38, "Charged Bolt", "ltng", 24, 4, 5, 0),
			42: skill(42, "Static Field", "ltng", 9, 0, 8, 0),
			40: skill(40, "Frozen Armor", "", 7, 0, 8, 0),
			59: skill(59, "Blizzard", "cold", 23, 1, 8, 45),
			55: skill(55, "Glacial Spike", "cold", 20, 1, 7, 0),
			45: skill(45, "Ice Blast", "cold", 12, 1, 7, 0),
		},
		Items: map[int]ItemClass{
			itHP1: {Potion: PotionHealing, Tier: 1}, itHP3: {Potion: PotionHealing, Tier: 3},
			itMP1: {Potion: PotionMana, Tier: 1}, itRV: {Potion: PotionRejuv, Tier: 2},
			itTSC: {Portal: true},
		},
	}
}

func mon(id uint32, class, x, y int) d2.Unit {
	return d2.Unit{ID: id, Type: d2.TypeMonster, Class: class, Mode: d2.MonModeStand, X: x, Y: y, Monster: &d2.MonsterInfo{LifePct: 100}}
}

func potion(class, col int) d2.Unit {
	return d2.Unit{Type: d2.TypeItem, Class: class, Item: &d2.Item{Where: d2.ContBelt, GX: col}}
}

func baseSnap() *d2.Snapshot {
	return &d2.Snapshot{
		State: d2.StateInGame,
		Player: &d2.Player{ID: 1, Class: d2.Sorceress, X: 100, Y: 100, Area: 2, Level: 1, Life: 100, MaxLife: 100, Mana: 40, MaxMana: 40,
			Mode: 1, Skills: []d2.Skill{{ID: 36, Level: 0, Bonus: 1}}},
		Inventory: &d2.Inventory{},
	}
}
