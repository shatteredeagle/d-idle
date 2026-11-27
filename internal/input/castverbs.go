package input

import (
	"errors"
	"fmt"
	"time"
)

// Cast verbs (card dlv-cast-verbs; docs/design/input-cast-verbs.md). A cast is select and cast fused into one control
// message ("cmd skill <id> x y [unit] [left]" on the d.idle fork). The bot sends them as input only behind the knob
// input.planner and only when the connected build lists the capability; until then the legacy cast path acts alone.
// U: no fork build lists these names yet; the fused-versus-select-then-cast differential (question Q3) comes first.
const (
	CapCastUnit  = "input.cast-unit"  // skill on a unit (a living monster or a corpse) by unit id
	CapCastPoint = "input.cast-point" // skill on a ground point, no unit
	CapCastHold  = "input.cast-hold"  // skill kept pressed for a duration, with an explicit release
)

// Cast is one cast request: the skill, the hand, where, and how long to keep it pressed.
type Cast struct {
	Skill int
	Left  bool
	At    Point
	// Unit is the target unit id; 0 casts at the point.
	Unit uint32
	// Hold > 0 keeps the skill pressed this long, then releases (repeat and channel presses).
	Hold time.Duration
}

// Cap is the capability the request needs.
func (c Cast) Cap() string {
	switch {
	case c.Hold > 0:
		return CapCastHold
	case c.Unit != 0:
		return CapCastUnit
	}
	return CapCastPoint
}

// Name is the action name in the planner's log lines (the capability without its prefix).
func (c Cast) Name() string { return c.Cap()[len("input."):] }

// Caster is the verb side of a tap cast. The answer is only a request; the expectation ledger reads the effect.
type Caster interface {
	CastUnit(skill int, left bool, at Point, unit uint32) error
	CastPoint(skill int, left bool, at Point) error
}

// HoldCaster presses a skill for d and releases it before it returns, also on error.
type HoldCaster interface {
	CastHold(skill int, left bool, at Point, unit uint32, d time.Duration) error
}

// ErrNoCastVerb is returned when the backend has no verb for the request (the caller's legacy path then acts).
var ErrNoCastVerb = errors.New("input: backend has no cast verb for this request")

// Send sends c through the verbs v has.
func (c Cast) Send(v any) error {
	if c.Hold > 0 {
		h, ok := v.(HoldCaster)
		if !ok {
			return fmt.Errorf("%s: %w", c.Name(), ErrNoCastVerb)
		}
		return h.CastHold(c.Skill, c.Left, c.At, c.Unit, c.Hold)
	}
	k, ok := v.(Caster)
	if !ok {
		return fmt.Errorf("%s: %w", c.Name(), ErrNoCastVerb)
	}
	if c.Unit != 0 {
		return k.CastUnit(c.Skill, c.Left, c.At, c.Unit)
	}
	return k.CastPoint(c.Skill, c.Left, c.At)
}

// RunCast sends one cast: the verb when the planner is on and the build lists c.Cap(), else legacy. With the knob off
// nothing is counted (the minute line stays as it was); on, a legacy cast counts as legacyPath. It never acts twice:
// only a verb that was not sent (an error reply, an older build, no verb on this backend) falls back to legacy; a sent
// verb is the cast, and its effect is the expectation ledger's to judge, not a reason to cast again by another path. verify may be nil (the channel's acceptance is the postcondition).
func (p *Planner) RunCast(c Cast, verbs any, legacy func() error, legacyPath Path, verify func() (bool, error)) (Path, error) {
	if !p.On() {
		return legacyPath, legacy()
	}
	if !p.HasCap(c.Cap()) {
		if err := legacy(); err != nil {
			return "", err
		}
		p.Count(legacyPath)
		return legacyPath, nil
	}
	a := Action{Name: c.Name(), Cap: c.Cap(), Verb: func() error { return c.Send(verbs) }, Verify: verify, Once: true}
	switch legacyPath {
	case PathClick:
		a.Click = legacy
	default:
		a.Key = legacy
	}
	return p.Run(a)
}
