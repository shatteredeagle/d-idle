package goals

// Catalog is static reference data for the goal form. The bot's data layer (internal/data) will be the
// source of truth for area ids; here ids are only included where the value is well known, 0 = let the
// bot resolve the name. UNVERIFIED against the user's game tables.
type CatalogArea struct {
	Name string `json:"name"`
	Act  int    `json:"act"`
	ID   int    `json:"id,omitempty"`
}

type CatalogBoss struct {
	Name string `json:"name"`
	Act  int    `json:"act"`
	Area string `json:"area"`
}

type CatalogRun struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	Boss  string `json:"boss,omitempty"`
}

type CatalogData struct {
	Areas      []CatalogArea `json:"areas"`
	Bosses     []CatalogBoss `json:"bosses"`
	Runs       []CatalogRun  `json:"runs"`
	ItemTypes  []string      `json:"itemTypes"`
	Qualities  []string      `json:"qualities"`
	Difficulty []string      `json:"difficulties"`
}

func Catalog() CatalogData {
	return CatalogData{
		Areas: []CatalogArea{
			{"Blood Moor", 1, 2}, {"Cold Plains", 1, 3}, {"Stony Field", 1, 4}, {"Dark Wood", 1, 5},
			{"Black Marsh", 1, 6}, {"Den of Evil", 1, 8}, {"Burial Grounds", 1, 17}, {"Crypt", 1, 18},
			{"Mausoleum", 1, 19}, {"Cave Level 2", 1, 13}, {"Pit Level 1", 1, 12}, {"Tower Cellar Level 5", 1, 0},
			{"Catacombs Level 4", 1, 37}, {"Halls of the Dead", 2, 0}, {"Maggot Lair", 2, 0},
			{"Ancient Tunnels", 2, 0}, {"Arcane Sanctuary", 2, 0}, {"Sewers", 2, 0},
			{"Spider Forest", 3, 0}, {"Flayer Jungle", 3, 0}, {"Travincal", 3, 0}, {"Durance of Hate Level 3", 3, 102},
			{"Outer Steppes", 4, 0}, {"City of the Damned", 4, 0}, {"River of Flame", 4, 107},
			{"Chaos Sanctuary", 4, 108}, {"Bloody Foothills", 5, 0}, {"Frigid Highlands", 5, 0},
			{"Arreat Plateau", 5, 0}, {"Crystalline Passage", 5, 0}, {"Glacial Trail", 5, 0},
			{"Nihlathak's Temple", 5, 0}, {"Throne of Destruction", 5, 131}, {"Worldstone Chamber", 5, 132},
		},
		Bosses: []CatalogBoss{
			{"Blood Raven", 1, "Burial Grounds"}, {"The Countess", 1, "Tower Cellar Level 5"},
			{"The Smith", 1, "Barracks"}, {"Andariel", 1, "Catacombs Level 4"},
			{"Radament", 2, "Sewers"}, {"The Summoner", 2, "Arcane Sanctuary"}, {"Duriel", 2, "Tal Rasha's Chamber"},
			{"Mephisto", 3, "Durance of Hate Level 3"}, {"Izual", 4, "City of the Damned"}, {"Diablo", 4, "Chaos Sanctuary"},
			{"Pindleskin", 5, "Nihlathak's Temple"}, {"Nihlathak", 5, "Halls of Vaught"}, {"Eldritch", 5, "Frigid Highlands"},
			{"Shenk", 5, "Bloody Foothills"}, {"Baal", 5, "Worldstone Chamber"},
		},
		Runs: []CatalogRun{
			{"run:mephisto", "Mephisto run", "Mephisto"}, {"run:andariel", "Andariel run", "Andariel"},
			{"run:pindleskin", "Pindleskin run", "Pindleskin"}, {"run:countess", "Countess run", "The Countess"},
			{"run:baal", "Baal run", "Baal"}, {"run:diablo", "Diablo run", "Diablo"},
			{"run:act5-bosses", "Eldritch + Shenk + Pindleskin", ""}, {"run:cows", "Secret Cow Level", ""},
		},
		ItemTypes:  []string{"rune", "gem", "charm", "ring", "amulet", "helm", "armor", "weapon", "shield", "boots", "gloves", "belt", "jewel", "potion", "key", "gold"},
		Qualities:  append([]string{"any"}, Qualities...),
		Difficulty: []string{"normal", "nightmare", "hell"},
	}
}
