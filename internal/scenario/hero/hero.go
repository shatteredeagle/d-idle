// Package hero is the Proving Ground hero library (design docs/design/fable-review-2026-11-21/proving-ground-gate.md
// section 6.2): it turns a short declaration (profile, class, build, level, weapon) into the setup verbs of a prepared
// hero, the player commands that spend its points the way its build table does, and the expectations the self-check
// compares with the game after the prep. Builds are data: the points come from the leveler's plan table, the main attack
// from the build file or the class's combat profile; an unknown class or build is an error, never a Sorceress default.
//
// A prepared hero is CHEATED by the test-character provenance rule: its level, experience and points are setup writes.
package hero

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/plandata"
	"d2lod-bot/internal/points"
	"d2lod-bot/internal/power"
	"d2lod-bot/internal/telemetry"
)

// Profiles.
const (
	// Played: stat and skill points spent as the build's table spends them at that level, the main attack bound.
	Played = "played"
	// Banked: every point unspent, for scenarios about spending points.
	Banked = "banked"
)

// Decl is the declaration of a hero.
type Decl struct {
	Profile string `json:"profile"`
	Class   string `json:"class"` // class name, any case (paladin, Sorceress)
	Build   string `json:"build,omitempty"`
	Level   int    `json:"level"`
	// Weapon is the item code worn in the right hand; "" takes the class's starting weapon from the tables, "none" wears none.
	Weapon string `json:"weapon,omitempty"`
	// QuestSkillPts are the skill points of quests the hero has done (the Den of Evil gives 1, data quest-points.json). The
	// build tables fold them into their rows, so a hero without them reaches a row's skill later than the table says.
	QuestSkillPts int `json:"quest_skill_pts,omitempty"`
}

// Tables answers what the library needs from the game's tables (skills.txt, charstats.txt). DBTables reads a data.DB.
type Tables interface {
	// SkillID is the skills.txt id of a skill name.
	SkillID(name string) (int, bool)
	// SkillIndex is the index `cmd skillpt` takes for a skill id of the class (its place among the class's skills).
	SkillIndex(class string, id int) (int, bool)
	// Rule is the game's learn rule for the class (required level, prerequisites); nil allows everything.
	Rule(class string) leveler.Rule
	// StarterWeapon is the class's starting right-hand weapon code, "" when it has none.
	StarterWeapon(class string) string
}

// Verb is one line sent to the prep d2d with the reason written to the ledger first.
type Verb struct {
	Line   string `json:"line"`
	Reason string `json:"reason"`
	// Write is true for a setup write (a lab `debug` verb); false for a player command (`cmd ...`), which spends points
	// through the game's own path like the skill tree and the character panel do.
	Write bool `json:"write"`
}

// Hero is a resolved declaration.
type Hero struct {
	Decl
	Build string `json:"build"`
	Verbs []Verb `json:"verbs"`
	// StatPts and SkillPts are the unspent points the hero must have after the verbs (the table may bank a point when
	// no planned skill is learnable yet).
	StatPts  int `json:"stat_pts"`
	SkillPts int `json:"skill_pts"`
	// Skills are the hard points the plan buys, by skill name.
	Skills map[string]int `json:"skills,omitempty"`
	// Attack is the build's main attack at the level; AttackID its skills.txt id (-1 unresolved) and AttackLeft its hand.
	Attack     string `json:"attack"`
	AttackID   int    `json:"attack_id"`
	AttackLeft bool   `json:"attack_left"`
	// AttackIndex is the attack's place in the dump's skills.base_levels (-1: the plain Attack, nothing to learn).
	AttackIndex int      `json:"attack_index"`
	Weapon      string   `json:"weapon,omitempty"`
	Tags        []string `json:"tags"`
}

// Resolve builds the hero of d. t may be nil for the banked profile only (no skill to resolve).
func Resolve(d Decl, t Tables) (Hero, error) {
	class := strings.ToLower(strings.TrimSpace(d.Class))
	if d.Level < 2 || d.Level > 99 {
		return Hero{}, fmt.Errorf("hero: level %d out of range 2..99", d.Level)
	}
	if d.Profile == "" {
		d.Profile = Played
	}
	if d.Profile != Played && d.Profile != Banked {
		return Hero{}, fmt.Errorf("hero: profile %q (want %s or %s)", d.Profile, Played, Banked)
	}
	if d.Profile == Played && t == nil {
		return Hero{}, fmt.Errorf("hero: the played profile needs the game tables")
	}
	build := d.Build
	if build == "" {
		build = leveler.SelectBuild(class, 1, nil) // the table a fresh hero of the class gets
	}
	if build == "" {
		return Hero{}, fmt.Errorf("hero: no build table for class %q", d.Class)
	}
	b, err := leveler.LoadBuild(build, "")
	if err != nil {
		return Hero{}, fmt.Errorf("hero: %w", err)
	}
	if b.Class != class {
		return Hero{}, fmt.Errorf("hero: build %s is a %s table, the hero is a %q", build, b.Class, d.Class)
	}
	h := Hero{Decl: d, Build: build, AttackID: -1, AttackIndex: -1}
	h.Class = class
	reason := func(what string) string {
		return fmt.Sprintf("SETUP: hero %s %s clvl %d (%s): %s", d.Profile, class, d.Level, build, what)
	}
	exp, ok := telemetry.DefaultExpForLevel(d.Level)
	if !ok {
		return Hero{}, fmt.Errorf("hero: no experience for level %d", d.Level)
	}
	if d.QuestSkillPts < 0 || d.QuestSkillPts > 12 {
		return Hero{}, fmt.Errorf("hero: quest skill points %d out of range 0..12", d.QuestSkillPts)
	}
	statPts, skillPts := 5*(d.Level-1), d.Level-1+d.QuestSkillPts
	write := func(what, line string) {
		h.Verbs = append(h.Verbs, Verb{Line: line, Reason: reason(what), Write: true})
	}
	write("level", fmt.Sprintf("debug stat 12 %d", d.Level))
	write("experience of that level", fmt.Sprintf("debug stat 13 %d", exp))
	write("stat points the levels give", fmt.Sprintf("debug statpts %d", statPts))
	write("skill points the levels and the declared quests give", fmt.Sprintf("debug skillpts %d", skillPts))
	tags := map[string]bool{"level-write": true}
	if d.QuestSkillPts > 0 {
		tags["quest-points"] = true
	}

	switch d.Weapon {
	case "none":
	case "":
		if t != nil {
			h.Weapon = t.StarterWeapon(class)
		}
	default:
		h.Weapon = d.Weapon
	}
	if h.Weapon != "" {
		write("weapon in the right hand", fmt.Sprintf("debug equip %s %d", h.Weapon, WeaponSlot))
		tags["worn"] = true
	}

	h.StatPts, h.SkillPts = statPts, skillPts
	if d.Profile == Played {
		if err := h.spend(b, t, statPts, skillPts, reason); err != nil {
			return Hero{}, err
		}
	}
	h.Tags = append([]string{"CHEATED", d.Profile}, sortedKeys(tags)...)
	return h, nil
}

// WeaponSlot is the right-hand body slot of `debug equip` (V tools/ab/prep: the weapon reads back in slot 4).
const WeaponSlot = 4

func (h *Hero) spend(b *leveler.Build, t Tables, statPts, skillPts int, reason func(string) string) error {
	plan := b.Plan(leveler.Have{Level: h.Level, StatPts: statPts, SkillPts: skillPts, Skills: map[string]int{}}, t.Rule(h.Class), nil)
	cmd := func(what, line string) { h.Verbs = append(h.Verbs, Verb{Line: line, Reason: reason(what)}) }
	var attr [4]int
	h.Skills = map[string]int{}
	for _, a := range plan {
		if a.Kind == leveler.KindStat {
			attr[a.Attr]++
		} else {
			h.Skills[a.Skill]++
		}
	}
	for i, n := range attr { // the leveler's attribute order is the sheet's (points.Stats)
		if n > 0 {
			cmd("stat points by the table", fmt.Sprintf("cmd stat %d %d", points.Stats[i].ID, n))
			h.StatPts -= n
		}
	}
	for _, a := range plan { // table order: a prerequisite is bought before the skill that needs it
		if a.Kind != leveler.KindSkill {
			continue
		}
		id, ok := t.SkillID(a.Skill)
		if !ok {
			return fmt.Errorf("hero: build %s: skill %q is not in the tables", h.Build, a.Skill)
		}
		idx, ok := t.SkillIndex(h.Class, id)
		if !ok {
			return fmt.Errorf("hero: build %s: skill %q (id %d) is not a %s skill", h.Build, a.Skill, id, h.Class)
		}
		cmd("skill point by the table: "+a.Skill, fmt.Sprintf("cmd skillpt %d", idx))
		h.SkillPts--
	}

	spec, err := power.SpecFromBuild(plandata.FS, "build-"+h.Build+".json", h.Level, classAttacks)
	if err != nil {
		return fmt.Errorf("hero: main attack of %s at clvl %d: %w", h.Build, h.Level, err)
	}
	h.Attack = spec.Attack
	if h.Attack == "" {
		return fmt.Errorf("hero: build %s has no main attack at clvl %d", h.Build, h.Level)
	}
	if h.Attack != power.PlainAttack && h.Skills[h.Attack] == 0 {
		return fmt.Errorf("hero: build %s at clvl %d: main attack %s gets no point from the plan (banked: %d skill points)", h.Build, h.Level, h.Attack, h.SkillPts)
	}
	id, ok := t.SkillID(h.Attack)
	if !ok {
		return fmt.Errorf("hero: main attack %q is not in the tables", h.Attack)
	}
	h.AttackID = id
	if h.Attack != power.PlainAttack {
		if h.AttackIndex, ok = t.SkillIndex(h.Class, id); !ok {
			return fmt.Errorf("hero: main attack %q is not a %s skill", h.Attack, h.Class)
		}
	}
	h.AttackLeft = true
	for _, k := range b.Hotkeys {
		if k.Skill == h.Attack {
			h.AttackLeft = k.Hand == "left"
		}
	}
	hand := map[bool]string{true: "left", false: "right"}[h.AttackLeft]
	cmd("bind the main attack "+h.Attack+" on the "+hand+" hand", fmt.Sprintf("cmd select %d %d", id, map[bool]int{true: 1, false: 0}[h.AttackLeft]))
	return nil
}

// classAttacks adapts the combat profiles to the archetype resolver (the same adapter cmd/d2bot uses).
func classAttacks(class string) ([]power.Band, bool, error) {
	var pick *combat.Profile
	for _, p := range combat.Profiles() {
		if strings.EqualFold(p.Class, class) && (pick == nil || (pick.Build != "" && p.Build == "")) {
			pf := p
			pick = &pf
		}
	}
	if pick == nil {
		return nil, false, fmt.Errorf("no combat profile for class %q", class)
	}
	bs := make([]power.Band, 0, len(pick.Bands))
	for _, b := range pick.Bands {
		bs = append(bs, power.Band{MinLevel: b.MinLevel, Requires: b.Requires, Attack: b.Attack})
	}
	return bs, pick.Melee != nil, nil
}

// Provenance is the one-line record for docs/probes/test-characters.md.
func (h Hero) Provenance(name string) string {
	spent := "points banked"
	if h.Profile == Played {
		spent = fmt.Sprintf("points spent by the table (%d stat, %d skill left), %s bound", h.StatPts, h.SkillPts, h.Attack)
	}
	w := "no weapon"
	if h.Weapon != "" {
		w = "weapon " + h.Weapon
	}
	return fmt.Sprintf("%s | %s | hero library %s, build %s, clvl %d by setup writes; %s; %s | %s", name, h.Class, h.Profile, h.Build, h.Level, spent, w, strings.Join(h.Tags, " "))
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
