package diffready

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

// Consent holds the owner's go for a hop. The bot computes readiness and waits; it never hops on its own (user
// decision 2026-11-27). A go is for one target and is consumed by the hop. With Persist the go survives a restart
// of the bot (a small JSON file) until it is consumed or Keep drops it because the difficulty moved on.
type Consent struct {
	mu      sync.Mutex
	granted map[string]bool
	file    string
}

// Persist loads the go kept in path (a missing file is none) and keeps the file current from now on.
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
		Granted []string `json:"granted"`
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return err
	}
	c.granted = map[string]bool{}
	for _, t := range st.Granted {
		c.granted[t] = true
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
	var ts []string
	for t := range c.granted {
		ts = append(ts, t)
	}
	sort.Strings(ts)
	b, _ := json.Marshal(struct {
		Granted []string `json:"granted"`
	}{ts})
	tmp := c.file + ".tmp"
	if os.MkdirAll(filepath.Dir(c.file), 0o755) == nil && os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, c.file)
	}
}

// Grant records the owner's go for target ("nightmare" or "hell").
func (c *Consent) Grant(target string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.granted == nil {
		c.granted = map[string]bool{}
	}
	c.granted[target] = true
	c.save()
}

// Granted says whether a go is waiting for target.
func (c *Consent) Granted(target string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.granted[target]
}

// Take consumes the go for target.
func (c *Consent) Take(target string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	ok := c.granted[target]
	if ok {
		delete(c.granted, target)
		c.save()
	}
	return ok
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
