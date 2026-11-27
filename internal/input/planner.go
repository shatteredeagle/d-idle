package input

import (
	"errors"
	"fmt"
	"sync"
)

// Path is the mechanism an action ran through, best first: a control-channel verb (the game's own client message),
// then a key, then a click whose effect is read back (docs/design/fable-review-2026-11-19/07-input-layer.md 4.2).
type Path string

const (
	PathVerb  Path = "verb"
	PathKey   Path = "key"
	PathClick Path = "click"
)

// Action is one intent with its way of doing it per path. A nil Do is a path the caller does not have. Verify is the
// postcondition read from game state: a path counts only when it returns true. Cap names the capability a verb needs
// (a name from the backend's caps); the verb path is skipped unless the planner's capability set lists it. A nil
// Verify accepts every path that was sent without error. Once ends the run at the first path that was sent without
// error, verified or not: an action whose repeat by another path would act twice (a cast) sets it.
type Action struct {
	Name   string
	Cap    string
	Verb   func() error
	Key    func() error
	Click  func() error
	Verify func() (bool, error)
	Once   bool
}

// ErrNoPath is returned when no path was available or none reached the postcondition.
var ErrNoPath = errors.New("input: no path reached the postcondition")

// Planner chooses verb > key > verified click per action. It is opt-in: with Enabled false (or nil) Run is not used
// by callers, who keep their legacy code (the knob input.planner, default 0). Safe for concurrent use.
type Planner struct {
	// Enabled reports the knob; nil means off.
	Enabled func() bool
	// Caps is the backend capability set; nil means no verbs.
	Caps *BackendCaps
	// CapsFn, when set, supplies the capability names the connected backend advertises (the "caps" reply); it is
	// asked once, on the first Run that needs it, and its result is merged with Caps.
	CapsFn func() []string
	// Logf receives one line per fallback and per final path (nil: silent).
	Logf func(msg string, kv ...any)

	mu     sync.Mutex
	asked  bool
	live   map[string]bool
	counts [3]int
	fail   int
}

// On reports whether the planner is enabled.
func (p *Planner) On() bool { return p != nil && p.Enabled != nil && p.Enabled() }

func idx(x Path) int {
	switch x {
	case PathVerb:
		return 0
	case PathKey:
		return 1
	}
	return 2
}

// has says whether the verb capability name is available.
func (p *Planner) has(name string) bool {
	if name == "" {
		return true
	}
	if p.Caps != nil && p.Caps.Has(name) {
		return true
	}
	p.mu.Lock()
	if !p.asked && p.CapsFn != nil {
		p.asked = true
		p.mu.Unlock()
		names := p.CapsFn()
		p.mu.Lock()
		p.live = map[string]bool{}
		for _, n := range names {
			p.live[n] = true
		}
	}
	defer p.mu.Unlock()
	return p.live[name]
}

// Run tries the available paths in order and returns the one whose postcondition held. A path that errors or does not
// verify is logged and the next one is tried; the last error is returned (wrapped in ErrNoPath when nothing failed
// loudly). Every success is counted.
func (p *Planner) Run(a Action) (Path, error) {
	type step struct {
		path Path
		do   func() error
	}
	steps := []step{{PathVerb, a.Verb}, {PathKey, a.Key}, {PathClick, a.Click}}
	var last error
	for _, s := range steps {
		if s.do == nil || (s.path == PathVerb && !p.has(a.Cap)) {
			continue
		}
		err := s.do()
		sent, ok := err == nil, err == nil
		if sent && a.Verify != nil {
			ok, err = a.Verify()
		}
		if err == nil && ok {
			p.mu.Lock()
			p.counts[idx(s.path)]++
			p.mu.Unlock()
			if p.Logf != nil {
				p.Logf("input path", "action", a.Name, "path", string(s.path))
			}
			return s.path, nil
		}
		if err == nil {
			err = ErrNoPath
		}
		last = err
		if p.Logf != nil {
			p.Logf("input path failed", "action", a.Name, "path", string(s.path), "err", err.Error())
		}
		if sent && a.Once {
			break
		}
	}
	p.mu.Lock()
	p.fail++
	p.mu.Unlock()
	if last == nil {
		last = ErrNoPath
	}
	return "", fmt.Errorf("%s: %w", a.Name, last)
}

// Count records one use of a path done outside Run (legacy code under the knob off keeps its behaviour and only counts).
func (p *Planner) Count(x Path) {
	if p == nil {
		return
	}
	p.mu.Lock()
	p.counts[idx(x)]++
	p.mu.Unlock()
}

// Counts returns how often each path was used, and how many actions found no path.
func (p *Planner) Counts() (verb, key, click, failed int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.counts[0], p.counts[1], p.counts[2], p.fail
}

// Summary is the minute-line value "verb/key/click" (input=3/0/7).
func (p *Planner) Summary() string {
	v, k, c, _ := p.Counts()
	return fmt.Sprintf("%d/%d/%d", v, k, c)
}
