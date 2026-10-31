package env

// Tuning is the single place where efficiency-affecting parameters live (PLAN.md: optimal first, handicap by
// configuration). Defaults are the best the bot can do; a handicap profile lowers knobs, the progression layer
// raises them. No behavior package may hard-code a number that belongs here.
//
// Each domain owns its own file (tuning_nav.go, tuning_combat.go, tuning_town.go, tuning_loot.go, tuning_bot.go)
// so agents never edit the same file.
type Tuning struct {
	Nav      NavTuning
	Combat   CombatTuning
	Town     TownTuning
	Loot     LootTuning
	Bot      BotTuning
	Boss     BossTuning
	Merc     MercTuning
	Campaign CampaignTuning
	Route    RouteTuning
	// Resource is the mana/health policy (tuning_resource.go); its knobs are registered by cmd/d2bot.
	Resource ResourceTuning
}

// DefaultTuning returns the optimal profile.
func DefaultTuning() Tuning {
	return Tuning{
		Nav:      DefaultNavTuning(),
		Combat:   DefaultCombatTuning(),
		Town:     DefaultTownTuning(),
		Loot:     DefaultLootTuning(),
		Bot:      DefaultBotTuning(),
		Boss:     DefaultBossTuning(),
		Merc:     DefaultMercTuning(),
		Campaign: DefaultCampaignTuning(),
		Route:    DefaultRouteTuning(),
		Resource: DefaultResourceTuning(),
	}
}

// Knob is the metadata the UI uses to render and validate a tuning parameter without hard-coding it.
type Knob struct {
	Key            string  `json:"key"`   // dotted path, e.g. "combat.chickenLifePct"
	Label          string  `json:"label"` // human label
	Unit           string  `json:"unit"`  // "%", "ms", "subtiles", "count", ...
	Desc           string  `json:"desc"`
	Default        float64 `json:"default"`
	Min            float64 `json:"min"` // range in which the bot still survives and works
	Max            float64 `json:"max"`
	HigherIsBetter bool    `json:"higherIsBetter"`
	NoHandicap     bool    `json:"noHandicap,omitempty"` // a switch or policy, not a skill: handicap profiles leave it at its default
}

// RetiredKnobs are keys of knobs that no longer exist. Old config.json "tuning" objects and saved tuner state may
// still name them; they load and are ignored.
var RetiredKnobs = map[string]bool{"bot.starveGold": true, "bot.starveRecoverGold": true}

// Knobs lists every tuning knob across domains.
func Knobs() []Knob {
	var k []Knob
	k = append(k, navKnobs()...)
	k = append(k, combatKnobs()...)
	k = append(k, townKnobs()...)
	k = append(k, lootKnobs()...)
	k = append(k, botKnobs()...)
	k = append(k, bossKnobs()...)
	k = append(k, mercKnobs()...)
	k = append(k, routeKnobs()...)
	k = append(k, registeredKnobs()...)
	return k
}
