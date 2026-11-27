package points

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// tables is a two-class toy: a Sorceress (sor) and a Paladin (pal), three skills each. Gains are in quarter points, the
// way charstats.txt holds them (V internal/cheat/memory_ops.go).
func tables() Tables {
	return Tables{
		Classes: []*data.CharClass{
			{Name: "Sorceress", Code: "sor", Str: 10, Dex: 25, Vit: 10, Int: 35, LifePerVit: 8, StaminaPerVit: 4, ManaPerEnergy: 8},
			{Name: "Paladin", Code: "pal", Str: 25, Dex: 20, Vit: 25, Int: 15, LifePerVit: 12, StaminaPerVit: 4, ManaPerEnergy: 6},
		},
		Skills: []*data.Skill{
			{ID: 36, Key: "Fire Bolt", Name: "Fire Bolt", Class: "sor", Page: 1, ReqLevel: 1, MaxLevel: 20},
			{ID: 37, Key: "Warmth", Name: "Warmth", Class: "sor", Page: 1, ReqLevel: 1, MaxLevel: 20, Passive: true},
			{ID: 41, Key: "Fire Ball", Name: "Fire Ball", Class: "sor", Page: 1, ReqLevel: 18, MaxLevel: 20, ReqSkills: [3]string{"Fire Bolt"}},
			{ID: 96, Key: "Might", Name: "Might", Class: "pal", Page: 1, ReqLevel: 1, MaxLevel: 20, Aura: true},
			{ID: 97, Key: "Sacrifice", Name: "Sacrifice", Class: "pal", Page: 1, ReqLevel: 1, MaxLevel: 20},
			{ID: 112, Key: "Blessed Hammer", Name: "Blessed Hammer", Class: "pal", Page: 2, ReqLevel: 18, MaxLevel: 20},
		},
	}
}

func legal() Legal {
	return Legal{
		T:        tables(),
		Playable: func(n string) bool { return n != "Blessed Hammer" && n != "Fire Ball" },
		Attack:   func(n string) bool { return n != "Warmth" && n != "Might" },
	}
}

func TestEffectsFromClassTable(t *testing.T) {
	for _, c := range []struct {
		class     string
		life, man float64
	}{{"Sorceress", 2, 2}, {"Paladin", 3, 1.5}} {
		cc := tables().Class(c.class)
		if v := EffectOf(cc, "vit"); v.Life != c.life || v.Stamina != 1 {
			t.Errorf("%s vit: %+v", c.class, v)
		}
		if e := EffectOf(cc, "ene"); e.Mana != c.man {
			t.Errorf("%s ene: %+v", c.class, e)
		}
	}
	if p := Preview(Stats[2], EffectOf(tables().Class("Paladin"), "vit")); p != "+1 Vitality: +3 life, +1 stamina" {
		t.Errorf("preview %q", p)
	}
	if p := Preview(Stats[0], EffectOf(nil, "str")); !strings.Contains(p, "worn gear requirements") {
		t.Errorf("str preview %q", p)
	}
}

func TestSheetBankedAndGear(t *testing.T) {
	p := &d2.Player{Class: d2.Paladin, Level: 12, Strength: 30, Dexterity: 22, Vitality: 40, Energy: 15, StatPoints: 5, SkillPoints: 2}
	s := BuildSheet(Input{Hero: "Palli", Player: p, Class: tables().Class("Paladin"), Enabled: true,
		Verbs: Verbs{Stat: true, Skill: true, Via: "d2d verbs"},
		Gear:  []GearReq{{Slot: "torso", Item: "Splint Mail", Str: 51}, {Slot: "hand", Item: "Sword", Str: 25, Dex: 25}}})
	if s.Banked != 7 || s.Waiting != "5 stat points and 2 skill points waiting for you" {
		t.Fatalf("banked %d waiting %q", s.Banked, s.Waiting)
	}
	str := s.Stats[0]
	if str.NeedsGear != 51 || str.Base != 30 || str.Spent != 5 || !str.CanAdd {
		t.Errorf("str row %+v", str)
	}
	if s.Gear[0].ShortStr != 21 || s.Gear[1].ShortDex != 3 {
		t.Errorf("gear shortages %+v", s.Gear)
	}
	if !strings.Contains(s.Note, "cannot split") {
		t.Errorf("note %q", s.Note)
	}
	base := [4]int{28, 22, 30, 15}
	s = BuildSheet(Input{Player: p, Class: tables().Class("Paladin"), Base: &base, Verbs: Verbs{Stat: true}})
	if s.Stats[2].Spent != 5 || s.Note != "" {
		t.Errorf("with a base split: %+v note %q", s.Stats[2], s.Note)
	}
	p.StatPoints = 0
	if s = BuildSheet(Input{Player: p, Verbs: Verbs{Stat: true}}); s.Stats[0].CanAdd || s.Stats[0].Why == "" || s.Note == "" {
		t.Errorf("no points: %+v", s.Stats[0])
	}
}

func TestSheetWithoutHero(t *testing.T) {
	if s := BuildSheet(Input{}); s.Note == "" || len(s.Stats) != 0 {
		t.Errorf("%+v", s)
	}
}

func TestCheckSkillTwoClasses(t *testing.T) {
	l := legal()
	mk := func(code, name string, lvl, unspent int, pts map[int]int) d2.Player {
		p := d2.Player{Class: map[string]d2.Class{"sor": d2.Sorceress, "pal": d2.Paladin}[code], Level: lvl, SkillPoints: unspent}
		for id, n := range pts {
			p.Skills = append(p.Skills, d2.Skill{ID: id, Level: n})
		}
		return p
	}
	check := func(l Legal, p d2.Player, id int, confirm bool) SkillVerdict {
		dk := NewDesk(Config{T: tables()})
		return Legal{T: l.T, Playable: l.Playable, Attack: l.Attack}.CheckSkill(dk.hero(&p), id, confirm)
	}
	// a legal point, a passive and an aura never ask
	for _, tc := range []struct {
		p    d2.Player
		id   int
		want string
	}{
		{mk("sor", "", 5, 1, nil), 36, TierSupported},
		{mk("sor", "", 5, 1, nil), 37, TierSupported},
		{mk("pal", "", 5, 1, nil), 96, TierSupported},
	} {
		if v := check(l, tc.p, tc.id, false); !v.OK || v.Tier != tc.want {
			t.Errorf("skill %d: %+v", tc.id, v)
		}
	}
	// refusals by the game's own rules
	for _, tc := range []struct {
		p    d2.Player
		id   int
		code string
		sub  string
	}{
		{mk("sor", "", 5, 0, nil), 36, CodeRule, "no unspent"},
		{mk("sor", "", 10, 1, nil), 41, CodeRule, "level 18"},
		{mk("sor", "", 20, 1, nil), 41, CodeRule, "needs Fire Bolt"},
		{mk("sor", "", 20, 1, nil), 96, CodeOtherClass, "another class"},
		{mk("pal", "", 5, 1, map[int]int{97: 20}), 97, CodeRule, "maximum"},
		{mk("pal", "", 5, 1, nil), 9999, CodeUnknownSkill, "no skill"},
	} {
		if v := check(l, tc.p, tc.id, false); v.OK || v.Code != tc.code || !strings.Contains(v.Reason, tc.sub) {
			t.Errorf("skill %d want %s/%s got %+v", tc.id, tc.code, tc.sub, v)
		}
	}
	// an unsupported attack asks once; an unsupported non-attack does not
	if v := check(l, mk("pal", "", 18, 1, nil), 112, false); v.OK || !v.NeedsConfirm || v.Code != CodeConfirm || v.Tier != TierUnsupported {
		t.Errorf("hammer without confirm: %+v", v)
	}
	if v := check(l, mk("pal", "", 18, 1, nil), 112, true); !v.OK || v.Tier != TierUnsupported {
		t.Errorf("hammer confirmed: %+v", v)
	}
	// the game has no per-level cap: a second point in a level-18 skill at clvl 18 is legal
	p := mk("pal", "", 18, 1, map[int]int{112: 1})
	if v := check(l, p, 112, true); !v.OK {
		t.Errorf("no cap: %+v", v)
	}
	if SkillCap(18, 18) != 1 || SkillCap(19, 18) != 2 || SkillCap(5, 18) != 0 {
		t.Error("SkillCap")
	}
}

func TestClassSkillIndex(t *testing.T) {
	tb := tables()
	for _, tc := range []struct {
		class string
		id    int
		idx   int
		ok    bool
	}{{"sor", 36, 0, true}, {"sor", 41, 2, true}, {"pal", 112, 2, true}, {"pal", 36, 0, false}} {
		if i, ok := ClassSkillIndex(tb, tc.class, tc.id); i != tc.idx || ok != tc.ok {
			t.Errorf("%s %d: %d %v", tc.class, tc.id, i, ok)
		}
	}
}

func TestTableViolations(t *testing.T) {
	req := map[string]int{"Blessed Hammer": 18, "Might": 1}
	rows := []TableRow{{17, map[string]int{"Might": 5}}, {18, map[string]int{"Blessed Hammer": 5}}, {19, map[string]int{"Blessed Hammer": 1}}}
	got := TableViolations(rows, func(s string) (int, bool) { r, ok := req[s]; return r, ok })
	if len(got) != 1 || !strings.Contains(got[0], "clvl 18: Blessed Hammer has 5 points, the cap is 1") {
		t.Errorf("%v", got)
	}
}

// fake is a game that takes a point when told and can be made to ignore the verb.
type fake struct {
	p                     d2.Player
	ignore, busy, verbOff bool
	err                   error
	statCalls, skillCalls []int
}

func (f *fake) desk(t *testing.T, dir string) *Desk {
	return NewDesk(Config{
		Hero: "Palli", T: tables(), Dir: dir,
		Player: func() *d2.Player { q := f.p; return &q },
		Fresh:  func(context.Context) (*d2.Player, error) { q := f.p; return &q, nil },
		Busy:   func() bool { return f.busy },
		Verbs:  func() Verbs { return Verbs{Stat: !f.verbOff, Skill: !f.verbOff, Via: "d2d verbs", Why: "stock d2d"} },
		SpendStat: func(_ context.Context, id int) error {
			f.statCalls = append(f.statCalls, id)
			if f.err != nil || f.ignore {
				return f.err
			}
			f.p.StatPoints--
			switch id {
			case 0:
				f.p.Strength++
			case 2:
				f.p.Dexterity++
			case 3:
				f.p.Vitality++
			case 1:
				f.p.Energy++
			}
			return nil
		},
		SpendSkill: func(_ context.Context, idx int) error {
			f.skillCalls = append(f.skillCalls, idx)
			if f.err != nil || f.ignore {
				return f.err
			}
			f.p.SkillPoints--
			f.p.Skills = append(f.p.Skills, d2.Skill{ID: []int{96, 97, 112}[idx], Level: 1})
			return nil
		},
		Playable: func(n string) bool { return n != "Blessed Hammer" },
		Attack:   func(n string) bool { return n == "Sacrifice" || n == "Blessed Hammer" },
	})
}

func TestSpendStatByVerbAndReadBack(t *testing.T) {
	f := &fake{p: d2.Player{Class: d2.Paladin, Level: 5, Strength: 25, Vitality: 25, StatPoints: 2}}
	d := f.desk(t, "")
	r := d.SpendStat(context.Background(), "vit")
	if !r.OK || r.Before != 25 || r.After != 26 || r.Left != 1 || len(f.statCalls) != 1 || f.statCalls[0] != 3 {
		t.Fatalf("%+v calls %v", r, f.statCalls)
	}
	// every stat key reaches its own verb id
	for key, id := range map[string]int{"str": 0, "dex": 2, "ene": 1} {
		f.p.StatPoints = 1
		if r := d.SpendStat(context.Background(), key); !r.OK || f.statCalls[len(f.statCalls)-1] != id {
			t.Errorf("%s: %+v calls %v", key, r, f.statCalls)
		}
	}
	for _, tc := range []struct {
		name   string
		mod    func()
		stat   string
		status int
		code   string
	}{
		{"unknown stat", func() {}, "luck", http.StatusBadRequest, "bad_stat"},
		{"no points", func() { f.p.StatPoints = 0 }, "str", http.StatusConflict, "no_points"},
		{"fight on", func() { f.p.StatPoints, f.busy = 1, true }, "str", http.StatusConflict, "busy"},
		{"no verb", func() { f.busy, f.verbOff = false, true }, "str", http.StatusNotImplemented, "no_verb"},
		{"verb error", func() { f.verbOff, f.err = false, errors.New("err unknown cmd") }, "str", http.StatusBadGateway, "verb_failed"},
		{"point not taken", func() { f.err, f.ignore = nil, true }, "str", http.StatusBadGateway, "not_taken"},
	} {
		tc.mod()
		if r := d.SpendStat(context.Background(), tc.stat); r.OK || r.Status != tc.status || r.Code != tc.code {
			t.Errorf("%s: %+v", tc.name, r)
		}
	}
}

func TestSpendSkillLegalAndIllegal(t *testing.T) {
	f := &fake{p: d2.Player{Class: d2.Paladin, Level: 18, SkillPoints: 3}}
	d := f.desk(t, "")
	ctx := context.Background()
	if r := d.SpendSkill(ctx, 97, false); !r.OK || r.After != 1 || f.skillCalls[0] != 1 {
		t.Fatalf("legal: %+v calls %v", r, f.skillCalls)
	}
	n := len(f.skillCalls)
	// an illegal point never reaches the verb
	if r := d.SpendSkill(ctx, 36, false); r.OK || r.Code != CodeOtherClass || r.Status != http.StatusConflict {
		t.Errorf("other class: %+v", r)
	}
	if r := d.SpendSkill(ctx, 9999, false); r.Code != CodeUnknownSkill || r.Status != http.StatusBadRequest {
		t.Errorf("unknown: %+v", r)
	}
	// the unsupported attack needs the confirmation
	if r := d.SpendSkill(ctx, 112, false); r.OK || !r.NeedsConfirm || r.Code != CodeConfirm {
		t.Errorf("hammer: %+v", r)
	}
	if len(f.skillCalls) != n {
		t.Fatalf("a refused point reached the verb: %v", f.skillCalls)
	}
	if r := d.SpendSkill(ctx, 112, true); !r.OK || r.Tier != TierUnsupported {
		t.Errorf("hammer confirmed: %+v", r)
	}
	f.verbOff = true
	if r := d.SpendSkill(ctx, 97, false); r.Code != "no_verb" || r.Status != http.StatusNotImplemented {
		t.Errorf("no verb: %+v", r)
	}
}

func TestAutopilotDefaultOffAndPersists(t *testing.T) {
	dir := t.TempDir()
	f := &fake{p: d2.Player{Class: d2.Paladin, StatPoints: 4, SkillPoints: 1}}
	d := f.desk(t, dir)
	if d.Autopilot() {
		t.Fatal("autopilot must default to off")
	}
	if txt, st, sk, ok := d.Waiting(); !ok || st != 4 || sk != 1 || txt != "4 stat points and 1 skill point waiting for you" {
		t.Errorf("waiting %q %d %d %v", txt, st, sk, ok)
	}
	if r := d.SetAutopilot(true); !r.OK || r.Already {
		t.Errorf("%+v", r)
	}
	if r := d.SetAutopilot(true); !r.OK || !r.Already {
		t.Errorf("repeat %+v", r)
	}
	if !f.desk(t, dir).Autopilot() { // a new desk of the same hero reads the file
		t.Error("switch did not persist")
	}
}

func TestAuditRingKeepsNewestAndNoAddress(t *testing.T) {
	a := NewAudit(t.TempDir(), "Palli")
	for i := 0; i < AuditMax+5; i++ {
		_ = a.Append(Event{At: time.Unix(int64(i), 0), Action: "stat", Target: "vit", Path: "/api/hero/points/stat", Remote: "lan", OK: true, Status: 200, Before: i})
	}
	r := a.Recent(3)
	if len(r) != 3 || r[0].Before != AuditMax+4 {
		t.Fatalf("%+v", r)
	}
	if got := len(a.Recent(0)); got != AuditMax {
		t.Errorf("kept %d", got)
	}
	if NewAudit("", "x").Append(Event{}) != nil || NewAudit(t.TempDir(), "../x").Recent(1) != nil {
		t.Error("unsafe or empty audit must be inert")
	}
}
