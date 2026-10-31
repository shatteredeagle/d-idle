package main

import (
	"context"
	"errors"
	"testing"

	"d2lod-bot/internal/boss"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/task"
)

func TestBossViewReadsPoisonMonstersDangersAndAntidotes(t *testing.T) {
	cat := combat.MapCatalog{
		Monsters: map[int]*data.Monster{267: {Key: "bloodraven", Name: "Blood Raven", Killable: true}, 1: {Key: "skeleton", Name: "Skeleton", Killable: true}},
		Items:    map[int]combat.ItemClass{587: {Potion: combat.PotionHealing, Tier: 1}, 9100: {Potion: combat.PotionAntidote, Tier: 1}, 529: {Portal: true}},
	}
	g := &bossGame{cat: cat, cloudIDs: map[int]bool{217: true}}
	var states d2.StateSet
	states[0] = 1 << d2.StatePoison
	pot := func(class, col int) d2.Unit {
		return d2.Unit{Type: d2.TypeItem, Class: class, Item: &d2.Item{Where: d2.ContBelt, GX: col}}
	}
	s := &d2.Snapshot{
		State:  d2.StateInGame,
		Player: &d2.Player{ID: 1, X: 100, Y: 100, Area: 17, Level: 12, Life: 50, MaxLife: 100, States: states, Resist: d2.Resists{Poison: 30}, Mode: 1},
		Inventory: &d2.Inventory{Items: []d2.Unit{pot(587, 0), pot(9100, 1), pot(9100, 1),
			{Type: d2.TypeItem, Class: 529, Item: &d2.Item{Where: d2.ContInventory}}}},
		Monsters: []d2.Unit{
			{ID: 10, Class: 267, X: 120, Y: 100, Mode: 1, Monster: &d2.MonsterInfo{Flags: 1 << 3, UniqueNo: 7, LifePct: 80}},
			{ID: 11, Class: 1, X: 105, Y: 100, Mode: 1, Monster: &d2.MonsterInfo{Flags: 1 << 4}},
		},
		Missiles: []d2.Unit{{Class: 217, X: 103, Y: 100}, {Class: 5, X: 1, Y: 1}},
	}
	v := g.viewOf(s)
	if !v.Poisoned || v.PoisonRes != 30 || v.Antidotes != 2 || v.Potions != 1 || !v.HasTP || v.Area != 17 {
		t.Fatalf("view %+v", v)
	}
	if len(v.Dangers) != 1 || v.Dangers[0].Kind != "poison-cloud" || v.Dangers[0].Pos != (boss.Pt{X: 103, Y: 100}) {
		t.Fatalf("dangers %+v", v.Dangers)
	}
	br := v.Monsters[0]
	if br.Name != "Blood Raven" || br.UniqueNo != 7 || br.LifePct != 80 || !br.Hostile {
		t.Fatalf("boss %+v", br)
	}
	if !v.Monsters[1].Minion || v.Monsters[1].UniqueNo != -1 {
		t.Fatalf("minion %+v", v.Monsters[1])
	}
	if !(boss.Match{Class: 267, UniqueNo: -1, Name: "Blood Raven"}).Is(br) {
		t.Fatal("script match must find the boss")
	}
}

func TestMapKillErrAndBossErr(t *testing.T) {
	parent := context.Background()
	slice, cancel := context.WithCancel(parent)
	cancel()
	if err := mapKillErr(parent, slice, context.Canceled); err != nil {
		t.Errorf("the slice running out is not an error: %v", err)
	}
	if err := mapKillErr(parent, parent, combat.ErrTownPortal); !errors.Is(err, boss.ErrEscaped) {
		t.Errorf("portal: %v", err)
	}
	if err := mapKillErr(parent, parent, combat.ErrChicken); !errors.Is(err, boss.ErrChicken) {
		t.Errorf("chicken: %v", err)
	}
	if err := mapKillErr(parent, parent, combat.ErrDead); !errors.Is(err, boss.ErrDead) {
		t.Errorf("dead: %v", err)
	}
	dead, dcancel := context.WithCancel(parent)
	dcancel()
	if err := mapKillErr(dead, slice, context.Canceled); !errors.Is(err, context.Canceled) {
		t.Errorf("parent cancel must surface: %v", err)
	}
	if err := mapKillErr(parent, parent, combat.ErrTargetGone); !errors.Is(err, combat.ErrTargetGone) {
		t.Errorf("target gone stays an error: %v", err)
	}
	cases := []struct {
		in   error
		want func(error) bool
	}{
		{boss.ErrEscaped, func(e error) bool { return errors.Is(e, bot.ErrPortalOpened) }},
		{boss.ErrChicken, func(e error) bool { return errors.Is(e, bot.ErrLeftGame) }},
		{boss.ErrTooWeak, func(e error) bool { return task.Classify(e) == task.KindDanger }},
		{boss.ErrAborted, func(e error) bool { return task.Classify(e) == task.KindDanger }},
		{boss.ErrBossNotFound, func(e error) bool { return task.Classify(e) == task.KindNotFound }},
		{boss.ErrBossLost, func(e error) bool { return task.Classify(e) == task.KindNotFound }},
		{boss.ErrTimeout, func(e error) bool { return task.Classify(e) == task.KindTimeout }},
		{boss.ErrDead, func(e error) bool { return task.Classify(e) == task.KindDeath }},
		{nil, func(e error) bool { return e == nil }},
	}
	for _, c := range cases {
		if got := mapBossErr(c.in); !c.want(got) {
			t.Errorf("%v mapped to %v", c.in, got)
		}
	}
}

func TestAct45RunnerKnowsEveryCampaignBoss(t *testing.T) {
	r := newBossRunner(&bossGame{})
	for _, n := range []string{task.BossIzual, task.BossHephasto, task.BossDiablo, task.BossShenk, task.BossNihlathak, task.BossAncients, task.BossBaal, task.BossMephisto, task.BossCouncil} {
		if r.Scripts[n] == nil && n != task.BossAncients {
			t.Errorf("no script for %q", n)
		}
	}
	if r.Ops == nil {
		t.Error("the runner has no ObjectOps: Diablo, the Ancients and Baal would abort")
	}
}
