package watch

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/decide"
)

// The hero proposes the next zone (TASK-131, docs/proposezone.md). The desk asks the proposer (internal/decide) each time
// the bot settles, lists what is waiting on the return screen and on /api/proposals, and turns a "go" into an order
// through the same placer as POST /api/orders (so the charter flow applies). The desk never orders by itself.

// ProposalsRoute is the GET route (viewer); ProposalsPostRoute is the POST route (answer listener, same gates as the
// other answer routes).
const (
	ProposalsRoute     = "/api/proposals"
	ProposalsPostRoute = "/api/proposals/answer"
	// ProposalsSectionID is the return screen section the desk fills (it replaces the placeholder in place).
	ProposalsSectionID = "proposals"
	// DefaultAwayMin is the viewer absence in minutes after which a proposal counts as raised while away
	// (knob ladder.awayMin, default 20; the knob row is not added yet, the process passes the value).
	DefaultAwayMin = 20
)

// ProposalFacts is what only the bot knows. Ok false: the facts are not read yet and nothing is proposed.
type ProposalFacts struct {
	Kit     string // OK, THIN, NONE (catalogue field kit)
	Potions int
	Deaths  int // deaths in the zone lately
	Ok      bool
}

// ProposalConfig builds a ProposalDesk. Proposer, Clock and Facts are required; Visit nil means a viewer that is always
// present; Intro nil means no first-time line.
type ProposalConfig struct {
	Proposer *decide.Proposer
	Clock    decide.Clock
	Facts    func() ProposalFacts
	Visit    *Visit
	AwayMin  int
	// Intro gives the Codex line for a zone the hero has not seen ("" when seen or unknown).
	Intro func(area string) string
}

// ProposalDesk is the watch side of the proposer. Safe for concurrent use.
type ProposalDesk struct {
	hub *Hub
	c   ProposalConfig

	mu    sync.Mutex
	field string // the last known field the hero farmed (towns are not fields)
	errs  []string
}

// NewProposalDesk checks the config.
func NewProposalDesk(h *Hub, c ProposalConfig) (*ProposalDesk, error) {
	if h == nil || c.Proposer == nil || c.Clock == nil || c.Facts == nil {
		return nil, errors.New("watch: the proposal desk needs a hub, a proposer, a clock and a facts reader")
	}
	if c.AwayMin <= 0 {
		c.AwayMin = DefaultAwayMin
	}
	return &ProposalDesk{hub: h, c: c}, nil
}

// Errors lists what the desk could not do (an unknown class), oldest first.
func (d *ProposalDesk) Errors() []string {
	d.mu.Lock()
	defer d.mu.Unlock()
	return append([]string(nil), d.errs...)
}

func (d *ProposalDesk) away() bool {
	if d.c.Visit == nil {
		return false
	}
	a, ok := d.c.Visit.Away()
	return ok && a > time.Duration(d.c.AwayMin)*time.Minute
}

// Tick looks at the hero and raises a proposal when one is due. It places nothing.
func (d *ProposalDesk) Tick() {
	hero := d.hub.heroNow()
	if hero.Class == "" || hero.Level < 1 {
		return // the status poll has not reported the character yet
	}
	d.mu.Lock()
	if hero.Area != "" && d.c.Proposer.Knows(hero.Area) {
		d.field = hero.Area
	}
	field := d.field
	d.mu.Unlock()
	if field == "" {
		return
	}
	f := d.c.Facts()
	if !f.Ok {
		return
	}
	_, err := d.c.Proposer.Look(decide.ProposeInput{Class: hero.Class, Clvl: hero.Level, Area: field, Kit: f.Kit, Potions: f.Potions, Deaths: f.Deaths, Away: d.away()})
	if err != nil {
		d.mu.Lock()
		if len(d.errs) < 20 {
			d.errs = append(d.errs, err.Error())
		}
		d.mu.Unlock()
	}
}

// ProposalView is one proposal as the page and the API show it.
type ProposalView struct {
	decide.Proposal
	Intro       string `json:"intro,omitempty"` // first time in the zone: the Codex line
	FarmingMins int    `json:"farmingMins"`     // minutes since the hero was ready, farming where it is
	Farming     string `json:"farming"`         // the sentence for the return screen
}

// ProposalsView is GET /api/proposals.
type ProposalsView struct {
	Schema  string         `json:"schema"`
	Away    bool           `json:"away"`
	Pending []ProposalView `json:"pending"`
}

const proposalsSchema = "watch.proposals/1"

func (d *ProposalDesk) view(p decide.Proposal) ProposalView {
	mins := int((d.c.Clock.Now().Seconds() - p.ReadyAt) / 60)
	if mins < 0 {
		mins = 0
	}
	v := ProposalView{Proposal: p, FarmingMins: mins,
		Farming: fmt.Sprintf("The hero has been farming %s for %d min since it was ready.", p.From, mins)}
	if d.c.Intro != nil {
		v.Intro = d.c.Intro(p.To)
	}
	return v
}

// View is the pending list.
func (d *ProposalDesk) View() *ProposalsView {
	out := &ProposalsView{Schema: proposalsSchema, Away: d.away(), Pending: []ProposalView{}}
	for _, p := range d.c.Proposer.Pending() {
		out.Pending = append(out.Pending, d.view(p))
	}
	return out
}

// Handler serves GET /api/proposals (viewer).
func (d *ProposalDesk) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, d.View())
	})
}

// Section is the return screen's proposals section (replaces the placeholder).
func (d *ProposalDesk) Section() ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		sv := &SectionView{ID: ProposalsSectionID, Title: "Waiting for your yes",
			Placeholder: "Nothing is waiting. The hero keeps farming where it is."}
		for _, p := range d.c.Proposer.Pending() {
			v := d.view(p)
			sv.Lines = append(sv.Lines, v.Text)
			if v.Intro != "" {
				sv.Lines = append(sv.Lines, v.Intro)
			}
			sv.Lines = append(sv.Lines, strings.Join(v.Why, "; ")+".", v.Farming)
			id := fmt.Sprint(p.ID)
			sv.Buttons = append(sv.Buttons,
				SectionButton{Label: "Accept: go to " + p.To, Row: id, Option: decide.ProposeGo, Act: "proposal"},
				SectionButton{Label: "Decline: stay in " + p.From, Row: id, Option: decide.ProposeStay, Act: "proposal"},
				SectionButton{Label: "Ask me later", Row: id, Option: decide.ProposeLater, Act: "proposal"})
		}
		return sv
	})
}

// Answer records the player's answer. "go" places the order first (through the charter placer) and records the answer
// only when the order was taken; a charter or a refusal leaves the proposal pending.
func (d *ProposalDesk) Answer(id int, answer string, approve bool) (decide.Proposal, *Order, *Charter, error) {
	p, ok := d.c.Proposer.Get(id)
	if !ok {
		return p, nil, nil, fmt.Errorf("%w: no proposal %d", ErrNoProposal, id)
	}
	var o *Order
	var ch *Charter
	if answer == decide.ProposeGo && p.State == decide.ProposalPending {
		ord, c, err := d.hub.place(OrderRequest{Kind: OrderGo, Area: p.To, Forever: true, Replace: true}, approve)
		ch = c
		if err != nil {
			return p, nil, ch, err
		}
		o = &ord
	}
	p, err := d.c.Proposer.Answer(id, answer)
	if err != nil {
		return p, o, ch, fmt.Errorf("%w: %v", ErrBadProposal, err)
	}
	return p, o, ch, nil
}

// Errors of the desk.
var (
	ErrNoProposal  = errors.New("watch: no such proposal")
	ErrBadProposal = errors.New("watch: bad proposal answer")
)

type proposalBody struct {
	ID      int    `json:"id"`
	Answer  string `json:"answer"`
	Approve bool   `json:"approve"`
}

type proposalReply struct {
	OK      bool     `json:"ok"`
	Changed bool     `json:"changed,omitempty"`
	Code    string   `json:"code,omitempty"`
	Error   string   `json:"error,omitempty"`
	State   string   `json:"state,omitempty"`
	Order   *Order   `json:"order,omitempty"`
	Charter *Charter `json:"charter,omitempty"`
}

// PostHandler is POST /api/proposals/answer {id, answer, approve}. Same gate as the other answer routes (origin list,
// token, JSON only, small body).
func (d *ProposalDesk) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string, ord *Order, ch *Charter) {
			writeJSON(w, code, proposalReply{Code: c, Error: msg, Order: ord, Charter: ch})
		}
		switch {
		case r.Method == http.MethodOptions && allowed:
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		case r.Method != http.MethodPost:
			w.Header().Set("Allow", "POST, OPTIONS")
			fail(http.StatusMethodNotAllowed, "method", "POST only", nil, nil)
			return
		case origin != "" && !allowed:
			fail(http.StatusForbidden, "origin", "origin not allowed", nil, nil)
			return
		case !o.tokenOK(r):
			fail(http.StatusUnauthorized, "token", "unauthorized", nil, nil)
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			fail(http.StatusUnsupportedMediaType, "content-type", "send application/json", nil, nil)
			return
		}
		var b proposalBody
		if err := decodeSmall(r, &b); err != nil {
			fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields", nil, nil)
			return
		}
		p, ord, ch, err := d.Answer(b.ID, b.Answer, b.Approve)
		switch {
		case err == nil:
			writeJSON(w, 200, proposalReply{OK: true, Changed: true, State: p.State, Order: ord, Charter: ch})
		case errors.Is(err, ErrNoProposal):
			fail(http.StatusNotFound, "not-found", err.Error(), nil, nil)
		case errors.Is(err, ErrBadProposal):
			fail(http.StatusConflict, "bad-answer", err.Error(), nil, nil)
		default:
			code, rep := status(err)
			fail(code, rep.Code, rep.Error, nil, ch)
		}
	})
}

// CodexIntro makes the Intro function from a Codex and its firsts table: for a zone whose area first is not in the Codex
// yet it returns "First time in <zone>: <tip>".
func CodexIntro(c *decide.Codex, t *decide.Firsts) func(string) string {
	return func(area string) string {
		for _, f := range t.Firsts {
			if f.Kind != "area" {
				continue
			}
			for _, m := range f.Match {
				if strings.EqualFold(m, strings.TrimSpace(area)) {
					if c.Has(f.ID) {
						return ""
					}
					return fmt.Sprintf("First time in %s: %s", area, f.Tip)
				}
			}
		}
		return ""
	}
}
