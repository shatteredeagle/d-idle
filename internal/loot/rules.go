package loot

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"path"
	"strings"
)

// The pickit is data: an ordered list of rules, first match wins. File format (JSON):
//
//	{ "default": Verdict-ish, "rules": [ {"name", "match": Match, "action", "tier", "prio", "needId", "reason"} ] }
//
// action: keep (carry in the pack), stash (carry home and store), sell (sell to a vendor), drop (leave it / dump it).
// tier:   S always wanted, A wanted when there is room, B only to sell, C ignore. The strictness knob filters by tier.
// prio:   pickup order, higher first (gold/runes/gems/uniques sit at 70-95, junk at 0-10).
// needId: the item matched but its stats cannot be judged before identification; town identifies it and asks again.
//
// Match fields are ANDed; an empty field matches anything:
//
//	codes      base item codes with path.Match globs ("r??", "gz?")
//	types      item type codes, ancestors count ("weap" matches swords) -- evaluated through data.DB.IsA
//	notTypes   types that must NOT match
//	qualities  low|normal|superior|magic|set|rare|unique|crafted
//	quest      base item is a quest item
//	identified true/false
//	ethereal   true/false
//	clvlMin/clvlMax  player level range
//	ilvlMin/ilvlMax  item level range
//	stats      all must hold; anyStats at least one must hold. {stat, min, max, shift}: value>>shift in [min,max].
//	           Stat conditions need the item's stat list; a rule with stat conditions does not match an item whose
//	           stats are unreadable (the reader failed or the item is unidentified and the stat is hidden).
//	needed     "potion"|"scroll"|"tome"|"key"|"gold"|"gem"|"runeword": only matches while the looter's needs say we want more of it.

// Match is the condition part of a Rule.
type Match struct {
	Codes      []string   `json:"codes,omitempty"`
	Types      []string   `json:"types,omitempty"`
	NotTypes   []string   `json:"notTypes,omitempty"`
	Qualities  []string   `json:"qualities,omitempty"`
	Quest      bool       `json:"quest,omitempty"`
	Identified *bool      `json:"identified,omitempty"`
	Ethereal   *bool      `json:"ethereal,omitempty"`
	ClvlMin    int        `json:"clvlMin,omitempty"`
	ClvlMax    int        `json:"clvlMax,omitempty"`
	ILvlMin    int        `json:"ilvlMin,omitempty"`
	ILvlMax    int        `json:"ilvlMax,omitempty"`
	Stats      []StatCond `json:"stats,omitempty"`
	AnyStats   []StatCond `json:"anyStats,omitempty"`
	Needed     string     `json:"needed,omitempty"`
}

// StatCond compares a (shifted) item stat. Max 0 means no upper bound.
type StatCond struct {
	Stat  int `json:"stat"`
	Min   int `json:"min"`
	Max   int `json:"max,omitempty"`
	Shift int `json:"shift,omitempty"`
}

// Rule is one pickit line.
type Rule struct {
	Name   string `json:"name"`
	Match  Match  `json:"match"`
	Action string `json:"action"`
	Tier   string `json:"tier"`
	Prio   int    `json:"prio"`
	NeedID bool   `json:"needId,omitempty"`
	Reason string `json:"reason"`
}

// Pickit is a parsed rule set.
type Pickit struct {
	Comment string `json:"comment,omitempty"`
	Version int    `json:"version"`
	Default Rule   `json:"default"`
	Rules   []Rule `json:"rules"`
}

//go:embed pickit_starter.json
var starterJSON []byte

// Starter returns the built-in starter pickit.
func Starter() *Pickit {
	p, err := ParsePickit(strings.NewReader(string(starterJSON)))
	if err != nil {
		panic("loot: bad embedded pickit: " + err.Error())
	}
	return p
}

// ParsePickit reads and validates a pickit file.
func ParsePickit(r io.Reader) (*Pickit, error) {
	var p Pickit
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("pickit: %w", err)
	}
	if p.Default.Action == "" {
		p.Default = Rule{Name: "default", Action: "drop", Tier: "C", Reason: "no rule matches"}
	}
	check := func(r Rule) error {
		if _, ok := parseAction(r.Action); !ok {
			return fmt.Errorf("pickit rule %q: bad action %q", r.Name, r.Action)
		}
		if tierRank(r.Tier) < 0 {
			return fmt.Errorf("pickit rule %q: bad tier %q", r.Name, r.Tier)
		}
		for _, q := range r.Match.Qualities {
			if qualityNo(q) == 0 {
				return fmt.Errorf("pickit rule %q: bad quality %q", r.Name, q)
			}
		}
		for _, c := range r.Match.Codes {
			if _, err := path.Match(c, "x"); err != nil {
				return fmt.Errorf("pickit rule %q: bad code pattern %q", r.Name, c)
			}
		}
		switch r.Match.Needed {
		case "", "potion", "scroll", "tome", "key", "gold", "gem", "runeword":
		default:
			return fmt.Errorf("pickit rule %q: bad needed %q", r.Name, r.Match.Needed)
		}
		return nil
	}
	if err := check(p.Default); err != nil {
		return nil, err
	}
	for _, r := range p.Rules {
		if err := check(r); err != nil {
			return nil, err
		}
	}
	return &p, nil
}

// Action is what to do with an item.
type Action int

const (
	Drop  Action = iota // leave it on the ground / throw it out of the pack
	Sell                // sell to a vendor
	Keep                // carry it in the pack
	Stash               // store it in the stash
)

func (a Action) String() string { return [...]string{"drop", "sell", "keep", "stash"}[a] }

func parseAction(s string) (Action, bool) {
	switch strings.ToLower(s) {
	case "keep":
		return Keep, true
	case "stash":
		return Stash, true
	case "sell":
		return Sell, true
	case "drop":
		return Drop, true
	}
	return Drop, false
}

// tierRank orders tiers: S=3 ... C=0, -1 invalid.
func tierRank(s string) int {
	switch strings.ToUpper(s) {
	case "S":
		return 3
	case "A":
		return 2
	case "B":
		return 1
	case "C":
		return 0
	}
	return -1
}

// qualityNo maps a quality name to d2 quality numbers (1 low .. 8 crafted); 0 for unknown.
func qualityNo(s string) int {
	switch strings.ToLower(s) {
	case "low":
		return 1
	case "normal":
		return 2
	case "superior":
		return 3
	case "magic":
		return 4
	case "set":
		return 5
	case "rare":
		return 6
	case "unique":
		return 7
	case "crafted":
		return 8
	}
	return 0
}

// QualityName is the inverse of qualityNo for logs.
func QualityName(q int) string {
	names := [...]string{"?", "low", "normal", "superior", "magic", "set", "rare", "unique", "crafted"}
	if q < 0 || q >= len(names) {
		return "?"
	}
	return names[q]
}
