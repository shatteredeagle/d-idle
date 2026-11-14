package levelsim

import (
	"fmt"
	"sort"
	"strings"
)

type node struct {
	st   State
	prev *node
	step Step
}

// Best is one optimiser answer.
type Best struct {
	H      float64
	XP     float64
	Clvl   int
	Steps  []Step
	End    State
	Reach  map[int]float64 // minutes to reach clvl 10, 15, 20 (absent = not reached inside H)
	Nodes  int
	Width  int
	DenOK  bool
	BROK   bool
	Reason string
	Seeded bool   // the policy seed beat the beam
	Top    []Best // the best distinct alternatives (different area orders), best first
}

func (n *node) path() []Step {
	var rev []Step
	for ; n != nil && n.prev != nil; n = n.prev {
		rev = append(rev, n.step)
	}
	out := make([]Step, len(rev))
	for i, s := range rev {
		out[len(rev)-1-i] = s
	}
	return out
}

func (e *Env) key(s *State) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%d|%d|%v%v|%d", s.Loc, int(s.XP/60), s.Den, s.BR, s.Games)
	for _, k := range s.Kills {
		fmt.Fprintf(&b, ",%d", int(k/12))
	}
	return b.String()
}

// reachLevels are the levels the "time to clvl" answers cover.
var ReachLevels = []int{10, 15, 20}

// Optimise runs a beam search over farm slices (area x exit rule) and restarts for H minutes and returns
// the highest XP at H, with the hard constraints: entry floors of the mode, the Den quest done, Blood
// Raven done if the final clvl is 9 or more, the Den not farmed again unless AllowDenRefarm.
func (e *Env) Optimise(H float64, width int) Best {
	buckets := make([][]*node, int(H)+2)
	root := &node{st: NewState()}
	buckets[0] = []*node{root}
	var best *node
	reach := map[int]*node{}
	total := 0
	var finals []*node
	finish := func(n *node) {
		cl := n.st.Clvl()
		if (!n.st.Den && !e.NoDenRequired) || (cl >= 9 && !n.st.BR && !e.NoDenRequired) {
			return
		}
		finals = append(finals, n)
		if best == nil || n.st.XP > best.st.XP {
			best = n
		}
	}
	exits := []float64{0.5, 0.25, e.P.Depleted}
	for b := 0; b < len(buckets); b++ {
		list := e.prune(buckets[b], width)
		buckets[b] = nil
		for _, n := range list {
			total++
			e.expand(n, H, exits, buckets, finish, reach)
		}
	}
	out := Best{H: H, Width: width, Nodes: total, Reach: map[int]float64{}}
	if best == nil {
		out.Reason = "no strategy satisfied the constraints"
		return out
	}
	out.XP, out.Clvl, out.Steps, out.End = best.st.XP, best.st.Clvl(), best.path(), best.st
	out.DenOK, out.BROK = best.st.Den, best.st.BR
	out.Top = topDistinct(finals, 5)
	for l, n := range reach {
		out.Reach[l] = n.st.LvlT[l]
	}
	return out
}

func (e *Env) expand(n *node, H float64, exits []float64, buckets [][]*node, finish func(*node), reach map[int]*node) {
	cl := n.st.Clvl()
	push := func(s2 State, st Step) {
		c := &node{st: s2, prev: n, step: st}
		for _, l := range ReachLevels {
			if s2.LvlT[l] > 0 && s2.LvlT[l] <= H && s2.Den && (l < 9 || s2.BR) {
				if old, ok := reach[l]; !ok || s2.LvlT[l] < old.st.LvlT[l] {
					reach[l] = c
				}
			}
		}
		if s2.T >= H-1e-6 {
			finish(c)
			return
		}
		b := int(s2.T)
		if b >= len(buckets) {
			return
		}
		buckets[b] = append(buckets[b], c)
	}
	for _, id := range e.D.Order {
		a := e.D.Areas[id]
		if cl < e.Floor(id) || (id == 8 && n.st.Den && !e.AllowDenRefarm) {
			continue
		}
		f := e.Remaining(&n.st, a)
		if f <= e.P.Depleted+0.01 || n.st.Kills[id] >= a.Units {
			continue
		}
		type variant struct {
			minF    float64
			onLevel bool
		}
		var vs []variant
		for _, m := range exits {
			if f > m+0.02 {
				vs = append(vs, variant{m, false})
			}
		}
		vs = append(vs, variant{e.P.Depleted, true})
		for _, v := range vs {
			s2 := n.st
			st := Step{Kind: "farm", Area: id, T0: s2.T, XP0: s2.XP, Clvl0: cl}
			st.Travel = e.Travel(&s2, id)
			if s2.T >= H {
				continue
			}
			k0 := s2.TotalK
			e.Farm(&s2, a, Exit{MinF: v.minF, OnLevel: v.onLevel, TMax: H}, nil)
			st.T1, st.XP1, st.Clvl1, st.Kills = s2.T, s2.XP, s2.Clvl(), s2.TotalK-k0
			if v.onLevel {
				st.Exit = "next level-up"
			} else {
				st.Exit = fmt.Sprintf("f<=%.2f", v.minF)
			}
			push(s2, st)
		}
	}
	if n.step.Kind != "restart" && n.st.TotalK > 0 {
		s2 := n.st
		st := Step{Kind: "restart", Area: 1, T0: s2.T, XP0: s2.XP, Clvl0: cl, Clvl1: cl}
		e.Restart(&s2)
		st.T1, st.XP1 = s2.T, s2.XP
		push(s2, st)
	}
}

// prune keeps the best `width` states of each (Den, Blood Raven) class and drops near duplicates.
func (e *Env) prune(list []*node, width int) []*node {
	if len(list) <= width {
		return list
	}
	sort.Slice(list, func(i, j int) bool { return list[i].st.XP > list[j].st.XP })
	seen := map[string]bool{}
	count := map[int]int{}
	var out []*node
	for _, n := range list {
		c := 0
		if n.st.Den {
			c |= 1
		}
		if n.st.BR {
			c |= 2
		}
		if count[c] >= width {
			continue
		}
		k := e.key(&n.st)
		if seen[k] {
			continue
		}
		seen[k] = true
		count[c]++
		out = append(out, n)
	}
	return out
}

// signature is the order of areas a path farms (consecutive repeats and restarts dropped).
func signature(steps []Step) string {
	var b strings.Builder
	last := -1
	for _, s := range steps {
		if s.Kind != "farm" || s.Area == last {
			continue
		}
		fmt.Fprintf(&b, "%d>", s.Area)
		last = s.Area
	}
	return b.String()
}

func topDistinct(finals []*node, k int) []Best {
	sort.Slice(finals, func(i, j int) bool { return finals[i].st.XP > finals[j].st.XP })
	seen := map[string]bool{}
	var out []Best
	for _, n := range finals {
		steps := n.path()
		sg := signature(steps)
		if seen[sg] {
			continue
		}
		seen[sg] = true
		out = append(out, Best{XP: n.st.XP, Clvl: n.st.Clvl(), Steps: steps, End: n.st, DenOK: n.st.Den, BROK: n.st.BR})
		if len(out) == k {
			break
		}
	}
	return out
}

// Signature of a Best (the farm order), for tables.
func (b Best) Signature() string { return signature(b.Steps) }

// OptimiseSeeded is Optimise, but a feasible policy (today's plan: it breaks no floor and does the Den quest)
// that beats the beam's answer replaces it: the beam is approximate, a known strategy is a lower bound.
func (e *Env) OptimiseSeeded(H float64, width int, pol Policy) Best {
	b := e.Optimise(H, width)
	r := e.RunPolicy(pol, H)
	ok := len(r.Violations) == 0 && (r.End.Den || e.NoDenRequired) && (r.End.Clvl() < 9 || r.End.BR || e.NoDenRequired)
	if ok && r.End.XP > b.XP {
		b.XP, b.Clvl, b.Steps, b.End, b.DenOK, b.BROK = r.End.XP, r.End.Clvl(), r.Steps, r.End, r.End.Den, r.End.BR
		b.Seeded = true
	}
	return b
}
