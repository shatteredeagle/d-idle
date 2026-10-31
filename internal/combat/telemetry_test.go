package combat

import (
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

func TestDeathLine(t *testing.T) {
	di := DeathInfo{Difficulty: d2.Hell, Area: 107, CharLevel: 70, MonsterLevel: 85, MonsterName: "Oblivion Knight",
		Affixes: AffixCursed | AffixManaBurn, Distance: 4}
	if di.LevelGap() != 15 {
		t.Fatal(di.LevelGap())
	}
	l := di.Line()
	for _, want := range []string{"diff=Hell", "gap=15", "killer=Oblivion_Knight", "affixes=cursed,mana-burn"} {
		if !strings.Contains(l, want) {
			t.Fatalf("%q missing in %s", want, l)
		}
	}
	m, a := di.Weights("doomknight3")
	if m != 2 || a["cursed"] != 2 {
		t.Fatalf("%v %v", m, a)
	}
	di.Difficulty = d2.Nightmare
	if m, _ := di.Weights("x"); m != 1 {
		t.Fatal(m)
	}
	if (DeathInfo{}).LevelGap() != 0 || !strings.Contains((DeathInfo{}).Line(), "affixes=-") {
		t.Fatal("empty")
	}
}

func TestAffixTags(t *testing.T) {
	if AffixTags(nil) != "-" {
		t.Fatal()
	}
	got := AffixTags([]Target{{Affixes: AffixCursed}, {Affixes: AffixExtraFast}})
	if got != "extra-fast,cursed" {
		t.Fatal(got)
	}
}

func TestLevelGapOK(t *testing.T) {
	cases := []struct {
		d    d2.Difficulty
		c, m int
		want bool
	}{
		{d2.Normal, 10, 12, true}, {d2.Normal, 9, 12, false}, {d2.Nightmare, 31, 36, true},
		{d2.Hell, 77, 85, true}, {d2.Hell, 74, 85, false}, {d2.Hell, 80, 0, true},
	}
	for _, c := range cases {
		if LevelGapOK(c.d, c.c, c.m) != c.want {
			t.Errorf("%+v", c)
		}
	}
}
