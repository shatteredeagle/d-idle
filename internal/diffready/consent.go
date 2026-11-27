package diffready

import "sync"

// Consent holds the owner's go for a hop. The bot computes readiness and waits; it never hops on its own (user
// decision 2026-11-27). A go is for one target and is consumed by the hop. In process memory only (U: a restart asks again).
type Consent struct {
	mu      sync.Mutex
	granted map[string]bool
}

// Grant records the owner's go for target ("nightmare" or "hell").
func (c *Consent) Grant(target string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.granted == nil {
		c.granted = map[string]bool{}
	}
	c.granted[target] = true
}

// Granted says whether a go is waiting for target; Take also consumes it.
func (c *Consent) Granted(target string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.granted[target]
}

func (c *Consent) Take(target string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ok := c.granted[target]
	delete(c.granted, target)
	return ok
}
