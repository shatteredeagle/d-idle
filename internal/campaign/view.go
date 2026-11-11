package campaign

import "time"

// ViewSchema tags the GET /api/campaign body.
const ViewSchema = "campaign/1"

// Chapter states on the journey strip.
const (
	StDone    = "done"
	StSkipped = "skipped"
	StHeld    = "held"
	StCurrent = "current" // the hero is on it now (a go decision)
	StNext    = "next"    // the chapter the hero levels or asks for
	StOpen    = "open"    // prerequisites met, not the next one
	StWaiting = "waiting" // a prerequisite is not done yet (never a lock: Go now still works)
)

// ChapterView is one chapter on the strip.
type ChapterView struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Area        string   `json:"area"`
	Kind        string   `json:"kind"`
	Optional    bool     `json:"optional,omitempty"`
	Reward      string   `json:"reward"`
	Why         string   `json:"why"`
	State       string   `json:"state"`
	Band        Band     `json:"band"`
	Floor       int      `json:"floor"`
	Recommended int      `json:"recommended"`
	Reasons     []string `json:"reasons,omitempty"`
	Intent      Intent   `json:"intent"`
	Asked       bool     `json:"asked,omitempty"` // the floor question is open
}

// Meter is the progress of a farm step toward its level.
type Meter struct {
	From       int     `json:"from"`
	To         int     `json:"to"`
	Pct        float64 `json:"pct"`
	ETASeconds int     `json:"etaSeconds,omitempty"`
}

// View is the GET /api/campaign body.
type View struct {
	Schema     string        `json:"schema"`
	Difficulty string        `json:"difficulty"`
	Mode       Mode          `json:"mode"`
	GoAt       GoAt          `json:"goAt"`
	Level      int           `json:"level"`
	Decision   *Decision     `json:"decision,omitempty"` // nil until the planner ran once
	Meter      *Meter        `json:"meter,omitempty"`
	AskWaitSec int           `json:"askWaitSeconds,omitempty"`
	Chapters   []ChapterView `json:"chapters"`
}

// View is the journey of the current hero from the last decision's inputs.
func (c *Controller) View() View {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.store.Hero(c.hero())
	v := View{Schema: ViewSchema, Difficulty: c.spine.Difficulty, Mode: h.Mode, GoAt: h.GoAt, Level: c.lastProg.Level,
		AskWaitSec: int(c.AskWait / time.Second), Chapters: []ChapterView{}}
	if c.seen {
		d := c.last
		v.Decision = &d
		if (d.Action == ActFarm || d.Action == ActAsk) && d.Level > 0 {
			m := &Meter{From: c.lastProg.Level, To: d.Level}
			if c.Pace != nil {
				if pct, eta, ok := c.Pace(d.Level); ok {
					m.Pct, m.ETASeconds = pct, int(eta/time.Second)
				}
			}
			v.Meter = m
		}
	}
	p := c.lastProg
	sat := func(id string) bool {
		ch := c.spine.Chapter(id)
		return ch.IsDone(p) || h.Intent(ch, p.Class) == Skip
	}
	for i := range c.spine.Chapters {
		ch := &c.spine.Chapters[i]
		in := h.Intent(ch, p.Class)
		cv := ChapterView{ID: ch.ID, Title: ch.Title, Area: ch.Area, Kind: ch.Kind, Optional: ch.Optional, Reward: ch.Reward,
			Why: ch.Why, Intent: in, Asked: !h.Asked[ch.ID].IsZero()}
		if r, err := c.spine.rate(ch, p, c.lastOpts); err == nil && p.Class != "" {
			cv.Band, cv.Floor, cv.Recommended, cv.Reasons = r.Band, r.Floor, r.Recommended, r.Reasons
		}
		open := true
		for _, r := range ch.Requires {
			open = open && sat(r)
		}
		switch {
		case ch.IsDone(p) && c.seen:
			cv.State = StDone
		case in == Skip:
			cv.State = StSkipped
		case in == Hold:
			cv.State = StHeld
		case c.seen && c.last.Chapter == ch.ID && c.last.Action == ActGo:
			cv.State = StCurrent
		case c.seen && c.last.Chapter == ch.ID:
			cv.State = StNext
		case open:
			cv.State = StOpen
		default:
			cv.State = StWaiting
		}
		v.Chapters = append(v.Chapters, cv)
	}
	return v
}
