package combat

import (
	"context"
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

const (
	idAmplify, idBoneArmor, idRaise, idClay, idWeaken, idLowerRes = 66, 68, 70, 75, 72, 91
	clsSkel, clsGolem                                             = 363, 289
)

// nworld is a one-snapshot world: casts are recorded and may spawn minions, walking moves the player.
type nworld struct {
	*sim
	snap     *d2.Snapshot
	casts    []string
	castAt   []input.Point
	spawn    map[string]int // skill key -> class spawned on cast (0: nothing)
	cost     int
	curseSt  int // state a curse cast adds to the unit at the click (0: none)
	nextID   uint32
	walkedTo []input.Point
}

func necroCat() MapCatalog {
	sk := func(id int, key string, mana int) *data.Skill {
		return &data.Skill{ID: id, Key: key, Name: key, Mana: mana, ManaShift: 8, MinMana: mana}
	}
	fallen := &data.Monster{Key: "fallen1", Name: "fallen1", Killable: true, Enabled: true, Melee: true}
	boss := &data.Monster{Key: "andariel", Name: "andariel", Killable: true, Enabled: true, Boss: true}
	return MapCatalog{
		Monsters: map[int]*data.Monster{1: fallen, 6: boss},
		Skills: map[int]*data.Skill{idAmplify: sk(idAmplify, "Amplify Damage", 4), idBoneArmor: sk(idBoneArmor, "Bone Armor", 12),
			idRaise: sk(idRaise, "Raise Skeleton", 9), idClay: sk(idClay, "Clay Golem", 15), idWeaken: sk(idWeaken, "Weaken", 4),
			idLowerRes: sk(idLowerRes, "Lower Resist", 10)},
	}
}

func newNecro(t *testing.T) (*harness, *nworld) {
	h := newHarness(t)
	h.f.SetCatalog(necroCat())
	w := &nworld{sim: h.s, spawn: map[string]int{"f5": clsSkel, "f8": clsGolem}, nextID: 500}
	w.snap = &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, Class: d2.Necromancer, X: 100, Y: 100, Area: 2, Level: 20,
		Life: 100, MaxLife: 100, Mana: 60, MaxMana: 60, Mode: 1, RightSkill: idRaise,
		Skills: []d2.Skill{{ID: idRaise, Level: 3}, {ID: idClay, Level: 1}, {ID: idAmplify, Level: 1}, {ID: idBoneArmor, Level: 2},
			{ID: idWeaken, Level: 1}, {ID: idLowerRes, Level: 1}}}, Inventory: &d2.Inventory{}}
	h.f.SetHotkeys(map[int]string{idRaise: "f5", idClay: "f8", idAmplify: "f4", idBoneArmor: "f3", idWeaken: "f6", idLowerRes: "f7"})
	h.f.e.In = w
	h.f.mv = w
	h.f.snapFn = func() (*d2.Snapshot, error) { c := *w.snap; p := *w.snap.Player; c.Player = &p; return &c, nil }
	return h, w
}

func (w *nworld) CastAt(key string, at input.Point) error {
	w.casts, w.castAt = append(w.casts, key), append(w.castAt, at)
	w.snap.Player.Mana -= w.cost
	if cls := w.spawn[key]; cls != 0 {
		w.nextID++
		w.snap.Monsters = append(w.snap.Monsters, d2.Unit{ID: w.nextID, Type: d2.TypeMonster, Class: cls, Mode: d2.MonModeStand,
			X: w.snap.Player.X, Y: w.snap.Player.Y, Monster: &d2.MonsterInfo{LifePct: 100, States: []uint8{d2.StateMinion}}})
	}
	if w.curseSt != 0 {
		for i := range w.snap.Monsters {
			u := &w.snap.Monsters[i]
			if u.X == at.X && u.Y == at.Y && u.Mode == d2.MonModeStand {
				u.Monster.States = append(u.Monster.States, uint8(w.curseSt))
			}
		}
	}
	return nil
}
func (w *nworld) GoTo(_ context.Context, p input.Point) error {
	w.walkedTo = append(w.walkedTo, p)
	w.snap.Player.X, w.snap.Player.Y = p.X, p.Y
	return nil
}
func (w *nworld) GoToUnit(context.Context, uint32) error    { return nil }
func (w *nworld) PathTo(input.Point) ([]input.Point, error) { return nil, nil }
func (w *nworld) corpse(id uint32, x, y int) {
	u := mon(id, 1, x, y)
	u.Mode = d2.MonModeDead
	w.snap.Monsters = append(w.snap.Monsters, u)
}

func TestCountMinionsAndCorpses(t *testing.T) {
	cat := necroCat()
	s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{ID: 1, Class: d2.Necromancer}}
	s.Monsters = []d2.Unit{mon(2, clsSkel, 1, 1), mon(3, clsSkel, 1, 1), mon(4, clsGolem, 1, 1), mon(5, 1, 50, 50), mon(6, 271, 1, 1)}
	dead := mon(7, clsSkel, 1, 1)
	dead.Mode = d2.MonModeDead
	s.Monsters = append(s.Monsters, dead)
	if m := CountMinions(s, cat); m.Skeletons != 2 || m.Golems != 1 || m.Total() != 3 {
		t.Fatalf("%+v", m)
	}
	c1, c2, boss, friend := mon(10, 1, 12, 10), mon(11, 1, 20, 10), mon(12, 6, 11, 10), mon(13, clsSkel, 10, 10)
	for _, u := range []*d2.Unit{&c1, &c2, &boss, &friend} {
		u.Mode = d2.MonModeDead
	}
	live := mon(14, 1, 11, 10)
	s.Monsters = []d2.Unit{c2, c1, boss, friend, live}
	cs := Corpses(s, cat, input.Point{X: 10, Y: 10}, 14, map[uint32]bool{})
	if len(cs) != 2 || cs[0].ID != 10 || cs[1].ID != 11 {
		t.Fatalf("corpses %v", cs)
	}
	if cs := Corpses(s, cat, input.Point{X: 10, Y: 10}, 14, map[uint32]bool{10: true}); len(cs) != 1 || cs[0].ID != 11 {
		t.Fatalf("tried not skipped %v", cs)
	}
	if cs := Corpses(s, cat, input.Point{X: 10, Y: 10}, 4, nil); len(cs) != 1 {
		t.Fatalf("range %v", cs)
	}
}

func TestSummonUpkeep(t *testing.T) {
	h, w := newNecro(t)
	w.cost = 9
	w.corpse(20, 100, 102)
	w.corpse(21, 101, 100)
	w.corpse(22, 99, 100)
	w.corpse(23, 140, 100) // out of walking range
	n, err := h.f.KeepSummons(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// golem first (no corpse), then a skeleton per corpse until the wish (level 3) is met
	if w.casts[0] != "f8" || n != 4 || len(w.walkedTo) != 0 {
		t.Fatalf("casts %v n %d walked %v", w.casts, n, w.walkedTo)
	}
	if m := CountMinions(w.snap, h.f.cat); m.Golems != 1 || m.Skeletons != 3 {
		t.Fatalf("%+v", m)
	}
	if w.castAt[1] == (input.Point{X: 100, Y: 100}) {
		t.Fatalf("skeleton aimed at self: %v", w.castAt)
	}
	// a second pass with nothing missing casts nothing
	if n, _ := h.f.KeepSummons(context.Background()); n != 0 {
		t.Fatalf("recast with full army: %d", n)
	}
	// one skeleton dies: the count, not a timer, triggers the recast (corpse 23 is now within walking range)
	for i := range w.snap.Monsters {
		if w.snap.Monsters[i].Class == clsSkel {
			w.snap.Monsters[i].Mode = d2.MonModeDead
			break
		}
	}
	w.snap.Player.X, w.snap.Player.Y = 130, 100
	w.snap.Player.Mana = 60
	n, _ = h.f.KeepSummons(context.Background())
	if n != 1 || len(w.walkedTo) != 1 || w.walkedTo[0] != (input.Point{X: 140, Y: 100}) {
		t.Fatalf("n %d walked %v", n, w.walkedTo)
	}
}

func TestSummonCapByEvidence(t *testing.T) {
	h, w := newNecro(t)
	w.cost = 9
	w.spawn = map[string]int{"f8": clsGolem} // skeleton casts pay mana and raise nothing: the cap
	for i := uint32(0); i < 3; i++ {
		w.corpse(20+i, 100, 101)
	}
	n, _ := h.f.KeepSummons(context.Background())
	if n != 2 || h.f.necro.capSeen["skeleton"] != 0 {
		t.Fatalf("n %d cap %v casts %v", n, h.f.necro.capSeen, w.casts) // golem + one probe
	}
	if n, _ := h.f.KeepSummons(context.Background()); n != 0 {
		t.Fatalf("probed again: %d", n)
	}
	// a cast that spends no mana (consumed corpse) is not a cap
	h2, w2 := newNecro(t)
	w2.spawn, w2.cost = map[string]int{}, 0
	w2.snap.Player.Skills = w2.snap.Player.Skills[:1]
	w2.corpse(30, 100, 101)
	h2.f.KeepSummons(context.Background())
	if len(h2.f.necro.capSeen) != 0 || !h2.f.necro.tried[30] {
		t.Fatalf("cap %v tried %v", h2.f.necro.capSeen, h2.f.necro.tried)
	}
}

func TestSummonNotInTownOrWhenNotNecro(t *testing.T) {
	h, w := newNecro(t)
	w.corpse(20, 100, 101)
	w.snap.Player.Area = 1
	if n, _ := h.f.KeepSummons(context.Background()); n != 0 {
		t.Fatal("cast in town")
	}
	w.snap.Player.Area, w.snap.Player.Class = 2, d2.Sorceress
	if n, _ := h.f.KeepSummons(context.Background()); n != 0 {
		t.Fatal("sorceress summoned")
	}
}

func TestCurseTargetAndChoice(t *testing.T) {
	mk := func(id uint32, x, y int) Target {
		return Target{Unit: mon(id, 1, x, y), Name: "f", Dist: float64(x - 100), LifePct: 100}
	}
	lone := mk(1, 110, 100)
	if _, ok := pickCurseTarget([]Target{lone}); ok {
		t.Fatal("cursed a lone trash monster")
	}
	a, b, c := mk(2, 112, 100), mk(3, 113, 101), mk(4, 130, 100)
	if tg, ok := pickCurseTarget([]Target{c, a, b}); !ok || tg.Unit.ID != 2 {
		t.Fatalf("pack target %v", tg.Unit.ID)
	}
	boss := mk(5, 118, 100)
	boss.Boss = true
	if tg, ok := pickCurseTarget([]Target{a, b, boss}); !ok || tg.Unit.ID != 5 {
		t.Fatal("boss first")
	}
	far := mk(6, 140, 100)
	far.Boss = true
	if _, ok := pickCurseTarget([]Target{far}); ok {
		t.Fatal("out of range")
	}

	h, w := newNecro(t)
	known := KnownSkills(w.snap.Player, h.f.cat, h.f.keys)
	p := w.snap.Player
	if sp, ok := curseFor(known, p, a); !ok || sp.Name != "Amplify Damage" {
		t.Fatalf("default %v", sp.Name)
	}
	a.HasRes = true
	a.Res[data.Magic] = 75
	if sp, _ := curseFor(known, p, a); sp.Name != "Lower Resist" {
		t.Fatalf("magic resistant: %v", sp.Name)
	}
	delete(known, "Amplify Damage")
	a.HasRes = false
	if _, ok := curseFor(known, p, a); ok {
		t.Fatal("weaken on trash")
	}
	if sp, ok := curseFor(known, p, boss); !ok || sp.Name != "Weaken" {
		t.Fatal("weaken on boss")
	}
	p.Mana = 5
	if _, ok := curseFor(known, p, boss); ok {
		t.Fatal("cursed without mana to spare")
	}
}

func TestCurseOncePerPackByState(t *testing.T) {
	h, w := newNecro(t)
	w.cost, w.curseSt = 4, stateAmplify
	w.snap.Monsters = []d2.Unit{mon(2, 1, 112, 100), mon(3, 1, 113, 101)}
	ctx := context.Background()
	hs := func() []Target { return Hostiles(w.snap, h.f.hostileCtx(w.snap.Player)) }
	if done, err := h.f.Curse(ctx, w.snap, hs()); !done || err != nil || len(w.casts) != 1 {
		t.Fatalf("first %v %v", done, err)
	}
	if w.castAt[0] != (input.Point{X: 112, Y: 100}) && w.castAt[0] != (input.Point{X: 113, Y: 101}) {
		t.Fatalf("not at a monster: %v", w.castAt)
	}
	if !h.f.necro.learned[stateAmplify] {
		t.Fatal("state not learned")
	}
	h.f.Curse(ctx, w.snap, hs())
	h.f.Curse(ctx, w.snap, hs())
	if len(w.casts) != 1 {
		t.Fatalf("recast while the state is up: %v", w.casts)
	}
	// the curse wears off: the state vanishes and the next tick recasts without waiting for a timer
	for i := range w.snap.Monsters {
		w.snap.Monsters[i].Monster.States = nil
	}
	h.s.Sleep(3e9)
	if done, _ := h.f.Curse(ctx, w.snap, hs()); !done || len(w.casts) != 2 {
		t.Fatalf("no recast: %v", w.casts)
	}
}

func TestCurseGivesUpWhenNothingRegisters(t *testing.T) {
	h, w := newNecro(t)
	w.snap.Monsters = []d2.Unit{mon(2, 1, 112, 100), mon(3, 1, 113, 101)}
	for i := 0; i < 10; i++ {
		h.s.Sleep(40e9)
		h.f.Curse(context.Background(), w.snap, Hostiles(w.snap, h.f.hostileCtx(w.snap.Player)))
	}
	if len(w.casts) != 3 {
		t.Fatalf("%d casts at an uncursable target", len(w.casts))
	}
}

func TestBuffDueByState(t *testing.T) {
	h, w := newNecro(t)
	p := w.snap.Player
	sp := KnownSkills(p, h.f.cat, h.f.keys)["Bone Armor"]
	if !h.f.buffDue(p, sp) {
		t.Fatal("never cast: due")
	}
	h.f.buffAt[sp.ID] = h.f.now()
	if h.f.buffDue(p, sp) {
		t.Fatal("unproven state: the timer rules")
	}
	p.States[stateBoneArmor/32] |= 1 << (stateBoneArmor % 32)
	if h.f.buffDue(p, sp) || !h.f.necro.learned[stateBoneArmor] {
		t.Fatal("state present: not due, and learned")
	}
	p.States = d2.StateSet{}
	if h.f.buffDue(p, sp) {
		t.Fatal("grace after the cast")
	}
	h.s.Sleep(5e9)
	if !h.f.buffDue(p, sp) {
		t.Fatal("armor gone (absorb used up): recast long before the timer")
	}
}

func TestBuffSkipsCurses(t *testing.T) {
	h, w := newNecro(t)
	w.snap.Player.RightSkill = idAmplify
	h.f.Buff(context.Background())
	for _, k := range w.casts {
		if k == "f4" || k == "f6" {
			t.Fatalf("curse cast at own feet: %v", w.casts)
		}
	}
}

func TestArmyStands(t *testing.T) {
	h, w := newNecro(t)
	if h.f.armyStands(w.snap) {
		t.Fatal("no minions: no army")
	}
	for i := 0; i < armyMin; i++ {
		w.nextID++
		w.snap.Monsters = append(w.snap.Monsters, d2.Unit{ID: w.nextID, Type: d2.TypeMonster, Class: clsSkel, Mode: d2.MonModeStand,
			Monster: &d2.MonsterInfo{LifePct: 100, States: []uint8{d2.StateMinion}}})
	}
	if !h.f.armyStands(w.snap) {
		t.Fatal("army of armyMin minions should skip the mana rest")
	}
}

func TestCurseCoversNeighbours(t *testing.T) {
	h, w := newNecro(t)
	f := h.f
	f.necro.init()
	sp := SkillSpec{ID: idAmplify, Name: "Amplify Damage"}
	f.necro.curseAt[curseKey{7, idAmplify}] = curseRec{at: f.now(), tries: 1, x: 50, y: 50}
	near := d2.Unit{ID: 8, X: 53, Y: 52, Monster: &d2.MonsterInfo{}}
	far := d2.Unit{ID: 9, X: 70, Y: 50, Monster: &d2.MonsterInfo{}}
	_ = w
	if !f.curseUp(near, sp) {
		t.Fatal("neighbour of a fresh cast should count as cursed")
	}
	if f.curseUp(far, sp) {
		t.Fatal("distant unit should not")
	}
}
