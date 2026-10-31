package combat

import (
	"testing"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

func hctx() HostileCtx { return HostileCtx{Cat: testCatalog(), Diff: d2.Normal, ClassLvl: 1} }

func TestHostilesFilters(t *testing.T) {
	s := baseSnap()
	dead := mon(4, clsFallen, 104, 100)
	dead.Mode = d2.MonModeDead
	dying := mon(5, clsFallen, 105, 100)
	dying.Mode = d2.MonModeDeath
	merc := mon(6, clsFallen, 101, 100)
	merc.OwnerID, merc.OwnerType = 1, 0
	summon := mon(7, clsFallen, 101, 101)
	summon.OwnerID, summon.OwnerType = 6, 1 // owned by the merc
	s.Monsters = []d2.Unit{
		mon(2, clsFallen, 110, 100), mon(3, clsTownNPC, 102, 100), dead, dying, merc, summon, mon(8, clsFallen, 103, 100),
	}
	got := Hostiles(s, hctx())
	if len(got) != 2 || got[0].Unit.ID != 8 || got[1].Unit.ID != 2 {
		t.Fatalf("want ids [8 2] nearest first, got %+v", ids(got))
	}
	s.Player.Area = 1 // Rogue Encampment
	if h := Hostiles(s, hctx()); len(h) != 0 {
		t.Fatalf("nothing is hostile in town, got %v", ids(h))
	}
	if h := Hostiles(&d2.Snapshot{State: d2.StateLoading}, hctx()); h != nil {
		t.Fatal("loading snapshot must give nothing")
	}
}

func ids(ts []Target) []uint32 {
	var o []uint32
	for _, t := range ts {
		o = append(o, t.Unit.ID)
	}
	return o
}

func TestRankPriorities(t *testing.T) {
	s := baseSnap()
	champ := mon(10, clsFallen, 112, 100)
	champ.Monster.Flags = 1 << 2
	s.Monsters = []d2.Unit{mon(2, clsFallen, 108, 100), mon(3, clsShaman, 120, 100), champ, mon(4, clsFallen, 140, 100)}
	r := Rank(Hostiles(s, hctx()), 100, 0)
	if r[0].Unit.ID != 3 {
		t.Fatalf("shaman first, got %v", ids(r))
	}
	// something adjacent beats a distant elite but not a shaman
	s.Monsters = []d2.Unit{mon(2, clsFallen, 104, 100), champ}
	r = Rank(Hostiles(s, hctx()), 100, 0)
	if r[0].Unit.ID != 2 {
		t.Fatalf("adjacent grunt first, got %v", ids(r))
	}
	// careless pick: nearest even though a shaman exists
	s.Monsters = []d2.Unit{mon(2, clsFallen, 108, 100), mon(3, clsShaman, 120, 100)}
	r = Rank(Hostiles(s, hctx()), 50, 0.99)
	if r[0].Unit.ID != 2 {
		t.Fatalf("careless roll should pick the nearest, got %v", ids(r))
	}
	r = Rank(Hostiles(s, hctx()), 50, 0.1)
	if r[0].Unit.ID != 3 {
		t.Fatalf("lucky roll keeps the shaman first, got %v", ids(r))
	}
}

func TestAffixesAndDanger(t *testing.T) {
	s := baseSnap()
	s.Player.Difficulty = d2.Hell
	m := mon(2, clsFallen, 110, 100)
	m.Monster.Flags = 1 << 2
	m.Monster.Enchants = [9]uint8{17, 9, 0}
	s.Monsters = []d2.Unit{m, mon(3, clsFallen, 112, 100)}
	hc := hctx()
	hc.Diff = d2.Hell
	hs := Hostiles(s, hc)
	var le *Target
	for i := range hs {
		if hs[i].Unit.ID == 2 {
			le = &hs[i]
		}
	}
	if le.Affixes != AffixLightningEnchanted|AffixFireEnchanted || !le.Avoid {
		t.Fatalf("lightning enchanted at clvl 1 in Hell is avoided: %+v", le)
	}
	if got := Rank(hs, 100, 0); len(got) != 1 || got[0].Unit.ID != 3 {
		t.Fatalf("avoided target is dropped, got %v", ids(got))
	}
	hc.ClassLvl = 80
	for _, h := range Hostiles(s, hc) {
		if h.Avoid {
			t.Fatal("clvl 80 may fight it")
		}
	}
	hc.Diff, hc.ClassLvl = d2.Normal, 1
	for _, h := range Hostiles(s, hc) {
		if h.Avoid {
			t.Fatal("Normal never avoids")
		}
	}
}

func TestImmunityFromData(t *testing.T) {
	s := baseSnap()
	s.Monsters = []d2.Unit{mon(2, clsColdImm, 110, 100), mon(3, clsFallen, 120, 100)}
	hs := Hostiles(s, hctx())
	if !hs[0].Immune(data.Cold) || hs[0].Immune(data.Fire) || hs[1].Immune(data.Cold) {
		t.Fatalf("immunity from monstats wrong: %+v", hs[0].Res)
	}
	if !hs[0].ImmuneAll(data.Cold) || hs[0].ImmuneAll(data.Cold, data.Fire) || hs[0].ImmuneAll() {
		t.Fatal("ImmuneAll")
	}
}

func TestResistsFromStats(t *testing.T) {
	mon := testCatalog().Monsters[clsColdImm]
	r := ResistsFromStats(d2.Stats{39: 100, 43: 20}, mon, d2.Normal)
	if r[data.Fire] != 100 || r[data.Cold] != 20 || r[data.Lightning] != 0 {
		t.Fatalf("stat list wins where present, monstats elsewhere: %v", r)
	}
	r = ResistsFromStats(d2.Stats{}, mon, d2.Normal)
	if r[data.Cold] != 100 {
		t.Fatalf("empty stat list falls back to monstats: %v", r)
	}
	r = ResistsFromStats(d2.Stats{43: -100}, mon, d2.Normal)
	if r[data.Cold] != -100 {
		t.Fatalf("negative resist: %v", r)
	}
}

func TestPickAttack(t *testing.T) {
	cat := testCatalog()
	fb, _ := makeSpec(cat, "Fire Bolt", 1)
	cb, _ := makeSpec(cat, "Charged Bolt", 5)
	bz, _ := makeSpec(cat, "Blizzard", 20)
	if fb.Cost != 3 {
		t.Fatalf("fire bolt level 1 cost = %d, want 3 (2.5 rounded up)", fb.Cost)
	}
	if cb.Cost != 5 {
		t.Fatalf("charged bolt level 5 cost = %d, want 5", cb.Cost)
	}
	if bz.Delay.Milliseconds() != 1800 {
		t.Fatalf("blizzard cooldown %v", bz.Delay)
	}
	grunt := &Target{HasRes: true}
	coldImm := &Target{HasRes: true, Res: [6]int{data.Cold: 100}}
	order := []SkillSpec{bz, cb, fb}
	if s, why := PickAttack(order, grunt, 100, nil); why != PickOK || s.Name != "Blizzard" {
		t.Fatalf("main skill first: %v %v", s.Name, why)
	}
	if s, why := PickAttack(order, coldImm, 100, nil); why != PickOK || s.Name != "Charged Bolt" {
		t.Fatalf("cold immune falls back to the lightning skill: %v %v", s.Name, why)
	}
	if s, _ := PickAttack(order, grunt, 5, nil); s.Name != "Charged Bolt" {
		t.Fatalf("low mana falls back to the cheaper skill: %v", s.Name)
	}
	if _, why := PickAttack(order, grunt, 1, nil); why != PickNeedMana {
		t.Fatalf("no mana: %v", why)
	}
	notBlizz := func(s SkillSpec) bool { return s.Name != "Blizzard" }
	if s, _ := PickAttack(order, grunt, 100, notBlizz); s.Name != "Charged Bolt" {
		t.Fatalf("cooling timed skill is skipped: %v", s.Name)
	}
	all := &Target{HasRes: true, Res: [6]int{data.Cold: 100, data.Lightning: 100, data.Fire: 100}}
	if _, why := PickAttack(order, all, 100, nil); why != PickImmune {
		t.Fatalf("immune to all: %v", why)
	}
	if _, why := PickAttack(nil, grunt, 100, nil); why != PickNoSkill {
		t.Fatalf("no skills: %v", why)
	}
	if _, why := PickAttack([]SkillSpec{bz}, grunt, 100, notBlizz); why != PickCooling {
		t.Fatalf("cooling: %v", why)
	}
}

func TestAttackOrder(t *testing.T) {
	has := func(...string) func(string) bool { return func(string) bool { return true } }
	if o := attackOrder(BuildBlizzard, 10, has()); o[0] != "Charged Bolt" {
		t.Fatalf("early: %v", o)
	}
	if o := attackOrder(BuildBlizzard, 30, has()); o[0] != "Blizzard" {
		t.Fatalf("after respec: %v", o)
	}
	if o := attackOrder(BuildBlizzard, 30, func(string) bool { return false }); o[0] != "Charged Bolt" {
		t.Fatalf("no respec yet: %v", o)
	}
	if o := attackOrder(BuildLightning, 30, has()); o[0] != "Chain Lightning" {
		t.Fatalf("fallback build: %v", o)
	}
}

func TestLayoutHasUniqueKeys(t *testing.T) {
	for _, b := range []Build{BuildBlizzard, BuildLightning} {
		seen := map[string]string{}
		for skill, key := range Layout(b) {
			if o, dup := seen[key]; dup {
				t.Fatalf("build %d: %s and %s share %s", b, skill, o, key)
			}
			seen[key] = skill
		}
		if len(seen) > 8 {
			t.Fatal("the game has F1-F8 only")
		}
	}
}

func TestBeltAndDecide(t *testing.T) {
	cat := testCatalog()
	inv := &d2.Inventory{Items: []d2.Unit{potion(itHP1, 0), potion(itHP1, 0), potion(itHP3, 1), potion(itMP1, 2), potion(itRV, 3), potion(9999, 3)}}
	b := BeltOf(inv, cat)
	if b.Count(PotionHealing) != 3 || b.Count(PotionMana) != 1 || b.Count(PotionRejuv) != 1 {
		t.Fatalf("counts: %+v", b)
	}
	if c, _ := b.ColumnFor(PotionHealing, true); c != 1 {
		t.Fatalf("best healing column = %d, want 1 (tier 3)", c)
	}
	if c, _ := b.ColumnFor(PotionHealing, false); c != 0 {
		t.Fatalf("weakest healing column = %d, want 0", c)
	}
	if _, ok := b.ColumnFor(PotionRejuv, true); !ok {
		t.Fatal("rejuv column")
	}
	if BeltOf(nil, cat).Count(PotionHealing) != 0 {
		t.Fatal("nil inventory")
	}

	d := env.DefaultCombatTuning()
	full := Belt{}
	full.Cols[0] = BeltColumn{PotionHealing, 3, 4, false}
	full.Cols[2] = BeltColumn{PotionMana, 3, 4, false}
	full.Cols[3] = BeltColumn{PotionRejuv, 2, 2, false}
	empty := Belt{}
	in := func(life, mana int, b Belt, fight bool) SafetyInput {
		return SafetyInput{Life: life, Max: 100, Mana: mana, MMax: 100, Belt: b, InFight: fight, HasPortal: true,
			HealReady: true, ManaReady: true, RejuvReady: true, Tune: d}
	}
	cases := []struct {
		name string
		in   SafetyInput
		want SafetyAction
	}{
		{"healthy", in(100, 100, full, true), SafetyNone},
		{"scratch", in(60, 100, full, true), SafetyNone},
		{"drink life", in(54, 100, full, true), DrinkLife},
		{"rejuv band", in(34, 100, full, true), DrinkRejuv},
		{"chicken line with rejuv", in(25, 100, full, true), DrinkRejuv},
		{"chicken line no rejuv", in(25, 100, withoutRejuv(full), true), Chicken},
		{"mana", in(100, 24, full, true), DrinkMana},
		{"mana potion missing", in(100, 24, empty, true), SafetyNone},
		{"no potions mid-fight portal", in(50, 100, empty, true), TownPortal},
		{"no potions no portal", func() SafetyInput { x := in(50, 100, empty, true); x.HasPortal = false; return x }(), SafetyNone},
		{"hurt quiet few potions restocks", in(50, 100, fewHP(1), false), TownPortal},
		{"hurt quiet enough potions drinks", in(50, 100, full, false), DrinkLife},
		{"low quiet drinks heal", in(20, 100, fewHP(5), false), DrinkLife},
		{"heal on cooldown", func() SafetyInput { x := in(50, 100, full, true); x.HealReady = false; return x }(), DrinkRejuv},
	}
	for _, c := range cases {
		if got := Decide(c.in); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
	h := in(29, 100, withoutRejuv(full), true)
	h.Diff = d2.Hell
	if Decide(h) != Chicken {
		t.Error("Hell chicken at 30")
	}
	h.Diff = d2.Normal
	if Decide(h) == Chicken {
		t.Error("Normal does not chicken at 29")
	}
	tw := in(10, 10, empty, true)
	tw.InTown = true
	if Decide(tw) != SafetyNone {
		t.Error("town: nothing")
	}
	if Decide(SafetyInput{Tune: d}) != SafetyNone {
		t.Error("zero input")
	}
}

func withoutRejuv(b Belt) Belt { b.Cols[3] = BeltColumn{}; return b }
func fewHP(n int) Belt {
	var b Belt
	b.Cols[0] = BeltColumn{PotionHealing, 1, n, false}
	return b
}

func TestAwayPoint(t *testing.T) {
	me := input.Point{X: 100, Y: 100}
	threat := []input.Point{{X: 95, Y: 100}}
	pt, ok := awayPoint(me, threat, nil, 12, nil)
	if !ok || pt.X <= me.X {
		t.Fatalf("should retreat east: %v %v", pt, ok)
	}
	if pdist(pt, threat[0]) <= pdist(me, threat[0]) {
		t.Fatal("retreat must increase the distance")
	}
	// east is a wall: pick another direction that still gains distance
	pt, ok = awayPoint(me, threat, nil, 12, func(p input.Point) bool { return p.X <= 100 })
	if ok && pdist(pt, threat[0]) <= pdist(me, threat[0]) {
		t.Fatalf("walls respected, still gaining: %v", pt)
	}
	if _, ok := awayPoint(me, threat, nil, 12, func(input.Point) bool { return false }); ok {
		t.Fatal("nowhere to go")
	}
	if _, ok := awayPoint(me, nil, nil, 12, nil); ok {
		t.Fatal("no threat, no move")
	}
	// a hazard near the best cell pushes the choice elsewhere
	best, _ := awayPoint(me, threat, nil, 12, nil)
	alt, _ := awayPoint(me, threat, []input.Point{best}, 12, nil)
	if alt == best {
		t.Fatal("hazard avoidance")
	}
}

func TestApproachPoint(t *testing.T) {
	p := approachPoint(input.Point{X: 100, Y: 100}, input.Point{X: 130, Y: 100}, 10)
	if p.X != 120 || p.Y != 100 {
		t.Fatalf("approach = %v", p)
	}
	if q := approachPoint(input.Point{X: 5, Y: 5}, input.Point{X: 5, Y: 5}, 10); q != (input.Point{X: 5, Y: 5}) {
		t.Fatal("on top of the target")
	}
}

func TestOnScreen(t *testing.T) {
	s := baseSnap()
	// standing player: VX = 16(px-py)-400, VY = 8(px+py)-284 with the +0.5 rest offset
	s.Viewport = d2.Point{X: 16*0 - 400, Y: 8*(100.5+100.5) - 284}
	if !onScreen(s, input.Point{X: 100, Y: 100}) {
		t.Fatal("own tile is on screen")
	}
	if onScreen(s, input.Point{X: 160, Y: 100}) {
		t.Fatal("60 subtiles east is off screen")
	}
}

func TestTuningRanges(t *testing.T) {
	for _, k := range env.Knobs() {
		if k.Min > k.Max || k.Default < k.Min || k.Default > k.Max {
			t.Errorf("knob %s: default %v outside [%v,%v]", k.Key, k.Default, k.Min, k.Max)
		}
	}
	n := 0
	for _, k := range env.Knobs() {
		if len(k.Key) > 7 && k.Key[:7] == "combat." {
			n++
		}
	}
	if n < 25 {
		t.Fatalf("expected the combat knobs, got %d", n)
	}
}

func TestDied(t *testing.T) {
	cat := testCatalog()
	prev := baseSnap()
	prev.Player.Life = 12
	prev.Monsters = []d2.Unit{mon(2, clsFallen, 103, 100), mon(3, clsFallen, 140, 100)}
	cur := baseSnap()
	cur.Player.Life, cur.Player.Mode = 0, 0
	cur.Monsters = prev.Monsters
	di, ok := Died(prev, cur, cat)
	if !ok || di.MonsterID != 2 || di.MonsterName != "fallen1" || di.LastLife != 12 || di.Area != 2 || di.Pos.X != 100 {
		t.Fatalf("%v %+v", ok, di)
	}
	if _, ok := Died(cur, cur, cat); ok {
		t.Fatal("already dead is not a new death")
	}
	if _, ok := Died(prev, prev, cat); ok {
		t.Fatal("alive is not a death")
	}
	if _, ok := Died(nil, &d2.Snapshot{State: d2.StateLoading}, cat); ok {
		t.Fatal("loading is not a death")
	}
	if di, ok := Died(nil, cur, cat); !ok || di.MonsterID != 2 {
		t.Fatalf("first snapshot already dead: %+v", di)
	}
	cur.Player.Mode, cur.Player.Life = 17, 0
	if _, ok := Died(prev, cur, cat); !ok {
		t.Fatal("mode 17")
	}
}

func TestDeathWeights(t *testing.T) {
	s := baseSnap()
	s.Monsters = []d2.Unit{mon(2, clsFallen, 110, 100), mon(3, clsShaman, 112, 100), mon(4, clsFallen, 108, 100)}
	hc := hctx()
	hc.Weights = DeathWeights{Monster: map[string]float64{"fallen1": 1}}
	r := Rank(Hostiles(s, hc), 100, 0)
	if r[0].Unit.ID != 3 { // shaman still first
		t.Fatalf("%v", ids(r))
	}
	hc.Weights = DeathWeights{Monster: map[string]float64{"fallen1": 3}}
	hs := Hostiles(s, hc)
	if got := Rank(hs, 100, 0); len(got) != 1 || got[0].Unit.ID != 3 {
		t.Fatalf("weight 3 avoids the grunts: %v", ids(got))
	}
	m := mon(5, clsFallen, 110, 100)
	m.Monster.Flags = 1 << 3
	m.Monster.Enchants = [9]uint8{25}
	s.Monsters = []d2.Unit{m}
	hc.Weights = DeathWeights{Affix: map[string]float64{"mana-burn": 4}}
	if hs := Hostiles(s, hc); !hs[0].Avoid || hs[0].Danger != 4 {
		t.Fatalf("affix weight: %+v", hs[0])
	}
}

func TestAntidoteClassAndBelt(t *testing.T) {
	if c := classifyCode("yps"); c.Potion != PotionAntidote {
		t.Fatalf("yps = %+v", c)
	}
	if classifyCode("vps").Potion != PotionNone {
		t.Fatal("stamina potion is not an antidote")
	}
	cat := testCatalog()
	cat.Items[9100] = ItemClass{Potion: PotionAntidote, Tier: 1}
	inv := &d2.Inventory{Items: []d2.Unit{potion(itHP1, 0), potion(9100, 1), potion(9100, 1)}}
	b := BeltOf(inv, cat)
	if b.Count(PotionAntidote) != 2 || b.Count(PotionHealing) != 1 {
		t.Fatalf("counts %+v", b)
	}
	if c, ok := b.ColumnFor(PotionAntidote, false); !ok || c != 1 {
		t.Fatalf("antidote column %d %v", c, ok)
	}
	if PotionAntidote.String() != "antidote" {
		t.Fatal("name")
	}
}
