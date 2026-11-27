package points

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/skilltree"
)

// Skill verdict codes (the `code` of a refused POST).
const (
	CodeUnknownSkill = "unknown_skill"
	CodeOtherClass   = "other_class"
	CodeRule         = "rule"    // the game's own rule: points, max level, required level, prerequisites
	CodeConfirm      = "confirm" // an attack skill the hero cannot use yet: one confirmation
)

// Tier labels of the interim capability test (the generated capability table of the design review replaces it).
const (
	TierSupported   = "supported"   // the fighter has a rule for it today (internal/combat Playable) or it is a passive
	TierUnsupported = "unsupported" // legal to buy, the hero will not use it
)

// SkillVerdict is the answer to "may one more hard point go into this skill, and what will it mean".
type SkillVerdict struct {
	OK     bool   `json:"ok"`
	Code   string `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
	Skill  string `json:"skill,omitempty"`
	Tier   string `json:"tier,omitempty"`
	Attack bool   `json:"attack,omitempty"`
	// NeedsConfirm is true when the only thing in the way is the confirmation for an unsupported attack skill.
	NeedsConfirm bool `json:"needsConfirm,omitempty"`
}

// Tables is the part of the game tables the desk reads (data.DB's slices; a test builds it by hand).
type Tables struct {
	Skills  []*data.Skill
	Classes []*data.CharClass
}

// SkillByID finds a skill by its skills.txt id.
func (t Tables) SkillByID(id int) *data.Skill {
	for _, s := range t.Skills {
		if s.ID == id {
			return s
		}
	}
	return nil
}

// Class finds a class by name, case-insensitive.
func (t Tables) Class(name string) *data.CharClass {
	for _, c := range t.Classes {
		if strings.EqualFold(c.Name, name) {
			return c
		}
	}
	return nil
}

// Legal carries what CheckSkill needs besides the hero. Every function is data-driven: no skill name, class or
// level literal lives here (AGENTS.md build flexibility).
type Legal struct {
	T Tables
	// Playable says whether the fighter has a rule for the skill (combat.Playable); nil counts everything as playable.
	Playable func(skillName string) bool
	// Attack says whether the skill is a main-attack candidate (power's skill kinds: melee, missile, spell, summon).
	Attack func(skillName string) bool
}

// CheckSkill judges one more hard point into skill id for a hero. confirm is the second tap for an unsupported attack
// skill. The hero's class code gates the skill (a Paladin never gets a Sorceress skill); the rest is skilltree.CanAdd,
// which mirrors d2d's can_learn (V: components/rules/rules.hpp can_learn: max level, clvl >= reqlevel, every prerequisite
// learned; the same text is the decompiled game.exe icon check FUN_004ac200) plus the optional per-level cap.
func (l Legal) CheckSkill(h skilltree.Hero, id int, confirm bool) SkillVerdict {
	if len(l.T.Skills) == 0 {
		return SkillVerdict{Code: CodeUnknownSkill, Reason: "the game tables are not loaded"}
	}
	s := l.T.SkillByID(id)
	if s == nil || s.Page == 0 {
		return SkillVerdict{Code: CodeUnknownSkill, Reason: fmt.Sprintf("no skill with id %d on a skill page", id)}
	}
	name := s.Name
	if name == "" {
		name = s.Key
	}
	v := SkillVerdict{Skill: name}
	if h.ClassCode != "" && !strings.EqualFold(s.Class, h.ClassCode) {
		v.Code, v.Reason = CodeOtherClass, name+" belongs to another class"
		return v
	}
	byKey := make(map[string]*data.Skill, len(l.T.Skills))
	for _, k := range l.T.Skills {
		byKey[strings.ToLower(k.Key)] = k
	}
	if ok, why := skilltree.CanAdd(s, h, byKey); !ok {
		v.Code, v.Reason = CodeRule, why
		return v
	}
	v.Tier = TierSupported
	if !s.Passive && l.Playable != nil && !l.Playable(name) {
		v.Tier = TierUnsupported
	}
	v.Attack = !s.Passive && !s.Aura && l.Attack != nil && l.Attack(name)
	if v.Tier == TierUnsupported && v.Attack && !confirm {
		v.Code, v.NeedsConfirm = CodeConfirm, true
		v.Reason = name + " is an attack your hero cannot use yet: it will not cast it. Confirm to put the point there anyway"
		return v
	}
	v.OK = true
	return v
}

// SkillCap is the most hard points a skill may hold at character level clvl when it needs reqLevel: clvl - reqLevel + 1
// (kolbot AutoSkill.js:54-62 enforces it; d2d and game.exe do not, so only the table checker uses it).
func SkillCap(clvl, reqLevel int) int { return max(clvl-reqLevel+1, 0) }

// ClassSkillIndex is the index d2d's `cmd skillpt` takes: the position of the skill among its class's skills in
// ascending skills.txt id order (V: fork components/game/world/commands.cpp passes the verb's number to can_learn as
// skill_index 0..29 of the class; the order is the same one internal/d2dsrc uses to read base_levels, U for the
// ascending order). ok is false when the skill is not one of the class's.
func ClassSkillIndex(t Tables, classCode string, id int) (int, bool) {
	var ids []int
	for _, s := range t.Skills {
		if strings.EqualFold(s.Class, classCode) && s.Page > 0 {
			ids = append(ids, s.ID)
		}
	}
	sort.Ints(ids)
	for i, x := range ids {
		if x == id {
			return i, true
		}
	}
	return 0, false
}

// TableViolations lists the rows of a per-level plan table whose cumulative skill points break the per-level cap
// (kolbot rule), for the table checker. rows is clvl -> skill name -> points added at that level; reqLevel looks a skill's
// required level up. It names the first break of each skill.
type TableRow struct {
	Clvl   int
	Skills map[string]int
}

// TableViolations is pure: it replays the rows in order.
func TableViolations(rows []TableRow, reqLevel func(skill string) (int, bool)) []string {
	have := map[string]int{}
	seen := map[string]bool{}
	var out []string
	for _, r := range rows {
		names := make([]string, 0, len(r.Skills))
		for n := range r.Skills {
			names = append(names, n)
		}
		sort.Strings(names)
		for _, n := range names {
			have[n] += r.Skills[n]
			req, ok := reqLevel(n)
			if !ok || seen[n] {
				continue
			}
			if c := SkillCap(r.Clvl, req); have[n] > c {
				seen[n] = true
				out = append(out, fmt.Sprintf("clvl %d: %s has %d points, the cap is %d (needs level %d)", r.Clvl, n, have[n], c, req))
			}
		}
	}
	return out
}
