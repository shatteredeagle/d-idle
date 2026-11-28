package arbiter

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// ShadowConfig holds the shadow's own knobs.
type ShadowConfig struct {
	LogGapSec float64 // minimum virtual seconds between two log lines of the same (chain, arbiter) pair
}

// FamilyCount is one hook family's tally since the last Drain.
type FamilyCount struct{ Ticks, Diverged int }

// Shadow runs an Arbiter beside the if-chain. It only observes: Tick returns text, never an action. It recovers from
// any panic of a source and is safe to call from the goal loop (one short mutex, no I/O, no blocking).
type Shadow struct {
	mu      sync.Mutex
	arb     *Arbiter
	cfg     ShadowConfig
	counts  map[string]*FamilyCount
	lastLog map[string]float64
	panics  int
	last    Decision
	has     bool
}

func NewShadow(cfg Config, sc ShadowConfig, sources ...Source) *Shadow {
	return &Shadow{arb: New(cfg, sources...), cfg: sc, counts: map[string]*FamilyCount{}, lastLog: map[string]float64{}}
}

// Tick asks the arbiter what it would choose now and compares its family with chain (the family the chain acted for
// this tick, "" when unknown). It returns the divergence line to log, or "" when they agree, the chain family is
// unknown, or the line is rate-limited.
func (s *Shadow) Tick(v View, chain string) (line string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if recover() != nil {
			s.panics++
			line = ""
		}
	}()
	d := s.arb.Decide(v)
	s.last, s.has = d, true
	return s.compare(d, v.Now, chain)
}

// Decide asks the arbiter without comparing (the control path needs the decision BEFORE the chain acts). ok is false
// when a source panicked: the caller then takes the old path. Pair it with Compare after the chain acted.
func (s *Shadow) Decide(v View) (d Decision, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if recover() != nil {
			s.panics++
			d, ok = Decision{}, false
		}
	}()
	d = s.arb.Decide(v)
	s.last, s.has = d, true
	return d, true
}

// Compare is the second half of Tick for a decision taken with Decide.
func (s *Shadow) Compare(d Decision, now float64, chain string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.compare(d, now, chain)
}

func (s *Shadow) compare(d Decision, now float64, chain string) string {
	if chain == "" {
		return ""
	}
	c := s.counts[chain]
	if c == nil {
		c = &FamilyCount{}
		s.counts[chain] = c
	}
	c.Ticks++
	mine := "nothing"
	if d.Chosen != nil {
		mine = Family(d.Chosen.Source)
	}
	if mine == chain {
		return ""
	}
	c.Diverged++
	key := chain + ">" + mine
	if t, ok := s.lastLog[key]; ok && now-t < s.cfg.LogGapSec {
		return ""
	}
	s.lastLog[key] = now
	return fmt.Sprintf("chain chose %s, arbiter would choose %s", chain, d.Compact()) + because(d)
}

// because adds the rejection reasons of the arbiter's own choice when they explain the difference.
func because(d Decision) string {
	var parts []string
	for _, r := range d.Rejected {
		parts = append(parts, r.Intent.Key()+": "+r.Why)
		if len(parts) == 2 {
			break
		}
	}
	if len(parts) == 0 {
		return ""
	}
	return " because " + strings.Join(parts, "; ")
}

// Last is the arbiter's latest decision (the watch API reads this), false before the first tick.
func (s *Shadow) Last() (Decision, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.last, s.has
}

// Drain returns the per-family counters since the previous Drain as "family=diverged/ticks" pairs (sorted, families
// with ticks only) plus the panic count, and resets them.
func (s *Shadow) Drain() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	var names []string
	for n := range s.counts {
		names = append(names, n)
	}
	sort.Strings(names)
	var b strings.Builder
	for _, n := range names {
		fmt.Fprintf(&b, "%s=%d/%d ", n, s.counts[n].Diverged, s.counts[n].Ticks)
	}
	fmt.Fprintf(&b, "panics=%d", s.panics)
	s.counts = map[string]*FamilyCount{}
	return b.String()
}

// Report tells the arbiter how an intent ended (see Arbiter.Report).
func (s *Shadow) Report(in Intent, now float64, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.arb.Report(in, now, ok)
}

// Decline tells the arbiter that the chosen intent's hook had nothing to do (see Arbiter.Decline).
func (s *Shadow) Decline(in Intent, now float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.arb.Decline(in, now)
}

// Refused posts the temporary intent a typed refusal asks for (see Arbiter.Refused); false when none applies.
func (s *Shadow) Refused(need, wake string, now float64) (posted bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	defer func() {
		if recover() != nil {
			s.panics++
			posted = false
		}
	}()
	return s.arb.Refused(need, wake, now)
}
