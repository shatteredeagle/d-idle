package scenario

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// classCode maps a matrix class name to the charclass column of skills.txt.
var classCode = map[string]string{"amazon": "ama", "sorceress": "sor", "necromancer": "nec", "paladin": "pal",
	"barbarian": "bar", "druid": "dru", "assassin": "ass"}

// skillIDExtra are skills the damage table does not list (no damage row). V: docs/coverage/ledger.tsv row skills.sor 54
// (SKTELE trace: "the Sorceress skill 54").
var skillIDExtra = map[int]string{54: "sor"}

// Every pre.skills id of a scenario names a skill of the scenario's class (V: skills.txt id and charclass columns as
// read into internal/power/skilldamage.json). A hand-typed id that is wrong would put points into another skill or fail
// the `debug points` verb only in the lab.
func TestPreSkillIDsExistForTheClass(t *testing.T) {
	b, err := os.ReadFile("../power/skilldamage.json")
	if err != nil {
		t.Fatal(err)
	}
	var tab struct {
		Skills map[string]struct {
			ID    int    `json:"id"`
			Class string `json:"class"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(b, &tab); err != nil {
		t.Fatal(err)
	}
	byID := map[int]string{}
	for _, s := range tab.Skills {
		byID[s.ID] = s.Class
	}
	for id, c := range skillIDExtra {
		byID[id] = c
	}
	scs, err := LoadDir("../../tools/provingground/scenarios")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, s := range scs {
		for _, sk := range s.Pre.Skills {
			n++
			want, ok := byID[sk.Skill]
			if !ok {
				t.Errorf("%s: pre.skills id %d is in no skills table", s.ID, sk.Skill)
				continue
			}
			for _, c := range s.Matrix.Classes {
				if classCode[strings.ToLower(c)] != want {
					t.Errorf("%s: skill %d is a %q skill, matrix class %s", s.ID, sk.Skill, want, c)
				}
			}
		}
	}
	if n == 0 {
		t.Fatal("no scenario has pre.skills: the test checks nothing")
	}
}
