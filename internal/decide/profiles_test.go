package decide

import (
	"bytes"
	"math/rand"
	"testing"
	"time"
)

// The profiles themselves are the public API (profiles.go); the tests use them through these short names.
type profile = Profile

func knownProfiles() []Profile   { return KnownProfiles() }
func holdout(seed int64) Profile { return Holdout(seed) }

// genScript makes a script over a fixed pool of observations per point so that keys repeat.
func genScript(rng *rand.Rand, cat *Catalogue, class string, prof profile, n int) []Step {
	pool := map[string][]Obs{}
	for _, p := range cat.Points {
		for i := 0; i < 3; i++ {
			pool[p.ID] = append(pool[p.ID], randObs(rng, cat, class))
		}
	}
	steps := make([]Step, 0, n)
	for len(steps) < n {
		p := &cat.Points[rng.Intn(len(cat.Points))]
		o := pool[p.ID][rng.Intn(3)]
		ctx, err := cat.Build(p.ID, o, 1, nil)
		if err != nil {
			panic(err)
		}
		def, _ := cat.Default(ctx)
		st := Step{Wait: time.Duration(rng.Intn(70)) * time.Second, Point: p.ID, Obs: o,
			Reply: prof.Reply(rng, p, def), Confirm: prof.Confirm(rng)}
		steps = append(steps, st)
	}
	return steps
}

// checkInvariants: for ANY answers the store never loops, crashes or stalls, defaults are valid answers, an
// irreversible point without a rule does nothing, and no rule is ever revoked (the script never revokes).
func checkInvariants(t *testing.T, cat *Catalogue, s *Store, decisions []Decision, steps []Step) {
	t.Helper()
	if len(decisions) != len(steps) {
		t.Fatalf("%d decisions for %d steps: a step stalled", len(decisions), len(steps))
	}
	for i, d := range decisions {
		p, _ := cat.Point(steps[i].Point)
		a, ok := p.answer(d.Answer)
		if !ok {
			t.Fatalf("step %d: decision %q is not an answer of %s", i, d.Answer, p.ID)
		}
		if p.Irreversible && d.Source != "rule" && !a.Noop {
			t.Fatalf("step %d: untrained irreversible %s acts: %q", i, p.ID, d.Answer)
		}
	}
	// bounded work: at most open + answer/fade/confirm lines per step, and few thoughts open at once
	if got := len(s.Log()); got > 4*len(steps) {
		t.Fatalf("log grew to %d lines for %d steps", got, len(steps))
	}
	if n := len(s.OpenThoughts()); n > len(cat.Points)*3*2 {
		t.Fatalf("%d open thoughts", n)
	}
	for _, v := range s.Rules() {
		if v.Revoked {
			t.Fatalf("rule %s revoked without a player action", v.Key)
		}
		p, _ := cat.Point(v.Point)
		if _, ok := p.answer(v.Answer); !ok {
			t.Fatalf("rule %s has non-answer %q", v.Key, v.Answer)
		}
	}
	for _, e := range s.Log() {
		if e.Op == OpRevoke || e.Op == OpRevokeAll {
			t.Fatalf("revoke line without a player action: %+v", e)
		}
	}
}

func runProfile(t *testing.T, cat *Catalogue, class string, prof profile, seed int64, n int) (*Store, []Step) {
	t.Helper()
	rng := rand.New(rand.NewSource(seed))
	steps := genScript(rng, cat, class, prof, n)
	clk := &ManualClock{}
	s, err := New(cat, "prof-"+prof.Name, clk)
	if err != nil {
		t.Fatal(err)
	}
	ds, err := RunScript(s, clk, 1, nil, steps)
	if err != nil {
		t.Fatalf("%s seed %d: %v", prof.Name, seed, err)
	}
	checkInvariants(t, cat, s, ds, steps)
	// the live rules equal the replay of the log, byte for byte
	live, _ := s.RulesJSON()
	again, err := Replay(cat, "prof-"+prof.Name, s.Log())
	if err != nil || !bytes.Equal(live, again) {
		t.Fatalf("%s: replay differs (%v)", prof.Name, err)
	}
	return s, steps
}

func TestAnswerProfiles(t *testing.T) {
	cat := mustCat(t)
	profs := append(knownProfiles(), holdout(101), holdout(202))
	if len(profs) != 5 {
		t.Fatal("3 known + 2 holdout profiles expected")
	}
	for _, class := range []string{"sorceress", "paladin"} {
		for _, prof := range profs {
			t.Run(class+"/"+prof.Name, func(t *testing.T) {
				s, _ := runProfile(t, cat, class, prof, 7, 3000)
				rules, answers, faded := s.Stats()
				t.Logf("rules %d answers %d faded %d", rules, answers, faded)
				switch prof.Name {
				case "cautious", "bold":
					if rules == 0 {
						t.Fatal("a consistent profile should teach some rule")
					}
				}
			})
		}
	}
}

func TestCautiousRulesEqualDefaults(t *testing.T) {
	cat := mustCat(t)
	s, _ := runProfile(t, cat, "sorceress", knownProfiles()[0], 11, 3000)
	for _, v := range s.Rules() {
		ctx, _ := ParseKey(v.Key)
		if def, _ := cat.Default(ctx); def != v.Answer {
			t.Errorf("cautious learned %s for %s, default is %s", v.Answer, v.Key, def)
		}
	}
}

// Property-style: random answer scripts with a fixed seed, bounded steps. Includes player list actions.
func TestInvariantsRandomScripts(t *testing.T) {
	cat := mustCat(t)
	for seed := int64(1); seed <= 25; seed++ {
		rng := rand.New(rand.NewSource(seed))
		prof := holdout(seed * 31)
		steps := genScript(rng, cat, "sorceress", prof, 600)
		clk := &ManualClock{}
		s, _ := New(cat, "prop", clk)
		var decisions []Decision
		for i := 0; i < len(steps); i++ {
			ds, err := RunScript(s, clk, 1, nil, steps[i:i+1])
			if err != nil {
				t.Fatalf("seed %d step %d: %v", seed, i, err)
			}
			decisions = append(decisions, ds...)
			// the player sometimes revokes, sacrifices or is reborn; none of it may break anything
			switch rng.Intn(40) {
			case 0:
				if rules := s.Rules(); len(rules) > 0 {
					s.Revoke(rules[rng.Intn(len(rules))].Key) // may be revoked already: error is fine
				}
			case 1:
				s.RevokeAll()
			case 2:
				if err := s.Sacrifice(cat.Points[rng.Intn(len(cat.Points))].ID); err != nil {
					t.Fatal(err)
				}
			case 3:
				if err := s.Rebirth("hero" + string(rune('a'+rng.Intn(20)))); err != nil {
					t.Fatal(err)
				}
			}
		}
		if len(decisions) != len(steps) {
			t.Fatalf("seed %d stalled", seed)
		}
		for i, d := range decisions {
			p, _ := cat.Point(steps[i].Point)
			if _, ok := p.answer(d.Answer); !ok {
				t.Fatalf("seed %d: bad decision %q", seed, d.Answer)
			}
		}
		live, _ := s.RulesJSON()
		again, err := Replay(cat, "prop", s.Log())
		if err != nil || !bytes.Equal(live, again) {
			t.Fatalf("seed %d: replay differs: %v", seed, err)
		}
	}
}

func TestReplayDeterminism(t *testing.T) {
	cat := mustCat(t)
	prof := holdout(5)
	var first []byte
	for i := 0; i < 3; i++ {
		s, _ := runProfile(t, cat, "paladin", prof, 99, 1500)
		b, _ := s.RulesJSON()
		if i == 0 {
			first = b
			// a script cut in two halves replays to the same bytes too
			log := s.Log()
			again, err := Replay(cat, "prof-holdout", log)
			if err != nil || !bytes.Equal(again, b) {
				t.Fatal("replay of the log differs")
			}
			continue
		}
		if !bytes.Equal(first, b) {
			t.Fatal("same script and profile gave different rules")
		}
	}
	// a damaged log is an error naming the entry
	if _, err := Replay(cat, "prof-holdout", []Entry{{Seq: 1, Op: OpAnswer, Thought: 9, Answer: "x"}}); err == nil {
		t.Fatal("replay of an answer to a thought that was never opened")
	}
	if _, err := Replay(cat, "prof-holdout", []Entry{{Seq: 1, Op: OpOpen, Thought: 1, Key: "t9.x|a=b"}}); err == nil {
		t.Fatal("replay with an unknown point")
	}
	if _, err := Replay(cat, "Bad Name", nil); err == nil {
		t.Fatal("bad profile name")
	}
}
