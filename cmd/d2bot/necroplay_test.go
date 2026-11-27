package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
)

// Card necroepic: every plan table of a class resolves to a combat profile whose layout has a key for each skill the
// table uses as a main attack (a skill without a key is never cast: the corpse-explosion bug of card necroaudit). It runs over every class (card classkeys). The
// rule (a key, or a band that attacks with the skill) is data-driven, so it runs over the Necromancer tables and over a second class (Paladin) alike.
func TestMainAttackSkillsHaveAKey(t *testing.T) {
	files, _ := filepath.Glob(filepath.Join("..", "..", "internal", "plandata", "build-*.json"))
	perClass := map[string]int{}
	for _, f := range files {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var b struct {
			Class      string `json:"class"`
			Build      string `json:"build"`
			MainAttack []struct {
				Skill string `json:"skill"`
			} `json:"main_attack"`
		}
		if json.Unmarshal(raw, &b) != nil || b.Build == "" || len(b.MainAttack) == 0 {
			continue
		}
		c, ok := classByName(b.Class)
		if !ok {
			continue
		}
		cp := planForClass(c, b.Build, 30, nil)
		if cp.Err != nil {
			t.Errorf("%s: %v", b.Build, cp.Err)
			continue
		}
		pr, err := combat.LookupProfile(c, cp.CombatBuild)
		if err != nil {
			t.Errorf("%s: %v", b.Build, err)
			continue
		}
		perClass[strings.ToLower(b.Class)]++
		if c == d2.Necromancer && cp.CombatBuild == "" {
			t.Errorf("%s: a Necromancer table without a combat line fights with the class default layout", b.Build)
		}
		for _, m := range b.MainAttack {
			if m.Skill == "Attack" || m.Skill == "Throw" || m.Skill == "" { // the generic attacks (capabilities.json "generic") are the weapon's basic attack, no key
				continue
			}
			if !inLayoutOrBand(pr, m.Skill) {
				t.Errorf("%s: main attack %q has no key and no band in profile %s (layout %v)", b.Build, m.Skill, pr.ProfileID(), pr.Layout)
			}
		}
	}
	if perClass["necromancer"] < 3 || perClass["sorceress"] < 1 {
		t.Fatalf("tables seen %v", perClass)
	}
}

func inLayoutOrBand(p *combat.Profile, skill string) bool {
	if _, ok := p.Layout[skill]; ok {
		return true
	}
	for _, b := range p.Bands {
		for _, a := range b.Attack {
			if a == skill {
				return true
			}
		}
	}
	return false
}
