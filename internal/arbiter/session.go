package arbiter

import "sort"

// Session scoring (docs/arbiter.md, "Outcome over a recorded session"): the outcome rules of an OutcomeFile applied to
// every tick of a replayed log instead of one decision point. Both sides see the logged facts; the chain side is the
// family the log says the chain acted for, the arbiter side is what the shadow chose on that tick (with its lease,
// back-off and travel budget, so a whole session is not the same as many independent cases).

// SessionPoint is one replayed tick.
type SessionPoint struct {
	Now     float64
	Facts   map[string]int
	Chain   string // "" when the chain did not act or its family is unknown
	Arbiter string // "" when the arbiter chose nothing
}

// ScoreSession turns the ticks into case scores, one per tick on which either side chose something. A hit counts as an
// EPISODE: it is counted on the tick it starts for that side and not again while it holds on the following ticks, which
// is how the monitor logs a row once per state change (U: a model of the monitor, as the rules are). A tick counts for
// the exit-rule family of the arbiter when the two differ and the arbiter chose something, else for the chain's.
// NewKinds are judged over the whole session: an arbiter hit whose kind the chain never hit in the session.
func (f OutcomeFile) ScoreSession(pts []SessionPoint) []CaseScore {
	var out []CaseScore
	chainSeen := map[string]bool{}
	var prevC, prevA map[string]bool
	for _, p := range pts {
		chain, mine := orNothing(p.Chain), orNothing(p.Arbiter)
		hc, ha := f.Hits(chain, p.Facts), f.Hits(mine, p.Facts)
		for _, h := range hc {
			chainSeen[h] = true
		}
		nc, na := fresh(hc, prevC), fresh(ha, prevA)
		prevC, prevA = set(hc), set(ha)
		if chain == "nothing" && mine == "nothing" {
			continue
		}
		fam := chain
		if mine != chain && mine != "nothing" {
			fam = mine
		}
		out = append(out, CaseScore{Family: fam, Chain: chain, Arbiter: mine, ChainHits: nc, ArbiterHits: na})
	}
	for i := range out {
		for _, h := range out[i].ArbiterHits {
			if !chainSeen[h] {
				out[i].NewKinds = append(out[i].NewKinds, h)
			}
		}
	}
	return out
}

func orNothing(f string) string {
	if f == "" {
		return "nothing"
	}
	return f
}

func set(l []string) map[string]bool {
	m := make(map[string]bool, len(l))
	for _, x := range l {
		m[x] = true
	}
	return m
}

func fresh(l []string, prev map[string]bool) []string {
	var out []string
	for _, x := range l {
		if !prev[x] {
			out = append(out, x)
		}
	}
	return out
}

// MergeRules joins the rules of several outcome files, by name (the first file that names a rule wins), sorted.
func MergeRules(files ...OutcomeFile) OutcomeFile {
	seen := map[string]bool{}
	m := OutcomeFile{Version: OutcomeVersion}
	for _, f := range files {
		for _, r := range f.Rules {
			if !seen[r.Name] {
				seen[r.Name] = true
				m.Rules = append(m.Rules, r)
			}
		}
	}
	sort.Slice(m.Rules, func(i, j int) bool { return m.Rules[i].Name < m.Rules[j].Name })
	return m
}
