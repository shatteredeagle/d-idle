package leveler

import "testing"

// The allocator asks the same planner the leveler uses: one point at a time, in table order, and the answer moves on
// when the point is owned (so a restart or a manual spend never double-spends).
func TestAutoAllocPlanFollowsRowsTwoClasses(t *testing.T) {
	for _, name := range []string{PaladinV2, SorceressNormal30} {
		b := mustBuild(t, name)
		var order []string
		for _, r := range b.Rows {
			for _, s := range r.Skills {
				if s.Quest != "" {
					continue // a quest-tagged point waits for its quest point (Have.QuestPts)
				}
				for i := 0; i < s.Points; i++ {
					order = append(order, s.Skill)
				}
			}
			if len(order) >= 4 {
				break
			}
		}
		owned := map[string]int{}
		for i, want := range order {
			h := Have{Level: 30, SkillPts: 1, Skills: owned}
			var got []Action
			for _, a := range b.Plan(h, DBRule(nil, ""), nil) {
				if a.Kind == KindSkill {
					got = append(got, a)
				}
			}
			if len(got) == 0 || got[0].Skill != want {
				t.Fatalf("%s point %d: plan %v, want %s", name, i, got, want)
			}
			owned[want]++
		}
	}
}

func TestOffBuildAndKeys(t *testing.T) {
	b := mustBuild(t, PaladinV2)
	off := b.OffBuild(map[string]int{"Zeal": 1, "Definitely Not A Skill": 2, "Might": 0})
	if len(off) != 1 || off[0] != "Definitely Not A Skill" {
		t.Errorf("off-build %v", off)
	}
	if StatKey(Str) != "str" || StatKey(Ene) != "ene" || StatKey(Vit) != "vit" || StatKey(Dex) != "dex" {
		t.Error("stat keys")
	}
	if names := TableNames(); len(names) < 5 {
		t.Errorf("table names %v", names)
	}
}
