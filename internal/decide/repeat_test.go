package decide

import (
	"testing"
	"time"
)

// The wired points that re-fired in the first end-to-end container run, each on both builds (TASK-137).
var repeatPoints = []string{"t0.area.toohard", "t0.unique.fight", "t1.repair", "t1.restock", "t1.return", "t2.mercenary"}

func (r *rig) rigClass(point, class string) Context {
	r.t.Helper()
	c, err := r.cat.Build(point, baseObs(r.cat, class), 1, nil)
	if err != nil {
		r.t.Fatal(err)
	}
	return c
}

func (r *rig) policy() { r.s.SetRepeatPolicy(DefaultRepeatCooldown, DefaultQuietAfter) }

func (r *rig) otherAnswer(point string, not string) string {
	p, _ := r.cat.Point(point)
	for _, a := range p.Answers {
		if a.ID != not {
			return a.ID
		}
	}
	r.t.Fatal("point has one answer")
	return ""
}

// fire replays the defect: the situation persists, so the decision point fires every minute for the span, and the
// player never answers. It returns how many thoughts were opened.
func (r *rig) fire(c Context, every, span time.Duration) int {
	opened := 0
	r.s.Subscribe(func(e Event) {
		if e.Kind == EvThoughtOpened {
			opened++
		}
	})
	for t := time.Duration(0); t < span; t += every {
		r.ask(c)
		r.clk.Advance(every)
		if _, err := r.s.Tick(); err != nil {
			r.t.Fatal(err)
		}
	}
	return opened
}

// Reproduction (V, the t0 drill and the container run): a fade clears the open thought, so the next firing opens a new
// one. With no policy (the old behaviour) that is one thought per firing once the 45 s fade has passed.
func TestRepeatReproduction(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		for _, p := range repeatPoints {
			r := newRig(t)
			c := r.rigClass(p, class)
			if n := r.fire(c, time.Minute, 10*time.Minute); n != 10 {
				t.Errorf("%s/%s: without a policy %d thoughts in 10 minutes, want 10 (the defect)", p, class, n)
			}
			r = newRig(t)
			r.policy()
			c = r.rigClass(p, class)
			if n := r.fire(c, time.Minute, 10*time.Minute); n != 1 {
				t.Errorf("%s/%s: with the policy %d thoughts in 10 minutes, want 1", p, class, n)
			}
		}
	}
}

// Unanswered, a key asks QuietAfter times, one per cooldown, then stops for good, with one journal event.
func TestRepeatGoesQuiet(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		for _, p := range repeatPoints {
			r := newRig(t)
			r.policy()
			quiet := 0
			r.s.Subscribe(func(e Event) {
				if e.Kind == EvQuiet {
					quiet++
				}
			})
			c := r.rigClass(p, class)
			if n := r.fire(c, time.Minute, 6*time.Hour); n != DefaultQuietAfter {
				t.Errorf("%s/%s: %d thoughts in 6 h, want %d", p, class, n, DefaultQuietAfter)
			}
			if quiet != 1 {
				t.Errorf("%s/%s: %d quiet events, want 1", p, class, quiet)
			}
			if d := r.ask(c); d.Thought != nil || d.Source == "rule" {
				t.Errorf("%s/%s: quiet key still asks: %+v", p, class, d)
			}
		}
	}
}

// An answered question is not put again within the cooldown and its answer acts; after the cooldown it asks again,
// and the streak still counts toward a rule.
func TestRepeatReusesTheAnswer(t *testing.T) {
	for _, class := range []string{"sorceress", "paladin"} {
		for _, p := range repeatPoints {
			r := newRig(t)
			r.policy()
			c := r.rigClass(p, class)
			def, _ := r.cat.Default(c)
			ans := r.otherAnswer(p, def)
			r.say(c, ans)
			for i := 0; i < 20; i++ {
				r.clk.Advance(time.Minute)
				d := r.ask(c)
				if d.Thought != nil || d.Answer != ans || d.Source != "answer" {
					t.Fatalf("%s/%s: repeat %d = %+v, want the answer %q without a thought", p, class, i, d, ans)
				}
			}
			r.clk.Advance(DefaultRepeatCooldown)
			if d := r.ask(c); d.Thought == nil {
				t.Fatalf("%s/%s: no thought after the cooldown", p, class)
			}
		}
	}
}

// "Not sure" is an answer for the cooldown (the player saw the question) but the default acts.
func TestRepeatNotSureKeepsDefault(t *testing.T) {
	r := newRig(t)
	r.policy()
	c := r.rigClass("t0.area.toohard", "sorceress")
	def, _ := r.cat.Default(c)
	r.say(c, NotSure)
	r.clk.Advance(time.Minute)
	if d := r.ask(c); d.Thought != nil || d.Answer != def {
		t.Fatalf("after not sure: %+v, want default %q and no thought", d, def)
	}
}

// An answer resets the unanswered count, so a player who answers now and then never silences the key.
func TestRepeatAnswerResetsQuietCount(t *testing.T) {
	r := newRig(t)
	r.policy()
	c := r.rigClass("t1.repair", "sorceress")
	for round := 0; round < DefaultQuietAfter*2; round++ {
		d := r.ask(c)
		if d.Thought == nil {
			t.Fatalf("round %d: key went quiet although it was answered between fades", round)
		}
		if round%2 == 0 { // ignore it
			r.clk.Advance(fade)
			r.s.Tick()
		} else if err := r.s.Answer(d.Thought.ID, []string{"skip", "repair"}[round/2%2]); err != nil { // never two alike: no rule
			t.Fatal(err)
		}
		r.clk.Advance(DefaultRepeatCooldown)
	}
}

// Persistence: the answer, the fades and the quiet state come back from the saved log after a restart (a new run
// whose clock starts at 0); the cooldown windows start anew.
func TestRepeatSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	cat := mustCat(t)
	open := func() (*Store, *ManualClock) {
		clk := &ManualClock{}
		s, err := Open(dir, "p1", cat, clk)
		if err != nil {
			t.Fatal(err)
		}
		s.SetRepeatPolicy(DefaultRepeatCooldown, DefaultQuietAfter)
		return s, clk
	}
	ctxFor := func(p string) Context {
		c, err := cat.Build(p, baseObs(cat, "paladin"), 1, nil)
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	s, clk := open()
	ca, cq := ctxFor("t1.repair"), ctxFor("t0.area.toohard")
	d, _ := s.Ask(ca)
	if err := s.Answer(d.Thought.ID, "skip"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < DefaultQuietAfter; i++ {
		if d, _ := s.Ask(cq); d.Thought == nil {
			t.Fatal("expected a thought")
		}
		clk.Advance(fade)
		s.Tick()
		clk.Advance(DefaultRepeatCooldown)
	}
	s, clk = open() // container recreate
	if d, _ := s.Ask(ca); d.Thought != nil || d.Answer != "skip" || d.Source != "answer" {
		t.Fatalf("answer lost across reload: %+v", d)
	}
	if d, _ := s.Ask(cq); d.Thought != nil {
		t.Fatalf("quiet state lost across reload: %+v", d)
	}
	clk.Advance(DefaultRepeatCooldown)
	if d, _ := s.Ask(ca); d.Thought == nil {
		t.Fatal("answered key must ask again once the cooldown passed on the new clock")
	}
}

// Policy off (the default of a bare store): every behaviour of before is unchanged.
func TestRepeatOffByDefault(t *testing.T) {
	r := newRig(t)
	c := r.rigClass("t1.repair", "sorceress")
	r.say(c, "skip")
	if d := r.ask(c); d.Thought == nil || d.Source == "answer" {
		t.Fatalf("a bare store must ask again: %+v", d)
	}
}
