package plandata

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Chore defaults per class (card classchores, ADR 0018 Amendment 2026-11-07): data, never class literals in Go.
// Every value comes from docs/research/class-early-game.md section 4 (veteran experience; EXP, UNVERIFIED).

// ChorePotions is the potion mix as a ratio (health to mana); mana 0 means no mana potions.
type ChorePotions struct {
	Health int    `json:"health"`
	Mana   int    `json:"mana"`
	Note   string `json:"note,omitempty"`
}

// ChoreRepair: cadence is "every_trip" or "rare".
type ChoreRepair struct {
	Cadence            string `json:"cadence"`
	BelowDurabilityPct int    `json:"below_durability_pct"`
	Note               string `json:"note,omitempty"`
}

// ChoreAmmo is set for the Amazon bow only.
type ChoreAmmo struct {
	Kind           string `json:"kind"`
	QuiverMin      int    `json:"quiver_min"`
	QuiverMax      int    `json:"quiver_max"`
	RestockBelowPc int    `json:"restock_below_pct"`
	Note           string `json:"note,omitempty"`
}

// ChoreTrip are the class-specific town-trip triggers; 0 means the class has no such trigger.
type ChoreTrip struct {
	HealthPotionsBelow int  `json:"health_potions_below,omitempty"`
	ManaPotionsBelow   int  `json:"mana_potions_below,omitempty"`
	DurabilityBelowPct int  `json:"durability_below_pct,omitempty"`
	AmmoBelowPct       int  `json:"ammo_below_pct,omitempty"`
	BeltBelowPct       int  `json:"belt_below_pct,omitempty"`
	PackFull           bool `json:"pack_full,omitempty"`
	TPTomeBelow        int  `json:"tp_tome_below,omitempty"`
}

// ClassChores is one class row.
type ClassChores struct {
	Potions  ChorePotions `json:"potions"`
	Repair   ChoreRepair  `json:"repair"`
	Ammo     *ChoreAmmo   `json:"ammo"`
	Summons  []string     `json:"summons"`
	Buffs    []string     `json:"buffs"`
	TownTrip ChoreTrip    `json:"town_trip"`
	// Start maps a policy sheet row id to the option the row starts on for this class. A row not listed starts on its
	// default. The sheet checks the ids against its table (internal/watch).
	Start map[string]string `json:"start"`
}

// ChoreTable is chores-defaults.json.
type ChoreTable struct {
	Version    int    `json:"version"`
	Source     string `json:"source"`
	AllClasses struct {
		TownTrip ChoreTrip `json:"town_trip"`
	} `json:"all_classes"`
	Classes map[string]ClassChores `json:"classes"`
}

// ParseChores decodes and checks a table; an unknown field, class or value is an error naming it.
func ParseChores(data []byte) (*ChoreTable, error) {
	var t ChoreTable
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("plandata: chores defaults: %w", err)
	}
	if t.Version != 1 || len(t.Classes) == 0 {
		return nil, fmt.Errorf("plandata: chores defaults: version %d with %d classes", t.Version, len(t.Classes))
	}
	for name, c := range t.Classes {
		switch c.Repair.Cadence {
		case "every_trip", "rare":
		default:
			return nil, fmt.Errorf("plandata: chores defaults: class %q: unknown repair cadence %q", name, c.Repair.Cadence)
		}
		if c.Potions.Health < 0 || c.Potions.Mana < 0 || c.Potions.Health+c.Potions.Mana == 0 {
			return nil, fmt.Errorf("plandata: chores defaults: class %q: bad potion mix %d:%d", name, c.Potions.Health, c.Potions.Mana)
		}
		if c.Ammo != nil && (c.Ammo.Kind == "" || c.Ammo.QuiverMin <= 0 || c.Ammo.QuiverMax < c.Ammo.QuiverMin) {
			return nil, fmt.Errorf("plandata: chores defaults: class %q: bad ammo row", name)
		}
	}
	return &t, nil
}

// Chores returns the class row from the embedded table. A class the table does not hold is an error naming it (no
// fallback to another class).
func Chores(class string) (ClassChores, error) {
	data, err := FS.ReadFile("chores-defaults.json")
	if err != nil {
		return ClassChores{}, err
	}
	t, err := ParseChores(data)
	if err != nil {
		return ClassChores{}, err
	}
	c, ok := t.Classes[class]
	if !ok {
		return ClassChores{}, fmt.Errorf("plandata: chores defaults: unknown class %q", class)
	}
	return c, nil
}
