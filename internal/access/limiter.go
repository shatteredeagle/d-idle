package access

import (
	"sync"
	"time"
)

// Limiter counts failed logins per client: Max failures inside Window lock that client for Lockout. A success
// clears the client. In memory only; a restart clears it (acceptable: a restart needs access to the container).
type Limiter struct {
	Max     int
	Window  time.Duration
	Lockout time.Duration
	Now     func() time.Time

	mu sync.Mutex
	m  map[string]*client
}

type client struct {
	fails  []time.Time
	locked time.Time // zero or the end of the lockout
}

// NewLimiter is the project default: 5 failures in a minute lock the client for 5 minutes.
func NewLimiter() *Limiter {
	return &Limiter{Max: 5, Window: time.Minute, Lockout: 5 * time.Minute, Now: time.Now, m: map[string]*client{}}
}

const maxClients = 4096

// Allowed reports whether key may try now, and when it may try again if not.
func (l *Limiter) Allowed(key string) (ok bool, retry time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()
	c := l.m[key]
	if c == nil {
		return true, 0
	}
	if now := l.Now(); now.Before(c.locked) {
		return false, c.locked.Sub(now)
	}
	return true, 0
}

// Fail records a failed attempt and starts the lockout when the count reaches Max.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.Now()
	if len(l.m) >= maxClients {
		for k, c := range l.m {
			if now.After(c.locked) && (len(c.fails) == 0 || now.Sub(c.fails[len(c.fails)-1]) > l.Window) {
				delete(l.m, k)
			}
		}
		if len(l.m) >= maxClients {
			l.m = map[string]*client{} // under attack from many clients: forget, never grow
		}
	}
	c := l.m[key]
	if c == nil {
		c = &client{}
		l.m[key] = c
	}
	kept := c.fails[:0]
	for _, t := range c.fails {
		if now.Sub(t) <= l.Window {
			kept = append(kept, t)
		}
	}
	c.fails = append(kept, now)
	if len(c.fails) >= l.Max {
		c.locked, c.fails = now.Add(l.Lockout), nil
	}
}

// Success forgets the client's failures.
func (l *Limiter) Success(key string) {
	l.mu.Lock()
	delete(l.m, key)
	l.mu.Unlock()
}
