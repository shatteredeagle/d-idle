package diffready

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// GoTTL is how long an unconsumed go stays valid: given today, it must not fire weeks later after many deaths.
var GoTTL = 7 * 24 * time.Hour

// Go is one recorded go: for one target, one hero, until Expires.
type Go struct {
	Target  string    `json:"target"`
	Hero    string    `json:"hero"`
	At      time.Time `json:"at"`
	Expires time.Time `json:"expires"`
}

// Consent holds the owner's go for a hop. The bot computes readiness and waits; it never hops on its own (user
// decision 2026-11-27). A go is for one target and one hero, ends after GoTTL and is consumed by the hop. With Persist
// it survives a restart of the bot (a small JSON file) until it is consumed, ends, or Keep drops it because the
// difficulty moved on. A go that belongs to another hero or has ended is dropped with a log line, never used.
type Consent struct {
	// Logf, when set, receives the line about a dropped go.
	Logf func(msg string, args ...any)

	mu      sync.Mutex
	granted map[string]Go
	file    string
}

// Persist loads the gos kept in path (a missing file is none) and keeps the file current from now on.
func (c *Consent) Persist(path string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.file = path
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	var st struct {
		Granted []Go `json:"granted"`
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	c.granted = map[string]Go{}
	for _, g := range st.Granted {
		c.granted[g.Target] = g
	}
	return nil
}

// save writes the file (caller holds mu); an empty set removes it. Best effort: a failed write only means the go
// does not survive a restart, and the owner can give it again.
func (c *Consent) save() {
	if c.file == "" {
		return
	}
	if len(c.granted) == 0 {
		os.Remove(c.file)
		return
	}
	var gs []Go
	for _, g := range c.granted {
		gs = append(gs, g)
	}
	sort.Slice(gs, func(i, j int) bool { return gs[i].Target < gs[j].Target })
	b, _ := json.Marshal(struct {
		Granted []Go `json:"granted"`
	}{gs})
	tmp := c.file + ".tmp"
	if os.MkdirAll(filepath.Dir(c.file), 0o755) == nil && os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, c.file)
	}
}

// Grant records the owner's go for target ("nightmare" or "hell") and hero, valid for GoTTL from now.
func (c *Consent) Grant(target, hero string, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.granted == nil {
		c.granted = map[string]Go{}
	}
	c.granted[target] = Go{Target: target, Hero: hero, At: now.UTC(), Expires: now.Add(GoTTL).UTC()}
	c.save()
}

// valid reports a usable go for target and hero (caller holds mu); a go of another hero or an ended one is dropped.
func (c *Consent) valid(target, hero string, now time.Time) bool {
	g, ok := c.granted[target]
	if !ok {
		return false
	}
	why := ""
	switch {
	case g.Hero == "" || g.Hero != hero:
		why = "it was given for another hero"
	case !now.Before(g.Expires):
		why = "it has ended"
	default:
		return true
	}
	delete(c.granted, target)
	c.save()
	if c.Logf != nil {
		c.Logf("difficulty go ignored", "target", target, "why", why, "given_for", g.Hero, "hero", hero, "expires", g.Expires)
	}
	return false
}

// Granted says whether a usable go is waiting for target and hero.
func (c *Consent) Granted(target, hero string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.valid(target, hero, now)
}

// Take consumes the go for target and hero.
func (c *Consent) Take(target, hero string, now time.Time) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.valid(target, hero, now) {
		return false
	}
	delete(c.granted, target)
	c.save()
	return true
}

// Keep drops every go except the one for target ("" drops all): the difficulty changed, an old go is stale.
func (c *Consent) Keep(target string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := len(c.granted)
	for t := range c.granted {
		if t != target {
			delete(c.granted, t)
		}
	}
	if len(c.granted) != n {
		c.save()
	}
}
