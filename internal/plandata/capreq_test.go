package plandata_test

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	capdata "d2lod-bot/data/capabilities"
)

// Card cls2 (review 8 section 3.1, last check): the required level of a capability row equals the skills.txt value.
// The build box has no tables cache, so the reference is the skills.txt extract the repo does carry, the req_level
// column of internal/power/skilldamage.json (131 damage skills; passives and buffs are not in it and stay S). A row
// of a class in mustHave without a level fails, so a new Amazon or Assassin row cannot ship empty again.
func TestCapabilityReqLevelMatchesSkillExtract(t *testing.T) {
	raw, err := os.ReadFile("../power/skilldamage.json")
	if err != nil {
		t.Fatal(err)
	}
	var dmg struct {
		Skills map[string]struct {
			ReqLevel *int `json:"req_level"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(raw, &dmg); err != nil {
		t.Fatal(err)
	}
	var caps struct {
		Skills map[string]struct {
			Class    string `json:"class"`
			ReqLevel *int   `json:"reqLevel"`
		} `json:"skills"`
	}
	if err := json.Unmarshal(capdata.JSON, &caps); err != nil {
		t.Fatal(err)
	}
	mustHave := map[string]bool{"amazon": true, "assassin": true}
	names := make([]string, 0, len(caps.Skills))
	for n := range caps.Skills {
		names = append(names, n)
	}
	sort.Strings(names)
	compared := map[string]int{}
	for _, n := range names {
		row := caps.Skills[n]
		if mustHave[row.Class] && row.ReqLevel == nil {
			t.Errorf("%s (%s): no reqLevel", n, row.Class)
		}
		ref, ok := dmg.Skills[n]
		if !ok || ref.ReqLevel == nil {
			continue
		}
		compared[row.Class]++
		if row.ReqLevel == nil || *row.ReqLevel != *ref.ReqLevel {
			t.Errorf("%s (%s): reqLevel %v, the skills.txt extract says %d", n, row.Class, deref(row.ReqLevel), *ref.ReqLevel)
		}
	}
	for c := range mustHave {
		if compared[c] < 10 {
			t.Errorf("%s: only %d rows compared, the extract or the names moved", c, compared[c])
		}
	}
}

func deref(p *int) any {
	if p == nil {
		return "none"
	}
	return *p
}
