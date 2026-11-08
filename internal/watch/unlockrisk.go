package watch

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"d2lod-bot/internal/decide"
)

// The hero risk meter (TASK-159, docs/unlock-risk.md). A gauge over the hero's own numbers that turns into one verdict
// before a boss: ready, marginal or turn back. It never has a formula of its own for the bot's decisions: the kit is the
// bot's kit enum (ladder.KitFor), the guard verdict is what the bot's route guard says for the zone (areaGuardWhy), the
// life lines are the bot's engage lines (resource tuning), the potion floor and the class lead are the zone proposer's
// table and the zone range is the tally zone table. So the meter flips to "turn back" exactly where a guard or the hero's
// own engage rule would refuse, and it cannot contradict the bot. It reads nothing from the game and orders nothing.

const (
	// RiskID is the ladder row; RiskSeen is the viewer-side moment that opens it (trigger kind seen).
	RiskID   = "risk"
	RiskSeen = "near_death_survived"
	// RiskRoute is GET (viewer); RiskSectionID is the return screen section "Closest call".
	RiskRoute     = "/api/risk"
	RiskSectionID = "risk"

	riskSchema = "watch.risk/1"
)

// Verdicts.
const (
	RiskReady    = "ready"
	RiskMarginal = "marginal"
	RiskTurnBack = "turn_back"
)

// Check states.
const (
	checkOK    = "ok"
	checkWarn  = "warn"
	checkBlock = "block"
)

// RiskFacts is the snapshot the bot supplies; every field is a number or a decision the bot already uses.
type RiskFacts struct {
	Zone         string // the target zone or boss field
	Kit          string // OK, THIN, NONE: ladder.KitFor, the same enum the proposer and the tower guard read
	Potions      int
	LifePct      int    // current life in percent, 0 to 100
	Dead         bool   // the hero is dead now (a dip that ends in death is not a survived close call)
	EngagePct    int    // resource.EngageLifePct: below it the hero does not start fights
	MinEngagePct int    // resource.MinEngageLifePct: below it the hero does not engage at all
	GuardWhy     string // the bot's route guard for Zone ("" = open): areaGuardWhy
	// ResistGap is how many resist points the hero lacks against Zone (0 = none known). U: no resist table for zones
	// exists in the repo; the bot supplies it when one does.
	ResistGap int
	Ok        bool
}

// RiskTable holds the meter's own thresholds (risk.json).
type RiskTable struct {
	Schema             string `json:"schema"`
	Doc                string `json:"doc,omitempty"`
	NearDeathPct       int    `json:"nearDeathPct"`
	RecoverPct         int    `json:"recoverPct"`
	CloseCallsMarginal int    `json:"closeCallsMarginal"`
	CloseCallWindowMin int    `json:"closeCallWindowMin"`
	Keep               int    `json:"keep"`
}

//go:embed risk.json
var shippedRisk []byte

// ShippedRiskTable parses the embedded table.
func ShippedRiskTable() (*RiskTable, error) { return ParseRiskTable(shippedRisk) }

// ParseRiskTable checks the table; errors name the field.
func ParseRiskTable(b []byte) (*RiskTable, error) {
	var t RiskTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("risk table: %w", err)
	}
	switch {
	case t.Schema != riskSchema:
		return nil, fmt.Errorf("risk table: schema %q, want %q", t.Schema, riskSchema)
	case t.NearDeathPct < 1 || t.NearDeathPct >= t.RecoverPct || t.RecoverPct > 100:
		return nil, fmt.Errorf("risk table: nearDeathPct %d and recoverPct %d must satisfy 1 <= near < recover <= 100", t.NearDeathPct, t.RecoverPct)
	case t.CloseCallsMarginal < 1 || t.CloseCallWindowMin < 1 || t.Keep < 1:
		return nil, errors.New("risk table: closeCallsMarginal, closeCallWindowMin and keep must be 1 or more")
	}
	return &t, nil
}

// RiskCheck is one line of the gauge.
type RiskCheck struct {
	Name   string `json:"name"`
	State  string `json:"state"` // ok, warn, block
	Detail string `json:"detail"`
}

// RiskVerdict is the meter's answer for one zone.
type RiskVerdict struct {
	Verdict string      `json:"verdict"`
	Gauge   int         `json:"gauge"` // 0 to 100: turn back stays below 30, marginal below 70
	Zone    string      `json:"zone"`
	Reasons []string    `json:"reasons"`
	Checks  []RiskCheck `json:"checks"`
}

// RiskInput is what Assess reads besides the facts.
type RiskInput struct {
	Class      string
	Clvl       int
	Facts      RiskFacts
	CloseCalls int // close calls inside the window
}

// RiskRules are the shared tables: the proposer's (potion floor, class lead) and the zone range function.
type RiskRules struct {
	Propose *decide.ProposeTable
	Ranges  decide.RangeFn
	Table   *RiskTable
}

// Assess is the verdict. Block checks mean the bot itself would refuse (its guard, an empty kit, the life line below
// which it does not engage); warn checks are the proposer's own readiness conditions that fail, a life below the engage
// line, a resist gap and repeated close calls. An unknown class or a zone without a range is an error naming it.
func (r RiskRules) Assess(in RiskInput) (RiskVerdict, error) {
	f := in.Facts
	cl, ok := r.Propose.Classes[strings.ToLower(in.Class)]
	if !ok {
		return RiskVerdict{}, fmt.Errorf("risk: no lead for class %q in the proposer table", in.Class)
	}
	lo, hi, ok := r.Ranges(f.Zone)
	if !ok {
		return RiskVerdict{}, fmt.Errorf("risk: zone %q has no range in the zone table", f.Zone)
	}
	v := RiskVerdict{Zone: f.Zone, Reasons: []string{}}
	add := func(name, state, detail string) {
		v.Checks = append(v.Checks, RiskCheck{name, state, detail})
		if state != checkOK {
			v.Reasons = append(v.Reasons, detail)
		}
	}
	switch {
	case f.GuardWhy != "":
		add("guard", checkBlock, "the hero's route guard is closed: "+f.GuardWhy)
	default:
		add("guard", checkOK, "the route guard is open")
	}
	switch f.Kit {
	case "NONE":
		add("kit", checkBlock, "no healing potion and no town portal")
	case "THIN":
		add("kit", checkWarn, "the kit is thin")
	default:
		add("kit", checkOK, "the kit is complete")
	}
	if f.Potions < r.Propose.MinPotions {
		add("potions", checkWarn, fmt.Sprintf("%d potion%s in the belt, the hero wants %d", f.Potions, riskS(f.Potions), r.Propose.MinPotions))
	} else {
		add("potions", checkOK, fmt.Sprintf("%d potion%s in the belt", f.Potions, riskS(f.Potions)))
	}
	switch {
	case f.MinEngagePct > 0 && f.LifePct < f.MinEngagePct:
		add("life", checkBlock, fmt.Sprintf("life %d%% is below the %d%% the hero fights at", f.LifePct, f.MinEngagePct))
	case f.EngagePct > 0 && f.LifePct < f.EngagePct:
		add("life", checkWarn, fmt.Sprintf("life %d%% is below the %d%% the hero starts a fight at", f.LifePct, f.EngagePct))
	default:
		add("life", checkOK, fmt.Sprintf("life %d%%", f.LifePct))
	}
	if need := lo + cl.Lead; in.Clvl < need {
		add("level", checkWarn, fmt.Sprintf("level %d is below %d (zone %d to %d plus a lead of %d)", in.Clvl, need, lo, hi, cl.Lead))
	} else {
		add("level", checkOK, fmt.Sprintf("level %d, zone %d to %d", in.Clvl, lo, hi))
	}
	if f.ResistGap > 0 {
		add("resist", checkWarn, fmt.Sprintf("%d resist point%s short for the zone", f.ResistGap, riskS(f.ResistGap)))
	} else {
		add("resist", checkOK, "no resist gap known")
	}
	if in.CloseCalls >= r.Table.CloseCallsMarginal {
		add("close calls", checkWarn, fmt.Sprintf("%d close calls in the last %d minutes", in.CloseCalls, r.Table.CloseCallWindowMin))
	} else {
		add("close calls", checkOK, fmt.Sprintf("%d close call%s lately", in.CloseCalls, riskS(in.CloseCalls)))
	}
	okN, blocks, warns := 0, 0, 0
	for _, c := range v.Checks {
		switch c.State {
		case checkOK:
			okN++
		case checkBlock:
			blocks++
		default:
			warns++
		}
	}
	v.Gauge = 100 * okN / len(v.Checks)
	switch {
	case blocks > 0:
		v.Verdict, v.Gauge = RiskTurnBack, min(v.Gauge, 29)
	case warns > 0:
		v.Verdict, v.Gauge = RiskMarginal, min(v.Gauge, 69)
	default:
		v.Verdict = RiskReady
	}
	return v, nil
}

func riskS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// CloseCall is one near-death the hero survived.
type CloseCall struct {
	At     float64 `json:"at"` // Hub time
	Zone   string  `json:"zone"`
	LowPct int     `json:"lowPct"`
}

// RiskConfig builds a RiskMeter. Facts is called with the zone asked for ("" = the bot's next target).
type RiskConfig struct {
	Propose *decide.ProposeTable
	Ranges  decide.RangeFn
	Table   *RiskTable // nil: the shipped table
	Facts   func(zone string) RiskFacts
}

// RiskMeter is the watch side of the meter. Safe for concurrent use.
type RiskMeter struct {
	u     *Unlocks
	hub   *Hub
	rules RiskRules
	facts func(zone string) RiskFacts

	mu    sync.Mutex
	armed bool // life fell below nearDeathPct and has not recovered yet
	low   int
	zone  string
	calls []CloseCall
	errs  []string
}

// NewRiskMeter checks the config. A ladder without the row "risk" is an error naming it.
func NewRiskMeter(u *Unlocks, h *Hub, c RiskConfig) (*RiskMeter, error) {
	if _, err := u.Engine.Check(RiskID); err != nil {
		return nil, err
	}
	if h == nil || c.Propose == nil || c.Ranges == nil || c.Facts == nil {
		return nil, errors.New("watch: the risk meter needs a hub, the proposer table, a range function and a facts reader")
	}
	t := c.Table
	if t == nil {
		var err error
		if t, err = ShippedRiskTable(); err != nil {
			return nil, err
		}
	}
	return &RiskMeter{u: u, hub: h, rules: RiskRules{Propose: c.Propose, Ranges: c.Ranges, Table: t}, facts: c.Facts}, nil
}

func (m *RiskMeter) now() float64 { return m.hub.rel(m.hub.cfg.Clock.Now()) }

func (m *RiskMeter) fail(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.errs) < 20 {
		m.errs = append(m.errs, err.Error())
	}
}

// Errors lists what the meter could not do, oldest first.
func (m *RiskMeter) Errors() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string(nil), m.errs...)
}

// Tick reads the life once (call it from the loop that polls the status). Life below nearDeathPct arms the detector; a
// recovery to recoverPct or more records one close call and reports the moment "near_death_survived" to the ladder. A
// death while armed drops the dip without a record.
func (m *RiskMeter) Tick() {
	f := m.facts("")
	if !f.Ok {
		return
	}
	t := m.rules.Table
	m.mu.Lock()
	if f.Dead {
		m.armed = false
		m.mu.Unlock()
		return
	}
	var done *CloseCall
	switch {
	case f.LifePct < t.NearDeathPct:
		if !m.armed || f.LifePct < m.low {
			m.low = f.LifePct
		}
		if !m.armed {
			m.zone = f.Zone
		}
		m.armed = true
	case m.armed && f.LifePct >= t.RecoverPct:
		cc := CloseCall{At: m.now(), Zone: m.zone, LowPct: m.low}
		m.calls = append(m.calls, cc)
		if len(m.calls) > t.Keep {
			m.calls = m.calls[len(m.calls)-t.Keep:]
		}
		m.armed, done = false, &cc
	}
	m.mu.Unlock()
	if done != nil {
		m.u.Seen(RiskSeen)
	}
}

func (m *RiskMeter) recent() int {
	cut := m.now() - float64(m.rules.Table.CloseCallWindowMin)*60
	n := 0
	for _, c := range m.calls {
		if c.At >= cut {
			n++
		}
	}
	return n
}

// RiskView is GET /api/risk. Locked: the teaser only.
type RiskView struct {
	Schema  string       `json:"schema"`
	Open    bool         `json:"open"`
	Title   string       `json:"title"`
	Teaser  string       `json:"teaser,omitempty"`
	Verdict *RiskVerdict `json:"verdict,omitempty"`
	Error   string       `json:"error,omitempty"`
	Closest *CloseCall   `json:"closest,omitempty"`
	Recent  int          `json:"recentCloseCalls"`
}

// View answers for zone ("" = the bot's next target).
func (m *RiskMeter) View(zone string) RiskView {
	st, _ := m.u.state(RiskID)
	v := RiskView{Schema: riskSchema, Open: st.Open, Title: st.Title}
	if !st.Open {
		v.Teaser = st.Teaser
		return v
	}
	m.mu.Lock()
	v.Recent = m.recent()
	for i := range m.calls {
		if v.Closest == nil || m.calls[i].LowPct < v.Closest.LowPct {
			c := m.calls[i]
			v.Closest = &c
		}
	}
	m.mu.Unlock()
	f := m.facts(zone)
	if !f.Ok {
		v.Error = "the hero's numbers are not read yet"
		return v
	}
	hero := m.hub.heroNow()
	verdict, err := m.rules.Assess(RiskInput{Class: hero.Class, Clvl: hero.Level, Facts: f, CloseCalls: v.Recent})
	if err != nil {
		m.fail(err)
		v.Error = err.Error()
		return v
	}
	v.Verdict = &verdict
	return v
}

// Handler serves GET /api/risk[?zone=Name]. The zone is matched by the table, never echoed unchecked.
func (m *RiskMeter) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, m.View(safeText(r.URL.Query().Get("zone"), 40)))
	})
}

// Section is the return screen's "Closest call": the lowest dip since the viewer left. Hidden when there was none or the
// row is shut.
func (m *RiskMeter) Section() ReturnSection {
	return SectionFunc(func(d *ReturnData) *SectionView {
		if !m.u.Engine.Allowed(RiskID) {
			return nil
		}
		m.mu.Lock()
		defer m.mu.Unlock()
		var best *CloseCall
		for i := range m.calls {
			if c := m.calls[i]; c.At >= d.Since && (best == nil || c.LowPct < best.LowPct) {
				best = &c
			}
		}
		if best == nil {
			return nil
		}
		line := fmt.Sprintf("Life fell to %d%% and the hero got through", best.LowPct)
		if best.Zone != "" {
			line += " in " + best.Zone
		}
		return &SectionView{ID: RiskSectionID, Title: "Closest call", Lines: []string{line + "."}}
	})
}
