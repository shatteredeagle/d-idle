package supply

import (
	"fmt"
	"sort"
	"strings"
)

// Verdict is the one answer the guards, the shop and the viewer share.
type Verdict string

const (
	Ready     Verdict = "ready"      // every target is covered
	ReadyLean Verdict = "ready-lean" // every min is covered, a target is not
	ShortGold Verdict = "short-gold" // a min line is unaffordable
	Blocked   Verdict = "blocked"    // a min line cannot be bought here at any price
)

// Prices are the gold per unit of each line; a line absent from the map cannot be bought (price < 0 says so too).
type Prices map[Line]int

// Step is one entry of the spending order: a line filled up to its min, its target, or one unit.
type Step struct {
	Line Line
	To   string // "one", "min" or "target"
}

// Policy holds the order gold is spent in (empty: the data of priority.json for the need's profile) and the gold the plan must not spend.
type Policy struct {
	Priority []Step
	Reserve  int // gold kept back (merc revive, floor), taken from the budget first
}

// PlanLine is one row of the plan.
type PlanLine struct {
	Line                   Line
	Have, Min, Target, Buy int
	Cost                   int
	Status                 string // ok, buy, short, dropped, clamped
	Why                    string
}

// Plan is the result of Solve.
type Plan struct {
	Need    Need
	Lines   []PlanLine
	Budget  int // gold the plan could spend
	Spent   int
	Verdict Verdict
	Short   int    // gold missing for the first open min line
	Open    string // the first open min line, "" when none
}

// Solve is pure: the same need, count, prices and policy give the same plan (invariant T8). The budget is the purse plus
// the stash gold the counter could read plus the estimated sale, minus the reserve.
func Solve(n Need, c Count, p Prices, pol Policy, saleEstimate int) Plan {
	order := pol.Priority
	if len(order) == 0 {
		order = PriorityOf(n.Profile)
	}
	names := make([]string, 0, len(n.Lines))
	for l := range n.Lines {
		names = append(names, string(l))
	}
	sort.Strings(names)
	pl := Plan{Need: n, Budget: max(c.Wealth()+saleEstimate-pol.Reserve, 0)}
	idx := map[Line]int{}
	for _, name := range names {
		l := Line(name)
		q := n.Lines[l]
		st := "ok"
		if q.Max > 0 && (q.Min > q.Max || q.Target > q.Max) {
			q.Min, q.Target = min(q.Min, q.Max), min(q.Target, q.Max)
			st = "clamped" // an impossible need is lowered to what the hero can carry (data error, no guard waits on it)
		}
		have := c.Have(l)
		if l == GoldCarry {
			have = 0 // gold the excursion keeps is reserved, not carried: it consumes budget at its priority
		}
		idx[l] = len(pl.Lines)
		pl.Lines = append(pl.Lines, PlanLine{Line: l, Have: have, Min: q.Min, Target: q.Target, Status: st, Why: q.MinWhy})
	}
	left := pl.Budget
	for _, step := range order {
		i, ok := idx[step.Line]
		if !ok {
			continue
		}
		ln := &pl.Lines[i]
		goal := ln.Min
		switch step.To {
		case "one":
			goal = min(ln.Min, 1)
		case "target":
			goal = ln.Target
		}
		want := goal - ln.Have - ln.Buy
		if want <= 0 {
			continue
		}
		price, buyable := p[step.Line]
		if step.Line == GoldCarry {
			price, buyable = 1, true
		}
		if !buyable || price < 0 {
			ln.Status = "blocked"
			continue
		}
		k := want
		if price > 0 {
			k = min(want, left/price)
		}
		ln.Buy += k
		ln.Cost += k * price
		left -= k * price
		pl.Spent += k * price
		if k < want {
			ln.Status = "dropped"
		} else if ln.Status == "ok" {
			ln.Status = "buy"
		}
	}
	pl.Verdict = Ready
	for i := range pl.Lines {
		ln := &pl.Lines[i]
		if ln.Line == GoldCarry {
			continue
		}
		got := ln.Have + ln.Buy
		price, buyable := p[ln.Line]
		switch {
		case got < ln.Min && (!buyable || price < 0):
			pl.Verdict, pl.Open = Blocked, string(ln.Line)
		case got < ln.Min:
			if pl.Verdict != Blocked {
				pl.Verdict = ShortGold
			}
			if pl.Open == "" {
				pl.Open, pl.Short = string(ln.Line), (ln.Min-got)*price
			}
			ln.Status = "short"
		case got < ln.Target && pl.Verdict == Ready:
			pl.Verdict = ReadyLean
		}
	}
	if g, ok := idx[GoldCarry]; ok && pl.Lines[g].Buy < pl.Lines[g].Min && pl.Verdict == Ready {
		pl.Verdict = ReadyLean
	}
	return pl
}

// Trace is the one-line shadow form of a plan: "needs X, has Y, would buy Z" (town-planner section 8).
func (pl Plan) Trace() string {
	var need, has, buy []string
	for _, ln := range pl.Lines {
		need = append(need, fmt.Sprintf("%s %d/%d", ln.Line, ln.Min, ln.Target))
		has = append(has, fmt.Sprintf("%s %d", ln.Line, ln.Have))
		if ln.Buy > 0 {
			buy = append(buy, fmt.Sprintf("%s x%d(%d)", ln.Line, ln.Buy, ln.Cost))
		}
	}
	s := fmt.Sprintf("for=%s profile=%s verdict=%s needs %s; has %s; would buy %s", pl.Need.Excursion, pl.Need.Profile, pl.Verdict,
		strings.Join(need, ","), strings.Join(has, ","), orNone(buy))
	if pl.Open != "" {
		s += fmt.Sprintf("; open=%s short=%d", pl.Open, pl.Short)
	}
	return s
}

func orNone(s []string) string {
	if len(s) == 0 {
		return "nothing"
	}
	return strings.Join(s, ",")
}

// MinOf is the effective (clamped) min of a line, 0 when the plan has no such line.
func (p Plan) MinOf(l Line) int {
	for _, ln := range p.Lines {
		if ln.Line == l {
			return ln.Min
		}
	}
	return 0
}

// Why is the sentence a guard says when the plan cannot close a min line with the gold the hero owns, "" when it can
// (town-planner section 5.4: a guard blocks only on a line the plan cannot close, and names it).
func (p Plan) Why() string {
	switch p.Verdict {
	case ShortGold:
		return fmt.Sprintf("short of gold: needs %d more for %s", p.Short, p.Open)
	case Blocked:
		return fmt.Sprintf("blocked: %s cannot be bought here", p.Open)
	}
	return ""
}
