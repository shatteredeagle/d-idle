// Package currency is the metacurrency registry and the account ledger (docs/currency.md,
// docs/research/metacurrency-design.md). A currency is a data row; the ledger is append-only and hash-chained.
package currency

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

const (
	ScopePerCharacter = "per_character"
	ScopeAccount      = "account"
	StorageItem       = "item"
	StorageLedger     = "ledger"

	RegistrySchema = "currency.registry/1"
)

// Converts says what banking turns a currency into: n units become n*Ratio of Currency, through the recipe Via.
type Converts struct {
	Currency string `json:"currency"`
	Ratio    int64  `json:"ratio"`
	Via      string `json:"via"`
}

// Item is the in-game form of an item-stored currency. Code is the overlay Misc code (empty until the overlay row exists).
type Item struct {
	Code     string `json:"code"`
	ArtDonor string `json:"art_donor,omitempty"`
}

// Display says where the currency shows on the watch screen.
type Display struct {
	Icon  string   `json:"icon"`
	Label string   `json:"label"` // after the amount in the header strip: "in pack", "banked"
	Order int      `json:"order"`
	Where []string `json:"where"`
}

// Currency is one registry row.
type Currency struct {
	ID        string    `json:"id"`
	Scope     string    `json:"scope"`
	Name      string    `json:"name"`
	Blurb     string    `json:"blurb"`
	Storage   string    `json:"storage"`
	Item      *Item     `json:"item,omitempty"`
	Stackable bool      `json:"stackable"`
	Cap       int64     `json:"cap"` // stack cap of an item currency; 0 for a ledger currency
	Sources   []string  `json:"sources"`
	Sinks     []string  `json:"sinks"`
	Converts  *Converts `json:"converts_to,omitempty"`
	Display   Display   `json:"display"`
}

// Source is a way to earn a currency. DailyCap is the soft cap of credits per UTC day (0 = none): the part above it is
// booked but sealed, not spendable.
type Source struct {
	ID       string `json:"id"`
	Currency string `json:"currency"`
	Name     string `json:"name"`
	DailyCap int64  `json:"daily_cap"`
}

// Upgrade is one row of the spend sheet: a sink priced per level. Costs[i] buys level i+1.
type Upgrade struct {
	ID       string  `json:"id"`
	Sink     string  `json:"sink"`
	Currency string  `json:"currency"`
	Tier     string  `json:"tier"`
	Name     string  `json:"name"`
	Effect   string  `json:"effect"`
	Costs    []int64 `json:"costs"`
	Wired    bool    `json:"wired"` // false: shown on the spend sheet but cannot be bought yet
}

// Registry is the validated currency table.
type Registry struct {
	Schema     string     `json:"schema"`
	Currencies []Currency `json:"currencies"`
	Sources    []Source   `json:"sources"`
	Upgrades   []Upgrade  `json:"upgrades"`
}

//go:embed registry.json
var defaultJSON []byte

// Default is the registry shipped with the bot.
func Default() *Registry {
	r, err := Parse(defaultJSON)
	if err != nil {
		panic("currency: embedded registry.json is invalid: " + err.Error())
	}
	return r
}

// Load reads and validates a registry file.
func Load(path string) (*Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Parse(b)
}

// Parse decodes (unknown fields are an error) and validates a registry.
func Parse(b []byte) (*Registry, error) {
	var r Registry
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&r); err != nil {
		return nil, fmt.Errorf("currency registry: %w", err)
	}
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return &r, nil
}

// Validate checks the table; every message names the row. No silent defaults.
func (r *Registry) Validate() error {
	if r.Schema != RegistrySchema {
		return fmt.Errorf("currency registry: schema %q, want %q", r.Schema, RegistrySchema)
	}
	if len(r.Currencies) == 0 {
		return errors.New("currency registry: no currencies")
	}
	cur := map[string]*Currency{}
	for i := range r.Currencies {
		c := &r.Currencies[i]
		if c.ID == "" {
			return fmt.Errorf("currency registry: currency #%d has no id", i)
		}
		if cur[c.ID] != nil {
			return fmt.Errorf("currency registry: duplicate currency id %q", c.ID)
		}
		cur[c.ID] = c
		if c.Scope != ScopePerCharacter && c.Scope != ScopeAccount {
			return fmt.Errorf("currency %q: unknown scope %q (want %s or %s)", c.ID, c.Scope, ScopePerCharacter, ScopeAccount)
		}
		if c.Name == "" {
			return fmt.Errorf("currency %q: no name", c.ID)
		}
		switch c.Storage {
		case StorageItem:
			if c.Item == nil {
				return fmt.Errorf("currency %q: storage item needs an item block", c.ID)
			}
			if c.Scope != ScopePerCharacter {
				return fmt.Errorf("currency %q: an item currency is per_character", c.ID)
			}
		case StorageLedger:
			if c.Item != nil {
				return fmt.Errorf("currency %q: a ledger currency has no item block", c.ID)
			}
			if c.Scope != ScopeAccount {
				return fmt.Errorf("currency %q: a ledger currency is account scoped", c.ID)
			}
		default:
			return fmt.Errorf("currency %q: unknown storage %q (want %s or %s)", c.ID, c.Storage, StorageItem, StorageLedger)
		}
		if c.Stackable && c.Cap <= 0 {
			return fmt.Errorf("currency %q: a stackable currency needs a stack cap > 0", c.ID)
		}
		if !c.Stackable && c.Cap != 0 {
			return fmt.Errorf("currency %q: cap %d on a non-stackable currency", c.ID, c.Cap)
		}
	}
	src := map[string]*Source{}
	for i := range r.Sources {
		s := &r.Sources[i]
		if s.ID == "" || src[s.ID] != nil {
			return fmt.Errorf("currency registry: source #%d: empty or duplicate id %q", i, s.ID)
		}
		src[s.ID] = s
		if cur[s.Currency] == nil {
			return fmt.Errorf("source %q: unknown currency %q", s.ID, s.Currency)
		}
		if s.DailyCap < 0 {
			return fmt.Errorf("source %q: negative daily_cap", s.ID)
		}
	}
	for _, c := range r.Currencies {
		for _, id := range c.Sources {
			if src[id] == nil || src[id].Currency != c.ID {
				return fmt.Errorf("currency %q: source %q is unknown or belongs to another currency", c.ID, id)
			}
		}
		if cv := c.Converts; cv != nil {
			if cur[cv.Currency] == nil {
				return fmt.Errorf("currency %q: converts_to unknown currency %q", c.ID, cv.Currency)
			}
			if cv.Ratio <= 0 || cv.Via == "" {
				return fmt.Errorf("currency %q: converts_to needs ratio > 0 and a via recipe", c.ID)
			}
		}
	}
	for _, c := range r.Currencies { // converts_to cycle: follow the chain at most len steps
		at, n := c.ID, 0
		for cur[at].Converts != nil {
			at = cur[at].Converts.Currency
			if n++; n > len(cur) {
				return fmt.Errorf("currency %q: converts_to cycle", c.ID)
			}
		}
	}
	up := map[string]bool{}
	for _, u := range r.Upgrades {
		if u.ID == "" || up[u.ID] {
			return fmt.Errorf("currency registry: upgrade with empty or duplicate id %q", u.ID)
		}
		up[u.ID] = true
		c := cur[u.Currency]
		if c == nil {
			return fmt.Errorf("upgrade %q: unknown currency %q", u.ID, u.Currency)
		}
		if c.Storage != StorageLedger {
			return fmt.Errorf("upgrade %q: is paid in %q, which is not a ledger currency", u.ID, u.Currency)
		}
		if !contains(c.Sinks, u.Sink) {
			return fmt.Errorf("upgrade %q: sink %q is not a sink of currency %q", u.ID, u.Sink, u.Currency)
		}
		if u.Name == "" || len(u.Costs) == 0 {
			return fmt.Errorf("upgrade %q: needs a name and at least one cost", u.ID)
		}
		for _, k := range u.Costs {
			if k <= 0 {
				return fmt.Errorf("upgrade %q: cost %d is not positive", u.ID, k)
			}
		}
	}
	return nil
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

// Get returns the currency row or an error naming the id.
func (r *Registry) Get(id string) (*Currency, error) {
	for i := range r.Currencies {
		if r.Currencies[i].ID == id {
			return &r.Currencies[i], nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownCurrency, id)
}

// Source returns the source row.
func (r *Registry) Source(id string) (*Source, bool) {
	for i := range r.Sources {
		if r.Sources[i].ID == id {
			return &r.Sources[i], true
		}
	}
	return nil, false
}

// Upgrade returns the upgrade row.
func (r *Registry) Upgrade(id string) (*Upgrade, bool) {
	for i := range r.Upgrades {
		if r.Upgrades[i].ID == id {
			return &r.Upgrades[i], true
		}
	}
	return nil, false
}

// ByRecipe finds the currency whose converts_to goes through the recipe id.
func (r *Registry) ByRecipe(recipe string) (*Currency, error) {
	for i := range r.Currencies {
		if c := &r.Currencies[i]; c.Converts != nil && c.Converts.Via == recipe {
			return c, nil
		}
	}
	return nil, fmt.Errorf("%w: %q", ErrUnknownRecipe, recipe)
}

// Ordered returns the currencies in display order.
func (r *Registry) Ordered() []Currency {
	out := append([]Currency(nil), r.Currencies...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].Display.Order < out[j].Display.Order })
	return out
}
