package invdesk

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
)

// Verb-first item actions beyond equip (docs/design/item-verbs.md section 8). The town asks a Mover at the one moment
// the verb is allowed: identify, drop and gold only when the click path would also do them, and every stash move or
// stash gold only while the town has the stash open at the chest (the fork refuses with "far" anywhere else, so the
// rule holds on both sides). A Mover answers done=false for "not mine, use the click path"; a rule refusal is done=true
// with the error. Nothing here writes game memory.

// Capability names (d2dctl.Cap*).
const (
	CapItemMove     = "item.move"
	CapItemIdentify = "item.identify"
	CapItemDrop     = "item.drop"
	CapGoldStash    = "gold.stash"
)

// Mover is what the town consults before it clicks.
type Mover interface {
	// Identify reads a tool on pack, worn or stash item id.
	Identify(ctx context.Context, id uint32) (done bool, err error)
	// Move carries item id between the pack and the open stash (first free spot).
	Move(ctx context.Context, id uint32, toStash bool) (done bool, err error)
	// Gold moves gold with the open stash; the sentence is the request's result.
	Gold(ctx context.Context, amount int, deposit bool) (sentence string, done bool, err error)
	// Drop puts pack item id on the ground.
	Drop(ctx context.Context, id uint32) (done bool, err error)
}

// HowSource says how an action was done ("verb" or "click"); the desk reads it when a request settles. The entry is
// consumed by the read.
type HowSource interface {
	HowOf(kind string, id uint32) string
}

// MoverVerbs is the backend surface VerbMover needs (d2dctl.ItemVerbs).
type MoverVerbs interface {
	Caps(ctx context.Context) ([]string, error)
	ItemMove(ctx context.Context, id uint32, panel string, col, row int) (noop bool, err error)
	ItemIdentify(ctx context.Context, id uint32) (noop bool, err error)
	ItemDrop(ctx context.Context, id uint32) error
	GoldStash(ctx context.Context, amount int, deposit bool) (bank, carried int, err error)
}

// VerbMover implements Mover and HowSource on a backend that may offer the verbs.
type VerbMover struct {
	Verbs MoverVerbs
	On    func() bool // the knob invdesk.verbs
	// Fresh reads a snapshot taken now; Settle polls it for the end state after a verb (the verb answers "queued").
	Fresh func() (*d2.Snapshot, error)
	Pause func(ctx context.Context, d time.Duration) error
	Now   func() time.Time

	mu      sync.Mutex
	caps    map[string]bool
	capsAt  time.Time
	hows    map[string]string
	TTL     time.Duration
	Settles int // polls of 100 ms; 0 = 25
}

// NewVerbMover builds a mover; wire Fresh before use.
func NewVerbMover(v MoverVerbs, on func() bool) *VerbMover {
	return &VerbMover{Verbs: v, On: on, Now: time.Now, TTL: time.Minute, hows: map[string]string{}}
}

func hkey(kind string, id uint32) string { return fmt.Sprintf("%s:%d", kind, id) }

func (m *VerbMover) note(kind string, id uint32, how string) {
	m.mu.Lock()
	m.hows[hkey(kind, id)] = how
	m.mu.Unlock()
}

// HowOf is HowSource.
func (m *VerbMover) HowOf(kind string, id uint32) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	k := hkey(kind, id)
	h := m.hows[k]
	delete(m.hows, k)
	return h
}

func (m *VerbMover) has(ctx context.Context, name string) bool {
	if m == nil || m.Verbs == nil || m.On == nil || !m.On() {
		return false
	}
	m.mu.Lock()
	if m.caps != nil && m.Now().Sub(m.capsAt) < m.TTL {
		ok := m.caps[name]
		m.mu.Unlock()
		return ok
	}
	m.mu.Unlock()
	list, err := m.Verbs.Caps(ctx)
	if err != nil {
		return false
	}
	set := map[string]bool{}
	for _, c := range list {
		set[c] = true
	}
	m.mu.Lock()
	m.caps, m.capsAt = set, m.Now()
	m.mu.Unlock()
	return set[name]
}

func (m *VerbMover) dropCaps() {
	m.mu.Lock()
	m.caps, m.capsAt = map[string]bool{}, m.Now()
	m.mu.Unlock()
}

// failed classifies a verb error: done=true for a rule refusal (final) or a context error, false to fall back.
func (m *VerbMover) failed(err error) (done bool, out error) {
	var refusal RuleRefusal
	switch {
	case errors.As(err, &refusal) && refusal.RuleRefusal():
		return true, err
	case errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded):
		return true, err
	}
	m.dropCaps()
	return false, nil
}

// settle polls fresh snapshots until ok or the polls run out.
func (m *VerbMover) settle(ctx context.Context, ok func(*d2.Snapshot) bool) error {
	if m.Fresh == nil {
		return nil // no read-back wired: trust the reply
	}
	n := m.Settles
	if n == 0 {
		n = 25
	}
	for i := 0; i < n; i++ {
		s, err := m.Fresh()
		if err == nil && s != nil && ok(s) {
			return nil
		}
		pause := m.Pause
		if pause == nil {
			pause = func(ctx context.Context, d time.Duration) error {
				select {
				case <-time.After(d):
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		if err := pause(ctx, 100*time.Millisecond); err != nil {
			return err
		}
	}
	return errors.New("the item verb answered ok but the game did not show the change")
}

func findUnit(s *d2.Snapshot, id uint32) (d2.Unit, bool) {
	if s == nil || s.Inventory == nil {
		return d2.Unit{}, false
	}
	for _, u := range s.Inventory.Items {
		if u.ID == id {
			return u, true
		}
	}
	return d2.Unit{}, false
}

// Identify is Mover.
func (m *VerbMover) Identify(ctx context.Context, id uint32) (bool, error) {
	if !m.has(ctx, CapItemIdentify) {
		return false, nil
	}
	noop, err := m.Verbs.ItemIdentify(ctx, id)
	if err != nil {
		if done, out := m.failed(err); done {
			m.note("identify", id, HowVerb)
			return true, out
		}
		m.note("identify", id, HowClick)
		return false, nil
	}
	m.note("identify", id, HowVerb)
	if noop {
		return true, nil
	}
	return true, m.settle(ctx, func(s *d2.Snapshot) bool {
		u, ok := findUnit(s, id)
		return !ok || (u.Item != nil && u.Item.Identified())
	})
}

// Move is Mover.
func (m *VerbMover) Move(ctx context.Context, id uint32, toStash bool) (bool, error) {
	if !m.has(ctx, CapItemMove) {
		return false, nil
	}
	panel, want, kind := "inv", d2.ContInventory, "take"
	if toStash {
		panel, want, kind = "stash", d2.ContStash, "stash"
	}
	noop, err := m.Verbs.ItemMove(ctx, id, panel, -1, -1)
	if err != nil {
		if done, out := m.failed(err); done {
			m.note(kind, id, HowVerb)
			return true, out
		}
		m.note(kind, id, HowClick)
		return false, nil
	}
	m.note(kind, id, HowVerb)
	if noop {
		return true, nil
	}
	return true, m.settle(ctx, func(s *d2.Snapshot) bool {
		u, ok := findUnit(s, id)
		return ok && u.Item != nil && u.Item.Where == want
	})
}

// Gold is Mover.
func (m *VerbMover) Gold(ctx context.Context, amount int, deposit bool) (string, bool, error) {
	if !m.has(ctx, CapGoldStash) {
		return "", false, nil
	}
	bank, carried, err := m.Verbs.GoldStash(ctx, amount, deposit)
	if err != nil {
		if done, out := m.failed(err); done {
			m.note("gold", 0, HowVerb)
			return "", true, out
		}
		m.note("gold", 0, HowClick)
		return "", false, nil
	}
	m.note("gold", 0, HowVerb)
	err = m.settle(ctx, func(s *d2.Snapshot) bool {
		return s.Player != nil && s.Player.StashGold == bank && s.Player.Gold == carried
	})
	if err != nil {
		return "", true, err
	}
	if deposit {
		return fmt.Sprintf("deposited %d gold", amount), true, nil
	}
	return fmt.Sprintf("withdrew %d gold", amount), true, nil
}

// Drop is Mover.
func (m *VerbMover) Drop(ctx context.Context, id uint32) (bool, error) {
	if !m.has(ctx, CapItemDrop) {
		return false, nil
	}
	if err := m.Verbs.ItemDrop(ctx, id); err != nil {
		if done, out := m.failed(err); done {
			m.note("trash", id, HowVerb)
			return true, out
		}
		m.note("trash", id, HowClick)
		return false, nil
	}
	m.note("trash", id, HowVerb)
	return true, m.settle(ctx, func(s *d2.Snapshot) bool { _, ok := findUnit(s, id); return !ok })
}

// VerbTrasher wraps the click Trasher: the drop goes through the mover when it can.
type VerbTrasher struct {
	Click Trasher
	Mover Mover
}

// CheckPlayerTrash is the bot's own pre-check, unchanged.
func (v VerbTrasher) CheckPlayerTrash(s *d2.Snapshot, id uint32) (string, error) {
	return v.Click.CheckPlayerTrash(s, id)
}

// PlayerTrash drops by verb when the mover handles it, else clicks.
func (v VerbTrasher) PlayerTrash(ctx context.Context, id uint32) error {
	if v.Mover != nil {
		if done, err := v.Mover.Drop(ctx, id); done {
			return err
		}
	}
	return v.Click.PlayerTrash(ctx, id)
}

var _ Mover = (*VerbMover)(nil)
var _ HowSource = (*VerbMover)(nil)
var _ Trasher = VerbTrasher{}

// Equipped proves from fresh snapshots that item id is worn on body (the Verify hook of VerbEquipper).
func (m *VerbMover) Equipped(ctx context.Context, id uint32, body int) error {
	return m.settle(ctx, func(s *d2.Snapshot) bool {
		u, ok := findUnit(s, id)
		return ok && u.Item != nil && u.Item.Where == d2.ContEquipment && u.Item.Body == body
	})
}
