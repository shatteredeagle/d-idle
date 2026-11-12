package skilltree

import (
	"strings"
	"testing"

	"d2lod-bot/internal/data"
)

// Fake tables for two classes: the builder holds no class, so both go through the same code.
func fixture() Tables {
	sk := []*data.Skill{
		{ID: 36, Key: "Fire Bolt", Name: "Fire Bolt", Class: "sor", Page: 1, Row: 1, Column: 2, ReqLevel: 1, MaxLevel: 20, Mana: 5, LvlMana: 1, ManaShift: 7, MinMana: 1, ShortKey: "sd36", LongKey: "ld36"},
		{ID: 41, Key: "Warmth", Name: "Warmth", Class: "sor", Page: 1, Row: 1, Column: 1, ReqLevel: 1, MaxLevel: 20, Passive: true},
		{ID: 47, Key: "Fire Ball", Name: "Fire Ball", Class: "sor", Page: 1, Row: 5, Column: 2, ReqLevel: 18, MaxLevel: 20, ReqSkills: [3]string{"Fire Bolt"}, Mana: 10, ManaShift: 8, EDmgSymPerCalc: "skill('Fire Bolt'.blvl)"},
		{ID: 40, Key: "Frozen Armor", Name: "Frozen Armor", Class: "sor", Page: 3, Row: 2, Column: 2, ReqLevel: 1, MaxLevel: 20},
		{ID: 97, Key: "Sacrifice", Name: "Sacrifice", Class: "pal", Page: 1, Row: 1, Column: 1, ReqLevel: 1, MaxLevel: 20},
		{ID: 107, Key: "Charge", Name: "Charge", Class: "pal", Page: 1, Row: 5, Column: 3, ReqLevel: 24, MaxLevel: 1, ReqSkills: [3]string{"Sacrifice"}},
		{ID: 98, Key: "Smite", Name: "Smite", Class: "pal", Page: 1, Row: 2, Column: 2, ReqLevel: 6, MaxLevel: 20, ReqSkills: [3]string{"Sacrifice"}},
		{ID: 0, Key: "attack", Class: ""},
	}
	return Tables{Skills: sk, Text: func(k string) string {
		return map[string]string{"sd36": "short fire", "ld36": "long fire"}[k]
	}}
}

func node(v View, name string) Node {
	for _, t := range v.Trees {
		for _, n := range t.Nodes {
			if n.Name == name {
				return n
			}
		}
	}
	return Node{}
}

func TestTwoClasses(t *testing.T) {
	tb := fixture()
	sor := Build(tb, Hero{ClassCode: "sor", ClassName: "Sorceress", Level: 20, Unspent: 2, Points: map[int]Points{36: {Base: 5, Bonus: 1}, 41: {Bonus: 2}}}, nil)
	if !sor.Available || len(sor.Trees) != 2 || sor.Trees[0].Tab != 1 || sor.Trees[1].Tab != 3 {
		t.Fatalf("sor trees: %+v", sor)
	}
	if sor.Spent != 5 || sor.Trees[0].Spent != 5 || sor.Unspent != 2 {
		t.Errorf("totals: spent %d tree %d unspent %d", sor.Spent, sor.Trees[0].Spent, sor.Unspent)
	}
	fb := node(sor, "Fire Bolt")
	if fb.State != "learned" || fb.Level != 6 || fb.Short != "short fire" || fb.Long != "long fire" || fb.ManaNow == nil || fb.ManaNext == nil || *fb.ManaNow >= *fb.ManaNext {
		t.Errorf("fire bolt: %+v", fb)
	}
	if w := node(sor, "Warmth"); w.State != "bonus" || w.ManaNow != nil || w.ManaNext != nil {
		t.Errorf("warmth: %+v", w)
	}
	ball := node(sor, "Fire Ball")
	if !ball.CanAdd || ball.State != "learnable" || len(ball.Prereqs) != 1 || !ball.Prereqs[0].Have || len(ball.Synergy) != 1 || ball.Synergy[0] != "Fire Bolt" {
		t.Errorf("fire ball: %+v", ball)
	}
	if ball.ManaNow != nil {
		t.Error("an unlearned skill has no current mana cost")
	}

	pal := Build(tb, Hero{ClassCode: "pal", ClassName: "Paladin", Level: 10, Unspent: 0, Points: map[int]Points{97: {Base: 1}}}, nil)
	if len(pal.Trees) != 1 || len(pal.Trees[0].Nodes) != 3 || node(pal, "Fire Bolt").ID != 0 {
		t.Fatalf("pal trees: %+v", pal.Trees)
	}
	if s := node(pal, "Smite"); s.CanAdd || s.Why != WhyNoPoints || s.State != "learnable" {
		t.Errorf("smite with no points: %+v", s)
	}
	if c := node(pal, "Charge"); c.State != "locked" || c.Why != "needs character level 24" {
		t.Errorf("charge: %+v", c)
	}
}

func TestCanAddReasons(t *testing.T) {
	tb := fixture()
	by := map[string]*data.Skill{}
	for _, s := range tb.Skills {
		by[strings.ToLower(s.Key)] = s
	}
	cases := []struct {
		skill string
		h     Hero
		ok    bool
		why   string
	}{
		{"Fire Ball", Hero{Level: 30, Unspent: 1}, false, "needs Fire Bolt"},
		{"Fire Ball", Hero{Level: 17, Unspent: 1, Points: map[int]Points{36: {Base: 1}}}, false, "needs character level 18"},
		{"Fire Ball", Hero{Level: 18, Unspent: 0, Points: map[int]Points{36: {Base: 1}}}, false, WhyNoPoints},
		{"Fire Ball", Hero{Level: 18, Unspent: 1, Points: map[int]Points{36: {Bonus: 3}}}, false, "needs Fire Bolt"}, // bonus points do not satisfy a prerequisite (U)
		{"Fire Ball", Hero{Level: 18, Unspent: 1, Points: map[int]Points{36: {Base: 1}}}, true, ""},
		{"Charge", Hero{Level: 40, Unspent: 3, Points: map[int]Points{97: {Base: 1}, 107: {Base: 1}}}, false, WhyMax},
		{"Smite", Hero{Level: 6, Unspent: 1, Points: map[int]Points{97: {Base: 1}}}, true, ""},
	}
	for i, c := range cases {
		ok, why := CanAdd(by[strings.ToLower(c.skill)], c.h, by)
		if ok != c.ok || why != c.why {
			t.Errorf("case %d %s: got %v %q want %v %q", i, c.skill, ok, why, c.ok, c.why)
		}
	}
}

func TestUnavailable(t *testing.T) {
	tb := fixture()
	for _, code := range []string{"", "zzz"} {
		v := Build(tb, Hero{ClassCode: code}, nil)
		if v.Available || len(v.Trees) != 0 || v.Schema != Schema {
			t.Errorf("code %q: %+v", code, v)
		}
	}
}
