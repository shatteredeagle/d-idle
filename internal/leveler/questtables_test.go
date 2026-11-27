package leveler

import (
	"encoding/json"
	"strings"
	"testing"

	"d2lod-bot/internal/plandata"
)

// skillQuests returns the quest-points.json ids whose reward is a skill point, with the points one completion grants
// in one difficulty (Izual: 2).
func skillQuests(t *testing.T) map[string]int {
	t.Helper()
	raw, err := plandata.FS.ReadFile("quest-points.json")
	if err != nil {
		t.Fatal(err)
	}
	var qp struct {
		Quests []struct {
			ID     string `json:"id"`
			Reward string `json:"reward"`
		} `json:"quests"`
	}
	if err := json.Unmarshal(raw, &qp); err != nil {
		t.Fatal(err)
	}
	out := map[string]int{}
	for _, q := range qp.Quests {
		switch {
		case strings.Contains(q.Reward, "+2 skill points"):
			out[q.ID] = 2
		case strings.Contains(q.Reward, "skill point"):
			out[q.ID] = 1
		}
	}
	if out["den_of_evil"] != 1 || out["radament"] != 1 || out["izual"] != 2 {
		t.Fatalf("quest-points.json skill quests changed: %v", out)
	}
	return out
}

// Every build table counts level points only: up to clvl N its untagged skill points never exceed the N-1 the levels
// grant, and its stat points never exceed 5 per level (Lam Esen's Tome is stats and has no tag). A quest bonus point
// lives in a skill entry with a quest key naming a skill-point quest, and no quest is tagged more often than its three
// difficulties grant (a table may tag fewer: a spare quest point pulls the next rows forward).
func TestBuildTablesTagQuestPoints(t *testing.T) {
	quests := skillQuests(t)
	ents, err := plandata.FS.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}
	n := 0
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, "build-") || !strings.HasSuffix(name, ".json") || name == plandata.CatalogFile {
			continue
		}
		raw, _ := plandata.FS.ReadFile(name)
		b, err := ParseBuild(raw)
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		n++
		skills, stats := 0, 0
		tagged := map[string]int{}
		for _, r := range b.Rows {
			for _, a := range r.Attr {
				stats += a
			}
			for _, sp := range r.Skills {
				if sp.Quest == "" {
					skills += sp.Points
					continue
				}
				if quests[sp.Quest] == 0 {
					t.Errorf("%s clvl %d: %s tagged %q, not a skill-point quest", name, r.Clvl, sp.Skill, sp.Quest)
				}
				tagged[sp.Quest] += sp.Points
			}
			if skills > r.Clvl-1 {
				t.Errorf("%s clvl %d: %d untagged skill points by this row, the levels grant %d (tag the quest point)", name, r.Clvl, skills, r.Clvl-1)
			}
			if stats > 5*(r.Clvl-1) {
				t.Errorf("%s clvl %d: %d stat points by this row, the levels grant %d", name, r.Clvl, stats, 5*(r.Clvl-1))
			}
		}
		for q, pts := range tagged {
			if pts > 3*quests[q] {
				t.Errorf("%s: %d points tagged %s, the three difficulties grant %d", name, pts, q, 3*quests[q])
			}
		}
	}
	if n < 20 {
		t.Fatalf("only %d build tables checked", n)
	}
}
