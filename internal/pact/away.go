package pact

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// Away safety (TASK-169, docs/pact-away.md, design note section 5): who counts as a present human, the low-life floor,
// the plain-language offer text and the separate Hardcore flag. Nothing here writes game memory.

const (
	defaultPresenceMin = 5
	defaultLifeFloor   = 35
	lifeResumeMargin   = 10 // a paused Pact resumes this many percent above the floor, so it does not flap
)

// PresenceMin is the minutes a look at the viewer counts as a human being there (limits.presence_min, default 5).
func (l Limits) PresenceMinutes() int {
	if l.PresenceMin > 0 {
		return l.PresenceMin
	}
	return defaultPresenceMin
}

// LifeFloor is the life percent under which the Pacts pause (limits.life_floor_pct, default 35).
func (l Limits) LifeFloor() int {
	if l.LifeFloorPct > 0 {
		return l.LifeFloorPct
	}
	return defaultLifeFloor
}

// Presence says whether a human is connected: any viewer request within the window, or a video stream still open.
// The watch server calls Touch for every authenticated viewer GET.
type Presence struct {
	mu     sync.Mutex
	last   time.Time
	window time.Duration
	now    func() time.Time
	Open   func() int // open video streams; nil = none known
}

// NewPresence makes a tracker; window <= 0 takes the default (5 minutes), a nil now is time.Now.
func NewPresence(window time.Duration, now func() time.Time) *Presence {
	if window <= 0 {
		window = defaultPresenceMin * time.Minute
	}
	if now == nil {
		now = time.Now
	}
	return &Presence{window: window, now: now}
}

// SetWindow changes the window (the catalogue's presence_min, known only after it is loaded).
func (p *Presence) SetWindow(d time.Duration) {
	if d > 0 {
		p.mu.Lock()
		p.window = d
		p.mu.Unlock()
	}
}

// Touch records a viewer request now.
func (p *Presence) Touch() {
	p.mu.Lock()
	p.last = p.now()
	p.mu.Unlock()
}

// Present is true while a stream is open or the last request is inside the window.
func (p *Presence) Present() bool {
	if p.Open != nil && p.Open() > 0 {
		return true
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.last.IsZero() && p.now().Sub(p.last) < p.window
}

// checkLife moves the low-life state and writes the journal line on each change. Unknown life (MaxLife 0) clears it.
func (c *Controller) checkLife(w Where) {
	floor := c.cfg.Cat.Limits.LifeFloor()
	was := c.lowLife
	switch {
	case w.MaxLife <= 0:
		c.lowLife = false
	case w.Life*100 < floor*w.MaxLife:
		c.lowLife = true
	case w.Life*100 >= min(floor+lifeResumeMargin, 100)*w.MaxLife:
		c.lowLife = false
	}
	if c.lowLife != was && c.cfg.Journal != nil {
		if c.lowLife {
			c.cfg.Journal(fmt.Sprintf("Pacts paused: life under %d%% (%d of %d)", floor, w.Life, w.MaxLife))
		} else {
			c.cfg.Journal("Pacts resume: life is back up")
		}
	}
}

// OfferLine is one chosen Pact in the plain-language offer.
type OfferLine struct {
	Pact       string   `json:"pact"`
	Name       string   `json:"name"`
	Rank       int      `json:"rank"`
	Effect     string   `json:"effect"`
	CanLose    []string `json:"canLose"`
	Consent    string   `json:"consent"` // "none", or "until <time>" for a valid one
	TypeToGive string   `json:"typeToGive"`
}

// Offer is what GET /api/pacts/offer returns: what the chosen Pacts can cost while the player is away, with numbers
// from the catalogue and the game's difficulty table, never typed text.
type Offer struct {
	Difficulty int         `json:"difficulty"`
	Lines      []OfferLine `json:"lines"`
	Warnings   []string    `json:"warnings"`
	Rules      []string    `json:"rules"`
	MaxHours   int         `json:"maxAwayHours"`
	Nothing    string      `json:"nothing,omitempty"`
}

// Nightmare and Hell entry numbers (V: DifficultyLevels in docs/probes/pactnative.md).
var difficultyWarn = map[int]string{
	1: "Nightmare: your resistances are 40 lower and a death costs 5% of your experience.",
	2: "Hell: your resistances are 100 lower and a death costs 10% of your experience.",
}

// Offer builds the offer for the hero's difficulty (0 normal, 1 nightmare, 2 hell).
func (c *Controller) Offer(difficulty int) Offer {
	c.mu.Lock()
	defer c.mu.Unlock()
	l := c.cfg.Cat.Limits
	o := Offer{Difficulty: difficulty, MaxHours: l.ConsentDays * 24, Warnings: []string{}, Lines: []OfferLine{}}
	if w := difficultyWarn[difficulty]; w != "" {
		o.Warnings = append(o.Warnings, w, "A consent given on another difficulty does not count here: it is asked for again after the move.")
	}
	o.Rules = []string{
		fmt.Sprintf("A Pact acts only while you are watching (a viewer open in the last %d minutes) or after you type its name below.", l.PresenceMinutes()),
		fmt.Sprintf("Every Pact stops by itself when life is under %d%%, in town, and for the first %d minutes and below level %d.", l.LifeFloor(), l.SafeMinutes, l.SafeClvl),
		fmt.Sprintf("An away-consent ends after at most %d hours, when you change the Pact's rank, or when the difficulty changes.", l.ConsentDays*24),
	}
	for _, pk := range c.st.Picks {
		p, err := c.cfg.Cat.Get(pk.ID)
		if err != nil {
			continue
		}
		r, err := p.Rank(pk.Rank)
		if err != nil {
			continue
		}
		ln := OfferLine{Pact: p.ID, Name: p.Name, Rank: pk.Rank, Effect: r.EffectLine, Consent: "none", TypeToGive: p.Name}
		ln.CanLose = loseText(p, r)
		for _, co := range c.st.Consents {
			if co.Pact == p.ID && c.consentedAt(co, pk.Rank, difficulty) {
				ln.Consent = "until " + co.End(l.ConsentDays).UTC().Format("2006-01-02 15:04 UTC")
			}
		}
		o.Lines = append(o.Lines, ln)
	}
	if len(o.Lines) == 0 {
		o.Nothing = "No Pact is chosen, so there is nothing to consent to."
	}
	return o
}

// End is when the consent stops (days is the catalogue's consent_days).
func (co Consent) End(days int) time.Time {
	d := time.Duration(days) * 24 * time.Hour
	if co.Hours > 0 {
		d = min(d, time.Duration(co.Hours)*time.Hour)
	}
	return co.At.Add(d)
}

func (c *Controller) consentedAt(co Consent, rank, diff int) bool {
	return co.Rank == rank && co.Difficulty == diff && c.cfg.Now().Before(co.End(c.cfg.Cat.Limits.ConsentDays))
}

// loseText says in numbers what the rank can cost, from its params (nothing is invented: a cost without a number in
// the data is said without one).
func loseText(p *Pact, r *Rank) []string {
	var out []string
	pr := r.Params
	if pr.DeathLimit > 0 {
		out = append(out, fmt.Sprintf("Up to %d deaths before the hero stops and waits for you; each death costs the game's own penalty (gold or items on the corpse until it is fetched).", pr.DeathLimit))
	}
	if pr.UntilClvl > 0 && pr.UntilClvl < 99 {
		out = append(out, fmt.Sprintf("Applies until level %d.", pr.UntilClvl))
	} else if pr.UntilClvl >= 99 {
		out = append(out, "Applies for the whole run (it is never lifted by level).")
	}
	if len(pr.Knobs) > 0 || pr.Row != "" {
		out = append(out, "The hero plays with fewer safety nets: "+r.EffectLine)
	}
	if len(out) == 0 {
		out = append(out, "Nothing is lost beyond the effect above.")
	}
	return out
}

// ---- Hardcore --------------------------------------------------------------------------------------------------

// ErrHardcoreInactive is the refusal of the stub: the bot cannot create or play a hardcore hero yet.
var ErrHardcoreInactive = errors.New("hardcore is not active yet: the bot cannot create or play a hardcore hero, so nothing is switched on")

// HardcoreState is the separate Hardcore mode flag (decision F12): its own file, its own typed confirmation (the hero's
// name), never on the Pact sheet and never heat-costed. Away play with Hardcore is a second consent, off by default (U).
type HardcoreState struct {
	Confirmed   bool      `json:"confirmed"`
	Hero        string    `json:"hero,omitempty"`
	At          time.Time `json:"at,omitempty"`
	AwayAllowed bool      `json:"awayAllowed"`
	Active      bool      `json:"active"` // always false until the bot can play a hardcore hero
	Note        string    `json:"note"`
}

// Hardcore stores the flag.
type Hardcore struct {
	mu   sync.Mutex
	path string
	now  func() time.Time
	st   HardcoreState
}

// OpenHardcore loads the flag (a missing file is "not confirmed"; path "" keeps it in memory).
func OpenHardcore(path string, now func() time.Time) (*Hardcore, error) {
	if now == nil {
		now = time.Now
	}
	h := &Hardcore{path: path, now: now}
	if path != "" {
		b, err := os.ReadFile(path)
		switch {
		case err == nil:
			if err := json.Unmarshal(b, &h.st); err != nil {
				return nil, fmt.Errorf("pact: hardcore flag %s: %w", path, err)
			}
		case !errors.Is(err, os.ErrNotExist):
			return nil, err
		}
	}
	h.st.Active = false
	return h, nil
}

// State is the flag as data, with the plain "not active yet" note.
func (h *Hardcore) State() HardcoreState {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := h.st
	s.Active = false
	s.Note = "Hardcore is a separate mode with permanent death. It is not active yet: the bot cannot create or play a hardcore hero. A confirmation is only remembered."
	return s
}

func (h *Hardcore) save() error {
	if h.path == "" {
		return nil
	}
	b, err := json.MarshalIndent(h.st, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(h.path, b, 0o600)
}

// Confirm records the confirmation. typed must be the hero's name exactly (case aside): a click or a Pact name is not
// enough. The away allowance stays off.
func (h *Hardcore) Confirm(hero, typed string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if strings.TrimSpace(hero) == "" {
		return errors.New("pact: hardcore confirmation needs the hero's name, and the hero is not known yet")
	}
	if !strings.EqualFold(strings.TrimSpace(typed), strings.TrimSpace(hero)) {
		return fmt.Errorf("pact: hardcore needs the hero's name typed out (%q)", hero)
	}
	old := h.st
	h.st = HardcoreState{Confirmed: true, Hero: hero, At: h.now()}
	if err := h.save(); err != nil {
		h.st = old
		return err
	}
	return nil
}

// AllowAway turns the away allowance on or off; it needs a confirmation first.
func (h *Hardcore) AllowAway(on bool) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if on && !h.st.Confirmed {
		return errors.New("pact: confirm hardcore first")
	}
	old := h.st
	h.st.AwayAllowed = on
	if err := h.save(); err != nil {
		h.st = old
		return err
	}
	return nil
}

// Revoke drops the confirmation and the away allowance.
func (h *Hardcore) Revoke() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	old := h.st
	h.st = HardcoreState{}
	if err := h.save(); err != nil {
		h.st = old
		return err
	}
	return nil
}

// Activate is the stub of the switch: it always refuses (ErrHardcoreInactive), confirmed or not.
func (h *Hardcore) Activate() error { return ErrHardcoreInactive }
