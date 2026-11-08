// Package unlock is the shared unlock framework of the watch screen (docs/unlock-ladder.md, design note
// docs/research/unlockables.md): a table of entries, each opened once by a trigger, persisted per profile. The table is
// data (data/unlocks/ladder.json). The engine reads events that already exist (Codex firsts, level, quest, counts, tally
// spend) and never reads the game. Basics are not entries: nothing here may hide them.
package unlock

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	unlockdata "d2lod-bot/data/unlocks"
)

// Schema names the ladder document.
const Schema = "unlock.ladder/1"

// Trigger kinds.
const (
	CodexFirst  = "codex_first"   // args: sub (Codex kind: monster, area, ...), ids (Codex ids, empty: any)
	Quest       = "quest"         // args: quest (name)
	Level       = "level"         // args: min (character level)
	TallySpend  = "tally_spend"   // opened by Buy; the entry's price is paid from the tally
	RetireCount = "retire_count"  // args: min (heroes retired so far; the source reports the running count)
	KillsOfKind = "kills_of_kind" // args: kind (monster name, or * for any), min (running count reported by the source)
	Seen        = "seen"          // args: what (a viewer-side moment such as expedition_card)
	BossKill    = "boss_kill"     // args: boss (monster name), difficulty (normal|nightmare|hell; absent = any)
)

// Trigger is one way to open an entry. Args depend on Kind (see the constants).
type Trigger struct {
	Kind string         `json:"kind"`
	Args map[string]any `json:"args,omitempty"`
}

// Reveal is what the watcher is told when an entry opens. Teaser is what a locked entry shows in its place.
type Reveal struct {
	Title  string `json:"title"`
	Text   string `json:"text"`
	Teaser string `json:"teaser,omitempty"`
}

// Entry is one row of the ladder. Group says who consumes it ("senses": Tier is the senses tier; "ui": a view or
// control; "pacts": a Pact the catalogue names as "pact_<unlock_trigger>"). Alt triggers open it too (first one to fire wins). A tally_spend trigger needs Price > 0.
type Entry struct {
	ID      string    `json:"id"`
	Group   string    `json:"group"`
	Tier    int       `json:"tier,omitempty"`
	Trigger Trigger   `json:"trigger"`
	Alt     []Trigger `json:"alt,omitempty"`
	Reveal  Reveal    `json:"reveal"`
	Price   int64     `json:"price,omitempty"`
}

// Ladder is the loaded table.
type Ladder struct {
	Schema  string  `json:"schema"`
	Doc     string  `json:"doc,omitempty"`
	Entries []Entry `json:"entries"`
}

var (
	ErrUnknownTrigger = errors.New("unlock: unknown trigger kind")
	ErrUnknownEntry   = errors.New("unlock: unknown entry")
	ErrUnknownEvent   = errors.New("unlock: unknown event type")
	ErrNotBuyable     = errors.New("unlock: entry is not for sale")
	ErrAlreadyOpen    = errors.New("unlock: entry is already open")
)

var groups = []string{"senses", "ui", "pacts"}

var difficulties = []string{"normal", "nightmare", "hell"}

// Shipped loads the embedded ladder.
func Shipped() (*Ladder, error) { return Parse(unlockdata.JSON) }

// Parse decodes and validates a ladder. Every error names the entry and the value.
func Parse(b []byte) (*Ladder, error) {
	var l Ladder
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&l); err != nil {
		return nil, fmt.Errorf("unlock: ladder: %w", err)
	}
	if l.Schema != Schema {
		return nil, fmt.Errorf("unlock: ladder schema %q, want %q", l.Schema, Schema)
	}
	seen := map[string]bool{}
	tiers := map[int]string{}
	for _, e := range l.Entries {
		if e.ID == "" || strings.ContainsAny(e.ID, " /\\\"") {
			return nil, fmt.Errorf("unlock: entry id %q is empty or has a space, slash or quote", e.ID)
		}
		if seen[e.ID] {
			return nil, fmt.Errorf("unlock: duplicate entry %q", e.ID)
		}
		seen[e.ID] = true
		if !contains(groups, e.Group) {
			return nil, fmt.Errorf("unlock: entry %q: unknown group %q", e.ID, e.Group)
		}
		if e.Group == "senses" {
			if e.Tier < 1 {
				return nil, fmt.Errorf("unlock: entry %q: senses tier %d must be 1 or more", e.ID, e.Tier)
			}
			if o, dup := tiers[e.Tier]; dup {
				return nil, fmt.Errorf("unlock: entries %q and %q both own senses tier %d", o, e.ID, e.Tier)
			}
			tiers[e.Tier] = e.ID
		} else if e.Tier != 0 {
			return nil, fmt.Errorf("unlock: entry %q: tier is for the senses group only", e.ID)
		}
		if e.Reveal.Title == "" || e.Reveal.Text == "" {
			return nil, fmt.Errorf("unlock: entry %q: reveal needs a title and a text (nothing unlocks silently)", e.ID)
		}
		spend := false
		for _, t := range append([]Trigger{e.Trigger}, e.Alt...) {
			if err := t.validate(); err != nil {
				return nil, fmt.Errorf("unlock: entry %q: %w", e.ID, err)
			}
			spend = spend || t.Kind == TallySpend
		}
		if spend && e.Price <= 0 {
			return nil, fmt.Errorf("unlock: entry %q: a tally_spend trigger needs a price above 0", e.ID)
		}
		if !spend && e.Price != 0 {
			return nil, fmt.Errorf("unlock: entry %q: a price without a tally_spend trigger", e.ID)
		}
	}
	return &l, nil
}

func (t Trigger) validate() error {
	need := func(keys ...string) error {
		for _, k := range keys {
			if _, ok := t.Args[k]; !ok {
				return fmt.Errorf("trigger %s needs arg %q", t.Kind, k)
			}
		}
		return nil
	}
	switch t.Kind {
	case CodexFirst:
		return need("sub")
	case Quest:
		return need("quest")
	case Seen:
		return need("what")
	case Level, RetireCount:
		if err := need("min"); err != nil {
			return err
		}
	case KillsOfKind:
		if err := need("kind", "min"); err != nil {
			return err
		}
	case BossKill:
		if err := need("boss"); err != nil {
			return err
		}
		if t.str("boss") == "" {
			return errors.New("trigger boss_kill: arg boss is empty")
		}
		if d, ok := t.Args["difficulty"]; ok {
			if s, _ := d.(string); !contains(difficulties, s) {
				return fmt.Errorf("trigger boss_kill: difficulty %v is not one of %v", d, difficulties)
			}
		}
		return nil
	case TallySpend:
		return nil
	default:
		return fmt.Errorf("%w %q", ErrUnknownTrigger, t.Kind)
	}
	if n, ok := t.num("min"); !ok || n < 1 {
		return fmt.Errorf("trigger %s: arg min must be a number of 1 or more", t.Kind)
	}
	return nil
}

func (t Trigger) num(k string) (int, bool) {
	f, ok := t.Args[k].(float64)
	return int(f), ok
}

func (t Trigger) str(k string) string { s, _ := t.Args[k].(string); return s }

func (t Trigger) strs(k string) []string {
	var out []string
	if l, ok := t.Args[k].([]any); ok {
		for _, v := range l {
			if s, ok := v.(string); ok {
				out = append(out, s)
			}
		}
	}
	return out
}

// Entry returns the row with id, or an error naming it.
func (l *Ladder) Entry(id string) (Entry, error) {
	for _, e := range l.Entries {
		if e.ID == id {
			return e, nil
		}
	}
	return Entry{}, fmt.Errorf("%w %q", ErrUnknownEntry, id)
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}
