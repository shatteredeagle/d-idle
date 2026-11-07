// Package watch is the backend of the player screen (ADR 0016 items 20 to 23, docs/research/watch-screen-design.md).
//
// A Hub adapts internal/decide to the screen: it subscribes to the store's events (no polling), implements the
// ladder.Sink and ladder.AnswerSource interfaces so the real bot can be connected without screen changes, keeps the
// parts the store does not hold (idle dialogue, incidents, orders, the activity timeline for uptime) and builds the
// JSON views. Two http.Handlers expose it: ViewHandler (GET only, mounted on the read-only d2dview) and AnswerHandler
// (POST only, a separate listener that carries every write).
package watch

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// Config builds a Hub. Store, Cat and Clock are required; the Clock must be the store's own clock.
type Config struct {
	Store *decide.Store
	Cat   *decide.Catalogue
	Clock decide.Clock
	// Base is the virtual time shown as 0 (default: the clock at New). Display times are relative to it.
	Base time.Duration
	// Areas are the names the player may order the hero to; empty disables "go to" orders.
	Areas []string
	// Orders takes validated orders; nil keeps them on the screen only.
	Orders OrderSink
	// AreaGuard says why the hero's own guard closes the game area id now ("" = open or not guarded). It must only
	// read: no question, no log, no game access. nil: every area is open. The map shows it as "too hard now".
	AreaGuard func(areaID int) string
	// WaitForSink hides a thought until the ladder's Sink reports it shown (the ladder holds questions during a
	// fight). Leave false when no ladder is connected.
	WaitForSink bool
	// Speed is virtual seconds per wall second, used by the page to run the countdown between polls (default 1).
	Speed float64
	// WindowSec is the span of the recent uptime figure (default 600).
	WindowSec int
	// FeedKeep is how many idle, story and incident items stay in the feed (default 12).
	FeedKeep int
	// AnswerBase is the base URL of the answer route the page posts to; empty: same host, AnswerPort.
	AnswerBase string
	AnswerPort int
	// FrameURL is the viewer's frame stream for the game area ("/stream.mjpg"); empty when no game is attached.
	FrameURL string
}

type thoughtRec struct {
	info      decide.ThoughtInfo
	shown     bool
	meanwhile string // the answer id taken meanwhile
}

type extra struct {
	id      string
	t       time.Duration
	kind    string
	text    string
	key     string
	thought int
	rules   []string
}

// Hub is safe for concurrent use. Lock order: the store's lock, then the Hub's (store events call in), so no Hub
// method holds its own lock while it calls the store.
type Hub struct {
	cfg   Config
	base  time.Duration
	video *Video                          // nil: no game video
	cur   atomic.Pointer[currency.Wallet] // nil: no currencies (vanilla; internal/watch/currency.go)

	mu      sync.Mutex
	seq     uint64
	ch      chan struct{}
	recs    map[int]*thoughtRec
	extras  []extra
	nextX   int
	orders  []*Order
	nextO   int
	up      uptimeTracker
	hero    HeroInfo
	cleared map[int]bool  // game area ids known to be cleared (MarkCleared)
	xp0     map[int]int64 // order id -> hero xp when the order started
}

// New makes a Hub and subscribes it to the store.
func New(c Config) (*Hub, error) {
	if c.Store == nil || c.Cat == nil || c.Clock == nil {
		return nil, errors.New("watch: store, catalogue and clock are required")
	}
	if c.Speed <= 0 {
		c.Speed = 1
	}
	if c.WindowSec <= 0 {
		c.WindowSec = 600
	}
	if c.FeedKeep <= 0 {
		c.FeedKeep = 12
	}
	h := &Hub{cfg: c, base: c.Base, ch: make(chan struct{}), recs: map[int]*thoughtRec{}, nextO: 1}
	if c.Base == 0 {
		h.base = c.Clock.Now()
	}
	c.Store.Subscribe(h.onEvent)
	// A thought opened before the Hub existed is picked up here (the store's events are not replayed).
	for _, t := range c.Store.OpenThoughts() {
		if info, ok := c.Store.Thought(t.ID); ok {
			h.recs[t.ID] = &thoughtRec{info: info, shown: true, meanwhile: h.defaultFor(info.Key)}
		}
	}
	return h, nil
}

func (h *Hub) rel(t time.Duration) float64 { return secs(t - h.base) }

// SetAnswerPort and SetFrameURL tell the page where the answer route and the frame stream are. Call them before the
// handlers serve; they are plain configuration, not state.
func (h *Hub) SetAnswerPort(p int)  { h.cfg.AnswerPort = p }
func (h *Hub) SetFrameURL(u string) { h.cfg.FrameURL = u }

// notifyLocked wakes everybody waiting on Changed.
func (h *Hub) notifyLocked() {
	h.seq++
	close(h.ch)
	h.ch = make(chan struct{})
}

// Seq is the change counter every GET response carries.
func (h *Hub) Seq() uint64 { h.mu.Lock(); defer h.mu.Unlock(); return h.seq }

// Changed returns the current sequence and a channel closed at the next change.
func (h *Hub) Changed() (uint64, <-chan struct{}) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.seq, h.ch
}

func (h *Hub) defaultFor(key string) string {
	ctx, err := decide.ParseKey(key)
	if err != nil {
		return ""
	}
	d, _ := h.cfg.Cat.Default(ctx)
	return d
}

// onEvent runs inside the store: it records and wakes, it never calls the store.
func (h *Hub) onEvent(e decide.Event) {
	h.mu.Lock()
	defer h.mu.Unlock()
	switch e.Kind {
	case decide.EvThoughtOpened, decide.EvConfirmOpened:
		if e.Info != nil {
			shown := !h.cfg.WaitForSink || e.Kind == decide.EvConfirmOpened
			h.recs[e.Thought] = &thoughtRec{info: *e.Info, shown: shown, meanwhile: h.defaultFor(e.Info.Key)}
		}
	case decide.EvRuleLearned:
		h.addExtraLocked(extra{kind: "learned", key: e.Key, text: h.learnedText(e.Key, e.Answer)})
	case decide.EvLateAnswer:
		q := h.questionOf(e.Thought, e.Key)
		h.addExtraLocked(extra{kind: "late", key: e.Key, thought: e.Thought,
			text: fmt.Sprintf("Too late: you answered \"%s\" after the question faded, so I carried on and nothing was counted.", q)})
	case decide.EvThoughtFaded, decide.EvRevoked:
	}
	h.notifyLocked()
}

type snapshot struct {
	seq    uint64
	recs   map[int]thoughtRec
	extras []extra
}

// snap copies the Hub's state so a view can then call the store without holding the Hub lock.
func (h *Hub) snap() snapshot {
	h.mu.Lock()
	defer h.mu.Unlock()
	s := snapshot{seq: h.seq, recs: make(map[int]thoughtRec, len(h.recs)), extras: append([]extra(nil), h.extras...)}
	for id, r := range h.recs {
		s.recs[id] = *r
	}
	return s
}

func (h *Hub) questionOf(id int, key string) string {
	if r := h.recs[id]; r != nil {
		return r.info.Question
	}
	return h.questionFor(key)
}

func (h *Hub) questionFor(key string) string {
	if ctx, err := decide.ParseKey(key); err == nil {
		if p, err := h.cfg.Cat.Point(ctx.Point); err == nil {
			return p.Question
		}
	}
	return "a question"
}

func (h *Hub) answerLabel(key, answer string) string {
	ctx, err := decide.ParseKey(key)
	if err != nil {
		return answer
	}
	return h.pointAnswerLabel(ctx.Point, answer)
}

func (h *Hub) pointAnswerLabel(point, answer string) string {
	if answer == decide.NotSure {
		return "not sure, you decide"
	}
	if p, err := h.cfg.Cat.Point(point); err == nil {
		for _, a := range p.Answers {
			if a.ID == answer {
				return a.Label
			}
		}
	}
	return answer
}

func (h *Hub) learnedText(key, answer string) string {
	return fmt.Sprintf("Rule learned: %s, I will now always \"%s\" on my own. See the rules list to revoke it.",
		PlainContext(key), h.answerLabel(key, answer))
}

func (h *Hub) addExtraLocked(x extra) {
	h.nextX++
	x.id = fmt.Sprintf("X%d", h.nextX)
	if x.t == 0 {
		x.t = h.cfg.Clock.Now()
	}
	h.extras = append(h.extras, x)
}

// Sink implements ladder.Sink: the ladder reports what it shows. The store's own events already carry the facts, so
// the adapter only needs "thought" (the question is now allowed on the screen, with the action taken meanwhile).
func (h *Hub) Event(kind string, v ladder.ThoughtView, detail string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if kind == "thought" {
		if r := h.recs[v.ID]; r != nil {
			r.shown = true
			if v.Default != "" {
				r.meanwhile = v.Default
			}
		}
	}
	h.notifyLocked()
}

// Reply implements ladder.AnswerSource. The ladder asks once, at the moment a thought is first shown, and a person
// answers later, so the adapter never answers here: the answer arrives on the answer route and goes straight to the
// store (the source of truth). See "API needs" in the design doc.
func (h *Hub) Reply(ladder.ThoughtView) ladder.Reply { return ladder.Reply{} }

var (
	_ ladder.Sink         = (*Hub)(nil)
	_ ladder.AnswerSource = (*Hub)(nil)
)

// Errors of the answer route; the handler maps each to a status.
var (
	ErrUnknownThought  = errors.New("watch: no such thought")
	ErrAlreadyAnswered = errors.New("watch: thought already answered")
	ErrThoughtFaded    = errors.New("watch: thought faded")
	ErrBadChoice       = errors.New("watch: not one of the offered answers")
	ErrWrongThought    = errors.New("watch: wrong kind of thought for this action")
	ErrNoRule          = errors.New("watch: no such rule")
)

// closedState says what became of a thought that is no longer open, from the store's log: "answered", "faded" or "".
func (h *Hub) closedState(id int) string {
	for _, e := range h.cfg.Store.Log() {
		if e.Thought != id {
			continue
		}
		switch e.Op {
		case decide.OpAnswer, decide.OpConfirm:
			return "answered"
		case decide.OpFade:
			return "faded"
		}
	}
	return ""
}

func (h *Hub) missing(id int) error {
	switch h.closedState(id) {
	case "answered":
		return fmt.Errorf("%w: %d", ErrAlreadyAnswered, id)
	case "faded":
		return fmt.Errorf("%w: %d", ErrThoughtFaded, id)
	}
	return fmt.Errorf("%w: %d", ErrUnknownThought, id)
}

// Answer is the answer route's action: one of the offered answers or "notsure" to an open question.
func (h *Hub) Answer(id int, answer string) error {
	info, ok := h.cfg.Store.Thought(id)
	if !ok {
		return h.missing(id)
	}
	if info.Confirm {
		return fmt.Errorf("%w: thought %d is a confirm; reply yes or no", ErrWrongThought, id)
	}
	if answer != decide.NotSure && !hasChoice(info.Choices, answer) {
		return fmt.Errorf("%w: %q", ErrBadChoice, safeText(answer, 40))
	}
	return h.mapStore(id, h.cfg.Store.Answer(id, answer))
}

// Confirm answers a confirm thought yes or no.
func (h *Hub) Confirm(id int, yes bool) error {
	info, ok := h.cfg.Store.Thought(id)
	if !ok {
		return h.missing(id)
	}
	if !info.Confirm {
		return fmt.Errorf("%w: thought %d is a question; send one of its answers", ErrWrongThought, id)
	}
	return h.mapStore(id, h.cfg.Store.Confirm(id, yes))
}

func (h *Hub) mapStore(id int, err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, decide.ErrFaded):
		return fmt.Errorf("%w: %d", ErrThoughtFaded, id)
	case errors.Is(err, decide.ErrNoThought):
		return h.missing(id)
	case errors.Is(err, decide.ErrBadAnswer):
		return fmt.Errorf("%w: %v", ErrBadChoice, err)
	case errors.Is(err, decide.ErrWrongKind):
		return fmt.Errorf("%w: %v", ErrWrongThought, err)
	}
	return err
}

func hasChoice(cs []decide.Answer, id string) bool {
	for _, c := range cs {
		if c.ID == id {
			return true
		}
	}
	return false
}

// Revoke is the player's revoke of one rule; only the player's route calls it.
func (h *Hub) Revoke(key string) error {
	err := h.cfg.Store.Revoke(key)
	if errors.Is(err, decide.ErrNoRule) {
		return fmt.Errorf("%w: %q", ErrNoRule, safeText(key, 120))
	}
	return err
}

// RevokeAll revokes every live rule and reports how many there were.
func (h *Hub) RevokeAll() (int, error) {
	n, _, _ := h.cfg.Store.Stats()
	return n, h.cfg.Store.RevokeAll()
}

// Tick fades the thoughts whose time is up. The bot side calls it (the ladder does on every Decide); a GET never does,
// so the viewer stays read-only.
func (h *Hub) Tick() (int, error) { return h.cfg.Store.Tick() }

// Activity is called by the bot when what the hero is doing changes: kind is one of the Kind constants, label a short
// plain text ("fighting in Blood Moor"). Uptime is derived from this timeline.
func (h *Hub) Activity(kind, label string) {
	if !activityKinds[kind] {
		kind = KindStall
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.up.set(h.cfg.Clock.Now(), kind, safeText(label, 80))
	h.notifyLocked()
}

// Note puts a line in the feed and the journal: kind "idle" (the hero talks), "story" or "incident". ruleKeys name the
// rules in force; the screen offers a revoke next to each (ADR 0016 item 23). The text is plain text.
func (h *Hub) Note(kind, text string, ruleKeys ...string) {
	switch kind {
	case "idle", "story", "incident":
	default:
		kind = "idle"
	}
	text = safeText(text, 400)
	if text == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	h.addExtraLocked(extra{kind: kind, text: text, rules: append([]string(nil), ruleKeys...)})
	h.notifyLocked()
}

// Incident is Note("incident", ...).
func (h *Hub) Incident(text string, ruleKeys ...string) { h.Note("incident", text, ruleKeys...) }

// Say is Note("idle", ...): the hero's idle dialogue.
func (h *Hub) Say(text string) { h.Note("idle", text) }

// OrderRequest is what the answer route passes to Give.
type OrderRequest struct {
	Kind    string
	Area    string
	Minutes int
	Forever bool // no time limit
	Replace bool // "Go now" in slice 1: every open order is cancelled first
}

// GiveOrder is Give for a timed order that queues like before.
func (h *Hub) GiveOrder(kind, area string, minutes int) (Order, error) {
	return h.Give(OrderRequest{Kind: kind, Area: area, Minutes: minutes})
}

// Give validates and records an order, then hands it to the bot's OrderSink. The order only asks: the hero's own guard
// still decides when the walk starts, and a refusal ends the order failed with the guard's reason (OrderState).
func (h *Hub) Give(r OrderRequest) (Order, error) {
	o, err := validateOrder(r.Kind, r.Area, r.Minutes, r.Forever, h.cfg.Areas)
	if err != nil {
		return o, err
	}
	if r.Replace {
		for _, id := range h.openIDs() {
			if err := h.CancelOrder(id); err != nil && !errors.Is(err, errOrderClosed) {
				return o, err
			}
		}
	}
	h.mu.Lock()
	open := 0
	for _, x := range h.orders {
		if x.State == OrderQueued || x.State == OrderActive {
			open++
		}
	}
	if open >= MaxOpenOrders {
		h.mu.Unlock()
		return o, ErrOrdersFull
	}
	o.ID, o.State, o.GivenAt = h.nextO, OrderQueued, h.rel(h.cfg.Clock.Now())
	h.nextO++
	rec := o
	h.orders = append(h.orders, &rec)
	h.addExtraLocked(extra{kind: "order", text: "Order given: " + o.Text})
	h.notifyLocked()
	h.mu.Unlock()
	if h.cfg.Orders != nil {
		if err := h.cfg.Orders.Order(o); err != nil {
			h.OrderState(o.ID, OrderFailed, err.Error())
			return o, fmt.Errorf("%w: %v", ErrBadOrder, err)
		}
	}
	return o, nil
}

func (h *Hub) openIDs() []int {
	h.mu.Lock()
	defer h.mu.Unlock()
	var ids []int
	for _, x := range h.orders {
		if x.State == OrderQueued || x.State == OrderActive {
			ids = append(ids, x.ID)
		}
	}
	return ids
}

// CancelOrder cancels an open order.
func (h *Hub) CancelOrder(id int) error {
	h.mu.Lock()
	var o *Order
	for _, x := range h.orders {
		if x.ID == id {
			o = x
		}
	}
	if o == nil {
		h.mu.Unlock()
		return fmt.Errorf("%w: %d", ErrNoOrder, id)
	}
	if o.State != OrderQueued && o.State != OrderActive {
		h.mu.Unlock()
		return errOrderClosed
	}
	h.mu.Unlock()
	if h.cfg.Orders != nil {
		if err := h.cfg.Orders.Cancel(id); err != nil {
			return err
		}
	}
	h.OrderState(id, OrderCancelled, "")
	return nil
}

// OrderState is called by the bot when an order starts, ends or fails. It also writes the journal line and keeps the
// progress (start time, xp gained) of the Hero-now panel.
func (h *Hub) OrderState(id int, state, note string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, o := range h.orders {
		if o.ID != id || o.State == state {
			continue
		}
		o.State, o.Note = state, safeText(note, 200)
		switch state {
		case OrderActive:
			o.StartedAt = h.rel(h.cfg.Clock.Now())
			if h.xp0 == nil {
				h.xp0 = map[int]int64{}
			}
			h.xp0[id] = h.hero.XP
		case OrderDone, OrderCancelled, OrderFailed:
			if x0, ok := h.xp0[id]; ok {
				o.XPGained = max(0, h.hero.XP-x0)
				delete(h.xp0, id)
			}
		}
		h.addExtraLocked(extra{kind: "order", text: orderLine(*o, state)})
		h.notifyLocked()
	}
}

// orderLine is the journal line of an order change.
func orderLine(o Order, state string) string {
	switch state {
	case OrderActive:
		return "Order started: " + o.Text
	case OrderDone:
		return "Order done: " + o.Text + progressText(o)
	case OrderCancelled:
		return "Order cancelled: " + o.Text + progressText(o)
	}
	if reason, ok := guardRefusal(o.Note); ok {
		return "Order refused by the hero's guard: " + reason
	}
	if o.Note != "" {
		return "Order failed: " + o.Text + " (" + o.Note + ")"
	}
	return "Order failed: " + o.Text
}

func progressText(o Order) string {
	switch {
	case o.Kills > 0 && o.XPGained > 0:
		return fmt.Sprintf(" (%d kills, +%d xp)", o.Kills, o.XPGained)
	case o.Kills > 0:
		return fmt.Sprintf(" (%d kills)", o.Kills)
	}
	return ""
}

// guardRefusal says whether a failed order's note is the hero's guard turning the walk down, and returns the reason.
// U: matches the wording of bot.errAreaGated ("area not allowed now: AREA (ID): route guard (WHY)") as the runner
// reports it in the goal's failure message; a different wording just reads "Order failed".
func guardRefusal(note string) (string, bool) {
	const marker = "area not allowed now"
	i := strings.Index(note, marker)
	if i < 0 {
		return "", false
	}
	return strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(note[i:], marker), ":")), true
}

// NoteKill counts a kill for the open order (the bot calls it on every kill event).
func (h *Hub) NoteKill() {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, o := range h.orders {
		if o.State == OrderActive {
			o.Kills++
		}
	}
}
