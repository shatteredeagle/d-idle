package agency

import "testing"

func TestDriveOf(t *testing.T) {
	sup := func(n string) *Choice { return &Choice{Skill: n, Tier: Supported} }
	cases := []struct {
		name      string
		pl        Plan
		atk, aura string
	}{
		{"paladin: planner beats the profile", Plan{Archetype: "hammerdin", Prior: "Zeal", Attack: sup("Blessed Hammer"), PriorAura: "Might", Aura: sup("Concentration")}, "Blessed Hammer", "Concentration"},
		{"sorceress: agrees", Plan{Archetype: "caster", Prior: "Fire Ball", Attack: sup("Fire Ball")}, "", ""},
		{"unsupported is never driven", Plan{Archetype: "x", Prior: "Zeal", Attack: &Choice{Skill: "Fist of the Heavens", Tier: Unsupported}}, "", ""},
		{"no archetype keeps the profile", Plan{Prior: "Zeal", Attack: sup("Blessed Hammer")}, "", ""},
		{"no profile pick keeps the profile", Plan{Archetype: "x", Attack: sup("Fire Ball")}, "", ""},
	}
	for _, c := range cases {
		d := DriveOf(c.pl)
		if d.Attack != c.atk || d.Aura != c.aura || d.Why == "" {
			t.Errorf("%s: %+v", c.name, d)
		}
	}
}
