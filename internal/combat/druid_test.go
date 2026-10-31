package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// skill ids are skills.txt (V data.json)
const (
	idOak, idRaven, idWolf, idFenris, idGrizzly       = 226, 221, 227, 237, 247
	idFireTrauma, idWake, idQuick, idShadow, idLtSent = 251, 262, 258, 268, 271
)

func classCat() MapCatalog {
	c := necroCat()
	for _, s := range []struct {
		id    int
		name  string
		mana  int
		etype string
	}{{idOak, "Oak Sage", 25, ""}, {idRaven, "Raven", 7, ""}, {idWolf, "Summon Spirit Wolf", 15, ""}, {idFenris, "Summon Fenris", 20, ""},
		{idGrizzly, "Summon Grizzly", 30, ""}, {idFireTrauma, "Fire Trauma", 5, "fire"}, {idWake, "Wake of Fire Sentry", 10, "fire"},
		{idQuick, "Quickness", 10, ""}, {idShadow, "Shadow Warrior", 25, ""}, {idLtSent, "Lightning Sentry", 15, "ltng"}} {
		c.Skills[s.id] = &data.Skill{ID: s.id, Key: s.name, Name: s.name, Mana: s.mana, ManaShift: 8, MinMana: s.mana, EType: s.etype}
	}
	return c
}

// newClassWorld is the necro world with another class, its skills and hotkeys.
func newClassWorld(t *testing.T, class d2.Class, skills map[int]int, keys map[int]string, spawn map[string]int) (*harness, *nworld) {
	h, w := newNecro(t)
	h.f.SetCatalog(classCat())
	p := w.snap.Player
	p.Class, p.Skills, p.Mana, p.MaxMana = class, nil, 200, 200
	for id, lv := range skills {
		p.Skills = append(p.Skills, d2.Skill{ID: id, Level: lv})
	}
	p.RightSkill = 0
	h.f.SetHotkeys(keys)
	w.spawn = spawn
	return h, w
}

func TestCountDruidPets(t *testing.T) {
	cat := classCat()
	s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, Class: d2.Druid}}
	for i, c := range []int{419, 419, 420, 421, 428, 423, 425, 1} {
		u := mon(uint32(10+i), c, 1, 1)
		if c != 1 {
			u.Monster.States = []uint8{d2.StateMinion}
		}
		s.Monsters = append(s.Monsters, u)
	}
	d := CountDruidPets(s, cat)
	if d != (DruidPets{Ravens: 2, SpiritWolves: 1, DireWolves: 1, Grizzlies: 1, Spirits: 1, Vines: 1}) || d.Total() != 7 {
		t.Fatalf("%+v", d)
	}
}

func TestDruidPetOrderAndCap(t *testing.T) {
	h, w := newClassWorld(t, d2.Druid, map[int]int{idOak: 1, idRaven: 3, idWolf: 2},
		map[int]string{idOak: "f5", idRaven: "f6", idWolf: "f7"}, map[string]int{"f5": 423, "f6": 419, "f7": 420})
	w.cost = 7
	ctx := context.Background()
	n, err := h.f.KeepPets(ctx)
	if err != nil {
		t.Fatal(err)
	}
	// Oak Sage once, 2 wolves (skill level 2), 3 ravens (skill level 3): spirits first, then wolves, then ravens
	want := []string{"f5", "f7", "f7", "f6", "f6", "f6"}
	if n != len(want) || len(w.casts) != len(want) {
		t.Fatalf("casts %v (n=%d)", w.casts, n)
	}
	for i := range want {
		if w.casts[i] != want[i] {
			t.Fatalf("order %v", w.casts)
		}
	}
	if n, _ := h.f.KeepPets(ctx); n != 0 {
		t.Fatalf("recast with everything standing: %v", w.casts)
	}
	// a paid cast that raises nothing records the cap: no more casts of that kind until the count falls
	w.snap.Monsters = nil
	w.spawn = map[string]int{"f5": 423}
	h.f.necro.capSeen = map[string]int{"raven": 0}
	h.f.KeepPets(ctx)
	for _, k := range w.casts[6:] {
		if k == "f6" {
			t.Fatalf("raven cast past its cap: %v", w.casts)
		}
	}
}

func TestDruidCapRecordedWhenPaidAndNothingRaised(t *testing.T) {
	h, w := newClassWorld(t, d2.Druid, map[int]int{idRaven: 5}, map[int]string{idRaven: "f6"}, map[string]int{})
	w.cost = 7
	n, _ := h.f.KeepPets(context.Background())
	if n != 1 || h.f.necro.capSeen["raven"] != 0 {
		t.Fatalf("n=%d cap=%v", n, h.f.necro.capSeen)
	}
}

func TestDruidNoPetsInTownOrForOtherClasses(t *testing.T) {
	h, w := newClassWorld(t, d2.Druid, map[int]int{idRaven: 5}, map[int]string{idRaven: "f6"}, map[string]int{"f6": 419})
	w.snap.Player.Area = 1 // Rogue Encampment
	if n, _ := h.f.KeepPets(context.Background()); n != 0 || len(w.casts) != 0 {
		t.Fatalf("cast in town: %v", w.casts)
	}
	h, w = newClassWorld(t, d2.Sorceress, map[int]int{idRaven: 5}, map[int]string{idRaven: "f6"}, map[string]int{"f6": 419})
	if n, _ := h.f.KeepPets(context.Background()); n != 0 {
		t.Fatal("sorceress summoned")
	}
	// low life: no summon in a fight tick
	h, w = newClassWorld(t, d2.Druid, map[int]int{idRaven: 5}, map[int]string{idRaven: "f6"}, map[string]int{"f6": 419})
	w.snap.Player.Life = 20
	if acted, _ := h.f.classTick(context.Background(), w.snap); acted {
		t.Fatal("summoned at 20% life")
	}
	w.snap.Player.Life = 100
	if acted, _ := h.f.classTick(context.Background(), w.snap); !acted || len(w.casts) != 1 {
		t.Fatalf("no summon tick: %v", w.casts)
	}
}

func TestDruidProfile(t *testing.T) {
	cat := classCat()
	p := &d2.Player{Class: d2.Druid, Level: 14, Skills: []d2.Skill{{ID: idOak, Level: 1}}}
	pr := SelectProfile(p, cat, BuildBlizzard)
	if pr == nil || pr.Layout["Eruption"] != "f3" || pr.Layout["Oak Sage"] != "f5" || len(pr.Buffs) != 1 || pr.Buffs[0][0] != "Cyclone Armor" {
		t.Fatalf("%+v", pr)
	}
	ap := SelectProfile(&d2.Player{Class: d2.Assassin, Level: 14, Skills: []d2.Skill{{ID: idWake, Level: 1}}}, cat, BuildBlizzard)
	if ap == nil || ap.Layout["Wake of Fire Sentry"] != "f2" || ap.Buffs[0][0] != "Quickness" {
		t.Fatalf("%+v", ap)
	}
	for _, n := range ap.AttackNames(20, func(string) bool { return true }) {
		if n == "Wake of Fire Sentry" {
			t.Fatal("traps are laid by assassinTick, not picked as an attack")
		}
	}
}
