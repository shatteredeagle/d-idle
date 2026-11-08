package watch

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"d2lod-bot/internal/goals"
)

// The whyline (docs/whyline.md): one plain line under the video saying why the hero does what it does. The bot supplies
// a reason code and parameters (goals.Why, set where it already decides); the sentence comes from whyline.json, never
// from the page. A line is held for whyHold seconds unless the goal changes, and the bot's reasons are read at most
// once per whyEvery seconds, so the line does not flicker.

//go:embed whyline.json
var whylineJSON []byte

const (
	WhylineRoute  = "/api/whyline"
	whylineSchema = "whyline/1"
	whyEvery      = 1.0 // seconds between two reads of the bot's reason
	whyHold       = 3.0 // seconds a line stays unless the goal changes
	whyKeep       = 5   // reasons in the short history
	whyParamMax   = 80
)

// whyCode is one row of the template table.
type whyCode struct {
	Params []string `json:"params"`
	Say    []string `json:"say"`
}

type whyTable struct {
	Neutral string             `json:"neutral"`
	Codes   map[string]whyCode `json:"codes"`
}

// WhyEntry is one reason of the history.
type WhyEntry struct {
	T    float64 `json:"t"`
	Code string  `json:"code"`
	Text string  `json:"text"`
}

// WhylineView is GET /api/whyline.
type WhylineView struct {
	Schema  string     `json:"schema"`
	Code    string     `json:"code,omitempty"`
	Text    string     `json:"text"`
	Since   float64    `json:"since"`
	History []WhyEntry `json:"history"`
}

// Whyline turns the bot's reason codes into the held line. Safe for concurrent use.
type Whyline struct {
	// Debug shows an unknown code or a missing parameter as such; production shows the neutral line.
	Debug bool

	tab     whyTable
	now     func() float64
	mu      sync.Mutex
	cur     WhyEntry
	hist    []WhyEntry
	count   map[string]int
	seen    bool
	read    float64 // time of the last accepted read
	goal    string  // goal id of the last accepted read
	curGoal string  // goal id when the shown line was set
	n       int     // lines shown, picks the phrasing
}

// NewWhyline loads the embedded template table. now returns seconds on the hub clock.
func NewWhyline(now func() float64) (*Whyline, error) {
	var t whyTable
	if err := json.Unmarshal(whylineJSON, &t); err != nil {
		return nil, fmt.Errorf("whyline: template table: %w", err)
	}
	for code, c := range t.Codes {
		if len(c.Say) == 0 {
			return nil, fmt.Errorf("whyline: code %q has no phrasing", code)
		}
	}
	return &Whyline{tab: t, now: now, count: map[string]int{}}, nil
}

// WhyCodes lists the codes of the table, sorted (the bot's constants are tested against it).
func WhyCodes() []string {
	var t whyTable
	_ = json.Unmarshal(whylineJSON, &t)
	out := make([]string, 0, len(t.Codes))
	for c := range t.Codes {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}

// render fills the template of code. A code or parameter the table does not know returns ok false with the debug text.
func (w *Whyline) render(code string, params map[string]string, variant int) (text string, ok bool) {
	c, known := w.tab.Codes[code]
	if !known {
		return fmt.Sprintf("unknown reason code %s", safeText(code, 40)), false
	}
	for _, p := range c.Params {
		if _, has := params[p]; !has {
			return fmt.Sprintf("reason code %s lacks parameter %s", code, p), false
		}
	}
	text = c.Say[variant%len(c.Say)]
	for _, p := range c.Params {
		text = strings.ReplaceAll(text, "{"+p+"}", safeText(params[p], whyParamMax))
	}
	return text, true
}

// Observe reads the bot's current reason. goalID is the running goal; hold is the text of the oldest open question
// ("" when none or when the hero is busy), shown only when the bot gave no reason of its own.
func (w *Whyline) Observe(why goals.Why, activity, goalID, hold string) {
	t := w.now()
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.seen && goalID == w.goal && t-w.read < whyEvery {
		return
	}
	w.read, w.goal = t, goalID
	code, params := why.Code, why.Params
	switch {
	case code == "" && hold != "":
		code, params = "hold", map[string]string{"question": hold}
	case code == "":
		code, params = "activity", map[string]string{"activity": activity}
		if strings.TrimSpace(activity) == "" {
			return // nothing to say yet
		}
	}
	text, ok := w.render(code, params, w.n)
	if !ok {
		if !w.Debug {
			text = w.tab.Neutral
		}
		code = "unknown"
	}
	if w.seen && text == w.cur.Text {
		return
	}
	if w.seen && goalID == w.curGoalLocked() && t-w.cur.T < whyHold {
		return
	}
	w.n++
	changed := !w.seen || code != w.cur.Code
	w.cur, w.seen = WhyEntry{T: t, Code: code, Text: text}, true
	w.curGoal = goalID
	if changed {
		w.count[code]++
		w.hist = append(w.hist, w.cur)
		if len(w.hist) > whyKeep {
			w.hist = w.hist[len(w.hist)-whyKeep:]
		}
	}
}

func (w *Whyline) curGoalLocked() string { return w.curGoal }

// View is the current line and the short history, newest first.
func (w *Whyline) View() WhylineView {
	w.mu.Lock()
	defer w.mu.Unlock()
	v := WhylineView{Schema: whylineSchema, Code: w.cur.Code, Text: w.cur.Text, Since: w.cur.T, History: []WhyEntry{}}
	for i := len(w.hist) - 1; i >= 0; i-- {
		v.History = append(v.History, w.hist[i])
	}
	return v
}

// History is the last reasons, newest first (the Chronicle detail).
func (w *Whyline) History() []WhyEntry { return w.View().History }

// Counts is how often each code started a line, for the session record (counts only, never the text).
func (w *Whyline) Counts() map[string]int {
	w.mu.Lock()
	defer w.mu.Unlock()
	out := make(map[string]int, len(w.count))
	for k, v := range w.count {
		out[k] = v
	}
	return out
}

// Handler serves GET /api/whyline.
func (w *Whyline) Handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(rw, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(rw, 200, w.View())
	})
}
