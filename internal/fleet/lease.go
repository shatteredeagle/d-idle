package fleet

import (
	"fmt"
	"time"
)

// Lease is in-process ownership of one slot's input and control: two callers never drive one instance. It
// replaces the /tmp flock of the dev tools in product mode (design-gaps D8). A lease expires after its TTL
// unless renewed, so a crashed holder cannot block a slot forever.
type Lease struct {
	ID      string // slot id
	Owner   string
	Expires time.Time

	f   *Fleet
	tok uint64
}

// LeasedError says who holds the slot.
type LeasedError struct {
	ID, Holder string
	Until      time.Time
}

func (e *LeasedError) Error() string {
	return fmt.Sprintf("slot %s is leased by %s until %s", e.ID, e.Holder, e.Until.Format(time.RFC3339))
}
func (e *LeasedError) Unwrap() error { return ErrLeased }

func (f *Fleet) liveLeaseLocked(s *slot) *Lease {
	if s.lease != nil && f.cfg.Clock().Before(s.lease.Expires) {
		return s.lease
	}
	s.lease = nil
	return nil
}

// Lease takes the slot for owner for ttl. A slot that is draining, failed or unknown cannot be leased; one
// that somebody else holds returns a *LeasedError (errors.Is ErrLeased). The same owner asking again is
// refused too: it must Renew, so a forgotten holder is visible instead of silently shared.
func (f *Fleet) Lease(id, owner string, ttl time.Duration) (*Lease, error) {
	if owner == "" || ttl <= 0 {
		return nil, fmt.Errorf("fleet: lease needs an owner and a positive ttl")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.slots[id]
	if !ok {
		return nil, ErrNoSlot
	}
	if s.state == Draining {
		return nil, fmt.Errorf("fleet: slot %s is draining", id)
	}
	if l := f.liveLeaseLocked(s); l != nil {
		return nil, &LeasedError{ID: id, Holder: l.Owner, Until: l.Expires}
	}
	f.leaseN++
	l := &Lease{ID: id, Owner: owner, Expires: f.cfg.Clock().Add(ttl), f: f, tok: f.leaseN}
	s.lease = l
	return l, nil
}

// Valid reports whether this lease is still the slot's current one.
func (l *Lease) Valid() bool {
	l.f.mu.Lock()
	defer l.f.mu.Unlock()
	return l.validLocked()
}

func (l *Lease) validLocked() bool {
	s, ok := l.f.slots[l.ID]
	return ok && s.lease != nil && s.lease.tok == l.tok && l.f.cfg.Clock().Before(s.lease.Expires)
}

// Renew extends an unexpired, still-current lease; an expired or revoked one cannot be revived.
func (l *Lease) Renew(ttl time.Duration) error {
	l.f.mu.Lock()
	defer l.f.mu.Unlock()
	if !l.validLocked() {
		return fmt.Errorf("fleet: lease on %s is no longer valid", l.ID)
	}
	l.Expires = l.f.cfg.Clock().Add(ttl)
	return nil
}

// Release frees the slot; releasing a stale lease (expired, revoked, replaced) does nothing.
func (l *Lease) Release() {
	l.f.mu.Lock()
	defer l.f.mu.Unlock()
	if s, ok := l.f.slots[l.ID]; ok && s.lease != nil && s.lease.tok == l.tok {
		s.lease = nil
	}
}
