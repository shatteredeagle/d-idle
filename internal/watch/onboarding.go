package watch

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	onboardingdata "d2lod-bot/data/onboarding"
)

// The onboarding routes (docs/onboarding.md): GET /api/onboarding gives the texts (data/onboarding/en.json), what this
// hero has dismissed or ticked, the checklist steps that ticked themselves, and the player's absence; POST on the answer
// listener (the same gates as the other answer routes) records a dismissal, a collapse, a "show again" or a manual tick.
// The state is one small file keyed by hero name, so a second hero sees the welcome card again.

const (
	onboardingSchema      = "watch.onboarding/1"
	onboardingStateSchema = "watch.onboarding.state/1"
	// OnboardingRoute is the viewer GET route; the POST goes to the same path on the answer listener.
	OnboardingRoute = "/api/onboarding"
)

// OnboardingRoutes are the viewer GET routes.
var OnboardingRoutes = []string{OnboardingRoute}

// OnboardingTexts is data/onboarding/en.json, sent to the panel as it is so the words can be edited without code.
type OnboardingTexts struct {
	Schema  string           `json:"schema"`
	Welcome json.RawMessage  `json:"welcome"`
	Help    json.RawMessage  `json:"help"`
	Check   json.RawMessage  `json:"checklist"`
	Steps   []OnboardingStep `json:"steps"`
	Recon   json.RawMessage  `json:"reconnect"`
	Art     json.RawMessage  `json:"art"`
}

// OnboardingStep is one checklist line.
type OnboardingStep struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Hint  string `json:"hint,omitempty"`
	Href  string `json:"href,omitempty"`
}

// LoadOnboardingTexts parses the embedded texts and checks the step ids.
func LoadOnboardingTexts(b []byte) (OnboardingTexts, error) {
	var t OnboardingTexts
	if err := json.Unmarshal(b, &t); err != nil {
		return t, fmt.Errorf("onboarding texts: %w", err)
	}
	if t.Schema != "watch.onboarding.texts/1" || len(t.Steps) == 0 {
		return t, errors.New("onboarding texts: schema or steps missing")
	}
	seen := map[string]bool{}
	for _, s := range t.Steps {
		if s.ID == "" || seen[s.ID] || s.Label == "" {
			return t, fmt.Errorf("onboarding texts: bad step %q", s.ID)
		}
		seen[s.ID] = true
	}
	return t, nil
}

type onboardingHero struct {
	WelcomeDismissed bool              `json:"welcomeDismissed,omitempty"`
	Collapsed        bool              `json:"collapsed,omitempty"`
	ListForced       bool              `json:"listForced,omitempty"` // Help brought the finished list back
	Ticks            map[string]string `json:"ticks,omitempty"`      // step -> RFC 3339 time it ticked
}

type onboardingFile struct {
	Schema string                    `json:"schema"`
	Heroes map[string]onboardingHero `json:"heroes"`
}

// OnboardingConfig wires the parts. Every function may be nil.
type OnboardingConfig struct {
	Path    string                 // "" keeps the state in memory
	Hero    func() string          // the hero's name; "" while unknown
	Signals func() map[string]bool // steps the bot's own state shows as done
	Away    func() (time.Duration, bool)
	Now     func() time.Time
}

// Onboarding is the state and the routes. Safe for concurrent use.
type Onboarding struct {
	cfg   OnboardingConfig
	texts OnboardingTexts
	boot  string

	mu sync.Mutex
	st onboardingFile
}

// NewOnboarding loads the texts and the state file (a missing or unreadable file means a fresh start).
func NewOnboarding(c OnboardingConfig) (*Onboarding, error) {
	t, err := LoadOnboardingTexts(onboardingdata.JSON)
	if err != nil {
		return nil, err
	}
	if c.Now == nil {
		c.Now = time.Now
	}
	o := &Onboarding{cfg: c, texts: t, boot: strconv.FormatInt(c.Now().UnixMilli(), 36), st: onboardingFile{Schema: onboardingStateSchema, Heroes: map[string]onboardingHero{}}}
	if b, err := os.ReadFile(c.Path); c.Path != "" && err == nil {
		var f onboardingFile
		if json.Unmarshal(b, &f) == nil && f.Heroes != nil {
			o.st = f
			o.st.Schema = onboardingStateSchema
		}
	}
	return o, nil
}

func (o *Onboarding) key() string {
	if o.cfg.Hero != nil {
		if n := strings.TrimSpace(o.cfg.Hero()); n != "" {
			return safeText(n, 40)
		}
	}
	return "_"
}

// saveLocked writes the file next to itself and renames it, so a crash never leaves half a file.
func (o *Onboarding) saveLocked() {
	if o.cfg.Path == "" {
		return
	}
	b, err := json.MarshalIndent(o.st, "", " ")
	if err != nil {
		return
	}
	tmp := o.cfg.Path + ".tmp"
	if os.MkdirAll(filepath.Dir(o.cfg.Path), 0o700) == nil && os.WriteFile(tmp, b, 0o600) == nil {
		os.Rename(tmp, o.cfg.Path)
	}
}

func (o *Onboarding) hasStep(id string) bool {
	for _, s := range o.texts.Steps {
		if s.ID == id {
			return true
		}
	}
	return false
}

// Latch records a step the bot saw happen (called by the answer wrapper and by View for the event signals).
func (o *Onboarding) Latch(step string) {
	if o == nil || !o.hasStep(step) {
		return
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.latchLocked(o.key(), step)
}

func (o *Onboarding) latchLocked(k, step string) bool {
	h := o.st.Heroes[k]
	if _, ok := h.Ticks[step]; ok {
		return false
	}
	if h.Ticks == nil {
		h.Ticks = map[string]string{}
	}
	h.Ticks[step] = o.cfg.Now().UTC().Format(time.RFC3339)
	o.st.Heroes[k] = h
	return true
}

// OnboardingStepView is a step with its state.
type OnboardingStepView struct {
	OnboardingStep
	Done bool `json:"done"`
}

// OnboardingAway is the viewer's absence, from the same tracker as the return screen.
type OnboardingAway struct {
	Absent  bool    `json:"absent"`
	Seconds float64 `json:"seconds"`
}

// OnboardingView is GET /api/onboarding.
type OnboardingView struct {
	Schema      string               `json:"schema"`
	Boot        string               `json:"boot"` // changes when the bot process restarts
	Hero        string               `json:"hero"`
	ShowWelcome bool                 `json:"showWelcome"`
	ShowList    bool                 `json:"showList"`
	Collapsed   bool                 `json:"collapsed"`
	AllDone     bool                 `json:"allDone"`
	Done        int                  `json:"done"`
	Steps       []OnboardingStepView `json:"steps"`
	Away        OnboardingAway       `json:"away"`
	Texts       OnboardingTexts      `json:"texts"`
}

// View builds the reply and latches the signals it sees.
func (o *Onboarding) View() OnboardingView {
	var sig map[string]bool
	if o.cfg.Signals != nil {
		sig = o.cfg.Signals()
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	k := o.key()
	changed := false
	for id, on := range sig {
		if on && o.hasStep(id) && o.latchLocked(k, id) {
			changed = true
		}
	}
	if changed {
		o.saveLocked()
	}
	h := o.st.Heroes[k]
	v := OnboardingView{Schema: onboardingSchema, Boot: o.boot, Hero: k, ShowWelcome: !h.WelcomeDismissed, Collapsed: h.Collapsed, Texts: o.texts}
	if k == "_" {
		v.Hero = ""
	}
	for _, s := range o.texts.Steps {
		_, done := h.Ticks[s.ID]
		if done {
			v.Done++
		}
		v.Steps = append(v.Steps, OnboardingStepView{OnboardingStep: s, Done: done})
	}
	v.AllDone = v.Done == len(v.Steps)
	v.ShowList = !v.AllDone || h.ListForced
	if o.cfg.Away != nil {
		if d, ok := o.cfg.Away(); ok && d > AwayAfter {
			v.Away = OnboardingAway{Absent: true, Seconds: d.Seconds()}
		}
	}
	return v
}

type onboardingBody struct {
	Action string `json:"action"`
	Step   string `json:"step"`
}

// Apply does one POST action: dismiss_welcome, show_welcome, collapse, expand, show_list, tick (needs step).
func (o *Onboarding) Apply(b onboardingBody) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	k := o.key()
	h := o.st.Heroes[k]
	switch b.Action {
	case "dismiss_welcome":
		h.WelcomeDismissed = true
	case "show_welcome":
		h.WelcomeDismissed = false
	case "collapse":
		h.Collapsed = true
	case "expand":
		h.Collapsed = false
	case "show_list":
		h.ListForced, h.Collapsed = true, false
	case "tick":
		if !o.hasStep(b.Step) {
			return errors.New("unknown step")
		}
		o.st.Heroes[k] = h
		o.latchLocked(k, b.Step)
		o.saveLocked()
		return nil
	default:
		return errors.New("unknown action")
	}
	o.st.Heroes[k] = h
	o.saveLocked()
	return nil
}

// Handler serves the viewer GET route.
func (o *Onboarding) Handler() http.Handler {
	m := http.NewServeMux()
	m.HandleFunc(OnboardingRoute, func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, o.View()) })
	return readOnly(m)
}

// PostHandler serves the POST route on the answer listener.
func (o *Onboarding) PostHandler(ao AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var b onboardingBody
		if !policyGate(ao, w, r, &b) {
			return
		}
		if err := o.Apply(b); err != nil {
			writeJSON(w, http.StatusBadRequest, policyReply{Code: "body", Error: err.Error()})
			return
		}
		writeJSON(w, 200, policyReply{OK: true})
	})
}

// Observe wraps an answer handler: a 200 from the equip route ticks "equip", from the mark route with the mark "sell"
// ticks "junk", from the policy route ticks "policy". The body is read once and handed back unchanged.
func (o *Onboarding) Observe(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body []byte
		if r.Method == http.MethodPost && r.Body != nil && r.URL.Path == "/api/inventory/mark" {
			body, _ = io.ReadAll(io.LimitReader(r.Body, maxBody+1))
			r.Body = io.NopCloser(bytes.NewReader(body))
		}
		sw := &statusWriter{ResponseWriter: w, code: 200}
		next.ServeHTTP(sw, r)
		if sw.code != 200 || r.Method != http.MethodPost {
			return
		}
		switch r.URL.Path {
		case "/api/inventory/equip":
			o.Latch("equip")
		case "/api/inventory/mark":
			var b struct {
				Mark string `json:"mark"`
			}
			if json.Unmarshal(body, &b) == nil && b.Mark == "sell" {
				o.Latch("junk")
			}
		case PolicyPostRoute:
			o.Latch("policy")
		}
	})
}

type statusWriter struct {
	http.ResponseWriter
	code int
}

func (s *statusWriter) WriteHeader(c int) { s.code = c; s.ResponseWriter.WriteHeader(c) }

// Flush keeps the wrapped writer's streaming behaviour.
func (s *statusWriter) Flush() {
	if f, ok := s.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

// opened wraps a viewer page: a 200 answer to its address ticks the step.
func (o *Onboarding) opened(step string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sw := &statusWriter{ResponseWriter: w, code: 200}
		next.ServeHTTP(sw, r)
		if sw.code == 200 && r.Method == http.MethodGet {
			o.Latch(step)
		}
	})
}

// onboardingSignals reads the steps the bot's own events show as done: a kill is the first fight.
func onboardingSignals(l *AwayLog) map[string]bool {
	out := map[string]bool{}
	for _, e := range l.Events() {
		if e.Kind == AwayKill {
			out["fight"] = true
			break
		}
	}
	return out
}
