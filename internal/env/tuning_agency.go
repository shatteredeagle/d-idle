package env

// AgencyTuning holds the knobs of the stat and skill agency work (docs/agency.md, ADR 0020). Owner: the agencyp1a card.
type AgencyTuning struct {
	// Planner is the skill planner: 0 off, 1 shadow, 2 drive (logs what it would pick next to what the combat profile picks, serves
	// GET /api/hero/plan; the hero plays exactly as before). Mode 2 hands the fighter the planner's attack and aura (agency.DriveOf).
	Planner int
	// AutoAlloc is the automatic skill allocator: 0 off (default), 1 on. A hero whose build is locked (POST /api/hero/build)
	// then gets each new stat and skill point from that build's rows. docs/autoalloc.md.
	AutoAlloc int
}

func DefaultAgencyTuning() AgencyTuning { return AgencyTuning{Planner: 0} }

func agencyKnobs() []Knob {
	return []Knob{
		{Key: "agency.autoAlloc", Label: "Automatic Allocate Skills", Unit: "0/1", Default: 0, Min: 0, Max: 1, NoHandicap: true,
			Desc: "0 (default): off, points are banked for the player. 1: a hero whose build is locked (chosen on the Points page) puts every new stat and skill point where that build's table says, at a calm moment, by verb. A skill the hero cannot use yet is skipped with a visible reason and the point stays banked (docs/autoalloc.md)."},
		{Key: "agency.planner", Label: "Skill planner", Unit: "mode", Default: 0, Min: 0, Max: 2, NoHandicap: true,
			Desc: "0 (default): off. 1: shadow. Once a minute the hero logs a `planner shadow:` line with the attack and aura the planner would pick from the skills the player actually bought, next to what the combat profile picks, and GET /api/hero/plan and the Points page show the derived rotation, the warnings and the viability line. Modes 0 and 1 never change what the hero does (docs/agency.md). 2: drive, gated by the owner go: the fighter plays the planner's attack (and aura) instead of the combat profile's first pick, only when the plan is supported, differs from the profile's pick and the archetype resolved (agency.DriveOf); a `planner drive:` line says why when it changes. [dark-launch]"},
	}
}
