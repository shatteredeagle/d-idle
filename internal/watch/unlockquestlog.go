package watch

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	questdata "d2lod-bot/data/quests"
	"d2lod-bot/internal/goals"
)

// The Act 1 quest log (TASK-156, docs/unlock-questlog.md): the quests the hero holds, each with its current objective and
// why the hero is going there. Akara handing out the Den of Evil quest opens it (ladder row "questlog", trigger
// quest_stage den_of_evil >= 1). The states come from the goals EvQuest events, which the bot's quest reader already
// emits (memory words first, the save as prior knowledge; cmd/d2bot/task.go). Nothing here reads the game.

const (
	// QuestLogRoute is GET (viewer); QuestLogID is the ladder row.
	QuestLogRoute = "/api/questlog"
	QuestLogID    = "questlog"

	questLogSchema = "watch.questlog/1"
)

// questStages are the stage names of the table, indexed by the task layer's stage number minus 1.
var questStages = []string{"started", "cleared", "done"}

// QuestStageText is the objective and the reason for one stage.
type QuestStageText struct {
	Objective string `json:"objective"`
	Why       string `json:"why"`
}

// QuestDef is one row of the table.
type QuestDef struct {
	ID     string                    `json:"id"`
	Name   string                    `json:"name"`
	Giver  string                    `json:"giver"`
	Stages map[string]QuestStageText `json:"stages"`
	Reward map[string]string         `json:"reward,omitempty"` // by difficulty name
}

// QuestTable is the parsed Act 1 table.
type QuestTable struct {
	Schema string     `json:"schema"`
	Doc    string     `json:"doc"`
	Note   string     `json:"note"`
	Quests []QuestDef `json:"quests"`
}

var questDiffs = []string{"normal", "nightmare", "hell"}

// ParseQuestTable checks the table: each quest has a name, a giver and all three stages with an objective and a reason;
// ids are unique; a reward names a known difficulty. Errors name the quest.
func ParseQuestTable(b []byte) (*QuestTable, error) {
	var t QuestTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("quest table: %w", err)
	}
	if t.Schema != "questlog.quests/1" {
		return nil, fmt.Errorf("quest table: schema %q", t.Schema)
	}
	seen := map[string]bool{}
	for _, q := range t.Quests {
		if q.ID == "" || q.Name == "" || q.Giver == "" {
			return nil, fmt.Errorf("quest table: quest %q needs an id, a name and a giver", q.ID)
		}
		if seen[q.ID] {
			return nil, fmt.Errorf("quest table: duplicate quest %q", q.ID)
		}
		seen[q.ID] = true
		for _, s := range questStages {
			if st := q.Stages[s]; st.Objective == "" || st.Why == "" {
				return nil, fmt.Errorf("quest table: quest %q stage %q needs an objective and a reason", q.ID, s)
			}
		}
		for s := range q.Stages {
			if !containsStr(questStages, s) {
				return nil, fmt.Errorf("quest table: quest %q: unknown stage %q", q.ID, s)
			}
		}
		for d := range q.Reward {
			if !containsStr(questDiffs, d) {
				return nil, fmt.Errorf("quest table: quest %q: unknown difficulty %q", q.ID, d)
			}
		}
	}
	return &t, nil
}

// QuestRow is one quest of the log as the viewer sees it.
type QuestRow struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Giver     string  `json:"giver"`
	State     string  `json:"state"` // started, cleared, done
	Objective string  `json:"objective"`
	Why       string  `json:"why"`
	Reward    string  `json:"reward,omitempty"`
	Since     float64 `json:"since"` // Hub time of the last stage change
}

// QuestLogView is GET /api/questlog. Quests the hero has not been given are not listed, only counted (no spoilers).
type QuestLogView struct {
	Schema     string     `json:"schema"`
	Open       bool       `json:"open"`
	Title      string     `json:"title"`
	Teaser     string     `json:"teaser,omitempty"`
	Difficulty string     `json:"difficulty,omitempty"`
	Quests     []QuestRow `json:"quests"`
	NotGiven   int        `json:"notGiven"`
}

type questSeen struct {
	diff, stage int
	t           float64
}

// QuestLog keeps the stage of every table quest. Safe for concurrent use.
type QuestLog struct {
	u     *Unlocks
	table *QuestTable
	now   func() float64

	mu   sync.Mutex
	seen map[string]questSeen // by quest id: the highest difficulty, then the highest stage
}

// NewQuestLog builds the log over the ladder. A ladder without the row "questlog" is an error naming it; table nil reads
// the embedded table.
func NewQuestLog(u *Unlocks, table *QuestTable, now func() float64) (*QuestLog, error) {
	if _, err := u.Engine.Check(QuestLogID); err != nil {
		return nil, err
	}
	if table == nil {
		var err error
		if table, err = ParseQuestTable(questdata.JSON); err != nil {
			return nil, err
		}
	}
	if now == nil {
		now = func() float64 { return 0 }
	}
	return &QuestLog{u: u, table: table, now: now, seen: map[string]questSeen{}}, nil
}

func (q *QuestLog) def(id string) (QuestDef, bool) {
	for _, d := range q.table.Quests {
		if d.ID == id {
			return d, true
		}
	}
	return QuestDef{}, false
}

// OnGoalEvent records a quest event. A quest the Act 1 table does not know is ignored (other acts); a stage that does
// not rise (a replay) changes nothing. Recording does not depend on the log being open.
func (q *QuestLog) OnGoalEvent(ev goals.Event) {
	if ev.Type != goals.EvQuest || ev.Stage < 1 || ev.Stage > len(questStages) || ev.Diff < 0 || ev.Diff >= len(questDiffs) {
		return
	}
	if _, ok := q.def(ev.Quest); !ok {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	cur, have := q.seen[ev.Quest]
	if have && (ev.Diff < cur.diff || (ev.Diff == cur.diff && ev.Stage <= cur.stage)) {
		return
	}
	q.seen[ev.Quest] = questSeen{diff: ev.Diff, stage: ev.Stage, t: q.now()}
}

// View is the payload. Locked: the teaser only.
func (q *QuestLog) View() QuestLogView {
	st, _ := q.u.state(QuestLogID)
	v := QuestLogView{Schema: questLogSchema, Open: st.Open, Title: st.Title, Quests: []QuestRow{}}
	if !st.Open {
		v.Teaser = st.Teaser
		return v
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	maxDiff := -1
	for _, d := range q.table.Quests {
		s, ok := q.seen[d.ID]
		if !ok {
			v.NotGiven++
			continue
		}
		if s.diff > maxDiff {
			maxDiff = s.diff
		}
		name := questStages[s.stage-1]
		text := d.Stages[name]
		row := QuestRow{ID: d.ID, Name: d.Name, Giver: d.Giver, State: name, Objective: text.Objective, Why: text.Why, Since: s.t}
		row.Reward = d.Reward[questDiffs[s.diff]]
		v.Quests = append(v.Quests, row)
	}
	if maxDiff >= 0 {
		v.Difficulty = questDiffs[maxDiff]
	}
	return v
}

// Handler serves GET /api/questlog.
func (q *QuestLog) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, q.View())
	})
}
