package combat

import (
	"math"
	"strings"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Build selects the skill preferences (docs/plan/leveling.md section 1).
type Build int

const (
	BuildBlizzard  Build = iota // primary: Charged Bolt, then Blizzard after the respec at clvl 26
	BuildLightning              // fallback: Charged Bolt, then Chain Lightning
)

// respecLevel is when the primary build switches its main attack (plan: respec at 26).
const respecLevel = 26

// SkillKind says how a skill is aimed.
type SkillKind int

const (
	KindBolt    SkillKind = iota // aimed at the monster
	KindGround                   // aimed at the ground under the monster
	KindSelf                     // centred on the caster (Static Field, Nova)
	KindBuff                     // cast on self, no damage
	KindMelee                    // melee skill or basic attack: the button is clicked (or held) on the monster
	KindAura                     // right-hand aura: no cost, stays active until another aura replaces it
	KindSummon                   // minions, cast on self
	KindCharged                  // charge-up skill released on the target
)

// skillMeta is what skills.txt does not give us: how the skill is aimed, with which hand and its reach in subtiles.
// The table is classdata/skills.json (classdata.go). Ranges are Kolbot's SkillData (R, docs/prior-art/04-combat.md
// section 4) and UNVERIFIED in 1.14d; the live runs exercised Fire Bolt and the Paladin's Zeal/Smite.
type skillMeta struct {
	Kind  SkillKind
	Range int
	Hand  Hand
	Hold  bool
}

// staticRange: Kolbot's floor((lvl+4)*2/3) (R).
func staticRange(level int) int { return (level + 4) * 2 / 3 }

// SkillSpec is a skill the character can use right now.
type SkillSpec struct {
	ID    int
	Name  string
	Level int // effective level (points plus item bonus)
	Cost  int // mana, rounded up
	Elem  data.DamageType
	Kind  SkillKind
	Range int
	Delay time.Duration // cooldown of timed skills (Blizzard), zero otherwise
	Key   string        // bound hotkey, "" when none

	Hand     Hand     // which button uses the skill (hotkeys only ever select the right one)
	Hold     bool     // hold the button on the target
	ReqLevel int      // character level required (skills.txt)
	Prereq   []string // skills that must be learned first (skills.txt reqskill1..3)
}

func elemOf(etype string) data.DamageType {
	switch strings.ToLower(etype) {
	case "fire":
		return data.Fire
	case "ltng":
		return data.Lightning
	case "cold":
		return data.Cold
	case "pois":
		return data.Poison
	case "mag":
		return data.Magic
	}
	return data.Physical
}

// makeSpec builds a SkillSpec for a skill name; ok is false for names neither the table nor a profile knows.
// Cost is the mana formula of skills.txt; auras cost nothing to switch (VERIFIED LIVE: Might, mana unchanged).
func makeSpec(cat Catalog, name string, level int) (SkillSpec, bool) {
	db := loadClassData()
	s := cat.SkillByName(name)
	if s == nil {
		return SkillSpec{}, false
	}
	meta, ok := db.skills[name]
	if !ok {
		if !db.names[name] {
			return SkillSpec{}, false
		}
		if meta, ok = deriveMeta(s); !ok {
			return SkillSpec{}, false
		}
	}
	sp := SkillSpec{ID: s.ID, Name: name, Level: level, Elem: elemOf(s.EType), Kind: meta.Kind, Range: meta.Range, Hand: meta.Hand, Hold: meta.Hold,
		ReqLevel: s.ReqLevel}
	for _, r := range s.ReqSkills {
		if r != "" {
			sp.Prereq = append(sp.Prereq, r)
		}
	}
	if meta.Kind != KindAura {
		sp.Cost = int(math.Ceil(s.ManaCost(level)))
	}
	if name == "Static Field" {
		sp.Range = staticRange(level)
	}
	if sp.Kind == KindMelee && sp.Range == 0 {
		sp.Range = defaultMeleeRange
	}
	if s.Delay > 0 {
		sp.Delay = time.Duration(s.Delay) * 40 * time.Millisecond // 25 frames per second
	}
	return sp, true
}

// attackOrder is the Sorceress preference list of attack skill names for a build and character level (the data
// lives in classdata/profiles.json; Fighter uses the profile of the character's class).
func attackOrder(b Build, clvl int, known func(string) bool) []string {
	return ProfileFor(d2.Sorceress, b).AttackNames(clvl, known)
}

// buffOrder: one armor (they replace each other) then the shield of the fallback build.
func buffOrder(b Build) [][]string { return ProfileFor(d2.Sorceress, b).Buffs }

// Layout is the fixed F-key assignment of the Sorceress per build: docs/plan/data/hotkeys.json, adjusted because the
// game has only F1-F8: Telekinesis is dropped, Teleport gets F2 (nav casts it), Charged Bolt and Fire Bolt get keys.
func Layout(b Build) map[string]string { return ProfileFor(d2.Sorceress, b).Layout }

// KnownSkills returns the character's usable skills (effective level at least 1) as specs, with Key filled from keys
// (skill id to key name).
func KnownSkills(p *d2.Player, cat Catalog, keys map[int]string) map[string]SkillSpec {
	out := map[string]SkillSpec{}
	for _, sk := range p.Skills {
		if sk.Effective() < 1 && sk.ID > 3 { // Attack, Kick, Throw, Unsummon are usable at level 0
			continue
		}
		def := cat.Skill(sk.ID)
		if def == nil {
			continue
		}
		sp, ok := makeSpec(cat, def.Key, max(sk.Effective(), 1))
		if !ok {
			continue
		}
		sp.Key = keys[sk.ID]
		out[sp.Name] = sp
	}
	return out
}

// PickReason explains a failed pick.
type PickReason int

const (
	PickOK       PickReason = iota
	PickNoSkill             // no attack skill known and bound
	PickImmune              // every skill is blocked by an immunity
	PickNeedMana            // a usable skill exists but costs more mana than we have
	PickCooling             // only timed skills that are still cooling down
)

// PickAttack chooses the first skill in preference order that is not immune-blocked, affordable and ready.
// A skill that is too expensive falls through to a cheaper one (low-mana fallback); a target immune to the main
// element falls through to the fallback element (plan: Static Field + merc is the plan for cold immunes and is
// handled by the fight loop, not here).
func PickAttack(order []SkillSpec, t *Target, mana int, ready func(SkillSpec) bool) (SkillSpec, PickReason) {
	if len(order) == 0 {
		return SkillSpec{}, PickNoSkill
	}
	needMana, cooling, anyOpen := false, false, false
	for _, sp := range order {
		if t != nil && t.Immune(sp.Elem) {
			continue
		}
		anyOpen = true
		if sp.Cost > mana {
			needMana = true
			continue
		}
		if ready != nil && !ready(sp) {
			cooling = true
			continue
		}
		return sp, PickOK
	}
	switch {
	case !anyOpen:
		return SkillSpec{}, PickImmune
	case needMana:
		return SkillSpec{}, PickNeedMana
	case cooling:
		return SkillSpec{}, PickCooling
	}
	return SkillSpec{}, PickNoSkill
}
