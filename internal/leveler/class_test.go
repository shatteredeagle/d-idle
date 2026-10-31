package leveler

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
	"d2lod-bot/internal/town/hand"
)

func quietEnv() *env.Env {
	return &env.Env{Log: slog.New(slog.NewTextHandler(io.Discard, nil)), Clock: &fakeClock{t: time.Unix(0, 0)}}
}

type fakeClock struct{ t time.Time }

func (c *fakeClock) Now() time.Time        { return c.t }
func (c *fakeClock) Sleep(d time.Duration) { c.t = c.t.Add(d) }

func TestPaladinBuildLoads(t *testing.T) {
	b := mustBuild(t, "paladin-normal30")
	if b.Class != "paladin" {
		t.Fatalf("class %q", b.Class)
	}
	if len(b.Rows) != 29 || b.Rows[0].Clvl != 2 || b.Rows[len(b.Rows)-1].Clvl != 30 {
		t.Fatalf("rows %d", len(b.Rows))
	}
	if len(b.Hotkeys) == 0 || len(b.RespecPool) == 0 || b.Respec != 19 {
		t.Fatalf("hotkeys %d pool %d respec %d", len(b.Hotkeys), len(b.RespecPool), b.Respec)
	}
	if got := mustBuild(t, "fallback").Class; got != "sorceress" {
		t.Fatalf("old files default to sorceress, got %q", got)
	}
}

func TestPaladinStatPlanIsNotAllVitality(t *testing.T) {
	b := mustBuild(t, "paladin-normal30")
	str := 0
	for _, r := range b.Rows {
		str += r.Attr[Str]
	}
	if str < 25 {
		t.Fatalf("only %d strength points to clvl 30", str)
	}
	h := Have{Attr: [4]int{25, 20, 25, 15}, Start: [4]int{25, 20, 25, 15}, Skills: map[string]int{}, Level: 6, StatPts: 25, SkillPts: 5}
	var s [4]int
	for _, a := range b.Plan(h, nil, nil) {
		if a.Kind == KindStat {
			s[a.Attr]++
		}
	}
	if s[Str] == 0 || s[Vit] == 0 || s[Dex] == 0 || s[Vit] == 25 {
		t.Fatalf("stat split %v", s)
	}
}

func TestPaladinSkillOrder(t *testing.T) {
	b := mustBuild(t, "paladin-normal30")
	h := Have{Attr: [4]int{25, 20, 25, 15}, Start: [4]int{25, 20, 25, 15}, Skills: map[string]int{}, Level: 30, SkillPts: 29}
	var order []string
	for _, a := range b.Plan(h, nil, nil) {
		order = append(order, a.Skill)
	}
	want := []string{"Might", "Sacrifice", "Resist Fire", "Resist Fire", "Holy Fire", "Holy Fire", "Holy Fire", "Holy Fire", "Smite", "Zeal", "Charge"}
	for i, w := range want {
		if order[i] != w {
			t.Fatalf("skill %d = %s, want %s (%v)", i, order[i], w, order)
		}
	}
	for _, s := range order {
		switch s {
		case "Charged Bolt", "Static Field", "Telekinesis", "Frozen Armor", "Warmth", "Fire Bolt":
			t.Fatalf("Sorceress skill %s in the Paladin build", s)
		}
	}
}

func TestBuildForClass(t *testing.T) {
	l := New(quietEnv(), Options{})
	cases := []struct {
		class d2.Class
		want  string
		err   error
	}{
		{d2.Sorceress, "fallback", nil},
		{d2.Paladin, "paladin_normal30", nil},
		{d2.Necromancer, "necromancer_summoner_normal30", nil},
		{d2.Barbarian, "barbarian_melee_normal30", nil},
	}
	for _, c := range cases {
		b, err := l.buildFor(&d2.Player{Class: c.class})
		if !errors.Is(err, c.err) {
			t.Fatalf("%v: err %v", c.class, err)
		}
		if err == nil && b.Name == "" {
			t.Fatalf("%v: empty build", c.class)
		}
	}
	// the Sorceress table on a Paladin is refused, never run
	l = New(quietEnv(), Options{Build: "primary"})
	if _, err := l.buildFor(&d2.Player{Class: d2.Paladin}); !errors.Is(err, ErrWrongClass) {
		t.Fatalf("err %v", err)
	}
	if _, err := l.buildFor(&d2.Player{Class: d2.Sorceress}); err != nil {
		t.Fatal(err)
	}
}

func TestDoPausesOnWrongClassBuild(t *testing.T) {
	var reason string
	l := New(quietEnv(), Options{Build: "fallback", Pause: func(r string) { reason = r }})
	snap := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: d2.Paladin, Level: 6, StatPoints: 5, SkillPoints: 1}}
	rep := l.Do(context.Background(), snap)
	if !errors.Is(rep.Err, ErrWrongClass) || reason == "" || len(rep.Done) != 0 {
		t.Fatalf("rep %+v reason %q", rep, reason)
	}
}

func TestCheckLearned(t *testing.T) {
	if err := checkLearned(nil, 98, []int{98}); err != nil {
		t.Fatal(err)
	}
	for _, got := range [][]int{nil, {97}, {98, 103}} {
		if err := checkLearned(nil, 98, got); !errors.Is(err, ErrSkillMismatch) {
			t.Fatalf("%v: %v", got, err)
		}
	}
}

func TestGained(t *testing.T) {
	before := []d2.Skill{{ID: 0}, {ID: 98, Level: 1}}
	after := []d2.Skill{{ID: 0}, {ID: 98, Level: 1}, {ID: 96, Level: 1}}
	if g := gained(before, after); len(g) != 1 || g[0] != 96 {
		t.Fatalf("%v", g)
	}
}

func TestPrefsSorceressUnchanged(t *testing.T) {
	p := PrefsFor("Sorceress")
	if p.DmgWeight != wWeaponDmg || p.TwoHandMult != 1 || p.WantsShield || p.TypeBonus["staf"] != wCasterBase || p.TypeBonus["wand"] != wCasterBase || p.TypeBonus["orb"] != wCasterBase {
		t.Fatalf("%+v", p)
	}
	if q := PrefsFor("Necromancer"); q.DmgWeight != p.DmgWeight {
		t.Fatal("unknown class must keep the old behaviour")
	}
	pal := PrefsFor("Paladin")
	if !pal.WantsShield || pal.TwoHandMult >= 1 || len(pal.TypeBonus) != 0 {
		t.Fatalf("%+v", pal)
	}
}

func TestChooseGearShieldBuildSkipsTwoHandedEvenWithShieldInPack(t *testing.T) {
	two := Cand{Unit: d2.Unit{ID: 5}, Name: "great sword", Score: 30, Usable: true, Slots: []int{d2.BodyRHand}, TwoHanded: true, AvoidTwoHanded: true}
	sword := cand(6, "sword", 12, true, d2.BodyRHand)
	shield := cand(7, "shield", 15, true, d2.BodyLHand)
	sw := ChooseGear(map[int]Cand{}, []Cand{two, sword, shield})
	if len(sw) != 2 || sw[0].Item.ID() != 6 || sw[1].Item.ID() != 7 {
		t.Fatalf("swaps %v", sw)
	}
	// no shield anywhere: the two-hander is allowed
	sw = ChooseGear(map[int]Cand{}, []Cand{two, sword})
	if len(sw) != 1 || sw[0].Item.ID() != 5 {
		t.Fatalf("swaps %v", sw)
	}
}

func TestGearStateOnSwappedSet(t *testing.T) {
	mk := func(body int) d2.Unit {
		return d2.Unit{Item: &d2.Item{Where: d2.ContEquipment, Body: body}}
	}
	inv := &d2.Inventory{Items: []d2.Unit{mk(d2.BodyAltR), mk(d2.BodyAltL)}}
	sets := hand.SetsOf(inv)
	if !sets.OnEmptySet() || sets.Active || !sets.Alt {
		t.Fatalf("%+v", sets)
	}
	if !hand.FilledEither(inv, d2.BodyRHand) || !hand.FilledEither(inv, d2.BodyLHand) || hand.FilledEither(inv, d2.BodyHead) {
		t.Fatal("FilledEither")
	}
	inv = &d2.Inventory{Items: []d2.Unit{mk(d2.BodyRHand), mk(d2.BodyAltL)}}
	if hand.SetsOf(inv).OnEmptySet() {
		t.Fatal("active set holds gear")
	}
}

// fakePanel is a skill tree in which a click on an icon learns the skill id its table says. Real tables needed
// (/tmp/d2mpq, run on the worker).
type fakePanel struct {
	*input.Fake
	db     *data.DB
	geo    Geometry
	page   int
	pts    int
	skills map[int]int
	class  string
	// misroute: planned skill id -> id that really gets the point (the CLASSPAL failure)
	misroute map[int]int
	noop     map[int]bool // clicks that the game refuses (no point spent)
}

func (f *fakePanel) Click(b input.Button, x, y int, mods ...input.Mod) error {
	for p := 1; p <= 3; p++ {
		if (Pt{x, y}) == f.geo.Tab(p) {
			f.page = p
			return nil
		}
	}
	for _, s := range f.db.Skills {
		if s.Class != f.class || s.Page != f.page || s.Page == 0 {
			continue
		}
		if (Pt{x, y}) == f.geo.Icon(s.Row, s.Column) && f.pts > 0 && !f.noop[s.ID] {
			id := s.ID
			if to, ok := f.misroute[id]; ok {
				id = to
			}
			f.skills[id]++
			f.pts--
		}
	}
	return nil
}

func (f *fakePanel) player() (*d2.Player, error) {
	p := &d2.Player{SkillPoints: f.pts, Class: d2.Paladin}
	for id, lv := range f.skills {
		p.Skills = append(p.Skills, d2.Skill{ID: id, Level: lv})
	}
	return p, nil
}

func realDB(t *testing.T) *data.DB {
	t.Helper()
	if _, err := os.Stat("/tmp/d2mpq/d2data.mpq"); err != nil {
		t.Skip("no real install at /tmp/d2mpq (run on the worker)")
	}
	db, err := data.Open("/tmp/d2mpq", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func newFakeRig(t *testing.T) (*Leveler, *fakePanel, *string) {
	db := realDB(t)
	e := quietEnv()
	e.Data = db
	fp := &fakePanel{Fake: input.NewFake(input.Point{}), db: db, geo: DefaultGeometry(), pts: 3, skills: map[int]int{}, class: "pal"}
	e.In = fp
	var reason string
	l := New(e, Options{Pause: func(r string) { reason = r }})
	l.player = fp.player
	return l, fp, &reason
}

func TestSpendSkillReadBack(t *testing.T) {
	l, fp, reason := newFakeRig(t)
	id := fp.db.SkillByName("Might").ID
	ok, err := l.spendSkill(context.Background(), Action{Kind: KindSkill, Skill: "Might"})
	if !ok || err != nil || fp.skills[id] != 1 || *reason != "" {
		t.Fatalf("ok %v err %v skills %v", ok, err, fp.skills)
	}
}

func TestSpendSkillMisrouteFailsLoudly(t *testing.T) {
	l, fp, reason := newFakeRig(t)
	smite, thorns := fp.db.SkillByName("Smite").ID, fp.db.SkillByName("Thorns").ID
	fp.misroute = map[int]int{fp.db.SkillByName("Sacrifice").ID: smite}
	_ = thorns
	ok, err := l.spendSkill(context.Background(), Action{Kind: KindSkill, Skill: "Sacrifice"})
	if ok || !errors.Is(err, ErrSkillMismatch) || *reason == "" {
		t.Fatalf("ok %v err %v reason %q", ok, err, *reason)
	}
}

func TestSpendSkillRefusedIsNotAnError(t *testing.T) {
	l, fp, _ := newFakeRig(t)
	fp.noop = map[int]bool{fp.db.SkillByName("Might").ID: true}
	ok, err := l.spendSkill(context.Background(), Action{Kind: KindSkill, Skill: "Might"})
	if ok || err != nil {
		t.Fatalf("ok %v err %v", ok, err)
	}
}

func TestRuleRefusesOtherClassSkills(t *testing.T) {
	db := realDB(t)
	e := quietEnv()
	e.Data = db
	l := New(e, Options{})
	for _, sk := range []string{"Charged Bolt", "Telekinesis", "Frozen Armor", "Warmth"} {
		if l.rule("pal", sk, 30, map[string]int{}) {
			t.Fatalf("%s allowed for a paladin", sk)
		}
		if sk == "Charged Bolt" && !l.rule("sor", sk, 30, map[string]int{}) {
			t.Fatal("sorceress rule broke")
		}
	}
	if !l.rule("pal", "Might", 2, map[string]int{}) || l.rule("pal", "Zeal", 11, map[string]int{"Sacrifice": 1}) {
		t.Fatal("paladin rule")
	}
	// every skill in the Paladin build exists and belongs to the Paladin
	b := mustBuild(t, "paladin-normal30")
	for _, r := range b.Rows {
		for _, sp := range r.Skills {
			s := db.SkillByName(sp.Skill)
			if s == nil || s.Class != "pal" || s.Page == 0 {
				t.Fatalf("build skill %q: %+v", sp.Skill, s)
			}
		}
	}
	for _, hk := range b.Hotkeys {
		if s := db.SkillByName(hk.Skill); s == nil || s.Class != "pal" {
			t.Fatalf("hotkey skill %q", hk.Skill)
		}
	}
}

func TestFallbackSorceressBuildHasEnergy(t *testing.T) {
	b, err := LoadBuild("fallback", "")
	if err != nil {
		t.Fatal(err)
	}
	ene := 0
	for _, r := range b.Rows {
		ene += r.Attr[Ene]
		if r.Attr[Vit]+r.Attr[Ene] > 0 && r.Attr[Str]+r.Attr[Dex]+r.Attr[Vit]+r.Attr[Ene] != 5 {
			t.Fatalf("row %d does not add up to 5 points: %v", r.Clvl, r.Attr)
		}
	}
	if ene < 30 {
		t.Fatalf("energy points in the table: %d", ene)
	}
}
