package hero

import (
	"strings"
	"testing"

	"d2lod-bot/internal/leveler"
)

// fakeTables numbers every skill name in the order it is first asked; the index is the id (one class per test).
type fakeTables struct{ ids map[string]int }

func (f *fakeTables) SkillID(name string) (int, bool) {
	if f.ids == nil {
		f.ids = map[string]int{"Attack": 0}
	}
	id, ok := f.ids[name]
	if !ok {
		id = len(f.ids)
		f.ids[name] = id
	}
	return id, true
}
func (f *fakeTables) SkillIndex(class string, id int) (int, bool) { return id, id > 0 }
func (f *fakeTables) Rule(string) leveler.Rule                    { return nil }
func (f *fakeTables) StarterWeapon(class string) string {
	return map[string]string{"sorceress": "sst", "paladin": "ssd", "necromancer": "wnd"}[class]
}

// played builds the played hero and the game state a correct prep leaves: every planned point spent, attack bound.
func played(t *testing.T, class string, level int) (Hero, Observed) {
	t.Helper()
	ft := &fakeTables{}
	h, err := Resolve(Decl{Profile: Played, Class: class, Level: level, QuestSkillPts: 1}, ft) // the Den point the tables tag
	if err != nil {
		t.Fatalf("%s clvl %d: %v", class, level, err)
	}
	lv := make([]int, 40)
	for name, n := range h.Skills {
		id, _ := ft.SkillID(name)
		lv[id] = n
	}
	l, sp, kp, w := h.Level, h.StatPts, h.SkillPts, h.Weapon
	o := Observed{Level: &l, StatPts: &sp, SkillPts: &kp, SkillLevels: lv, Weapon: &w}
	if h.AttackLeft {
		o.LeftSkill = &h.AttackID
	} else {
		o.RightSkill = &h.AttackID
	}
	return h, o
}

// The guard of design 6.2: the played profile of two classes spends its points by the table, learns and binds its main
// attack and passes the self-check against the state it declares. A default change that strands heroes fails here.
func TestPlayedTwoClassesPassSelfCheck(t *testing.T) {
	for _, c := range []struct {
		class string
		level int
	}{{"sorceress", 12}, {"paladin", 16}, {"necromancer", 12}} {
		h, o := played(t, c.class, c.level)
		if h.Attack == "" || h.Attack == "Attack" || h.Skills[h.Attack] == 0 {
			t.Errorf("%s: main attack %q has %d points", c.class, h.Attack, h.Skills[h.Attack])
		}
		if h.StatPts != 0 {
			t.Errorf("%s: %d stat points left; the table spends 5 a level", c.class, h.StatPts)
		}
		var cmds, skillpts int
		for _, v := range h.Verbs {
			if !v.Write {
				cmds++
				if strings.HasPrefix(v.Line, "cmd skillpt ") {
					skillpts++
				}
			}
			if v.Reason == "" {
				t.Errorf("%s: verb %q without a reason", c.class, v.Line)
			}
		}
		if skillpts != c.level-h.SkillPts {
			t.Errorf("%s: %d skillpt commands, want %d", c.class, skillpts, c.level-h.SkillPts)
		}
		if h.Tags[0] != "CHEATED" {
			t.Errorf("%s: tags %v, a setup hero is CHEATED", c.class, h.Tags)
		}
		if f := Failed(h.SelfCheck(o)); f != "" {
			t.Errorf("%s: self-check failed %s: %+v", c.class, f, h.SelfCheck(o))
		}
	}
}

// featuresoak pass 1: the hero came out of the prep with every point banked. The self-check of a played hero must fail.
func TestSelfCheckCatchesBankedPlayedHero(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		h, o := played(t, class, 12)
		all, sk := 55, 12
		o.StatPts, o.SkillPts, o.SkillLevels, o.LeftSkill, o.RightSkill = &all, &sk, make([]int, 40), nil, nil
		f := Failed(h.SelfCheck(o))
		for _, want := range []string{"statpts", "skillpts", "attack"} {
			if !strings.Contains(f, want) {
				t.Errorf("%s: failed %q, want %s in it", class, f, want)
			}
		}
	}
}

func TestBankedProfileKeepsEveryPoint(t *testing.T) {
	h, err := Resolve(Decl{Profile: Banked, Class: "Paladin", Level: 10, Weapon: "none"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if h.StatPts != 45 || h.SkillPts != 9 || h.Weapon != "" || len(h.Skills) != 0 {
		t.Fatalf("banked hero %+v", h)
	}
	for _, v := range h.Verbs {
		if !v.Write {
			t.Errorf("banked hero sends a player command %q", v.Line)
		}
	}
	l, s, k := 10, 45, 9
	if f := Failed(h.SelfCheck(Observed{Level: &l, StatPts: &s, SkillPts: &k})); f != "" {
		t.Errorf("banked self-check failed %s", f)
	}
	if !strings.Contains(h.Provenance("PgX"), "CHEATED") {
		t.Errorf("provenance %q", h.Provenance("PgX"))
	}
}

func TestResolveErrors(t *testing.T) {
	ft := &fakeTables{}
	for _, d := range []Decl{
		{Profile: Played, Class: "Monk", Level: 12},                                 // unknown class: no Sorceress default
		{Profile: Played, Class: "paladin", Build: "sorceress_normal30", Level: 12}, // table of another class
		{Profile: "half", Class: "paladin", Level: 12},
		{Profile: Played, Class: "paladin", Level: 1},
	} {
		if _, err := Resolve(d, ft); err == nil {
			t.Errorf("%+v: no error", d)
		}
	}
	// the Den point is quest-tagged in the Paladin table, so a missing Den no longer delays Zeal at clvl 12 (QUESTPTS)
	if h, err := Resolve(Decl{Profile: Played, Class: "paladin", Level: 12}, ft); err != nil || h.Skills["Zeal"] == 0 {
		t.Errorf("paladin clvl 12 without the Den point: %v, skills %v", err, h.Skills)
	}
	if _, err := Resolve(Decl{Profile: Played, Class: "paladin", Level: 12}, nil); err == nil {
		t.Error("played without tables: no error")
	}
}

func TestReadPointDump(t *testing.T) {
	var o Observed
	if err := ReadPointDump(&o, `{"player":{"level":12,"stat_pts":0,"skill_pts":1},"skills":{"base_levels":[0,3,1]}}`); err != nil {
		t.Fatal(err)
	}
	if *o.Level != 12 || *o.StatPts != 0 || *o.SkillPts != 1 || o.SkillLevels[1] != 3 {
		t.Fatalf("%+v", o)
	}
}
