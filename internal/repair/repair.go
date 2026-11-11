// Package repair is the player's say over gear repair (docs/repair.md): the policy mode, the "Repair now" request and
// the gear condition the watch screen shows. The town asks Permit at a visit; nothing here touches the game.
package repair

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// StateFile is the persisted request state inside the bot's state directory.
	StateFile = "repair-ui.json"

	StateRequested = "requested"
	StateDone      = "done"
	StateFailed    = "failed"
	// StateCancel (see trip.go) is a request the player or the bot withdrew.
)

// Cond is the condition of one worn item.
type Cond struct {
	Slot   string `json:"slot"`
	Name   string `json:"name"`
	Dur    int    `json:"dur"`
	Max    int    `json:"max"`
	Pct    int    `json:"pct"`
	Broken bool   `json:"broken,omitempty"`
	Warn   bool   `json:"warn,omitempty"` // under the warn percent (and not broken)
}

// Facts is what the town knows at a visit.
type Facts struct {
	Now     time.Time
	Lowest  int  // lowest worn durability percent, 0..100
	Damaged bool // any worn item below full durability
	Broken  bool // any worn item at zero
}

// Request is the player's "repair now".
type Request struct {
	State string `json:"state"`
	At    int64  `json:"at"`
	Done  int64  `json:"doneAt,omitempty"`
	Note  string `json:"note,omitempty"`
	Short int    `json:"short,omitempty"` // gold still missing for the repair (0 = not waiting for gold)
	Need  int    `json:"need,omitempty"`  // the estimate it waits for

	// The repair trip (schema 2, trip.go). All omitted in files and replies of the old shape.
	Kind      string `json:"kind,omitempty"`      // KindRepair or KindFarm, fixed by the live gold at the click
	Phase     string `json:"phase,omitempty"`     // while State is requested
	Trips     int    `json:"trips,omitempty"`     // trips made so far
	TripAt    int64  `json:"tripAt,omitempty"`    // start of the last trip
	FarmLegs  int    `json:"farmLegs,omitempty"`  // farm legs begun
	Have      int    `json:"have,omitempty"`      // farm leg counter
	Target    int    `json:"target,omitempty"`    // farm leg target
	Area      string `json:"area,omitempty"`      // zone being farmed
	DurBefore int    `json:"durBefore,omitempty"` // lowest worn durability percent at the trip start
	DurAfter  int    `json:"durAfter,omitempty"`  // and after
	Auto      bool   `json:"auto,omitempty"`      // withdrawn by the bot, not by the player
}

type stored struct {
	Request    *Request `json:"request,omitempty"`
	LastRepair int64    `json:"lastRepair,omitempty"`
	LastEnd    int64    `json:"lastEnd,omitempty"` // when the last request ended (done, failed or cancelled)
}

// Refusal is an error with a short code and a reason line for the player (the routes answer 409 with them).
type Refusal struct{ Code, Reason string }

func (r *Refusal) Error() string { return r.Reason }
func (r *Refusal) Kind() string  { return r.Code }

// Config wires the controller to the bot. Every func may be nil.
type Config struct {
	Path     string                   // state file; empty: nothing is persisted
	Mode     func() string            // the policy row's current option id; nil or "" = the default mode
	Allowed  func(unlock string) bool // ladder test; nil: every mode is open
	Gear     func() []Cond            // worn items with durability (the bot reads it); nil: no readings
	Estimate func() int               // gold Repair All would cost now (a lower bound); nil: unknown
	Gold     func() int               // live gold in the pack (not the stash); nil: unknown
	InTown   func() bool
	Now      func() time.Time
}

// Controller holds the request state. Safe for concurrent use.
type Controller struct {
	c  Config
	t  *Table
	mu sync.Mutex
	st stored
}

// New makes the controller over the shipped modes table and loads the state file (missing: empty; damaged: error).
func New(c Config) (*Controller, error) {
	t, err := Shipped()
	if err != nil {
		return nil, err
	}
	return NewWithTable(c, t)
}

// NewWithTable is New over a given table (tests).
func NewWithTable(c Config, t *Table) (*Controller, error) {
	if c.Now == nil {
		c.Now = time.Now
	}
	r := &Controller{c: c, t: t}
	if c.Path == "" {
		return r, nil
	}
	b, err := os.ReadFile(c.Path)
	switch {
	case errors.Is(err, os.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		if err := json.Unmarshal(b, &r.st); err != nil {
			return nil, fmt.Errorf("repair: %s: %w", c.Path, err)
		}
		r.st.restart()
	}
	return r, nil
}

// restart makes a request loaded from the file safe to resume: the trip itself lived in memory, so a phase that was
// mid-trip starts over (a hero already in town goes straight to the smith); a request whose repair was paid is done.
func (s *stored) restart() {
	q := s.Request
	if q == nil || q.State != StateRequested {
		return
	}
	switch q.Phase {
	case PhaseReturning:
		q.State, q.Phase = StateDone, ""
		s.LastEnd = max(s.LastEnd, q.Done)
	case PhaseGoing, PhaseRepairing, PhaseWaiting:
		q.Phase = PhaseQueued
	}
}

// Bind sets the mode source (the policy row's current option) and the ladder test once they exist; nil keeps the
// default mode and leaves every mode open (no ladder to buy from).
func (r *Controller) Bind(mode func() string, allowed func(string) bool) {
	r.mu.Lock()
	r.c.Mode, r.c.Allowed = mode, allowed
	r.mu.Unlock()
}

// Gear is the worn-gear reading (nil without a reader).
func (r *Controller) Gear() []Cond {
	if r.c.Gear == nil {
		return nil
	}
	return r.c.Gear()
}

// Table is the modes table.
func (r *Controller) Table() *Table { return r.t }

// ModeID is the mode in force: the policy option when it names an open mode, else the default. A mode that is locked or
// unknown acts as manual, so a hand-edited policy file can never buy a mode.
func (r *Controller) ModeID() string {
	id := r.t.Default
	mode, _ := r.sources()
	if mode != nil {
		if v := mode(); v != "" {
			id = v
		}
	}
	m, err := r.t.Mode(id)
	if err != nil || !r.open(m) {
		return r.t.Default
	}
	return m.ID
}

func (r *Controller) open(m *Mode) bool {
	_, allowed := r.sources()
	return m.Unlock == "" || allowed == nil || allowed(m.Unlock)
}

func (r *Controller) sources() (func() string, func(string) bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.c.Mode, r.c.Allowed
}

// Request is the "Repair now" button. The live pack gold decides the kind: enough for the estimate x goldMult makes a
// repair trip, less makes a farm leg first. Refused with a reason line when a Pact forbids repairs, nothing worn is
// damaged, a request is already open, or the last one ended less than the cooldown ago.
func (r *Controller) Request() (Request, error) {
	if r.pending() {
		return Request{}, &Refusal{"active", "a repair request is already open"}
	}
	b, kind, need, gold, _ := r.idleButton()
	if !b.Enabled {
		code := "undamaged"
		switch {
		case b.Reason == "a Pact forbids repairs":
			code = "pact"
		case b.Action == "none" && r.damaged():
			code = "cooldown"
		}
		return Request{}, &Refusal{code, b.Reason}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	now := r.c.Now()
	req := Request{State: StateRequested, At: now.Unix(), Kind: kind, Phase: PhaseQueued}
	if kind == KindFarm {
		req.Phase, req.FarmLegs, req.Target, req.Have = PhaseFarming, 1, need, gold
	}
	r.st.Request = &req
	slog.Info("repair requested by the player", "kind", kind, "gold", gold, "need", need)
	return req, r.saveLocked()
}

func (r *Controller) pending() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.pendingLocked()
}

// Cancel withdraws a pending request.
func (r *Controller) Cancel() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.st.Request == nil || r.st.Request.State != StateRequested {
		return &Refusal{"none", "no repair is requested"}
	}
	slog.Info("repair request cancelled by the player", "phase", r.st.Request.Phase)
	r.endLocked(StateCancel, "", r.c.Now(), false)
	return nil
}

// Permit says whether this town visit may repair. requested is true when the player's request is the reason (then the
// town does not make a trip of its own for it). Anything undamaged is never repaired.
func (r *Controller) Permit(f Facts) (ok, requested bool) {
	if !f.Damaged {
		return false, false
	}
	r.mu.Lock()
	pending := r.st.Request != nil && r.st.Request.State == StateRequested
	last := r.st.LastRepair
	r.mu.Unlock()
	m, err := r.t.Mode(r.ModeID())
	if err != nil {
		return false, false
	}
	if pending && !m.Never {
		return true, true
	}
	switch {
	case m.Never:
		return false, false
	case m.Always:
		return true, false
	case m.BelowPct > 0:
		return f.Lowest < m.BelowPct, false
	case m.OnBreak:
		return f.Broken, false
	case m.EverySecs > 0:
		return last == 0 || f.Now.Unix()-last >= int64(m.EverySecs), false
	}
	return false, false
}

// Blocked notes why a requested repair did not happen at this visit (for example not enough gold); it stays requested.
func (r *Controller) Blocked(reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.st.Request != nil && r.st.Request.State == StateRequested && r.st.Request.Note != reason {
		r.st.Request.Note = reason
		_ = r.saveLocked()
	}
}

// Short notes that a requested repair waits for gold: missing more, need in all. It stays requested.
func (r *Controller) Short(missing, need int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if q := r.st.Request; q != nil && q.State == StateRequested && (q.Short != missing || q.Need != need) {
		q.Short, q.Need = missing, need
		q.Note = fmt.Sprintf("waiting for gold: need %d more", missing)
		_ = r.saveLocked()
	}
}

// Requested says whether a repair request is pending.
func (r *Controller) Requested() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.st.Request != nil && r.st.Request.State == StateRequested
}

// Repaired records a finished Repair All (the town confirmed durability rose).
func (r *Controller) Repaired(now time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.st.LastRepair = now.Unix()
	if q := r.st.Request; q != nil && q.State == StateRequested {
		if q.Phase == PhaseRepairing { // on a trip: done once the hero is back in the field (EndTrip)
			q.Phase, q.Note, q.Short, q.Need = PhaseReturning, "", 0, 0
			q.Done = now.Unix()
		} else {
			r.endLocked(StateDone, "", now, false)
		}
	}
	_ = r.saveLocked()
}

// Failed records an attempt that did not repair. A request stays pending (the next visit tries again) with the reason.
func (r *Controller) Failed(reason string) { r.Blocked(reason) }

func (r *Controller) saveLocked() error {
	if r.c.Path == "" {
		return nil
	}
	b, err := json.MarshalIndent(r.st, "", " ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(r.c.Path), 0o755); err != nil {
		return err
	}
	tmp := r.c.Path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, r.c.Path)
}

// ModeView is a mode for the page.
type ModeView struct {
	ID      string `json:"id"`
	Label   string `json:"label"`
	Current bool   `json:"current"`
	Locked  bool   `json:"locked,omitempty"`
}

// View is GET /api/repair.
type View struct {
	Schema   int        `json:"schema"` // 2 since the repair trip: `trip` and `gold` are new, every v1 field is unchanged
	Mode     string     `json:"mode"`
	Modes    []ModeView `json:"modes"`
	WarnPct  int        `json:"warnPct"`
	InTown   bool       `json:"inTown"`
	Gear     []Cond     `json:"gear"`
	Estimate int        `json:"estimate,omitempty"` // gold, a lower bound; 0 = unknown or nothing to repair
	Request  *Request   `json:"request,omitempty"`
	Worst    string     `json:"worst,omitempty"` // one line for the alert and the strip: "" when all is well
	Trip     *TripView  `json:"trip,omitempty"`  // schema 2: the button state and the trip's phase
}

// View reads the gear and the request.
func (r *Controller) View() View {
	cur := r.ModeID()
	v := View{Schema: 2, Mode: cur, WarnPct: r.t.WarnPct, Modes: []ModeView{}, Gear: []Cond{}}
	for i := range r.t.Modes {
		m := &r.t.Modes[i]
		if m.Hidden && m.ID != cur {
			continue
		}
		v.Modes = append(v.Modes, ModeView{ID: m.ID, Label: m.Label, Current: m.ID == cur, Locked: !r.open(m)})
	}
	if r.c.InTown != nil {
		v.InTown = r.c.InTown()
	}
	if r.c.Gear != nil {
		v.Gear = r.c.Gear()
	}
	if r.c.Estimate != nil {
		v.Estimate = r.c.Estimate()
	}
	r.mu.Lock()
	if r.st.Request != nil {
		q := *r.st.Request
		v.Request = &q
	}
	r.mu.Unlock()
	v.Worst = Worst(v.Gear)
	tv := r.tripView()
	v.Trip = &tv
	return v
}

// Worst is the alert line for a gear list: the broken items first, else the lowest one under the warn flag; "" when
// nothing is low. The list's Warn and Broken flags decide, so the percent is applied once, in Classify.
func Worst(g []Cond) string {
	var broken, low []string
	for _, c := range g {
		switch {
		case c.Broken:
			broken = append(broken, c.Name)
		case c.Warn:
			low = append(low, fmt.Sprintf("%s %d%%", c.Name, c.Pct))
		}
	}
	switch {
	case len(broken) > 0:
		return "Broken: " + join(broken)
	case len(low) > 0:
		return "Low durability: " + join(low)
	}
	return ""
}

func join(s []string) string {
	out := ""
	for i, x := range s {
		if i > 0 {
			out += ", "
		}
		out += x
	}
	return out
}

// Classify fills Pct, Broken and Warn of a reading. maxDur <= 0 (no durability, for instance a ring) gives ok false.
func Classify(slot, name string, dur, maxDur, warnPct int) (Cond, bool) {
	if maxDur <= 0 {
		return Cond{}, false
	}
	dur = min(max(dur, 0), maxDur)
	pct := dur * 100 / maxDur
	return Cond{Slot: slot, Name: name, Dur: dur, Max: maxDur, Pct: pct, Broken: dur == 0, Warn: dur > 0 && pct < warnPct}, true
}
