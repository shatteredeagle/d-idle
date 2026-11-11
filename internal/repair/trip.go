package repair

import (
	"fmt"
	"log/slog"
	"math"
	"time"
)

// The repair trip (docs/repair.md, user decision 2026-11-11): "Repair now" is a trip, not a wait for the next town
// visit. This file holds the data knobs, the button state from live gold, the request's phases and the guards; the bot
// (internal/bot/repairtrip.go) drives the phases and the town does the shopping. Nothing here touches the game.

// Trip is the "trip" block of modes.json. Missing fields take the defaults below.
type Trip struct {
	GoldMult         float64 `json:"goldMult"`         // the estimate is a lower bound: gold needed = estimate x this
	CooldownSecs     int     `json:"cooldownSecs"`     // least time between two trips of a request, and before a new request after one ended
	MaxTrips         int     `json:"maxTrips"`         // trips one request may make
	SafeWaitSecs     int     `json:"safeWaitSecs"`     // longest wait for a safe moment before the hero retreats first
	PortalClearTiles int     `json:"portalClearTiles"` // no hostile this close when a portal is cast or the way home starts
	FarmLegs         int     `json:"farmLegs"`         // farm legs one request may use; a request still unpayable after them is cancelled
	ElitePotionMin   int     `json:"elitePotionMin"`   // healing potions needed to finish an engaged boss or elite before leaving
}

func (t *Trip) fill() {
	d := Trip{GoldMult: 1.3, CooldownSecs: 1200, MaxTrips: 2, SafeWaitSecs: 60, PortalClearTiles: 10, FarmLegs: 1, ElitePotionMin: 2}
	if t.GoldMult == 0 {
		t.GoldMult = d.GoldMult
	}
	if t.CooldownSecs == 0 {
		t.CooldownSecs = d.CooldownSecs
	}
	if t.MaxTrips == 0 {
		t.MaxTrips = d.MaxTrips
	}
	if t.SafeWaitSecs == 0 {
		t.SafeWaitSecs = d.SafeWaitSecs
	}
	if t.PortalClearTiles == 0 {
		t.PortalClearTiles = d.PortalClearTiles
	}
	if t.FarmLegs == 0 {
		t.FarmLegs = d.FarmLegs
	}
	if t.ElitePotionMin == 0 {
		t.ElitePotionMin = d.ElitePotionMin
	}
}

func (t *Trip) check() error {
	switch {
	case t.GoldMult < 1:
		return fmt.Errorf("repair: modes: trip.goldMult %.2f is under 1 (the estimate is a lower bound)", t.GoldMult)
	case t.CooldownSecs < 0 || t.MaxTrips < 1 || t.SafeWaitSecs < 0 || t.PortalClearTiles < 1 || t.FarmLegs < 0 || t.ElitePotionMin < 0:
		return fmt.Errorf("repair: modes: trip knobs out of range: %+v", *t)
	}
	return nil
}

// Need is the gold a repair wants for an estimate (a lower bound), rounded up.
func (t Trip) Need(estimate int) int { return int(math.Ceil(float64(estimate) * t.GoldMult)) }

// outcomeShowSecs is how long a finished request (done, failed, cancelled) stays on the page before the idle state returns.
const outcomeShowSecs = 1800

// Kinds of request. The button decides from live gold at the click.
const (
	KindRepair = "repair" // enough gold: make the trip
	KindFarm   = "farm"   // not enough: farm until there is, then make the trip
)

// Phases of a pending request (Request.Phase). The bot moves them; the page shows them.
const (
	PhaseQueued    = "queued"    // asked; the bot has not looked at a safe moment yet
	PhaseWaiting   = "waiting"   // waiting for a safe moment (a pack is being fought, monsters close)
	PhaseGoing     = "going"     // casting the portal or walking home
	PhaseRepairing = "repairing" // in town: sell, repair, kit
	PhaseReturning = "returning" // the repair is paid, walking back through the own portal
	PhaseFarming   = "farming"   // earning the gold first
)

// Page states of TripView.State: a phase, a finished request, or what the button would do.
const (
	StateReady    = "ready"    // idle, enough gold: the button reads Repair now
	StateFarmable = "farm"     // idle, not enough gold: Farm until repair
	StateDisabled = "disabled" // idle, the button is off (Reason says why)
	StateCancel   = "cancelled"
)

// Button is what the page may press.
type Button struct {
	Action  string `json:"action"` // request, cancel or none
	Label   string `json:"label"`
	Enabled bool   `json:"enabled"`
	Reason  string `json:"reason,omitempty"` // why it is off
}

// Progress is a farming leg's counter.
type Progress struct {
	Have   int    `json:"have"`
	Target int    `json:"target"`
	Area   string `json:"area,omitempty"`
}

// TripView is the additive `trip` field of GET /api/repair (schema 2).
type TripView struct {
	State    string    `json:"state"`
	Button   Button    `json:"button"`
	Line     string    `json:"line,omitempty"` // the one line under the buttons
	Gold     int       `json:"gold"`           // live pack gold
	Estimate int       `json:"estimate,omitempty"`
	Need     int       `json:"need,omitempty"` // estimate x goldMult
	Kind     string    `json:"kind,omitempty"`
	Trips    int       `json:"trips,omitempty"`
	MaxTrips int       `json:"maxTrips"`
	Progress *Progress `json:"progress,omitempty"`
}

// gold is the live pack gold (not the stash, not a stale reading); -1 when unknown.
func (r *Controller) gold() int {
	if r.c.Gold == nil {
		return -1
	}
	return r.c.Gold()
}

// damaged: some worn item is below full durability; true without readings.
func (r *Controller) damaged() bool {
	if r.c.Gear == nil {
		return true
	}
	for _, g := range r.c.Gear() {
		if g.Pct < 100 {
			return true
		}
	}
	return false
}

// lowest is the lowest worn durability percent (100 without readings).
func (r *Controller) lowest() int {
	low := 100
	if r.c.Gear != nil {
		for _, g := range r.c.Gear() {
			low = min(low, g.Pct)
		}
	}
	return low
}

func (r *Controller) cooldownLeftLocked() time.Duration {
	if r.st.LastEnd == 0 {
		return 0
	}
	left := time.Duration(r.t.Trip.CooldownSecs)*time.Second - r.c.Now().Sub(time.Unix(r.st.LastEnd, 0))
	return max(left, 0)
}

func mins(d time.Duration) int { return int(math.Ceil(d.Minutes())) }

// pending: the request is open (queued, waiting, going, repairing, returning or farming).
func (r *Controller) pendingLocked() bool {
	return r.st.Request != nil && r.st.Request.State == StateRequested
}

// idleButton is the button when no request is open: the live-gold decision. Locks must not be held (Gold, Gear and
// Estimate are callbacks).
func (r *Controller) idleButton() (b Button, kind string, need, gold, est int) {
	gold = r.gold()
	if r.c.Estimate != nil {
		est = r.c.Estimate()
	}
	need = r.t.Trip.Need(est)
	if m, err := r.t.Mode(r.ModeID()); err == nil && m.Never {
		return Button{Action: "none", Label: "Repair now", Reason: "a Pact forbids repairs"}, "", need, gold, est
	}
	if !r.damaged() {
		return Button{Action: "none", Label: "Repair now", Reason: "nothing worn is damaged"}, "", need, gold, est
	}
	r.mu.Lock()
	left := r.cooldownLeftLocked()
	r.mu.Unlock()
	if left > 0 {
		return Button{Action: "none", Label: "Repair now", Reason: fmt.Sprintf("the last repair request failed or was cancelled by the bot: try again in %d min", mins(left))}, "", need, gold, est
	}
	if gold < 0 || est == 0 || gold >= need {
		return Button{Action: "request", Label: "Repair now", Enabled: true}, KindRepair, need, gold, est
	}
	return Button{Action: "request", Label: "Farm until repair", Enabled: true}, KindFarm, need, gold, est
}

// TripView builds the page state. The request's phase wins; otherwise a request that ended recently shows its outcome;
// otherwise the button decides.
func (r *Controller) tripView() TripView {
	v := TripView{MaxTrips: r.t.Trip.MaxTrips, Gold: max(r.gold(), 0)}
	r.mu.Lock()
	var q Request
	have := r.st.Request != nil
	if have {
		q = *r.st.Request
	}
	r.mu.Unlock()
	if have && q.State == StateRequested {
		v.Kind, v.Trips = q.Kind, q.Trips
		v.State = q.Phase
		if v.State == "" {
			v.State = PhaseQueued
		}
		v.Button = Button{Action: "cancel", Label: "Cancel repair", Enabled: true}
		if q.Kind == KindFarm {
			v.Button.Label = "Cancel"
		}
		if r.c.Estimate != nil {
			v.Estimate = r.c.Estimate()
			v.Need = r.t.Trip.Need(v.Estimate)
		}
		if q.Phase == PhaseFarming && q.Target > 0 {
			v.Progress = &Progress{Have: q.Have, Target: q.Target, Area: q.Area}
		}
		v.Line = q.Line()
		return v
	}
	b, kind, need, gold, est := r.idleButton()
	v.Button, v.Kind, v.Need, v.Estimate, v.Gold = b, kind, need, est, max(gold, 0)
	switch {
	case !b.Enabled:
		v.State = StateDisabled
		v.Line = b.Reason
	case kind == KindFarm:
		v.State = StateFarmable
		v.Line = fmt.Sprintf("need %d more gold (have %d, estimate %d)", need-gold, gold, est)
	default:
		v.State = StateReady
	}
	if have && q.State != StateRequested && r.c.Now().Unix()-q.Done < outcomeShowSecs { // the outcome of the last request, over the idle line
		v.State, v.Line = q.State, q.Line()
	}
	return v
}

// Line is the request's own one-line status.
func (q Request) Line() string {
	switch q.State {
	case StateDone:
		return q.durLine("Repair done")
	case StateFailed:
		return "Repair failed: " + q.Note
	case StateCancel:
		if q.Note != "" {
			return "Repair cancelled: " + q.Note
		}
		return "Repair cancelled."
	}
	switch q.Phase {
	case PhaseFarming:
		if q.Target > 0 {
			return fmt.Sprintf("farming for repair: %d of %d", q.Have, q.Target)
		}
		return "farming for repair"
	case PhaseWaiting:
		return join2("Repair queued: waiting for a safe moment", q.Note)
	case PhaseGoing:
		return join2("Repair trip: going home", q.Note)
	case PhaseRepairing:
		return "Repair trip: repairing in town."
	case PhaseReturning:
		return "Repair trip: returning to the field."
	}
	return join2("Repair queued", q.Note)
}

func (q Request) durLine(s string) string {
	if q.DurBefore > 0 || q.DurAfter > 0 {
		return fmt.Sprintf("%s: durability %d%% to %d%%.", s, q.DurBefore, q.DurAfter)
	}
	return s + "."
}

func join2(a, b string) string {
	if b == "" {
		return a + "."
	}
	return a + " (" + b + ")."
}

// ---- the bot's side -----------------------------------------------------------------------------

// Active is a copy of the open request.
func (r *Controller) Active() (Request, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingLocked() {
		return Request{}, false
	}
	return *r.st.Request, true
}

// SetPhase moves the open request to a phase with a note for the Why and status lines ("" clears the note).
func (r *Controller) SetPhase(phase, note string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingLocked() || (r.st.Request.Phase == phase && r.st.Request.Note == note) {
		return
	}
	r.st.Request.Phase, r.st.Request.Note = phase, note
	_ = r.saveLocked()
}

// FarmProgress records the farming leg's counter (the page's "farming for repair: 140 of 300").
func (r *Controller) FarmProgress(have, target int, area string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingLocked() || r.st.Request.Kind != KindFarm {
		return
	}
	q := r.st.Request
	if q.Have != have || q.Target != target || q.Area != area {
		q.Have, q.Target, q.Area = have, target, area
		_ = r.saveLocked()
	}
}

// FarmReached ends the farming leg: the request becomes a repair trip, queued. It reports whether it changed.
func (r *Controller) FarmReached() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingLocked() || r.st.Request.Kind != KindFarm {
		return false
	}
	q := r.st.Request
	q.Kind, q.Phase, q.Note = KindRepair, PhaseQueued, ""
	slog.Info("repair: farm target reached, the repair trip is next", "have", q.Have, "target", q.Target)
	_ = r.saveLocked()
	return true
}

// BeginTrip starts a trip for the open request at the bot's safe moment. It refuses (and says why) when the request
// has used its trips or the cooldown since the last trip has not passed; the second case leaves the request queued.
func (r *Controller) BeginTrip(now time.Time) error {
	low := r.lowest()
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingLocked() {
		return &Refusal{"none", "no repair is requested"}
	}
	q := r.st.Request
	if q.Trips >= r.t.Trip.MaxTrips {
		r.endLocked(StateFailed, fmt.Sprintf("%d trips did not repair the gear", q.Trips), now, false)
		return &Refusal{"trips", q.Note}
	}
	if q.Trips > 0 {
		if left := time.Duration(r.t.Trip.CooldownSecs)*time.Second - now.Sub(time.Unix(q.TripAt, 0)); left > 0 {
			q.Phase, q.Note = PhaseQueued, fmt.Sprintf("next try in %d min", mins(left))
			_ = r.saveLocked()
			return &Refusal{"cooldown", q.Note}
		}
	}
	q.Trips++
	q.TripAt, q.DurBefore, q.DurAfter = now.Unix(), low, 0
	q.Phase, q.Note = PhaseGoing, ""
	slog.Info("repair trip begins", "trip", q.Trips, "of", r.t.Trip.MaxTrips, "durabilityPct", low)
	return r.saveLocked()
}

// EndTrip is the trip's close. repaired true: the request is done (durability before and after are logged). Otherwise
// the request stays open for another trip, or fails when it has used them all. short true: the smith asked for more
// gold than the pack had (the estimate grew, or the gold went elsewhere): one more farm leg, then a cancel.
func (r *Controller) EndTrip(now time.Time, repaired, short bool, reason string) {
	low := r.lowest()
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingLocked() {
		return
	}
	q := r.st.Request
	q.DurAfter = low
	switch {
	case repaired:
		slog.Info("repair trip done", "before", q.DurBefore, "after", low, "trips", q.Trips)
		r.endLocked(StateDone, "", now, false)
	case short && q.FarmLegs < r.t.Trip.FarmLegs:
		q.FarmLegs++
		q.Kind, q.Phase, q.Note, q.Have, q.Target = KindFarm, PhaseFarming, "", 0, 0
		slog.Info("repair trip: gold short at the smith, farming first", "reason", reason)
		_ = r.saveLocked()
	case short:
		slog.Info("repair request cancelled: still unpayable after a farm leg", "reason", reason)
		r.endLocked(StateCancel, "still short of gold after farming ("+reason+")", now, true)
	case q.Trips >= r.t.Trip.MaxTrips:
		r.endLocked(StateFailed, firstNonEmpty(reason, "the smith could not be reached"), now, false)
	default:
		q.Phase, q.Note = PhaseQueued, firstNonEmpty(reason, "will try again")
		_ = r.saveLocked()
	}
}

// CancelAuto withdraws the open request on the bot's own decision, with the reason shown to the player.
func (r *Controller) CancelAuto(reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.pendingLocked() {
		return
	}
	slog.Info("repair request cancelled by the bot", "reason", reason)
	r.endLocked(StateCancel, reason, r.c.Now(), true)
}

// endLocked closes the open request. Only a failure or a cancel by the bot starts the cooldown for a new request (the
// loop guard); a repair that worked or a click the player took back does not.
func (r *Controller) endLocked(state, note string, now time.Time, auto bool) {
	q := r.st.Request
	q.State, q.Note, q.Phase, q.Auto = state, note, "", auto
	q.Done = now.Unix()
	q.Short, q.Need = 0, 0
	if state == StateFailed || auto {
		r.st.LastEnd = now.Unix()
	}
	_ = r.saveLocked()
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

// Table returns the trip knobs.
func (r *Controller) TripCfg() Trip { return r.t.Trip }
