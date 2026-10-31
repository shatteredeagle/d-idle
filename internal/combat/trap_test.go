package combat

import (
	"context"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

const idDeathSent = 276 // skills.txt id (S: from memory; only needs to be distinct here)

func trapWorld(t *testing.T, skills map[int]int) (*harness, *nworld) {
	keys := map[int]string{idWake: "f2", idLtSent: "f6", idDeathSent: "f7"}
	h, w := newClassWorld(t, d2.Assassin, skills, keys, nil)
	c := classCat()
	c.Skills[idDeathSent] = &data.Skill{ID: idDeathSent, Key: "Death Sentry", Name: "Death Sentry", Mana: 20, ManaShift: 8, MinMana: 20, EType: "fire"}
	h.f.SetCatalog(c)
	return h, w
}

func trapUnit(id uint32, class, x, y int) d2.Unit {
	u := mon(id, class, x, y)
	u.Monster.States = []uint8{d2.StateMinion}
	return u
}

func TestTrapCapFivePerSkill(t *testing.T) {
	h, w := trapWorld(t, map[int]int{idWake: 10, idLtSent: 5})
	h.f.trap.init()
	h.f.trap.classSkill[410], h.f.trap.classSkill[411] = "Wake of Fire Sentry", "Lightning Sentry"
	w.snap.Monsters = []d2.Unit{mon(2, 1, 108, 100), mon(3, 1, 109, 101)}
	for i := 0; i < 5; i++ { // five wake traps far from the pack
		w.snap.Monsters = append(w.snap.Monsters, trapUnit(uint32(60+i), 410, 80, 90+i*6))
	}
	acted, err := h.f.assassinTick(context.Background(), w.snap)
	if !acted || err != nil || len(w.casts) != 1 || w.casts[0] != "f6" {
		t.Fatalf("wake at its cap must give way to lightning: %v %v", w.casts, err)
	}
	// both at the cap: no cast at all
	for i := 0; i < 5; i++ {
		w.snap.Monsters = append(w.snap.Monsters, trapUnit(uint32(70+i), 411, 80, 40+i*6))
	}
	h.s.Sleep(5e9)
	h.f.trap.laid = nil
	if acted, _ := h.f.assassinTick(context.Background(), w.snap); acted || len(w.casts) != 1 {
		t.Fatalf("sixth trap laid: %v", w.casts)
	}
}

func TestTrapExpiringDoesNotCount(t *testing.T) {
	h, w := trapWorld(t, map[int]int{idWake: 10})
	h.f.trap.init()
	h.f.trap.classSkill[410] = "Wake of Fire Sentry"
	w.snap.Monsters = []d2.Unit{mon(2, 1, 108, 100), mon(3, 1, 109, 101)}
	for i := 0; i < 5; i++ {
		w.snap.Monsters = append(w.snap.Monsters, trapUnit(uint32(60+i), 410, 80, 90+i*6))
	}
	ctx := context.Background()
	h.f.assassinTick(ctx, w.snap) // first sighting: age 0
	if len(w.casts) != 0 {
		t.Fatalf("cap ignored: %v", w.casts)
	}
	h.s.Sleep(time.Duration(trapLifeSec-1) * time.Second)
	if acted, _ := h.f.assassinTick(ctx, w.snap); !acted {
		t.Fatal("traps about to expire still block the cap")
	}
}

func TestTrapLaidAtPackCentre(t *testing.T) {
	me := input.Point{X: 100, Y: 100}
	mk := func(id uint32, x, y int) Target {
		return Target{Unit: mon(id, 1, x, y), Name: "f", Dist: dist(100, 100, x, y), LifePct: 100}
	}
	hs := []Target{mk(2, 108, 100), mk(3, 112, 100), mk(4, 110, 104)}
	if c := packCentre(me, hs[0], hs, 15); c.X != 110 || c.Y != 101 {
		t.Fatalf("centre %v", c)
	}
	if c := packCentre(me, hs[0], hs, 9); c.X != 108 {
		t.Fatalf("centre out of range must fall back to the target: %v", c)
	}
}

func TestDeathSentryNeedsCorpseAtPack(t *testing.T) {
	h, w := trapWorld(t, map[int]int{idDeathSent: 10})
	w.snap.Monsters = []d2.Unit{mon(2, 1, 112, 100), mon(3, 1, 113, 101)}
	ctx := context.Background()
	if acted, _ := h.f.assassinTick(ctx, w.snap); acted {
		t.Fatalf("Death Sentry without a corpse: %v", w.casts)
	}
	w.corpse(20, 110, 100)
	acted, err := h.f.assassinTick(ctx, w.snap)
	if !acted || err != nil || len(w.casts) != 1 || w.casts[0] != "f7" || w.castAt[0].X != 110 {
		t.Fatalf("Death Sentry on the corpse: %v %v %v", acted, err, w.castAt)
	}
}

func TestTrapClassLearnedAndLifetimeFallback(t *testing.T) {
	h, w := trapWorld(t, map[int]int{idLtSent: 5})
	w.snap.Monsters = []d2.Unit{mon(2, 1, 108, 100), mon(3, 1, 109, 101)}
	ctx := context.Background()
	h.f.assassinTick(ctx, w.snap)
	if len(w.casts) != 1 {
		t.Fatal("no cast")
	}
	// the trap shows up next to the cast point: its class is learned for the skill
	at := w.castAt[0]
	w.snap.Monsters = append(w.snap.Monsters, trapUnit(60, 412, at.X, at.Y))
	h.s.Sleep(5e9)
	h.f.assassinTick(ctx, w.snap)
	if h.f.trap.classSkill[412] != "Lightning Sentry" || !h.f.trap.visible || len(h.f.trap.laid) != 0 {
		t.Fatalf("learned %v visible=%v laid=%v", h.f.trap.classSkill, h.f.trap.visible, h.f.trap.laid)
	}
	// census never shows traps: casts count for their lifetime, so the cap still holds without ids
	h2, w2 := trapWorld(t, map[int]int{idLtSent: 5})
	spots := [][2]int{{108, 100}, {92, 100}, {100, 108}, {100, 92}, {108, 108}, {92, 92}, {108, 92}, {92, 108}}
	for _, sp := range spots {
		w2.snap.Monsters = []d2.Unit{mon(2, 1, sp[0], sp[1]), mon(3, 1, sp[0]+1, sp[1]+1)}
		h2.f.assassinTick(ctx, w2.snap)
		h2.s.Sleep(2e9)
	}
	if len(w2.casts) != maxTraps {
		t.Fatalf("blind cap: %d casts", len(w2.casts))
	}
}

func TestOtherClassesNoTrapTick(t *testing.T) {
	h, w := newClassWorld(t, d2.Sorceress, map[int]int{idLtSent: 5}, map[int]string{idLtSent: "f6"}, nil)
	w.snap.Monsters = []d2.Unit{mon(2, 1, 108, 100), mon(3, 1, 109, 101)}
	if acted, _ := h.f.assassinTick(context.Background(), w.snap); acted || len(w.casts) != 0 {
		t.Fatalf("non-assassin laid a trap: %v", w.casts)
	}
	if acted, _ := h.f.necroTick(context.Background(), w.snap); acted {
		t.Fatal("non-necro ran the necro tick")
	}
}
