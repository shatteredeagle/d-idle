package watch

import (
	"encoding/json"
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"strings"
	"sync"

	questdata "d2lod-bot/data/quests"
	"d2lod-bot/internal/goals"
)

// Quest hints (TASK-197, docs/quest-hints.md): when the hero walks into the Act 1 town it thinks aloud about the quest
// givers. A quest that is available and not yet given, with the hero at or above its level for the hero's class family,
// gets a "go" line once; below it, a "too weak" line at most once per town visit, and again only after the level rose by
// 2. The hints only SPEAK (feed, Chronicle, return screen); they never change which quest the campaign works on.
// The policy row "questhints" (Quiet, Hints, Hints and ask me) switches them; Quiet is today's behaviour.

const (
	// QuestHintsRow is the policy sheet row; its option ids.
	QuestHintsRow = "questhints"
	HintsQuiet    = "quiet"
	HintsOn       = "hints"
	HintsAsk      = "ask"

	// Thought kinds of the hint lines.
	HintGo   = "quest.hint.go"
	HintWeak = "quest.hint.weak"

	// QuestHintsSectionID is the return screen section.
	QuestHintsSectionID = "questhints"

	hintsFileVersion = 1
	hintLinesKeep    = 40
	hintRearmLevels  = 2
)

var (
	hintFamilies = []string{"melee", "caster", "summoner"}
	hintHows     = []string{"npc_dialog", "auto_on_area", "auto_on_kill"}
	hintTags     = []string{"SRC", "EXP", "UNSURE"}
	hintFields   = []string{"giver", "how", "clvl", "available", "lines"}
)

// HintArea names an area; either the id or the name matches a visited area.
type HintArea struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// HintQuestReq asks another quest to have reached a stage (1 started, 2 cleared, 3 done).
type HintQuestReq struct {
	ID       string `json:"id"`
	StageMin int    `json:"stage_min"`
}

// HintAvail says when a quest giver has something for the hero besides "not given yet".
type HintAvail struct {
	Area  *HintArea     `json:"area,omitempty"`
	Quest *HintQuestReq `json:"quest,omitempty"`
}

// HintAsk is the question the "Hints and ask me" option queues: a catalogue point, its quest field value, and the
// lowest level it is asked at (it is never asked before the quest's own level).
type HintAsk struct {
	Point   string `json:"point"`
	Quest   string `json:"quest"`
	MinClvl int    `json:"min_clvl"`
}

// HintQuest is one row of the table. An empty Giver means the quest has no NPC: it is kept as data and never spoken.
type HintQuest struct {
	ID        string            `json:"id"`
	Giver     string            `json:"giver"`
	How       string            `json:"how"`
	Clvl      map[string]int    `json:"clvl"`
	Available HintAvail         `json:"available"`
	Ask       *HintAsk          `json:"ask,omitempty"`
	Go        []string          `json:"go"`
	TooWeak   []string          `json:"too_weak"`
	Verified  map[string]string `json:"verified"`
}

// HintFamily is the class family of one class.
type HintFamily struct {
	Family   string `json:"family"`
	Verified string `json:"verified"`
}

// HintTown is the town whose entry triggers the hints.
type HintTown struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Verified string `json:"verified"`
}

// HintTable is data/quests/act1-hints.json.
type HintTable struct {
	Schema   string                `json:"schema"`
	Doc      string                `json:"doc"`
	Note     string                `json:"note"`
	Town     HintTown              `json:"town"`
	Families map[string]HintFamily `json:"families"`
	Quests   []HintQuest           `json:"quests"`
}

// ParseHintTable checks the table: families and tags are known, every quest has a level for every family and a tag for
// every field, a quest with a giver has at least 2 lines of each kind and one without has none. Errors name the quest.
func ParseHintTable(b []byte) (*HintTable, error) {
	var t HintTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("quest hints: %w", err)
	}
	if t.Schema != "quests.hints/1" {
		return nil, fmt.Errorf("quest hints: schema %q", t.Schema)
	}
	if t.Town.ID == 0 || t.Town.Name == "" || !containsStr(hintTags, t.Town.Verified) {
		return nil, errors.New("quest hints: the town needs an id, a name and a verified tag")
	}
	if len(t.Families) == 0 {
		return nil, errors.New("quest hints: no class families")
	}
	for c, f := range t.Families {
		if !containsStr(hintFamilies, f.Family) || !containsStr(hintTags, f.Verified) {
			return nil, fmt.Errorf("quest hints: class %q: bad family %q or tag %q", c, f.Family, f.Verified)
		}
	}
	seen := map[string]bool{}
	for _, q := range t.Quests {
		if q.ID == "" || seen[q.ID] {
			return nil, fmt.Errorf("quest hints: bad or duplicate quest id %q", q.ID)
		}
		seen[q.ID] = true
		if !containsStr(hintHows, q.How) {
			return nil, fmt.Errorf("quest hints: quest %q: unknown how %q", q.ID, q.How)
		}
		for _, f := range hintFamilies {
			if q.Clvl[f] < 1 {
				return nil, fmt.Errorf("quest hints: quest %q: no level for family %q", q.ID, f)
			}
		}
		if len(q.Clvl) != len(hintFamilies) {
			return nil, fmt.Errorf("quest hints: quest %q: unknown family in clvl", q.ID)
		}
		fields := hintFields
		if q.Ask != nil {
			fields = append(append([]string(nil), hintFields...), "ask")
			if q.Ask.Point == "" || q.Ask.Quest == "" || q.Ask.MinClvl < 0 {
				return nil, fmt.Errorf("quest hints: quest %q: bad ask", q.ID)
			}
		}
		for _, f := range fields {
			if !containsStr(hintTags, q.Verified[f]) {
				return nil, fmt.Errorf("quest hints: quest %q: field %q needs a verified tag (SRC, EXP or UNSURE)", q.ID, f)
			}
		}
		if q.Giver == "" {
			if len(q.Go) != 0 || len(q.TooWeak) != 0 || q.Ask != nil {
				return nil, fmt.Errorf("quest hints: quest %q has no giver and must have no lines or ask", q.ID)
			}
			continue
		}
		if len(q.Go) < 2 || len(q.TooWeak) < 2 {
			return nil, fmt.Errorf("quest hints: quest %q needs 2 or more go lines and 2 or more too_weak lines", q.ID)
		}
		for _, l := range append(append([]string(nil), q.Go...), q.TooWeak...) {
			if l == "" || len(l) > 160 {
				return nil, fmt.Errorf("quest hints: quest %q: a line is empty or over 160 bytes", q.ID)
			}
		}
	}
	return &t, nil
}

// Family is the class family of a class; an unknown class is an error naming it (never a fallback to another class).
func (t *HintTable) Family(class string) (string, error) {
	f, ok := t.Families[strings.ToLower(class)]
	if !ok {
		return "", fmt.Errorf("quest hints: no class family for class %q", class)
	}
	return f.Family, nil
}

// HintLine is one thing the hero said in town.
type HintLine struct {
	T     float64 `json:"t"`
	Kind  string  `json:"kind"`
	Quest string  `json:"quest"`
	Text  string  `json:"text"`
}

type hintQuestState struct {
	GoSaid bool `json:"go_said,omitempty"`
	WeakAt int  `json:"weak_at,omitempty"` // the level of the last "too weak" line
	Said   int  `json:"said,omitempty"`    // lines said for this quest (seeds the pick)
	Asked  bool `json:"asked,omitempty"`
}

type hintFile struct {
	Version int                        `json:"version"`
	Quests  map[string]*hintQuestState `json:"quests"`
	Stage   map[string]int             `json:"stage"`
	Areas   []int                      `json:"areas"`
	Names   []string                   `json:"names"`
	Errand  string                     `json:"errand,omitempty"`
	Lines   []HintLine                 `json:"lines"`
}

// QuestHintsConfig wires a QuestHints. Mode, Class, Level and Say are required.
type QuestHintsConfig struct {
	Table   *HintTable                      // nil: the embedded table
	Mode    func() string                   // the policy option now chosen for the row
	Class   func() string                   // lower case; "" while unknown
	Level   func() int                      // 0 while unknown
	Now     func() float64                  // Hub time
	Say     func(text string)               // feed, Chronicle
	Ask     func(point, quest string) error // queues the question card; nil: asking is off
	Fail    func(error)                     // reports what went wrong; nil: dropped
	Dir     string                          // "" keeps the state in memory
	Profile string
}

// QuestHints is the hint desk. Safe for concurrent use.
type QuestHints struct {
	c     QuestHintsConfig
	table *HintTable
	path  string

	mu      sync.Mutex
	f       hintFile
	areas   map[int]bool
	names   map[string]bool
	pending bool // a town entry waits for the class and level to be known
	weakNow bool // a "too weak" line was said in this town visit
	noted   map[string]bool
}

// NewQuestHints builds the desk and loads its state file (questhints-<profile>.json in Dir) when there is one.
func NewQuestHints(c QuestHintsConfig) (*QuestHints, error) {
	if c.Mode == nil || c.Class == nil || c.Level == nil || c.Say == nil {
		return nil, errors.New("quest hints: Mode, Class, Level and Say are required")
	}
	q := &QuestHints{c: c, table: c.Table, areas: map[int]bool{}, names: map[string]bool{}, noted: map[string]bool{}}
	if q.table == nil {
		t, err := ParseHintTable(questdata.HintsJSON)
		if err != nil {
			return nil, err
		}
		q.table = t
	}
	if c.Now == nil {
		q.c.Now = func() float64 { return 0 }
	}
	q.f = hintFile{Version: hintsFileVersion, Quests: map[string]*hintQuestState{}, Stage: map[string]int{}}
	if c.Dir != "" {
		q.path = filepath.Join(c.Dir, "questhints-"+c.Profile+".json")
		b, err := os.ReadFile(q.path)
		switch {
		case errors.Is(err, os.ErrNotExist):
		case err != nil:
			return nil, err
		default:
			if err := json.Unmarshal(b, &q.f); err != nil {
				return nil, fmt.Errorf("quest hints: %s: %w", q.path, err)
			}
			if q.f.Version != hintsFileVersion {
				return nil, fmt.Errorf("quest hints: %s: version %d, want %d", q.path, q.f.Version, hintsFileVersion)
			}
			if q.f.Quests == nil {
				q.f.Quests = map[string]*hintQuestState{}
			}
			if q.f.Stage == nil {
				q.f.Stage = map[string]int{}
			}
		}
	}
	for _, a := range q.f.Areas {
		q.areas[a] = true
	}
	for _, n := range q.f.Names {
		q.names[n] = true
	}
	return q, nil
}

func (q *QuestHints) fail(err error) {
	if q.c.Fail != nil {
		q.c.Fail(err)
	}
}

func (q *QuestHints) saveLocked() {
	if q.path == "" {
		return
	}
	b, err := json.MarshalIndent(q.f, "", " ")
	if err == nil {
		tmp := q.path + ".tmp"
		if err = os.WriteFile(tmp, append(b, '\n'), 0o644); err == nil {
			err = os.Rename(tmp, q.path)
		}
	}
	if err != nil {
		q.fail(fmt.Errorf("quest hints: %w", err))
	}
}

func (q *QuestHints) isTown(ev goals.Event) bool {
	return ev.AreaID == q.table.Town.ID || strings.EqualFold(ev.Area, q.table.Town.Name)
}

// OnGoalEvent learns quest stages and visited areas and, on entering the town, looks at the quest givers. Only Normal
// difficulty quest events count (the table is Act 1 Normal).
func (q *QuestHints) OnGoalEvent(ev goals.Event) {
	switch ev.Type {
	case goals.EvQuest:
		if ev.Diff != 0 || ev.Stage < 1 {
			return
		}
		q.mu.Lock()
		if ev.Stage > q.f.Stage[ev.Quest] {
			q.f.Stage[ev.Quest] = ev.Stage
			q.saveLocked()
		}
		q.mu.Unlock()
	case goals.EvAreaChange:
		q.mu.Lock()
		changed := false
		if ev.AreaID != 0 && !q.areas[ev.AreaID] {
			q.areas[ev.AreaID] = true
			q.f.Areas = append(q.f.Areas, ev.AreaID)
			changed = true
		}
		if ev.Area != "" && !q.names[strings.ToLower(ev.Area)] {
			q.names[strings.ToLower(ev.Area)] = true
			q.f.Names = append(q.f.Names, strings.ToLower(ev.Area))
			changed = true
		}
		if q.isTown(ev) {
			q.pending, q.weakNow = true, false
		} else {
			q.pending = false
		}
		if changed {
			q.saveLocked()
		}
		q.mu.Unlock()
		q.Tick()
	}
}

// Tick looks at the quest givers if a town entry is waiting for the hero's class and level (call it from the status loop).
func (q *QuestHints) Tick() {
	q.mu.Lock()
	if !q.pending {
		q.mu.Unlock()
		return
	}
	mode := q.c.Mode()
	if mode != HintsOn && mode != HintsAsk {
		q.pending = false // Quiet: nothing is said and nothing is kept
		q.mu.Unlock()
		return
	}
	class, level := strings.ToLower(q.c.Class()), q.c.Level()
	if class == "" || level < 1 {
		q.mu.Unlock()
		return
	}
	q.pending = false
	fam, err := q.table.Family(class)
	if err != nil {
		q.mu.Unlock()
		if !q.noteOnce("family/" + class) {
			q.fail(err)
		}
		return
	}
	var lines []string
	var asks [][2]string
	for i := range q.table.Quests {
		d := &q.table.Quests[i]
		if d.Giver == "" || q.f.Stage[d.ID] >= 1 || !q.availableLocked(d) {
			continue
		}
		st := q.f.Quests[d.ID]
		if st == nil {
			st = &hintQuestState{}
			q.f.Quests[d.ID] = st
		}
		thr := d.Clvl[fam]
		switch {
		case level >= thr:
			if !st.GoSaid {
				st.GoSaid = true
				q.f.Errand = d.ID
				lines = append(lines, q.recordLocked(d, st, HintGo, d.Go))
			}
			if mode == HintsAsk && d.Ask != nil && !st.Asked && level >= max(thr, d.Ask.MinClvl) {
				st.Asked = true
				asks = append(asks, [2]string{d.Ask.Point, d.Ask.Quest})
			}
		case !q.weakNow && (st.WeakAt == 0 || level >= st.WeakAt+hintRearmLevels):
			st.WeakAt, q.weakNow = level, true
			lines = append(lines, q.recordLocked(d, st, HintWeak, d.TooWeak))
		}
	}
	if len(lines) > 0 || len(asks) > 0 {
		q.saveLocked()
	}
	q.mu.Unlock()
	for _, l := range lines {
		q.c.Say(l)
	}
	for _, a := range asks {
		if q.c.Ask == nil {
			continue
		}
		if err := q.c.Ask(a[0], a[1]); err != nil {
			q.fail(fmt.Errorf("quest hints: asking %s: %w", a[0], err))
		}
	}
}

func (q *QuestHints) noteOnce(k string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	was := q.noted[k]
	q.noted[k] = true
	return was
}

func (q *QuestHints) availableLocked(d *HintQuest) bool {
	if a := d.Available.Area; a != nil && !q.areas[a.ID] && !q.names[strings.ToLower(a.Name)] {
		return false
	}
	if r := d.Available.Quest; r != nil && q.f.Stage[r.ID] < r.StageMin {
		return false
	}
	return true
}

// recordLocked picks the line (seeded by quest, kind and how many lines the quest has said, so a replay says the same
// thing) and keeps it for the return screen.
func (q *QuestHints) recordLocked(d *HintQuest, st *hintQuestState, kind string, pool []string) string {
	h := fnv.New32a()
	fmt.Fprintf(h, "%s/%s/%d", d.ID, kind, st.Said)
	st.Said++
	text := safeText(pool[int(h.Sum32())%len(pool)], 200)
	q.f.Lines = append(q.f.Lines, HintLine{T: q.c.Now(), Kind: kind, Quest: d.ID, Text: text})
	if len(q.f.Lines) > hintLinesKeep {
		q.f.Lines = append([]HintLine(nil), q.f.Lines[len(q.f.Lines)-hintLinesKeep:]...)
	}
	return text
}

// Errand is the quest id of the latest "go" hint, "" when there is none. It is read-only information: the campaign plan
// still chooses the quests.
func (q *QuestHints) Errand() string {
	q.mu.Lock()
	defer q.mu.Unlock()
	return q.f.Errand
}

// Lines are the things said so far, oldest first.
func (q *QuestHints) Lines() []HintLine {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]HintLine(nil), q.f.Lines...)
}

// Section is the return screen's "What I noticed in town": the hint lines since the player left, hidden when empty.
func (q *QuestHints) Section() ReturnSection {
	return SectionFunc(func(d *ReturnData) *SectionView {
		var out []string
		for _, l := range q.Lines() {
			if l.T >= d.Since {
				out = append(out, l.Text)
			}
		}
		if len(out) == 0 {
			return nil
		}
		return &SectionView{ID: QuestHintsSectionID, Title: "What I noticed in town", Lines: out}
	})
}
