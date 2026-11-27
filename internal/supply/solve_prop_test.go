package supply

import (
	"math/rand/v2"
	"reflect"
	"sort"
	"testing"
)

// Property tests on Solve (roadmap T3, docs/design/proptest-t3.md). Fixed seeds, no clock. The two builds are the
// `builds` inputs of solve_test.go.

const solveCases = 5000

var verdictRank = map[Verdict]int{Blocked: 0, ShortGold: 1, ReadyLean: 2, Ready: 3}

type solveCase struct {
	n     Need
	c     Count
	p     Prices
	pol   Policy
	sale  int
	build string
}

func randCount(r *rand.Rand) Count {
	var c Count
	for k := range kinds {
		c.Belt[k], c.Pack[k] = r.IntN(6), r.IntN(4)
	}
	c.PortalCharges = r.IntN(6)
	c.Purse = r.IntN(4000)
	c.Stash = r.IntN(20000)
	c.StashKnown = r.IntN(2) == 0
	return c
}

func randPrices(r *rand.Rand) Prices {
	p := Prices{}
	for _, l := range []Line{Healing, Mana, Rejuvenation, Portal} {
		switch r.IntN(8) {
		case 0: // absent: cannot be bought here
		case 1:
			p[l] = -1
		case 2:
			p[l] = 0
		default:
			p[l] = 1 + r.IntN(400)
		}
	}
	return p
}

func randPriority(r *rand.Rand) []Step {
	if r.IntN(2) == 0 {
		return nil // DefaultPriority
	}
	// any order of the min and one steps, then any order of the target steps: a policy that spends on a target before
	// a min can trade a min for a target as gold grows, which the monotone property below does not claim
	var mins, targets []Step
	for _, s := range DefaultPriority {
		if s.To == "target" {
			targets = append(targets, s)
		} else {
			mins = append(mins, s)
		}
	}
	r.Shuffle(len(mins), func(i, j int) { mins[i], mins[j] = mins[j], mins[i] })
	r.Shuffle(len(targets), func(i, j int) { targets[i], targets[j] = targets[j], targets[i] })
	return append(mins, targets...)
}

// forEachSolve runs f on random cases for every build and every profile row, one fixed seed per build.
func forEachSolve(t *testing.T, f func(t *testing.T, sc solveCase)) {
	t.Helper()
	profiles, err := Profiles()
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(builds))
	for name := range builds {
		names = append(names, name)
	}
	sort.Strings(names)
	for i, name := range names {
		t.Run(name, func(t *testing.T) {
			r := rand.New(rand.NewPCG(uint64(980+i), 5))
			for range solveCases {
				in := builds[name]
				in.BaseTarget = map[Line]int{Healing: r.IntN(12), Mana: r.IntN(8), Portal: r.IntN(10)}
				n, err := NeedFor("prop", profiles[r.IntN(len(profiles))], in)
				if err != nil {
					t.Fatal(err)
				}
				sc := solveCase{n: n, c: randCount(r), p: randPrices(r), pol: Policy{Priority: randPriority(r), Reserve: r.IntN(3) * r.IntN(2000)}, sale: r.IntN(3) * r.IntN(1500), build: name}
				f(t, sc)
				if t.Failed() {
					return
				}
			}
		})
	}
}

func (sc solveCase) solve() Plan { return Solve(sc.n, sc.c, sc.p, sc.pol, sc.sale) }

// Solve is pure (invariant T8) and keeps its own books: it never spends more than the budget, the spend is the sum of
// the line costs, and no line buys a negative amount.
func TestPropSolveBooks(t *testing.T) {
	forEachSolve(t, func(t *testing.T, sc solveCase) {
		pl := sc.solve()
		if again := sc.solve(); !reflect.DeepEqual(pl, again) {
			t.Fatalf("Solve not deterministic:\n%s\n%s", pl.Trace(), again.Trace())
		}
		sum := 0
		for _, ln := range pl.Lines {
			if ln.Buy < 0 || ln.Cost < 0 {
				t.Errorf("negative line %+v", ln)
			}
			sum += ln.Cost
		}
		if pl.Spent != sum || pl.Spent > pl.Budget || pl.Budget < 0 {
			t.Errorf("books: spent %d, sum %d, budget %d: %s", pl.Spent, sum, pl.Budget, pl.Trace())
		}
	})
}

// The verdict says what the lines say: ready covers every target, ready-lean every min, and short-gold or blocked names
// an open min line.
func TestPropSolveVerdictMatchesLines(t *testing.T) {
	forEachSolve(t, func(t *testing.T, sc solveCase) {
		pl := sc.solve()
		openMin := false
		for _, ln := range pl.Lines {
			got := ln.Have + ln.Buy
			if ln.Line == GoldCarry {
				got = ln.Buy
			}
			if ln.Line != GoldCarry && got < ln.Min {
				openMin = true
			}
			switch pl.Verdict {
			case Ready:
				if got < ln.Min || got < ln.Target && ln.Line != GoldCarry {
					t.Errorf("ready with %s at %d (min %d, target %d): %s", ln.Line, got, ln.Min, ln.Target, pl.Trace())
				}
			case ReadyLean:
				if ln.Line != GoldCarry && got < ln.Min {
					t.Errorf("ready-lean with %s below min: %s", ln.Line, pl.Trace())
				}
			}
		}
		if (pl.Verdict == ShortGold || pl.Verdict == Blocked) != openMin {
			t.Errorf("verdict %s but open min line %v: %s", pl.Verdict, openMin, pl.Trace())
		}
		if openMin && pl.Open == "" {
			t.Errorf("open min line not named: %s", pl.Trace())
		}
	})
}

// More potions, scrolls or gold never make the verdict worse.
func TestPropSolveMoreNeverWorse(t *testing.T) {
	more := map[string]func(*Count){
		"healing": func(c *Count) { c.Pack[HP]++ },
		"mana":    func(c *Count) { c.Belt[MP]++ },
		"rejuv":   func(c *Count) { c.Pack[RV]++ },
		"portal":  func(c *Count) { c.PortalCharges++ },
		"purse":   func(c *Count) { c.Purse += 1 + c.Purse/3 },
		"stash":   func(c *Count) { c.StashKnown = true },
	}
	forEachSolve(t, func(t *testing.T, sc solveCase) {
		before := sc.solve()
		for name, add := range more {
			sc2 := sc
			add(&sc2.c)
			after := sc2.solve()
			if verdictRank[after.Verdict] < verdictRank[before.Verdict] {
				t.Errorf("%s: one more %s turns %s into %s\nbefore %s\nafter  %s", sc.build, name, before.Verdict, after.Verdict, before.Trace(), after.Trace())
			}
		}
	})
}
