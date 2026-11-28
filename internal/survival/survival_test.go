package survival

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/danger"
	"d2lod-bot/internal/power"
	"d2lod-bot/internal/store"
)

var t0 = time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)

func params(t *testing.T) *Params {
	t.Helper()
	p, err := LoadParams()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func pmodel(t *testing.T) *power.Model {
	t.Helper()
	m, err := power.NewModel(danger.DefaultParams())
	if err != nil {
		t.Fatal(err)
	}
	return m
}

// field is a made-up zone (real tables are not in the tests; docs/survival/power.md has the live steps).
func field(area int, name string, mlvl, melee, ranged int, life, dmg, xp float64) *danger.Zone {
	mon := func(reach string) danger.Monster {
		return danger.Monster{Key: reach, Share: 0.5, Life: int(life), Def: 30, ToHit: 60 + 5*mlvl, Dmg: dmg, Reach: reach}
	}
	return &danger.Zone{Area: area, Name: name, Diff: "normal", MLvl: mlvl, Monsters: []danger.Monster{mon("melee"), mon("ranged")},
		Pay: danger.Pay{XPPerKill: xp},
		Encounters: danger.Encounters{
			Typical: danger.Encounter{N: melee + ranged, Melee: melee, Ranged: ranged},
			Bad:     danger.Encounter{N: 2 * (melee + ranged), Melee: 2 * melee, Ranged: 2 * ranged},
			Elite:   danger.Encounter{N: melee + ranged + 1, Melee: melee + 1, Ranged: ranged, Leader: "unique"}}}
}

var zones = []*danger.Zone{
	field(2, "Cold Plains", 5, 2, 1, 40, 8, 40),
	field(3, "Stony Field", 7, 3, 1, 55, 12, 90),
	field(4, "Black Marsh", 9, 6, 2, 80, 16, 160),
}

func paladin(lvl int) power.Hero {
	h := power.Hero{Class: "Paladin", Level: lvl, MaxLife: 120 + 12*lvl, MaxMana: 40, Defence: 60 + 8*lvl, BlockPct: 30, AR: 60 + 12*lvl,
		DmgMin: 3 + lvl, DmgMax: 8 + 2*lvl, DmgPct: 20, Potions: 6, Resist: [4]int{10, 10, 10, 10}, Merc: power.Merc{Hired: true, Alive: true, Level: lvl}}
	for _, term := range []string{power.TermLife, power.TermDefence, power.TermBlock, power.TermResist, power.TermOffence, power.TermPotions, power.TermMerc} {
		h.SetQuality(term, power.Measured)
	}
	return h
}

func sorceress(lvl int) power.Hero {
	h := power.Hero{Class: "Sorceress", Level: lvl, MaxLife: 60 + 9*lvl, MaxMana: 60 + 10*lvl, Defence: 20, MeasuredDPS: 3 + 1.5*float64(lvl), Potions: 6,
		Merc: power.Merc{Hired: true, Alive: true, Level: lvl}}
	for _, term := range []string{power.TermLife, power.TermDefence, power.TermOffence, power.TermPotions, power.TermResist, power.TermMerc} {
		h.SetQuality(term, power.Measured)
	}
	return h
}

type env struct {
	t    *testing.T
	m    *power.Model
	p    *Params
	arch *power.Archetype
}

func newEnv(t *testing.T, archID string) env {
	m := pmodel(t)
	a, err := m.P.Archetype(archID)
	if err != nil {
		t.Fatal(err)
	}
	return env{t, m, params(t), a}
}

func (e env) cands(h power.Hero) []Candidate {
	var out []Candidate
	for _, z := range zones {
		zm, err := e.m.Zone(h, e.arch, z, "balanced")
		if err != nil {
			e.t.Fatal(err)
		}
		out = append(out, Candidate{Key: ZoneKey{z.Area, z.Diff}, Name: z.Name, ZM: zm, Eco: Economy(z, zm, h.Level, e.p), TravelMin: 2})
	}
	return out
}

func (e env) input(h power.Hero, cur ZoneKey, now time.Time) Input {
	d, err := e.m.P.Dial("balanced")
	if err != nil {
		e.t.Fatal(err)
	}
	return Input{Now: now, Dial: d, Current: cur, Power: PowerOf(h), Purse: Purse{Gold: 5000, Need: 500}, Cands: e.cands(h)}
}

func TestBothBuildsPickAndRatchet(t *testing.T) {
	for _, tc := range []struct {
		name, arch string
		hero       func(int) power.Hero
	}{{"paladin", "bruiser", paladin}, {"sorceress", "caster", sorceress}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, tc.arch)
			l := NewLedger(e.p)
			rank := func(lvl int) int {
				d, err := l.Choose(e.input(tc.hero(lvl), ZoneKey{}, t0))
				if err != nil {
					t.Fatal(err)
				}
				if d.Why.Code == "" || d.Line() == "" {
					t.Fatalf("lvl %d: no why: %+v", lvl, d)
				}
				return d.Area
			}
			prev := 0
			for _, lvl := range []int{3, 8, 14, 22, 32} {
				a := rank(lvl)
				if a < prev {
					t.Fatalf("lvl %d chose area %d after %d: the choice must ratchet up with power", lvl, a, prev)
				}
				prev = a
			}
			if rank(3) == 4 {
				t.Fatal("a clvl 3 hero entered the hardest zone")
			}
		})
	}
}

// swap7 shape: the hero is in Black Marsh, the merc dies twice, the hero leaves, and must not walk back in while its
// power is unchanged (design 5.3 and 5.4).
func TestNoSecondBlackMarshEntry(t *testing.T) {
	e := newEnv(t, "bruiser")
	var lvl int
	var h power.Hero
	for lvl = 6; lvl <= 40; lvl++ {
		h = paladin(lvl)
		in := e.input(h, ZoneKey{}, t0)
		l := NewLedger(e.p)
		if d, _ := l.Choose(in); d.Area == 4 {
			break
		}
	}
	if lvl > 40 {
		t.Skip("no level in 6..40 where the test zones make Black Marsh the best pick")
	}
	l := NewLedger(e.p)
	bm := ZoneKey{4, "normal"}
	in := e.input(h, bm, t0)
	d, _ := l.Choose(in)
	if d.Area != 4 {
		t.Fatalf("setup: wanted Black Marsh, got %d", d.Area)
	}
	now := t0
	for i := 0; i < 2; i++ {
		now = now.Add(8 * time.Minute)
		l.Observe(bm, now, Event{Kind: EvMercDeath}, in.Power)
	}
	h.Merc.Alive = false
	in = e.input(h, bm, now)
	d, _ = l.Choose(in)
	if d.Area == 4 {
		t.Fatalf("stayed in Black Marsh after two merc deaths: %s", d.Line())
	}
	for _, later := range []time.Duration{time.Minute, 45 * time.Minute, 6 * time.Hour} {
		in = e.input(h, ZoneKey{d.Area, "normal"}, now.Add(later))
		d2, _ := l.Choose(in)
		if d2.Area == 4 {
			t.Fatalf("re-entered Black Marsh %s later at unchanged power: %s", later, d2.Line())
		}
	}
	// stronger by more than 10 percent: the lock lifts
	h.Merc.Alive = true
	h.MaxLife = h.MaxLife * 13 / 10
	in = e.input(h, ZoneKey{d.Area, "normal"}, now.Add(6*time.Hour))
	if l.Locked(bm, in.Power) {
		t.Fatal("lock must lift when power grew by more than 10 percent")
	}
}

func TestDialStepDown(t *testing.T) {
	p := params(t)
	for _, tc := range []struct {
		dial   string
		events []string
		out    bool
	}{
		{"cautious", []string{EvChicken}, true},
		{"balanced", []string{EvChicken}, false},
		{"balanced", []string{EvChicken, EvMercDeath}, true},
		{"balanced", []string{EvDeath}, true},
		{"bold", []string{EvChicken, EvMercDeath, EvChicken}, false},
		{"bold", []string{EvDeath}, true},
	} {
		l := NewLedger(p)
		k := ZoneKey{3, "normal"}
		for _, ev := range tc.events {
			l.Observe(k, t0, Event{Kind: ev}, 100)
		}
		if got, _ := l.ForcedOut(k, tc.dial, t0.Add(time.Minute)); got != tc.out {
			t.Errorf("%s %v: forced out %v, want %v", tc.dial, tc.events, got, tc.out)
		}
		if got, _ := l.ForcedOut(k, tc.dial, t0.Add(2*time.Hour)); got {
			t.Errorf("%s %v: still forced out two hours later", tc.dial, tc.events)
		}
	}
}

func TestKZoneAndKHero(t *testing.T) {
	p := params(t)
	l := NewLedger(p)
	k := ZoneKey{3, "normal"}
	if l.KZone(k, t0) != 1 || l.KHero() != 1 {
		t.Fatal("no data must mean 1")
	}
	// the zone takes three times the predicted damage, offence as predicted
	l.Observe(k, t0, Event{Kind: EvFight, Kills: 300, Seconds: 600, Taken: 3000, PredTaken: 1000, PredKillSec: 600}, 100)
	if v := l.KZone(k, t0); v < 2 || v > 3 {
		t.Fatalf("k_zone %.2f, want about 2.8 (3 shrunk by 300/330)", v)
	}
	if v := l.KHero(); v < 0.95 || v > 1.05 {
		t.Fatalf("k_hero %.2f, want 1", v)
	}
	// a slow hero everywhere: k_hero up, and k_zone no longer double-counts it
	l2 := NewLedger(p)
	l2.Observe(k, t0, Event{Kind: EvFight, Kills: 300, Seconds: 1200, Taken: 2000, PredTaken: 1000, PredKillSec: 600}, 100)
	if l2.KHero() < 1.7 {
		t.Fatalf("k_hero %.2f, want about 1.9", l2.KHero())
	}
	if v := l2.KZone(k, t0); v > 1.3 {
		t.Fatalf("k_zone %.2f: the offence error is in k_hero, not here", v)
	}
	// hard events multiply at once, relax with time away and reset when power moves
	l3 := NewLedger(p)
	l3.Observe(k, t0, Event{Kind: EvDeath}, 100)
	if v := l3.KZone(k, t0); v < 2.4 || v > 2.6 {
		t.Fatalf("death k_zone %.2f, want 2.5", v)
	}
	if v := l3.KZone(k, t0.Add(4*time.Hour)); v >= 1.5 {
		t.Fatalf("k_zone %.2f four hours later did not relax", v)
	}
	l3.Refresh(k, 130)
	if v := l3.KZone(k, t0); v != 1 {
		t.Fatalf("k_zone %.2f after a 30 percent power change, want 1", v)
	}
}

func TestDepletedFreshGame(t *testing.T) {
	e := newEnv(t, "bruiser")
	l := NewLedger(e.p)
	h := paladin(14)
	in := e.input(h, ZoneKey{}, t0)
	base, _ := l.Choose(in)
	for i := range in.Cands {
		if in.Cands[i].Key.Area == base.Area {
			in.Cands[i].Depleted = true
		}
	}
	in.FreshAllowed = true
	d, _ := l.Choose(in)
	if d.Why.Code != WhyFresh && d.Why.Code != WhyDepleted {
		t.Fatalf("depleted best zone: %s", d.Line())
	}
	for i := range in.Cands { // every eligible zone is cleared: a fresh game, same zone
		in.Cands[i].Depleted = true
	}
	d, _ = l.Choose(in)
	if !d.Fresh {
		t.Fatalf("every safe zone cleared: want a fresh game: %s", d.Line())
	}
	in.FreshAllowed = false
	if d, _ = l.Choose(in); d.Fresh {
		t.Fatal("fresh game chosen while not allowed")
	}
}

func TestHoleIsFirstFieldNeedsYou(t *testing.T) {
	e := newEnv(t, "caster")
	l := NewLedger(e.p)
	h := sorceress(2)
	h.MaxLife = 12
	in := e.input(h, ZoneKey{}, t0)
	d, err := l.Choose(in)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Hole || !d.NeedsYou || d.Area == 0 || d.Why.Code != WhyHole {
		t.Fatalf("a hero with 12 life: %+v", d)
	}
	if _, err := l.Choose(Input{Now: t0}); err == nil {
		t.Fatal("no candidates must be an error")
	}
}

func reviveIn(e env, h power.Hero, l *Ledger, quote, gold int, cur ZoneKey, now time.Time) ReviveInput {
	dead := h
	dead.Merc.Alive = false
	alive := h
	alive.Merc.Alive = true
	base := e.input(alive, cur, now)
	return ReviveInput{Base: base, With: e.cands(alive), Without: e.cands(dead), Quote: quote, Reserve: 300, Gold: gold}
}

func TestReviveValue(t *testing.T) {
	for _, tc := range []struct {
		name, arch string
		hero       func(int) power.Hero
	}{{"paladin", "bruiser", paladin}, {"sorceress", "caster", sorceress}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, tc.arch)
			l := NewLedger(e.p)
			h := tc.hero(14)
			cheap := l.Revive(reviveIn(e, h, l, 1, 100000, ZoneKey{}, t0))
			dear := l.Revive(reviveIn(e, h, l, 10000000, 100000000, ZoneKey{}, t0))
			if !cheap.Revive || cheap.Why.Code != WhyMercRevive {
				t.Fatalf("a 1 gold revive must be worth it: %+v", cheap)
			}
			if dear.Revive || dear.Why.Code != WhyMercDead || !strings.Contains(dear.Why.Text, "left dead") {
				t.Fatalf("a revive costing millions must be refused with a reason: %+v", dear)
			}
			if poor := l.Revive(reviveIn(e, h, l, 400, 500, ZoneKey{}, t0)); poor.Revive || poor.Why.Code != WhyMercDead {
				t.Fatalf("the purse rule keeps the kit reserve: %+v", poor)
			}
			// the more often he dies where the hero farms, the less he is worth
			rIn := reviveIn(e, h, l, 200, 100000, ZoneKey{}, t0)
			before := l.Revive(rIn)
			for _, c := range rIn.With {
				l.Observe(c.Key, t0, Event{Kind: EvFight, Kills: 50, Seconds: 600}, rIn.Base.Power)
				l.Observe(c.Key, t0, Event{Kind: EvMercDeath}, rIn.Base.Power)
			}
			after := l.Revive(reviveIn(e, h, l, 200, 100000, ZoneKey{}, t0.Add(time.Minute)))
			if after.Gain >= before.Gain && after.Why.Code != WhyMercNoRevive {
				t.Fatalf("a merc that keeps dying must be worth less: before %.1f after %.1f", before.Gain, after.Gain)
			}
		})
	}
}

// swap7 paid 360 gold twice and was refused a third time; here a second death in the window closes the zone to reviving.
func TestTwoMercDeathsCloseRevive(t *testing.T) {
	e := newEnv(t, "bruiser")
	l := NewLedger(e.p)
	h := paladin(16)
	in := reviveIn(e, h, l, 200, 100000, ZoneKey{}, t0)
	for _, c := range in.With {
		l.Observe(c.Key, t0, Event{Kind: EvMercDeath}, in.Base.Power)
		l.Observe(c.Key, t0.Add(10*time.Minute), Event{Kind: EvMercDeath}, in.Base.Power)
	}
	later := t0.Add(15 * time.Minute)
	d := l.Revive(reviveIn(e, h, l, 200, 100000, ZoneKey{}, later))
	if d.Revive || d.Why.Code != WhyMercNoRevive {
		t.Fatalf("every zone killed him twice: %+v", d)
	}
	if l.MercDeathsIn(in.With[0].Key, later) != 2 {
		t.Fatal("window count")
	}
	if l.NoReviveNow(in.With[0].Key, later.Add(3*time.Hour)) {
		t.Fatal("the no-revive mark must expire")
	}
}

func TestLedgerPersistsAndUpgrades(t *testing.T) {
	p := params(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "ledger.json")
	l, info, err := LoadLedger(path, p)
	if err != nil || info.Exists || len(l.Zones) != 0 {
		t.Fatalf("a missing file is an empty ledger: %v %+v", err, info)
	}
	k := ZoneKey{4, "normal"}
	l.Observe(k, t0, Event{Kind: EvFight, Kills: 40, Seconds: 300, Taken: 900, PredTaken: 600, PredKillSec: 280, XP: 500}, 150)
	l.Observe(k, t0, Event{Kind: EvMercDeath}, 150)
	l.MarkLeft(k, t0, 150)
	if err := l.Save(path); err != nil {
		t.Fatal(err)
	}
	back, _, err := LoadLedger(path, p)
	if err != nil {
		t.Fatal(err)
	}
	if back.KZone(k, t0) != l.KZone(k, t0) || back.KHero() != l.KHero() || !back.Locked(k, 150) || back.MercMinutes(k) != l.MercMinutes(k) {
		t.Fatal("reloaded ledger differs")
	}
	// an older document without any of the newer fields loads with defaults
	old := `{"version":1,"data":{"zones":{"4/normal":{"kills":10,"taken":50,"predTaken":50}}}}`
	if err := os.WriteFile(path, []byte(old), 0o600); err != nil {
		t.Fatal(err)
	}
	o, _, err := LoadLedger(path, p)
	if err != nil {
		t.Fatalf("old document: %v", err)
	}
	if v := o.KZone(k, t0); v < 0.9 || v > 1.1 || o.KHero() != 1 {
		t.Fatalf("defaults for missing fields: %.2f %.2f", v, o.KHero())
	}
	// a document with fields this build does not know loads too
	if err := os.WriteFile(path, []byte(`{"version":1,"data":{"zones":{},"futureField":[1,2]}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadLedger(path, p); err != nil {
		t.Fatalf("unknown fields: %v", err)
	}
	// a newer version is refused and left alone
	if err := os.WriteFile(path, []byte(`{"version":9,"data":{}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadLedger(path, p); !errors.Is(err, store.ErrFutureVersion) {
		t.Fatalf("newer version: %v", err)
	}
	if b, _ := os.ReadFile(path); !strings.Contains(string(b), `"version":9`) {
		t.Fatal("a newer file must not be overwritten by a load")
	}
}

var digits = regexp.MustCompile(`[0-9]`)

// The watch screen never shows a number for the margin (user decision 2026-11-14): the block has the three words, ids,
// names, the dial and sentences without digits.
func TestGlanceIsWordsOnly(t *testing.T) {
	e := newEnv(t, "bruiser")
	l := NewLedger(e.p)
	h := paladin(14)
	in := e.input(h, ZoneKey{}, t0)
	d, _ := l.Choose(in)
	merc := ReviveDecision{Why: Why{WhyMercDead, "left dead: in Stony Field he adds too little for what he costs"}}
	g := GlanceOf(d, "balanced", &merc)
	b, err := json.Marshal(g)
	if err != nil {
		t.Fatal(err)
	}
	var tree any
	_ = json.Unmarshal(b, &tree)
	var walk func(path string, v any)
	walk = func(path string, v any) {
		switch x := v.(type) {
		case map[string]any:
			for k, c := range x {
				walk(path+"."+k, c)
			}
		case []any:
			for _, c := range x {
				walk(path, c)
			}
		case float64:
			if !strings.HasSuffix(path, ".id") {
				t.Errorf("number at %s: the glance block must not carry numbers", path)
			}
		case string:
			if strings.HasSuffix(path, ".margin") && x != "safe" && x != "risky" && x != "too hard" {
				t.Errorf("%s = %q, want a word", path, x)
			}
			if strings.HasSuffix(path, ".text") && digits.MatchString(x) {
				t.Errorf("%s carries a digit: %q", path, x)
			}
		}
	}
	walk("", tree)
	if g.Merc == nil || g.Zone.Margin == "" || g.Zone.Why.Code == "" {
		t.Fatalf("missing parts: %s", b)
	}
	for _, a := range g.Alternatives {
		if a.Why.Code == "" {
			t.Errorf("alternative %q has no reason", a.Name)
		}
	}
	var known = map[string]bool{}
	for _, c := range Codes {
		known[c] = true
	}
	if !known[g.Zone.Why.Code] {
		t.Errorf("code %q is not in the closed list", g.Zone.Why.Code)
	}
}

func TestUnknownDialIsAnError(t *testing.T) {
	l := NewLedger(params(t))
	if _, err := l.P.stepDown("reckless"); err == nil || !strings.Contains(err.Error(), "reckless") {
		t.Fatalf("unknown dial: %v", err)
	}
	if out, _ := l.ForcedOut(ZoneKey{1, "normal"}, "reckless", t0); out {
		t.Fatal("no data cannot force out")
	}
}

// Card survcal: a near-zero prediction must not inflate k_zone (sandbox replay: 0.3 to 3 life per kill observed).
func TestPredTakenFloor(t *testing.T) {
	p := params(t)
	k := ZoneKey{2, "normal"}
	l := NewLedger(p)
	// predicted 0.05 life per kill, observed 0.5: the raw ratio is 10 and would sit on the clamp
	l.Observe(k, t0, Event{Kind: EvFight, Kills: 300, Seconds: 600, Taken: 150, PredTaken: 15, PredKillSec: 600}, 100)
	if v := l.KZone(k, t0); v > 1 {
		t.Fatalf("k_zone %.2f: a trivial zone must not read as a hazard", v)
	}
	if got := l.peek(k).PredTaken; got != 300*p.PredTakenFloor {
		t.Fatalf("predicted %.0f, want the floor %.0f", got, 300*p.PredTakenFloor)
	}
	// a fight without any prediction adds none (no ratio to form)
	l2 := NewLedger(p)
	l2.Observe(k, t0, Event{Kind: EvFight, Kills: 50, Seconds: 60, Taken: 20}, 100)
	if l2.peek(k).PredTaken != 0 {
		t.Fatal("no prediction must stay no prediction")
	}
}

func TestKHeroUnreadIsBounded(t *testing.T) {
	p := params(t)
	k := ZoneKey{2, "normal"}
	l := NewLedger(p)
	// the model's default kill speed is 5 times too slow for this hero
	l.Observe(k, t0, Event{Kind: EvFight, Kills: 600, Seconds: 3000, Taken: 600, PredTaken: 600, PredKillSec: 600}, 100)
	if v := l.KHeroFor(false); v < 2.5 || v > p.KMax {
		t.Fatalf("a read weapon keeps the full correction up to kMax: %.2f", v)
	}
	if v := l.KHeroFor(true); v != p.KHeroUnreadMax {
		t.Fatalf("an unread weapon is capped at %.1f, got %.2f", p.KHeroUnreadMax, v)
	}
	if v := l.KHero(); v > p.KMax {
		t.Fatalf("k_hero %.2f above kMax %.1f", v, p.KMax)
	}
}

// Both builds: a log without weapon terms must degrade to the plan tables (a zone the hero clears easily is not "too hard").
func TestUnreadOffenceDoesNotShutEveryZone(t *testing.T) {
	for _, tc := range []struct {
		name, arch string
		hero       func(int) power.Hero
	}{{"paladin", "bruiser", paladin}, {"sorceress", "caster", sorceress}} {
		t.Run(tc.name, func(t *testing.T) {
			e := newEnv(t, tc.arch)
			h := tc.hero(14)
			h.MeasuredDPS, h.DmgMin, h.DmgMax = 0, 0, 0
			h.SetQuality(power.TermOffence, power.Default)
			if !h.OffenceUnread() {
				t.Fatal("quality default must read as unread")
			}
			l := NewLedger(e.p)
			k := ZoneKey{2, "normal"}
			// observed kill time 6 times the default model's: all of it the missing weapon
			l.Observe(k, t0, Event{Kind: EvFight, Kills: 600, Seconds: 3600, Taken: 400, PredTaken: 400, PredKillSec: 600}, PowerOf(h))
			in := e.input(h, ZoneKey{}, t0.Add(time.Minute))
			in.OffenceUnread = true
			capped, err := l.Choose(in)
			if err != nil {
				t.Fatal(err)
			}
			in2 := e.input(h, ZoneKey{}, t0.Add(time.Minute))
			uncapped, err := l.Choose(in2)
			if err != nil {
				t.Fatal(err)
			}
			if capped.Hole && !uncapped.Hole {
				t.Fatal("the cap made it worse")
			}
			if kc, ku := l.KHeroFor(true), l.KHeroFor(false); kc > ku || kc > e.p.KHeroUnreadMax {
				t.Fatalf("capped %.2f uncapped %.2f", kc, ku)
			}
		})
	}
}

// swap: merc deaths 40 minutes apart count for the `either` row (window mercWindowMinutes), chickens do not.
func TestMercDeathsAcrossTheWideWindow(t *testing.T) {
	p := params(t)
	k := ZoneKey{3, "normal"}
	l := NewLedger(p)
	l.Observe(k, t0, Event{Kind: EvMercDeath}, 100)
	l.Observe(k, t0.Add(40*time.Minute), Event{Kind: EvMercDeath}, 100)
	if out, why := l.ForcedOut(k, "balanced", t0.Add(41*time.Minute)); !out || why != "hard_events" {
		t.Fatalf("two merc deaths 40 minutes apart must close the zone: %v %q", out, why)
	}
	if l.NoReviveNow(k, t0.Add(41*time.Minute)) {
		t.Fatal("the revive rule keeps its own short window")
	}
	if out, _ := l.ForcedOut(k, "balanced", t0.Add(3*time.Hour)); out {
		t.Fatal("three hours later the zone is open again")
	}
	l2 := NewLedger(p)
	l2.Observe(k, t0, Event{Kind: EvChicken}, 100)
	l2.Observe(k, t0.Add(40*time.Minute), Event{Kind: EvChicken}, 100)
	if out, _ := l2.ForcedOut(k, "balanced", t0.Add(41*time.Minute)); out {
		t.Fatal("chickens keep the 30 minute window")
	}
}

func TestParamsCalibration(t *testing.T) {
	p := params(t)
	if p.KMax != 3 || p.MercPriorMinutes != 25 || p.MercWindowMinutes < p.WindowMinutes || p.PredTakenFloor <= 0 || p.KHeroUnreadMax > p.KMax {
		t.Fatalf("calibration constants: %+v", p)
	}
	for _, k := range []string{"kMax", "kHeroUnreadMax", "predTakenFloorPerKill", "mercWindowMinutes", "mercPriorMinutes"} {
		if p.Tags[k] == "" {
			t.Errorf("no tag for %s", k)
		}
	}
}

// Observed intake per zone: decayed life lost over decayed stay seconds; no data is 0; the row round-trips (additive keys).
func TestLedgerIntake(t *testing.T) {
	l := NewLedger(params(t))
	k := ZoneKey{Area: 2, Diff: "normal"}
	if pm, sec := l.IntakePerMin(k); pm != 0 || sec != 0 {
		t.Fatalf("empty ledger: %v %v", pm, sec)
	}
	l.Observe(k, t0, Event{Kind: EvIntake, LifeLost: 120, Seconds: 600}, 100)
	if pm, sec := l.IntakePerMin(k); pm < 11.9 || pm > 12.1 || sec != 600 {
		t.Errorf("per min %v over %v s, want 12", pm, sec)
	}
	if s := l.peek(k); s.Seconds != 0 || s.Taken != 0 {
		t.Error("intake must not touch the fight sums")
	}
	path := filepath.Join(t.TempDir(), "ledger.json")
	if err := l.Save(path); err != nil {
		t.Fatal(err)
	}
	l2, _, err := LoadLedger(path, params(t))
	if err != nil {
		t.Fatal(err)
	}
	if pm, _ := l2.IntakePerMin(k); pm < 11.9 || pm > 12.1 {
		t.Errorf("reloaded per min %v", pm)
	}
}

// The ledger compares event stamps with the caller's now: both must come from one clock. Under lockstep the bot clock
// counts from the Unix epoch; a wall-clock now made every event look 56 years old (survfix, 2026-11-16).
func TestLedgerNeedsOneClock(t *testing.T) {
	l := NewLedger(params(t))
	k := ZoneKey{3, "normal"}
	virtual := time.Unix(0, 0).Add(10 * time.Minute)
	l.Observe(k, virtual, Event{Kind: EvDeath}, 100)
	if out, _ := l.ForcedOut(k, "balanced", virtual.Add(time.Minute)); !out {
		t.Error("same clock: a death a minute ago must force the zone out")
	}
	if out, _ := l.ForcedOut(k, "balanced", time.Now()); out {
		t.Error("mixed clocks: the event reads as ancient (this is the failure the wiring must avoid)")
	}
}

// L2 (live review 2026-11-28): a hole whose best zone has no margin at all is not a fallback to walk to (Catacombs
// Level 3 at margin 0.0 was "the easiest field" 58 times). A hole with some margin stays the first-field fallback.
func TestHoleWithoutMarginHolds(t *testing.T) {
	for _, tc := range []struct {
		arch string
		hero func(int) power.Hero
	}{{"bruiser", paladin}, {"caster", sorceress}} {
		e := newEnv(t, tc.arch)
		in := e.input(tc.hero(17), ZoneKey{}, t0)
		for i := range in.Cands {
			in.Cands[i].ZM.Bad.Best, in.Cands[i].ZM.Bad.Low, in.Cands[i].ZM.Typical.Low = 0.02, 0, 0
		}
		d, err := NewLedger(e.p).Choose(in)
		if err != nil {
			t.Fatal(err)
		}
		if !d.Hole || !d.NeedsYou || !d.Hold {
			t.Fatalf("%s: margin 0.0 everywhere must be a hole that holds: %+v", tc.arch, d)
		}
		for i := range in.Cands {
			in.Cands[i].ZM.Bad.Best = 0.9
		}
		if d, _ = NewLedger(e.p).Choose(in); !d.Hole || d.Hold {
			t.Fatalf("%s: a hole with margin 0.9 is the first-field fallback, not a hold: %+v", tc.arch, d)
		}
	}
}
