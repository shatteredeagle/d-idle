package questboard

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

// The desk (slice S4): the API-shaped face of the Exec for the watch routes (internal/watch/questsapi.go). Every
// action answers a Reply that carries `ok`, a refusal code and a reason in words (the Pause bug lesson: a reply
// without them was shown as "could not do that"); the page reads the result back from View, never from the reply.
// A hand-in is asynchronous (the dialogue runs on the bot loop), so its reply is 202 and the progress is in View.

const (
	SchemaQuests = "watch.quests/1"

	// Refusal codes of Reply.Code.
	CodeOff      = "off"
	CodeNoHero   = "no_hero"
	CodePaused   = "paused"
	CodeBusy     = "busy"
	CodeNotReady = "not_ready"
	// CodeNotOverridable refuses a try anyway for a part whose refusal is not a readiness one (locked, done, parked, running).
	CodeNotOverridable = "not_overridable"
	CodeNotAwait       = "not_awaiting"
	CodeUnknown        = "unknown"
	CodeDone           = "done"
	CodeBadMode        = "bad_mode"
	CodeAutoNever      = "auto_never"
	CodeInternal       = "internal"
	viewCacheTTL       = time.Second
	handInTimeout      = 5 * time.Minute
	historyMax         = 20
)

// DeskConfig is what the desk needs from the bot side. Every field but Exec may be nil.
type DeskConfig struct {
	// Exec returns the executor and the plan store of the loaded hero; a nil Exec means no hero yet.
	Exec func() (*Exec, *Store)
	// Paused reports the player's Pause.
	Paused func() bool
	// Explain is the full readiness (reasons, legacy floor, sentence) of a chunk; without it the offers carry the
	// word and the legacy floor only.
	Explain func(q Quest, c Chunk) (Readiness, bool)
	// Class is the hero's class, for the legacy floor.
	Class func() string
	// Level is the hero's clvl, for "levels to go" on an offer that is not safe yet; nil: the field is left out.
	Level func() int
	// Meter words the mode and its readiness meter (slice S6); it gets the stored mode and the derived status. Without
	// it the view carries the mode only.
	Meter func(mode string, st Status) *Meter
	// Queued says why an accepted chunk is not running yet ("" when it runs); the strip and the running card show it.
	Queued func() string
	// HandInWhy says what holds a queued hand-in tap ("" when the loop is serving it or nothing is queued): not in
	// town, a chore, the goal kind. nil: the generic "on its way" reason.
	HandInWhy func() string
	Now       func() time.Time
	// Refusal is the hero's freshest typed refusal for the "Why not?" row; nil: the row carries the readiness facts only.
	Refusal func() (RefusalFacts, bool)
	// Ctx bounds the asynchronous hand-in goroutines (the process); default Background.
	Ctx context.Context
}

// Desk serves the API of one process. It is safe for concurrent use.
type Desk struct {
	b   *Board
	cfg DeskConfig

	mu     sync.Mutex
	cache  *View
	cached time.Time
	hand   map[string]*handRun // by chunk key
	last   *HandInView

	refused map[string]time.Time // last refused start by chunk key (whynot.go)
}

type handRun struct{ started time.Time }

// NewDesk builds the desk over a board.
func NewDesk(b *Board, cfg DeskConfig) *Desk {
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Ctx == nil {
		cfg.Ctx = context.Background()
	}
	return &Desk{b: b, cfg: cfg, hand: map[string]*handRun{}}
}

// Reply is the answer of every action.
type Reply struct {
	OK      bool   `json:"ok"`
	Pending bool   `json:"pending,omitempty"` // the action runs on; read the result from GET /api/quests
	Already bool   `json:"already,omitempty"` // a repeated press: nothing changed
	Code    string `json:"code,omitempty"`    // refusal code
	Reason  string `json:"reason,omitempty"`  // the refusal (or the news) in words
	Quest   string `json:"quest,omitempty"`
	Chunk   string `json:"chunk,omitempty"`
	State   string `json:"state,omitempty"` // the chunk's state after the call
	Mode    string `json:"mode,omitempty"`
	Awarded []Pay  `json:"awarded,omitempty"` // a hand-in that finished at once (already done) booked this now
	// Changed lists every run the verb touched (the chunk and its chain or dependants) with state and reason after it.
	Changed []ChangedRow `json:"changed,omitempty"`
	// Override says the start was a try anyway and carries the readiness refusal it skipped (for the audit line).
	Override string `json:"override,omitempty"`
	Status   int    `json:"-"` // HTTP status the route uses
}

func refuse(status int, code, format string, a ...any) Reply {
	return Reply{Code: code, Reason: fmt.Sprintf(format, a...), Status: status}
}

// ---- views -----------------------------------------------------------------------------------------------

// View is GET /api/quests.
type View struct {
	Schema     string       `json:"schema"`
	Enabled    bool         `json:"enabled"`
	Reason     string       `json:"reason,omitempty"` // why the board shows nothing (off, no hero)
	Difficulty string       `json:"difficulty,omitempty"`
	Paused     bool         `json:"paused,omitempty"`
	Mode       string       `json:"mode"`
	ModeActs   bool         `json:"modeActs"` // the modes act (slice S6); true whenever the board is on
	Meter      *Meter       `json:"meter,omitempty"`
	AutoStart  AutoView     `json:"autoStart"`
	Offers     []OfferView  `json:"offers"`
	Quests     []QuestRow   `json:"quests"`
	Running    *RunView     `json:"running,omitempty"`
	Awaiting   *RunView     `json:"awaiting,omitempty"`
	Blocked    *RunView     `json:"blocked,omitempty"`
	HandIn     *HandInView  `json:"handIn,omitempty"`
	History    []HistoryRow `json:"history"`
	Stopped    []StoppedRow `json:"stopped,omitempty"` // quests the player stopped in the last day (audit ring)
	Needs      []NeedRow    `json:"needs"`
	Strip      *Strip       `json:"strip,omitempty"`
	Extra      []string     `json:"extra,omitempty"`     // further accepted chunks in the plan file, each waits for the running one
	WhileAway  *AwayCard    `json:"whileAway,omitempty"` // what the board did by itself while the player was away (slice S8)
	Generated  time.Time    `json:"generated"`
}

// AutoView is the auto-start block (slice S8). Available is true while the board is on; the per-quest flag is
// QuestRow.Auto and the toggle posts to /api/quests/<id>/auto.
type AutoView struct {
	Available bool       `json:"available"`
	Note      string     `json:"note"`
	DelaySec  int        `json:"delaySec"`
	Why       string     `json:"why,omitempty"` // why an opted-in offer does not start by itself now, in words
	Countdown *Countdown `json:"countdown,omitempty"`
}

// Countdown is an auto-start that waits out its delay; the page shows it and offers Cancel.
type Countdown struct {
	Quest       string    `json:"quest"`
	QuestTitle  string    `json:"questTitle"`
	Chunk       string    `json:"chunk"`
	EndsAt      time.Time `json:"endsAt"`
	SecondsLeft int       `json:"secondsLeft"`
}

// AwayCard is the "while you were away" summary: the controller's events since the player last dismissed it.
type AwayCard struct {
	Events []AwayEvent `json:"events"`
}

// AwayEvent is one line of the card; Text is in words, the page formats At.
type AwayEvent struct {
	At         time.Time `json:"at"`
	Quest      string    `json:"quest"`
	QuestTitle string    `json:"questTitle"`
	Chunk      string    `json:"chunk"`
	Kind       string    `json:"kind"` // started, handedIn, handInFailed
	Awarded    []Pay     `json:"awarded,omitempty"`
	Text       string    `json:"text"`
}

// OfferView is one chunk that raises an offer.
type OfferView struct {
	Quest       string   `json:"quest"`
	QuestTitle  string   `json:"questTitle"`
	Chunk       string   `json:"chunk"`
	Title       string   `json:"title"`
	Act         int      `json:"act"`
	Risk        string   `json:"risk,omitempty"`
	Readiness   Word     `json:"readiness"`
	Reasons     []Reason `json:"reasons"`
	LegacyFloor int      `json:"legacyFloor,omitempty"`
	Sentence    string   `json:"sentence,omitempty"`
	Minutes     [2]int   `json:"minutes"`
	Startable   bool     `json:"startable"`
	Why         string   `json:"why,omitempty"` // why Start is refused now
	Auto        bool     `json:"auto"`
	AutoOK      bool     `json:"autoOk"`               // the quest may be started by itself (autoStart data is not `never`)
	LevelsToGo  int      `json:"levelsToGo,omitempty"` // by the legacy clvl floor; the page words it ("two levels to go"), never a number
	Pays        []Pay    `json:"pays,omitempty"`       // what Hand in will pay; omitted without a wallet or when nothing is left to earn
}

// QuestRow is one quest of the difficulty with its chunks (the board sheet).
type QuestRow struct {
	ID     string     `json:"id"`
	Title  string     `json:"title"`
	Act    int        `json:"act"`
	State  State      `json:"state"`
	Auto   bool       `json:"auto"`
	AutoOK bool       `json:"autoOk"` // false: the data says this quest always needs the player (hazard); the toggle is greyed
	Chunks []ChunkRow `json:"chunks"`
	Paid   []Pay      `json:"paid,omitempty"` // the Ether the quest and its chunks booked so far (shown under the reward line)
}

// ChunkRow is one part of a quest.
type ChunkRow struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	State     State  `json:"state"`
	Readiness Word   `json:"readiness,omitempty"`
	Why       string `json:"why,omitempty"`
	Confirmed bool   `json:"confirmed,omitempty"`
	// Reason is why a live part does not move (closed code of appendix B.1 plus its sentence); nil for a part that
	// moves or is not live.
	Reason *ReasonView `json:"reason,omitempty"`
	// Action is the one button this part offers the player (the web row renders it, the scenario tapper presses it);
	// nil: nothing to press, and Reason or Why says why.
	Action *ActionView `json:"action,omitempty"`
	// WhyNot is the "Why not?" row: readiness word, floor, need, wake, last refusal; nil for a part that moves.
	WhyNot *WhyNotView `json:"whyNot,omitempty"`
}

// RunView is the accepted chunk: running, awaiting the tap or parked.
type RunView struct {
	Quest      string      `json:"quest"`
	QuestTitle string      `json:"questTitle"`
	Chunk      string      `json:"chunk"`
	Title      string      `json:"title"`
	State      State       `json:"state"`
	Step       string      `json:"step,omitempty"`
	StepNo     int         `json:"stepNo,omitempty"`    // 1-based position of the current step ("step 2 of 4")
	StepCount  int         `json:"stepCount,omitempty"` // steps of the chunk definition
	Steps      []StepRow   `json:"steps,omitempty"`
	PartNo     int         `json:"partNo,omitempty"`    // 1-based chunk position inside the quest ("part 2 of 4"); 0 for a one-chunk quest
	PartCount  int         `json:"partCount,omitempty"` // chunks of the quest: the honest length of the chain
	NPC        string      `json:"npc,omitempty"`       // who the hero walks to or waits at
	Why        string      `json:"why,omitempty"`
	Failures   int         `json:"failures,omitempty"`
	Deaths     int         `json:"deaths,omitempty"`
	Auto       bool        `json:"auto,omitempty"` // the board started this chunk by itself
	Queued     string      `json:"queued,omitempty"`
	Reason     *ReasonView `json:"reason,omitempty"` // why the run does not move now (hold, wait, park, hand-in)
}

// StepRow is one step of the running chunk, by its data name; State is done, now or todo.
type StepRow struct {
	Name  string `json:"name"`
	State string `json:"state"`
}

// HandInView is the hand-in in flight or the last one that finished.
type HandInView struct {
	Quest   string `json:"quest"`
	Chunk   string `json:"chunk"`
	State   string `json:"state"` // running, done, confirming, blocked, failed
	Reason  string `json:"reason,omitempty"`
	Attempt int    `json:"attempts,omitempty"`
	At      string `json:"at,omitempty"`
	// Why is the closed-code reason a running hand-in waits for (handin_in_town or handin_waiting); nil once it ended.
	Why     *ReasonView `json:"why,omitempty"`
	Awarded []Pay       `json:"awarded,omitempty"` // the Ether this hand-in booked (the payoff toast and the row read it)
}

// HistoryRow is a chunk the board ran and that is done now.
type HistoryRow struct {
	Quest     string    `json:"quest"`
	Chunk     string    `json:"chunk"`
	Title     string    `json:"title"`
	Accepted  time.Time `json:"accepted"`
	Confirmed bool      `json:"confirmed"`
	Paid      []Pay     `json:"paid,omitempty"` // what the ledger booked for it (omitted without a wallet)
}

// Verb is a button on a need row.
type Verb struct {
	Label string         `json:"label"`
	Post  string         `json:"post"`
	Body  map[string]any `json:"body,omitempty"`
}

// NeedRow is a Needs-you row (design 4.6), glance-ready.
type NeedRow struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Sev     string `json:"sev"`
	Text    string `json:"text"`
	Sub     string `json:"sub,omitempty"` // the chip: the readiness word of an offer
	Actions []Verb `json:"actions"`
}

// Strip is the video-card strip: the current chunk, step and NPC.
type Strip struct {
	Quest      string `json:"quest"`
	QuestTitle string `json:"questTitle"`
	Chunk      string `json:"chunk"`
	Title      string `json:"title"`
	State      State  `json:"state"`
	Step       string `json:"step,omitempty"`
	StepNo     int    `json:"stepNo,omitempty"`
	StepCount  int    `json:"stepCount,omitempty"`
	NPC        string `json:"npc,omitempty"`
	Queued     string `json:"queued,omitempty"` // accepted but not running: the reason, never silent
}

// QuestRoute is the path of one quest's action.
func QuestRoute(quest, action string) string { return "/api/quests/" + quest + "/" + action }

func (d *Desk) exec() (*Exec, *Store) {
	if d.cfg.Exec == nil {
		return nil, nil
	}
	return d.cfg.Exec()
}

func (d *Desk) paused() bool { return d.cfg.Paused != nil && d.cfg.Paused() }

func (d *Desk) invalidate() { d.mu.Lock(); d.cache = nil; d.mu.Unlock() }

// View derives the board (cached for one second: the page polls, derive reads the game evidence).
func (d *Desk) View() View {
	if x, _ := d.exec(); x != nil {
		x.Touch() // a client asking for the board is the player being present (slice S8, away detection)
	}
	d.mu.Lock()
	if d.cache != nil && d.cfg.Now().Sub(d.cached) < viewCacheTTL {
		v := *d.cache
		d.mu.Unlock()
		return v
	}
	d.mu.Unlock()
	v := d.build()
	d.mu.Lock()
	d.cache, d.cached = &v, d.cfg.Now()
	d.mu.Unlock()
	return v
}

func (d *Desk) build() View {
	v := View{Schema: SchemaQuests, Mode: DefaultMode, Generated: d.cfg.Now().UTC(),
		AutoStart: AutoView{Note: "Starts a quest by itself only when it is safe, the hero is calm and nobody paused a moment ago."},
		Offers:    []OfferView{}, Quests: []QuestRow{}, History: []HistoryRow{}, Needs: []NeedRow{}}
	// Read the hand-in outcome BEFORE the evidence: a "done" outcome then always comes with evidence read after it,
	// so the history row is there (reading it last paired a fresh outcome with older evidence).
	d.mu.Lock()
	last := d.last
	d.mu.Unlock()
	x, st := d.exec()
	if x == nil || st == nil {
		v.Reason = "No hero is loaded yet."
		return v
	}
	x.Settle() // a hand-in the board saw only as T3 pays once T1/T2 confirm it; a booked key pays nothing twice
	if m, err := st.Mode(); err == nil {
		v.Mode = m
	}
	v.Paused = d.paused()
	status, err := x.State()
	if err == ErrOff {
		v.Reason = "The quest board is off (knob quests.board is 0)."
		return v
	}
	if err != nil {
		v.Reason = "The quest board could not be read: " + err.Error()
		return v
	}
	v.Enabled, v.ModeActs = true, true
	v.AutoStart.Available = true
	as := x.AutoState()
	v.AutoStart.DelaySec, v.AutoStart.Why = as.Opt.DelaySec, as.Why
	if p := as.Pending; p != nil {
		left := int(p.Ends.Sub(d.cfg.Now()).Seconds() + 0.999)
		if left < 0 {
			left = 0
		}
		v.AutoStart.Countdown = &Countdown{Quest: p.Quest, QuestTitle: p.Title, Chunk: p.Chunk, EndsAt: p.Ends.UTC(), SecondsLeft: left}
	}
	level := 0
	if d.cfg.Level != nil {
		level = d.cfg.Level()
	}
	if d.cfg.Meter != nil {
		v.Meter = d.cfg.Meter(v.Mode, status)
	}
	v.Difficulty = Difficulties[status.Diff]
	plan, _, _ := st.Load()
	busy := status.Active != nil
	l := x.Ledger()
	smRuns := x.SMRuns()
	headTitle := ""
	if a := status.Active; a != nil {
		if c, err := d.b.Chunk(a.Quest, a.Chunk); err == nil {
			headTitle = c.Title
		}
	}
	d.mu.Lock()
	handing := map[string]bool{}
	for k := range d.hand {
		handing[k] = true
	}
	d.mu.Unlock()
	var rfacts *RefusalFacts
	if d.cfg.Refusal != nil {
		if f, ok := d.cfg.Refusal(); ok {
			rfacts = &f
		}
	}
	class := ""
	if d.cfg.Class != nil {
		class = d.cfg.Class()
	}
	for _, qv := range status.Views {
		q, qerr := d.b.Quest(qv.ID)
		if qerr != nil {
			continue
		}
		row := QuestRow{ID: qv.ID, Title: qv.Title, Act: q.Act, State: qv.State, Auto: plan.AutoStart(status.Diff, qv.ID), AutoOK: q.AutoStart == "allowed", Chunks: []ChunkRow{}}
		for _, cv := range qv.Chunks {
			c, cerr := d.b.Chunk(qv.ID, cv.ID)
			if cerr != nil {
				continue
			}
			if l != nil {
				row.Paid = append(row.Paid, paidFor(q, c, status.Diff, l)...)
			}
			why := cv.Why
			// A chunk held back by the readiness word says the model's reasons, not only "too hard for now" (card andariel1).
			if cv.State == Available && cv.Reason == ReadyTooHard && d.cfg.Explain != nil {
				if rd, ok := d.cfg.Explain(q, c); ok && rd.Sentence != "" {
					why = rd.Sentence
				}
			}
			cr := ChunkRow{ID: cv.ID, Title: c.Title, State: cv.State, Readiness: cv.Reason, Why: why, Confirmed: cv.Confirmed}
			key := ChunkKey(qv.ID, cv.ID)
			var run *Run
			if r, ok := smRuns[key]; ok {
				run = &r
			}
			npc := ""
			switch {
			case c.Giver != nil:
				npc = c.Giver.NPC
			case q.Accept != nil:
				npc = q.Accept.NPC
			}
			if run != nil && afterOnly(cv) {
				// A queued run waits for an earlier part: the sheet says "accepted" (wire state of RunQueued, B.4), not locked.
				cv.State, cr.State = Accepted, Accepted
			}
			cr.Reason = d.chunkReason(qv.ID, cv, run, status.Active, handing[key], map[string]string{"head": headTitle, "quest": q.Title, "npc": npc})
			if cv.State == Awaiting && !plan.accepted(status.Diff, qv.ID, cv.ID) {
				cr.Reason = handInReady() // case C5: the game counts the field work done, Start goes straight to the hand-in
			}
			var off *OfferView
			if cv.State == Ready {
				o := d.offer(q, c, cv, plan.AutoStart(status.Diff, qv.ID), !busy, class, level, v.Paused, status.Diff, l)
				off = &o
				v.Offers = append(v.Offers, o)
			}
			cr.Action = chunkAction(cv.State, off, run, cr.Reason, v.Paused)
			cr.WhyNot = d.whyNot(q, c, cv, cr, rfacts)
			row.Chunks = append(row.Chunks, cr)
			if cv.State == Done && plan.accepted(status.Diff, qv.ID, cv.ID) {
				h := HistoryRow{Quest: qv.ID, Chunk: cv.ID, Title: c.Title, Accepted: acceptedAt(plan, status.Diff, qv.ID, cv.ID), Confirmed: cv.Confirmed}
				if l != nil {
					h.Paid = paidFor(q, c, status.Diff, l)
				}
				v.History = append(v.History, h)
			}
		}
		v.Quests = append(v.Quests, row)
	}
	// newest first, capped
	for i, j := 0, len(v.History)-1; i < j; i, j = i+1, j-1 {
		v.History[i], v.History[j] = v.History[j], v.History[i]
	}
	if len(v.History) > historyMax {
		v.History = v.History[:historyMax]
	}
	if a := status.Active; a != nil {
		rv := d.run(*a)
		rv.Auto = plan.autoRecord(status.Diff, a.Quest, a.Chunk)
		if a.State == Accepted || a.State == InProgress {
			rv.Queued = d.queued(a.State)
		}
		for _, row := range v.Quests {
			if row.ID != a.Quest {
				continue
			}
			for _, cr := range row.Chunks {
				if cr.ID == a.Chunk {
					rv.Reason = cr.Reason
				}
			}
		}
		switch {
		case a.State == Awaiting:
			v.Awaiting = &rv
		case a.State == Blocked:
			v.Blocked = &rv
		default:
			v.Running = &rv
		}
		v.Strip = &Strip{Quest: rv.Quest, QuestTitle: rv.QuestTitle, Chunk: rv.Chunk, Title: rv.Title, State: rv.State, Step: rv.Step, StepNo: rv.StepNo, StepCount: rv.StepCount, NPC: rv.NPC, Queued: rv.Queued}
	}
	v.WhileAway = d.awayCard(x)
	for _, r := range status.Extra {
		v.Extra = append(v.Extra, r.String()+" "+WaitsFor(Ref{status.Active.Quest, status.Active.Chunk}))
	}
	if last != nil && last.State == "running" {
		l := *last
		l.Why = d.handInWaitReason()
		last = &l
	}
	v.HandIn = last
	v.Needs = d.needs(v)
	v.Stopped = d.stopped(st, v)
	return v
}

func acceptedAt(p Plan, diff int, quest, chunk string) time.Time {
	for _, r := range p.Accepted {
		if r.Quest == quest && r.Chunk == chunk && r.Diff == Difficulties[diff] {
			return r.At
		}
	}
	return time.Time{}
}

func (d *Desk) offer(q Quest, c Chunk, cv ChunkView, auto, startable bool, class string, level int, paused bool, diff int, l Ledger) OfferView {
	risk := c.Risk
	if risk == "" {
		risk = q.Risk
	}
	o := OfferView{Quest: q.ID, QuestTitle: q.Title, Chunk: c.ID, Title: c.Title, Act: q.Act, Risk: risk,
		Readiness: cv.Reason, Reasons: []Reason{}, Minutes: c.Minutes, Startable: startable && !paused, Auto: auto, AutoOK: q.AutoStart == "allowed"}
	if class != "" {
		if f, ok, err := d.b.Floor(q, class); err == nil && ok {
			o.LegacyFloor = f
		}
	}
	if d.cfg.Explain != nil {
		if r, ok := d.cfg.Explain(q, c); ok {
			if r.Reasons != nil {
				o.Reasons = r.Reasons
			}
			o.Sentence = r.Sentence
			if r.LegacyFloor > 0 {
				o.LegacyFloor = r.LegacyFloor
			}
		}
	}
	if cv.Reason != ReadySafe && o.LegacyFloor > level && level > 0 {
		o.LevelsToGo = o.LegacyFloor - level
	}
	if l != nil {
		o.Pays = payFor(q, c, diff, l)
	}
	switch {
	case paused:
		o.Why = "The bot is paused."
	case !startable:
		o.Why = "Another chunk is running; finish or stop it first."
	}
	return o
}

func (d *Desk) run(a Active) RunView {
	rv := RunView{Quest: a.Quest, Chunk: a.Chunk, State: a.State, Step: a.Step, Why: a.Why, Failures: a.Failures, Deaths: a.Deaths}
	if c, err := d.b.Chunk(a.Quest, a.Chunk); err == nil && len(c.Steps) > 0 {
		cur := a.StepIdx
		switch {
		case a.State == Accepted:
			cur = 0 // nothing has run yet
		case a.State == Awaiting || cur < 0 || cur > len(c.Steps):
			cur = len(c.Steps) // every step is done: the hero waits at the NPC
		}
		rv.StepCount = len(c.Steps)
		rv.StepNo = cur + 1
		if rv.StepNo > rv.StepCount {
			rv.StepNo = rv.StepCount
		}
		for i, n := range c.Steps {
			st := "todo"
			switch {
			case i < cur:
				st = "done"
			case i == cur:
				st = "now"
			}
			rv.Steps = append(rv.Steps, StepRow{Name: n, State: st})
		}
	}
	if q, err := d.b.Quest(a.Quest); err == nil {
		rv.QuestTitle = q.Title
		if len(q.Chunks) > 1 {
			for i, c := range q.Chunks {
				if c.ID == a.Chunk {
					rv.PartNo, rv.PartCount = i+1, len(q.Chunks)
				}
			}
		}
		if c, err := d.b.Chunk(a.Quest, a.Chunk); err == nil {
			rv.Title = c.Title
			switch {
			case c.Giver != nil:
				rv.NPC = c.Giver.NPC
			case q.Accept != nil:
				rv.NPC = q.Accept.NPC
			}
		}
	}
	return rv
}

// needs are the Needs-you rows (design 4.6): the hand-in the hero waits at, a parked chunk. Offers are raised by
// the page once per offer (S5); the "paused: you were away" row belongs to S8.
func (d *Desk) needs(v View) []NeedRow {
	out := []NeedRow{}
	if a := v.Awaiting; a != nil {
		who := "The quest giver"
		if a.NPC != "" {
			who = a.NPC
		}
		out = append(out, NeedRow{ID: "quest:handin:" + a.Quest + "/" + a.Chunk, Kind: "quest_handin", Sev: "event",
			Text:    fmt.Sprintf("%s is waiting: hand in %s", who, a.QuestTitle),
			Actions: []Verb{{Label: "Hand in", Post: QuestRoute(a.Quest, "handin"), Body: map[string]any{"chunk": a.Chunk}}}})
	}
	if b := v.Blocked; b != nil {
		why := b.Why
		if why == "" {
			why = "the step failed"
		}
		body := map[string]any{"chunk": b.Chunk}
		retry := Verb{Label: "Try again", Post: QuestRoute(b.Quest, "retry"), Body: body}
		stop := Verb{Label: "Stop", Post: QuestRoute(b.Quest, "cancel"), Body: body}
		acts := []Verb{retry, stop}
		if b.Reason != nil { // B.1: the code decides the buttons, never the sentence
			switch b.Reason.Code {
			case "hold_timeout":
				retry.Label = "Keep waiting"
				acts = []Verb{retry, stop}
			case "prereq_missing", "quest_locked", "unsupported":
				acts = []Verb{stop}
			case "game_failed":
				acts = []Verb{}
			}
		}
		out = append(out, NeedRow{ID: "quest:blocked:" + b.Quest + "/" + b.Chunk, Kind: "quest_blocked", Sev: "warn",
			Text:    fmt.Sprintf("%s is stuck: %s", b.QuestTitle, why),
			Actions: acts})
	}
	if w := v.WhileAway; w != nil && len(w.Events) > 0 {
		text := "While you were away: " + strings.ToLower(w.Events[0].Text[:1]) + w.Events[0].Text[1:]
		if n := len(w.Events) - 1; n > 0 {
			text += fmt.Sprintf(", and %d more", n)
		}
		out = append(out, NeedRow{ID: "quest:away", Kind: "quest_away", Sev: "info", Text: text,
			Actions: []Verb{{Label: "Got it", Post: "/api/quests/away/dismiss"}}})
	}
	if c := v.AutoStart.Countdown; c != nil {
		out = append(out, NeedRow{ID: "quest:auto:" + c.Quest + "/" + c.Chunk, Kind: "quest_auto", Sev: "event",
			Text:    fmt.Sprintf("%s starts by itself in %d s", c.QuestTitle, c.SecondsLeft),
			Actions: []Verb{{Label: "Cancel", Post: QuestRoute(c.Quest, "cancel"), Body: map[string]any{"chunk": c.Chunk}}}})
	}
	// An offer is a Needs-you row only while it stands: it goes when the chunk starts, the offer changes or the bot pauses.
	if v.Strip == nil && !v.Paused && v.AutoStart.Countdown == nil {
		var best *OfferView
		for i := range v.Offers {
			o := &v.Offers[i]
			if !o.Startable {
				continue
			}
			if best == nil || (best.Readiness != ReadySafe && o.Readiness == ReadySafe) {
				best = o
			}
		}
		if best != nil {
			out = append(out, NeedRow{ID: "quest:offer:" + best.Quest + "/" + best.Chunk, Kind: "quest_offer", Sev: "event",
				Text: best.QuestTitle + " is ready: Start", Sub: string(best.Readiness),
				Actions: []Verb{{Label: "Start", Post: QuestRoute(best.Quest, "start"), Body: map[string]any{"chunk": best.Chunk}}}})
		}
	}
	if h := v.HandIn; h != nil && (h.State == "blocked" || h.State == "failed") && v.Blocked == nil {
		out = append(out, NeedRow{ID: "quest:handin-failed:" + h.Quest + "/" + h.Chunk, Kind: "quest_handin_failed", Sev: "warn",
			Text:    "The hand-in did not go through: " + h.Reason,
			Actions: []Verb{{Label: "Hand in", Post: QuestRoute(h.Quest, "handin"), Body: map[string]any{"chunk": h.Chunk}}}})
	}
	return out
}

// awayCard is the "while you were away" summary: nil when there is nothing to show.
func (d *Desk) awayCard(x *Exec) *AwayCard {
	evs, err := x.AwayEvents()
	if err != nil || len(evs) == 0 {
		return nil
	}
	card := &AwayCard{Events: []AwayEvent{}}
	for _, ev := range evs {
		t := d.title(ev.Quest)
		a := AwayEvent{At: ev.At, Quest: ev.Quest, QuestTitle: t, Chunk: ev.Chunk}
		switch ev.Kind {
		case EvAutoStart:
			a.Kind, a.Text = "started", "Started "+t+" by itself"
		case EvAutoHandIn:
			a.Kind, a.Text, a.Awarded = "handedIn", "Handed in "+t, ev.Awarded
			if ev.Note != "" {
				a.Text += " (" + ev.Note + ")"
			}
		default:
			a.Kind, a.Text = "handInFailed", "Could not hand in "+t
			if ev.Note != "" {
				a.Text += ": " + ev.Note
			}
		}
		card.Events = append(card.Events, a)
	}
	return card
}

// ---- actions ---------------------------------------------------------------------------------------------

// gate is the common refusal of every action: no hero, board off.
func (d *Desk) gate() (*Exec, *Store, *Reply) {
	x, st := d.exec()
	if x == nil || st == nil {
		r := refuse(http.StatusServiceUnavailable, CodeNoHero, "No hero is loaded yet; try again in a moment.")
		return nil, nil, &r
	}
	if !x.Enabled() {
		r := refuse(http.StatusConflict, CodeOff, "The quest board is off: set the knob quests.board to 1 to use it.")
		return nil, nil, &r
	}
	return x, st, nil
}

// pick finds the chunk an action means: the named one, else the quest's current chunk (the Ready one for Start,
// the accepted one otherwise).
func (d *Desk) pick(x *Exec, quest, chunk string, start bool) (string, State, *Reply) {
	if _, err := d.b.Quest(quest); err != nil {
		r := refuse(http.StatusNotFound, CodeUnknown, "There is no quest %q.", quest)
		return "", "", &r
	}
	if chunk != "" {
		if _, err := d.b.Chunk(quest, chunk); err != nil {
			r := refuse(http.StatusNotFound, CodeUnknown, "Quest %s has no part %q.", quest, chunk)
			return "", "", &r
		}
	}
	st, err := x.State()
	if err != nil {
		r := refuse(http.StatusInternalServerError, CodeInternal, "The board could not be read: %v", err)
		return "", "", &r
	}
	qv, ok := QuestView{}, false
	for _, v := range st.Views {
		if v.ID == quest {
			qv, ok = v, true
		}
	}
	if !ok {
		r := refuse(http.StatusConflict, CodeNotReady, "This quest does not exist in %s.", Difficulties[st.Diff])
		return "", "", &r
	}
	if chunk != "" {
		for _, c := range qv.Chunks {
			if c.ID == chunk {
				return chunk, c.State, nil
			}
		}
	}
	// the current chunk: the offered one for Start; else one that runs, waits or is parked
	order := func(want func(State) bool) (string, State, bool) {
		for _, c := range qv.Chunks {
			if want(c.State) {
				return c.ID, c.State, true
			}
		}
		return "", "", false
	}
	if start {
		if len(qv.Offered) > 0 {
			for _, c := range qv.Chunks {
				if c.ID == qv.Offered[0] {
					return c.ID, c.State, nil
				}
			}
		}
		if id, s, ok := order(func(s State) bool { return s == Accepted || s == InProgress || s == Awaiting }); ok {
			return id, s, nil
		}
	} else if id, s, ok := order(func(s State) bool { return s == Accepted || s == InProgress || s == Awaiting || s == Blocked }); ok {
		return id, s, nil
	}
	if len(qv.Chunks) > 0 {
		c := qv.Chunks[0]
		return c.ID, c.State, nil
	}
	r := refuse(http.StatusConflict, CodeNotReady, "%s has no part to start.", qv.Title)
	return "", "", &r
}

// changedRows reads the state and reason of the touched runs from a fresh view.
func (d *Desk) changedRows(refs []Ref) []ChangedRow {
	if len(refs) == 0 {
		return nil
	}
	v := d.View()
	var out []ChangedRow
	for _, r := range refs {
		row := ChangedRow{Quest: r.Quest, Chunk: r.Chunk}
		for _, q := range v.Quests {
			for _, c := range q.Chunks {
				if q.ID == r.Quest && c.ID == r.Chunk {
					row.State, row.Reason = c.State, c.Reason
				}
			}
		}
		out = append(out, row)
	}
	return out
}

func (d *Desk) title(quest string) string {
	if q, err := d.b.Quest(quest); err == nil {
		return q.Title
	}
	return quest
}

// Start accepts the current chunk of a quest (the player's Start tap; it never covers the next part).
func (d *Desk) Start(quest, chunk string) Reply {
	return d.noteRefusal(quest, chunk, d.start(quest, chunk))
}

func (d *Desk) start(quest, chunk string) Reply {
	x, st, bad := d.gate()
	if bad != nil {
		return *bad
	}
	chunk, state, bad := d.pick(x, quest, chunk, true)
	if bad != nil {
		return *bad
	}
	base := Reply{Quest: quest, Chunk: chunk}
	handInReady := x.SMOn() && state == Awaiting && !x.HasRecord(quest, chunk) // C5: the game counts the field work done
	switch {
	case state == Accepted, state == InProgress, state == Awaiting && !handInReady:
		base.OK, base.Already, base.State, base.Status = true, true, string(state), http.StatusOK
		base.Reason = d.title(quest) + " is already started."
		return base
	case state == Done:
		return withBase(base, refuse(http.StatusConflict, CodeDone, "%s is already done.", d.title(quest)))
	}
	if d.paused() {
		return withBase(base, refuse(http.StatusConflict, CodePaused, "The bot is paused; resume it before starting %s.", d.title(quest)))
	}
	if m, merr := st.Mode(); merr == nil {
		x.SetMode(m) // the record carries the mode the player picked, also after a restart
	}
	refs, err := x.AcceptRuns(quest, chunk)
	if err != nil {
		code, status := CodeNotReady, http.StatusConflict
		if s := err.Error(); strings.Contains(s, "already accepted") {
			code = CodeBusy
		}
		return withBase(base, refuse(status, code, "%s", cleanErr(err)))
	}
	d.invalidate()
	base.OK, base.State, base.Status = true, string(Accepted), http.StatusOK
	base.Reason = d.title(quest) + " started."
	if handInReady {
		base.State, base.Reason = string(Awaiting), d.title(quest)+": the hero goes to hand it in."
	}
	base.Changed = d.changedRows(refs)
	return base
}

// StartAnyway is the owner's try-anyway start (ADR 0030, amended 2026-11-24): the same as Start, but a chunk that reads
// Available because of a READINESS refusal (the board word, a level or life floor, the Andariel hold) is accepted for this
// one attempt. The caller has already checked the owner's auth and the confirm; the caller audits it with Reply.Override.
// Nothing but the readiness words is skipped: the kit gates and the combat reactions stay.
func (d *Desk) StartAnyway(quest, chunk string) Reply {
	return d.noteRefusal(quest, chunk, d.startAnyway(quest, chunk))
}

func (d *Desk) startAnyway(quest, chunk string) Reply {
	x, _, bad := d.gate()
	if bad != nil {
		return *bad
	}
	if chunk == "" { // the quest's first chunk that reads Available
		if st, err := x.State(); err == nil {
			for _, v := range st.Views {
				if v.ID != quest {
					continue
				}
				for _, c := range v.Chunks {
					if c.State == Available {
						chunk = c.ID
						break
					}
				}
			}
		}
	}
	if _, err := d.b.Quest(quest); err != nil {
		return refuse(http.StatusNotFound, CodeUnknown, "There is no quest %q.", quest)
	}
	if chunk == "" {
		return refuse(http.StatusConflict, CodeNotOverridable, "%s has no part that waits for readiness; there is nothing to try anyway.", d.title(quest))
	}
	if _, err := d.b.Chunk(quest, chunk); err != nil {
		return refuse(http.StatusNotFound, CodeUnknown, "Quest %s has no part %q.", quest, chunk)
	}
	base := Reply{Quest: quest, Chunk: chunk}
	if d.paused() {
		return withBase(base, refuse(http.StatusConflict, CodePaused, "The bot is paused; resume it before starting %s.", d.title(quest)))
	}
	refs, refused, err := d.tryAccept(x, quest, chunk)
	if err != nil {
		code, status := CodeNotReady, http.StatusConflict
		var no ErrNotOverridable
		switch s := err.Error(); {
		case errors.As(err, &no):
			code = CodeNotOverridable
		case strings.Contains(s, "already accepted"):
			code = CodeBusy
		}
		return withBase(base, refuse(status, code, "%s", cleanErr(err)))
	}
	d.invalidate()
	base.OK, base.State, base.Status = true, string(Accepted), http.StatusOK
	base.Override = refused
	base.Reason = d.title(quest) + " started anyway: " + refused + ". One attempt; the hero still drinks, flees and goes home."
	base.Changed = d.changedRows(refs)
	return base
}

func (d *Desk) tryAccept(x *Exec, quest, chunk string) ([]Ref, string, error) {
	refused, err := x.AcceptTry(quest, chunk)
	if err != nil {
		return nil, refused, err
	}
	x.mu.Lock()
	refs := x.lastChanged
	if x.boardLegacy() {
		refs = []Ref{{quest, chunk}}
	}
	x.mu.Unlock()
	return refs, refused, nil
}

// Cancel stops an accepted chunk and puts it back to ready. Nothing to stop is an `already` no-op.
func (d *Desk) Cancel(quest, chunk string) Reply {
	x, _, bad := d.gate()
	if bad != nil {
		return *bad
	}
	if x.CancelAuto(quest, chunk) {
		d.invalidate()
		return Reply{OK: true, Quest: quest, Chunk: chunk, Status: http.StatusOK, Reason: d.title(quest) + " will not start by itself this time; your auto-start setting stays."}
	}
	chunk, state, bad := d.pick(x, quest, chunk, false)
	if bad != nil {
		return *bad
	}
	base := Reply{Quest: quest, Chunk: chunk}
	switch state {
	case Done:
		return withBase(base, refuse(http.StatusConflict, CodeDone, "%s is already done; nothing to stop.", d.title(quest)))
	case Accepted, InProgress, Awaiting, Blocked:
	default:
		// A queued chunk (accepted, but its predecessor is not done) derives Locked; the player's Stop still withdraws it.
		if !x.HasRecord(quest, chunk) {
			base.OK, base.Already, base.State, base.Status = true, true, string(state), http.StatusOK
			base.Reason = d.title(quest) + " is not running."
			return base
		}
	}
	deps := x.Dependants(quest, chunk)
	refs, err := x.CancelRuns(quest, chunk)
	if err != nil {
		return withBase(base, refuse(http.StatusInternalServerError, CodeInternal, "Could not stop it: %v", err))
	}
	d.invalidate()
	base.OK, base.State, base.Status, base.Reason = true, string(Ready), http.StatusOK, d.title(quest)+" stopped."
	base.Changed = d.changedRows(refs)
	if len(deps) > 0 {
		var ts []string
		for _, id := range deps {
			if c, err := d.b.Chunk(quest, id); err == nil {
				ts = append(ts, c.Title)
			}
		}
		base.Reason += " The step that waited for it was stopped too: " + strings.Join(ts, ", ") + "."
	}
	return base
}

// Retry clears the park of a blocked chunk (the "Try again" button of a Needs-you row).
func (d *Desk) Retry(quest, chunk string) Reply {
	x, _, bad := d.gate()
	if bad != nil {
		return *bad
	}
	chunk, state, bad := d.pick(x, quest, chunk, false)
	if bad != nil {
		return *bad
	}
	base := Reply{Quest: quest, Chunk: chunk, State: string(state)}
	if state != Blocked {
		base.OK, base.Already, base.Status, base.Reason = true, true, http.StatusOK, d.title(quest)+" is not stuck."
		return base
	}
	refs := x.UnblockRuns(quest, chunk)
	d.invalidate()
	base.OK, base.Status, base.Reason = true, http.StatusOK, d.title(quest)+": trying again."
	base.Changed = d.changedRows(refs)
	return base
}

// HandIn is the player's hand-in tap. The dialogue runs on the bot loop and can take a while (a town trip, a
// death), so the reply is 202 and the outcome is in View.HandIn.
func (d *Desk) HandIn(quest, chunk string) Reply {
	x, _, bad := d.gate()
	if bad != nil {
		return *bad
	}
	chunk, state, bad := d.pick(x, quest, chunk, false)
	if bad != nil {
		return *bad
	}
	base := Reply{Quest: quest, Chunk: chunk, State: string(state)}
	k := ChunkKey(quest, chunk)
	d.mu.Lock()
	_, running := d.hand[k]
	d.mu.Unlock()
	switch {
	case running:
		base.OK, base.Already, base.Pending, base.Status, base.Reason = true, true, true, http.StatusAccepted, "The hand-in is already running."
		return base
	case state == Done:
		// The evidence says done: settle (a crash between the read and the ledger write pays here) and say so.
		base.OK, base.Status, base.Reason = true, http.StatusOK, d.title(quest)+" is already handed in."
		base.Awarded = x.settleChunk(quest, chunk)
		base.Already = len(base.Awarded) == 0
		return base
	case state != Awaiting:
		return withBase(base, refuse(http.StatusConflict, CodeNotAwait, "%s is not waiting for the hand-in%s.", d.title(quest), whyState(state)))
	case d.paused():
		return withBase(base, refuse(http.StatusConflict, CodePaused, "The bot is paused; resume it to hand in."))
	}
	d.mu.Lock()
	d.hand[k] = &handRun{started: d.cfg.Now()}
	d.last = &HandInView{Quest: quest, Chunk: chunk, State: "running"}
	d.cache = nil
	d.mu.Unlock()
	go d.runHandIn(x, quest, chunk, k)
	base.OK, base.Pending, base.Status, base.Reason = true, true, http.StatusAccepted, "Handing in "+d.title(quest)+"."
	return base
}

func (d *Desk) runHandIn(x *Exec, quest, chunk, k string) {
	ctx, cancel := context.WithTimeout(d.cfg.Ctx, handInTimeout)
	defer cancel()
	res, err := x.HandIn(ctx, quest, chunk)
	hv := HandInView{Quest: quest, Chunk: chunk, Attempt: res.Attempts, At: d.cfg.Now().UTC().Format(time.RFC3339)}
	switch {
	case err != nil:
		hv.State, hv.Reason = "failed", cleanErr(err)
	case res.Outcome == HandInDone:
		hv.State, hv.Awarded = "done", res.Awarded
	case res.Outcome == HandInConfirming:
		hv.State, hv.Reason = "confirming", "The reward was seen; waiting for the game to confirm it."
	default:
		hv.State, hv.Reason = "blocked", res.Why
	}
	d.mu.Lock()
	delete(d.hand, k)
	d.last, d.cache = &hv, nil
	d.mu.Unlock()
}

// SetAuto turns the opt-in for a quest on or off in the current difficulty (the page's auto switch). A quest whose data
// says it always needs the player refuses with the reason.
func (d *Desk) SetAuto(quest string, on bool) Reply {
	x, st, bad := d.gate()
	if bad != nil {
		return *bad
	}
	q, err := d.b.Quest(quest)
	if err != nil {
		return refuse(http.StatusNotFound, CodeUnknown, "There is no quest %q.", quest)
	}
	base := Reply{Quest: quest}
	if on && q.AutoStart != "allowed" {
		return withBase(base, refuse(http.StatusConflict, CodeAutoNever, "%s always needs you; it cannot start by itself.", q.Title))
	}
	stt, err := x.State()
	if err != nil {
		return withBase(base, refuse(http.StatusInternalServerError, CodeInternal, "The board could not be read: %v", err))
	}
	plan, _, perr := st.Load()
	if perr != nil {
		return withBase(base, refuse(http.StatusInternalServerError, CodeInternal, "Could not read the plan: %v", perr))
	}
	if plan.AutoStart(stt.Diff, quest) == on {
		base.OK, base.Already, base.Status = true, true, http.StatusOK
		base.Reason = autoWords(q.Title, on, true)
		return base
	}
	if err := st.SetAutoStart(quest, stt.Diff, on); err != nil {
		return withBase(base, refuse(http.StatusInternalServerError, CodeInternal, "Could not save it: %v", err))
	}
	if !on {
		x.CancelAuto(quest, "")
	}
	d.invalidate()
	base.OK, base.Status, base.Reason = true, http.StatusOK, autoWords(q.Title, on, false)
	return base
}

func autoWords(title string, on, already bool) string {
	state := "on"
	if !on {
		state = "off"
	}
	if already {
		return "Auto-start for " + title + " is already " + state + "."
	}
	if on {
		return "Auto-start for " + title + " is on: it starts by itself when it is safe."
	}
	return "Auto-start for " + title + " is off."
}

// DismissAway clears the "while you were away" card.
func (d *Desk) DismissAway() Reply {
	_, st, bad := d.gate()
	if bad != nil {
		return *bad
	}
	if err := st.DismissAway(d.cfg.Now()); err != nil {
		return refuse(http.StatusInternalServerError, CodeInternal, "Could not save it: %v", err)
	}
	d.invalidate()
	return Reply{OK: true, Status: http.StatusOK, Reason: "Cleared."}
}

// SetMode stores the between-chunks mode (train, idle, farm). The hero adopts it at the next safe moment (slice S6).
func (d *Desk) SetMode(mode string) Reply {
	x, st := d.exec()
	if x == nil || st == nil {
		return refuse(http.StatusServiceUnavailable, CodeNoHero, "No hero is loaded yet; try again in a moment.")
	}
	if !ValidMode(mode) {
		return refuse(http.StatusBadRequest, CodeBadMode, "Mode must be train, idle or farm, not %q.", mode)
	}
	cur, err := st.Mode()
	if err != nil {
		return refuse(http.StatusInternalServerError, CodeInternal, "Could not read the mode: %v", err)
	}
	if cur == mode {
		return Reply{OK: true, Already: true, Mode: mode, Status: http.StatusOK, Reason: "The mode is already " + mode + "."}
	}
	if err := st.SetMode(mode); err != nil {
		return refuse(http.StatusInternalServerError, CodeInternal, "Could not save the mode: %v", err)
	}
	x.SetMode(mode)
	d.invalidate()
	return Reply{OK: true, Mode: mode, Status: http.StatusOK, Reason: "Mode set to " + mode + "."}
}

func withBase(b, r Reply) Reply {
	r.Quest, r.Chunk = b.Quest, b.Chunk
	if b.State != "" {
		r.State = b.State
	}
	return r
}

func whyState(s State) string {
	if s == "" {
		return ""
	}
	return " (it is " + string(s) + ")"
}

func cleanErr(err error) string {
	s := err.Error()
	return strings.TrimPrefix(s, "questboard: ")
}

// queued is the reason an accepted chunk is not running: the bot's own words, or (for a chunk that has not started) a
// generic line, so the strip never shows a Start the hero seems to ignore.
func (d *Desk) queued(st State) string {
	if d.cfg.Queued != nil {
		if q := d.cfg.Queued(); q != "" {
			return q
		}
	}
	if st == Accepted {
		return "Queued: starts at the hero's next free moment."
	}
	return ""
}
