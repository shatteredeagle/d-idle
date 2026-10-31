package env

// CampaignTuning holds the campaign policy knobs read by internal/task (optional quests).
type CampaignTuning struct {
	ImbueMinLevel     int // clvl below which the optional Imbue quest (Tools of the Trade) is skipped
	DenMinLevelCaster int // clvl at which a caster (or a bow-less Amazon) enters the Den of Evil
	DenMinLevelMelee  int // same for melee classes and a bowed Amazon
	ImbueNeedsMerc    int // 1: the Imbue quest also needs a living merc (the Barracks walk is the Soaky death road)
}

// RouteTuning holds the route guard knob (internal/task/route.go).
type RouteTuning struct {
	MonasteryMinLevel     int // clvl that opens the Monastery road (Tamoe Highland to the Catacombs) without a living merc; 0 = guard off
	MonasteryMinLevelMerc int // same with a living merc; 0 = a living merc opens the road at any level (the old rule)
	TowerMinLevel         int // clvl that opens the Forgotten Tower (areas 20-25, Countess) without a living merc; 0 = guard off
	TowerMinLevelMerc     int // same with a living merc (used while TowerNeedsMerc is 0)
	TowerNeedsMerc        int // 1: a living merc opens the tower at any level (the old rule); 0: a living merc only lowers the floor to TowerMinLevelMerc
}

func DefaultCampaignTuning() CampaignTuning {
	return CampaignTuning{ImbueMinLevel: 20, ImbueNeedsMerc: 1, DenMinLevelCaster: 5, DenMinLevelMelee: 3}
}
func DefaultRouteTuning() RouteTuning {
	return RouteTuning{MonasteryMinLevel: 16, MonasteryMinLevelMerc: 14, TowerMinLevel: 16, TowerMinLevelMerc: 14, TowerNeedsMerc: 0}
}

func routeKnobs() []Knob {
	return []Knob{
		{Key: "campaign.imbueMinLevel", Label: "Imbue quest level floor", Unit: "levels", Default: 20, Min: 8, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "Below this character level the optional Tools of the Trade quest (kill Smith in the Barracks, Charsi imbues an item) is skipped; Sorceress, Necromancer, Druid and Amazon never do it. G1 attempts 11 to 15 died on its walk three times at clvl 12."},
		{Key: "campaign.imbueNeedsMerc", Label: "Imbue quest needs a merc", Unit: "bool", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "1: the Imbue quest is also skipped while no mercenary is alive (an unknown merc counts as none). 0: the level floor alone decides."},
		{Key: "campaign.denMinLevelCaster", Label: "Den of Evil level floor (casters)", Unit: "levels", Default: 5, Min: 1, Max: 12, HigherIsBetter: true, NoHandicap: true,
			Desc: "Sorceress, Necromancer, Druid and a bow-less Amazon enter the Den of Evil (always the first quest) only from this clvl; below it they level in Blood Moor (log: den: waiting for clvl N in Blood Moor). SME: a ~40 life caster dies to Fallen packs on floor 1; 5 is the safe end of clvl 4-5."},
		{Key: "campaign.denMinLevelMelee", Label: "Den of Evil level floor (melee)", Unit: "levels", Default: 3, Min: 1, Max: 12, HigherIsBetter: true, NoHandicap: true,
			Desc: "Same for Barbarian, Paladin, Assassin and a bowed Amazon."},
		{Key: "route.monasteryMinLevel", Label: "Monastery road level floor (no merc)", Unit: "levels", Default: 16, Min: 10, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The Monastery road (Tamoe Highland, Monastery Gate, Cloisters, Barracks, Jail, Cathedral, Catacombs, so Andariel too) is entered without a living merc only from this clvl, and always with the supply kit (4 healing, 2 portals, mana for casters). Otherwise the bot levels in Cold Plains to Black Marsh and the caves. Interim (ADR 0011, grill 0003): 16, was 20; if road deaths fail the death rule at 16 in the act1run soak raise it to 18 and re-grill, if none try 14. 20 restores the old number."},
		{Key: "route.monasteryMinLevelMerc", Label: "Monastery road level floor (living merc)", Unit: "levels", Default: 14, Min: 0, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The same floor with a living merc (unknown counts as none). Interim (ADR 0011): 14. 0 restores the old rule: a living merc opens the road at any level."},
		{Key: "route.towerMinLevel", Label: "Forgotten Tower level floor (no merc)", Unit: "levels", Default: 16, Min: 0, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The Forgotten Tower and Tower Cellar 1 to 5 (areas 20-25, the Countess, a no-reward quest in Normal) are entered without a living merc only from this clvl, and with the supply kit (4 healing, a portal, mana for casters). Below it the quest is skipped as optional (log: route guard: blocked tower (no merc, clvl N < 16)), corpses there are abandoned and a character standing there walks out first. G1 attempt 17: a clvl 14 caster died twice in the cellars. Interim (ADR 0011): 16, was 20. 0 turns the tower guard off."},
		{Key: "route.towerMinLevelMerc", Label: "Forgotten Tower level floor (living merc)", Unit: "levels", Default: 14, Min: 0, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The Forgotten Tower floor with a living merc while route.towerNeedsMerc is 0. Interim (ADR 0011): 14. 0 means the no-merc floor applies to a merc too."},
		{Key: "route.towerNeedsMerc", Label: "Tower: a merc opens it at any level", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "0: a living mercenary only lowers the Forgotten Tower floor to route.towerMinLevelMerc (an unknown merc counts as none). 1: a living merc opens the tower at any level (the old rule)."},
	}
}
