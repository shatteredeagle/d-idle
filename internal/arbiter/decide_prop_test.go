package arbiter

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
)

// Property tests on Decide (roadmap T3, docs/design/proptest-t3.md): random sources, reports, area changes and refusals
// over a run of ticks, one fixed seed per build. No clock: View.Now is virtual.

const (
	propRuns  = 300
	propTicks = 200
)

// propBuild is the input shape of one build: how many potions it wants before it leaves town and how many it carries.
// Labels only; the arbiter takes no class.
type propBuild struct {
	name               string
	potionsMin, potMax int
}

var propBuilds = []propBuild{{"sorc", 4, 12}, {"pal", 8, 16}}

var propAreas = []string{Town, "cold_plains", "dark_wood", "tristram"}

// randSource proposes, each tick, nothing or one intent with a random kind, class, urgency and area.
func randSource(r *rand.Rand, name string) Source {
	return FuncSource{N: name, F: func(View) *Intent {
		if r.IntN(3) == 0 {
			return nil
		}
		area := Any
		if r.IntN(2) == 0 {
			area = propAreas[r.IntN(len(propAreas))]
		}
		return &Intent{
			Kind:      fmt.Sprintf("k%d", r.IntN(3)),
			Class:     Class(r.IntN(int(Reflex) + 1)),
			Urgency:   float64(r.IntN(4)) / 4,
			NeedsArea: area,
			EstCost:   float64(r.IntN(100)),
			Resumable: r.IntN(2) == 0,
		}
	}}
}

func refused(why string) bool {
	return strings.HasPrefix(why, "backed off") || strings.HasPrefix(why, "travel budget")
}

// propRun drives one random run and checks every decision.
type propRun struct {
	t     *testing.T
	b     propBuild
	r     *rand.Rand
	a     *Arbiter
	cfg   Config
	src   []Source
	props map[string]Intent // what the sources proposed this tick, by key
	// starvation model: since when a key has been a candidate that lost, and whether Decide flagged it this episode
	since   map[string]float64
	flagged map[string]bool
	// temporary intents from refusals: key -> expiry (TTL), until the Until condition holds or a success report
	temps map[string]float64
}

func newPropRun(t *testing.T, b propBuild, r *rand.Rand) *propRun {
	cfg := Config{
		MinDwell:     float64(r.IntN(200)),
		TravelWindow: float64(r.IntN(900)),
		BackoffBase:  float64(1 + r.IntN(60)),
		BackoffMax:   float64(60 + r.IntN(600)),
		StarveSec:    float64(r.IntN(400)),
	}
	p := &propRun{t: t, b: b, r: r, cfg: cfg, since: map[string]float64{}, flagged: map[string]bool{}, temps: map[string]float64{}}
	for i := range 4 {
		p.src = append(p.src, &recSource{Source: randSource(r, fmt.Sprintf("s%d", i)), run: p})
	}
	p.a = New(cfg, p.src...)
	return p
}

// recSource records what its inner source proposed so the checks know the proposals.
type recSource struct {
	Source
	run *propRun
}

func (s *recSource) Propose(v View) *Intent {
	in := s.Source.Propose(v)
	if in != nil {
		c := *in
		c.Source = s.Name()
		s.run.props[c.Key()] = c
	}
	return in
}

func (p *propRun) run() {
	now, area, progress := 0.0, Town, uint64(0)
	potions := p.r.IntN(p.b.potMax + 1)
	for tick := range propTicks {
		now += float64(1 + p.r.IntN(30))
		if p.r.IntN(5) == 0 {
			area = propAreas[p.r.IntN(len(propAreas))]
		}
		if p.r.IntN(4) == 0 {
			progress++
		}
		potions = max(0, min(p.b.potMax, potions+p.r.IntN(5)-2))
		if p.r.IntN(10) == 0 {
			needs := []string{"needs_potions", "needs_gold_farm", "too_weak", "unknown_need"}
			need := needs[p.r.IntN(len(needs))]
			if p.a.Refused(need, "", now) {
				in := RefusalIntents[need]
				p.temps[in.Key()] = now + in.TTL
			}
		}
		facts := map[string]int{"potions": potions, "potions_min": p.b.potionsMin}
		for k, until := range p.temps { // the arbiter drops a temp at expiry or once its condition holds
			in := tempIntent(k)
			if now >= until || in.Until != "" && holds(in.Until, facts) {
				delete(p.temps, k)
			}
		}
		p.props = map[string]Intent{}
		d := p.a.Decide(View{Now: now, Area: area, Progress: progress, Facts: facts})
		p.check(tick, d)
		if p.t.Failed() {
			return
		}
		if d.Chosen != nil && p.r.IntN(4) == 0 {
			ok := p.r.IntN(2) == 0
			p.a.Report(*d.Chosen, now, ok)
			if ok {
				delete(p.temps, d.Chosen.Key())
			}
		}
	}
}

func tempIntent(key string) Intent {
	for _, in := range RefusalIntents {
		if in.Key() == key {
			return in
		}
	}
	return Intent{}
}

func (p *propRun) check(tick int, d Decision) {
	t := p.t
	fail := func(format string, args ...any) {
		t.Helper()
		t.Errorf("%s tick %d: %s\n%s", p.b.name, tick, fmt.Sprintf(format, args...), d.String())
	}
	seen := map[string]bool{}
	var cands []Intent
	if d.Chosen != nil {
		seen[d.Chosen.Key()] = true
		cands = append(cands, *d.Chosen)
	}
	for _, rj := range d.Rejected {
		seen[rj.Intent.Key()] = true
		if refused(rj.Why) {
			if rj.Intent.Class >= Player {
				fail("a %s intent was refused: %s", rj.Intent.Class, rj.Why)
			}
			continue
		}
		cands = append(cands, rj.Intent)
	}
	// every proposal and every live temporary intent is accounted for, and nothing else
	for k := range p.props {
		if !seen[k] {
			fail("proposal %s neither chosen nor rejected", k)
		}
	}
	for k := range p.temps {
		if !seen[k] {
			fail("live refusal intent %s not proposed", k)
		}
	}
	for k := range seen {
		_, prop := p.props[k]
		_, tmp := p.temps[k]
		if !prop && !tmp {
			fail("%s decided on but neither proposed nor live (a met refusal must not come back)", k)
		}
	}
	if (d.Chosen == nil) != (len(cands) == 0) {
		fail("chosen %v with %d admissible candidates", d.Chosen, len(cands))
	}
	if d.Chosen == nil {
		p.since, p.flagged = map[string]float64{}, map[string]bool{}
		return
	}
	// a reflex always wins, and in general only a higher class can win: the lease never holds against one
	for _, c := range cands {
		if c.Class > d.Chosen.Class {
			fail("%s (%s) lost to %s (%s)", c.Key(), c.Class, d.Chosen.Key(), d.Chosen.Class)
		}
	}
	// no intent starves unseen: a candidate outranked continuously for StarveSec is flagged once per episode
	limit := p.cfg.StarveSec
	if limit <= 0 {
		limit = 600
	}
	starved := map[string]bool{}
	for _, s := range d.Starved {
		starved[s.Key()] = true
	}
	lost := map[string]bool{}
	for _, c := range cands {
		k := c.Key()
		if k == d.Chosen.Key() {
			continue
		}
		lost[k] = true
		if _, ok := p.since[k]; !ok {
			p.since[k] = d.At
		}
		long := d.At-p.since[k] >= limit
		switch {
		case starved[k] && (!long || p.flagged[k]):
			fail("%s flagged starved after %.0fs (limit %.0f, already flagged %v)", k, d.At-p.since[k], limit, p.flagged[k])
		case long && !p.flagged[k] && !starved[k]:
			fail("%s outranked %.0fs (limit %.0f) and not flagged", k, d.At-p.since[k], limit)
		}
		p.flagged[k] = p.flagged[k] || starved[k]
	}
	for k := range p.since {
		if !lost[k] {
			delete(p.since, k)
			delete(p.flagged, k)
		}
	}
	for k := range starved {
		if !lost[k] {
			fail("%s flagged starved but did not lose", k)
		}
	}
}

func TestPropArbiterDecide(t *testing.T) {
	for i, b := range propBuilds {
		t.Run(b.name, func(t *testing.T) {
			r := rand.New(rand.NewPCG(uint64(980+i), 7))
			for range propRuns {
				newPropRun(t, b, r).run()
				if t.Failed() {
					return
				}
			}
		})
	}
}

// Decide is deterministic: the same seed gives the same trace.
func TestPropArbiterDeterministic(t *testing.T) {
	trace := func(seed uint64) string {
		r := rand.New(rand.NewPCG(seed, 7))
		p := newPropRun(t, propBuilds[0], r)
		var lines []string
		for tick := range propTicks {
			p.props = map[string]Intent{}
			d := p.a.Decide(View{Now: float64(tick * 10), Area: propAreas[tick%len(propAreas)], Facts: map[string]int{"potions": tick % 9, "potions_min": 4}})
			lines = append(lines, d.String())
			if d.Chosen != nil && tick%3 == 0 {
				p.a.Report(*d.Chosen, float64(tick*10), tick%2 == 0)
			}
		}
		return strings.Join(lines, "\n")
	}
	if a, b := trace(11), trace(11); a != b {
		t.Fatalf("same seed, different traces")
	}
}
