package leveler

import (
	"reflect"
	"testing"

	"d2lod-bot/internal/d2"
)

// palzeal: a table point tagged with a quest is paid by that quest's bonus point. Without the quest the rows keep their
// level (no shift); with it the bonus point is spent at once. Second build: a Necromancer Teeth table.
func TestQuestTaggedPointsSpentOnGrantOnly(t *testing.T) {
	b, err := ParseBuild([]byte(`{"build":"necro_q","class":"necromancer","rows":[
		{"clvl":2,"skills":[{"skill":"Teeth","points":1}]},
		{"clvl":3,"skills":[{"skill":"Teeth","points":1},{"skill":"Bone Armor","points":1,"quest":"den_of_evil"}]},
		{"clvl":4,"skills":[{"skill":"Corpse Explosion","points":1}]},
		{"clvl":5,"skills":[{"skill":"Teeth","points":1}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	skills := func(acts []Action) (out []string) {
		for _, a := range acts {
			out = append(out, a.Skill)
		}
		return out
	}
	cases := []struct {
		name string
		h    Have
		want []string
	}{
		{"clvl 4 no Den: the row 4 point", Have{Level: 4, SkillPts: 1, Skills: map[string]int{"Teeth": 2}}, []string{"Corpse Explosion"}},
		{"Den granted at clvl 3: spent at once", Have{Level: 3, SkillPts: 1, QuestPts: 1, Skills: map[string]int{"Teeth": 2}}, []string{"Bone Armor"}},
		{"clvl 4 with Den banked", Have{Level: 4, SkillPts: 2, QuestPts: 1, Skills: map[string]int{"Teeth": 2}}, []string{"Bone Armor", "Corpse Explosion"}},
		{"clvl 5 no Den: no shift", Have{Level: 5, SkillPts: 1, Skills: map[string]int{"Teeth": 2, "Corpse Explosion": 1}}, []string{"Teeth"}},
	}
	for _, c := range cases {
		if got := skills(b.Plan(c.h, nil, nil)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: %v want %v", c.name, got, c.want)
		}
	}
}

// The Paladin table: a fresh Zealot-lite hero at clvl 12 without the Den spends its point on Zeal (live 2026-11-27 it
// went to Might because row 5 assumed the Den point).
func TestZealotLiteZealAtTwelveWithoutDen(t *testing.T) {
	b, err := LoadBuild("paladin-zealot-lite", "")
	if err != nil {
		t.Fatal(err)
	}
	h := Have{Level: 12, SkillPts: 1, Skills: map[string]int{"Might": 8, "Sacrifice": 1, "Prayer": 1}}
	if got := b.Plan(h, nil, nil); len(got) != 1 || got[0].Skill != "Zeal" {
		t.Fatalf("clvl 12 plan %v", got)
	}
}

// QuestPoints counts hard points in the class tree plus unspent beyond clvl-1; Attack and the scroll skills do not count.
func TestQuestPointsFromSkills(t *testing.T) {
	db := realDB(t)
	might, zeal := db.SkillByName("Might"), db.SkillByName("Zeal")
	if might == nil || zeal == nil {
		t.Fatal("skills missing")
	}
	p := &d2.Player{Class: d2.Paladin, Level: 5, SkillPoints: 1, Skills: []d2.Skill{{ID: 0, Level: 1}, {ID: might.ID, Level: 4}}}
	if n := QuestPoints(db, p); n != 1 {
		t.Errorf("Den done: %d", n)
	}
	p.SkillPoints = 0
	if n := QuestPoints(db, p); n != 0 {
		t.Errorf("no Den: %d", n)
	}
}
