package combat

import (
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/resource"
)

// Per-monster and per-affix rules of docs/difficulty.md P1 items 9-13 and P2 items 16-22 (DIFFP2). Everything here is
// pure: it reads a Target and says what to do, the fight loop applies it. Numbers from the Arreat tables are V (see the
// doc), thresholds are R (reasoned, tune live), skill lists and monster behaviour tagged U need a live check.

// Plan is the positioning and skill advice for one target.
type Plan struct {
	Kite        bool   // may be kited (back off while shooting); false for Extra Fast and Teleporting units
	NoMelee     bool   // never trade blows with it (Mana Burn on a mana user in Nightmare/Hell)
	KeepRange   int    // stay at least this many subtiles away (0 = no constraint)
	ManaReserve int    // percent of max mana to keep unspent (Mana Burn: the pool is the target)
	KillFirst   bool   // fight before the rest of the pack
	Skip        bool   // do not engage at all
	Notes       string // why, for the log
}

// Spacing and reserve numbers (R: planning values, not measured). Lightning Enchanted releases 8 bolts when it is hit
// (V Arreat), so ranged characters should hit from further away than the bolt nova reaches; the Stygian Doll explosion
// radius is U.
const (
	lightningSpacing = 8
	dollSpacing      = 4 // docs/difficulty.md P2 item 17
	manaBurnSpacing  = 6
	manaBurnReserve  = 40
)

// monsterRule is a class-id keyed rule (monstats Id, V tables); effects are U unless noted.
type monsterRule struct {
	match func(key string) bool
	apply func(p *Plan)
	note  string
}

func hasPrefixAny(key string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(key, p) {
			return true
		}
	}
	return false
}

func oneOf(key string, keys ...string) bool {
	for _, k := range keys {
		if key == k {
			return true
		}
	}
	return false
}

var monsterRules = []monsterRule{
	{ // Oblivion Knight (and Lord De Seis, same class): U curse caster
		match: func(k string) bool { return k == "doomknight3" },
		apply: func(p *Plan) { p.KillFirst = true; p.KeepRange = max(p.KeepRange, 5) },
		note:  "oblivion knight: kill first, keep range (curse caster, U)",
	},
	{ // Stygian Doll line: fire explosion on death (U), fire immune in Hell (V tables)
		match: func(k string) bool {
			return oneOf(k, "fetish5", "fetish7", "fetishblow5", "fetishblow7", "fetishshaman5", "fetishshaman7")
		},
		apply: func(p *Plan) { p.KeepRange = max(p.KeepRange, dollSpacing) },
		note:  "stygian doll: do not stand next to it when it dies (fire explosion, U)",
	},
	{ // Finger Mage line (the community "Mind Flayer", the string does not exist here, V): lightning immune
		match: func(k string) bool { return strings.HasPrefix(k, "fingermage") },
		apply: func(p *Plan) { p.NoMelee = true; p.KeepRange = max(p.KeepRange, 5) },
		note:  "finger mage: lightning immune caster, do not melee (U)",
	},
	{ // Pit Lord row with all five resists 145 in Hell (V tables)
		match: func(k string) bool { return k == "megademon6" },
		apply: func(p *Plan) { p.Skip = true },
		note:  "megademon6: every resist 145, hard skip (V tables)",
	},
	{ // Horadrim Ancient / Unraveler: raises skeletons, magic immune in Hell
		match: func(k string) bool { return strings.HasPrefix(k, "unraveler") },
		apply: func(p *Plan) { p.KillFirst = true },
		note:  "unraveler: raises skeletons, kill first",
	},
}

// PlanForTarget combines the affix rules and the class rules for one target. ManaNeed of the class decides who a Mana Burn
// hurts; ranged says the character attacks from a distance (a melee build cannot keep range, it uses NoMelee to
// leave or skip instead).
func PlanForTarget(t Target, c d2.Class, diff d2.Difficulty, clvl int) Plan {
	p := Plan{Kite: true}
	var notes []string
	a := t.Affixes
	hard := diff != d2.Normal
	if a&(AffixExtraFast|AffixTeleportation) != 0 {
		// Extra Fast is +100 movement (V Arreat), teleporters close at once: kiting only wastes mana and life.
		p.Kite = false
		notes = append(notes, "no kiting (fast/teleport)")
	}
	if a&AffixManaBurn != 0 && resource.ManaNeed(c) > 0 {
		p.KillFirst = true
		p.ManaReserve = manaBurnReserve
		p.KeepRange = max(p.KeepRange, manaBurnSpacing)
		if hard {
			// melee Mana Burn is multiplied by 256 by a game bug (V Arreat bonus.shtml): one hit empties the pool
			p.NoMelee = true
		}
		notes = append(notes, "mana burn: ranged kill, keep mana")
	}
	if a&AffixCursed != 0 {
		p.KillFirst = true
		notes = append(notes, "cursed: amplify damage, kill first")
	}
	if a&AffixLightningEnchanted != 0 {
		// bolts fire when it is hit (V Arreat): hit it from outside the bolt reach, and not by melee below clvl 75 in Hell
		p.KeepRange = max(p.KeepRange, lightningSpacing)
		if diff == d2.Hell && clvl < avoidHellLevel {
			p.NoMelee = true
		}
		notes = append(notes, "lightning enchanted: spacing")
	}
	if a&AffixAura != 0 || a&AffixMultiShot != 0 {
		p.KillFirst = true
	}
	if t.Mon != nil {
		key := strings.ToLower(t.Mon.Key)
		for _, r := range monsterRules {
			if r.match(key) {
				r.apply(&p)
				notes = append(notes, r.note)
			}
		}
	}
	p.Notes = strings.Join(notes, "; ")
	return p
}

// expFactor is the share of damage an element still does against a resistance (100 = full, immune and above = 0,
// negative resist amplifies).
func expFactor(res int) float64 {
	if res >= 100 {
		return 0
	}
	return float64(100-res) / 100
}

// DamageFactor is the expected share of a skill's damage that gets through the target, including Stone Skin
// (+50% damage resist, V Arreat) for physical skills. Without resistance data (HasRes false) it is 1.
func DamageFactor(sp SkillSpec, t *Target) float64 {
	if t == nil {
		return 1
	}
	f := 1.0
	if t.HasRes {
		f = expFactor(t.Res[sp.Elem])
	}
	if sp.Elem == data.Physical && t.Affixes&AffixStoneSkin != 0 && (!t.HasRes || t.Res[data.Physical] < 50) {
		f *= 0.5 // the live stat list may already hold it (then Res[Physical] >= 50); U which
	}
	return f
}

// resistTolerance: a preferred skill stays chosen while it does at least this share of the best alternative's
// damage factor (R). The profile order encodes cost and range preferences, so a mildly worse element is fine, but
// Magic Resistant / Hell resists (60+ points) switch the element.
const resistTolerance = 0.6

// PickByResist chooses among the skills that PickAttack would allow, preferring the profile order but moving to a
// skill of a better-resisted element when the preferred one does under resistTolerance of the best. Same signature
// and reasons as PickAttack.
func PickByResist(order []SkillSpec, t *Target, mana int, ready func(SkillSpec) bool) (SkillSpec, PickReason) {
	first, why := PickAttack(order, t, mana, ready)
	if why != PickOK || t == nil {
		return first, why
	}
	best := DamageFactor(first, t)
	var alt SkillSpec
	altF := best
	for _, sp := range order {
		if sp.Kind == KindBuff || sp.Kind == KindAura || sp.Kind == KindSummon || sp.Cost > mana || t.Immune(sp.Elem) {
			continue
		}
		if ready != nil && !ready(sp) {
			continue
		}
		if f := DamageFactor(sp, t); f > altF {
			alt, altF = sp, f
		}
	}
	if altF > 0 && best < resistTolerance*altF {
		return alt, PickOK
	}
	return first, PickOK
}

// Workaround is what to do about an immunity (docs/difficulty.md section 3.1).
type Workaround struct {
	Skills []string // attack skills of the class that do not hit the blocked elements, best first
	Merc   bool     // leave the kill to the merc / pets
	Skip   bool     // nothing the class can do
}

// classOptions lists each class's attack skills by damage type (U: names from memory of skills.txt, the builds
// differ; PickByResist/PickAttack still require the skill to be known and bound).
var classOptions = map[d2.Class]map[data.DamageType][]string{
	d2.Sorceress: {
		data.Fire: {"Fire Ball", "Fire Bolt", "Meteor", "Hydra"}, data.Cold: {"Blizzard", "Glacial Spike", "Ice Blast", "Frost Nova"},
		data.Lightning: {"Chain Lightning", "Lightning", "Nova", "Charged Bolt"},
	},
	d2.Necromancer: {
		data.Magic: {"Bone Spear", "Bone Spirit", "Teeth"}, data.Poison: {"Poison Nova", "Poison Dagger"},
		data.Physical: {"Corpse Explosion"}, data.Fire: {"Corpse Explosion"},
	},
	d2.Paladin: {
		data.Physical: {"Zeal", "Smite", "Sacrifice"}, data.Fire: {"Holy Fire", "Fist of the Heavens"},
		data.Cold: {"Holy Freeze"}, data.Lightning: {"Holy Shock", "Charged Strike"}, data.Magic: {"Vengeance"},
	},
	d2.Barbarian: {
		data.Physical: {"Double Swing", "Frenzy", "Whirlwind"}, data.Magic: {"Berserk"}, data.Fire: {"Berserk"},
	},
	d2.Amazon: {
		data.Physical: {"Multiple Shot", "Strafe", "Fend"}, data.Lightning: {"Lightning Fury", "Charged Strike"},
		data.Fire: {"Exploding Arrow"}, data.Cold: {"Freezing Arrow"}, data.Poison: {"Poison Javelin", "Plague Javelin"},
	},
	d2.Druid: {
		data.Fire: {"Fissure", "Volcano", "Fire Claws"}, data.Cold: {"Arctic Blast", "Twister"},
		data.Lightning: {"Shock Wave"}, data.Physical: {"Maul", "Feral Rage"},
	},
	d2.Assassin: {
		data.Physical: {"Dragon Talon", "Tiger Strike"}, data.Fire: {"Dragon Flight", "Fire Blast"}, data.Lightning: {"Lightning Sentry", "Charged Bolt Sentry"},
		data.Cold: {"Dragon Claw"}, data.Poison: {"Venom", "Death Sentry"},
	},
}

// ImmuneWorkaround lists the class skills that still hurt a target with the given resistances. Every damage type
// the class knows, whose element the target is not immune to, contributes its skills (listed order of the damage
// types is by remaining damage factor). Merc is set when nothing is left but the merc may still do it (the usual
// plan for elemental builds, docs/difficulty.md section 3.1); Skip when the target is also a unique or pack member
// that none of it can hurt.
func ImmuneWorkaround(c d2.Class, t *Target) Workaround {
	var w Workaround
	opts := classOptions[c]
	if opts == nil || t == nil || !t.HasRes {
		return w
	}
	type cand struct {
		d data.DamageType
		f float64
	}
	var cs []cand
	for d := range opts {
		if f := expFactor(t.Res[d]); f > 0 {
			cs = append(cs, cand{d, f})
		}
	}
	// insertion sort, three to six entries; ties by damage type id for determinism
	for i := 1; i < len(cs); i++ {
		for j := i; j > 0 && (cs[j].f > cs[j-1].f || cs[j].f == cs[j-1].f && cs[j].d < cs[j-1].d); j-- {
			cs[j], cs[j-1] = cs[j-1], cs[j]
		}
	}
	seen := map[string]bool{}
	for _, c := range cs {
		for _, n := range opts[c.d] {
			if !seen[n] {
				seen[n] = true
				w.Skills = append(w.Skills, n)
			}
		}
	}
	if len(w.Skills) == 0 {
		w.Merc = true
		w.Skip = t.Elite || t.Boss
	}
	return w
}

// PlayerResists are the character's resistances after gear (fire, cold, lightning, poison), without the difficulty
// penalty (U: whether the stat list read from the game already includes it; check live, see docs/difficulty.md).
type PlayerResists struct{ Fire, Cold, Lightning, Poison int }

// ResistPenalty is the player's resist penalty for a difficulty: 0 / -40 / -100 (V tables difficultylevels.tsv).
func ResistPenalty(d d2.Difficulty) int { return d.ResistPenalty() }

// Effective applies the penalty. The game caps the maximum at 75 and lets the value fall as low as -100 (S).
func (r PlayerResists) Effective(d d2.Difficulty) PlayerResists {
	p := ResistPenalty(d)
	f := func(v int) int { return max(min(v, 75)+p, -100) }
	return PlayerResists{f(r.Fire), f(r.Cold), f(r.Lightning), f(r.Poison)}
}

// elementResist returns the effective resist against an elemental damage type.
func (r PlayerResists) of(e data.DamageType) (int, bool) {
	switch e {
	case data.Fire:
		return r.Fire, true
	case data.Cold:
		return r.Cold, true
	case data.Lightning:
		return r.Lightning, true
	case data.Poison:
		return r.Poison, true
	}
	return 0, false
}

// minSafeResist is the effective resist under which a monster of that element is rated one step more dangerous (R).
const minSafeResist = 0

// ResistDanger raises a pack danger by one (up to DangerDeadly) when a unit's elemental modifier hits an element
// in which the character's effective resist (after the difficulty penalty) is below zero (R threshold). Unknown
// affix elements (Spectral Hit) use the lowest of the three.
func ResistDanger(danger int, ts []Target, res PlayerResists, diff d2.Difficulty) int {
	eff := res.Effective(diff)
	low := min(eff.Fire, eff.Cold, eff.Lightning)
	for _, t := range ts {
		a := t.Affixes
		bad := a&AffixFireEnchanted != 0 && eff.Fire < minSafeResist ||
			a&AffixColdEnchanted != 0 && eff.Cold < minSafeResist ||
			a&AffixLightningEnchanted != 0 && eff.Lightning < minSafeResist ||
			a&AffixSpectralHit != 0 && low < minSafeResist
		if bad {
			return min(danger+1, DangerDeadly)
		}
	}
	return danger
}
