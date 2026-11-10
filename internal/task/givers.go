package task

import (
	"encoding/json"
	"fmt"
	"strings"

	questdata "d2lod-bot/data/quests"
)

// Quest givers in town (docs/quest-givers.md). In the original game a quest is given by talking to an NPC; when the hero
// walks into the Rogue Encampment with a giver that has something for it (the quest is not given yet, it is available and
// the hero's level is reached), the campaign visits that NPC first. The table is data (data/quests/act1-givers.json) joined
// by quest id to the hint table (act1-hints.json), which owns levels per class family and availability. It only ADDS a
// town visit before the usual plan; the quest handlers, gates and floors are not changed.

type giverAction struct {
	Pick     string `json:"pick"`
	Title    string `json:"title"`
	Why      string `json:"why"`
	Verified string `json:"verified"`
}

type giverRow struct {
	Quest  string       `json:"quest"`
	NPC    string       `json:"npc"`
	Accept *giverAction `json:"accept,omitempty"`
	Turnin *giverAction `json:"turnin,omitempty"`
}

type giverFile struct {
	Schema string     `json:"schema"`
	Doc    string     `json:"doc"`
	Note   string     `json:"note"`
	Givers []giverRow `json:"givers"`
}

type hintRow struct {
	ID        string         `json:"id"`
	Giver     string         `json:"giver"`
	Clvl      map[string]int `json:"clvl"`
	Available struct {
		Area *struct {
			ID int `json:"id"`
		} `json:"area"`
		Quest *struct {
			ID       string `json:"id"`
			StageMin int    `json:"stage_min"`
		} `json:"quest"`
	} `json:"available"`
}

type hintFile struct {
	Families map[string]struct {
		Family string `json:"family"`
	} `json:"families"`
	Quests []hintRow `json:"quests"`
}

// Giver is one joined row.
type Giver struct {
	Quest, NPC string
	Accept     *giverAction
	Turnin     *giverAction
	Clvl       map[string]int // by class family
	AreaSeen   Area           // 0 = no area needed
	QuestReq   string
	QuestStage Stage
}

// GiverTable is the joined table.
type GiverTable struct {
	Givers   []Giver
	families map[string]string // class -> family
}

// ParseGivers joins the giver rows with the hint rows. Errors name the quest.
func ParseGivers(givers, hints []byte) (*GiverTable, error) {
	var gf giverFile
	dec := json.NewDecoder(strings.NewReader(string(givers)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&gf); err != nil {
		return nil, fmt.Errorf("quest givers: %w", err)
	}
	if gf.Schema != "quests.givers/1" {
		return nil, fmt.Errorf("quest givers: schema %q", gf.Schema)
	}
	var hf hintFile
	if err := json.Unmarshal(hints, &hf); err != nil {
		return nil, fmt.Errorf("quest givers: hint table: %w", err)
	}
	t := &GiverTable{families: map[string]string{}}
	for c, f := range hf.Families {
		t.families[strings.ToLower(c)] = f.Family
	}
	hintBy := map[string]hintRow{}
	for _, h := range hf.Quests {
		hintBy[h.ID] = h
	}
	seen := map[string]bool{}
	for _, r := range gf.Givers {
		h, ok := hintBy[r.Quest]
		switch {
		case r.Quest == "" || seen[r.Quest]:
			return nil, fmt.Errorf("quest givers: quest %q is empty or repeated", r.Quest)
		case !ok:
			return nil, fmt.Errorf("quest givers: quest %q has no row in the hint table", r.Quest)
		case r.NPC == "" || !strings.EqualFold(r.NPC, h.Giver):
			return nil, fmt.Errorf("quest givers: quest %q: giver %q, the hint table says %q", r.Quest, r.NPC, h.Giver)
		case r.Accept == nil && r.Turnin == nil:
			return nil, fmt.Errorf("quest givers: quest %q has neither accept nor turnin", r.Quest)
		}
		for _, a := range []*giverAction{r.Accept, r.Turnin} {
			if a != nil && (a.Pick == "" || a.Title == "" || a.Why == "") {
				return nil, fmt.Errorf("quest givers: quest %q: an action needs pick, title and why", r.Quest)
			}
		}
		seen[r.Quest] = true
		g := Giver{Quest: r.Quest, NPC: r.NPC, Accept: r.Accept, Turnin: r.Turnin, Clvl: h.Clvl}
		if a := h.Available.Area; a != nil {
			g.AreaSeen = Area(a.ID)
		}
		if q := h.Available.Quest; q != nil {
			g.QuestReq, g.QuestStage = q.ID, Stage(q.StageMin)
		}
		t.Givers = append(t.Givers, g)
	}
	if len(t.Givers) == 0 {
		return nil, fmt.Errorf("quest givers: no givers")
	}
	return t, nil
}

// ShippedGivers parses the embedded tables.
func ShippedGivers() (*GiverTable, error) {
	return ParseGivers(questdata.GiversJSON, questdata.HintsJSON)
}

// Family is the class family (melee, caster, summoner) of a class; an unknown class is an error naming it.
func (t *GiverTable) Family(class string) (string, error) {
	f, ok := t.families[strings.ToLower(strings.TrimSpace(class))]
	if !ok {
		return "", fmt.Errorf("quest givers: no class family for %q in the hint table", class)
	}
	return f, nil
}

// Errand kinds.
const (
	ErrandAccept = "accept"
	ErrandTurnin = "turnin"
)

// Errand is one visit the hero would make.
type Errand struct {
	Quest, NPC, Pick, Kind, Title, Why string
}

// Errands lists the visits for w, in table order. Nothing outside the Rogue Encampment on Normal. visited reports an
// area as reached (U: the planner has no visited-areas set, so the caller passes waypoint, cleared and swept areas).
func (t *GiverTable) Errands(w World, class string) ([]Errand, error) {
	if !w.InTown || w.Area != RogueEncampment || !normalDiff(w) {
		return nil, nil
	}
	fam, err := t.Family(class)
	if err != nil {
		return nil, err
	}
	var out []Errand
	for _, g := range t.Givers {
		st := w.Quest(g.Quest).Stage
		if g.Turnin != nil && st == StageCleared {
			out = append(out, Errand{g.Quest, g.NPC, g.Turnin.Pick, ErrandTurnin, g.Turnin.Title, g.Turnin.Why})
			continue
		}
		if g.Accept == nil || st != StageNone || w.Level < g.Clvl[fam] {
			continue
		}
		if g.AreaSeen != 0 && !(w.WPs[g.AreaSeen] || w.Cleared[g.AreaSeen] || w.Swept[g.AreaSeen]) {
			continue
		}
		if g.QuestReq != "" && w.Quest(g.QuestReq).Stage < g.QuestStage {
			continue
		}
		out = append(out, Errand{g.Quest, g.NPC, g.Accept.Pick, ErrandAccept, g.Accept.Title, g.Accept.Why})
	}
	return out, nil
}

// WithQuestGivers opts the campaign into town visits to quest givers (the bot opts in, like WithRogueClaim). class names
// the hero's class when asked; onErr gets a table or class error (nil: ignored). A nil table is a no-op.
func (c *Campaign) WithQuestGivers(t *GiverTable, class func() string, onErr func(error)) *Campaign {
	if c != nil && t != nil && class != nil {
		c.givers, c.giverClass, c.giverErr = t, class, onErr
	}
	return c
}

// giverGroup is the executor group of a visit; it is not a quest id, so progress and gates ignore it.
func giverGroup(quest, kind string) string { return "givers." + quest + "." + kind }

// giverPlan is the visit to make now, or false. A quest the campaign does not include (or skips) or a group that is
// blocked or finished this session is left alone, so a visit never repeats and never blocks the chain.
func (c *Campaign) giverPlan(w World, s Session) (Plan, bool) {
	if c.givers == nil {
		return Plan{}, false
	}
	errands, err := c.givers.Errands(w, c.giverClass())
	if err != nil {
		if c.giverErr != nil {
			c.giverErr(err)
		}
		return Plan{}, false
	}
	for _, e := range errands {
		var entry *Entry
		for i := range c.Entries {
			if c.Entries[i].Quest == e.Quest {
				entry = &c.Entries[i]
			}
		}
		if entry == nil || entry.skipWhy(w) != "" || c.complete(w, s, e.Quest) {
			continue
		}
		g := giverGroup(e.Quest, e.Kind)
		if s != nil {
			if _, b := s.Blocked(g); b || s.Finished(g) {
				continue
			}
		}
		done := stageAtLeast(e.Quest, StageStarted)
		if e.Kind == ErrandTurnin {
			done = not(turnIn(e.Quest))
		}
		step := StepTalk(id(g, "talk"), e.Title, e.NPC, e.Pick, done, nil)
		step.Optional = true // a visit that changes nothing is skipped after the failure budget, never a block
		return Plan{ID: g, Group: g, Why: e.Why, Nodes: []Node{{Step: step}}}, true
	}
	return Plan{}, false
}
