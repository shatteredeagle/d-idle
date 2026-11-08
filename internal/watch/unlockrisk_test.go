package watch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/unlock"
)

// riskState is the knob the tests turn: what the bot would report.
type riskState struct {
	facts RiskFacts
}

func (s *riskState) get(zone string) RiskFacts {
	f := s.facts
	if zone != "" {
		f.Zone = zone
	}
	return f
}

func riskRules(t *testing.T) RiskRules {
	t.Helper()
	pt, err := decide.ShippedProposeTable()
	if err != nil {
		t.Fatal(err)
	}
	rt, err := ShippedRiskTable()
	if err != nil {
		t.Fatal(err)
	}
	return RiskRules{Propose: pt, Table: rt, Ranges: tallyRanges(currency.DefaultTallyRows())}
}

func riskRig(t *testing.T, class string, st *riskState) (*funRig, *unlock.Engine) {
	t.Helper()
	l, err := unlock.Shipped()
	if err != nil {
		t.Fatal(err)
	}
	eng, err := unlock.Open(t.TempDir(), strings.ToLower(class), l, nil)
	if err != nil {
		t.Fatal(err)
	}
	pt, err := decide.ShippedProposeTable()
	if err != nil {
		t.Fatal(err)
	}
	r := newFunRigWith(t, class, 16, func(_ *funRig, fc *FunConfig) {
		fc.Unlock = eng
		fc.Risk = &RiskConfig{Propose: pt, Ranges: tallyRanges(currency.DefaultTallyRows()), Facts: st.get}
	})
	return r, eng
}

func goodFacts() RiskFacts {
	return RiskFacts{Zone: "Forgotten Tower", Kit: "OK", Potions: 5, LifePct: 100, EngagePct: 65, MinEngagePct: 40, Ok: true}
}

// The verdict flips to turn_back exactly where the bot's own guard (task.TowerWhy) or an empty kit refuses, for both
// classes, and the kit and the life lines are the bot's numbers.
func TestRiskVerdictAgainstTheGuards(t *testing.T) {
	rules := riskRules(t)
	rt := env.DefaultRouteTuning()
	for _, class := range []string{"Sorceress", "Paladin"} {
		caster := class == "Sorceress"
		for lvl := 8; lvl <= 22; lvl++ {
			for _, merc := range []bool{false, true} {
				for potions := 0; potions <= 6; potions += 2 {
					for _, tp := range []bool{false, true} {
						name := fmt.Sprintf("%s/clvl%d/merc=%v/potions%d/tp=%v", class, lvl, merc, potions, tp)
						w := task.World{Difficulty: "normal", Level: lvl, HasMerc: merc, Potions: potions, HasTP: tp, Caster: caster,
							TowerMinLevel: rt.TowerMinLevel, TowerMinLevelMerc: rt.TowerMinLevelMerc, TowerNeedsMerc: rt.TowerNeedsMerc != 0,
							TowerMinLife: rt.TowerMinLife, TowerMinLevelNoLife: rt.TowerMinLevelNoLife, MaxLife: 400}
						if caster {
							w.TowerKitMana, w.ManaPotions = rt.TowerKitMana, rt.TowerKitMana
						}
						why := task.TowerWhy(w)
						kit := ladder.KitFor(tp, potions)
						f := goodFacts()
						f.Kit, f.Potions, f.GuardWhy = kit, potions, why
						v, err := rules.Assess(RiskInput{Class: class, Clvl: lvl, Facts: f})
						if err != nil {
							t.Fatalf("%s: %v", name, err)
						}
						wantBack := why != "" || kit == ladder.KitNone
						if (v.Verdict == RiskTurnBack) != wantBack {
							t.Errorf("%s: verdict %s, guard says %q kit %s", name, v.Verdict, why, kit)
						}
						if v.Verdict == RiskReady && (why != "" || kit != ladder.KitOK || potions < rules.Propose.MinPotions) {
							t.Errorf("%s: ready against the guard %q kit %s", name, why, kit)
						}
						if v.Verdict == RiskTurnBack && v.Gauge >= 30 || v.Verdict == RiskMarginal && v.Gauge >= 70 {
							t.Errorf("%s: gauge %d does not fit %s", name, v.Gauge, v.Verdict)
						}
					}
				}
			}
		}
	}
}

func TestRiskVerdictLifeLevelResistAndCloseCalls(t *testing.T) {
	rules := riskRules(t)
	cases := []struct {
		name  string
		class string
		clvl  int
		mod   func(*RiskInput)
		want  string
	}{
		{"all good", "Paladin", 16, func(*RiskInput) {}, RiskReady},
		{"life below the engage line", "Paladin", 16, func(i *RiskInput) { i.Facts.LifePct = 60 }, RiskMarginal},
		{"life at the engage line", "Paladin", 16, func(i *RiskInput) { i.Facts.LifePct = 65 }, RiskReady},
		{"life below the fight line", "Paladin", 16, func(i *RiskInput) { i.Facts.LifePct = 39 }, RiskTurnBack},
		{"too few potions", "Paladin", 16, func(i *RiskInput) { i.Facts.Potions = 1 }, RiskMarginal},
		{"thin kit", "Paladin", 16, func(i *RiskInput) { i.Facts.Kit = "THIN" }, RiskMarginal},
		{"resist gap", "Paladin", 16, func(i *RiskInput) { i.Facts.ResistGap = 10 }, RiskMarginal},
		{"two close calls", "Paladin", 16, func(i *RiskInput) { i.CloseCalls = 2 }, RiskMarginal},
		{"one close call", "Paladin", 16, func(i *RiskInput) { i.CloseCalls = 1 }, RiskReady},
		{"sorceress lead: paladin minimum is not enough", "Sorceress", 15, func(*RiskInput) {}, RiskMarginal},
		{"sorceress lead met", "Sorceress", 16, func(*RiskInput) {}, RiskReady},
		{"paladin at the minimum", "Paladin", 15, func(*RiskInput) {}, RiskReady},
	}
	for _, c := range cases {
		in := RiskInput{Class: c.class, Clvl: c.clvl, Facts: goodFacts()}
		c.mod(&in)
		v, err := rules.Assess(in)
		if err != nil || v.Verdict != c.want {
			t.Errorf("%s: %s %v (reasons %v), want %s", c.name, v.Verdict, err, v.Reasons, c.want)
		}
	}
}

func TestRiskUnknownClassAndZoneAreErrors(t *testing.T) {
	rules := riskRules(t)
	if _, err := rules.Assess(RiskInput{Class: "Necromancer", Clvl: 16, Facts: goodFacts()}); err == nil || !strings.Contains(err.Error(), `"Necromancer"`) {
		t.Errorf("class: %v", err)
	}
	f := goodFacts()
	f.Zone = "Moo Moo Farm"
	if _, err := rules.Assess(RiskInput{Class: "Paladin", Clvl: 16, Facts: f}); err == nil || !strings.Contains(err.Error(), `"Moo Moo Farm"`) {
		t.Errorf("zone: %v", err)
	}
}

func TestRiskTableErrors(t *testing.T) {
	for _, bad := range []string{
		`{"schema":"x"}`,
		`{"schema":"watch.risk/1","nearDeathPct":60,"recoverPct":50,"closeCallsMarginal":2,"closeCallWindowMin":30,"keep":20}`,
		`{"schema":"watch.risk/1","nearDeathPct":25,"recoverPct":50,"closeCallsMarginal":0,"closeCallWindowMin":30,"keep":20}`,
		`{"schema":"watch.risk/1","nearDeathPct":25,"recoverPct":50,"closeCallsMarginal":2,"closeCallWindowMin":30,"keep":20,"extra":1}`,
	} {
		if _, err := ParseRiskTable([]byte(bad)); err == nil {
			t.Errorf("accepted %s", bad)
		}
	}
}

func riskOf(r *funRig, q string) RiskView {
	var v RiskView
	r.getJSON(RiskRoute+q, &v)
	return v
}

func TestRiskUnlockByNearDeathGoldenTwoClasses(t *testing.T) {
	for _, class := range []string{"Sorceress", "Paladin"} {
		t.Run(class, func(t *testing.T) {
			st := &riskState{facts: goodFacts()}
			r, eng := riskRig(t, class, st)
			if v := riskOf(r, ""); v.Open || v.Teaser == "" || v.Verdict != nil {
				t.Fatalf("locked view: %+v", v)
			}
			// a low dip that ends in death opens nothing
			st.facts.LifePct = 10
			r.fun.Settle()
			st.facts.Dead = true
			r.fun.Settle()
			st.facts.Dead, st.facts.LifePct = false, 100
			r.fun.Settle()
			if eng.Allowed(RiskID) {
				t.Fatal("a death opened the risk meter")
			}
			// a dip that is not low enough does not count either
			st.facts.LifePct = 30
			r.fun.Settle()
			st.facts.LifePct = 100
			r.fun.Settle()
			if eng.Allowed(RiskID) {
				t.Fatal("30% opened the risk meter")
			}
			st.facts.LifePct = 12 // near death
			r.fun.Settle()
			if eng.Allowed(RiskID) {
				t.Fatal("opened before the hero recovered")
			}
			st.facts.LifePct = 45 // not recovered yet
			r.fun.Settle()
			if eng.Allowed(RiskID) {
				t.Fatal("opened at 45%")
			}
			st.facts.LifePct = 70
			r.fun.Settle()
			if !eng.Allowed(RiskID) {
				t.Fatal("near-death survived did not open the risk meter")
			}
			found := false
			for _, l := range storyLines(r) {
				found = found || strings.Contains(l, "Unlocked: Risk meter.")
			}
			if !found {
				t.Fatalf("no reveal line: %v", storyLines(r))
			}
			v := riskOf(r, "")
			if !v.Open || v.Verdict == nil || v.Closest == nil || v.Closest.LowPct != 12 || v.Verdict.Verdict != RiskReady {
				t.Fatalf("view: %+v", v)
			}
			b, _ := json.MarshalIndent(v, "", "  ")
			golden(t, "risk-"+strings.ToLower(class), string(b)+"\n")
			// a replay: the same recovery again records a second close call but reveals nothing twice
			st.facts.LifePct = 5
			r.fun.Settle()
			st.facts.LifePct = 90
			r.fun.Settle()
			n := 0
			for _, l := range storyLines(r) {
				if strings.Contains(l, "Unlocked: Risk meter.") {
					n++
				}
			}
			if n != 1 {
				t.Errorf("%d reveal lines", n)
			}
			if v := riskOf(r, ""); v.Recent != 2 || v.Closest.LowPct != 5 || v.Verdict.Verdict != RiskMarginal {
				t.Errorf("two close calls should make it marginal: %+v", v)
			}
			if errs := r.fun.Risk.Errors(); len(errs) != 0 {
				t.Error(errs)
			}
		})
	}
}

func TestRiskRouteZoneGuardAndMethod(t *testing.T) {
	st := &riskState{facts: goodFacts()}
	r, eng := riskRig(t, "Paladin", st)
	eng.Observe(unlock.Event{Type: unlock.EvSeen, Name: RiskSeen})
	st.facts.GuardWhy = "no merc, clvl 12 < 15"
	v := riskOf(r, "?zone=Forgotten%20Tower")
	if v.Verdict.Verdict != RiskTurnBack || v.Verdict.Zone != "Forgotten Tower" || !strings.Contains(strings.Join(v.Verdict.Reasons, "|"), "no merc") {
		t.Fatalf("%+v", v.Verdict)
	}
	if v := riskOf(r, "?zone=Nowhere"); v.Error == "" || v.Verdict != nil || !strings.Contains(v.Error, `"Nowhere"`) {
		t.Fatalf("unknown zone: %+v", v)
	}
	if resp, _ := r.do("POST", r.view, RiskRoute, funToken, "{}"); resp.StatusCode != http.StatusMethodNotAllowed && resp.StatusCode != http.StatusNotFound {
		t.Errorf("POST: %d", resp.StatusCode)
	}
	if resp, _ := r.do("GET", r.view, RiskRoute, "", ""); resp.StatusCode == 200 {
		t.Error("the risk route answered without the viewer token")
	}
	st.facts.Ok = false
	if v := riskOf(r, ""); v.Error == "" || v.Verdict != nil {
		t.Errorf("unread facts: %+v", v)
	}
}

func TestRiskMissingRowIsAnError(t *testing.T) {
	l, _ := unlock.Parse([]byte(`{"schema":"unlock.ladder/1","entries":[
 {"id":"chronicle","group":"ui","trigger":{"kind":"kills_of_kind","args":{"kind":"*","min":1}},"reveal":{"title":"C","text":"c"}}]}`))
	eng, _ := unlock.Open("", "p", l, nil)
	pt, _ := decide.ShippedProposeTable()
	_, err := NewRiskMeter(&Unlocks{Engine: eng}, nil, RiskConfig{Propose: pt, Ranges: tallyRanges(currency.DefaultTallyRows()), Facts: func(string) RiskFacts { return RiskFacts{} }})
	if err == nil || !strings.Contains(err.Error(), `"risk"`) {
		t.Fatalf("%v", err)
	}
}

func TestRiskReturnSectionClosestCall(t *testing.T) {
	st := &riskState{facts: goodFacts()}
	r, _ := riskRig(t, "Sorceress", st)
	st.facts.LifePct = 8
	r.fun.Settle()
	st.facts.LifePct = 80
	r.fun.Settle()
	var d ReturnData
	r.getJSON("/api/return", &d)
	for _, s := range d.Sections {
		if s.ID == RiskSectionID {
			if len(s.Lines) != 1 || !strings.Contains(s.Lines[0], "8%") {
				t.Fatalf("section: %+v", s)
			}
			return
		}
	}
	t.Fatalf("no closest call section: %+v", d.Sections)
}
