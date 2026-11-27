package points

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

func TestRespecPrice(t *testing.T) {
	p := RespecParams{BaseGold: 500, PerLevel: 100, PerRespecPct: 50, FirstFree: 1, MinGold: 1000, MaxGold: 5000}
	for _, c := range []struct{ clvl, respecs, want int }{
		{10, 0, 0},     // the first one is free
		{10, 1, 1500},  // 500 + 100x10, no escalation yet (first paid)
		{10, 2, 2250},  // + 50% of one paid respec
		{3, 1, 1000},   // floor
		{90, 1, 5000},  // cap
		{90, 3, 10000}, // cap, then +100%
	} {
		if got := RespecPrice(p, c.clvl, c.respecs); got != c.want {
			t.Errorf("clvl %d respecs %d: %d, want %d", c.clvl, c.respecs, got, c.want)
		}
	}
	p.MaxGold, p.FirstFree = 0, 0
	if got := RespecPrice(p, 90, 0); got != 9500 {
		t.Errorf("no cap: %d", got)
	}
}

// allocFake is a two-class hero with build tables as plain skill-name lists.
type allocFake struct {
	p      d2.Player
	logs   []string
	on     bool
	busy   bool
	builds map[string][]string
	late   string // tier of the endgame tables when set (a test makes them experimental)
	prereq map[string]bool
}

func (f *allocFake) lateTier(def string) string {
	if f.late != "" {
		return f.late
	}
	return def
}

func (f *allocFake) desk(t *testing.T, dir string) *Desk {
	tb := tables()
	return NewDesk(Config{
		Hero: "Palli", T: tb, Dir: dir, Now: func() time.Time { return time.Unix(1000, 0) },
		Player: func() *d2.Player { q := f.p; return &q },
		Fresh:  func(context.Context) (*d2.Player, error) { q := f.p; return &q, nil },
		Busy:   func() bool { return f.busy },
		Verbs:  func() Verbs { return Verbs{Stat: true, Skill: true, Via: "d2d verbs"} },
		SpendStat: func(_ context.Context, id int) error {
			f.p.StatPoints--
			return nil
		},
		SpendSkill: func(_ context.Context, idx int) error {
			c := tb.Class(f.p.Class.String())
			for _, s := range tb.Skills {
				if i, ok := ClassSkillIndex(tb, c.Code, s.ID); ok && i == idx && s.Class == c.Code {
					f.p.SkillPoints--
					for k := range f.p.Skills {
						if f.p.Skills[k].ID == s.ID {
							f.p.Skills[k].Level++
							return nil
						}
					}
					f.p.Skills = append(f.p.Skills, d2.Skill{ID: s.ID, Level: 1})
					return nil
				}
			}
			return nil
		},
		Playable: func(n string) bool { return n != "Blessed Hammer" && n != "Fire Ball" },
		Attack:   func(n string) bool { return n != "Warmth" && n != "Might" },
		AutoOn:   func() bool { return f.on },
		Respec:   func() RespecParams { return RespecParams{BaseGold: 500, PerLevel: 100, MinGold: 1000} },
		Builds: func() []BuildInfo {
			return []BuildInfo{
				{Name: "pal_a", Class: "paladin", Tier: BuildSupported, Label: "pal a"},
				{Name: "pal_x", Class: "paladin", Tier: BuildExperimental, Label: "pal x"},
				{Name: "sor_a", Class: "sorceress", Tier: BuildSupported, Label: "sor a"},
				{Name: "necromancer_summoner_normal30", Class: "necromancer", Tier: BuildSupported},
				{Name: "necromancer_summoner_endgame90", Class: "necromancer", Tier: f.lateTier(BuildSupported)},
				{Name: "paladin-normal30-v2", Class: "paladin", Tier: BuildSupported},
				{Name: "paladin_hammerdin_endgame90", Class: "paladin", Tier: f.lateTier(BuildSupported)},
			}
		},
		Next: func(build string, p *d2.Player) ([]Step, error) {
			var out []Step
			for _, n := range f.builds[build] {
				if n == "+vit" {
					out = append(out, Step{Kind: "stat", Stat: "vit", Name: n})
					continue
				}
				for _, s := range tb.Skills {
					if s.Name == n {
						have := 0
						for _, k := range p.Skills {
							if k.ID == s.ID {
								have = k.Level
							}
						}
						if have == 0 {
							out = append(out, Step{Kind: "skill", Skill: s.ID, Name: n, Prereq: f.prereq[n]})
						}
					}
				}
			}
			return out, nil
		},
		Logf: func(msg string, kv ...any) { f.logs = append(f.logs, msg) },
	})
}

func newAllocFake(class d2.Class) *allocFake {
	return &allocFake{p: d2.Player{Class: class, Level: 2}, on: true, builds: map[string][]string{
		"pal_a": {"Might", "Sacrifice", "Blessed Hammer"},
		"pal_x": {"Blessed Hammer"},
		"sor_a": {"Fire Bolt", "Warmth", "Fire Ball"},
	}}
}

func TestSetBuildLocksAndRefusesSwitch(t *testing.T) {
	dir := t.TempDir()
	f := newAllocFake(d2.Paladin)
	d := f.desk(t, dir)
	if r := d.SetBuild("nope", false); r.OK || r.Status != 400 || r.Code != "unknown_build" || !strings.Contains(r.Reason, `"nope"`) {
		t.Errorf("unknown: %+v", r)
	}
	if r := d.SetBuild("sor_a", false); r.OK || r.Code != "other_class" {
		t.Errorf("other class: %+v", r)
	}
	if r := d.SetBuild("pal_x", false); r.OK || !r.NeedsConfirm || d.store.Get().Locked {
		t.Errorf("experimental without confirm: %+v", r)
	}
	if r := d.SetBuild("pal_a", false); !r.OK || r.Already {
		t.Errorf("lock: %+v", r)
	}
	if r := d.SetBuild("pal_a", false); !r.OK || !r.Already {
		t.Errorf("idempotent: %+v", r)
	}
	r := d.SetBuild("pal_x", true)
	if r.OK || r.Status != 409 || r.Code != "locked" || !strings.Contains(r.Reason, "pal_a") || !strings.Contains(r.Reason, "respec") {
		t.Errorf("switch while locked: %+v", r)
	}
	// the lock survives a restart (new desk, same state dir)
	if s := f.desk(t, dir).store.Get(); !s.Locked || s.Build != "pal_a" || s.LockedAtLevel != 2 {
		t.Errorf("stored %+v", s)
	}
	// a paid respec (a later card) clears the lock, counts, and the pick is free again
	if err := d.store.ClearBuild(time.Unix(2000, 0)); err != nil {
		t.Fatal(err)
	}
	if s := d.store.Get(); s.Locked || s.Build != "" || s.Respecs != 1 {
		t.Errorf("after respec %+v", s)
	}
	if r := d.SetBuild("pal_x", true); !r.OK || r.Tier != BuildExperimental {
		t.Errorf("experimental with confirm: %+v", r)
	}
}

func TestAutoAllocFollowsBuildTwoClasses(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		class d2.Class
		build string
		want  []string
	}{
		{d2.Paladin, "pal_a", []string{"Might", "Sacrifice"}},
		{d2.Sorceress, "sor_a", []string{"Fire Bolt", "Warmth"}},
	} {
		f := newAllocFake(c.class)
		d := f.desk(t, t.TempDir())
		f.p.Level, f.p.SkillPoints = 5, 1
		if got := d.AutoAllocOnce(ctx); got != "" {
			t.Fatalf("%s: spent without a lock: %q", c.build, got)
		}
		if r := d.SetBuild(c.build, false); !r.OK {
			t.Fatal(r)
		}
		f.on = false
		if got := d.AutoAllocOnce(ctx); got != "" || f.p.SkillPoints != 1 {
			t.Fatalf("%s: knob off still spent: %q", c.build, got)
		}
		f.on = true
		f.busy = true
		if got := d.AutoAllocOnce(ctx); got != "" || f.p.SkillPoints != 1 {
			t.Fatalf("%s: spent in a fight: %q", c.build, got)
		}
		f.busy = false
		for i, w := range c.want {
			if i > 0 {
				f.p.Level++
				f.p.SkillPoints++
			}
			if got := d.AutoAllocOnce(ctx); got != "skill "+w {
				t.Fatalf("%s level %d: got %q, want skill %s", c.build, f.p.Level, got, w)
			}
		}
		// a second run in the same state spends nothing (no points left): idempotent
		if got := d.AutoAllocOnce(ctx); got != "" {
			t.Errorf("%s: spent with no points: %q", c.build, got)
		}
		last := f.logs[len(f.logs)-1]
		if !strings.HasPrefix(last, "auto-alloc: clvl 6 -> skill "+c.want[1]) || !strings.HasSuffix(last, "(build "+c.build+")") {
			t.Errorf("%s: trace line %q", c.build, last)
		}
	}
}

func TestAutoAllocHoldsUnsupportedAttackAndSpendsNext(t *testing.T) {
	f := newAllocFake(d2.Paladin)
	f.p.Level, f.p.SkillPoints = 18, 1
	f.builds["pal_a"] = []string{"Blessed Hammer", "Might"}
	d := f.desk(t, "")
	d.SetBuild("pal_a", false)
	if got := d.AutoAllocOnce(context.Background()); got != "skill Might" {
		t.Fatalf("got %q", got)
	}
	v := d.BuildView()
	if len(v.Held) != 1 || v.Held[0].Skill != "Blessed Hammer" || !strings.Contains(v.Held[0].Reason, "cannot use yet") {
		t.Errorf("held %+v", v.Held)
	}
	if !v.State.Locked || v.Respec.Available || v.Respec.Price <= 0 {
		t.Errorf("view %+v", v)
	}
}

func TestAutoAllocStatThenSkill(t *testing.T) {
	f := newAllocFake(d2.Sorceress)
	f.p.StatPoints, f.p.SkillPoints = 1, 1
	f.builds["sor_a"] = []string{"+vit", "Fire Bolt"}
	d := f.desk(t, "")
	d.SetBuild("sor_a", false)
	if got := d.AutoAllocOnce(context.Background()); got != "stat vit" {
		t.Fatalf("got %q", got)
	}
	if got := d.AutoAllocOnce(context.Background()); got != "skill Fire Bolt" {
		t.Fatalf("got %q", got)
	}
}

func TestLockedGate(t *testing.T) {
	f := newAllocFake(d2.Paladin)
	d := f.desk(t, "")
	if d.Locked() {
		t.Fatal("locked without a build")
	}
	d.SetBuild("pal_a", false)
	if !d.Locked() {
		t.Fatal("not locked")
	}
	f.on = false
	if d.Locked() {
		t.Error("the allocator owns the points only while the knob is on")
	}
}

// The Points page must say what the allocator did and why it waits (user report 2026-11-23: a necromancer showed
// nothing for minutes while points were pending behind a fight).
func TestBuildViewSaysUnspentSpentAndWaiting(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		build string
	}{{d2.Sorceress, "sor_a"}, {d2.Paladin, "pal_a"}} {
		f := newAllocFake(c.class)
		f.p.Level, f.p.SkillPoints, f.p.StatPoints = 4, 1, 1
		d := f.desk(t, "")
		d.SetBuild(c.build, false)
		f.busy = true
		d.AutoAllocOnce(context.Background())
		v := d.BuildView()
		if v.Unspent.Skill != 1 || v.Unspent.Stat != 1 || v.Spent != 0 || !strings.Contains(v.Waiting, "fight") {
			t.Fatalf("%s busy view: %+v", c.build, v)
		}
		f.busy = false
		d.AutoAllocOnce(context.Background())
		v = d.BuildView()
		if v.Spent != 1 || v.Waiting != "" {
			t.Errorf("%s after a spend: spent %d waiting %q", c.build, v.Spent, v.Waiting)
		}
	}
}

// The horizon handoff: a locked 30-level table continues with the build's next SUPPORTED table (catalogue fromLevel),
// loudly, never into an experimental one.
func TestAutoAllocHandoffTwoClasses(t *testing.T) {
	ctx := context.Background()
	for _, c := range []struct {
		class        d2.Class
		first, later string
		want         string
	}{
		{d2.Necromancer, "necromancer_summoner_normal30", "necromancer_summoner_endgame90", "handoff normal30 -> endgame90 at clvl 30"},
		{d2.Paladin, "paladin-normal30-v2", "paladin_hammerdin_endgame90", "handoff paladin-normal30-v2 -> endgame90 at clvl 30"},
	} {
		f := newAllocFake(c.class)
		f.builds[c.first] = []string{"+vit"}
		f.builds[c.later] = []string{"+vit"}
		dir := t.TempDir()
		d := f.desk(t, dir)
		f.p.Level, f.p.StatPoints = 29, 1
		if r := d.SetBuild(c.first, false); !r.OK {
			t.Fatal(r)
		}
		if got := d.AutoAllocOnce(ctx); got != "stat vit" || d.store.Get().Plan != "" {
			t.Fatalf("%s clvl 29: %q plan %q", c.first, got, d.store.Get().Plan)
		}
		if v := d.BuildView(); !strings.Contains(v.Plan, "moves on to the full plan") {
			t.Errorf("%s plan line before: %q", c.first, v.Plan)
		}
		f.p.Level, f.p.StatPoints = 31, 1
		if got := d.AutoAllocOnce(ctx); got != "stat vit" {
			t.Fatalf("%s clvl 31: %q", c.first, got)
		}
		if s := d.store.Get(); s.Plan != c.later || s.PlanAtLevel != 31 {
			t.Errorf("%s stored %+v", c.first, s)
		}
		found := false
		for _, l := range f.logs {
			found = found || strings.HasSuffix(l, strings.TrimSuffix(c.want, "30")+"31") && strings.Contains(l, "auto-alloc: handoff")
		}
		if !found {
			t.Errorf("%s: no handoff trace in %v", c.first, f.logs)
		}
		if v := d.BuildView(); !strings.Contains(v.Plan, "now following the full plan from level 30") {
			t.Errorf("%s plan line after: %q", c.first, v.Plan)
		}
		if f.p.StatPoints = 1; true {
			d.AutoAllocOnce(ctx)
		}
		n := 0
		for _, l := range f.logs {
			if strings.Contains(l, "auto-alloc: handoff") {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s: handoff logged %d times", c.first, n)
		}
		if s := f.desk(t, dir).store.Get(); s.Plan != c.later {
			t.Errorf("%s: handoff lost over a restart: %+v", c.first, s)
		}
	}
}

func TestAutoAllocNoHandoffIntoExperimental(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		first string
	}{{d2.Necromancer, "necromancer_summoner_normal30"}, {d2.Paladin, "paladin-normal30-v2"}} {
		f := newAllocFake(c.class)
		f.late = BuildExperimental
		f.builds[c.first] = []string{"+vit"}
		d := f.desk(t, "")
		f.p.Level, f.p.StatPoints = 35, 1
		d.SetBuild(c.first, false)
		d.AutoAllocOnce(context.Background())
		if d.store.Get().Plan != "" {
			t.Errorf("%s: handed off into an experimental table", c.first)
		}
		if v := d.BuildView(); !strings.Contains(v.Plan, "not supported yet") {
			t.Errorf("%s plan line: %q", c.first, v.Plan)
		}
	}
}

// A table with no step left for unspent points hands off early to the nearest later supported table.
func TestAutoAllocHandoffWhenTableExhausted(t *testing.T) {
	f := newAllocFake(d2.Necromancer)
	f.builds["necromancer_summoner_endgame90"] = []string{"+vit"}
	d := f.desk(t, "")
	f.p.Level, f.p.StatPoints = 27, 1
	d.SetBuild("necromancer_summoner_normal30", false)
	if got := d.AutoAllocOnce(context.Background()); got != "stat vit" || d.store.Get().Plan != "necromancer_summoner_endgame90" {
		t.Fatalf("got %q plan %q", got, d.store.Get().Plan)
	}
}

// prereq (live 2026-11-27, Doofer): the capability table marks Sacrifice unsupported, so the allocator held it for a
// confirmation and Zeal, which needs it, was never learned. A prerequisite step is spent without the confirmation; the
// same skill as a plain step is still held, and the hold is logged once. Two builds: Paladin and Sorceress.
func TestAutoAllocSpendsUnsupportedPrerequisite(t *testing.T) {
	for _, c := range []struct {
		class d2.Class
		build string
		skill string
	}{{d2.Paladin, "pal_a", "Blessed Hammer"}, {d2.Sorceress, "sor_a", "Fire Ball"}} {
		f := newAllocFake(c.class)
		f.p.Level, f.p.SkillPoints = 18, 1
		if c.class == d2.Sorceress {
			f.p.Skills = []d2.Skill{{ID: 36, Level: 1}} // Fire Bolt, the prerequisite of Fire Ball
		}
		f.builds[c.build] = []string{c.skill}
		d := f.desk(t, "")
		d.SetBuild(c.build, false)
		if got := d.AutoAllocOnce(context.Background()); got != "" {
			t.Fatalf("%s: a plain unsupported attack was spent: %q", c.build, got)
		}
		d.AutoAllocOnce(context.Background())
		n := 0
		for _, l := range f.logs {
			if strings.HasPrefix(l, "auto-alloc: held "+c.skill) {
				n++
			}
		}
		if n != 1 {
			t.Errorf("%s: held lines %d in %v", c.build, n, f.logs)
		}
		f.prereq = map[string]bool{c.skill: true}
		if got := d.AutoAllocOnce(context.Background()); got != "skill "+c.skill {
			t.Fatalf("%s: prerequisite not spent: %q", c.build, got)
		}
	}
}
