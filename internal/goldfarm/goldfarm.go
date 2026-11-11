// Package goldfarm is the player's "Farm gold" order (docs/goldfarm.md): a persisted on/off switch the watch screen
// drives, plus the status the bot reports back (zone, gold so far, target). The bot reads On at its area choice; nothing
// here touches the game.
package goldfarm

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"time"

	"d2lod-bot/internal/resource"
)

// StateFile is the persisted switch inside the bot's state directory.
const StateFile = "goldfarm-ui.json"

// Want is the gold one farming leg aims at: the repair the gear needs (repairNeed, 0 = none), the minimal survival kit
// and the poverty margin (resource.PovertyCheck leaves poverty at KitCost+PovertyMargin). All three are existing knobs
// and data; the order adds none. Farming goes on after the target is met; the target only drives the status line.
func Want(repairNeed int, t resource.Tuning) int {
	return max(repairNeed, 0) + resource.KitCost(t) + t.PovertyMargin
}

// Status is what the bot reports every pass.
type Status struct {
	Area   string // zone being farmed ("" = none yet)
	Have   int    // pack plus stash gold
	Want   int    // Want(...)
	Repair int    // the repair part of Want
	Note   string // why nothing is happening ("no open zone", ...), "" = fine
}

// View is the JSON of GET /api/farmgold, schema 1 (additive: new fields only).
type View struct {
	Schema  int    `json:"schema"`
	On      bool   `json:"on"`
	Since   int64  `json:"since,omitempty"`   // unix seconds the order was given
	Earned  int    `json:"earned,omitempty"`  // gold gained since the order (never negative)
	Area    string `json:"area,omitempty"`    // zone being farmed
	Have    int    `json:"have,omitempty"`    // pack plus stash gold
	Want    int    `json:"want,omitempty"`    // target of this leg
	Reached bool   `json:"reached,omitempty"` // Have >= Want: still farming until told to stop
	Note    string `json:"note,omitempty"`    // a hold reason, or why the order ended by itself
	Text    string `json:"text"`              // the one-line status for the button
	For     string `json:"for,omitempty"`     // who gave the order: "" the player's Farm gold button, "repair" a Farm until repair request
}

type stored struct {
	On       bool   `json:"on"`
	Since    int64  `json:"since,omitempty"`
	StartSet bool   `json:"startSet,omitempty"`
	Start    int    `json:"start,omitempty"`
	Ended    string `json:"ended,omitempty"` // why the last order stopped by itself
	For      string `json:"for,omitempty"`   // "repair": started by a repair request (docs/repair.md), it ends with that request
}

// Controller holds the switch. Safe for concurrent use.
type Controller struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
	st   stored
	cur  Status
}

// New loads the state file (path "" = nothing persisted; missing = off; damaged = error).
func New(path string, now func() time.Time) (*Controller, error) {
	if now == nil {
		now = time.Now
	}
	c := &Controller{path: path, now: now}
	if path == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(b, &c.st); err != nil {
			return nil, fmt.Errorf("goldfarm: %s: %w", path, err)
		}
	}
	return c, nil
}

func (c *Controller) save() error {
	if c.path == "" {
		return nil
	}
	b, _ := json.Marshal(c.st)
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

// On says whether the player's order is in force.
func (c *Controller) On() bool { c.mu.Lock(); defer c.mu.Unlock(); return c.st.On }

// Start gives the order (idempotent).
func (c *Controller) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.st.On {
		return nil
	}
	c.st = stored{On: true, Since: c.now().Unix()}
	c.cur = Status{}
	return c.save()
}

// ForRepair says the order belongs to a Farm until repair request: the target carries the repair x margin and the order
// ends itself when the request does.
func (c *Controller) ForRepair() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.st.On && c.st.For == "repair"
}

// StartForRepair gives the order on behalf of a repair request. An order the player already gave is taken over as
// repair-owned, which keeps the single status line honest; it ends with the request.
func (c *Controller) StartForRepair() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.st.On && c.st.For == "repair" {
		return nil
	}
	if !c.st.On {
		c.st = stored{On: true, Since: c.now().Unix()}
		c.cur = Status{}
	}
	c.st.For = "repair"
	return c.save()
}

// StopForRepair ends the order only when a repair request owns it (the player's own Farm gold order goes on).
func (c *Controller) StopForRepair() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.st.On || c.st.For != "repair" {
		return nil
	}
	c.st = stored{}
	c.cur = Status{}
	return c.save()
}

// Stop withdraws the order (idempotent).
func (c *Controller) Stop() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.st = stored{}
	c.cur = Status{}
	return c.save()
}

// Halt ends the order by itself (no safe zone for this level) and keeps the reason on the screen.
func (c *Controller) Halt(reason string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.st.On {
		return
	}
	c.st = stored{Ended: reason}
	c.cur = Status{}
	_ = c.save()
}

// Report is the bot's per-pass status; the first one after Start fixes the gold the earned count starts from.
func (c *Controller) Report(s Status) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.st.On {
		return
	}
	c.cur = s
	if !c.st.StartSet {
		c.st.Start, c.st.StartSet = s.Have, true
		_ = c.save()
	}
}

// View is the status for the screen.
func (c *Controller) View() View {
	c.mu.Lock()
	defer c.mu.Unlock()
	v := View{Schema: 1, On: c.st.On, Since: c.st.Since, For: c.st.For}
	if !c.st.On {
		v.Note = c.st.Ended
		v.Text = "Not farming gold."
		if c.st.Ended != "" {
			v.Text = "Farming gold stopped: " + c.st.Ended
		}
		return v
	}
	s := c.cur
	lead := "Farming gold on your order"
	if c.st.For == "repair" {
		lead = "Farming for a repair"
	}
	v.Area, v.Have, v.Want, v.Note = s.Area, s.Have, s.Want, s.Note
	if c.st.StartSet {
		v.Earned = max(s.Have-c.st.Start, 0)
	}
	v.Reached = s.Want > 0 && s.Have >= s.Want
	switch {
	case s.Note != "":
		v.Text = lead + ": " + s.Note
	case s.Area == "":
		v.Text = lead + ": choosing a zone."
	case v.Reached:
		v.Text = fmt.Sprintf(lead+": %d of %d reached, still farming in %s until you stop.", s.Have, s.Want, s.Area)
	default:
		v.Text = fmt.Sprintf(lead+": %d of %d in %s.", s.Have, s.Want, s.Area)
	}
	return v
}
