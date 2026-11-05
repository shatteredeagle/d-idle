package ladder_test

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

type countSrc struct {
	n int
	r ladder.Reply
}

func (c *countSrc) Reply(ladder.ThoughtView) ladder.Reply { c.n++; return c.r }

// A scripted file teaches a rule: two answers (safe class, N 2) to "skip" at the same key make skip the rule.
func TestScriptedAnswersLearnARule(t *testing.T) {
	src, err := ladder.ParseScript([]byte(`{"rows":[{"point":"t1.repair","keyHas":"durability=20-30","answer":"skip","confirm":"yes","delaySec":1}]}`))
	if err != nil {
		t.Fatal(err)
	}
	l, clk, sink := newLadder(t, src)
	h := hero(d2.Sorceress, 300)
	for i := 0; i < 4; i++ {
		got, err := l.RepairWanted(h, 25, true)
		if err != nil {
			t.Fatal(err)
		}
		if i == 0 && !got {
			t.Fatal("untrained default must repair at 25 percent")
		}
		clk.Advance(5 * time.Second)
		l.Pump()
	}
	if got, _ := l.RepairWanted(h, 25, true); got {
		t.Fatal("rule skip not applied after two matching answers")
	}
	// other bands keep the default
	if got, _ := l.RepairWanted(h, 10, true); !got {
		t.Fatal("a rule for 20-30 changed the lt20 band")
	}
	if rules, answers, _ := l.Store().Stats(); rules != 1 || answers < 2 {
		t.Fatalf("rules %d answers %d", rules, answers)
	}
	var thought bool
	for _, r := range sink.Lines() {
		thought = thought || strings.HasPrefix(r, "thought t1.repair key=t1.repair|")
	}
	if !thought {
		t.Fatalf("no thought line in %v", sink.Lines())
	}
}

// A question never delays the action and is not asked during a fight.
func TestNoQuestionsDuringAFight(t *testing.T) {
	clk := &decide.ManualClock{}
	src := &countSrc{r: ladder.Reply{Answer: "skip"}}
	fight := true
	l, err := ladder.Open("", "t0", clk, nil, src, func() bool { return fight })
	if err != nil {
		t.Fatal(err)
	}
	got, err := l.RepairWanted(hero(d2.Sorceress, 100), 25, true)
	if err != nil || !got {
		t.Fatalf("default now: %v %v", got, err)
	}
	if src.n != 0 {
		t.Fatal("asked during a fight")
	}
	fight = false
	l.Pump()
	if src.n != 1 {
		t.Fatalf("asked %d times after the fight", src.n)
	}
}

// A thought nobody answers fades on the virtual clock.
func TestUnansweredThoughtFades(t *testing.T) {
	src := &countSrc{r: ladder.Reply{}}
	l, clk, sink := newLadder(t, src)
	l.RepairWanted(hero(d2.Sorceress, 100), 25, true)
	clk.Advance(46 * time.Second)
	l.RepairWanted(hero(d2.Sorceress, 100), 90, false) // Tick runs on every decision
	if st := l.Stats(); st.Faded != 1 {
		t.Fatalf("stats %+v lines %v", st, sink.Lines())
	}
}

func TestUnknownClassIsAnErrorNotACrash(t *testing.T) {
	l, _, _ := newLadder(t, nil)
	_, err := l.RepairWanted(hero(d2.Class(99), 100), 25, true)
	if err == nil || !strings.Contains(err.Error(), "class") {
		t.Fatalf("err %v", err)
	}
	l.Fault(err)
	if l.Stats().Faults != 1 {
		t.Fatal("fault not counted")
	}
}

func TestNoLadderInstalledByDefault(t *testing.T) {
	if ladder.Active() != nil {
		t.Fatal("a ladder is active without Install")
	}
}

// Invariants for every answer profile (3 known, 2 holdouts from seeds), two builds: a bounded random town session of
// ten virtual minutes never errors, stalls, loops or grows without bound, and every decision is a valid choice.
func TestProfilesKeepTheInvariants(t *testing.T) {
	type pr struct {
		name string
		seed int64
	}
	profs := []pr{{"cautious", 1}, {"bold", 2}, {"inconsistent", 3}, {"holdout", 101}, {"holdout", 202}}
	runs := 0
	for _, p := range profs {
		for _, c := range classes {
			src, err := ladder.NewProfileSource(p.name, p.seed)
			if err != nil {
				t.Fatal(err)
			}
			l, clk, _ := newLadder(t, src)
			rng := rand.New(rand.NewSource(p.seed))
			calls := 0
			for at := time.Duration(0); at < 10*time.Minute; at += 5 * time.Second {
				clk.Advance(5 * time.Second)
				h := hero(c, rng.Intn(2000))
				if err := callPoint(l, rng, h); err != nil {
					t.Fatalf("%s %v: %v", p.name, c, err)
				}
				calls++
				l.Pump()
			}
			st := l.Stats()
			if st.Asked != calls {
				t.Fatalf("%s: %d calls, %d asked: a call stalled", p.name, calls, st.Asked)
			}
			if n := len(l.Store().Log()); n > 4*calls {
				t.Fatalf("%s: log grew to %d lines", p.name, n)
			}
			for _, v := range l.Store().Rules() {
				if v.Revoked {
					t.Fatalf("%s: rule revoked without a player action", p.name)
				}
			}
			runs++
		}
	}
	t.Logf("profile x build runs: %d (5 profiles incl. 2 holdouts, 2 builds)", runs)
}

// Bounded random scripts: arbitrary replies, delays and confirms, 30 seeds.
func TestRandomScriptsNeverStall(t *testing.T) {
	cats := []string{"repair", "skip", "buy", "walk", "go", "continue", "notsure", "", "bogus", "yes", "no"}
	for seed := int64(0); seed < 30; seed++ {
		rng := rand.New(rand.NewSource(seed))
		var rows []string
		for i := 0; i < 5; i++ {
			rows = append(rows, fmt.Sprintf(`{"point":%q,"answer":%q,"confirm":%q,"delaySec":%d}`,
				[]string{"t1.repair", "t1.tp.lost", "t1.return", "t1.town.trigger", "t1.restock", ""}[rng.Intn(6)], cats[rng.Intn(len(cats))], cats[rng.Intn(len(cats))], rng.Intn(80)))
		}
		src, err := ladder.ParseScript([]byte(`{"rows":[` + strings.Join(rows, ",") + `]}`))
		if err != nil {
			t.Fatal(err)
		}
		l, clk, _ := newLadder(t, src)
		for i := 0; i < 200; i++ {
			clk.Advance(time.Duration(rng.Intn(20)) * time.Second)
			h := hero(classes[rng.Intn(2)], rng.Intn(1500))
			if err := callPoint(l, rng, h); err != nil {
				t.Fatalf("seed %d step %d: %v", seed, i, err)
			}
			l.Pump()
		}
		if n := len(l.Store().OpenThoughts()); n > 40 {
			t.Fatalf("seed %d: %d open thoughts", seed, n)
		}
	}
}

func TestUnknownProfileNamed(t *testing.T) {
	if _, err := ladder.NewProfileSource("reckless", 1); err == nil || !strings.Contains(err.Error(), "reckless") {
		t.Fatalf("err %v", err)
	}
}

// callPoint asks one of the five wired points with random state and a random legacy answer.
func callPoint(l *ladder.Ladder, rng *rand.Rand, h ladder.Hero) error {
	var err error
	lg := rng.Intn(2) == 0
	belt := []string{ladder.BeltFull, ladder.BeltHalf, ladder.BeltLow}[rng.Intn(3)]
	switch rng.Intn(5) {
	case 0:
		_, err = l.RepairWanted(h, rng.Intn(101), lg)
	case 1:
		_, err = l.TPBuyWanted(h, lg)
	case 2:
		_, err = l.ReturnWanted(h, []string{ladder.KitOK, ladder.KitThin, ladder.KitNone}[rng.Intn(3)], lg)
	case 3:
		_, err = l.TripWanted(h, belt, rng.Intn(101), lg)
	default:
		_, err = l.RestockWanted(h, belt, rng.Intn(600), lg)
	}
	return err
}
