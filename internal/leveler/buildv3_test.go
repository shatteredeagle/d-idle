package leveler

import "testing"

// v3Req is reqlevel and required skills of the Sorceress skills a lightning table uses (Skills.txt, copied from
// tools/emu/sk_pass_plan.py, V by sk_pass_tbl.py on the patched 1.14d tables). A table test must not need the game data.
var v3Req = map[string]struct {
	lvl  int
	need []string
}{
	"Charged Bolt": {1, nil}, "Frozen Armor": {1, nil}, "Warmth": {1, nil}, "Static Field": {6, nil},
	"Telekinesis": {6, nil}, "Frost Nova": {6, nil}, "Lightning": {12, []string{"Charged Bolt"}},
	"Chain Lightning": {18, []string{"Lightning"}}, "Teleport": {18, []string{"Telekinesis"}},
	"Lightning Mastery": {30, nil}, "Energy Shield": {18, nil},
}

// replaySkills spends skill points clvl by clvl over a table with the real skills.txt rule (as replayBuild does) and
// quest points on top (clvl -> points), through Plan. refused lists every point the rule turned down: with a table
// whose rows are legal when their point arrives it stays empty (the leveler would otherwise spend the point on the
// next legal row, BUILDV3 / docs/research/sk-passives-trace.md B1).
func replaySkills(t *testing.T, build string, to int, quests map[int]int) (spent []string, at map[int]map[string]int, refused []string) {
	t.Helper()
	b, err := LoadBuild(build, "")
	if err != nil {
		t.Fatal(err)
	}
	cur := map[string]int{}
	at = map[int]map[string]int{}
	free := 0
	for lv := 2; lv <= to; lv++ {
		free += 1 + quests[lv]
		rule := func(name string, level int, lv2 map[string]int) bool {
			s, ok := v3Req[name]
			okRule := ok && s.lvl <= level && lv2[name] < 20
			for _, r := range s.need {
				if lv2[r] == 0 {
					okRule = false
				}
			}
			if !okRule {
				refused = append(refused, name)
			}
			return okRule
		}
		for _, a := range b.Plan(Have{Skills: cur, Level: lv, SkillPts: free}, rule, nil) {
			if a.Kind == KindSkill {
				cur[a.Skill]++
				free--
				spent = append(spent, a.Skill)
			}
		}
		snap := map[string]int{}
		for k, v := range cur {
			snap[k] = v
		}
		at[lv] = snap
	}
	return spent, at, refused
}

// BUILDV3: the table is spent exactly in table order (no point is refused and pushed to a later row) with and without
// the quest points, the approved milestones come at the approved levels with them, and about two levels later without.
func TestBuildV3SpentInTableOrder(t *testing.T) {
	b, err := LoadBuild(FallbackV3, "")
	if err != nil {
		t.Fatal(err)
	}
	var order []string
	for _, r := range b.Rows {
		for _, s := range r.Skills {
			for i := 0; i < s.Points; i++ {
				order = append(order, s.Skill)
			}
		}
	}
	for name, q := range map[string]map[int]int{
		"quests":    {7: 1, 20: 1, 33: 2}, // the SKPASS replay's schedule
		"no quests": nil,
	} {
		spent, at, refused := replaySkills(t, FallbackV3, 60, q)
		if len(refused) > 0 {
			t.Errorf("%s: rule refused %v", name, refused)
		}
		for i, s := range spent {
			if s != order[i] {
				t.Fatalf("%s: point %d went to %s, table row says %s", name, i+1, s, order[i])
			}
		}
		for _, bad := range []string{"Static Field", "Warmth", "Energy Shield"} {
			if at[60][bad] != 0 {
				t.Errorf("%s: v3 learned %s", name, bad)
			}
		}
		if at[60]["Frozen Armor"] != 1 {
			t.Errorf("%s: Frozen Armor %d, want 1", name, at[60]["Frozen Armor"])
		}
		if q == nil {
			continue
		}
		// approved rows, quest replay: skill -> clvl at which it first shows
		for _, m := range []struct {
			skill string
			lvl   int
			n     int
		}{
			{"Telekinesis", 11, 1}, {"Lightning", 12, 1}, {"Charged Bolt", 17, 13}, {"Teleport", 18, 1},
			{"Chain Lightning", 19, 1}, {"Chain Lightning", 29, 12}, {"Lightning Mastery", 30, 1},
			{"Lightning Mastery", 33, 5}, {"Chain Lightning", 40, 20},
		} {
			if at[m.lvl][m.skill] != m.n || at[m.lvl-1][m.skill] >= m.n {
				t.Errorf("%s: %s at clvl %d: %d (clvl-1: %d), want first reaching %d", name, m.skill, m.lvl, at[m.lvl][m.skill], at[m.lvl-1][m.skill], m.n)
			}
		}
	}
}

// The legacy and v2 tables are not touched by BUILDV3: v2 stays the default of a fresh Sorceress and Soaky's table
// (Static Field at clvl 6) is still selected for a character that owns Static Field.
func TestBuildV3IsOptIn(t *testing.T) {
	if got := SelectBuild("sorceress", 1, nil); got != SorceressNormal30 {
		t.Errorf("fresh sorceress default %q, want %q (sorcready2, user decision 2026-11-10)", got, SorceressNormal30)
	}
	if got := SelectBuild("sorceress", 12, map[string]int{"Charged Bolt": 9, "Static Field": 1}); got != "fallback" {
		t.Errorf("Static Field owner: %q", got)
	}
	legacy, err := LoadBuild("fallback", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range legacy.Rows {
		if r.Clvl == 6 && (len(r.Skills) != 1 || r.Skills[0].Skill != "Static Field") {
			t.Errorf("legacy clvl 6 row changed: %+v", r.Skills)
		}
	}
	for _, n := range []string{FallbackV2, FallbackV3} {
		if TableCombat(n) != "lightning" {
			t.Errorf("%s declares combat %q", n, TableCombat(n))
		}
	}
	if TableCombat("fallback") != "" || TableCombat("primary") != "" {
		t.Errorf("tables without a combat key must report none")
	}
}
