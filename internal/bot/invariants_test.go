package bot

import (
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

// Two heroes with different numbers: the rows take their lines from the facts, so nothing names a class.
type invHero struct {
	name                       string
	maxLife, drinkPct, chicken int
	kitMissing                 []string
}

var invHeroes = []invHero{
	{"sorc", 176, 55, 25, []string{"town portal", "mana potions 0/4"}},
	{"pala", 520, 45, 30, []string{"healing potions 0/3"}},
}

func (h invHero) facts(now time.Time) invFacts {
	return invFacts{now: now, area: 3, life: h.maxLife, maxLife: h.maxLife, lifePct: 100, xp: 1000, kills: 10, gold: 5000,
		beltHP: 4, tp: 2, state: string(StateClearing), step: "clear", drinkPct: h.drinkPct, chickenPct: h.chicken}
}

// run feeds facts once a second for n seconds, mutating them with edit, and returns the names that fired.
func runInv(m *invMon, f invFacts, n int, edit func(*invFacts, int)) map[string]int {
	got := map[string]int{}
	for i := 0; i < n; i++ {
		f.now = f.now.Add(time.Second)
		if edit != nil {
			edit(&f, i)
		}
		for _, h := range m.tick(f) {
			got[h.Name]++
		}
	}
	return got
}

var t0 = time.Date(2026, 11, 25, 12, 0, 0, 0, time.UTC)

func TestInvariantTableComplete(t *testing.T) {
	seen := map[string]bool{}
	for _, r := range invRows {
		if seen[r.Name] || r.Why == "" {
			t.Fatalf("row %q duplicated or without a why", r.Name)
		}
		seen[r.Name] = true
	}
	if len(invRows) < 12 {
		t.Fatalf("%d rows", len(invRows))
	}
}

func TestInvariantHealthyHeroFiresNothing(t *testing.T) {
	for _, h := range invHeroes {
		m := &invMon{}
		if got := runInv(m, h.facts(t0), 3000, func(f *invFacts, i int) { f.xp += uint32(i); f.step = fmt.Sprint("clear ", i/100) }); len(got) != 0 {
			t.Errorf("%s: healthy hero fired %v", h.name, got)
		}
	}
}

func TestInvariantMustFail(t *testing.T) {
	cases := []struct {
		name string
		n    int
		edit func(h invHero) func(*invFacts, int)
		want string
	}{
		{"low life, potions, none drunk", 20, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, _ int) { f.life, f.lifePct = h.maxLife/5, 20 }
		}, "potion-not-drunk"},
		{"chicken line, scrolls, hostiles", 20, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, _ int) { f.lifePct, f.hostiles, f.life = h.chicken-1, 3, 1 }
		}, "portal-not-cast"},
		{"stuck in a pack at half life", 20, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, _ int) { f.lifePct, f.life, f.hostiles = 50, h.maxLife/2, 7 }
		}, "stuck-in-pack"},
		{"resupplying in a field", 30, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, _ int) { f.state, f.step = string(StateResupplying), "chores" }
		}, "resupply-in-field"},
		{"life zero while alive", 10, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, _ int) { f.life, f.lifePct = 0, 0; f.inTown = true }
		}, "life-zero-alive"},
		{"xp frozen in the field", 2000, func(h invHero) func(*invFacts, int) { return nil }, "xp-frozen"},
		{"mode without exit", 1500, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, _ int) { f.xp = 1000; f.inTown = true }
		}, "mode-no-exit"},
		{"corpse never resolved", 5, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, _ int) { f.corpsePending, f.corpseSince = true, t0.Add(-time.Hour) }
		}, "corpse-unresolved"},
		{"poor for two hours while killing", 7300, func(h invHero) func(*invFacts, int) {
			return func(f *invFacts, i int) { f.gold, f.kills = 3, 10+i; f.xp += 1 }
		}, "economy-stall"},
	}
	for _, h := range invHeroes {
		for _, c := range cases {
			m := &invMon{}
			if got := runInv(m, h.facts(t0), c.n, c.edit(h)); got[c.want] == 0 {
				t.Errorf("%s/%s: %s did not fire (%v)", h.name, c.name, c.want, got)
			}
		}
	}
}

func TestInvariantDrinkClearsIt(t *testing.T) {
	h := invHeroes[0]
	m := &invMon{}
	got := runInv(m, h.facts(t0), 60, func(f *invFacts, i int) { f.life, f.lifePct = 40, 20; f.drank = int64(i / 3) })
	if got["potion-not-drunk"] != 0 {
		t.Fatalf("a hero that drinks every 3 s fired: %v", got)
	}
}

func TestInvariantFieldEntry(t *testing.T) {
	for _, h := range invHeroes {
		m := &invMon{}
		f := h.facts(t0)
		f.inTown = true
		runInv(m, f, 3, nil)
		f.inTown, f.lifePct, f.life = false, 30, h.maxLife*3/10
		f.kitKnown, f.kitMissing = true, h.kitMissing
		var names []string
		for _, x := range m.tick(f) {
			names = append(names, x.Name)
		}
		if strings.Join(names, ",") != "field-entry-hurt,field-entry-no-kit" {
			t.Errorf("%s: %v", h.name, names)
		}
	}
}

func TestInvariantTownThrash(t *testing.T) {
	m := &invMon{}
	f := invHeroes[1].facts(t0)
	got := runInv(m, f, 40*60, func(f *invFacts, i int) { f.inTown = i%120 < 60 })
	if got["town-thrash"] == 0 {
		t.Fatalf("20 trips in 40 min did not fire: %v", got)
	}
}

func TestInvariantRefusals(t *testing.T) {
	m := &invMon{}
	f := invHeroes[0].facts(t0)
	runInv(m, f, 2, nil)
	for i := 0; i < 10; i++ {
		m.noteRefusal()
	}
	m.noteRefusalFailure("quest board step", errors.New("not ready: early road"))
	var names []string
	f.now = f.now.Add(time.Second)
	for _, x := range m.tick(f) {
		names = append(names, x.Name)
	}
	if strings.Join(names, ",") != "refusal-as-failure,refusal-storm" {
		t.Fatalf("%v", names)
	}
	if !isRefusal(fmt.Errorf("x: %w", errAreaGated)) || !isRefusal(errors.New("not ready: kit")) || isRefusal(errors.New("walk failed")) {
		t.Fatal("isRefusal")
	}
}

func TestInvariantRateLimitAndMinute(t *testing.T) {
	m := &invMon{}
	f := invHeroes[0].facts(t0)
	got := runInv(m, f, 300, func(f *invFacts, _ int) { f.life, f.lifePct = 1, 1 })
	if n := got["potion-not-drunk"]; n < 3 || n > 6 {
		t.Fatalf("a 5 minute episode logged %d lines, want one per minute or so", n)
	}
	kv := m.minuteKV()
	if len(kv) != 2 || !strings.Contains(kv[1].(string), "potion-not-drunk=") {
		t.Fatalf("minute kv %v", kv)
	}
	if m.minuteKV() != nil || m.Totals()["potion-not-drunk"] != got["potion-not-drunk"] {
		t.Fatal("minute counts are drained, totals are not")
	}
}

func TestInvariantDeadFiresNothing(t *testing.T) {
	m := &invMon{}
	f := invHeroes[0].facts(t0)
	f.dead = true
	if got := runInv(m, f, 100, func(f *invFacts, _ int) { f.life, f.lifePct = 0, 0 }); len(got) != 0 {
		t.Fatalf("%v", got)
	}
}

// The hook: invTick builds the facts from a snapshot, logs one `invariant` line and calls OnInvariant; knob 0 is silent.
func TestInvTickLogsAndCallsOracle(t *testing.T) {
	for _, knob := range []int{1, 0} {
		g := newRig(t)
		buf := &lockedBuf{}
		g.e.Log = slog.New(slog.NewTextHandler(buf, nil))
		g.e.Tune.Bot.Invariants = knob
		var hits []string
		g.r.d.OnInvariant = func(n, _ string) { hits = append(hits, n) }
		snap := &d2.Snapshot{Player: &d2.Player{Area: 3, Life: 0, MaxLife: 100, Level: 5, Mode: 1}}
		for i := 0; i < 5; i++ {
			g.clock.Sleep(time.Second)
			g.r.invTick(snap, control{})
		}
		out := buf.String()
		if knob == 1 && (!strings.Contains(out, "invariant life-zero-alive") || len(hits) != 1 || g.r.InvariantHits()["life-zero-alive"] != 1) {
			t.Fatalf("knob 1: hits %v log %q", hits, out)
		}
		if knob == 0 && (strings.Contains(out, "invariant") || len(hits) != 0) {
			t.Fatalf("knob 0 logged: %v %q", hits, out)
		}
	}
}

func TestInvariantCountersHardAndSince(t *testing.T) {
	var m invMon
	if len(m.Counters()) != 0 {
		t.Fatal("counters before the first tick")
	}
	for _, h := range invHeroes {
		var mm invMon
		f := h.facts(t0)
		f.life, f.lifePct, f.beltHP = 1, 1, 4
		runInv(&mm, f, 20, nil) // potion-not-drunk fires for every hero
		hard := map[string]bool{}
		var got *InvCounter
		cs := mm.Counters()
		for i := range cs {
			if cs[i].Hard {
				hard[cs[i].Name] = true
			}
			if cs[i].Name == "potion-not-drunk" {
				got = &cs[i]
			}
		}
		for _, n := range []string{"potion-not-drunk", "life-zero-alive", "refusal-storm", "xp-frozen", "mode-no-exit", "corpse-unresolved"} {
			if !hard[n] {
				t.Errorf("%s: %s is not hard", h.name, n)
			}
		}
		if hard["town-thrash"] || len(hard) != 6 {
			t.Errorf("%s: hard set %v", h.name, hard)
		}
		if got == nil || got.Count == 0 || got.Since.IsZero() || got.First.IsZero() || got.Last.Before(got.First) {
			t.Errorf("%s: potion-not-drunk counter %+v", h.name, got)
		}
	}
}

// Questy 2026-11-26: 37 minutes of healthy levelling in one step ("Quest: fighting (Level up to 13)", xp +6300) fired mode-no-exit
// 19 times. A mode whose progress signals move never fires; one that is stuck does; a chore state has a ten minute cap.
func TestModeNoExitNeedsNoProgress(t *testing.T) {
	for _, h := range invHeroes {
		for _, c := range []struct {
			name  string
			n     int
			state string
			edit  func(f *invFacts, i int)
			fires bool
		}{
			{"levelling, xp rising", 2400, string(StateClearing), func(f *invFacts, i int) {
				f.xp += uint32(i / 7)
				f.kills += i / 30
				f.step = "Quest: fighting (Level up to 13)"
			}, false},
			{"levelling, only gold moving", 2400, string(StateClearing), func(f *invFacts, i int) { f.xp, f.gold = 1000, 5000+i/50; f.step = "farm" }, false},
			{"stuck, nothing moves", 1500, string(StateClearing), func(f *invFacts, i int) { f.xp = 1000; f.step = "Quest: fighting (Level up to 13)" }, true},
			{"chore state past ten minutes", 700, string(StateResupplying), func(f *invFacts, i int) { f.xp += uint32(i); f.step = "chores" }, true},
			{"chore state under ten minutes", 500, string(StateResupplying), func(f *invFacts, i int) { f.xp += uint32(i); f.step = "chores" }, false},
		} {
			m := &invMon{}
			f := h.facts(t0)
			f.state = c.state
			f.inTown = c.state == string(StateResupplying)
			got := runInv(m, f, c.n, c.edit)
			if (got["mode-no-exit"] > 0) != c.fires {
				t.Errorf("%s/%s: mode-no-exit fired %d, want fires=%v", h.name, c.name, got["mode-no-exit"], c.fires)
			}
		}
	}
}

// aura-off: a hero with an aura skill learned whose state reads 0 in the field for 10 s; an active aura, a hero without an
// aura skill and a hero in town fire nothing. Both heroes (the rows read facts, not classes).
func TestInvariantAuraOff(t *testing.T) {
	for _, h := range invHeroes {
		off := func(f *invFacts, _ int) { f.auraWatch, f.aura = true, 0 }
		if got := runInv(&invMon{}, h.facts(t0), 20, off); got["aura-off"] == 0 {
			t.Errorf("%s: aura-off did not fire: %v", h.name, got)
		}
		for name, edit := range map[string]func(*invFacts, int){
			"aura up":       func(f *invFacts, _ int) { f.auraWatch, f.aura = true, 98 },
			"no aura skill": func(f *invFacts, _ int) { f.auraWatch, f.aura = false, 0 },
			"in town":       func(f *invFacts, _ int) { f.auraWatch, f.aura, f.inTown = true, 0, true },
		} {
			if got := runInv(&invMon{}, h.facts(t0), 60, edit); got["aura-off"] != 0 {
				t.Errorf("%s/%s: fired %v", h.name, name, got)
			}
		}
	}
}

// summon-missing: a hero with a summon skill learned, mana for it and hostiles near whose minion count reads 0 for 10 s.
// A minion standing, no summon skill, no mana, no hostiles and town fire nothing. Both heroes (the rows read facts).
func TestInvariantSummonMissing(t *testing.T) {
	for _, h := range invHeroes {
		fight := func(f *invFacts, _ int) { f.summonWatch, f.minions, f.hostiles = true, 0, 3 }
		if got := runInv(&invMon{}, h.facts(t0), 20, fight); got["summon-missing"] == 0 {
			t.Errorf("%s: summon-missing did not fire: %v", h.name, got)
		}
		for name, edit := range map[string]func(*invFacts, int){
			"minion up":   func(f *invFacts, _ int) { f.summonWatch, f.minions, f.hostiles = true, 2, 3 },
			"no skill":    func(f *invFacts, _ int) { f.summonWatch, f.minions, f.hostiles = false, 0, 3 },
			"no hostiles": func(f *invFacts, _ int) { f.summonWatch, f.minions, f.hostiles = true, 0, 0 },
			"in town":     func(f *invFacts, _ int) { f.summonWatch, f.minions, f.hostiles, f.inTown = true, 0, 3, true },
		} {
			if got := runInv(&invMon{}, h.facts(t0), 60, edit); got["summon-missing"] != 0 {
				t.Errorf("%s/%s: fired %v", h.name, name, got)
			}
		}
	}
}

// summonWatch reads the learned skills through the data column, so a Necromancer (Raise Skeleton) and a Druid (Raven) qualify
// alike and a Sorceress does not; mana below the cost and an unknown count switch the watch off.
func TestSummonWatchIsData(t *testing.T) {
	skills := map[int]*data.Skill{
		70:  {ID: 70, Key: "Raise Skeleton", Summon: "skeleton", Mana: 6, ManaShift: 8},
		221: {ID: 221, Key: "Raven", Summon: "raven", Mana: 20, ManaShift: 8},
		36:  {ID: 36, Key: "Fire Bolt", Mana: 3},
		99:  {ID: 99, Key: "Bone Mastery", Summon: "x", Passive: true},
	}
	by := func(id int) *data.Skill { return skills[id] }
	for _, c := range []struct {
		name  string
		skill int
		mana  int
		known bool
		want  bool
	}{
		{"necromancer", 70, 30, true, true}, {"druid", 221, 30, true, true},
		{"sorceress", 36, 30, true, false}, {"passive", 99, 30, true, false},
		{"no mana", 70, 1, true, false}, {"count unknown", 70, 30, false, false},
	} {
		p := &d2.Player{Mana: c.mana, PetsKnown: c.known, Pets: 0, Skills: []d2.Skill{{ID: c.skill, Level: 1}}}
		if got, _ := summonWatch(p, by); got != c.want {
			t.Errorf("%s: watch=%v want %v", c.name, got, c.want)
		}
	}
}

// Questy 2026-11-26 (deploy 83): the hero walked between the camp and a dry Blood Moor for 28 minutes, xp, kills and gold frozen;
// the area changes reset mode-no-exit. no-earn (soft) fires after five minutes without a change of xp, kills or gold, whatever the
// area does; earning, a pause or a parked hero keep it quiet. Two heroes.
func TestNoEarnIgnoresAreaChanges(t *testing.T) {
	for _, h := range invHeroes {
		for _, c := range []struct {
			name  string
			n     int
			edit  func(f *invFacts, i int)
			fires bool
		}{
			{"walking between two areas, nothing earned", 400, func(f *invFacts, i int) { f.area = 1 + i%2; f.xp, f.kills, f.gold = 1000, 10, 19 }, true},
			{"earning gold", 400, func(f *invFacts, i int) { f.area = 1 + i%2; f.gold = 19 + i/30 }, false},
			{"parked on purpose", 400, func(f *invFacts, i int) { f.parked = true; f.xp, f.kills, f.gold = 1000, 10, 19 }, false},
			{"under the five minutes", 250, func(f *invFacts, i int) { f.xp, f.kills, f.gold = 1000, 10, 19 }, false},
		} {
			m := &invMon{}
			got := runInv(m, h.facts(t0), c.n, c.edit)
			if (got["no-earn"] > 0) != c.fires {
				t.Errorf("%s/%s: no-earn fired %d, want fires=%v", h.name, c.name, got["no-earn"], c.fires)
			}
		}
	}
}
