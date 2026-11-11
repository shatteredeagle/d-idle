package campaign

import (
	"fmt"
	"slices"
	"sync"
	"time"
)

// StateFile is the per-hero state file in the bot's state directory.
const StateFile = "campaign.json"

// Controller owns the persisted per-hero state and the last decision. The bot's planner reads State and reports every
// decision with Decided; the watch screen reads View and changes intents with Apply. Safe for concurrent use.
type Controller struct {
	mu    sync.Mutex
	spine *Spine
	path  string // "" = in memory
	store *Store
	hero  func() string
	now   func() time.Time

	// AskWait and FarmBudget feed Opts (docs/campaign.md section 4).
	AskWait, FarmBudget time.Duration
	// OnDone is called (outside the lock) with chapters first seen done: the Chronicle line and the unlock ladder.
	OnDone func(hero string, ids []string)
	// DefaultGoAt is the policy of a hero that never chose one (knob campaign.goAtFloor; "" = recommended).
	DefaultGoAt GoAt
	// OnError gets a failed save (nil: dropped; the next change saves again).
	OnError func(error)
	// Pace gives the level meter of a farm step: share of the way from the current level to the target and the
	// time left at the hero's pace (ok false: unknown).
	Pace func(target int) (pct float64, eta time.Duration, ok bool)

	last     Decision
	lastProg Progress
	lastOpts Opts
	lastAt   time.Time
	seen     bool
}

// NewController loads path (missing = empty). hero names the current hero ("" = none yet: the default state).
func NewController(s *Spine, path string, hero func() string, now func() time.Time) (*Controller, error) {
	st := &Store{Schema: StateSchema}
	if path != "" {
		var err error
		if st, err = LoadStore(path); err != nil {
			return nil, err
		}
	}
	return &Controller{spine: s, path: path, store: st, hero: hero, now: now}, nil
}

// Spine is the spine the controller plans with.
func (c *Controller) Spine() *Spine { return c.spine }

// State is a copy of the current hero's state.
func (c *Controller) State() *HeroState {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.store.Hero(c.hero()).clone()
	if h.GoAt == "" {
		h.GoAt = c.DefaultGoAt
	}
	return h
}

// Unlocked reports whether a done chapter of the current hero opens feature (a chapter's unlocks list).
func (c *Controller) Unlocked(feature string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.store.Hero(c.hero())
	for i := range c.spine.Chapters {
		ch := &c.spine.Chapters[i]
		if _, done := h.Done[ch.ID]; done && slices.Contains(ch.Unlocks, feature) {
			return true
		}
	}
	return false
}

// Opts are the decision options at the controller's clock.
func (c *Controller) Opts() Opts {
	return Opts{Now: c.now(), AskWait: c.AskWait, FarmBudget: c.FarmBudget}
}

// Decided records a decision and the inputs it was made from: Observe on the persisted state (done chapters, the farm
// clock, the question time), a save when it changed, then OnDone.
func (c *Controller) Decided(d Decision, p Progress, o Opts) {
	c.mu.Lock()
	name := c.hero()
	h := c.store.Hero(name)
	before := h.fingerprint()
	done := c.spine.Observe(p, h, d, o.Now)
	c.last, c.lastProg, c.lastOpts, c.lastAt, c.seen = d, p, o, o.Now, true
	var err error
	if c.path != "" && h.fingerprint() != before {
		err = c.store.Save(c.path)
	}
	cb, eb := c.OnDone, c.OnError
	c.mu.Unlock()
	if err != nil && eb != nil {
		eb(fmt.Errorf("campaign: save: %w", err))
	}
	if cb != nil && len(done) > 0 {
		cb(name, done)
	}
}

// Request is one player change from the watch screen. Exactly one of the fields is used per call.
type Request struct {
	Chapter string `json:"chapter,omitempty"`
	Intent  Intent `json:"intent,omitempty"` // with Chapter
	GoNow   *bool  `json:"goNow,omitempty"`  // with Chapter: true sends the hero now, false cancels
	Answer  *bool  `json:"answer,omitempty"` // with Chapter: the reply to the floor question
	GoAt    GoAt   `json:"goAt,omitempty"`
	Mode    Mode   `json:"mode,omitempty"`
}

// Apply validates and applies r to the current hero and saves.
func (c *Controller) Apply(r Request) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	h := c.store.Hero(c.hero())
	var err error
	switch {
	case r.Mode != "":
		err = h.SetMode(r.Mode)
	case r.GoAt != "":
		err = h.SetGoAt(r.GoAt)
	case r.Chapter == "":
		return fmt.Errorf("campaign: request without chapter, mode or goAt")
	case c.spine.Chapter(r.Chapter) == nil:
		return fmt.Errorf("campaign: unknown chapter %q", r.Chapter)
	case r.Intent != "":
		err = h.SetIntent(c.spine, r.Chapter, r.Intent, c.now())
	case r.GoNow != nil && *r.GoNow:
		err = h.SetGoNow(c.spine, r.Chapter)
	case r.GoNow != nil:
		if h.GoNow == r.Chapter {
			h.GoNow = ""
		}
	case r.Answer != nil:
		h.SetAnswer(r.Chapter, *r.Answer, c.lastProg.Level)
	default:
		return fmt.Errorf("campaign: request for %q changes nothing", r.Chapter)
	}
	if err != nil || c.path == "" {
		return err
	}
	return c.store.Save(c.path)
}

func (h *HeroState) clone() *HeroState {
	n := *h
	n.Intents = cloneMap(h.Intents)
	n.Asked = cloneMap(h.Asked)
	n.Answers = cloneMap(h.Answers)
	n.Done = cloneMap(h.Done)
	n.Skipped = cloneMap(h.Skipped)
	if h.Farm != nil {
		f := *h.Farm
		n.Farm = &f
	}
	return &n
}

func cloneMap[K comparable, V any](m map[K]V) map[K]V {
	if m == nil {
		return nil
	}
	out := make(map[K]V, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// fingerprint is a cheap change test for the save after Observe.
func (h *HeroState) fingerprint() string {
	f := ""
	if h.Farm != nil {
		f = h.Farm.Chapter + h.Farm.Since.String()
	}
	return fmt.Sprint(len(h.Done), len(h.Asked), h.GoNow, f)
}
