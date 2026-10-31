package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

func TestAffixResists(t *testing.T) {
	// fire 60 + magic resistant 40 = 100 (immune), cold 0 -> 40
	r := AffixResists([6]int{0, 0, 60, 0, 0, 0}, AffixMagicResistant)
	if r[data.Fire] != 100 || r[data.Cold] != 40 || r[data.Lightning] != 40 {
		t.Fatalf("%v", r)
	}
	// already two immunities: a third is refused
	r = AffixResists([6]int{0, 0, 100, 30, 100, 0}, AffixLightningEnchanted)
	if r[data.Lightning] != 30 {
		t.Fatalf("third immunity created: %v", r)
	}
	r = AffixResists([6]int{}, AffixColdEnchanted|AffixManaBurn)
	if r[data.Cold] != 75 || r[data.Magic] != 20 {
		t.Fatalf("%v", r)
	}
}

func TestEnchantedFireBecomesImmuneFromTable(t *testing.T) {
	s := baseSnap()
	s.Player.Difficulty = d2.Hell
	m := mon(2, clsFallen, 110, 100)
	m.Monster.Flags = 1 << 3
	m.Monster.Enchants = [9]uint8{9}
	s.Monsters = []d2.Unit{m}
	cat := testCatalog()
	cat.Monsters[clsFallen].Res[2][data.Fire] = 30
	hc := HostileCtx{Cat: cat, Diff: d2.Hell, ClassLvl: 80}
	hs := Hostiles(s, hc)
	if len(hs) != 1 || hs[0].Res[data.Fire] != 105 || !hs[0].Immune(data.Fire) {
		t.Fatalf("%+v", hs)
	}
}

func TestTargetDanger(t *testing.T) {
	tg := func(a Affix) Target { return Target{Affixes: a, Elite: a != 0} }
	cases := []struct {
		name string
		t    Target
		diff d2.Difficulty
		lvl  int
		want int
	}{
		{"trash", Target{}, d2.Hell, 60, 0},
		{"fast normal", tg(AffixExtraFast), d2.Normal, 20, 1},
		{"fast hell", tg(AffixExtraFast), d2.Hell, 80, 2},
		{"tele nm", tg(AffixTeleportation), d2.Nightmare, 50, 2},
		{"stone skin", tg(AffixStoneSkin), d2.Hell, 80, 1},
		{"mag res", tg(AffixMagicResistant), d2.Hell, 80, 1},
		{"mana burn hell", tg(AffixManaBurn), d2.Hell, 80, 2},
		{"mana burn low", tg(AffixManaBurn), d2.Hell, 60, 3},
		{"strong+fast low", tg(AffixExtraStrong | AffixExtraFast), d2.Hell, 60, 3},
		{"strong+fast high", tg(AffixExtraStrong | AffixExtraFast), d2.Hell, 80, 2},
		{"fire ench normal", tg(AffixFireEnchanted), d2.Normal, 10, 1},
		{"fire ench hell", Target{Affixes: AffixFireEnchanted}, d2.Hell, 80, 0},
		{"fire ench hell elite", tg(AffixFireEnchanted), d2.Hell, 80, 1},
	}
	for _, c := range cases {
		if got := TargetDanger(c.t, c.diff, c.lvl); got != c.want {
			t.Errorf("%s: %d want %d", c.name, got, c.want)
		}
	}
}

func TestPackDangerMinions(t *testing.T) {
	var ts []Target
	for i := 0; i < 5; i++ {
		ts = append(ts, Target{Elite: true, Unit: d2.Unit{Monster: &d2.MonsterInfo{Flags: 1 << 4}}})
	}
	if got := PackDanger(ts, d2.Hell, 70); got != DangerDeadly {
		t.Fatalf("hell minions: %d", got)
	}
	if got := PackDanger(ts, d2.Hell, 80); got != DangerElite {
		t.Fatalf("strong char: %d", got)
	}
	if got := PackDanger(ts, d2.Normal, 10); got != DangerShaman {
		t.Fatalf("normal: %d", got)
	}
}

func TestEntryGate(t *testing.T) {
	cases := []struct {
		d       d2.Difficulty
		act, lv int
		mlvl    int
		dun     bool
		ok      bool
	}{
		{d2.Normal, 1, 1, 0, false, true},
		{d2.Nightmare, 1, 35, 0, false, false},
		{d2.Nightmare, 1, 36, 0, false, true},
		{d2.Nightmare, 3, 49, 0, false, false},
		{d2.Hell, 1, 66, 0, false, true},
		{d2.Hell, 1, 66, 0, true, false},
		{d2.Hell, 1, 75, 0, true, true},
		{d2.Hell, 2, 71, 0, false, false},
		{d2.Hell, 5, 82, 0, false, true},
		{d2.Hell, 1, 70, 85, false, false}, // monster level 85 - 8 = 77
		{d2.Hell, 9, 99, 0, false, false},
	}
	for i, c := range cases {
		if r := EntryGate(c.d, c.act, c.lv, c.mlvl, c.dun); r.OK != c.ok {
			t.Errorf("case %d: %+v", i, r)
		}
	}
}

func TestPackSkips(t *testing.T) {
	f := &Fighter{cat: testCatalog(), keys: map[int]string{36: "F1"}}
	s := baseSnap()
	p := s.Player
	mk := func(id uint32, cls int, elite bool) Target {
		u := mon(id, cls, 110, 100)
		return Target{Unit: u, Mon: testCatalog().Monster(cls), Elite: elite, HasRes: true, Res: [6]int{0, 100, 100, 100, 100, 100}}
	}
	if len(f.attackSkills(p)) == 0 {
		t.Skip("fixture sorc has no usable attack skill")
	}
	if got := f.packSkips(p, []Target{mk(2, clsAllImmun, false)}); len(got) != 0 {
		t.Fatalf("lone immune trash is still tried: %d", len(got))
	}
	if got := f.packSkips(p, []Target{mk(2, clsAllImmun, true)}); len(got) != 1 {
		t.Fatalf("immune elite is skipped: %d", len(got))
	}
	if got := f.packSkips(p, []Target{mk(2, clsAllImmun, false), mk(3, clsAllImmun, false), mk(4, clsAllImmun, false)}); len(got) != 3 {
		t.Fatalf("immune pack of 3 is skipped: %d", len(got))
	}
}
