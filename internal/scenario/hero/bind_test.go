package hero

import (
	"sort"
	"strings"
	"testing"
)

// bound builds a played hero, binds a second learned skill of its plan on hand and returns the game state a correct
// prep leaves (the main attack on its hand unless the bind took it).
func bound(t *testing.T, class string, level int, hand string) (Hero, Observed, Bind) {
	t.Helper()
	ft := &fakeTables{}
	h, err := Resolve(Decl{Profile: Played, Class: class, Level: level, QuestSkillPts: 1}, ft)
	if err != nil {
		t.Fatalf("%s: %v", class, err)
	}
	var names []string
	for n := range h.Skills {
		if n != h.Attack {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		t.Fatalf("%s clvl %d: the plan learns only the attack", class, level)
	}
	verbs := len(h.Verbs)
	if err := h.BindHand(names[0], hand, ft); err != nil {
		t.Fatalf("%s: %v", class, err)
	}
	if len(h.Verbs) != verbs+1 || !strings.HasPrefix(h.Verbs[verbs].Line, "cmd select ") || !strings.HasPrefix(h.Verbs[verbs].Reason, "SETUP: ") || h.Verbs[verbs].Write {
		t.Fatalf("%s: bind verb %+v", class, h.Verbs[verbs:])
	}
	lv := make([]int, 60)
	for name, n := range h.Skills {
		id, _ := ft.SkillID(name)
		lv[id] = n
	}
	l, sp, kp, w := h.Level, h.StatPts, h.SkillPts, h.Weapon
	o := Observed{Level: &l, StatPts: &sp, SkillPts: &kp, SkillLevels: lv, Weapon: &w}
	b := h.Binds[0]
	if h.AttackLeft {
		o.LeftSkill = &h.AttackID
	} else {
		o.RightSkill = &h.AttackID
	}
	if b.Left {
		o.LeftSkill = &b.ID
	} else {
		o.RightSkill = &b.ID
	}
	return h, o, b
}

// A declared hand skill passes the self-check when it is learned and on its hand, on either hand and for three
// classes; the attack row does not fail because the bind took the attack's hand.
func TestBindHandPassesSelfCheck(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin", "necromancer"} {
		for _, hand := range []string{"left", "right"} {
			h, o, _ := bound(t, class, 16, hand)
			if f := Failed(h.SelfCheck(o)); f != "" {
				t.Errorf("%s %s: self-check failed %s: %+v", class, hand, f, h.SelfCheck(o))
			}
		}
	}
}

// Must-fail: the hand holds something else (the select was not obeyed), or the skill has no point.
func TestBindHandSelfCheckMustFail(t *testing.T) {
	for _, class := range []string{"paladin", "necromancer"} {
		h, o, b := bound(t, class, 16, "right")
		other := b.ID + 1
		wrong := o
		wrong.RightSkill = &other
		if f := Failed(h.SelfCheck(wrong)); !strings.Contains(f, "bind") {
			t.Errorf("%s: another skill on the hand: failed %q, want bind", class, f)
		}
		wrong = o
		wrong.RightSkill = nil
		if f := Failed(h.SelfCheck(wrong)); !strings.Contains(f, "bind") {
			t.Errorf("%s: hand not read: failed %q, want bind", class, f)
		}
		wrong = o
		wrong.SkillLevels = append([]int(nil), o.SkillLevels...)
		wrong.SkillLevels[b.Index] = 0
		if f := Failed(h.SelfCheck(wrong)); f != "bind" {
			t.Errorf("%s: unlearned bind skill: failed %q, want bind", class, f)
		}
	}
}

// BindHand refuses what it cannot bind and names it: a skill that is not the class's, a bad hand, a second skill on a
// taken hand, a banked hero.
func TestBindHandErrors(t *testing.T) {
	for _, class := range []string{"sorceress", "necromancer"} {
		ft := &fakeTables{}
		h, err := Resolve(Decl{Profile: Played, Class: class, Level: 12, QuestSkillPts: 1}, ft)
		if err != nil {
			t.Fatal(err)
		}
		if err := h.BindHand("Attack", "right", ft); err == nil || !strings.Contains(err.Error(), "Attack") || !strings.Contains(err.Error(), class) { // fakeTables: id 0 is no class skill
			t.Errorf("%s: a skill of no class: %v", class, err)
		}
		if err := h.BindHand(h.Attack, "both", ft); err == nil {
			t.Errorf("%s: hand both accepted", class)
		}
		if err := h.BindHand(h.Attack, "right", ft); err != nil {
			t.Errorf("%s: %v", class, err)
		}
		if err := h.BindHand("Other Skill", "right", ft); err == nil || !strings.Contains(err.Error(), "both on the right hand") {
			t.Errorf("%s: two skills on one hand: %v", class, err)
		}
		b, err := Resolve(Decl{Profile: Banked, Class: class, Level: 12}, nil)
		if err != nil {
			t.Fatal(err)
		}
		if err := b.BindHand("Teeth", "right", ft); err == nil {
			t.Errorf("%s: a banked hero took a bind", class)
		}
	}
}

// The minions row: not read and too few fail, enough pass; no row without a declared summon.
func TestSelfCheckMinionsRow(t *testing.T) {
	for _, class := range []string{"necromancer", "paladin"} {
		h, o := played(t, class, 12)
		if strings.Contains(Failed(h.SelfCheck(o)), "minions") {
			t.Fatalf("%s: minions row without a declared summon", class)
		}
		h.MinMinions = 2
		if f := Failed(h.SelfCheck(o)); f != "minions" {
			t.Errorf("%s: not read: failed %q", class, f)
		}
		one, two := 1, 2
		o.Minions = &one
		if f := Failed(h.SelfCheck(o)); f != "minions" {
			t.Errorf("%s: one of two: failed %q", class, f)
		}
		o.Minions = &two
		if f := Failed(h.SelfCheck(o)); f != "" {
			t.Errorf("%s: two of two: failed %q", class, f)
		}
	}
}
