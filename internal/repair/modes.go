package repair

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed modes.json
var modesJSON []byte

// Mode is one row of modes.json. The behaviour fields are data: a mode repairs when its test holds at a town visit.
type Mode struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	Unlock    string `json:"unlock,omitempty"`    // ladder entry that opens the mode; empty: always available
	OnBreak   bool   `json:"onBreak,omitempty"`   // repair when any worn item is broken
	EverySecs int    `json:"everySecs,omitempty"` // repair when this long has passed since the last repair
	Always    bool   `json:"always,omitempty"`    // repair whenever anything is damaged
	BelowPct  int    `json:"belowPct,omitempty"`  // repair when the lowest worn item is under this durability percent
	Never     bool   `json:"never,omitempty"`     // never repairs, a request is refused (a Pact's promise)
	Hidden    bool   `json:"hidden,omitempty"`    // set by a Pact only: not offered on the sheet, needs no unlock
}

// Table is the parsed modes file.
type Table struct {
	Version int    `json:"version"`
	Doc     string `json:"doc"`
	WarnPct int    `json:"warnPct"`
	Default string `json:"default"`
	Modes   []Mode `json:"modes"`
}

// Mode returns the mode with this id; an unknown id is an error naming it.
func (t *Table) Mode(id string) (*Mode, error) {
	for i := range t.Modes {
		if t.Modes[i].ID == id {
			return &t.Modes[i], nil
		}
	}
	return nil, fmt.Errorf("repair: unknown mode %q", id)
}

// ParseTable checks a modes file: unique ids, a default mode that needs no unlock, a mode that does nothing is an error.
func ParseTable(b []byte) (*Table, error) {
	var t Table
	if err := json.Unmarshal(b, &t); err != nil {
		return nil, fmt.Errorf("repair: modes: %w", err)
	}
	if t.WarnPct <= 0 || t.WarnPct >= 100 {
		return nil, fmt.Errorf("repair: modes: warnPct %d is not in 1..99", t.WarnPct)
	}
	seen := map[string]bool{}
	for _, m := range t.Modes {
		switch {
		case m.ID == "" || m.Label == "":
			return nil, fmt.Errorf("repair: modes: a mode without id or label")
		case seen[m.ID]:
			return nil, fmt.Errorf("repair: modes: duplicate mode %q", m.ID)
		case m.ID != t.Default && !m.OnBreak && m.EverySecs <= 0 && !m.Always && m.BelowPct <= 0 && !m.Never:
			return nil, fmt.Errorf("repair: modes: mode %q has no test (onBreak, everySecs, always, belowPct or never)", m.ID)
		case m.ID == t.Default && (m.OnBreak || m.EverySecs > 0 || m.Always || m.Unlock != ""):
			return nil, fmt.Errorf("repair: modes: the default mode %q must be manual (no test, no unlock)", m.ID)
		case m.ID != t.Default && m.Unlock == "" && !m.Hidden:
			return nil, fmt.Errorf("repair: modes: mode %q needs an unlock row", m.ID)
		}
		seen[m.ID] = true
	}
	if !seen[t.Default] {
		return nil, fmt.Errorf("repair: modes: default mode %q is not in the table", t.Default)
	}
	return &t, nil
}

// Shipped is the embedded table.
func Shipped() (*Table, error) { return ParseTable(modesJSON) }
