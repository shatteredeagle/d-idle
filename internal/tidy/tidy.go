// Package tidy is the pure policy of the automatic pack and stash tidying (docs/tidy.md): a table of item classes to
// actions. It reads no game, clicks nothing and knows no class by name in code; per-class and per-build rows live in
// policy.json. The town executes what it decides (internal/town/tidy.go).
package tidy

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Action is what tidying does with an item.
type Action string

const (
	Keep   Action = "keep"   // stays in the pack
	Stash  Action = "stash"  // pack -> stash at a town visit
	Sell   Action = "sell"   // sold at the vendor (the existing sell chore runs it)
	Drop   Action = "drop"   // thrown on the ground, only when the pack is full
	Ignore Action = "ignore" // tidy does nothing (marks, equipment, unknown)
)

// Facts is everything the policy looks at for one item. The town fills it from the snapshot; tests build it by hand.
type Facts struct {
	Code       string // base item code
	Type       string // item type code (rune, rin, scha, ...)
	Quality    string // low, normal, superior, magic, set, rare, unique, crafted
	Place      string // pack, stash, belt, equipment, cube, ...; only "pack" is acted on
	Identified bool
	Ethereal   bool
	Sockets    int
	Stackable  bool
	Quest      bool
	StartItem  bool
	SellValue  int    // estimated sell price (a lower bound for identified magic and better)
	Mark       string // the player's mark: "keep", "sell" or ""
}

// Ctx is the situation: hero class (for the per-class rows) and how full the grids are.
type Ctx struct {
	Class        string
	PackFullPct  int
	StashFullPct int
}

// Decision is the answer for one item. Why reads "Will stash: rune" and is shown in the inventory tooltip.
type Decision struct {
	Action Action
	Rule   string
	Why    string
}

// Match holds the conditions of a row; a zero field is not checked. Pointers distinguish "false" from "unset".
type Match struct {
	Types        []string `json:"types,omitempty"`
	TypePrefixes []string `json:"typePrefixes,omitempty"`
	Codes        []string `json:"codes,omitempty"`
	Qualities    []string `json:"qualities,omitempty"`
	Quest        *bool    `json:"quest,omitempty"`
	Stackable    *bool    `json:"stackable,omitempty"`
	Identified   *bool    `json:"identified,omitempty"`
	MinSockets   int      `json:"minSockets,omitempty"`
	MinSell      int      `json:"minSell,omitempty"`     // sell value at least this
	BelowSell    int      `json:"belowSell,omitempty"`   // sell value under this
	MinPackFull  int      `json:"minPackFull,omitempty"` // pack at least this percent full
}

// Row is one rule; rows are tried in order, the first match wins.
type Row struct {
	Name   string `json:"name"`
	Match  Match  `json:"match"`
	Action Action `json:"action"`
	Reason string `json:"reason"`
}

// Policy is the loaded table. Overrides maps a hero class to rows tried before the common ones.
type Policy struct {
	Comment   string           `json:"comment,omitempty"`
	Rows      []Row            `json:"rows"`
	Overrides map[string][]Row `json:"overrides,omitempty"`
	// StashMaxPct: no stash action once the stash is this full (the existing relieve-the-stash logic owns that case).
	StashMaxPct int `json:"stashMaxPct"`
}

//go:embed policy.json
var defaultJSON []byte

// Default returns the embedded policy.
func Default() (*Policy, error) { return Parse(defaultJSON) }

// Parse reads and validates a policy.
func Parse(b []byte) (*Policy, error) {
	var p Policy
	if err := json.Unmarshal(b, &p); err != nil {
		return nil, fmt.Errorf("tidy: policy: %w", err)
	}
	check := func(where string, rows []Row) error {
		for i, r := range rows {
			switch r.Action {
			case Keep, Stash, Sell, Drop, Ignore:
			default:
				return fmt.Errorf("tidy: %s row %d (%s): unknown action %q", where, i, r.Name, r.Action)
			}
			if r.Name == "" || r.Reason == "" {
				return fmt.Errorf("tidy: %s row %d needs a name and a reason", where, i)
			}
		}
		return nil
	}
	if len(p.Rows) == 0 {
		return nil, fmt.Errorf("tidy: policy has no rows")
	}
	if err := check("common", p.Rows); err != nil {
		return nil, err
	}
	for c, rows := range p.Overrides {
		if err := check("override "+c, rows); err != nil {
			return nil, err
		}
	}
	return &p, nil
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func (m Match) ok(f Facts, c Ctx) bool {
	if len(m.Types) > 0 && !has(m.Types, f.Type) {
		return false
	}
	if len(m.TypePrefixes) > 0 {
		hit := false
		for _, p := range m.TypePrefixes {
			hit = hit || strings.HasPrefix(f.Type, p)
		}
		if !hit {
			return false
		}
	}
	if len(m.Codes) > 0 && !has(m.Codes, f.Code) {
		return false
	}
	if len(m.Qualities) > 0 && !has(m.Qualities, f.Quality) {
		return false
	}
	if m.Quest != nil && *m.Quest != f.Quest {
		return false
	}
	if m.Stackable != nil && *m.Stackable != f.Stackable {
		return false
	}
	if m.Identified != nil && *m.Identified != f.Identified {
		return false
	}
	return f.Sockets >= m.MinSockets && f.SellValue >= m.MinSell &&
		(m.BelowSell == 0 || f.SellValue < m.BelowSell) && c.PackFullPct >= m.MinPackFull
}

// Decide answers for one item. Order of precedence, none of it data: only the pack is acted on (equipment, belt, cube
// and the stash are never touched), the player's marks win, the start kit and quest items stay; then the class rows,
// then the common rows.
func (p *Policy) Decide(f Facts, c Ctx) Decision {
	switch {
	case f.Place != "pack":
		return Decision{Ignore, "place", "Tidy leaves it: not in the backpack"}
	case f.Mark == "keep":
		return Decision{Ignore, "mark", "Kept: you marked it keep"}
	case f.Mark == "sell":
		return Decision{Ignore, "mark", "Will sell: you marked it for sale"}
	case f.StartItem:
		return Decision{Keep, "start", "Kept: starting gear"}
	case f.Quest:
		return Decision{Keep, "quest", "Kept: quest item"}
	}
	for _, rows := range [][]Row{p.Overrides[c.Class], p.Rows} {
		for _, r := range rows {
			if !r.Match.ok(f, c) {
				continue
			}
			a := r.Action
			if a == Stash && p.StashMaxPct > 0 && c.StashFullPct >= p.StashMaxPct {
				return Decision{Keep, r.Name, "Kept: the stash is full (" + r.Reason + ")"}
			}
			return Decision{a, r.Name, why(a, r, f)}
		}
	}
	return Decision{Ignore, "default", "Tidy leaves it: no rule"}
}

func why(a Action, r Row, f Facts) string {
	t := r.Reason
	if a == Sell || a == Drop {
		t = fmt.Sprintf("%s, value %d", r.Reason, f.SellValue)
	}
	return map[Action]string{Keep: "Kept", Stash: "Will stash", Sell: "Will sell", Drop: "Will drop", Ignore: "Tidy leaves it"}[a] + ": " + t
}

// Item pairs an id with its facts for Plan.
type Item struct {
	ID    uint32
	Facts Facts
}

// Entry is one planned action.
type Entry struct {
	ID uint32
	Decision
}

// Plan decides every item and returns the stash and drop actions, stash first, best value first, at most max in total
// (the safety valve; max <= 0 means none). Sell and keep are counted by the caller from Decide.
func (p *Policy) Plan(items []Item, c Ctx, max int) []Entry {
	var st, dr []Entry
	for _, it := range items {
		d := p.Decide(it.Facts, c)
		switch d.Action {
		case Stash:
			st = append(st, Entry{it.ID, d})
		case Drop:
			dr = append(dr, Entry{it.ID, d})
		}
	}
	val := map[uint32]int{}
	for _, it := range items {
		val[it.ID] = it.Facts.SellValue
	}
	sort.SliceStable(st, func(i, j int) bool { return val[st[i].ID] > val[st[j].ID] })
	sort.SliceStable(dr, func(i, j int) bool { return val[dr[i].ID] < val[dr[j].ID] })
	out := append(st, dr...)
	if max < 0 {
		max = 0
	}
	if len(out) > max {
		out = out[:max]
	}
	return out
}
