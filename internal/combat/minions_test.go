package combat

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// withSummonKeys adds the Skills.txt summon column and the monstats keys of the summoned classes the census matches on
// (S: stock table names as docs/friendly.md lists them; test data only).
func withSummonKeys(c MapCatalog) MapCatalog {
	for id, s := range map[int]struct {
		key string
		cls int
	}{idRaise: {"necroskeleton", clsSkel}, idClay: {"claygolem", clsGolem}, idOak: {"oaksage", 423}, idRaven: {"raven1", 419},
		idWolf: {"spiritwolf", 420}, idFenris: {"fenris", 421}, idGrizzly: {"grizzly", 428}, idShadow: {"shadowwarrior", classShadowWarrior}} {
		if sk := c.Skills[id]; sk != nil {
			sk.Summon = s.key
		}
		c.Monsters[s.cls] = &data.Monster{ID: s.cls, Key: s.key, Enabled: true}
	}
	return c
}

// armyRun plays the summon scenario of TestSummonUpkeep through the after-clear hook at one knob mode and returns
// the world and the fighter.
func armyRun(t *testing.T, mode int) (*harness, *nworld) {
	h, w := newNecro(t)
	h.f.SetCatalog(withSummonKeys(necroCat()))
	h.f.e.Tune.Combat.DeliveryUpkeep = mode
	w.cost = 9
	w.corpse(20, 100, 102)
	w.corpse(21, 101, 100)
	w.corpse(22, 99, 100)
	w.corpse(23, 140, 100)
	ctx := context.Background()
	h.f.classUpkeep(ctx)
	h.f.classUpkeep(ctx) // nothing missing: no cast
	for i := range w.snap.Monsters {
		if w.snap.Monsters[i].Class == clsSkel {
			w.snap.Monsters[i].Mode = d2.MonModeDead
			break
		}
	}
	w.snap.Player.X, w.snap.Player.Y, w.snap.Player.Mana = 130, 100, 60
	h.f.classUpkeep(ctx) // walk to corpse 23 and raise
	// fight ticks: a skeleton falls, a fresh corpse lies at the feet, the tick raises it
	for round := 0; round < 10; round++ {
		for i := range w.snap.Monsters {
			if u := &w.snap.Monsters[i]; u.Class == clsSkel && u.Mode != d2.MonModeDead {
				u.Mode = d2.MonModeDead
				break
			}
		}
		w.corpse(uint32(40+round), w.snap.Player.X+1, w.snap.Player.Y)
		w.snap.Player.Mana = 60
		if _, err := h.f.classTick(ctx, w.snap); err != nil {
			t.Fatal(err)
		}
		if m := CountMinions(w.snap, h.f.cat); m.Skeletons < 3 || m.Golems != 1 {
			t.Fatalf("mode %d round %d: army fell to %+v", mode, round, m)
		}
	}
	return h, w
}

// Scenario combat.army-holds (Necromancer), offline: the driver (2) and the shadow (1) play exactly the old loop (0).
func TestScenarioArmyHolds(t *testing.T) {
	_, old := armyRun(t, 0)
	if len(old.casts) < 14 || old.casts[0] != "f8" {
		t.Fatalf("old loop casts %v", old.casts)
	}
	for _, mode := range []int{1, 2} {
		h, w := armyRun(t, mode)
		if !reflect.DeepEqual(w.casts, old.casts) || !reflect.DeepEqual(w.castAt, old.castAt) || !reflect.DeepEqual(w.walkedTo, old.walkedTo) {
			t.Fatalf("mode %d differs from the old loop:\n%v %v\n%v %v", mode, w.casts, w.walkedTo, old.casts, old.walkedTo)
		}
		if mode == 1 && h.f.UpkeepShadowDiffs() != 0 {
			t.Fatalf("shadow logged %d disagreements on the old loop's own scenario", h.f.UpkeepShadowDiffs())
		}
	}
}

// Scenario combat.pets-hold (Druid): every group fills to its want, the exclusive spirit group holds one spirit and is
// never recast while it stands, the in-fight life floor of the profile holds.
func TestScenarioPetsHold(t *testing.T) {
	h, w := newClassWorld(t, d2.Druid, map[int]int{idOak: 1, idRaven: 3, idWolf: 2},
		map[int]string{idOak: "f5", idRaven: "f6", idWolf: "f7"}, map[string]int{"f5": 423, "f6": 419, "f7": 420})
	h.f.SetCatalog(withSummonKeys(classCat()))
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	w.cost = 7
	ctx := context.Background()
	h.f.classUpkeep(ctx)
	if want := []string{"f5", "f7", "f7", "f6", "f6", "f6"}; !reflect.DeepEqual(w.casts, want) {
		t.Fatalf("casts %v want %v", w.casts, want)
	}
	for i := 0; i < 5; i++ {
		w.snap.Player.Mana = 200
		h.f.classUpkeep(ctx)
		if acted, _ := h.f.classTick(ctx, w.snap); acted {
			t.Fatalf("recast with every group full: %v", w.casts)
		}
	}
	// a wolf dies: in a fight at low life nothing, above the floor one wolf
	for i := range w.snap.Monsters {
		if w.snap.Monsters[i].Class == 420 {
			w.snap.Monsters[i].Mode = d2.MonModeDead
			break
		}
	}
	w.snap.Player.Life = 30
	if acted, _ := h.f.classTick(ctx, w.snap); acted {
		t.Fatal("summoned under the profile's life floor")
	}
	w.snap.Player.Life = 100
	if acted, _ := h.f.classTick(ctx, w.snap); !acted || w.casts[len(w.casts)-1] != "f7" {
		t.Fatalf("wolf not replaced: %v", w.casts)
	}
	spirits := 0
	for _, k := range w.casts {
		if k == "f5" {
			spirits++
		}
	}
	if spirits != 1 {
		t.Fatalf("spirit cast %d times", spirits)
	}
}

// The shadow agrees with the old Druid loop where the old rules hold one spirit, and reports the exclusive spirit
// group where they did not (old: the n-th spirit skill wanted while fewer than n spirits stand).
func TestMinionShadowDruid(t *testing.T) {
	h, w := newClassWorld(t, d2.Druid, map[int]int{idOak: 1, idRaven: 3}, map[int]string{idOak: "f5", idRaven: "f6"},
		map[string]int{"f5": 423, "f6": 419})
	h.f.SetCatalog(withSummonKeys(classCat()))
	h.f.e.Tune.Combat.DeliveryUpkeep = 1
	w.cost = 7
	ctx := context.Background()
	h.f.classUpkeep(ctx)
	for i := 0; i < 3; i++ {
		h.f.classTick(ctx, w.snap)
	}
	if h.f.UpkeepShadowDiffs() != 0 || len(w.casts) != 4 {
		t.Fatalf("diffs %d casts %v", h.f.UpkeepShadowDiffs(), w.casts)
	}
}

// Two builds with no minions block: the driver never casts (Paladin, Sorceress), even with a summon skill bound.
func TestMinionDriverIdleWithoutGroups(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		code  string
		own   string
		id    int
	}{{d2.Paladin, "pal", "Holy Shield", 117}, {d2.Sorceress, "sor", "Frozen Armor", 40}} {
		class := c.class
		h, w := newClassWorld(t, class, map[int]int{idRaven: 5, c.id: 1}, map[int]string{idRaven: "f6", c.id: "f3"}, map[string]int{"f6": 419})
		cat := withSummonKeys(classCat())
		cat.Skills[c.id] = &data.Skill{ID: c.id, Key: c.own, Name: c.own, Class: c.code, Mana: 10, ManaShift: 8, MinMana: 10}
		h.f.SetCatalog(cat)
		if pr := h.f.profileFor(w.snap.Player); pr.Class != strings.ToLower(class.String()) {
			t.Fatalf("%v: profile %s", class, pr.Class)
		}
		h.f.e.Tune.Combat.DeliveryUpkeep = 2
		ctx := context.Background()
		h.f.classUpkeep(ctx)
		if acted, err := h.f.classTick(ctx, w.snap); acted || err != nil || len(w.casts) != 0 {
			t.Fatalf("%v: casts %v err %v", class, w.casts, err)
		}
		if mp := h.f.profileFor(w.snap.Player).minionParams(); mp != nil {
			t.Fatalf("%v: minion groups %+v", class, mp)
		}
	}
}

// The Assassin's shadow group is afterFight and exclusive: the pass after a clear casts one, the fight tick none.
func TestMinionDriverShadow(t *testing.T) {
	h, w := newClassWorld(t, d2.Assassin, map[int]int{idShadow: 1}, map[int]string{idShadow: "f5"}, map[string]int{"f5": classShadowWarrior})
	h.f.SetCatalog(withSummonKeys(classCat()))
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	ctx := context.Background()
	if pl, ok := h.f.planMinion(w.snap, h.f.profileFor(w.snap.Player), true, false); ok {
		t.Fatalf("shadow planned in a fight: %+v", pl)
	}
	h.f.classUpkeep(ctx)
	h.f.classUpkeep(ctx)
	if !reflect.DeepEqual(w.casts, []string{"f5"}) {
		t.Fatalf("casts %v", w.casts)
	}
}

func TestMinionCensusFromEngineRead(t *testing.T) {
	h, w := newNecro(t)
	h.f.SetCatalog(withSummonKeys(necroCat()))
	h.f.e.Tune.Combat.MinionRead = 1
	h.f.MinionRead = func() ([]MinionRow, error) {
		return []MinionRow{{Skill: idRaise}, {Skill: idRaise}, {Skill: idClay}, {Skill: 999}}, nil
	}
	groups := h.f.profileFor(w.snap.Player).minionParams().Groups
	got := h.f.minionCensus(w.snap, groups)
	if got["skeleton"] != 2 || got["golem"] != 1 || censusTotal(got) != 3 {
		t.Fatalf("census %v", got)
	}
}

func TestMinionCapByEvidence(t *testing.T) {
	h, w := newNecro(t)
	h.f.SetCatalog(withSummonKeys(necroCat()))
	h.f.e.Tune.Combat.DeliveryUpkeep = 2
	w.cost = 9
	w.spawn = map[string]int{"f8": clsGolem}
	for i := uint32(0); i < 3; i++ {
		w.corpse(20+i, 100, 101)
	}
	if n, _ := h.f.keepMinions(context.Background()); n != 2 || h.f.necro.capSeen["skeleton"] != 0 {
		t.Fatalf("n %d cap %v casts %v", n, h.f.necro.capSeen, w.casts)
	}
	if n, _ := h.f.keepMinions(context.Background()); n != 0 {
		t.Fatalf("probed again: %d", n)
	}
}

func TestValidateMinions(t *testing.T) {
	ok := MinionGroup{Name: "wolf", Skills: []string{"x"}, Max: 2, Source: "U guess"}
	for _, c := range []struct {
		g    []MinionGroup
		life int
		want string
	}{
		{[]MinionGroup{ok}, 0, ""},
		{[]MinionGroup{ok, ok}, 0, "repeated"},
		{[]MinionGroup{{Name: "a"}}, 0, "no skills"},
		{[]MinionGroup{{Name: "a", Skills: []string{"x"}, Max: 2}}, 0, "source"},
		{[]MinionGroup{{Name: "a", Skills: []string{"x"}, Exclusive: true}}, 0, ""},
		{[]MinionGroup{ok}, 100, "lifePct"},
	} {
		err := validateMinions([]Profile{{Class: "druid", Build: "t", Minions: &MinionParams{LifePct: c.life, Groups: c.g}}})
		if (c.want == "") != (err == nil) || (err != nil && !strings.Contains(err.Error(), c.want)) {
			t.Fatalf("%+v: %v", c, err)
		}
	}
}
