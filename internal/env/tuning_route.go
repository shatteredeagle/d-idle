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
	TowerMinLife          int // max life the tower needs without a living merc; 0 = no life rule (then TowerMinLevel alone)
	TowerMinLevelNoLife   int // no-merc clvl floor while max life is unreadable (the pre-life rule)
	TowerKitMana          int // mana potions the tower kit wants for a class with a mana column in the belt plan
	FreshGapSec           int // min seconds between two fresh-game restarts while the last one gained under FreshMinXP; 0 = no gap
	FreshMaxNoGain        int // fresh-game restarts in a row without FreshMinXP gained before the blocked state; 0 = bound off
	FreshMinXP            int // experience a cycle must gain to count as progress and reset the no-gain count
	FreshBlockedSec       int // seconds between restart probes once FreshMaxNoGain was reached (the blocked state waits in town)
}

func DefaultCampaignTuning() CampaignTuning {
	return CampaignTuning{ImbueMinLevel: 20, ImbueNeedsMerc: 1, DenMinLevelCaster: 5, DenMinLevelMelee: 3}
}
func DefaultRouteTuning() RouteTuning {
	return RouteTuning{MonasteryMinLevel: 14, MonasteryMinLevelMerc: 14, TowerMinLevel: 15, TowerMinLevelMerc: 14, TowerNeedsMerc: 0,
		TowerMinLife: 150, TowerMinLevelNoLife: 16, TowerKitMana: 3,
		FreshGapSec: 600, FreshMaxNoGain: 3, FreshMinXP: 200, FreshBlockedSec: 1800}
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
		{Key: "route.monasteryMinLevel", Label: "Monastery road level floor (no merc)", Unit: "levels", Default: 14, Min: 10, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The Monastery road (Tamoe Highland, Monastery Gate, Cloisters, Barracks, Jail, Cathedral, Catacombs, so Andariel too) is entered without a living merc only from this clvl, and always with the supply kit (4 healing, 2 portals, mana for casters). Otherwise the bot levels in Cold Plains to Black Marsh and the caves. Interim (ADR 0011, grill 0003; merc decision 2026-11-02): 14 with a full kit, was 16 and 20, so a dead or missing merc does not block the road (the living-merc floors rate a 140-HP Rogue too high). If road deaths fail the death rule at 14 in the act1run soak raise it to 16 or 18 and re-grill. 20 restores the old number."},
		{Key: "route.monasteryMinLevelMerc", Label: "Monastery road level floor (living merc)", Unit: "levels", Default: 14, Min: 0, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The same floor with a living merc (unknown counts as none). Interim (ADR 0011): 14. 0 restores the old rule: a living merc opens the road at any level."},
		{Key: "route.towerMinLevel", Label: "Forgotten Tower level floor (no merc)", Unit: "levels", Default: 15, Min: 0, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The Forgotten Tower and Tower Cellar 1 to 5 (areas 20-25, the Countess, a no-reward quest in Normal) are entered without a living merc only from this clvl AND with route.towerMinLife max life, and with the supply kit (4 healing, a portal with charges, route.towerKitMana mana potions for a class that uses mana). Below it the quest is skipped as optional (log: route guard: blocked tower (no merc, clvl N < 15) or (no merc, life N < 150)), corpses there are abandoned and a character standing there walks out first. G1 attempt 17: a clvl 14 caster died twice in the cellars. SME 2026-11-04 (user-approved, numbers U): 15 plus life 150, was 16 (ADR 0011 interim) and 20. 0 turns the tower guard off."},
		{Key: "route.towerMinLevelMerc", Label: "Forgotten Tower level floor (living merc)", Unit: "levels", Default: 14, Min: 0, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The Forgotten Tower floor with a living merc while route.towerNeedsMerc is 0. Interim (ADR 0011): 14. 0 means the no-merc floor applies to a merc too."},
		{Key: "route.towerMinLife", Label: "Forgotten Tower max life floor (no merc)", Unit: "life", Default: 150, Min: 0, Max: 1000, HigherIsBetter: true, NoHandicap: true,
			Desc: "Without a living merc the Forgotten Tower also needs this max life (log: route guard: blocked tower (no merc, life N < 150)). U: SME number, tuned by the death loop. 0 turns the life rule off. While max life cannot be read the fallback floor route.towerMinLevelNoLife applies and the log says so."},
		{Key: "route.towerMinLevelNoLife", Label: "Forgotten Tower level floor (no merc, life unreadable)", Unit: "levels", Default: 16, Min: 0, Max: 40, HigherIsBetter: true, NoHandicap: true,
			Desc: "The no-merc clvl floor while max life is not readable (the pre-life rule, ADR 0011 interim). Log: route guard: blocked tower (no merc, clvl N < 16 (max life unreadable, fallback floor))."},
		{Key: "route.towerKitMana", Label: "Forgotten Tower kit: mana potions", Unit: "potions", Default: 3, Min: 0, Max: 12, HigherIsBetter: true, NoHandicap: true,
			Desc: "Mana potions the Forgotten Tower kit wants for a class that uses mana every attack (Sorceress, Necromancer, Druid; resource.ManaNeed 2). A melee build such as the Barbarian is not asked. U: SME number."},
		{Key: "route.towerNeedsMerc", Label: "Tower: a merc opens it at any level", Unit: "bool", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "0: a living mercenary only lowers the Forgotten Tower floor to route.towerMinLevelMerc (an unknown merc counts as none). 1: a living merc opens the tower at any level (the old rule)."},
		{Key: "route.freshGapSec", Label: "Fresh game: min gap after a no-gain restart", Unit: "s", Default: 600, Min: 0, Max: 7200, HigherIsBetter: true, NoHandicap: true,
			Desc: "When every leveling area is cleared or blocked the bot starts a fresh game. If the previous fresh game gained under route.freshMinXP the next restart waits until this many seconds have passed (in town, log: leveling: fresh game held back). Gate 2 rerun seed 13 (clvl 15) restarted 75 times in 4 h with no XP. 0 = no gap."},
		{Key: "route.freshMaxNoGain", Label: "Fresh game: no-gain restarts before blocked", Unit: "restarts", Default: 3, Min: 0, Max: 50, HigherIsBetter: true, NoHandicap: true,
			Desc: "After this many fresh games in a row that each gained under route.freshMinXP, the bot logs 'leveling: blocked: needs input' and only probes one more restart every route.freshBlockedSec, waiting in town in between. A cycle with enough XP resets the count. 0 turns the bound off (the old endless restart loop)."},
		{Key: "route.freshMinXP", Label: "Fresh game: XP that counts as progress", Unit: "xp", Default: 200, Min: 0, Max: 100000, HigherIsBetter: true, NoHandicap: true,
			Desc: "Experience gained between two fresh-game restarts that resets the no-gain count (log field xpGained). U: a starting number, tune from the Gate logs."},
		{Key: "route.freshBlockedSec", Label: "Fresh game: probe interval when blocked", Unit: "s", Default: 1800, Min: 0, Max: 14400, HigherIsBetter: true, NoHandicap: true,
			Desc: "Once the no-gain bound is reached the bot waits this long in town between restart probes, so the loop never spins; bans expire and the picker is asked again after each wait."},
	}
}
