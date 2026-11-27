package buildfile

import (
	"strings"

	"d2lod-bot/internal/data"
)

// DefaultMaxLevel is the hard-point cap when the data says nothing (skills.txt maxlvl is 20 for class skills; U).
const DefaultMaxLevel = 20

// SkillInfo is what the format needs to know about one skill.
type SkillInfo struct {
	Name      string // canonical skills.txt name
	Class     string // lower case full class name, "" for non-class skills
	ReqLevel  int
	MaxLevel  int
	ReqSkills []string
}

// Rules is the game data the format depends on. Tests use a fixture, production uses DataRules.
type Rules interface {
	// Skill looks a skill up by name, case-insensitively.
	Skill(name string) (SkillInfo, bool)
	// StartStats is the class's starting str, dex, vit, ene.
	StartStats(class string) ([4]int, bool)
}

var classCode = map[string]string{"ama": "amazon", "sor": "sorceress", "nec": "necromancer", "pal": "paladin", "bar": "barbarian", "dru": "druid", "ass": "assassin"}

// ClassNames are the accepted class values.
func ClassNames() []string {
	return []string{"amazon", "sorceress", "necromancer", "paladin", "barbarian", "druid", "assassin"}
}

// DataRules adapts data.Skill (skills.txt) to Rules. U: not exercised against real tables here (none in the repo);
// Lookup is typically db.SkillByName.
type DataRules struct {
	Lookup func(name string) *data.Skill
	Start  func(class string) ([4]int, bool)
}

func (d DataRules) Skill(name string) (SkillInfo, bool) {
	s := d.Lookup(name)
	if s == nil {
		return SkillInfo{}, false
	}
	in := SkillInfo{Name: s.Key, Class: classCode[strings.ToLower(s.Class)], ReqLevel: s.ReqLevel, MaxLevel: s.MaxLevel}
	for _, r := range s.ReqSkills {
		if r != "" {
			in.ReqSkills = append(in.ReqSkills, r)
		}
	}
	return in, true
}

func (d DataRules) StartStats(class string) ([4]int, bool) { return d.Start(class) }

func (in SkillInfo) maxLevel() int {
	if in.MaxLevel <= 0 {
		return DefaultMaxLevel
	}
	return in.MaxLevel
}

// StdQuests is the 12 extra skill points of Den of Evil, Radament and Izual in three difficulties, at the clvls the
// repo's endgame tables fold them into (V: internal/plandata/build-paladin_hammerdin_endgame90.json "schema" text
// names rows 8, 15, 26, 38, 44, 56, 68, 72, 82; the split per quest is U). No Lam Esen stat points (not in the tables).
func StdQuests() []QuestPoint {
	return []QuestPoint{{8, 1, 0}, {15, 1, 0}, {26, 2, 0}, {38, 1, 0}, {44, 1, 0}, {56, 2, 0}, {68, 1, 0}, {72, 1, 0}, {82, 2, 0}}
}

// questIDs is the order the skill-point quests pay out in each difficulty (data/quest-points.json ids; Izual is two
// points). questID labels the i-th quest skill point of a build with it; a schedule that is not StdQuests' order gets
// labels by count only (U), which the leveler does not mind: it only needs to know a point is a quest's.
var questIDs = [...]string{"den_of_evil", "radament", "izual", "izual"}

func questID(i int) string { return questIDs[i%len(questIDs)] }

func earnedSkill(q []QuestPoint, clvl int) int {
	n := clvl - 1
	for _, p := range q {
		if p.Clvl <= clvl {
			n += p.Skill
		}
	}
	return n
}

func questAt(q []QuestPoint, clvl int) (skill, stat int) {
	for _, p := range q {
		if p.Clvl == clvl {
			skill += p.Skill
			stat += p.Stat
		}
	}
	return
}
