package combat

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// The skill model and the per-class attack profiles are data, not code (docs/combat.md "Class profiles"):
// classdata/skills.json says how each skill is aimed (kind, hand, reach, hold), classdata/profiles.json says which
// skills a class attacks with per level band, which aura it keeps up and which F-keys it uses. Mana, level and
// prerequisites come from skills.txt through the Catalog. Tags: the Sorceress rows are the former skillTable
// (VERIFIED LIVE for Fire Bolt); every other row is U (Kolbot ranges and skills.txt, not exercised live) except
// the Paladin Zeal/Holy Fire melee loop (V, docs/classes/smoke-paladin.md).

//go:embed classdata/*.json
var classFS embed.FS

// Hand says which mouse button uses the skill. Hotkeys always select the RIGHT skill; the left skill is chosen in
// the left popup (bindleft.go).
type Hand int

const (
	HandRight Hand = iota
	HandLeft
)

func (h Hand) String() string { return [...]string{"right", "left"}[h] }

type skillJSON struct {
	Name  string `json:"name"`
	Kind  string `json:"kind"`
	Range int    `json:"range"`
	Hand  string `json:"hand"`
	Hold  bool   `json:"hold"` // the button is held on the target (Zeal's chain, Whirlwind)
}

// AuraPref is one entry of an aura ladder. Elem is the damage type the aura adds: it is skipped against targets
// immune to it (Holy Fire vs fire immune falls back to Might).
type AuraPref struct {
	Name string `json:"name"`
	Elem string `json:"elem"`
	// RestLifePct > 0 makes this a resting aura (Prayer): wanted only while resting with life below this percent and
	// no monster adjacent; in a fight it is the last resort when no other aura is known.
	RestLifePct int `json:"restLifePct"`
}

// Band is a level band of a profile: the first band that is reached and whose Requires skill is known wins.
type Band struct {
	MinLevel int        `json:"minLevel"`
	Requires string     `json:"requires"`
	Attack   []string   `json:"attack"`
	Aura     []AuraPref `json:"aura"`
}

// MeleeParams are the knobs of a melee profile (positioning and the left-hand loop).
type MeleeParams struct {
	Reach           int `json:"reach"`           // approach to this many subtiles before attacking
	HoldMs          int `json:"holdMs"`          // how long the left button is held per attack tick (hold skills)
	RetreatLifePct  int `json:"retreatLifePct"`  // step back at or below this life percent
	Surround        int `json:"surround"`        // this many monsters within 3 subtiles count as surrounded
	SurroundLifePct int `json:"surroundLifePct"` // back out of a surround only below this life percent
	ManaReserve     int `json:"manaReserve"`     // extra mana needed to switch TO a mana-costing left skill (hysteresis)
}

// Profile is the combat profile of a class (and Sorceress build).
type Profile struct {
	Class  string            `json:"class"`
	Build  string            `json:"build"`
	Bands  []Band            `json:"bands"`
	Aura   []AuraPref        `json:"aura"`
	Buffs  [][]string        `json:"buffs"`
	Layout map[string]string `json:"layout"`
	Melee  *MeleeParams      `json:"melee"`
}

type classData struct {
	skills   map[string]skillMeta
	profiles []Profile
	names    map[string]bool // every skill name some profile mentions
}

var (
	classOnce sync.Once
	classDB   classData
)

func loadClassData() *classData {
	classOnce.Do(func() {
		var sk struct{ Skills []skillJSON }
		var pr struct{ Profiles []Profile }
		for name, v := range map[string]any{"classdata/skills.json": &sk, "classdata/profiles.json": &pr} {
			b, err := classFS.ReadFile(name)
			if err == nil {
				err = json.Unmarshal(b, v)
			}
			if err != nil {
				panic(fmt.Sprintf("combat: %s: %v", name, err))
			}
		}
		classDB.skills = map[string]skillMeta{}
		for _, s := range sk.Skills {
			classDB.skills[s.Name] = skillMeta{Kind: kindOf(s.Kind), Range: s.Range, Hand: handOf(s.Hand), Hold: s.Hold}
		}
		classDB.profiles = pr.Profiles
		classDB.names = map[string]bool{}
		for _, p := range pr.Profiles {
			for _, b := range p.Bands {
				for _, n := range b.Attack {
					classDB.names[n] = true
				}
				for _, a := range b.Aura {
					classDB.names[a.Name] = true
				}
			}
			for _, a := range p.Aura {
				classDB.names[a.Name] = true
			}
			for _, g := range p.Buffs {
				for _, n := range g {
					classDB.names[n] = true
				}
			}
			for n := range p.Layout {
				classDB.names[n] = true
			}
		}
	})
	return &classDB
}

func kindOf(s string) SkillKind {
	switch strings.ToLower(s) {
	case "bolt":
		return KindBolt
	case "ground":
		return KindGround
	case "self":
		return KindSelf
	case "buff":
		return KindBuff
	case "melee":
		return KindMelee
	case "aura":
		return KindAura
	case "summon":
		return KindSummon
	case "charged":
		return KindCharged
	}
	panic("combat: unknown skill kind " + s)
}

func handOf(s string) Hand {
	if strings.EqualFold(s, "left") {
		return HandLeft
	}
	return HandRight
}

// defaultMeleeRange is the reach of a melee skill whose range the table does not give (subtiles; UNVERIFIED).
const defaultMeleeRange = 7

// deriveMeta classifies a skill from skills.txt alone, for skills a profile names but skills.json lacks.
func deriveMeta(s *data.Skill) (skillMeta, bool) {
	switch {
	case s.Passive:
		return skillMeta{}, false
	case s.Aura:
		return skillMeta{Kind: KindAura}, true
	case s.Summon != "":
		return skillMeta{Kind: KindSummon}, true
	}
	hand := HandRight
	if s.LeftSkill {
		hand = HandLeft
	}
	switch strings.ToLower(s.Range) {
	case "h2h":
		return skillMeta{Kind: KindMelee, Range: defaultMeleeRange, Hand: hand}, true
	case "rng", "both":
		if s.Missile != "" {
			return skillMeta{Kind: KindBolt, Range: 20, Hand: hand}, true
		}
		return skillMeta{Kind: KindGround, Range: 20, Hand: hand}, true
	}
	return skillMeta{Kind: KindBuff}, true
}

func classKey(c d2.Class) string { return strings.ToLower(c.String()) }

// ProfileFor returns the profile of a class. Only the Sorceress has builds; other classes have one profile.
// Never nil: an unknown class gets the Sorceress profile.
func ProfileFor(c d2.Class, b Build) *Profile {
	db := loadClassData()
	build := "blizzard"
	if b == BuildLightning {
		build = "lightning"
	}
	for i := range db.profiles {
		p := &db.profiles[i]
		if p.Class == classKey(c) && (p.Build == "" || p.Build == build) {
			return p
		}
	}
	return ProfileFor(d2.Sorceress, b)
}

// Profiles lists every profile (tests, docs).
func Profiles() []Profile { return loadClassData().profiles }

// band picks the active band.
func (p *Profile) band(clvl int, known func(string) bool) *Band {
	for i := range p.Bands {
		b := &p.Bands[i]
		if clvl >= b.MinLevel && (b.Requires == "" || known(b.Requires)) {
			return b
		}
	}
	return nil
}

// AttackNames is the preference list of attack skill names at a character level.
func (p *Profile) AttackNames(clvl int, known func(string) bool) []string {
	if b := p.band(clvl, known); b != nil {
		return b.Attack
	}
	return nil
}

// AuraLadder is the aura preference list of the active band (the profile's list when the band has none).
func (p *Profile) AuraLadder(clvl int, known func(string) bool) []AuraPref {
	if b := p.band(clvl, known); b != nil && len(b.Aura) > 0 {
		return b.Aura
	}
	return p.Aura
}

// MeleeClass reports whether the profile fights in melee with the left hand.
func (p *Profile) MeleeClass() bool { return p != nil && p.Melee != nil }

// classOfSkills guesses the class code from the character's skills (skills.txt charclass), "" without evidence.
func classOfSkills(p *d2.Player, cat Catalog) string {
	n := map[string]int{}
	for _, sk := range p.Skills {
		if def := cat.Skill(sk.ID); def != nil && def.Class != "" && sk.Effective() > 0 {
			n[def.Class]++
		}
	}
	best, bc := "", 0
	for c, k := range n {
		if k > bc {
			best, bc = c, k
		}
	}
	return best
}

var classByCode = map[string]d2.Class{"ama": d2.Amazon, "sor": d2.Sorceress, "nec": d2.Necromancer, "pal": d2.Paladin,
	"bar": d2.Barbarian, "dru": d2.Druid, "ass": d2.Assassin}

// SelectProfile picks the profile for a character: the class the skills belong to when skills.txt says so (a
// misread class byte must not send a Paladin into the Sorceress loop), else the Player.Class one; if that profile
// uses none of the character's skills, the profile that uses the most of them (ties: the Sorceress).
func SelectProfile(p *d2.Player, cat Catalog, b Build) *Profile {
	class := p.Class
	if c, ok := classByCode[classOfSkills(p, cat)]; ok {
		class = c
	}
	known := KnownSkills(p, cat, nil)
	score := func(pr *Profile) int {
		n := 0
		for name := range known {
			if name != "Attack" && profileUses(pr, name) {
				n++
			}
		}
		return n
	}
	pr := ProfileFor(class, b)
	if score(pr) > 0 {
		return pr
	}
	best, bs := pr, 0
	for _, c := range []d2.Class{d2.Sorceress, d2.Paladin, d2.Amazon, d2.Necromancer, d2.Barbarian, d2.Druid, d2.Assassin} {
		cand := ProfileFor(c, b)
		if s := score(cand); s > bs {
			best, bs = cand, s
		}
	}
	return best
}

func profileUses(p *Profile, name string) bool {
	for _, b := range p.Bands {
		for _, n := range b.Attack {
			if n == name {
				return true
			}
		}
	}
	for _, g := range p.Buffs {
		for _, n := range g {
			if n == name {
				return true
			}
		}
	}
	for _, a := range p.Aura {
		if a.Name == name {
			return true
		}
	}
	_, ok := p.Layout[name]
	return ok
}
