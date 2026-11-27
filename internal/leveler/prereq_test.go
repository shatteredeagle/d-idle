package leveler

import (
	"reflect"
	"testing"
)

// fakeReq mirrors the skills.txt reqskill columns the tests need (S: 1.14d skills.txt, Zeal needs Sacrifice, Bone
// Spear needs Corpse Explosion, Corpse Explosion needs Teeth), with the matching rule (required level and prerequisites).
var fakeReq = map[string][]string{"Zeal": {"Sacrifice"}, "Bone Spear": {"Corpse Explosion"}, "Corpse Explosion": {"Teeth"}}
var fakeReqLvl = map[string]int{"Zeal": 12, "Bone Spear": 18, "Corpse Explosion": 6}

func fakeRule(name string, level int, lv map[string]int) bool {
	if fakeReqLvl[name] > level {
		return false
	}
	for _, rq := range fakeReq[name] {
		if lv[rq] == 0 {
			return false
		}
	}
	return true
}

func planned(acts []Action) (out []string) {
	for _, a := range acts {
		if a.Kind == KindSkill {
			s := a.Skill
			if a.Prereq {
				s += "*"
			}
			out = append(out, s)
		}
	}
	return out
}

// prereq (live 2026-11-27, Doofer clvl 14): the Paladin never learned Sacrifice, so the Zeal rows were refused and the
// points went to later Might rows. A missing prerequisite is bought first (marked Prereq), the planned skill next.
func TestPlanBuysMissingPrerequisiteFirst(t *testing.T) {
	pal, err := ParseBuild([]byte(`{"build":"pal_p","class":"paladin","rows":[
		{"clvl":2,"skills":[{"skill":"Might","points":1}]},
		{"clvl":12,"skills":[{"skill":"Zeal","points":1}]},
		{"clvl":13,"skills":[{"skill":"Zeal","points":1}]},
		{"clvl":14,"skills":[{"skill":"Might","points":1}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	h := Have{Level: 12, SkillPts: 1, Skills: map[string]int{"Might": 1}, Req: func(n string) []string { return fakeReq[n] }}
	acts, why := pal.PlanWhy(h, fakeRule, nil)
	if got := planned(acts); !reflect.DeepEqual(got, []string{"Sacrifice*"}) {
		t.Fatalf("clvl 12: %v", got)
	}
	if len(why) != 0 {
		t.Errorf("clvl 12 blockers %v", why)
	}
	h.Level, h.SkillPts, h.Skills = 13, 1, map[string]int{"Might": 1, "Sacrifice": 1}
	if got := planned(pal.Plan(h, fakeRule, nil)); !reflect.DeepEqual(got, []string{"Zeal"}) {
		t.Fatalf("clvl 13: %v", got)
	}
	// Both at once when two points are free; the table's own Sacrifice row (zealot-lite row 3) is marked Prereq too.
	h.Level, h.SkillPts, h.Skills = 13, 2, map[string]int{"Might": 1}
	if got := planned(pal.Plan(h, fakeRule, nil)); !reflect.DeepEqual(got, []string{"Sacrifice*", "Zeal"}) {
		t.Fatalf("clvl 13 two points: %v", got)
	}
	zl, err := LoadBuild("paladin-zealot-lite", "")
	if err != nil {
		t.Fatal(err)
	}
	h = Have{Level: 3, SkillPts: 1, Skills: map[string]int{"Might": 1}, Req: func(n string) []string { return fakeReq[n] }}
	if got := planned(zl.Plan(h, fakeRule, nil)); !reflect.DeepEqual(got, []string{"Sacrifice*"}) {
		t.Fatalf("zealot-lite clvl 3: %v", got)
	}
}

// Second build: a Necromancer table that names Bone Spear but neither of its prerequisites gets the chain bought first.
// A skill the rule refuses for its level is reported as a blocker, never silently dropped.
func TestPlanPrerequisiteChainNecromancer(t *testing.T) {
	nec, err := ParseBuild([]byte(`{"build":"nec_p","class":"necromancer","rows":[
		{"clvl":18,"skills":[{"skill":"Bone Spear","points":3}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	h := Have{Level: 18, SkillPts: 3, Skills: map[string]int{}, Req: func(n string) []string { return fakeReq[n] }}
	if got := planned(nec.Plan(h, fakeRule, nil)); !reflect.DeepEqual(got, []string{"Teeth*", "Corpse Explosion*", "Bone Spear"}) {
		t.Fatalf("chain: %v", got)
	}
	h.Level = 17
	acts, why := nec.PlanWhy(h, fakeRule, nil)
	if got := planned(acts); !reflect.DeepEqual(got, []string{"Teeth*", "Corpse Explosion*"}) {
		t.Fatalf("clvl 17: %v", got)
	}
	if why["Bone Spear"] == "" {
		t.Errorf("no blocker for Bone Spear: %v", why)
	}
}
