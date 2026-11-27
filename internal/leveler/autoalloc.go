package leveler

import (
	"sort"
	"strings"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
)

// Pure helpers of the automatic allocator (docs/autoalloc.md): the same planner and the same game rule as the leveler,
// without a Leveler (no memory, no clicks), so the Points desk can ask "what does this build want next".

// DBRule is the skills.txt prerequisite check for a class code: unknown skills and skills of another class are refused,
// without tables (nil db) everything is allowed.
func DBRule(db *data.DB, code string) Rule {
	return func(name string, level int, lv map[string]int) bool {
		if db == nil {
			return true
		}
		s := db.SkillByName(name)
		if s == nil || s.Page == 0 {
			return false
		}
		if s.Class != "" && code != "" && s.Class != code {
			return false
		}
		if s.ReqLevel > level || (s.MaxLevel > 0 && lv[name] >= s.MaxLevel) {
			return false
		}
		for _, rq := range s.ReqSkills {
			if rq != "" && lv[rq] == 0 {
				return false
			}
		}
		return true
	}
}

// HaveFor is the planner input of a player. base (the item-free attributes) may be nil: the panel values are used then
// and gear bonuses count as spent points, so the caller should pass it when it can read it.
func HaveFor(db *data.DB, p *d2.Player, base *[4]int) Have {
	h := Have{Level: p.Level, StatPts: p.StatPoints, SkillPts: p.SkillPoints, Skills: map[string]int{}}
	h.Attr = [numAttr]int{p.Strength, p.Dexterity, p.Vitality, p.Energy}
	if base != nil {
		h.Attr = *base
	}
	h.Start = h.Attr
	if db != nil {
		if c := db.ClassByName(p.Class.String()); c != nil {
			h.Start = [numAttr]int{c.Str, c.Dex, c.Vit, c.Int}
		}
		h.Skills = OwnedSkills(db, p)
		h.QuestPts = QuestPoints(db, p)
	}
	return h
}

// QuestPoints is the number of skill points p holds beyond the clvl-1 its levels grant: hard points in skill-tree
// skills (the class tree, not Attack or the scroll skills) plus unspent ones. That surplus is what quests paid (Den of
// Evil, Radament, Izual), read from the points themselves so no quest flag or backend is needed.
func QuestPoints(db *data.DB, p *d2.Player) int {
	if db == nil || p == nil {
		return 0
	}
	n := p.SkillPoints
	for _, s := range p.Skills {
		if sk := db.SkillByID(s.ID); sk != nil && sk.Page > 0 {
			n += s.Level
		}
	}
	return max(0, n-max(0, p.Level-1))
}

// ClassCode is the skills.txt class code ("pal") of a class name, "" when unknown.
func ClassCode(db *data.DB, class string) string {
	if db == nil {
		return ""
	}
	if c := db.ClassByName(class); c != nil {
		return c.Code
	}
	return ""
}

// StatKey is the sheet key ("str", "dex", "vit", "ene") of an attribute index.
func StatKey(attr int) string { return [numAttr]string{"str", "dex", "vit", "ene"}[attr] }

// OffBuild names the skills the player owns that the build's table never buys (a manual spend the allocator does not
// undo; it reflows around them). Sorted.
func (b *Build) OffBuild(owned map[string]int) []string {
	in := map[string]bool{}
	for _, r := range b.Rows {
		for _, s := range r.Skills {
			in[s.Skill] = true
		}
	}
	var out []string
	for name, n := range owned {
		if n > 0 && !in[name] && !strings.HasPrefix(name, "_") {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// TableNames lists the build selectors that have a data file, sorted.
func TableNames() []string {
	out := make([]string, 0, len(buildFiles))
	for n := range buildFiles {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
