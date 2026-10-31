package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
)

func champUnit(id uint32, x int) d2.Unit {
	m := mon(id, clsFallen, x, 100)
	m.Monster.Flags = 1 << 2
	return m
}

func TestChampionIsEliteNotBoss(t *testing.T) {
	s := baseSnap()
	named := mon(3, clsFallen, 110, 100)
	named.Monster.Flags, named.Monster.Name = 1<<3, "Bishibosh"
	s.Monsters = []d2.Unit{champUnit(2, 108), named, mon(4, clsFallen, 112, 100)}
	for _, h := range Hostiles(s, hctx()) {
		switch h.Unit.ID {
		case 2:
			if !h.Champion || !h.Elite || h.Boss {
				t.Fatalf("champion must be elite, champion, not boss: %+v", h)
			}
		case 3:
			if h.Champion || !h.Elite || !h.Boss {
				t.Fatalf("named unique is a boss, not a champion: %+v", h)
			}
		case 4:
			if h.Champion || h.Elite || h.Boss {
				t.Fatalf("plain monster: %+v", h)
			}
		}
	}
}

func TestChampionRanksBetweenSupportAndNormal(t *testing.T) {
	s := baseSnap()
	s.Monsters = []d2.Unit{mon(2, clsFallen, 112, 100), champUnit(3, 114), mon(4, clsShaman, 124, 100)}
	r := Rank(Hostiles(s, hctx()), 100, 0)
	if got := ids(r); len(got) != 3 || got[0] != 4 || got[1] != 3 || got[2] != 2 {
		t.Fatalf("want shaman, champion, normal: %v", got)
	}
}

func TestChampionDangerBetweenEliteAndBoss(t *testing.T) {
	cases := []struct {
		name string
		t    Target
		want int
	}{
		{"normal", Target{}, DangerNone},
		{"champion", Target{Elite: true, Champion: true}, DangerShaman},
		{"boss", Target{Elite: true, Boss: true}, DangerElite},
	}
	for _, c := range cases {
		if got := TargetDanger(c.t, d2.Normal, 10); got != c.want {
			t.Errorf("%s: %d want %d", c.name, got, c.want)
		}
	}
}

func TestChampionGroupAvoidedAtLowLevel(t *testing.T) {
	s := baseSnap()
	s.Monsters = []d2.Unit{champUnit(2, 108), champUnit(3, 110), champUnit(4, 112)}
	count := func(lvl int, n int) int {
		hc := hctx()
		hc.ClassLvl = lvl
		s.Monsters = s.Monsters[:n]
		a := 0
		for _, h := range Hostiles(s, hc) {
			if h.Avoid {
				a++
			}
		}
		return a
	}
	if got := count(5, 3); got != 3 {
		t.Fatalf("3 champions at clvl 5: avoid all, got %d", got)
	}
	if got := count(5, 2); got != 0 {
		t.Fatalf("2 champions are fought, got %d avoided", got)
	}
	s.Monsters = []d2.Unit{champUnit(2, 108), champUnit(3, 110), champUnit(4, 112)}
	if got := count(8, 3); got != 0 {
		t.Fatalf("clvl 8 fights the group, got %d avoided", got)
	}
}
