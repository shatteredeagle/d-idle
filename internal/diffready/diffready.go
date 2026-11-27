// Package diffready decides whether a hero is READY for the next difficulty (design
// docs/design/difficulty-transition-2026-11-27.md section 4). Ready is information: the hop itself is opt-in (owner
// go, user decision 2026-11-27); nothing here starts a game. Rows are data (difficulty_ready.json); Go knows check
// kinds, never a class or a skill name. Hell has no resist term (user decision 2026-11-27).
package diffready

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"d2lod-bot/internal/d2"
)

//go:embed difficulty_ready.json
var rowsJSON []byte

type kit struct {
	Healing     int `json:"healing"`
	ManaOrRejuv int `json:"mana_or_rejuv"`
	Portal      int `json:"portal"`
}

type common struct {
	BaalDead          bool           `json:"baal_dead"`
	MercAlive         bool           `json:"merc_alive"`
	DeathsLastHourMax int            `json:"deaths_last_hour_max"`
	ResPostMin        map[string]int `json:"res_post_min"` // absent for Hell: no resist gate
	Kit               kit            `json:"kit"`
}

type classRow struct {
	ClvlMin     int `json:"clvl_min"`
	MaxLifeMin  int `json:"max_life_min"`
	KeySkillMin int `json:"key_skill_min"`
}

type target struct {
	Tag     string              `json:"tag"`
	All     common              `json:"all"`
	Classes map[string]classRow `json:"classes"`
}

// Table is the parsed data file.
type Table struct{ Targets map[string]target }

var (
	once   sync.Once
	shared *Table
	shErr  error
)

// Shipped parses the embedded file once.
func Shipped() (*Table, error) {
	once.Do(func() { shared, shErr = Parse(rowsJSON) })
	return shared, shErr
}

// Parse reads a rows file; unknown fields are an error so a typo cannot silently drop a term.
func Parse(b []byte) (*Table, error) {
	var raw struct {
		Note    string            `json:"_note"`
		Targets map[string]target `json:"targets"`
	}
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("diffready: %w", err)
	}
	return &Table{Targets: raw.Targets}, nil
}

// Hero is what the predicate reads. Res is the raw (pre-penalty) stat as leveler.RawResists gives it.
type Hero struct {
	Class          d2.Class
	Level          int
	MaxLife        int
	Res, MaxRes    d2.Resists
	Baal           Tri // Baal dead in the current difficulty (quest word, U flag position: quest.BaalDead)
	KeySkillLevel  int // -1 when not read
	Healing        int // healing potions in belt and pack; -1 when not read
	ManaOrRejuv    int // -1 when not read
	Portals        int // -1 when not read
	Merc           Tri
	DeathsLastHour int
}

// Tri is a fact that may be unread; the zero value is Unread, so a half-filled Hero never passes by accident.
type Tri int8

const (
	Unread Tri = iota
	No
	Yes
)

// Verdict is the answer. Missing lists what blocks (each with the numbers); Unknown lists terms that could not be
// read and therefore did not block (shown to the owner, never a silent pass).
type Verdict struct {
	Target  string
	Ready   bool
	Missing []string
	Unknown []string
	Tag     string // S for Nightmare rows, U for Hell rows
}

// Why is one line for ops status --why and the Pact/Return screen.
func (v Verdict) Why() string {
	switch {
	case v.Ready && len(v.Unknown) == 0:
		return "ready for " + v.Target + " (" + v.Tag + "): waiting for your go"
	case v.Ready:
		return "ready for " + v.Target + " (" + v.Tag + "), unread: " + strings.Join(v.Unknown, ", ") + ": waiting for your go"
	}
	return "not ready for " + v.Target + ": " + strings.Join(v.Missing, "; ")
}

// Check evaluates hero h for target ("nightmare" or "hell"). An unknown target or class is an error naming it.
func (t *Table) Check(targetName string, h Hero) (Verdict, error) {
	tg, ok := t.Targets[targetName]
	if !ok {
		return Verdict{}, fmt.Errorf("diffready: no readiness rows for difficulty %q (have %s)", targetName, strings.Join(t.names(), ", "))
	}
	cls := strings.ToLower(h.Class.String())
	row, ok := tg.Classes[cls]
	if !ok {
		return Verdict{}, fmt.Errorf("diffready: no %s readiness row for class %q", targetName, h.Class.String())
	}
	v := Verdict{Target: targetName, Tag: tg.Tag}
	miss := func(f string, a ...any) { v.Missing = append(v.Missing, fmt.Sprintf(f, a...)) }
	unk := func(s string) { v.Unknown = append(v.Unknown, s) }
	a := tg.All
	switch {
	case !a.BaalDead:
	case h.Baal == Unread:
		unk("baal state")
	case h.Baal == No:
		miss("Baal not dead in the current difficulty")
	}
	if h.Level < row.ClvlMin {
		miss("clvl %d, need %d", h.Level, row.ClvlMin)
	}
	if h.MaxLife < row.MaxLifeMin {
		miss("max life %d, need %d", h.MaxLife, row.MaxLifeMin)
	}
	if row.KeySkillMin > 0 {
		switch {
		case h.KeySkillLevel < 0:
			unk("key skill level")
		case h.KeySkillLevel < row.KeySkillMin:
			miss("key skill level %d, need %d", h.KeySkillLevel, row.KeySkillMin)
		}
	}
	if len(a.ResPostMin) > 0 {
		d, _ := difficultyOf(targetName)
		vals := map[string]int{
			"fire":      d2.ResistTable.Effective(h.Res.Fire, h.MaxRes.Fire, d),
			"cold":      d2.ResistTable.Effective(h.Res.Cold, h.MaxRes.Cold, d),
			"lightning": d2.ResistTable.Effective(h.Res.Lightning, h.MaxRes.Lightning, d),
			"poison":    d2.ResistTable.Effective(h.Res.Poison, h.MaxRes.Poison, d),
		}
		for _, k := range sortedKeys(a.ResPostMin) {
			got, known := vals[k]
			if !known {
				return Verdict{}, fmt.Errorf("diffready: resist %q in the %s row is not an element", k, targetName)
			}
			if got < a.ResPostMin[k] {
				miss("%s resist %d after the %s penalty, need %d", k, got, targetName, a.ResPostMin[k])
			}
		}
	}
	if a.MercAlive {
		switch h.Merc {
		case Unread:
			unk("merc")
		case No:
			miss("no living mercenary")
		}
	}
	count := func(name string, have, need int) {
		switch {
		case need == 0:
		case have < 0:
			unk(name)
		case have < need:
			miss("%s %d, need %d", name, have, need)
		}
	}
	count("healing potions", h.Healing, a.Kit.Healing)
	count("mana or rejuvenation potions", h.ManaOrRejuv, a.Kit.ManaOrRejuv)
	count("town portals", h.Portals, a.Kit.Portal)
	if h.DeathsLastHour > a.DeathsLastHourMax {
		miss("%d deaths in the last hour, at most %d", h.DeathsLastHour, a.DeathsLastHourMax)
	}
	v.Ready = len(v.Missing) == 0
	return v, nil
}

func (t *Table) names() []string {
	var n []string
	for k := range t.Targets {
		n = append(n, k)
	}
	sort.Strings(n)
	return n
}

func sortedKeys(m map[string]int) []string {
	var k []string
	for x := range m {
		k = append(k, x)
	}
	sort.Strings(k)
	return k
}

func difficultyOf(name string) (d2.Difficulty, bool) {
	switch name {
	case "nightmare":
		return d2.Nightmare, true
	case "hell":
		return d2.Hell, true
	}
	return d2.Normal, false
}
