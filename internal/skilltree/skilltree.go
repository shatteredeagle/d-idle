// Package skilltree builds the read-only view of a hero's class skill trees for the watch Skills page
// (docs/skilltrees.md). It is the model layer: tree and skill nodes with positions, prerequisites, levels and a
// CanAdd verdict with a reason line; the page only draws it. Everything is driven by the skills.txt / skilldesc.txt
// rows of the user's own install (data.Skill) and the hero's skill points: no class, skill id or tab is named here.
package skilltree

import (
	"fmt"
	"sort"
	"strings"

	"d2lod-bot/internal/data"
)

// Schema is the schema of the /api/skills body (additive changes keep the number).
const Schema = "watch.skills/1"

// Tables is what the builder reads from the data layer. Text resolves a string-table key to display text (it may
// return the key itself for an unknown one, as data.DB.Text does).
type Tables struct {
	Skills  []*data.Skill
	Text    func(key string) string
	Classes []string // class names by class id (data.DB.Classes order), for the view's class name
}

// Points are one skill's points: Base are the hard points (V live, d2.Skill.Level), Bonus the +skills from items and
// shrines on top (single-skill bonus V live, the all/class/tab sources U, see docs/skilltrees.md).
type Points struct{ Base, Bonus int }

// Hero is the hero data the view needs. ClassCode is the skills.txt charclass code of the hero's class.
type Hero struct {
	ClassCode string
	ClassName string
	Level     int
	Unspent   int
	Points    map[int]Points // by skill id; a skill the hero has no entry for has 0 points
}

// Prereq is one required skill.
type Prereq struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Have bool   `json:"have"` // the hero has at least one hard point in it
}

// Node is one skill square of a tree.
type Node struct {
	ID       int      `json:"id"`
	Name     string   `json:"name"`
	Row      int      `json:"row"`            // 1 to 6, top to bottom
	Col      int      `json:"col"`            // 1 to 3
	Icon     string   `json:"icon,omitempty"` // art key (internal/art), "" when the class has no icon file
	ReqLevel int      `json:"reqLevel"`
	MaxLevel int      `json:"maxLevel"`
	Passive  bool     `json:"passive,omitempty"`
	Aura     bool     `json:"aura,omitempty"`
	Prereqs  []Prereq `json:"prereqs,omitempty"`
	Base     int      `json:"base"`
	Bonus    int      `json:"bonus,omitempty"`
	Level    int      `json:"level"` // the level it casts at: Base + Bonus
	State    string   `json:"state"` // learned, bonus (only +skills), learnable, locked
	CanAdd   bool     `json:"canAdd"`
	Why      string   `json:"why,omitempty"` // when CanAdd is false: the reason; when true, empty
	Short    string   `json:"short,omitempty"`
	Long     string   `json:"long,omitempty"`
	ManaNow  *float64 `json:"manaNow,omitempty"`  // mana cost at Level (when Level >= 1)
	ManaNext *float64 `json:"manaNext,omitempty"` // mana cost at Level+1
	Synergy  []string `json:"synergy,omitempty"`  // names of the skills that add damage to this one
}

// Tree is one tab of the class.
type Tree struct {
	Tab   int    `json:"tab"` // skilldesc SkillPage, 1 to 3
	Spent int    `json:"spent"`
	Nodes []Node `json:"nodes"`
}

// View is the body of GET /api/skills.
type View struct {
	Schema    string `json:"schema"`
	Available bool   `json:"available"`
	Class     string `json:"class,omitempty"`
	Level     int    `json:"level,omitempty"`
	Unspent   int    `json:"unspent"`
	Spent     int    `json:"spent"`
	Trees     []Tree `json:"trees,omitempty"`
	// Allocate says whether the future allocation action is enabled; it is off today and the page shows no control.
	Allocate bool `json:"allocate"`
}

// Verdict codes of CanAdd.
const (
	WhyNoPoints = "no unspent skill points"
	WhyMax      = "already at the maximum level"
)

// CanAdd says whether one more hard point may go into s now, with the reason when not. Rules (game.exe's own check
// was not read; these are the standard rules of the tables, U live): the hero has an unspent point, the skill is
// under its maxlvl, the character level reaches reqlevel, and every reqskill has at least one hard point. Stat
// requirements (reqstr and so on) are not checked: the 1.14d tables hold none for class skills (U for other tables).
func CanAdd(s *data.Skill, h Hero, byKey map[string]*data.Skill) (bool, string) {
	base := h.Points[s.ID].Base
	if s.MaxLevel > 0 && base >= s.MaxLevel {
		return false, WhyMax
	}
	if h.Level < s.ReqLevel {
		return false, fmt.Sprintf("needs character level %d", s.ReqLevel)
	}
	for _, k := range s.ReqSkills {
		if k == "" {
			continue
		}
		p := byKey[strings.ToLower(k)]
		if p == nil {
			return false, "needs " + k
		}
		if h.Points[p.ID].Base < 1 {
			return false, "needs " + nameOf(p)
		}
	}
	if h.Unspent < 1 {
		return false, WhyNoPoints
	}
	return true, ""
}

func nameOf(s *data.Skill) string {
	if s.Name != "" {
		return s.Name
	}
	return s.Key
}

// Build makes the view for the hero's class. It returns an unavailable view (no error) for a hero without a class
// code or a class with no skill tab rows, so the page can say so instead of failing.
func Build(t Tables, h Hero, iconKey func(s *data.Skill) string) View {
	v := View{Schema: Schema, Class: h.ClassName, Level: h.Level, Unspent: h.Unspent}
	if h.ClassCode == "" {
		return v
	}
	text := func(k string) string {
		if k == "" || t.Text == nil {
			return ""
		}
		out := t.Text(k)
		if out == k { // unknown key: no text rather than the raw key
			return ""
		}
		return out
	}
	byKey := map[string]*data.Skill{}
	var mine []*data.Skill
	for _, s := range t.Skills {
		byKey[strings.ToLower(s.Key)] = s
		if strings.EqualFold(s.Class, h.ClassCode) && s.Page >= 1 && s.Row >= 1 && s.Column >= 1 {
			mine = append(mine, s)
		}
	}
	if len(mine) == 0 {
		return v
	}
	v.Available = true
	pages := map[int]*Tree{}
	for _, s := range mine {
		pt := h.Points[s.ID]
		n := Node{
			ID: s.ID, Name: nameOf(s), Row: s.Row, Col: s.Column, ReqLevel: s.ReqLevel, MaxLevel: s.MaxLevel,
			Passive: s.Passive, Aura: s.Aura, Base: pt.Base, Bonus: pt.Bonus, Level: pt.Base + pt.Bonus,
			Short: text(s.ShortKey), Long: text(s.LongKey),
		}
		if iconKey != nil {
			n.Icon = iconKey(s)
		}
		for _, k := range s.ReqSkills {
			if p := byKey[strings.ToLower(k)]; k != "" && p != nil {
				n.Prereqs = append(n.Prereqs, Prereq{ID: p.ID, Name: nameOf(p), Have: h.Points[p.ID].Base >= 1})
			}
		}
		n.CanAdd, n.Why = CanAdd(s, h, byKey)
		// state: what the player sees. locked = cannot learn it yet (level or prerequisite), whatever the points.
		switch {
		case pt.Base > 0:
			n.State = "learned"
		case pt.Bonus > 0:
			n.State = "bonus"
		case lockedBy(s, h, byKey):
			n.State = "locked"
		default:
			n.State = "learnable"
		}
		if !s.Passive && (s.Mana > 0 || s.MinMana > 0) {
			if n.Level >= 1 {
				m := s.ManaCost(n.Level)
				n.ManaNow = &m
			}
			m := s.ManaCost(n.Level + 1)
			n.ManaNext = &m
		}
		for _, k := range s.Synergies() {
			if p := byKey[strings.ToLower(k)]; p != nil {
				n.Synergy = append(n.Synergy, nameOf(p))
			}
		}
		tr := pages[s.Page]
		if tr == nil {
			tr = &Tree{Tab: s.Page}
			pages[s.Page] = tr
		}
		tr.Nodes = append(tr.Nodes, n)
		tr.Spent += pt.Base
		v.Spent += pt.Base
	}
	for _, tr := range pages {
		sort.Slice(tr.Nodes, func(i, j int) bool {
			a, b := tr.Nodes[i], tr.Nodes[j]
			if a.Row != b.Row {
				return a.Row < b.Row
			}
			return a.Col < b.Col
		})
		v.Trees = append(v.Trees, *tr)
	}
	sort.Slice(v.Trees, func(i, j int) bool { return v.Trees[i].Tab < v.Trees[j].Tab })
	return v
}

func lockedBy(s *data.Skill, h Hero, byKey map[string]*data.Skill) bool {
	if h.Level < s.ReqLevel {
		return true
	}
	for _, k := range s.ReqSkills {
		if p := byKey[strings.ToLower(k)]; k != "" && (p == nil || h.Points[p.ID].Base < 1) {
			return true
		}
	}
	return false
}
