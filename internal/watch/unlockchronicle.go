package watch

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"strconv"
	"strings"
	"sync"

	chronicledata "d2lod-bot/data/chronicle"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

// The Chronicle (TASK-154, docs/unlock-chronicle.md): a scrolling log of kills, finds and first sights in Diablo II
// words. The first kill opens the feed (ladder row "chronicle"); the first expedition card the viewer reads opens the
// run timeline tape (row "chronicle_tape"). View only: it reads events the tally already reads, never the game.

const (
	// ChronicleRoute is GET (viewer). ChronicleID and ChronicleTapeID are the ladder rows it needs.
	ChronicleRoute  = "/api/chronicle"
	ChronicleID     = "chronicle"
	ChronicleTapeID = "chronicle_tape"

	chronicleSchema = "watch.chronicle/1"
	chronicleMax    = 300
)

// chronicleKinds are the wording kinds the code uses; the table must carry every one with 3 or more variants.
var chronicleKinds = []string{"kill", "kill_xp", "boss", "drop", "area", "level", "death", "first"}

type wordingFile struct {
	Schema string              `json:"schema"`
	Doc    string              `json:"doc"`
	Note   string              `json:"note"`
	Kinds  map[string][]string `json:"kinds"`
}

// ChronicleLine is one line of the feed. N counts lines from 1 for the life of the process.
type ChronicleLine struct {
	N    int     `json:"n"`
	T    float64 `json:"t"`
	Kind string  `json:"kind"`
	Text string  `json:"text"`
}

// ChronicleView is GET /api/chronicle. While a part is locked it carries the teaser and no content.
type ChronicleView struct {
	Schema string          `json:"schema"`
	Open   bool            `json:"open"`
	Title  string          `json:"title"`
	Teaser string          `json:"teaser,omitempty"`
	Lines  []ChronicleLine `json:"lines"`
	Tape   TapeView        `json:"tape"`
	Why    []WhyEntry      `json:"why,omitempty"` // the last reasons, newest first (whyline.go); only while the feed is open
}

// TapeView is the run timeline: the expedition cards, oldest first.
type TapeView struct {
	Open   bool   `json:"open"`
	Title  string `json:"title"`
	Teaser string `json:"teaser,omitempty"`
	Trips  []Trip `json:"trips"`
}

// Chronicle holds the feed. Safe for concurrent use.
type Chronicle struct {
	u     *Unlocks
	words map[string][]string
	now   func() float64
	trips func() []Trip     // nil: the tape stays empty
	why   func() []WhyEntry // nil: no reason history

	mu    sync.Mutex
	n     int
	seq   uint64
	lines []ChronicleLine
}

// ParseWording loads a wording table: every kind in chronicleKinds with 3 or more variants, and no kind the code does not
// know (an error names it).
func ParseWording(b []byte) (map[string][]string, error) {
	var f wordingFile
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("chronicle wording: %w", err)
	}
	if f.Schema != "chronicle.wording/1" {
		return nil, fmt.Errorf("chronicle wording: schema %q", f.Schema)
	}
	for k := range f.Kinds {
		if !containsStr(chronicleKinds, k) {
			return nil, fmt.Errorf("chronicle wording: unknown kind %q", k)
		}
	}
	for _, k := range chronicleKinds {
		if len(f.Kinds[k]) < 3 {
			return nil, fmt.Errorf("chronicle wording: kind %q needs 3 or more variants, has %d", k, len(f.Kinds[k]))
		}
	}
	return f.Kinds, nil
}

// NewChronicle builds the feed over the ladder. A ladder without the row "chronicle" or "chronicle_tape" is an error
// naming the row. words nil reads the embedded table; now gives the Hub's time; trips gives the expedition cards.
func NewChronicle(u *Unlocks, words map[string][]string, now func() float64, trips func() []Trip) (*Chronicle, error) {
	for _, id := range []string{ChronicleID, ChronicleTapeID} {
		if _, err := u.Engine.Check(id); err != nil {
			return nil, err
		}
	}
	if words == nil {
		var err error
		if words, err = ParseWording(chronicledata.JSON); err != nil {
			return nil, err
		}
	}
	if now == nil {
		now = func() float64 { return 0 }
	}
	return &Chronicle{u: u, words: words, now: now, trips: trips}, nil
}

func (c *Chronicle) say(kind, wording string, n int, repl ...string) {
	vs := c.words[wording]
	h := fnv.New32a()
	h.Write([]byte(wording + "/" + strconv.Itoa(n)))
	text := strings.NewReplacer(repl...).Replace(vs[int(h.Sum32())%len(vs)])
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	c.lines = append(c.lines, ChronicleLine{N: c.n, T: c.now(), Kind: kind, Text: safeText(text, 200)})
	if len(c.lines) > chronicleMax {
		c.lines = append([]ChronicleLine(nil), c.lines[len(c.lines)-chronicleMax:]...)
	}
}

// Thought adds a line the hero thinks (a quest hint, TASK-197). It is not in the wording table: the text is the hero's own.
func (c *Chronicle) Thought(text string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.n++
	c.lines = append(c.lines, ChronicleLine{N: c.n, T: c.now(), Kind: "thought", Text: safeText(text, 200)})
	if len(c.lines) > chronicleMax {
		c.lines = append([]ChronicleLine(nil), c.lines[len(c.lines)-chronicleMax:]...)
	}
}

// OnGoalEvent turns one goals event into a line. An event with a Seq at or below the highest seen is a replay and adds
// nothing. Drops never name a quality: the identify reveal owns that.
func (c *Chronicle) OnGoalEvent(ev goals.Event) {
	if ev.Seq != 0 {
		c.mu.Lock()
		replay := ev.Seq <= c.seq
		if !replay {
			c.seq = ev.Seq
		}
		c.mu.Unlock()
		if replay {
			return
		}
	}
	n := int(ev.Seq)
	switch ev.Type {
	case goals.EvKill:
		switch {
		case ev.Monster == "":
		case ev.Boss:
			c.say("kill", "boss", n, "{name}", ev.Monster)
		case ev.XP > 0:
			c.say("kill", "kill_xp", n, "{name}", ev.Monster, "{xp}", strconv.FormatInt(ev.XP, 10))
		default:
			c.say("kill", "kill", n, "{name}", ev.Monster)
		}
	case goals.EvDrop:
		if ev.Item != nil && ev.Item.Name != "" && ev.Item.Quality != "" && ev.Item.Quality != "normal" {
			c.say("drop", "drop", n, "{name}", ev.Item.Name)
		}
	case goals.EvAreaChange:
		if ev.Area != "" {
			c.say("area", "area", n, "{name}", ev.Area)
		}
	case goals.EvLevelUp:
		if ev.Level > 0 {
			c.say("level", "level", n, "{n}", strconv.Itoa(ev.Level))
		}
	case goals.EvDeath:
		who := ev.Monster
		if who == "" {
			who = "an unseen foe"
		}
		c.say("death", "death", n, "{name}", who)
	}
}

// OnFired adds a first-sight line for every new Codex entry (never for something not yet met).
func (c *Chronicle) OnFired(fired []decide.Fired) {
	for _, f := range fired {
		if f.Entry.Subject != "" && f.Entry.Kind != "death" {
			c.say("first", "first", len(f.Entry.ID)+int(f.Entry.FirstSec), "{name}", f.Entry.Subject)
		}
	}
}

// View is the payload. since > 0 lists only the lines after that number.
func (c *Chronicle) View(since int) ChronicleView {
	v := ChronicleView{Schema: chronicleSchema, Lines: []ChronicleLine{}, Tape: TapeView{Trips: []Trip{}}}
	feed, _ := c.u.state(ChronicleID)
	tape, _ := c.u.state(ChronicleTapeID)
	v.Open, v.Title = feed.Open, feed.Title
	v.Tape.Open, v.Tape.Title = tape.Open, tape.Title
	if !feed.Open {
		v.Teaser = feed.Teaser
	}
	if !tape.Open {
		v.Tape.Teaser = tape.Teaser
	}
	if feed.Open {
		c.mu.Lock()
		for _, l := range c.lines {
			if l.N > since {
				v.Lines = append(v.Lines, l)
			}
		}
		c.mu.Unlock()
	}
	if feed.Open && c.why != nil {
		v.Why = c.why()
	}
	if tape.Open && c.trips != nil {
		ts := c.trips()
		for i := len(ts) - 1; i >= 0; i-- { // the return screen lists newest first; the tape runs oldest first
			v.Tape.Trips = append(v.Tape.Trips, ts[i])
		}
	}
	return v
}

// SetWhy gives the Chronicle detail the reason history of the whyline.
func (c *Chronicle) SetWhy(f func() []WhyEntry) { c.why = f }

// Handler serves GET /api/chronicle[?since=N].
func (c *Chronicle) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		since, _ := strconv.Atoi(r.URL.Query().Get("since"))
		writeJSON(w, 200, c.View(since))
	})
}

// state is the engine's view of one entry (zero State, false for an unknown id).
func (u *Unlocks) state(id string) (unlock.State, bool) {
	for _, s := range u.Engine.States("") {
		if s.ID == id {
			return s, true
		}
	}
	return unlock.State{}, false
}
