package hero

import (
	"strings"
	"testing"
)

// The static half passes the default played hero of two classes and refuses an unknown build, a build of another
// class and a main attack the plan does not reach at that level.
func TestStaticTwoClasses(t *testing.T) {
	for _, d := range []Decl{
		{Profile: Played, Class: "sorceress", Level: 12, QuestSkillPts: 1},
		{Profile: Played, Class: "Paladin", Level: 16},
		{Profile: Banked, Class: "necromancer", Build: "necromancer_summoner_normal30", Level: 4},
	} {
		if f := Failed(Static(d, NameTables())); f != "" {
			t.Errorf("%+v: %s: %+v", d, f, Static(d, NameTables()))
		}
	}
	for _, c := range []struct {
		d    Decl
		want string
	}{
		{Decl{Profile: Played, Class: "paladin", Build: "no-such-build", Level: 12}, "build"},
		{Decl{Profile: Played, Class: "paladin", Build: "sorceress_normal30", Level: 12}, "build"},
		{Decl{Profile: Played, Class: "barbarian", Level: 1}, "build"},
		// the summoner table folds the Den point in: without it Teeth gets no point at clvl 4 (V go run ./cmd/d2scenario validate)
		{Decl{Profile: Played, Class: "necromancer", Build: "necromancer_summoner_normal30", Level: 4}, "attack"},
	} {
		if f := Failed(Static(c.d, NameTables())); f != c.want {
			t.Errorf("%+v: failed %q, want %q", c.d, f, c.want)
		}
	}
}

// Must fail: a prep whose purse or potion verbs gave nothing is refused, and an unread purse is no pass.
func TestSelfCheckFloors(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		h, o := played(t, class, 12)
		f := Floors{Purse: 3000, Potions: 6}
		gold, pots := 3000, 6
		o.Gold, o.Potions = &gold, &pots
		if bad := Failed(h.SelfCheckFloors(o, f)); bad != "" {
			t.Errorf("%s: %s", class, bad)
		}
		empty := 0
		o.Gold = &empty
		if bad := Failed(h.SelfCheckFloors(o, f)); bad != "purse" {
			t.Errorf("%s empty purse: failed %q", class, bad)
		}
		o.Gold, o.Potions = nil, &empty
		if bad := Failed(h.SelfCheckFloors(o, f)); !strings.Contains(bad, "purse") || !strings.Contains(bad, "potions") {
			t.Errorf("%s unread purse, no potions: failed %q", class, bad)
		}
		if n := len(h.SelfCheckFloors(o, Floors{})); n != len(h.SelfCheck(o)) {
			t.Errorf("%s: zero floors add rows", class)
		}
	}
}
