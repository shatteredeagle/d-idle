package leveler

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"io"
	"log/slog"
	"reflect"
	"testing"
)

func mustBuild(t *testing.T, name string) *Build {
	t.Helper()
	b, err := LoadBuild(name, "")
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestPlanLevelTwo(t *testing.T) {
	b := mustBuild(t, "fallback")
	h := Have{Attr: [4]int{10, 25, 10, 35}, Start: [4]int{10, 25, 10, 35}, Skills: map[string]int{}, Level: 2, StatPts: 5, SkillPts: 1}
	got := b.Plan(h, nil, nil)
	want := []Action{{Kind: KindStat, Attr: Vit}, {Kind: KindStat, Attr: Vit}, {Kind: KindStat, Attr: Vit}, {Kind: KindStat, Attr: Ene}, {Kind: KindStat, Attr: Ene},
		{Kind: KindSkill, Skill: "Charged Bolt"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestPlanIdempotentAfterSpend(t *testing.T) {
	b := mustBuild(t, "fallback")
	// two stat points already in vitality (hand-spent or an interrupted run): only three are left to place
	h := Have{Attr: [4]int{10, 25, 12, 35}, Start: [4]int{10, 25, 10, 35}, Skills: map[string]int{"Charged Bolt": 1}, Level: 2, StatPts: 3, SkillPts: 0}
	got := b.Plan(h, nil, nil)
	if len(got) != 3 || got[0].Attr != Vit {
		t.Fatalf("got %v", got)
	}
	h.StatPts = 0
	if got := b.Plan(h, nil, nil); len(got) != 0 {
		t.Fatalf("nothing unspent, got %v", got)
	}
}

func TestPlanQuestPointPullsForward(t *testing.T) {
	b := mustBuild(t, "fallback")
	// level 3 character with all level points spent in the table (2 skill points owned) plus one quest point
	h := Have{Attr: [4]int{10, 25, 20, 35}, Start: [4]int{10, 25, 10, 35}, Skills: map[string]int{"Charged Bolt": 2}, Level: 3, StatPts: 0, SkillPts: 1}
	got := b.Plan(h, nil, nil)
	if len(got) != 1 || got[0].Skill != "Charged Bolt" {
		t.Fatalf("got %v", got)
	}
}

func TestPlanRespectsRuleAndSkip(t *testing.T) {
	b := mustBuild(t, "fallback")
	// level 10 table has Telekinesis; refuse it: the rule blocks it and the plan moves on to later skills
	h := Have{Attr: [4]int{10, 25, 55, 35}, Start: [4]int{10, 25, 10, 35}, Skills: map[string]int{"Charged Bolt": 7, "Static Field": 1}, Level: 10, StatPts: 0, SkillPts: 2}
	rule := func(name string, _ int, _ map[string]int) bool { return name != "Telekinesis" }
	got := b.Plan(h, rule, nil)
	for _, a := range got {
		if a.Skill == "Telekinesis" {
			t.Fatalf("refused skill planned: %v", got)
		}
	}
	got = b.Plan(h, nil, map[string]bool{"Telekinesis": true, "Charged Bolt": true})
	for _, a := range got {
		if a.Skill == "Telekinesis" || a.Skill == "Charged Bolt" {
			t.Fatalf("skipped skill planned: %v", got)
		}
	}
}

func TestPlanPrerequisiteChain(t *testing.T) {
	b := mustBuild(t, "fallback")
	// Teleport needs Telekinesis in the plan data order; the virtual levels let both appear in one batch
	rule := func(name string, _ int, lv map[string]int) bool {
		if name == "Teleport" {
			return lv["Telekinesis"] > 0
		}
		return true
	}
	h := Have{Attr: [4]int{10, 25, 10, 35}, Start: [4]int{10, 25, 10, 35}, Skills: map[string]int{}, Level: 99, StatPts: 0, SkillPts: 30}
	var tele, tk = -1, -1
	for i, a := range b.Plan(h, rule, nil) {
		switch a.Skill {
		case "Teleport":
			tele = i
		case "Telekinesis":
			tk = i
		}
	}
	if tk < 0 || tele < 0 || tk > tele {
		t.Fatalf("telekinesis %d teleport %d", tk, tele)
	}
}

func TestPlanFloorSpendsMissingStrengthFirst(t *testing.T) {
	b := mustBuild(t, "fallback")
	h := Have{Attr: [4]int{10, 25, 10, 35}, Start: [4]int{10, 25, 10, 35}, Skills: map[string]int{}, Level: 2, StatPts: 5, SkillPts: 0,
		Floor: [4]int{Str: 12}}
	got := b.Plan(h, nil, nil)
	if len(got) != 5 || got[0] != (Action{Kind: KindStat, Attr: Str}) || got[1] != (Action{Kind: KindStat, Attr: Str}) || got[2].Attr != Vit {
		t.Fatalf("got %v", got)
	}
	h.Attr[Str] = 12 // floor met: the table alone
	if got := b.Plan(h, nil, nil); got[0].Attr != Vit {
		t.Fatalf("floor met, got %v", got)
	}
	h.Attr[Str], h.StatPts = 10, 1 // fewer points than the gap: all of them go to the floor
	if got := b.Plan(h, nil, nil); len(got) != 1 || got[0].Attr != Str {
		t.Fatalf("got %v", got)
	}
}

func TestNeedStatsRaisesFloorAndClassFloor(t *testing.T) {
	l := &Leveler{}
	p := &d2.Player{Class: d2.Sorceress}
	if f := l.floor(p); f[Str] != 12 {
		t.Fatalf("class floor %v", f)
	}
	l.e = &env.Env{Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	l.NeedStats(20, 0)
	l.NeedStats(99, 0) // capped
	l.NeedStats(0, 30)
	if f := l.floor(p); f[Str] != 40 || f[Dex] != 30 {
		t.Fatalf("floor %v", f)
	}
}
