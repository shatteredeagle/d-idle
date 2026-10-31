// Package gear is the class gear and ammo model: which weapon a skill needs, whether the equipped weapon can attack
// at all (bows need a matching quiver with arrows, thrown weapons need a stack), when ammo is low, what to buy, and the
// Necromancer and Amazon build tables. It is pure data plus small functions; callers read the equipped items from memory.
//
// The tables are extracted from the user's own game files by d2tables (tables.go, not committed; V: Patch_D2.mpq over d2exp over d2data; skills.txt
// itypea/etypea, itemtypes.txt, weapons/misc/armor.txt, charstats.txt). Facts that are not in those tables are tagged
// V (live, with the source doc) or U (unverified) next to the code that relies on them. docs/gear.md has the integration notes.
package gear

import (
	"embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed builds/*.json
var buildFS embed.FS

type typeRow struct {
	Equiv     []string `json:"equiv"`
	Shoots    string   `json:"shoots"`
	Quiver    string   `json:"quiver"`
	Throwable bool     `json:"throwable"`
	Class     string   `json:"class"`
	Body      []string `json:"body"`
}

// Base is one item base (weapons, ammo and shields only).
type Base struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	TwoHanded bool   `json:"two_handed"`
	ReqStr    int    `json:"req_str"`
	ReqDex    int    `json:"req_dex"`
	LevelReq  int    `json:"level_req"`
	Stackable bool   `json:"stackable"`
	MinStack  int    `json:"min_stack"`
	MaxStack  int    `json:"max_stack"`
	Cost      int    `json:"cost"`
	PermStore bool   `json:"perm_store"` // misc.txt PermStoreItem: a vendor that sells the misc page always stocks it
}

type skillRow struct {
	Class    string   `json:"class"`
	ReqLevel int      `json:"req_level"`
	Req      []string `json:"req"`
	ItypeA   []string `json:"itypea"`
	EtypeA   []string `json:"etypea"`
	Left     bool     `json:"left"`
	Passive  bool     `json:"passive"`
	ID       int      `json:"id"`
}

// ClassInfo is the charstats.txt start state of a class.
type ClassInfo struct {
	Name  string `json:"name"`
	Str   int    `json:"str"`
	Dex   int    `json:"dex"`
	Vit   int    `json:"vit"`
	Ene   int    `json:"ene"`
	Start string `json:"start_skill"`
	Items []struct {
		Code  string `json:"code"`
		Loc   string `json:"loc"`
		Count int    `json:"count"`
	} `json:"items"`
}

type table struct {
	Types   map[string]typeRow   `json:"types"`
	Bases   map[string]Base      `json:"bases"`
	Skills  map[string]skillRow  `json:"skills"`
	Classes map[string]ClassInfo `json:"classes"`
}

// ClassCode maps "Amazon", "ama", "amazon" to the three-letter code used by skills.txt and charstats.
func ClassCode(class string) string {
	c := strings.ToLower(strings.TrimSpace(class))
	if len(c) > 3 {
		c = c[:3]
	}
	return c
}

// BaseByCode returns the base for a 3 or 4 character item code (weapons, ammo, shields only).
func BaseByCode(code string) (Base, bool) {
	b, ok := load().Bases[code]
	return b, ok
}

// TypeOf returns the itemtypes code of a base ("hbw" -> "bow"), or "" when the base is not in the table.
func TypeOf(code string) string { return load().Bases[code].Type }

// IsA walks the itemtypes Equiv1/Equiv2 graph ("abow" is a "bow" is a "miss").
func IsA(typ, ancestor string) bool {
	t := load()
	seen := map[string]bool{}
	todo := []string{typ}
	for len(todo) > 0 {
		x := todo[len(todo)-1]
		todo = todo[:len(todo)-1]
		if x == "" || seen[x] {
			continue
		}
		if x == ancestor {
			return true
		}
		seen[x] = true
		todo = append(todo, t.Types[x].Equiv...)
	}
	return false
}

// Equipped is what the bot reads from the equipment slots. Codes are item base codes (the 3-4 letter code behind the
// unit's class id); quantities are stat 70 (U for Quantity as a stat read; V live that stat 70 holds the count,
// docs/classes/smoke-amazon.md finding 3).
type Equipped struct {
	Weapon     string // body slot 4 (right hand), "" when empty
	WeaponQty  int    // thrown weapons: stack size left
	Offhand    string // body slot 5 (left hand): shield or quiver, "" when empty
	OffhandQty int    // quiver: arrows/bolts left
}

// WeaponType is the itemtypes code of the main-hand item, "" when empty or unknown.
func (e Equipped) WeaponType() string { return TypeOf(e.Weapon) }

// Launcher reports whether the main hand holds a bow or crossbow.
func (e Equipped) Launcher() bool { return IsA(e.WeaponType(), "miss") }

// Thrower reports whether the main hand holds a thrown weapon stack (javelins, throwing knives/axes).
func (e Equipped) Thrower() bool {
	t := load().Types[e.WeaponType()]
	return e.Weapon != "" && t.Throwable && IsA(e.WeaponType(), "thro")
}

// builtinReq are skills that are not class skills (skills.txt charclass empty, so not in gear.json): V Throw needs a
// throwable item (itemtypes Throwable), U for the exact set the game accepts.
var builtinReq = map[string]skillRow{
	"Throw": {ItypeA: []string{"thro"}},
}

func skillReq(skill string) (skillRow, bool) {
	if r, ok := load().Skills[skill]; ok {
		return r, true
	}
	r, ok := builtinReq[skill]
	return r, ok
}

// SkillWeapons returns the item types the skill accepts (skills.txt itypea, V) and the excluded ones (etypea).
// Empty accept means any weapon or none.
func SkillWeapons(skill string) (accept, exclude []string) {
	r, _ := skillReq(skill)
	return r.ItypeA, r.EtypeA
}

// ammoProblem is the shared "can this weapon shoot right now" check; "" when fine or not an ammo weapon.
func ammoProblem(e Equipped) string {
	switch {
	case e.Launcher():
		want := load().Types[e.WeaponType()].Shoots
		if want == "" {
			want = launcherShoots(e.WeaponType())
		}
		if e.Offhand == "" {
			return "launcher without quiver"
		}
		qt := TypeOf(e.Offhand)
		if !IsA(qt, "bowq") && !IsA(qt, "xboq") {
			return "launcher with a non-quiver in the left hand"
		}
		if !IsA(qt, want) {
			return fmt.Sprintf("quiver %s does not fit %s", e.Offhand, e.Weapon)
		}
		if e.OffhandQty <= 0 {
			return "quiver empty"
		}
	case e.Thrower():
		if e.WeaponQty <= 0 {
			return "thrown weapon stack empty"
		}
	}
	return ""
}

// launcherShoots finds the quiver type through the Equiv chain ("abow" -> "bow" shoots bowq).
func launcherShoots(typ string) string {
	t := load()
	seen := map[string]bool{}
	todo := []string{typ}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		if s := t.Types[x].Shoots; s != "" {
			return s
		}
		todo = append(todo, t.Types[x].Equiv...)
	}
	return ""
}

// SkillUsable says whether skill can be cast with the equipped items, and if not why. Skills without a weapon
// requirement (and unknown skills) are usable. Launcher skills also need working ammo.
func SkillUsable(skill string, e Equipped) (bool, string) {
	r, ok := skillReq(skill)
	if !ok || len(r.ItypeA) == 0 {
		return true, ""
	}
	if shieldOnly(r.ItypeA) {
		// Smite (skills.txt itypea1 = shld, V gear.json): the item must be in the off hand, any main-hand weapon will do.
		if !IsA(TypeOf(e.Offhand), "shld") {
			return false, skill + " needs a shield in the off hand"
		}
		return true, ""
	}
	wt := e.WeaponType()
	if e.Weapon == "" || wt == "" {
		return false, skill + " needs a " + strings.Join(r.ItypeA, "/") + " weapon, none equipped"
	}
	match := false
	for _, a := range r.ItypeA {
		if IsA(wt, a) {
			match = true
		}
	}
	for _, x := range r.EtypeA {
		if IsA(wt, x) {
			match = false
		}
	}
	if !match {
		return false, fmt.Sprintf("%s needs %s, %s is %s", skill, strings.Join(r.ItypeA, "/"), e.Weapon, wt)
	}
	if p := ammoProblem(e); p != "" {
		return false, skill + ": " + p
	}
	return true, ""
}

func shieldOnly(a []string) bool {
	for _, x := range a {
		if x != "shld" {
			return false
		}
	}
	return len(a) > 0
}

// UsableSkills filters skills to the ones SkillUsable accepts, keeping order.
func UsableSkills(skills []string, e Equipped) []string {
	var out []string
	for _, s := range skills {
		if ok, _ := SkillUsable(s, e); ok {
			out = append(out, s)
		}
	}
	return out
}

// WeaponProblem returns why the character should not attack with what it holds, or "" when it can. Rules:
// an item restricted to another class (itemtypes Class, V); a launcher with no fitting quiver or no ammo; a thrown
// stack that is empty; an Amazon with nothing in the main hand (U: bare hands is a legal but useless Amazon attack,
// the bot should buy or equip a weapon instead). Sorceress, Necromancer and others may fight with any weapon or none.
func WeaponProblem(class string, e Equipped) string {
	cc := ClassCode(class)
	if cc == "ama" && e.Weapon == "" {
		return "amazon has no weapon equipped"
	}
	if e.Weapon != "" && TypeOf(e.Weapon) == "" {
		return "" // unknown base: do not refuse on a table gap
	}
	for _, code := range []string{e.Weapon, e.Offhand} {
		if code == "" {
			continue
		}
		if c := classOf(TypeOf(code)); c != "" && c != cc {
			return fmt.Sprintf("%s is restricted to class %s", code, c)
		}
	}
	return ammoProblem(e)
}

// classOf is the class restriction of a type or of its ancestors ("abow" -> "ama").
func classOf(typ string) string {
	t := load()
	seen := map[string]bool{}
	todo := []string{typ}
	for len(todo) > 0 {
		x := todo[0]
		todo = todo[1:]
		if x == "" || seen[x] {
			continue
		}
		seen[x] = true
		if c := t.Types[x].Class; c != "" {
			return c
		}
		todo = append(todo, t.Types[x].Equiv...)
	}
	return ""
}

// StartItems is the charstats.txt starting equipment (V). Codes are base codes, Loc is the body location name
// ("rarm", "larm", "" for the pack).
func StartItems(class string) []StartItem {
	var out []StartItem
	for _, it := range load().Classes[ClassCode(class)].Items {
		out = append(out, StartItem{it.Code, it.Loc, it.Count})
	}
	return out
}

// StartItem is one starting item.
type StartItem struct {
	Code  string
	Loc   string
	Count int
}

// ClassStart returns the start attributes and items; ok is false for an unknown class.
func ClassStart(class string) (ClassInfo, bool) {
	c, ok := load().Classes[ClassCode(class)]
	return c, ok
}

// Slots (item location ids, d2s body locations). V live: quiver is in slot 5 (docs/classes/smoke-amazon.md), V itemtypes
// BodyLoc1/2 = rarm/larm for bows, quivers and shields alike.
const (
	SlotRightHand = 4
	SlotLeftHand  = 5
)

// Builds lists the embedded build names.
func Builds() []string {
	ents, _ := buildFS.ReadDir("builds")
	var out []string
	for _, e := range ents {
		out = append(out, strings.TrimSuffix(e.Name(), ".json"))
	}
	sort.Strings(out)
	return out
}

// BuildJSON returns one build file (same schema as internal/plandata/build-paladin-normal30.json, extra field
// main_attack). leveler.ParseBuild reads it unchanged.
func BuildJSON(name string) ([]byte, error) {
	b, err := buildFS.ReadFile("builds/" + name + ".json")
	if err != nil {
		return nil, fmt.Errorf("gear: unknown build %q (have %v)", name, Builds())
	}
	return b, nil
}

// DefaultBuild is the build name for a class (Normal to clvl 30), "" when none.
func DefaultBuild(class string) string {
	switch ClassCode(class) {
	case "ama":
		return "amazon_bow_normal30"
	case "nec":
		return "necromancer_summoner_normal30"
	case "dru":
		return "druid_fire_normal30"
	case "ass":
		return "assassin_fireblast_normal30"
	case "bar":
		return "barbarian_melee_normal30"
	}
	return ""
}

// MainAttack is one main_attack range of a build.
type MainAttack struct {
	From   int    `json:"from_clvl"`
	Until  int    `json:"until_clvl"`
	Skill  string `json:"skill"`
	Weapon string `json:"weapon"` // "any", "bow", "jave"
	Ammo   string `json:"ammo"`   // "bowq", "self" (the thrown stack), ""
	Note   string `json:"note"`
}

// MainAttackAt returns the build's main attack for a character level.
func MainAttackAt(build string, clvl int) (MainAttack, error) {
	b, err := BuildJSON(build)
	if err != nil {
		return MainAttack{}, err
	}
	var f struct {
		Main []MainAttack `json:"main_attack"`
	}
	if err := json.Unmarshal(b, &f); err != nil {
		return MainAttack{}, err
	}
	for _, m := range f.Main {
		if clvl >= m.From && clvl <= m.Until {
			return m, nil
		}
	}
	return MainAttack{}, fmt.Errorf("gear: build %s has no main attack at clvl %d", build, clvl)
}
