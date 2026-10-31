package bot

import (
	"encoding/json"
	"os"
	"strings"
)

// Lifetime counters for the "minute" line. observer.kills counts since the bot process started; the minute line's
// `kills` is a per-character running total that also survives bot restarts: it is kept in a small file next to the
// death log and reloaded when the character name matches (a new character starts at 0). Game restarts never reset
// the observer, so within one bot process the total only grows.

type counterFile struct {
	Name  string `json:"name"`
	Kills int64  `json:"kills"`
}

func (r *Runner) counterPath() string {
	p := r.cfg.DeathLogPath
	if p == "" {
		return ""
	}
	return strings.TrimSuffix(p, ".json") + ".counters.json"
}

// lifetimeKills returns the per-character kill total and persists it. Caller holds obMu.
func (r *Runner) lifetimeKills(name string) int64 {
	o := &r.ob
	if !o.cInit || o.cName != name {
		o.cBase, o.cMark = 0, 0
		if o.cInit { // the character changed mid-run: earlier kills belonged to the old one
			o.cMark = o.kills
		}
		o.cInit, o.cName = true, name
		if path := r.counterPath(); path != "" {
			if b, err := os.ReadFile(path); err == nil {
				var c counterFile
				if json.Unmarshal(b, &c) == nil && c.Name == name {
					o.cBase = c.Kills
				}
			}
		}
	}
	total := o.cBase + int64(o.kills-o.cMark)
	if path := r.counterPath(); path != "" && total != o.cSaved {
		if b, err := json.Marshal(counterFile{Name: name, Kills: total}); err == nil && os.WriteFile(path, b, 0o644) == nil {
			o.cSaved = total
		}
	}
	return total
}
