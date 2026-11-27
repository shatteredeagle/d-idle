package d2dctl_test

import (
	"testing"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
)

// d2d reads the "cmd skill" tail as numbers (fake.ParseSkill mirrors devctl/verbs.hpp): a left cast must arrive as
// the left hand, and a cast with no unit as unit -1, the value that takes the at-the-spot paths.
func TestCmdSkillHandAndUnitAsD2dReadsThem(t *testing.T) {
	srv, c := start(t, fake.Options{})
	casts := []d2dctl.SkillOpts{
		{Unit: 5, Left: true},
		{Left: true},
		{Unit: 5},
		{},
		{Item: 9},
		{Left: true, Item: 9},
	}
	for _, o := range casts {
		if err := c.CmdSkill(ctx(), 36, 13, 29.5, o); err != nil {
			t.Fatal(err)
		}
	}
	want := []fake.UseSkill{
		{Skill: 36, X: 13, Y: 29.5, Unit: 5, Left: true, Item: -1},
		{Skill: 36, X: 13, Y: 29.5, Unit: -1, Left: true, Item: -1},
		{Skill: 36, X: 13, Y: 29.5, Unit: 5, Item: -1},
		{Skill: 36, X: 13, Y: 29.5, Unit: -1, Item: -1},
		{Skill: 36, X: 13, Y: 29.5, Unit: -1, Item: 9},
		{Skill: 36, X: 13, Y: 29.5, Unit: -1, Left: true, Item: 9},
	}
	got := srv.Skills()
	if len(got) != len(want) {
		t.Fatalf("got %d casts %+v", len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("cast %d %+v: d2d reads %+v, want %+v (line %q)", i, casts[i], got[i], want[i], srv.Lines()[i])
		}
	}
}

// The old word form reads as the right hand, which is the bug this guards.
func TestParseSkillWordHandIsRight(t *testing.T) {
	u, ok := fake.ParseSkill([]string{"cmd", "skill", "0", "13", "29.5", "5", "left"})
	if !ok || u.Left {
		t.Fatalf("%+v %v: d2d's atoi reads the word left as 0 (right hand)", u, ok)
	}
}
