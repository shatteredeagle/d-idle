package plandata

import (
	"encoding/json"
	"os"
	"testing"
)

// Card classapply (gap list of the SME review 2026-11-27): every class that has a normal30 plan table has a row in the
// level plan and in the Act 2 plan, so neither errors by class name. The new rows are copies; their build tables exist.
func TestEveryPlayableClassHasPlanRows(t *testing.T) {
	for _, f := range []string{"levelplan-normal.json", "act2plan.json"} {
		raw, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var p struct {
			Classes map[string]struct {
				Build string `json:"build"`
				Tag   string `json:"tag"`
			} `json:"classes"`
		}
		if err := json.Unmarshal(raw, &p); err != nil {
			t.Fatal(err)
		}
		for _, c := range []string{"paladin", "sorceress", "necromancer", "barbarian", "druid", "amazon", "assassin"} {
			row, ok := p.Classes[c]
			if !ok {
				t.Errorf("%s has no %s row", f, c)
				continue
			}
			switch c {
			case "barbarian", "druid", "amazon", "assassin":
				if _, err := os.Stat("build-" + row.Build + ".json"); err != nil {
					t.Errorf("%s %s: build %q has no table: %v", f, c, row.Build, err)
				}
			}
		}
	}
}
