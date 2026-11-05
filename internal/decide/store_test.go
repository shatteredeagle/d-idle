package decide

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestStreakTable(t *testing.T) {
	cases := []struct {
		name    string
		point   string
		cat     map[string]string
		answer  string
		n       int
		confirm bool
	}{
		{"safe 2", "t1.restock", nil, "no", 2, false},
		{"normal 3", "t0.retreat.drink", nil, "run", 3, false},
		{"risky 5 confirms", "t2.bossunsafe", nil, "go", 5, true},
		{"hardcore drink is risky", "t0.retreat.drink", map[string]string{"hc": "hardcore"}, "run", 5, true},
		{"return kit OK is safe", "t1.return", map[string]string{"kit": "OK"}, "no", 2, false},
		{"return kit NONE is normal", "t1.return", map[string]string{"kit": "NONE"}, "yes", 3, false},
		{"keep rune is normal", "t1.keep.kind", map[string]string{"itemkind": "rune"}, "sell", 3, false},
		{"keep unique is risky", "t1.keep.kind", map[string]string{"itemkind": "unique"}, "sell", 5, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			r := newRig(t)
			c := r.ctx(tc.point, tc.cat, nil)
			n, err := r.cat.N(c)
			if err != nil || n != tc.n {
				t.Fatalf("N = %d, %v; want %d", n, err, tc.n)
			}
			for i := 1; i < tc.n; i++ {
				r.say(c, tc.answer)
				if r.learned(c) {
					t.Fatalf("learned after %d answers", i)
				}
			}
			r.say(c, tc.answer)
			if tc.confirm {
				id := r.pendingConfirm(c)
				if id == 0 || r.learned(c) {
					t.Fatal("risky point must open a confirm and not learn yet")
				}
				if err := r.s.Confirm(id, true); err != nil {
					t.Fatal(err)
				}
			}
			if !r.learned(c) {
				t.Fatal("rule not learned")
			}
			d := r.ask(c)
			if d.Source != "rule" || d.Answer != tc.answer || d.Thought != nil {
				t.Fatalf("decision after learning = %+v", d)
			}
		})
	}
}

func TestDifferentAnswerResets(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t0.retreat.drink", nil, nil)
	r.say(c, "run")
	r.say(c, "run")
	r.say(c, "drink")
	if rec := r.rec(c); rec.Streak != 1 || rec.Last != "drink" {
		t.Fatalf("after reset: %+v", rec)
	}
	r.say(c, "drink")
	r.say(c, "drink")
	if !r.learned(c) || r.rec(c).Rule.Answer != "drink" {
		t.Fatal("expected drink rule")
	}
}

func TestNewContextNewStreak(t *testing.T) {
	r := newRig(t)
	a := r.ctx("t1.restock", map[string]string{"belt": "low"}, nil)
	b := r.ctx("t1.restock", map[string]string{"belt": "half"}, nil)
	d := r.ctx("t1.restock", map[string]string{"act": "2", "belt": "low"}, nil)
	r.say(a, "yes")
	r.say(b, "yes")
	r.say(d, "yes")
	if r.learned(a) || r.learned(b) || r.learned(d) {
		t.Fatal("different keys must not share a streak")
	}
	r.say(a, "yes")
	if !r.learned(a) || r.learned(b) || r.learned(d) {
		t.Fatal("only the key that reached N learns")
	}
}

func TestFadeNeitherCountsNorResets(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t0.retreat.drink", nil, nil)
	r.say(c, "run")
	r.say(c, "run") // streak 2
	d := r.ask(c)   // a third thought opens; nobody answers
	r.clk.Advance(fade)
	if n, err := r.s.Tick(); err != nil || n != 1 {
		t.Fatalf("tick faded %d, %v", n, err)
	}
	if rec := r.rec(c); rec.Streak != 2 || rec.Last != "run" {
		t.Fatalf("fade changed the streak: %+v", rec)
	}
	if err := r.s.Answer(d.Thought.ID, "run"); !errors.Is(err, ErrNoThought) {
		t.Fatalf("answer to a faded thought: %v", err)
	}
	r.say(c, "run") // a new thought for the next occurrence
	if !r.learned(c) {
		t.Fatal("third counted answer should learn")
	}
}

func TestLateAnswerIsAFade(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t1.restock", nil, nil)
	d := r.ask(c)
	r.clk.Advance(fade)
	if err := r.s.Answer(d.Thought.ID, "yes"); !errors.Is(err, ErrFaded) {
		t.Fatalf("late answer: %v", err)
	}
	if _, ok := r.s.Record(c.Key); ok && r.rec(c).Streak != 0 {
		t.Fatal("late answer counted")
	}
	_, _, faded := r.s.Stats()
	if faded != 1 {
		t.Fatalf("faded = %d", faded)
	}
}

func TestNotSure(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t0.retreat.drink", nil, nil)
	r.say(c, "run")
	r.say(c, NotSure)
	if rec := r.rec(c); rec.Streak != 1 || rec.Last != "run" {
		t.Fatalf("notsure changed the streak: %+v", rec)
	}
	r.say(c, "run")
	r.say(c, NotSure)
	r.say(c, "run")
	if !r.learned(c) {
		t.Fatal("notsure must not block learning")
	}
	// the default is used for the hero while unsure
	c2 := r.ctx("t0.retreat.lethal", nil, nil)
	d := r.ask(c2)
	if d.Answer != "avoid" || d.Source != "default" {
		t.Fatalf("%+v", d)
	}
}

func TestConfirmFlow(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t2.bossunsafe", nil, nil)
	reach := func() int {
		for i := 0; i < 5; i++ {
			r.say(c, "go")
		}
		id := r.pendingConfirm(c)
		if id == 0 {
			t.Fatal("no confirm thought")
		}
		return id
	}
	// no: unlearned, streak reset
	if err := r.s.Confirm(reach(), false); err != nil {
		t.Fatal(err)
	}
	if rec := r.rec(c); r.learned(c) || rec.Streak != 0 || rec.Last != "" || rec.Pending != "" {
		t.Fatalf("after no: %+v", rec)
	}
	// fade: unlearned, nothing pending, streak kept
	id := reach()
	_ = id
	r.clk.Advance(fade)
	r.s.Tick()
	if rec := r.rec(c); r.learned(c) || rec.Pending != "" {
		t.Fatalf("after fade: %+v", rec)
	}
	// a faded confirm asks again on the next answer
	r.say(c, "go")
	if r.pendingConfirm(c) == 0 {
		t.Fatal("expected a new confirm")
	}
	// while a confirm is open there is no second ask thought
	if d := r.ask(c); d.Thought != nil {
		t.Fatal("ask thought opened during pending confirm")
	}
	if err := r.s.Answer(r.pendingConfirm(c), "go"); !errors.Is(err, ErrWrongKind) {
		t.Fatalf("answer to a confirm thought: %v", err)
	}
	if err := r.s.Confirm(r.pendingConfirm(c), true); err != nil || !r.learned(c) {
		t.Fatalf("yes: %v", err)
	}
}

func TestRevokeAndNoSelfRevoke(t *testing.T) {
	r := newRig(t)
	a := r.ctx("t1.restock", nil, nil)
	b := r.ctx("t1.repair", nil, nil)
	for _, c := range []Context{a, b} {
		for i := 0; i < 2; i++ {
			r.say(c, map[string]string{"t1.restock": "yes", "t1.repair": "skip"}[c.Point])
		}
	}
	if !r.learned(a) || !r.learned(b) {
		t.Fatal("setup: both rules should be learned")
	}
	// time passes, other things happen: no rule revokes itself
	r.clk.Advance(100 * fade)
	r.s.Tick()
	if len(r.s.Rules()) != 2 {
		t.Fatal("rules list")
	}
	if err := r.s.Revoke("t1.nope|x=y"); !errors.Is(err, ErrNoRule) {
		t.Fatalf("revoke of an unknown key: %v", err)
	}
	if err := r.s.Revoke(a.Key); err != nil {
		t.Fatal(err)
	}
	if r.learned(a) || !r.learned(b) {
		t.Fatal("revoke one revoked the wrong rules")
	}
	if rec := r.rec(a); rec.Streak != 0 || rec.Rule.RevokedBy != "player" {
		t.Fatalf("%+v", rec)
	}
	// after a revoke the hero asks again and the streak restarts at 0
	if d := r.ask(a); d.Source != "default" || d.Thought == nil {
		t.Fatalf("after revoke: %+v", d)
	}
	r.say(a, "yes")
	r.say(a, "yes")
	if !r.learned(a) {
		t.Fatal("relearn after revoke")
	}
	if err := r.s.RevokeAll(); err != nil {
		t.Fatal(err)
	}
	if r.learned(a) || r.learned(b) {
		t.Fatal("revoke all")
	}
	if rules, _, _ := r.s.Stats(); rules != 0 {
		t.Fatalf("live rules = %d", rules)
	}
	for _, v := range r.s.Rules() {
		if !v.Revoked {
			t.Fatal("rules list should mark revoked rules")
		}
	}
}

func TestSacrificeSkip(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t0.retreat.drink", nil, nil)
	r.say(c, "run")
	if err := r.s.Sacrifice("t0.retreat.drink"); err != nil {
		t.Fatal(err)
	}
	r.s.Sacrifice("t0.retreat.drink") // bounded by maxSkips 1
	r.say(c, "run")                   // counts 1 + 1 skip
	if rec := r.rec(c); rec.Streak != 3 || !r.learned(c) {
		t.Fatalf("skip: %+v", rec)
	}
	if err := r.s.Sacrifice("nope"); err == nil || !strings.Contains(err.Error(), "nope") {
		t.Fatalf("unknown point: %v", err)
	}
}

func TestAskNeverWaits(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t0.retreat.drink", map[string]string{"kit": "NONE"}, map[string]float64{"life": 40})
	d := r.ask(c)
	if d.Answer != "run" || d.Thought == nil || d.Source != "default" {
		t.Fatalf("%+v", d)
	}
	if d2 := r.ask(c); d2.Thought == nil || d2.Thought.ID != d.Thought.ID {
		t.Fatal("an open thought is reused, not duplicated")
	}
	if len(r.s.OpenThoughts()) != 1 {
		t.Fatal("one open thought expected")
	}
}

func TestTierLocked(t *testing.T) {
	r := newRig(t)
	r.s.SetUnlockedTier(0)
	d := r.ask(r.ctx("t1.restock", nil, map[string]float64{"tripgold": 300}))
	if d.Source != "default-locked" || d.Thought != nil || d.Answer != "yes" {
		t.Fatalf("%+v", d)
	}
	if d := r.ask(r.ctx("t0.retreat.lethal", nil, nil)); d.Thought == nil {
		t.Fatal("tier 0 should ask")
	}
}

func TestIrreversibleDefaultsDoNothing(t *testing.T) {
	cat := mustCat(t)
	n := 0
	for _, p := range cat.Points {
		if !p.Irreversible {
			continue
		}
		n++
		for _, d := range p.Default {
			if a, _ := p.answer(d.Answer); !a.Noop {
				t.Errorf("%s default %q does something", p.ID, d.Answer)
			}
		}
	}
	if n == 0 {
		t.Fatal("no irreversible point in the catalogue")
	}
	bad := strings.Replace(string(shippedCatalogue), `"default": [{"when": {}, "answer": "save"}]`, `"default": [{"when": {}, "answer": "gamble"}]`, 1)
	if _, err := ParseCatalogue([]byte(bad)); err == nil || !strings.Contains(err.Error(), "t1.gamble") {
		t.Fatalf("irreversible default that acts must be refused, got %v", err)
	}
}

func TestEvents(t *testing.T) {
	r := newRig(t)
	var kinds []EventKind
	r.s.Subscribe(func(e Event) { kinds = append(kinds, e.Kind) })
	c := r.ctx("t1.restock", nil, nil)
	r.say(c, "yes")
	r.say(c, "yes")
	want := []EventKind{EvThoughtOpened, EvThoughtOpened, EvRuleLearned}
	if len(kinds) != 3 || kinds[0] != want[0] || kinds[2] != want[2] {
		t.Fatalf("events %v", kinds)
	}
}

func TestRulesListPlainLanguage(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t1.restock", nil, nil)
	r.say(c, "yes")
	r.say(c, "yes")
	v := r.s.Rules()
	if len(v) != 1 || v[0].Label != "restock" || !strings.Contains(v[0].Context, "tripgold 60-300") || v[0].Question == "" {
		t.Fatalf("%+v", v)
	}
	if _, err := json.Marshal(v); err != nil {
		t.Fatal(err)
	}
}

func TestRebirthKeepsRules(t *testing.T) {
	r := newRig(t)
	c := r.ctx("t1.restock", nil, nil)
	r.say(c, "yes")
	r.say(c, "yes")
	r.say(r.ctx("t1.repair", nil, nil), "skip") // leaves a streak of 1
	open := r.ask(r.ctx("t1.return", nil, nil))
	r.s.SetQuality("altar", 3)
	if err := r.s.Rebirth("hero2"); err != nil {
		t.Fatal(err)
	}
	if !r.learned(c) || r.s.Quality("altar") != 3 {
		t.Fatal("rebirth lost rules or quality")
	}
	if err := r.s.Answer(open.Thought.ID, "yes"); !errors.Is(err, ErrNoThought) {
		t.Fatalf("thoughts of the dead hero must be gone: %v", err)
	}
	if rec := r.rec(r.ctx("t1.repair", nil, nil)); rec.Streak != 1 {
		t.Fatalf("streaks are per profile: %+v", rec)
	}
	if log := r.s.Log(); log[len(log)-1].Hero != "hero2" {
		t.Fatal("log lines carry the hero")
	}
}
