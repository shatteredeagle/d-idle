package invdesk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
)

// Verb-first equip (docs/design/item-verbs.md). A backend that advertises the item verbs in its capabilities does the
// player's equip through the server's own item handlers in one step; every other case, and every failure that is not a
// rule refusal, goes through the click path (the Equipper it wraps), which stays the oracle and the only path on
// Game.exe. Nothing here writes game memory. How a request was done is reported per action ("verb" or "click").

// How values reported by Howser.
const (
	HowVerb  = "verb"
	HowClick = "click"
)

// CapItemEquip is the capability name of the equip verb (d2dctl.CapItemEquip).
const CapItemEquip = "item.equip"

// ItemVerbs is the backend's item verb surface (d2dctl.ItemVerbs adapts a client).
type ItemVerbs interface {
	// Caps lists what the connected backend can do; a backend without the verb returns nothing and no error.
	Caps(ctx context.Context) ([]string, error)
	// ItemEquip wears backpack item id on body slot (0: the backend picks). noop: it was already worn there. A game-rule
	// refusal is an error that implements RuleRefusal.
	ItemEquip(ctx context.Context, id uint32, slot int) (noop bool, err error)
}

// RuleRefusal is implemented by errors that say the game's rules refuse the action (d2dctl.ItemRefused): the click path
// would fail the same way, so there is no fallback.
type RuleRefusal interface{ RuleRefusal() bool }

// Howser is implemented by an Equipper that can say how the last PlayerEquip was done.
type Howser interface{ LastHow() string }

// VerbEquipper is an Equipper that tries the backend's equip verb first.
type VerbEquipper struct {
	Click  Equipper                                             // the click path; checks always come from here
	Verbs  ItemVerbs                                            // nil: always click
	On     func() bool                                          // the knob invdesk.verbs; nil: off
	Verify func(ctx context.Context, id uint32, body int) error // proves from a fresh snapshot that id is worn on body; nil: trust the reply
	Now    func() time.Time

	mu       sync.Mutex
	capsAt   time.Time
	capsOK   bool
	lastHow  string
	capsLife time.Duration
}

// NewVerbEquipper wraps the click path.
func NewVerbEquipper(click Equipper, verbs ItemVerbs, on func() bool) *VerbEquipper {
	return &VerbEquipper{Click: click, Verbs: verbs, On: on, Now: time.Now, capsLife: time.Minute}
}

// CheckPlayerEquip is the bot's own pre-check (class, level, stats, identified, hands): the same for both paths.
func (v *VerbEquipper) CheckPlayerEquip(s *d2.Snapshot, id uint32, body int) (string, error) {
	return v.Click.CheckPlayerEquip(s, id, body)
}

// LastHow says how the last PlayerEquip was done.
func (v *VerbEquipper) LastHow() string {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.lastHow
}

func (v *VerbEquipper) setHow(h string) { v.mu.Lock(); v.lastHow = h; v.mu.Unlock() }

// hasCap reads the capability list at most once a minute (a restarted d2d may be another build).
func (v *VerbEquipper) hasCap(ctx context.Context) bool {
	v.mu.Lock()
	if !v.capsAt.IsZero() && v.Now().Sub(v.capsAt) < v.capsLife {
		ok := v.capsOK
		v.mu.Unlock()
		return ok
	}
	v.mu.Unlock()
	caps, err := v.Verbs.Caps(ctx)
	if err != nil {
		return false // do not cache a transport error
	}
	ok := false
	for _, c := range caps {
		if c == CapItemEquip {
			ok = true
		}
	}
	v.mu.Lock()
	v.capsAt, v.capsOK = v.Now(), ok
	v.mu.Unlock()
	return ok
}

func (v *VerbEquipper) dropCaps() { v.mu.Lock(); v.capsAt, v.capsOK = v.Now(), false; v.mu.Unlock() }

// PlayerEquip wears pack item id on body slot body: the verb when the knob is on and the backend advertises it, else
// the click path. A rule refusal from the verb is returned as is.
func (v *VerbEquipper) PlayerEquip(ctx context.Context, id uint32, body int) error {
	if v.Verbs == nil || v.On == nil || !v.On() || !v.hasCap(ctx) {
		v.setHow(HowClick)
		return v.Click.PlayerEquip(ctx, id, body)
	}
	noop, err := v.Verbs.ItemEquip(ctx, id, body)
	var refusal RuleRefusal
	switch {
	case err == nil:
		if !noop && v.Verify != nil {
			if verr := v.Verify(ctx, id, body); verr != nil {
				v.setHow(HowVerb)
				return fmt.Errorf("the item verb answered ok but the item is not worn: %w", verr)
			}
		}
		v.setHow(HowVerb)
		return nil
	case errors.As(err, &refusal) && refusal.RuleRefusal():
		v.setHow(HowVerb)
		return err
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		return err
	}
	// Not supported after all, or a transport hiccup: the click path takes over (the verb did not move anything).
	v.dropCaps()
	v.setHow(HowClick)
	return v.Click.PlayerEquip(ctx, id, body)
}

var _ Equipper = (*VerbEquipper)(nil)
var _ Howser = (*VerbEquipper)(nil)
