// Package campaign is the guided journey: the campaign spine as data (data/campaign/*.json), readiness bands per
// chapter, the per-hero intents and the next-chapter decision. It never locks a chapter: bands are advice, the player can
// always go now, skip or hold. Design: docs/campaign.md.
package campaign

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	campaigndata "d2lod-bot/data/campaign"
	"d2lod-bot/internal/plandata"
)

// Schema tags a spine file.
const Schema = "campaign/1"

// Chapter kinds.
const (
	KindQuest  = "quest"
	KindBoss   = "boss"
	KindChore  = "chore"
	KindUnlock = "unlock"
)

// Intent is what the player wants for one chapter.
type Intent string

const (
	Auto Intent = "auto" // go when the band allows it
	Ask  Intent = "ask"  // at the floor ask first (default yes after an idle window); comfortable goes like Auto
	Skip Intent = "skip" // done by skip: never planned, counts as done for prerequisites
	Hold Intent = "hold" // paused: not planned, later chapters that require it wait
)

func validIntent(i Intent) bool { return i == Auto || i == Ask || i == Skip || i == Hold }

// Levels is a chapter's level row: Floor is the lowest level the bot goes at (player policy "go at the floor"),
// Recommended the level a careful player has. The Merc pair, when set, applies with a living merc.
type Levels struct {
	Floor           int `json:"floor"`
	Recommended     int `json:"recommended"`
	FloorMerc       int `json:"floorMerc,omitempty"`
	RecommendedMerc int `json:"recommendedMerc,omitempty"`
}

// For returns the pair that applies with or without a living merc.
func (l Levels) For(merc bool) (floor, rec int) {
	floor, rec = l.Floor, l.Recommended
	if merc && l.FloorMerc > 0 {
		floor = l.FloorMerc
	}
	if merc && l.RecommendedMerc > 0 {
		rec = l.RecommendedMerc
	}
	return floor, rec
}

// Done is the completion rule of a chapter; exactly one field is set.
type Done struct {
	Quest    string `json:"quest,omitempty"`    // the task-layer quest id is done
	Waypoint int    `json:"waypoint,omitempty"` // the waypoint of this area id is known
	Level    int    `json:"level,omitempty"`    // the hero has reached this level
}

// Chapter is one step of the spine.
type Chapter struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Act         int               `json:"act"`
	Area        string            `json:"area"`
	AreaID      int               `json:"areaId"`
	Kind        string            `json:"kind"`
	Quest       string            `json:"quest,omitempty"` // the task-layer quest the bot runs for it ("" = no quest handler)
	Requires    []string          `json:"requires,omitempty"`
	Optional    bool              `json:"optional,omitempty"`
	Done        Done              `json:"done"`
	Reward      string            `json:"reward"`
	Why         string            `json:"why"`
	Levels      map[string]Levels `json:"levels,omitempty"`     // "default" plus per-class rows
	LevelsFrom  string            `json:"levelsFrom,omitempty"` // "plandata:<gate>": the class row of plandata.BossLevels
	Checks      []string          `json:"checks,omitempty"`     // internal/readiness check ids that cap the band
	Intent      Intent            `json:"intent,omitempty"`     // default intent ("" = auto)
	ClassIntent map[string]Intent `json:"classIntent,omitempty"`
}

// FarmZone is a place the bot levels in while the next chapter is too early.
type FarmZone struct {
	Area     string `json:"area"`
	AreaID   int    `json:"areaId"`
	MLvl     int    `json:"mlvl"`
	Requires string `json:"requires,omitempty"` // a chapter that must be done first ("" = always open)
}

// Spine is one difficulty's ordered chapter list.
type Spine struct {
	Schema     string     `json:"schema"`
	Doc        string     `json:"doc,omitempty"`
	Difficulty string     `json:"difficulty"`
	Note       string     `json:"note,omitempty"`
	Chapters   []Chapter  `json:"chapters"`
	Farm       []FarmZone `json:"farm"`

	byID map[string]int
}

// Validator holds the checks a spine is validated against. A nil func skips that check (tests of the loader only).
type Validator struct {
	Quest func(id string) bool // a known task-layer quest id
	Check func(id string) bool // a known readiness check id
}

// Shipped parses the embedded Normal spine with v.
func Shipped(v Validator) (*Spine, error) { return Parse(campaigndata.NormalJSON, v) }

// Parse decodes and validates a spine. Errors name the chapter (or farm zone) and the field.
func Parse(b []byte, v Validator) (*Spine, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var s Spine
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("campaign: %w", err)
	}
	if err := s.validate(v); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Spine) validate(v Validator) error {
	if s.Schema != Schema {
		return fmt.Errorf("campaign: schema %q, want %q", s.Schema, Schema)
	}
	if s.Difficulty == "" {
		return fmt.Errorf("campaign: no difficulty")
	}
	if len(s.Chapters) == 0 {
		return fmt.Errorf("campaign: no chapters")
	}
	s.byID = map[string]int{}
	for i := range s.Chapters {
		c := &s.Chapters[i]
		bad := func(f string, a ...any) error {
			return fmt.Errorf("campaign: chapter %q: %s", c.ID, fmt.Sprintf(f, a...))
		}
		if c.ID == "" {
			return fmt.Errorf("campaign: chapter %d has no id", i)
		}
		if _, dup := s.byID[c.ID]; dup {
			return bad("duplicate id")
		}
		if c.Title == "" || c.Why == "" || c.Reward == "" || c.Area == "" {
			return bad("title, area, reward and why are required")
		}
		switch c.Kind {
		case KindQuest, KindBoss, KindChore, KindUnlock:
		default:
			return bad("unknown kind %q", c.Kind)
		}
		for _, r := range c.Requires {
			if _, ok := s.byID[r]; !ok {
				return bad("requires %q, which is not an earlier chapter", r)
			}
		}
		n := 0
		for _, set := range []bool{c.Done.Quest != "", c.Done.Waypoint > 0, c.Done.Level > 0} {
			if set {
				n++
			}
		}
		if n != 1 {
			return bad("done needs exactly one of quest, waypoint, level")
		}
		for _, q := range []string{c.Quest, c.Done.Quest} {
			if q != "" && v.Quest != nil && !v.Quest(q) {
				return bad("unknown quest %q", q)
			}
		}
		for _, id := range c.Checks {
			if v.Check != nil && !v.Check(id) {
				return bad("unknown readiness check %q", id)
			}
		}
		if c.Intent != "" && !validIntent(c.Intent) {
			return bad("unknown intent %q", c.Intent)
		}
		for cl, in := range c.ClassIntent {
			if !validIntent(in) {
				return bad("unknown intent %q for class %q", in, cl)
			}
		}
		if err := c.validateLevels(); err != nil {
			return bad("%v", err)
		}
		s.byID[c.ID] = i
	}
	for _, f := range s.Farm {
		if f.Area == "" || f.MLvl <= 0 {
			return fmt.Errorf("campaign: farm zone %q: area and mlvl are required", f.Area)
		}
		if _, ok := s.byID[f.Requires]; f.Requires != "" && !ok {
			return fmt.Errorf("campaign: farm zone %q: requires unknown chapter %q", f.Area, f.Requires)
		}
	}
	return nil
}

func (c *Chapter) validateLevels() error {
	if (c.LevelsFrom == "") == (len(c.Levels) == 0) {
		return fmt.Errorf("needs exactly one of levels and levelsFrom")
	}
	if c.LevelsFrom != "" {
		gate, ok := strings.CutPrefix(c.LevelsFrom, "plandata:")
		if !ok || plandata.LowestFloor(gate) == 0 {
			return fmt.Errorf("unknown levelsFrom %q", c.LevelsFrom)
		}
		return nil
	}
	if _, ok := c.Levels["default"]; !ok {
		return fmt.Errorf("levels has no default row")
	}
	for cl, l := range c.Levels {
		if l.Floor <= 0 || l.Recommended < l.Floor || (l.FloorMerc > 0 && l.RecommendedMerc > 0 && l.RecommendedMerc < l.FloorMerc) {
			return fmt.Errorf("levels row %q: need 0 < floor <= recommended", cl)
		}
	}
	return nil
}

// Chapter returns the chapter with id (nil when unknown).
func (s *Spine) Chapter(id string) *Chapter {
	if i, ok := s.byID[id]; ok {
		return &s.Chapters[i]
	}
	return nil
}

// LevelsFor is the level row of the chapter for class (lowercased d2.Class name): the class row, else the default row;
// a levelsFrom chapter takes the plandata row and an unknown class is an error naming it (no fallback to another class).
func (c *Chapter) LevelsFor(class string) (Levels, error) {
	class = strings.ToLower(class)
	if c.LevelsFrom != "" {
		gate, _ := strings.CutPrefix(c.LevelsFrom, "plandata:")
		r, err := plandata.BossLevels(gate, class)
		if err != nil {
			return Levels{}, fmt.Errorf("campaign: chapter %q: %w", c.ID, err)
		}
		return Levels{Floor: r.Floor, Recommended: r.Recommended, FloorMerc: r.FloorMerc, RecommendedMerc: r.RecommendedMerc}, nil
	}
	if l, ok := c.Levels[class]; ok {
		return l, nil
	}
	return c.Levels["default"], nil
}

// DefaultIntent is the chapter's intent for class before the player changes it.
func (c *Chapter) DefaultIntent(class string) Intent {
	if i, ok := c.ClassIntent[strings.ToLower(class)]; ok {
		return i
	}
	if c.Intent != "" {
		return c.Intent
	}
	return Auto
}
