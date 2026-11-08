package pact

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

// The bot side of the Pacts (TASK-167, ADR 0019, docs/pact-botside.md): a Controller turns the chosen Pacts into
// policy-sheet options and tuner knobs, and a death limit into a pause. It acts only while a Pact is "live": the hero
// is out of the safe zones AND a human is connected or the player gave that Pact a typed, dated away-consent. Pacts are
// OFF in gates and measured runs (Off). Nothing here writes game memory.

// Off reports that Pacts are switched off for this process: D2BOT_PACTS=off, or a measured run (D2BOT_RUN is set by the
// runstart protocol of gates, soaks and baselines, AGENTS.md CRUTCH RULE). Callers build no Controller when it is true.
func Off(getenv func(string) string) bool {
	return strings.EqualFold(getenv("D2BOT_PACTS"), "off") || getenv("D2BOT_RUN") != ""
}

// KnobSetter is the tuner's edit (bot.Tuner.Edit through the sheet's KnobSetter): nil resets a knob to its profile value.
type KnobSetter func(over map[string]*float64) error

// Sheet is the policy sheet as the controller needs it.
type Sheet interface {
	Current(row string) string // the option now chosen
	Default(row string) string
	Set(row, option string) error
}

// Consent is the away-consent record of one Pact: typed by the player, dated, tied to a rank and a difficulty.
type Consent struct {
	Pact       string    `json:"pact"`
	Rank       int       `json:"rank"`
	Difficulty int       `json:"difficulty"`
	At         time.Time `json:"at"`
	Typed      string    `json:"typed"`
}

// State is what survives a restart (one JSON file, 0600). The death count and a pause do not: a new process is a new
// expedition (U).
type State struct {
	Picks     []Pick    `json:"picks"`
	Consents  []Consent `json:"consents"`
	Masteries int       `json:"masteries"`
}

// Config wires a Controller. Cat and Class are required; Path "" keeps the state in memory.
type Config struct {
	Cat     *Catalogue
	Class   string // lower case
	Path    string
	Knobs   KnobSetter
	Sheet   Sheet
	Pause   func(reason string) error // hold the hero in town and wait for the player
	Present func() bool               // a human is connected now; nil = nobody (consent only)
	Now     func() time.Time
}

// Where is the hero now, for the safe zones.
type Where struct {
	Level      int
	Difficulty int
	InTown     bool
	InDen      bool          // in the Den of Evil with its quest not done (U: the caller's approximation)
	Age        time.Duration // time since the hero was created or first seen
}

// Line is one effect line for the policy-changes return section: what the Pact does and whether it acts right now.
type Line struct {
	Pact   string `json:"pact"`
	Rank   int    `json:"rank"`
	Text   string `json:"text"`
	Active bool   `json:"active"`
	Status string `json:"status"` // "active", or why not: "safe zone", "nobody watching and no away-consent", "paused: ..."
}

// Controller applies the chosen Pacts. Safe for concurrent use.
type Controller struct {
	cfg Config

	mu      sync.Mutex
	st      State
	knobs   map[string]float64 // knobs this controller has set
	rows    map[string]string  // row -> the option it had before a Pact took it
	deaths  int
	paused  string // reason, "" = not paused by a Pact
	lastWho Where
}

// New loads the state file (a missing file is an empty state; a corrupt one is an error).
func New(cfg Config) (*Controller, error) {
	if cfg.Cat == nil || cfg.Class == "" {
		return nil, errors.New("pact: controller needs a catalogue and the hero's class")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	c := &Controller{cfg: cfg, knobs: map[string]float64{}, rows: map[string]string{}}
	if cfg.Path != "" {
		b, err := os.ReadFile(cfg.Path)
		switch {
		case err == nil:
			if err := json.Unmarshal(b, &c.st); err != nil {
				return nil, fmt.Errorf("pact: state %s: %w", cfg.Path, err)
			}
			if err := cfg.Cat.Check(c.st.Picks, cfg.Class, cfg.Cat.Limits.SlotsAtUnlock, c.st.Masteries); err != nil {
				return nil, fmt.Errorf("pact: state %s: %w", cfg.Path, err)
			}
		case !errors.Is(err, os.ErrNotExist):
			return nil, err
		}
	}
	return c, nil
}

func (c *Controller) save() error {
	if c.cfg.Path == "" {
		return nil
	}
	b, err := json.MarshalIndent(c.st, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(c.cfg.Path, b, 0o600)
}

// Select replaces the chosen set after the catalogue rules (Check). A consent of a Pact that left the set or changed
// rank ends. The effects follow at the next Evaluate.
func (c *Controller) Select(picks []Pick) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.cfg.Cat.Check(picks, c.cfg.Class, c.cfg.Cat.Limits.SlotsAtUnlock, c.st.Masteries); err != nil {
		return err
	}
	old := c.st
	c.st.Picks = append([]Pick(nil), picks...)
	var keep []Consent
	for _, co := range c.st.Consents {
		for _, p := range picks {
			if p.ID == co.Pact && p.Rank == co.Rank {
				keep = append(keep, co)
			}
		}
	}
	c.st.Consents = keep
	if err := c.save(); err != nil {
		c.st = old
		return err
	}
	return nil
}

// Grant records the player's away-consent for one chosen Pact. typed must be the Pact's name (case and spaces aside):
// the player writes it out, a click is not enough. It is dated now and ends after Limits.ConsentDays, when the Pact's
// rank or the difficulty changes, or at Revoke.
func (c *Controller) Grant(pactID, typed string, difficulty int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	p, err := c.cfg.Cat.Get(pactID)
	if err != nil {
		return err
	}
	rank := 0
	for _, pk := range c.st.Picks {
		if pk.ID == pactID {
			rank = pk.Rank
		}
	}
	if rank == 0 {
		return fmt.Errorf("pact: %q is not chosen, so it cannot be given an away-consent", pactID)
	}
	if !strings.EqualFold(strings.Join(strings.Fields(typed), " "), p.Name) {
		return fmt.Errorf("pact: away-consent for %q needs its name typed out (%q)", pactID, p.Name)
	}
	var out []Consent
	for _, co := range c.st.Consents {
		if co.Pact != pactID {
			out = append(out, co)
		}
	}
	old := c.st
	c.st.Consents = append(out, Consent{Pact: pactID, Rank: rank, Difficulty: difficulty, At: c.cfg.Now(), Typed: typed})
	if err := c.save(); err != nil {
		c.st = old
		return err
	}
	return nil
}

// Revoke ends the away-consent of a Pact.
func (c *Controller) Revoke(pactID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	old := c.st
	var out []Consent
	for _, co := range c.st.Consents {
		if co.Pact != pactID {
			out = append(out, co)
		}
	}
	c.st.Consents = out
	if err := c.save(); err != nil {
		c.st = old
		return err
	}
	return nil
}

func (c *Controller) consented(id string, rank, diff int) bool {
	ttl := time.Duration(c.cfg.Cat.Limits.ConsentDays) * 24 * time.Hour
	for _, co := range c.st.Consents {
		if co.Pact == id && co.Rank == rank && co.Difficulty == diff && c.cfg.Now().Sub(co.At) < ttl {
			return true
		}
	}
	return false
}

// safeZone says why Pacts stay off here ("" = they may act): town, the first minutes and levels, the Den.
func (c *Controller) safeZone(w Where) string {
	l := c.cfg.Cat.Limits
	switch {
	case w.InTown:
		return "town is a safe zone"
	case w.InDen:
		return "the Den of Evil is a safe zone until its quest is done"
	case w.Age < time.Duration(l.SafeMinutes)*time.Minute || w.Level < l.SafeClvl:
		return fmt.Sprintf("safe until %d minutes and clvl %d", l.SafeMinutes, l.SafeClvl)
	}
	return ""
}

// live says whether a chosen Pact acts now, and if not why.
func (c *Controller) live(p Pick, w Where) (bool, string) {
	if z := c.safeZone(w); z != "" {
		return false, z
	}
	if c.paused != "" {
		return false, "paused: " + c.paused
	}
	if c.cfg.Present != nil && c.cfg.Present() || c.consented(p.ID, p.Rank, w.Difficulty) {
		return true, "active"
	}
	return false, "nobody watching and no away-consent"
}

type want struct {
	knobs map[string]float64
	rows  map[string]string
}

// Evaluate brings the tuner and the sheet to what the live Pacts ask for, and back to what they were for the rest. It
// changes only the difference, so calling it every tick is cheap. An error from the tuner or the sheet leaves the old
// state and is returned (the next call tries again).
func (c *Controller) Evaluate(w Where) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastWho = w
	wt := want{knobs: map[string]float64{}, rows: map[string]string{}}
	for _, pk := range c.st.Picks {
		on, _ := c.live(pk, w)
		if !on {
			continue
		}
		p, err := c.cfg.Cat.Get(pk.ID)
		if err != nil {
			return err
		}
		r, err := p.Rank(pk.Rank)
		if err != nil {
			return err
		}
		if r.Params.UntilClvl > 0 && w.Level >= r.Params.UntilClvl {
			continue
		}
		for k, v := range r.Params.Knobs {
			wt.knobs[k] = v
		}
		if r.Params.Row != "" {
			wt.rows[r.Params.Row] = r.Params.Option
		}
	}
	return c.apply(wt)
}

func (c *Controller) apply(wt want) error {
	over := map[string]*float64{}
	for k := range c.knobs {
		if _, keep := wt.knobs[k]; !keep {
			over[k] = nil
		}
	}
	for k, v := range wt.knobs {
		if old, ok := c.knobs[k]; !ok || old != v {
			v := v
			over[k] = &v
		}
	}
	if len(over) > 0 {
		if c.cfg.Knobs == nil {
			return errors.New("pact: no tuner is connected, so a Pact cannot set knobs")
		}
		if err := c.cfg.Knobs(over); err != nil {
			return fmt.Errorf("pact: knobs: %w", err)
		}
		for k, v := range over {
			if v == nil {
				delete(c.knobs, k)
			} else {
				c.knobs[k] = *v
			}
		}
	}
	rows := make([]string, 0, len(wt.rows)+len(c.rows))
	for r := range wt.rows {
		rows = append(rows, r)
	}
	for r := range c.rows {
		if _, ok := wt.rows[r]; !ok {
			rows = append(rows, r)
		}
	}
	sort.Strings(rows)
	for _, r := range rows {
		if c.cfg.Sheet == nil {
			return errors.New("pact: no policy sheet is connected, so a Pact cannot set a row")
		}
		if opt, ok := wt.rows[r]; ok {
			if _, had := c.rows[r]; !had {
				c.rows[r] = c.cfg.Sheet.Current(r)
			}
			if c.cfg.Sheet.Current(r) != opt {
				if err := c.cfg.Sheet.Set(r, opt); err != nil {
					return fmt.Errorf("pact: row %s: %w", r, err)
				}
			}
			continue
		}
		back := c.rows[r]
		if back == "" {
			back = c.cfg.Sheet.Default(r)
		}
		if err := c.cfg.Sheet.Set(r, back); err != nil {
			return fmt.Errorf("pact: row %s back to %s: %w", r, back, err)
		}
		delete(c.rows, r)
	}
	return nil
}

// OnDeath counts a death while a Pact with a death limit is live; at the limit the hero is paused once and waits for
// the player (Resume). A death in a safe zone, or with nobody watching and no consent, is not counted.
func (c *Controller) OnDeath() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.paused != "" {
		return nil
	}
	// The death sends the hero to town, so the zone test uses the place the hero died in (the last Evaluate).
	w := c.lastWho
	w.InTown = false
	for _, pk := range c.st.Picks {
		p, err := c.cfg.Cat.Get(pk.ID)
		if err != nil {
			return err
		}
		r, err := p.Rank(pk.Rank)
		if err != nil {
			return err
		}
		if r.Params.DeathLimit == 0 {
			continue
		}
		if on, _ := c.live(pk, w); !on {
			continue
		}
		c.deaths++
		if c.deaths < r.Params.DeathLimit {
			continue
		}
		reason := fmt.Sprintf("%s: %d deaths, the limit is %d", p.Name, c.deaths, r.Params.DeathLimit)
		if c.cfg.Pause == nil {
			return errors.New("pact: " + reason + ", but nothing can pause the hero")
		}
		if err := c.cfg.Pause(reason); err != nil {
			return fmt.Errorf("pact: pause: %w", err)
		}
		c.paused = reason
		return nil
	}
	return nil
}

// Resume is the player taking the hero back: the death count starts again.
func (c *Controller) Resume() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.deaths, c.paused = 0, ""
}

// Paused is the reason a Pact paused the hero, "" if none.
func (c *Controller) Paused() string {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.paused
}

// Lines are the effect lines of the chosen Pacts as of the last Evaluate, from the catalogue text.
func (c *Controller) Lines() []Line {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []Line
	for _, pk := range c.st.Picks {
		p, err := c.cfg.Cat.Get(pk.ID)
		if err != nil {
			continue
		}
		r, err := p.Rank(pk.Rank)
		if err != nil {
			continue
		}
		on, why := c.live(pk, c.lastWho)
		out = append(out, Line{Pact: p.Name, Rank: pk.Rank, Text: r.EffectLine, Active: on, Status: why})
	}
	return out
}

// Picks returns the chosen set.
func (c *Controller) Picks() []Pick {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]Pick(nil), c.st.Picks...)
}
