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

// Grid geometry in logical units. The page draws a tree as a 3 column by 6 row board of TreeW by TreeH units and scales
// the whole board (tiles and lines) together, so nothing is measured in the browser: a tile's position and the end
// points of the prerequisite lines come from these constants only. Row and column are skilldesc SkillRow (1 to 6) and
// SkillColumn (1 to 3) as the table has them (V: read from the real 1.14d tables for the Paladin, Necromancer and
// Druid, every skill has its own cell).
const (
	Cols, Rows   = 3, 6
	CellW, CellH = 100, 84
	TileSize     = 64
	TreeW        = Cols * CellW
	TreeH        = Rows * CellH
)

// Link is one prerequisite line between two tiles of the same tree, from the edge midpoint of the required tile to the
// edge midpoint of the dependent one (bottom to top when the rows differ, side to side in the same row).
type Link struct {
	From int  `json:"from"` // skill id of the required skill
	To   int  `json:"to"`
	X1   int  `json:"x1"`
	Y1   int  `json:"y1"`
	X2   int  `json:"x2"`
	Y2   int  `json:"y2"`
	Have bool `json:"have"` // the required skill has a hard point
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
	Row      int      `json:"row"` // 1 to 6, top to bottom
	Col      int      `json:"col"` // 1 to 3
	X        int      `json:"x"`   // tile rectangle in grid units (TreeW by TreeH board)
	Y        int      `json:"y"`
	W        int      `json:"w"`
	H        int      `json:"h"`
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
	W     int    `json:"w"` // board size in grid units
	H     int    `json:"h"`
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links,omitempty"`
}

// tileRect is the tile of cell (row, col): centred in its cell.
func tileRect(row, col int) (x, y int) {
	return (col-1)*CellW + (CellW-TileSize)/2, (row-1)*CellH + (CellH-TileSize)/2
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
			ID: s.ID, Name: nameOf(s), Row: s.Row, Col: s.Column, W: TileSize, H: TileSize, ReqLevel: s.ReqLevel, MaxLevel: s.MaxLevel,
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
		n.X, n.Y = tileRect(s.Row, s.Column)
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
			tr = &Tree{Tab: s.Page, W: TreeW, H: TreeH}
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
		tr.Links = links(tr.Nodes)
		v.Trees = append(v.Trees, *tr)
	}
	sort.Slice(v.Trees, func(i, j int) bool { return v.Trees[i].Tab < v.Trees[j].Tab })
	return v
}

// links makes one Link per prerequisite whose required skill sits in the same tree.
func links(nodes []Node) []Link {
	at := map[int]Node{}
	for _, n := range nodes {
		at[n.ID] = n
	}
	var out []Link
	for _, n := range nodes {
		for _, p := range n.Prereqs {
			a, ok := at[p.ID]
			if !ok {
				continue
			}
			l := Link{From: a.ID, To: n.ID, Have: p.Have}
			switch {
			case a.Row < n.Row:
				l.X1, l.Y1, l.X2, l.Y2 = a.X+a.W/2, a.Y+a.H, n.X+n.W/2, n.Y
			case a.Row > n.Row:
				l.X1, l.Y1, l.X2, l.Y2 = a.X+a.W/2, a.Y, n.X+n.W/2, n.Y+n.H
			case a.Col < n.Col:
				l.X1, l.Y1, l.X2, l.Y2 = a.X+a.W, a.Y+a.H/2, n.X, n.Y+n.H/2
			default:
				l.X1, l.Y1, l.X2, l.Y2 = a.X, a.Y+a.H/2, n.X+n.W, n.Y+n.H/2
			}
			out = append(out, l)
		}
	}
	return out
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
