package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

const (
	idCorpseExp, idRevive = 74, 95 // skills.txt ids (S: from memory; the test only needs them distinct)
	clsRevived            = 1
)

func reviveWorld(t *testing.T, skills map[int]int, keys map[int]string) (*harness, *nworld) {
	h, w := newNecro(t)
	c := necroCat()
	for _, s := range []struct {
		id   int
		name string
	}{{idCorpseExp, "Corpse Explosion"}, {idRevive, "Revive"}} {
		c.Skills[s.id] = &data.Skill{ID: s.id, Key: s.name, Name: s.name, Mana: 10, ManaShift: 8, MinMana: 10}
	}
	c.Monsters[2] = &data.Monster{Key: "big", Killable: true, Enabled: true, Level: [3]int{30, 30, 30}}
	c.Monsters[1].Level = [3]int{5, 5, 5}
	h.f.SetCatalog(c)
	p := w.snap.Player
	p.Skills = nil
	for id, lv := range skills {
		p.Skills = append(p.Skills, d2.Skill{ID: id, Level: lv})
	}
	p.RightSkill = 0
	h.f.SetHotkeys(keys)
	w.cost = 10
	return h, w
}

func TestCorpsesHardening(t *testing.T) {
	s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, Class: d2.Necromancer}}
	a := mon(10, 1, 20, 20)
	a.Mode = d2.MonModeDead
	zero := mon(11, 1, 0, 0)
	zero.Mode = d2.MonModeDead
	s.Monsters = []d2.Unit{a, a, zero}
	if cs := Corpses(s, necroCat(), input.Point{X: 18, Y: 20}, 14, nil); len(cs) != 1 {
		t.Fatalf("duplicate or unread corpse listed: %v", cs)
	}
	if Corpses(nil, nil, input.Point{X: 0, Y: 0}, 5, nil) != nil {
		t.Fatal("nil snapshot")
	}
}

func TestReviveTakesStrongestCorpse(t *testing.T) {
	h, w := reviveWorld(t, map[int]int{idRevive: 3}, map[int]string{idRevive: "f6"})
	w.spawn = map[string]int{"f6": clsRevived}
	w.corpse(20, 102, 100)
	big := mon(21, 2, 103, 100)
	big.Mode = d2.MonModeDead
	w.snap.Monsters = append(w.snap.Monsters, big)
	n, err := h.f.KeepSummons(context.Background())
	if err != nil || n == 0 || len(w.castAt) == 0 {
		t.Fatalf("revive: %d %v %v", n, err, w.casts)
	}
	if w.castAt[0].X != 103 {
		t.Fatalf("revived the weak corpse: %v", w.castAt[0])
	}
	if !h.f.necro.tried[21] {
		t.Fatal("corpse not blacklisted")
	}
}

func TestReviveCapByEvidence(t *testing.T) {
	h, w := reviveWorld(t, map[int]int{idRevive: 5}, map[int]string{idRevive: "f6"})
	w.spawn = nil // paid casts raise nothing: the cap is 0 minions of the kind
	for i := 0; i < 4; i++ {
		w.corpse(uint32(20+i), 101+i, 100)
	}
	n, _ := h.f.KeepSummons(context.Background())
	if n != 1 || h.f.necro.capSeen["revive"] != 0 {
		t.Fatalf("cap not recorded after a paid empty cast: n=%d %v", n, h.f.necro.capSeen)
	}
}

func TestReviveUnknownSkillDoesNothing(t *testing.T) {
	h, w := reviveWorld(t, map[int]int{idRaise: 0}, nil)
	w.corpse(20, 102, 100)
	if n, _ := h.f.KeepSummons(context.Background()); n != 0 || len(w.casts) != 0 {
		t.Fatalf("cast without a learned skill: %v", w.casts)
	}
}

func TestPickExplosionCorpse(t *testing.T) {
	cat := necroCat()
	mk := func(id uint32, x int) Target { return Target{Unit: mon(id, 1, x, 100), LifePct: 100} }
	c1, c2 := mon(30, 1, 110, 100), mon(31, 1, 130, 100)
	c1.Mode, c2.Mode = d2.MonModeDead, d2.MonModeDead
	hs := []Target{mk(1, 112), mk(2, 113), mk(3, 131)}
	if c, ok := pickExplosionCorpse([]d2.Unit{c1, c2}, hs, cat, 2); !ok || c.ID != 30 {
		t.Fatalf("pack corpse: %v %v", c.ID, ok)
	}
	if _, ok := pickExplosionCorpse([]d2.Unit{c2}, hs, cat, 2); ok {
		t.Fatal("one hostile is below minHit")
	}
	boss := mk(4, 131)
	boss.Boss = true
	if c, ok := pickExplosionCorpse([]d2.Unit{c2}, []Target{boss}, cat, 2); !ok || c.ID != 31 {
		t.Fatal("a boss counts as three")
	}
}

func TestCorpseExplosionTick(t *testing.T) {
	h, w := reviveWorld(t, map[int]int{idCorpseExp: 5}, map[int]string{idCorpseExp: "f7"})
	w.corpse(20, 110, 100)
	w.snap.Monsters = append(w.snap.Monsters, mon(2, 1, 112, 100), mon(3, 1, 113, 101))
	ctx := context.Background()
	if acted, err := h.f.necroTick(ctx, w.snap); !acted || err != nil || len(w.casts) != 1 || w.casts[0] != "f7" || w.castAt[0].X != 110 {
		t.Fatalf("explosion: %v %v %v", acted, err, w.castAt)
	}
	if acted, _ := h.f.necroTick(ctx, w.snap); acted {
		t.Fatalf("second cast on a used corpse: %v", w.casts)
	}
}
