package env

// SensesTuning is the side-panel minimap switch (docs/senses-map.md, TASK-148). View only: nothing here changes what the
// bot does.
type SensesTuning struct {
	Tier int // highest senses tier the watch screen shows (watch.FixedTier); 0 = panel off, 1 = fogged area map and hero
}

func DefaultSensesTuning() SensesTuning { return SensesTuning{Tier: 1} }

func sensesKnobs() []Knob {
	return []Knob{
		{Key: "senses.tier", Label: "Watch screen senses tier", Unit: "tier", Default: 1, Min: 0, Max: 1, NoHandicap: true,
			Desc: "Highest senses tier the watch screen shows beside the video (needs --watch). 1: the fogged map of the area and the hero (docs/senses-map.md). 0: no panel and no /api/senses. The panel only reads what the bot already knows; bot inputs are identical either way. TASK-148."},
	}
}
