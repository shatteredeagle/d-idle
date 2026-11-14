// Package levelplan is the level plan v3 (docs/levelplan3.md): per class and difficulty an ordered list of farm and
// quest slices, loaded from internal/plandata/levelplan-<difficulty>.json. Farm slices say where the hero levels
// between quests; quest slices only name a chapter of the campaign spine (internal/campaign), which keeps deciding
// when a quest goes. The package also carries the AREAMON XP table and a minute model to predict a trajectory.
package levelplan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strings"

	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/plandata"
)

// Schema is the schema string of a plan file.
const Schema = "levelplan/3"

// Slice kinds.
const (
	KindFarm  = "farm"
	KindQuest = "quest"
)

// Slice is one entry of a class plan.
type Slice struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
	// Farm slices: the areas farmed, the band [Entry, Exit] (both inclusive), the extra levels without a living merc,
	// the level below which a full kit is needed, and the way in (via_wp: the waypoint the route starts from).
	Areas     []int  `json:"areas,omitempty"`
	Entry     int    `json:"entry,omitempty"`
	Exit      int    `json:"exit,omitempty"`
	NoMercAdd int    `json:"no_merc_add,omitempty"`
	KitClvl   int    `json:"kit_clvl,omitempty"`
	WayIn     string `json:"way_in,omitempty"`
	ViaWP     int    `json:"via_wp,omitempty"`
	// Quest slices: the spine chapter id.
	Chapter string `json:"chapter,omitempty"`
	Why     string `json:"why"`
}

// EntryFor is the entry level for a hero with or without a living merc.
func (s Slice) EntryFor(merc bool) int {
	if merc {
		return s.Entry
	}
	return s.Entry + s.NoMercAdd
}

// ClassPlan is the plan of one class.
type ClassPlan struct {
	Class  string  `json:"-"`
	Active bool    `json:"active"`
	Build  string  `json:"build"`
	Tag    string  `json:"tag"`
	Slices []Slice `json:"slices"`
}

// Farms are the farm slices in plan order.
func (c *ClassPlan) Farms() []Slice {
	var out []Slice
	for _, s := range c.Slices {
		if s.Kind == KindFarm {
			out = append(out, s)
		}
	}
	return out
}

// Plan is one difficulty's file.
type Plan struct {
	Schema     string                `json:"schema"`
	Doc        string                `json:"doc"`
	Difficulty string                `json:"difficulty"`
	Note       string                `json:"note"`
	Classes    map[string]*ClassPlan `json:"classes"`
}

// For is the plan of class (lowercased d2.Class name). An unknown class is an error naming it, never another class.
func (p *Plan) For(class string) (*ClassPlan, error) {
	class = strings.ToLower(class)
	if c, ok := p.Classes[class]; ok {
		return c, nil
	}
	names := make([]string, 0, len(p.Classes))
	for n := range p.Classes {
		names = append(names, n)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("levelplan: no %s plan for class %q (have %v)", p.Difficulty, class, names)
}

// Active is the plan of class when the bot should use it; ok false: the class has no plan or it is review-only.
func (p *Plan) Active(class string) (*ClassPlan, bool) {
	c, err := p.For(class)
	if err != nil || !c.Active {
		return nil, false
	}
	return c, true
}

// Refs are what a plan is validated against.
type Refs struct {
	Area    func(id int) bool    // a known area id (not a town)
	WP      func(id int) bool    // an area with a waypoint
	Chapter func(id string) bool // a chapter of the spine of the plan's difficulty
}

// Load reads levelplan-<diff>.json from the embedded plan data and validates it against the embedded areas.json and
// the shipped campaign spine (Normal only has a spine today).
func Load(diff string) (*Plan, error) {
	b, err := fs.ReadFile(plandata.FS, "levelplan-"+diff+".json")
	if err != nil {
		return nil, fmt.Errorf("levelplan: %w", err)
	}
	refs, err := ShippedRefs()
	if err != nil {
		return nil, err
	}
	return Parse(b, refs)
}

// ShippedRefs builds Refs from the embedded areas.json and the shipped Normal spine.
func ShippedRefs() (Refs, error) {
	b, err := fs.ReadFile(plandata.FS, "areas.json")
	if err != nil {
		return Refs{}, fmt.Errorf("levelplan: %w", err)
	}
	var top struct {
		Areas []struct {
			ID   int  `json:"id"`
			Town bool `json:"town"`
		} `json:"areas"`
		WPs map[string][]int `json:"waypoints_by_act"`
	}
	if err := json.Unmarshal(b, &top); err != nil {
		return Refs{}, fmt.Errorf("levelplan: areas.json: %w", err)
	}
	areas, wps := map[int]bool{}, map[int]bool{}
	for _, a := range top.Areas {
		areas[a.ID] = !a.Town
	}
	for _, ids := range top.WPs {
		for _, id := range ids {
			wps[id] = true
		}
	}
	sp, err := campaign.Shipped(campaign.Validator{})
	if err != nil {
		return Refs{}, err
	}
	return Refs{
		Area:    func(id int) bool { return areas[id] },
		WP:      func(id int) bool { return wps[id] },
		Chapter: func(id string) bool { return sp.Chapter(id) != nil },
	}, nil
}

// Parse decodes and validates a plan. Errors name the class, the slice and the rule.
func Parse(b []byte, r Refs) (*Plan, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var p Plan
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("levelplan: %w", err)
	}
	if p.Schema != Schema {
		return nil, fmt.Errorf("levelplan: schema %q, want %q", p.Schema, Schema)
	}
	if len(p.Classes) == 0 {
		return nil, fmt.Errorf("levelplan: no classes")
	}
	for name, c := range p.Classes {
		if c == nil {
			return nil, fmt.Errorf("levelplan: class %q is empty", name)
		}
		c.Class = name
		if err := c.validate(r); err != nil {
			return nil, fmt.Errorf("levelplan: %s %s: %w", p.Difficulty, name, err)
		}
	}
	return &p, nil
}

func (c *ClassPlan) validate(r Refs) error {
	if c.Build == "" {
		return fmt.Errorf("no build")
	}
	ids := map[string]bool{}
	var farms []Slice
	for _, s := range c.Slices {
		if s.ID == "" || ids[s.ID] {
			return fmt.Errorf("slice id %q empty or repeated", s.ID)
		}
		ids[s.ID] = true
		switch s.Kind {
		case KindQuest:
			if s.Chapter == "" || len(s.Areas) > 0 || s.Entry != 0 || s.Exit != 0 {
				return fmt.Errorf("quest slice %q: a chapter and no areas or levels (the journey owns them)", s.ID)
			}
			if r.Chapter != nil && !r.Chapter(s.Chapter) {
				return fmt.Errorf("quest slice %q: unknown chapter %q", s.ID, s.Chapter)
			}
		case KindFarm:
			if err := s.validateFarm(r); err != nil {
				return fmt.Errorf("farm slice %q: %w", s.ID, err)
			}
			farms = append(farms, s)
		default:
			return fmt.Errorf("slice %q: kind %q", s.ID, s.Kind)
		}
	}
	if len(farms) == 0 {
		return fmt.Errorf("no farm slice")
	}
	for i := 1; i < len(farms); i++ {
		a, b := farms[i-1], farms[i]
		if b.Entry < a.Entry || b.Exit < a.Exit {
			return fmt.Errorf("bands not monotone: %q [%d,%d] after %q [%d,%d]", b.ID, b.Entry, b.Exit, a.ID, a.Entry, a.Exit)
		}
	}
	for _, merc := range []bool{true, false} {
		if l, ok := Gap(farms, merc); !ok {
			return fmt.Errorf("no farm slice holds clvl %d (merc %v)", l, merc)
		}
	}
	return nil
}

func (s Slice) validateFarm(r Refs) error {
	if s.Chapter != "" {
		return fmt.Errorf("a farm slice names no chapter")
	}
	if len(s.Areas) == 0 {
		return fmt.Errorf("no areas")
	}
	for _, a := range s.Areas {
		if r.Area != nil && !r.Area(a) {
			return fmt.Errorf("area %d is unknown or a town", a)
		}
	}
	if s.Entry < 1 || s.Entry >= s.Exit {
		return fmt.Errorf("entry %d must be >= 1 and below exit %d", s.Entry, s.Exit)
	}
	if s.NoMercAdd < 0 || s.KitClvl < 0 {
		return fmt.Errorf("negative no_merc_add or kit_clvl")
	}
	if s.EntryFor(false) > s.Exit {
		return fmt.Errorf("entry without a merc %d is above exit %d", s.EntryFor(false), s.Exit)
	}
	switch s.WayIn {
	case "walk":
	case "waypoint":
		if s.ViaWP == 0 {
			return fmt.Errorf("way_in waypoint needs via_wp")
		}
	default:
		return fmt.Errorf("way_in %q (walk or waypoint)", s.WayIn)
	}
	if s.ViaWP != 0 && r.WP != nil && !r.WP(s.ViaWP) {
		return fmt.Errorf("via_wp %d has no waypoint", s.ViaWP)
	}
	return nil
}

// Gap returns the first clvl from the lowest entry to the highest exit that no farm slice holds; ok is true when
// there is none.
func Gap(farms []Slice, merc bool) (int, bool) {
	lo, hi := 0, 0
	for _, s := range farms {
		if lo == 0 || s.EntryFor(merc) < lo {
			lo = s.EntryFor(merc)
		}
		hi = max(hi, s.Exit)
	}
	for l := lo; l <= hi; l++ {
		held := false
		for _, s := range farms {
			if s.EntryFor(merc) <= l && l <= s.Exit {
				held = true
				break
			}
		}
		if !held {
			return l, false
		}
	}
	return 0, true
}
