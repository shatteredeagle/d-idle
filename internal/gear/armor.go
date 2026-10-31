package gear

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
)

// Armor is one armor.txt row (V: Patch_D2 over d2exp over d2data, extracted by d2tables): body armor, helms,
// circlets, shields, belts, boots, gloves and the class helms and shields. armor.txt has no reqdex column, so armor
// never needs Dex (V header). Semantics of Speed and StrBonus are U (not used by any decision yet).
type Armor struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Tier         int    `json:"tier"` // 0 normal, 1 exceptional, 2 elite (normcode/ubercode/ultracode columns)
	MinAC        int    `json:"min_ac"`
	MaxAC        int    `json:"max_ac"`
	ReqStr       int    `json:"req_str"`
	LevelReq     int    `json:"level_req"`
	Level        int    `json:"level"`
	Speed        int    `json:"speed"`
	Block        int    `json:"block"`
	Durability   int    `json:"durability"`
	NoDurability bool   `json:"no_durability"`
	MaxSockets   int    `json:"max_sockets"` // table cap; the item level limits it further (U)
	StrBonus     int    `json:"str_bonus"`
	DexBonus     int    `json:"dex_bonus"`
	InvWidth     int    `json:"inv_w"`
	InvHeight    int    `json:"inv_h"`
}

func armors() map[string]Armor {
	load()
	return armorTab
}

// ArmorByCode returns the armor.txt row of an item code.
func ArmorByCode(code string) (Armor, bool) {
	a, ok := armors()[code]
	return a, ok
}

// ArmorCodes lists every armor code, sorted.
func ArmorCodes() []string {
	out := make([]string, 0, len(armors()))
	for c := range armors() {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// Requirement returns the Str/Dex/level a base item (armor, weapon, shield, ammo) needs and whether the code is known.
func Requirement(code string) (str, dex, level int, ok bool) {
	if a, found := ArmorByCode(code); found {
		return a.ReqStr, 0, a.LevelReq, true
	}
	if b, found := BaseByCode(code); found {
		return b.ReqStr, b.ReqDex, b.LevelReq, true
	}
	return 0, 0, 0, false
}

// CanEquip reports whether a character with these stats may wear or wield the base, and why not. Unknown codes are
// refused: the caller must not guess a requirement.
func CanEquip(code string, str, dex, clvl int) (bool, string) {
	rs, rd, rl, ok := Requirement(code)
	switch {
	case !ok:
		return false, "unknown item " + code
	case str < rs:
		return false, fmt.Sprintf("needs %d strength", rs)
	case dex < rd:
		return false, fmt.Sprintf("needs %d dexterity", rd)
	case clvl < rl:
		return false, fmt.Sprintf("needs level %d", rl)
	}
	return true, ""
}

// Milestone is a planned piece of gear a build must be able to wear by a character level (milestones.json).
type Milestone struct {
	Clvl  int      `json:"clvl"`
	Items []string `json:"items"` // base codes, all worn together
	Note  string   `json:"note"`
}

//go:embed milestones.json
var milestonesJSON []byte

var (
	msOnce sync.Once
	msTab  map[string][]Milestone
)

// Milestones returns the gear milestones of an endgame build (nil when the build has none), ordered by clvl.
func Milestones(build string) []Milestone {
	msOnce.Do(func() {
		var f struct {
			Builds map[string][]Milestone `json:"builds"`
		}
		if err := json.Unmarshal(milestonesJSON, &f); err != nil {
			panic("gear: milestones.json: " + err.Error())
		}
		msTab = f.Builds
	})
	return msTab[build]
}

// StatNeeds is the Str and Dex required to wear every milestone item due by clvl (V: from the tables; the milestone
// lists themselves are U, see docs/gear.md).
func StatNeeds(build string, clvl int) (str, dex int) {
	for _, m := range Milestones(build) {
		if m.Clvl > clvl {
			continue
		}
		for _, c := range m.Items {
			s, d, _, _ := Requirement(c)
			str, dex = max(str, s), max(dex, d)
		}
	}
	return str, dex
}
