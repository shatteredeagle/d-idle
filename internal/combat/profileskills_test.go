package combat

import (
	"encoding/json"
	"os"
	"testing"
)

// Card classkeys: every skill a profile names (layout, bands, buffs, auras, minions, avoid) is a row of the skills table
// data/capabilities/capabilities.json and belongs to the profile's class, so a typo or a skill of another class fails here
// instead of being silently never cast. Skills are named, never numbered (AGENTS.md build flexibility).
func TestProfileSkillsExistForTheirClass(t *testing.T) {
	raw, err := os.ReadFile("../../data/capabilities/capabilities.json")
	if err != nil {
		t.Fatal(err)
	}
	var caps struct {
		Skills  map[string]struct{ Class string }
		Generic map[string]json.RawMessage
	}
	if err := json.Unmarshal(raw, &caps); err != nil || len(caps.Skills) == 0 {
		t.Fatalf("capabilities.json: %v (%d skills)", err, len(caps.Skills))
	}
	seen := map[string]int{}
	for _, p := range Profiles() {
		names := map[string]bool{}
		for n := range p.Layout {
			names[n] = true
		}
		for _, b := range p.Bands {
			names[b.Requires] = true
			for _, n := range b.Attack {
				names[n] = true
			}
			for _, a := range b.Aura {
				names[a.Name] = true
			}
		}
		for _, a := range p.Aura {
			names[a.Name] = true
		}
		for _, g := range p.Buffs {
			for _, n := range g {
				names[n] = true
			}
		}
		for _, n := range p.Avoid {
			names[n] = true
		}
		if p.Minions != nil {
			for _, g := range p.Minions.Groups {
				for _, n := range g.Skills {
					names[n] = true
				}
			}
		}
		for n := range names {
			if n == "" {
				continue
			}
			if _, ok := caps.Generic[n]; ok {
				continue
			}
			s, ok := caps.Skills[n]
			if !ok {
				t.Errorf("profile %s names skill %q, which is not in capabilities.json", p.ProfileID(), n)
				continue
			}
			if s.Class != p.Class {
				t.Errorf("profile %s names skill %q of class %s", p.ProfileID(), n, s.Class)
			}
			seen[p.Class]++
		}
	}
	for _, c := range []string{"sorceress", "paladin", "amazon", "necromancer", "barbarian", "druid"} {
		if seen[c] == 0 {
			t.Errorf("no skill checked for class %s", c)
		}
	}
}

// The four profiles of card classkeys lead with the table's main attack at the level the table names it (two classes
// with a Throw/Bash start and one with a ground skill, so the rule is not one build's).
func TestClassKeysProfilesLeadWithTheTablesAttack(t *testing.T) {
	all := func(string) bool { return true }
	for _, c := range []struct {
		class    string
		build    string
		clvl     int
		wantLead string
	}{
		{"druid", "tornado", 30, "Tornado"},
		{"druid", "tornado", 20, "Twister"},
		{"barbarian", "warcry", 40, "War Cry"},
		{"barbarian", "warcry", 10, "Bash"},
		{"amazon", "javelin", 20, "Lightning Bolt"},
		{"paladin", "hammerdin_zeal", 14, "Zeal"},
	} {
		var got *Profile
		for i, p := range Profiles() {
			if p.Class == c.class && p.Build == c.build {
				got = &Profiles()[i]
			}
		}
		if got == nil {
			t.Errorf("no profile %s/%s", c.class, c.build)
			continue
		}
		if n := got.AttackNames(c.clvl, all); len(n) == 0 || n[0] != c.wantLead {
			t.Errorf("%s/%s at clvl %d leads with %v, want %s", c.class, c.build, c.clvl, n, c.wantLead)
		}
	}
}
