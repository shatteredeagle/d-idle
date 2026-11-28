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

// skillNameExtra are skills the damage table does not list, by name (S: the build tables of that class learn them,
// internal/plandata; Teleport V as skillIDExtra).
var skillNameExtra = map[string]string{"Teleport": "sor", "Clay Golem": "nec", "Raise Skeleton": "nec", "Oak Sage": "dru"}

// skillNameProblems lists the pre.bind and pre.summon skills of scs that are no skill NAME of the scenario's class, and
// counts the rows it looked at.
func skillNameProblems(t *testing.T, scs []Scenario) (bad []string, rows int) {
	t.Helper()
	b, err := os.ReadFile("../power/skilldamage.json")
	if err != nil {
		t.Fatal(err)
	}
	var tab struct {
		Skills map[string]struct {
			Class string `json:"class"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(b, &tab); err != nil {
		t.Fatal(err)
	}
	for _, s := range scs {
		var names []string
		for _, x := range s.Pre.Bind {
			names = append(names, x.Skill)
		}
		for _, x := range s.Pre.Summon {
			names = append(names, x.Skill)
		}
		rows += len(names)
		for _, n := range names {
			want, ok := skillNameExtra[n]
			if r, in := tab.Skills[n]; in {
				want, ok = r.Class, true
			}
			if !ok {
				bad = append(bad, s.ID+": skill "+n+" is in no skills table")
				continue
			}
			for _, c := range s.Matrix.Classes {
				if classCode[strings.ToLower(c)] != want {
					bad = append(bad, s.ID+": "+n+" is a "+want+" skill, matrix class "+c)
				}
			}
		}
	}
	return bad, rows
}

// Every pre.bind and pre.summon skill of a shipped scenario is a NAME of a skill of the scenario's class. The runner
// resolves the id from the game's class table; this is the offline guard against a typo or another class's skill.
func TestPreSkillNamesExistForTheClass(t *testing.T) {
	scs, err := LoadDir("../../tools/provingground/scenarios")
	if err != nil {
		t.Fatal(err)
	}
	bad, rows := skillNameProblems(t, scs)
	for _, b := range bad {
		t.Error(b)
	}
	if rows == 0 {
		t.Fatal("no scenario has pre.bind or pre.summon: the test checks nothing")
	}
}

// The must-fail arm on two classes: another class's skill and a name no table has are both reported.
func TestPreSkillNameGuardMustFail(t *testing.T) {
	for _, class := range []string{"Paladin", "Sorceress"} {
		s := Scenario{ID: "x.names", Matrix: Matrix{Classes: []string{class}},
			Pre: Pre{Bind: []HandBind{{Skill: "Whirlwind"}, {Skill: "Whirlwnd", Hand: "left"}}, Summon: []Summon{{Skill: "Clay Golem"}}}}
		if bad, rows := skillNameProblems(t, []Scenario{s}); len(bad) != 3 || rows != 3 {
			t.Errorf("%s: %d rows, problems %v", class, rows, bad)
		}
	}
	ok := Scenario{ID: "x.ok", Matrix: Matrix{Classes: []string{"Necromancer"}}, Pre: Pre{Bind: []HandBind{{Skill: "Corpse Explosion"}}, Summon: []Summon{{Skill: "Clay Golem"}}}}
	if bad, _ := skillNameProblems(t, []Scenario{ok}); len(bad) != 0 {
		t.Errorf("a Necromancer with its own skills: %v", bad)
	}
}
