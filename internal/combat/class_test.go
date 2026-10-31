package combat

import (
	"context"
	"os"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/input"
)

// ---- data model ----

func TestClassDataLoads(t *testing.T) {
	db := loadClassData()
	if len(db.skills) < 100 {
		t.Fatalf("skills.json has %d skills", len(db.skills))
	}
	seen := map[string]bool{}
	for _, c := range []d2.Class{d2.Amazon, d2.Sorceress, d2.Necromancer, d2.Paladin, d2.Barbarian, d2.Druid, d2.Assassin} {
		p := ProfileFor(c, BuildBlizzard)
		if p.Class != classKey(c) {
			t.Fatalf("%v has no profile of its own: %s", c, p.Class)
		}
		seen[p.Class] = true
		last := 1 << 20
		for _, b := range p.Bands {
			if b.MinLevel > last {
				t.Errorf("%s: bands must be ordered by descending minLevel", p.Class)
			}
			last = b.MinLevel
			if len(b.Attack) == 0 {
				t.Errorf("%s: empty band", p.Class)
			}
			for _, n := range b.Attack {
				if _, ok := db.skills[n]; !ok {
					t.Errorf("%s: attack skill %q has no skills.json entry", p.Class, n)
				}
			}
		}
		if p.Bands[len(p.Bands)-1].MinLevel != 0 || p.Bands[len(p.Bands)-1].Requires != "" {
			t.Errorf("%s: the last band must always apply", p.Class)
		}
		for n := range p.Layout {
			if n == "Teleport" {
				continue
			}
			if _, ok := db.skills[n]; !ok {
				t.Errorf("%s: layout skill %q has no entry", p.Class, n)
			}
		}
		for _, a := range p.Aura {
			if m, ok := db.skills[a.Name]; !ok || m.Kind != KindAura {
				t.Errorf("%s: %q is not an aura entry", p.Class, a.Name)
			}
		}
	}
	if len(seen) != 7 {
		t.Fatalf("profiles for %v", seen)
	}
}

func TestSorceressProfileKeepsOldBehaviour(t *testing.T) {
	has := func(...string) func(string) bool { return func(string) bool { return true } }
	if o := attackOrder(BuildBlizzard, 30, has()); o[0] != "Blizzard" || len(o) != 6 {
		t.Fatalf("%v", o)
	}
	if o := attackOrder(BuildLightning, 25, has()); o[0] != "Charged Bolt" {
		t.Fatalf("%v", o)
	}
	if l := Layout(BuildBlizzard); l["Teleport"] != "f2" || l["Blizzard"] != "f5" {
		t.Fatalf("%v", l)
	}
	if b := buffOrder(BuildLightning); len(b) != 2 || b[1][0] != "Energy Shield" {
		t.Fatalf("%v", b)
	}
}

func palCatalog() MapCatalog {
	c := testCatalog()
	sk := func(id int, key, class, rng string, mana, lvl, shift int, aura, left bool, row, col int) *data.Skill {
		return &data.Skill{ID: id, Key: key, Name: key, Class: class, Range: rng, Mana: mana, LvlMana: lvl, ManaShift: shift,
			MinMana: 1, Aura: aura, LeftSkill: left, Row: row, Column: col, ReqLevel: 6, ReqSkills: [3]string{"Sacrifice"}}
	}
	defer func() { c.Skills[0].MinMana = 0 }()
	for _, s := range []*data.Skill{
		sk(0, "Attack", "", "h2h", 0, 0, 0, false, true, 0, 0),
		sk(106, "Zeal", "pal", "h2h", 2, 0, 8, false, true, 3, 1),
		sk(107, "Sacrifice", "pal", "h2h", 3, 0, 8, false, true, 1, 1),
		sk(112, "Smite", "pal", "h2h", 2, 0, 8, false, true, 2, 2),
		sk(114, "Holy Bolt", "pal", "rng", 2, 0, 8, false, true, 2, 3),
		sk(102, "Holy Fire", "pal", "none", 1, 0, 8, true, false, 0, 0),
		sk(98, "Might", "pal", "none", 1, 0, 8, true, false, 0, 0),
	} {
		c.Skills[s.ID] = s
	}
	return c
}

func palPlayer() *d2.Player {
	return &d2.Player{ID: 1, Class: d2.Paladin, X: 100, Y: 100, Area: 2, Level: 13, Life: 150, MaxLife: 150, Mana: 33, MaxMana: 33, Mode: 1,
		Skills:    []d2.Skill{{ID: 0, Level: 1}, {ID: 106, Level: 3}, {ID: 107, Level: 5}, {ID: 112, Level: 3}, {ID: 114, Level: 3}, {ID: 102, Level: 5}, {ID: 98, Level: 1}},
		LeftSkill: 106, RightSkill: 102}
}

func TestSpecFromData(t *testing.T) {
	cat := palCatalog()
	z, ok := makeSpec(cat, "Zeal", 3)
	if !ok || z.Hand != HandLeft || !z.Hold || z.Kind != KindMelee || z.Cost != 2 || z.Range != 7 || z.Prereq[0] != "Sacrifice" || z.ReqLevel != 6 {
		t.Fatalf("%+v", z)
	}
	a, _ := makeSpec(cat, "Might", 1)
	if a.Kind != KindAura || a.Cost != 0 || a.Hand != HandRight {
		t.Fatalf("auras cost nothing: %+v", a)
	}
	if hb, _ := makeSpec(cat, "Holy Bolt", 3); hb.Kind != KindBolt || hb.Hand != HandLeft || hb.Range != 20 {
		t.Fatalf("%+v", hb)
	}
	if _, ok := makeSpec(cat, "No Such Skill", 1); ok {
		t.Fatal("unknown")
	}
	if _, ok := makeSpec(cat, "Teleport", 1); ok {
		t.Fatal("a skill neither table nor profile knows stays unknown (Sorceress behaviour)")
	}
	// derived from skills.txt: a skill a profile names without a table entry
	cat.Skills[777] = &data.Skill{ID: 777, Key: "Holy Shield", Class: "pal", Range: "none", Mana: 3, ManaShift: 8, MinMana: 1}
	if hs, ok := makeSpec(cat, "Holy Shield", 1); !ok || hs.Kind != KindBuff {
		t.Fatalf("%+v %v", hs, ok)
	}
}

func TestSelectProfile(t *testing.T) {
	cat := palCatalog()
	p := palPlayer()
	if pr := SelectProfile(p, cat, BuildBlizzard); pr.Class != "paladin" {
		t.Fatalf("%s", pr.Class)
	}
	p.Class = d2.Amazon // misread class byte: the skills say Paladin
	if pr := SelectProfile(p, cat, BuildBlizzard); pr.Class != "paladin" {
		t.Fatalf("skills must win over the class byte: %s", pr.Class)
	}
	// no skills.txt class info: best matching profile
	for _, s := range cat.Skills {
		s.Class = ""
	}
	if pr := SelectProfile(p, cat, BuildBlizzard); pr.Class != "paladin" {
		t.Fatalf("best match: %s", pr.Class)
	}
	// the Sorceress test snapshots carry class 0 and Fire Bolt only
	sp := baseSnap().Player
	sp.Class = d2.Amazon
	if pr := SelectProfile(sp, testCatalog(), BuildBlizzard); pr.Class != "sorceress" {
		t.Fatalf("%s", pr.Class)
	}
}

func TestPaladinBands(t *testing.T) {
	pr := ProfileFor(d2.Paladin, BuildBlizzard)
	has := func(names ...string) func(string) bool {
		return func(n string) bool {
			for _, x := range names {
				if x == n {
					return true
				}
			}
			return false
		}
	}
	// BUILDV2: Sacrifice is never in the order (it costs life per hit and is only the Zeal prerequisite)
	if o := pr.AttackNames(10, has("Zeal")); o[0] != "Smite" || o[len(o)-1] != "Attack" {
		t.Fatalf("below clvl 12: %v", o)
	}
	if o := pr.AttackNames(13, has("Zeal")); o[0] != "Zeal" || o[len(o)-1] != "Attack" {
		t.Fatalf("%v", o)
	}
	if o := pr.AttackNames(13, has()); o[0] != "Smite" {
		t.Fatalf("Zeal not learned: %v", o)
	}
	if o := pr.AttackNames(30, has("Zeal", "Blessed Hammer")); o[0] != "Blessed Hammer" {
		t.Fatalf("%v", o)
	}
	for _, lv := range []int{1, 10, 13, 30} {
		for _, n := range pr.AttackNames(lv, has("Zeal", "Blessed Hammer", "Sacrifice")) {
			if n == "Sacrifice" {
				t.Errorf("clvl %d: Sacrifice in the attack order", lv)
			}
		}
	}
	if o := ProfileFor(d2.Barbarian, BuildBlizzard).AttackNames(5, has("Bash", "Double Swing")); o[0] != "Bash" {
		t.Errorf("barbarian opener %v, want Bash first", o)
	}
	if a := pr.AuraLadder(13, has()); a[1].Name != "Holy Fire" || a[1].Elem != "fire" || a[0].Name != "Prayer" || a[0].RestLifePct != 70 {
		t.Fatalf("%v", a)
	}
	if a := pr.AuraLadder(30, has("Blessed Hammer")); a[1].Name != "Concentration" {
		t.Fatalf("%v", a)
	}
	if !pr.MeleeClass() || ProfileFor(d2.Sorceress, BuildBlizzard).MeleeClass() {
		t.Fatal("melee flag")
	}
}

// ---- the hand binder with a fake input ----

// handGame fakes the left and right skill popups of a Paladin: the left button opens a grid at the verified
// geometry (Attack row at the bottom, combat row above in tree order), clicking an icon picks it and closes the popup;
// an F key selects the bound right skill; right clicks on the ground "cast" the selected right skill.
type handGame struct {
	*sim
	p         *d2.Player
	open      bool
	icons     map[input.Screen]int
	keys      map[string]int
	aura      int
	clicks    []input.Screen
	rightHits int
	holds     int
}

func newHandGame(t *testing.T) (*harness, *handGame) {
	h := newHarness(t)
	cat := palCatalog()
	h.f.SetCatalog(cat)
	p := palPlayer()
	g := &handGame{sim: h.s, p: p, icons: map[input.Screen]int{}, keys: map[string]int{"f2": 102, "f1": 98}}
	g.p.LeftSkill = 0
	L := LeftPopup
	g.icons[input.Screen{X: L.Col0X, Y: L.Row0Y}] = 0
	for i, id := range []int{107, 112, 114, 106} { // Sacrifice, Smite, Holy Bolt, Zeal
		g.icons[input.Screen{X: L.Col0X + i*L.ColDX, Y: L.Row0Y - L.RowDY}] = id
	}
	h.f.SetListRows(map[int]int{0: 0, 106: 1, 107: 1, 112: 1, 114: 1, 102: 3, 98: 2})
	h.f.e.In = g
	h.f.snapFn = g.snap
	h.f.SetHotkeys(map[int]string{102: "f2", 98: "f1"})
	h.f.auraFn = func(*d2.Player) (int, bool) { return g.aura, true }
	return h, g
}

func (g *handGame) snap() (*d2.Snapshot, error) {
	s, _ := g.sim.snapshot()
	pl := *g.p
	pl.Life, pl.Mana, pl.MaxLife, pl.MaxMana, pl.Mode, pl.Exp = s.Player.Life, s.Player.Mana, s.Player.MaxLife, s.Player.MaxMana, s.Player.Mode, s.Player.Exp
	pl.X, pl.Y = s.Player.X, s.Player.Y
	s.Player = &pl
	return s, nil
}

func (g *handGame) Click(b input.Button, x, y int, m ...input.Mod) error {
	at := input.Screen{X: x, Y: y}
	g.clicks = append(g.clicks, at)
	switch {
	case g.open:
		if id, ok := g.icons[at]; ok {
			g.p.LeftSkill = id
		}
		g.open = false
	case b == input.Left && x == LeftPopup.BtnX && y == LeftPopup.BtnY:
		g.open = true
	case b == input.Right:
		g.rightHits++
		g.aura = g.p.RightSkill
	default:
		return g.sim.Click(b, x, y, m...)
	}
	return nil
}
func (g *handGame) HoldClick(b input.Button, x, y int, d time.Duration) error {
	g.holds++
	g.mu.Lock()
	g.advance(d)
	g.mu.Unlock()
	return g.sim.Click(b, x, y)
}
func (g *handGame) Key(name string) error {
	if id, ok := g.keys[name]; ok {
		g.p.RightSkill = id
		if g.aura != 0 { // an active aura is switched by the key alone
			g.aura = id
		}
		return nil
	}
	return g.sim.Key(name)
}
func (g *handGame) CastAt(key string, w input.Point) error {
	if err := g.Key(key); err != nil {
		return err
	}
	g.rightHits++
	g.aura = g.p.RightSkill
	return nil
}
func (g *handGame) CloseAll() error { g.open = false; return nil }

func TestSetLeftVerifiesAndLeavesRightAlone(t *testing.T) {
	h, g := newHandGame(t)
	g.p.RightSkill = 102
	if err := h.f.SetLeft(context.Background(), 106); err != nil {
		t.Fatal(err)
	}
	if g.p.LeftSkill != 106 || g.p.RightSkill != 102 {
		t.Fatalf("left %d right %d", g.p.LeftSkill, g.p.RightSkill)
	}
	// Zeal is the 4th combat cell: the tree order puts it last (Row 3), the first try hits
	if n := len(g.clicks); n != 2 {
		t.Fatalf("one open + one pick expected, got %v", g.clicks)
	}
	if err := h.f.SetLeft(context.Background(), 106); err != nil || len(g.clicks) != 2 {
		t.Fatal("already selected: no clicks")
	}
	if err := h.f.SetLeft(context.Background(), 0); err != nil || g.p.LeftSkill != 0 {
		t.Fatalf("Attack: %v %d", err, g.p.LeftSkill)
	}
}

func TestSetLeftFindsMisplacedIcon(t *testing.T) {
	h, g := newHandGame(t)
	// the real tree order differs from the prediction: swap Smite and Holy Bolt on screen
	L := LeftPopup
	g.icons[input.Screen{X: L.Col0X + L.ColDX, Y: L.Row0Y - L.RowDY}] = 114
	g.icons[input.Screen{X: L.Col0X + 2*L.ColDX, Y: L.Row0Y - L.RowDY}] = 112
	if err := h.f.SetLeft(context.Background(), 112); err != nil || g.p.LeftSkill != 112 {
		t.Fatalf("%v left %d", err, g.p.LeftSkill)
	}
	if h.f.leftCell[112].X != L.Col0X+2*L.ColDX {
		t.Fatalf("the working cell is remembered: %v", h.f.leftCell)
	}
	g.p.LeftSkill = 0
	n := len(g.clicks)
	if err := h.f.SetLeft(context.Background(), 112); err != nil || len(g.clicks)-n != 2 {
		t.Fatalf("remembered cell goes first: %v", g.clicks[n:])
	}
}

func TestSetLeftFailsWhenSkillAbsent(t *testing.T) {
	h, g := newHandGame(t)
	g.icons = map[input.Screen]int{}
	if err := h.f.SetLeft(context.Background(), 106); err == nil {
		t.Fatal("must report a skill that never reads back")
	}
}

func TestSetRightByHotkey(t *testing.T) {
	h, g := newHandGame(t)
	g.p.RightSkill = 98
	if err := h.f.SetRight(context.Background(), 102); err != nil || g.p.RightSkill != 102 {
		t.Fatalf("%v %d", err, g.p.RightSkill)
	}
	if len(g.clicks) != 0 {
		t.Fatalf("a bound key needs no popup: %v", g.clicks)
	}
}

// ---- aura management ----

func TestKeepAuraAppliesAndSwitches(t *testing.T) {
	h, g := newHandGame(t)
	ctx := context.Background()
	s, _ := g.snap()
	g.p.RightSkill = 98
	g.aura = 0
	ok, err := h.f.KeepAura(ctx, s, nil, true)
	if err != nil || !ok || g.aura != 102 || g.rightHits != 1 {
		t.Fatalf("holy fire should be cast: ok=%v err=%v aura=%d hits=%d", ok, err, g.aura, g.rightHits)
	}
	if ok, _ := h.f.KeepAura(ctx, s, nil, true); ok || g.rightHits != 1 {
		t.Fatal("an active aura is left alone")
	}
}

func TestKeepAuraAfterDeathAndAreaChange(t *testing.T) {
	h, g := newHandGame(t)
	ctx := context.Background()
	s, _ := g.snap()
	g.aura = 102
	g.p.RightSkill = 102
	if ok, _ := h.f.KeepAura(ctx, s, nil, false); ok { // first call: area unknown counts as changed, aura is up
		t.Fatal("aura already up")
	}
	g.aura = 0 // respawn: the aura is gone, the selected skill stays
	h.f.auraDirty = true
	if ok, _ := h.f.KeepAura(ctx, s, nil, false); !ok || g.aura != 102 {
		t.Fatalf("respawn must re-apply: aura %d", g.aura)
	}
	g.aura = 0
	s.Player.Area = 3 // new area
	if ok, _ := h.f.KeepAura(ctx, s, nil, false); !ok || g.aura != 102 {
		t.Fatal("area change must re-apply")
	}
}

func TestWantAuraSkipsImmuneElement(t *testing.T) {
	h, _ := newHandGame(t)
	p := palPlayer()
	pr := h.f.profileFor(p)
	fireImmune := Target{HasRes: true, Res: fireImm()}
	if sp, ok := h.f.wantAura(p, pr, nil, false); !ok || sp.Name != "Holy Fire" {
		t.Fatalf("%v", sp)
	}
	sp, ok := h.f.wantAura(p, pr, &fireImmune, false)
	if !ok || sp.Name != "Might" {
		t.Fatalf("vs fire immune: %v %v", sp.Name, ok)
	}
}

// ---- the fight loop with the Paladin profile ----

func TestPaladinFightsWithLeftHand(t *testing.T) {
	h, g := newHandGame(t)
	g.p.LeftSkill = 107 // Sacrifice selected after the char creation: Zeal must be picked by the profile
	g.aura = 0
	h.s.stock(itHP1, 0, 2)
	for i := 0; i < 3; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 106+float64(i)*3, 103, 30, 3, 5)
	}
	h.s.meleeDmg = 12
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kills != 3 || h.s.dead {
		t.Fatalf("result %v dead=%v", res, h.s.dead)
	}
	if g.p.LeftSkill != 106 {
		t.Fatalf("Zeal must be the left skill, got %d", g.p.LeftSkill)
	}
	if g.holds == 0 {
		t.Fatal("Zeal is held, not clicked")
	}
	if g.aura != 102 || g.p.RightSkill != 102 {
		t.Fatalf("aura %d right %d", g.aura, g.p.RightSkill)
	}
	if h.s.casts != 0 {
		t.Fatal("no right-hand casts at monsters")
	}
}

func TestPaladinFallsBackToAttackWithoutMana(t *testing.T) {
	h, g := newHandGame(t)
	g.p.LeftSkill = 106
	g.aura = 102
	h.s.mana, h.s.manaRegen = 0, 0
	h.s.addMon(10, clsFallen, 106, 103, 30, 1, 5)
	h.s.meleeDmg = 15
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 1 {
		t.Fatalf("%v %v", res, err)
	}
	if g.p.LeftSkill != 0 {
		t.Fatalf("with no mana the free basic attack must be selected, got %d", g.p.LeftSkill)
	}
}

func TestMeleeGuardBacksOutWhenSurroundedAndHurt(t *testing.T) {
	h, g := newHandGame(t)
	g.p.LeftSkill = 106
	g.aura = 102
	h.s.life = 50 // 33 percent of 150? sim life is out of 100: use percent below 65
	for i := 0; i < 5; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 101+float64(i%3), 101+float64(i/3), 400, 0, 0)
	}
	snap, _ := g.snap()
	moved, err := h.f.meleeGuard(context.Background(), snap, ProfileFor(d2.Paladin, BuildBlizzard).Melee, newFightState())
	if err != nil || !moved || len(h.s.moves) != 1 {
		t.Fatalf("moved=%v err=%v moves=%v", moved, err, h.s.moves)
	}
}

func TestRealSkillNames(t *testing.T) {
	dir := os.Getenv("D2LOD_DATA_CACHE")
	if dir == "" {
		t.Skip("set D2LOD_DATA_CACHE to a data cache dir to check profile skill names against skills.txt")
	}
	db, err := data.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	for n := range loadClassData().skills {
		if db.SkillByName(n) == nil {
			t.Errorf("skills.json: %q is not in skills.txt", n)
		}
	}
	for _, p := range Profiles() {
		var names []string
		for _, b := range p.Bands {
			names = append(names, b.Attack...)
			for _, a := range b.Aura {
				names = append(names, a.Name)
			}
		}
		for _, a := range p.Aura {
			names = append(names, a.Name)
		}
		for _, g := range p.Buffs {
			names = append(names, g...)
		}
		for n := range p.Layout {
			names = append(names, n)
		}
		for _, n := range names {
			s := db.SkillByName(n)
			if s == nil {
				t.Errorf("%s: %q is not in skills.txt", p.Class, n)
				continue
			}
			if code := map[string]string{"sorceress": "sor", "paladin": "pal", "amazon": "ama", "necromancer": "nec", "barbarian": "bar", "druid": "dru", "assassin": "ass"}[p.Class]; s.Class != code && n != "Attack" {
				t.Errorf("%s: %q belongs to class %q", p.Class, n, s.Class)
			}
		}
	}
}

func fireImm() (r [6]int) { r[data.Fire] = 100; return }

func TestPaladinDoesNotRestForMana(t *testing.T) {
	h, g := newHandGame(t)
	g.p.LeftSkill = 106
	g.aura = 102
	h.s.mana, h.s.manaRegen = 1, 0.1
	h.s.addMon(10, clsFallen, 106, 103, 30, 1, 5)
	h.s.meleeDmg = 15
	t0 := h.s.Now()
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 1 {
		t.Fatalf("%v %v", res, err)
	}
	if el := h.s.Now().Sub(t0); el > 15*time.Second {
		t.Fatalf("a melee profile must not rest for a cast budget: %v", el)
	}
}

// Prayer (id 99) is a resting aura: only while resting below 70 % life; a fight keeps Might or Holy Fire.
func TestWantAuraPrayerWhileResting(t *testing.T) {
	h, _ := newHandGame(t)
	cat := palCatalog()
	cat.Skills[99] = &data.Skill{ID: 99, Key: "Prayer", Name: "Prayer", Class: "pal", Range: "none", Aura: true, ReqLevel: 6}
	h.f.SetCatalog(cat)
	h.f.SetHotkeys(map[int]string{102: "f2", 98: "f1", 99: "f8"})
	p := palPlayer()
	p.Level = 5
	p.Skills = append(p.Skills, d2.Skill{ID: 99, Level: 1})
	pr := h.f.profileFor(p)
	p.Life, p.MaxLife = 60, 100
	if sp, ok := h.f.wantAura(p, pr, nil, true); !ok || sp.Name != "Prayer" {
		t.Fatalf("resting at 60%%: %v", sp.Name)
	}
	if sp, ok := h.f.wantAura(p, pr, &Target{}, false); !ok || sp.Name == "Prayer" {
		t.Fatalf("a fight keeps its aura: %v", sp.Name)
	}
	p.Life = 80
	if sp, ok := h.f.wantAura(p, pr, nil, true); !ok || sp.Name == "Prayer" {
		t.Fatalf("resting at 80%%: %v", sp.Name)
	}
	p.Skills = []d2.Skill{{ID: 0, Level: 1}, {ID: 99, Level: 1}}
	if sp, ok := h.f.wantAura(p, pr, &Target{}, false); !ok || sp.Name != "Prayer" {
		t.Fatalf("only Prayer known: a fight uses it as the last resort: %v", sp.Name)
	}
}

func TestRestAuraSwitchesUnlessAdjacent(t *testing.T) {
	h, g := newHandGame(t)
	cat := palCatalog()
	cat.Skills[99] = &data.Skill{ID: 99, Key: "Prayer", Name: "Prayer", Class: "pal", Range: "none", Aura: true, ReqLevel: 6}
	h.f.SetCatalog(cat)
	h.f.SetHotkeys(map[int]string{102: "f2", 98: "f1", 99: "f8"})
	g.keys["f8"] = 99
	g.p.Skills = append(g.p.Skills, d2.Skill{ID: 99, Level: 1})
	g.aura = 102
	s, _ := g.snap()
	s.Player.Life, s.Player.MaxLife = 50, 100
	adj := []Target{{Dist: 2}}
	h.f.restAura(context.Background(), s, adj)
	if g.aura != 102 {
		t.Fatalf("hostile adjacent: aura stays, got %d", g.aura)
	}
	h.f.restAura(context.Background(), s, []Target{{Dist: 12}})
	if g.aura != 99 {
		t.Fatalf("resting at 50%%: Prayer, got %d", g.aura)
	}
}
