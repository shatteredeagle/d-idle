package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

func TestCountAssassinPets(t *testing.T) {
	s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, Class: d2.Assassin}}
	for i, c := range []int{410, 412, 413, 417, 1} {
		u := mon(uint32(10+i), c, 5*i, 0)
		if c != 1 {
			u.Monster.States = []uint8{d2.StateMinion}
		}
		s.Monsters = append(s.Monsters, u)
	}
	a := CountAssassinPets(s, classCat())
	if len(a.Traps) != 3 || a.Shadows != 1 {
		t.Fatalf("%+v", a)
	}
}

func TestPickTrapTarget(t *testing.T) {
	mk := func(id uint32, x int) Target {
		return Target{Unit: mon(id, 1, x, 100), Name: "f", Dist: float64(x - 100), LifePct: 100}
	}
	a, b, far, near := mk(2, 108), mk(3, 109), mk(4, 140), mk(5, 101)
	if tg, ok := pickTrapTarget([]Target{far, a, b}, nil, data.Fire, 12); !ok || (tg.Unit.ID != 2 && tg.Unit.ID != 3) {
		t.Fatalf("pack %v", tg.Unit.ID)
	}
	if _, ok := pickTrapTarget([]Target{far, near}, nil, data.Fire, 12); ok {
		t.Fatal("out of range or on our feet")
	}
	trap := mon(50, 410, 108, 101)
	if _, ok := pickTrapTarget([]Target{a, b}, []d2.Unit{trap}, data.Fire, 12); ok {
		t.Fatal("pack already burns")
	}
	imm := a
	imm.HasRes = true
	imm.Res[data.Fire] = 100
	if tg, ok := pickTrapTarget([]Target{imm, b}, nil, data.Fire, 12); !ok || tg.Unit.ID != 3 {
		t.Fatal("fire immune target chosen")
	}
	boss := mk(6, 111)
	boss.Boss = true
	if tg, _ := pickTrapTarget([]Target{a, b, boss}, nil, data.Fire, 12); tg.Unit.ID != 6 {
		t.Fatal("boss first")
	}
}

func TestAssassinTrapTick(t *testing.T) {
	h, w := newClassWorld(t, d2.Assassin, map[int]int{idWake: 3, idFireTrauma: 5, idShadow: 1},
		map[int]string{idWake: "f2", idFireTrauma: "f1", idShadow: "f5"}, map[string]int{"f5": classShadowWarrior})
	w.snap.Monsters = []d2.Unit{mon(2, 1, 108, 100), mon(3, 1, 109, 101)}
	ctx := context.Background()
	if acted, err := h.f.classTick(ctx, w.snap); !acted || err != nil || len(w.casts) != 1 || w.casts[0] != "f2" {
		t.Fatalf("no trap: %v %v", w.casts, err)
	}
	// the trap has not appeared yet but the cast is remembered: no second cast on the same pack
	if acted, _ := h.f.classTick(ctx, w.snap); acted || len(w.casts) != 1 {
		t.Fatalf("double trap: %v", w.casts)
	}
	// once the trap stands the census covers the pack
	trap := mon(60, 410, 108, 101)
	trap.Monster.States = []uint8{d2.StateMinion}
	w.snap.Monsters = append(w.snap.Monsters, trap)
	h.s.Sleep(5e9)
	if acted, _ := h.f.classTick(ctx, w.snap); acted {
		t.Fatalf("trap on a covered pack: %v", w.casts)
	}
}

func TestAssassinNeedsMana(t *testing.T) {
	h, w := newClassWorld(t, d2.Assassin, map[int]int{idWake: 3}, map[int]string{idWake: "f2"}, nil)
	w.snap.Player.Mana = 5
	w.snap.Monsters = []d2.Unit{mon(2, 1, 108, 100)}
	if acted, _ := h.f.classTick(context.Background(), w.snap); acted {
		t.Fatal("cast without mana")
	}
}

func TestKeepShadow(t *testing.T) {
	h, w := newClassWorld(t, d2.Assassin, map[int]int{idShadow: 1}, map[int]string{idShadow: "f5"}, map[string]int{"f5": classShadowWarrior})
	ctx := context.Background()
	if n, err := h.f.KeepShadow(ctx); n != 1 || err != nil || len(w.casts) != 1 {
		t.Fatalf("shadow: %d %v %v", n, err, w.casts)
	}
	if n, _ := h.f.KeepShadow(ctx); n != 0 || len(w.casts) != 1 {
		t.Fatalf("recast while it stands: %v", w.casts)
	}
}
