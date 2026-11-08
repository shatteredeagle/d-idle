package watch

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/goals"
)

// The order queue (TASK-161, docs/unlock-orderqueue.md): queue up to three orders that run in turn. The sequencer
// sits in front of the orders path: it hands ONE order at a time to the same placer as POST /api/orders (so the charter
// flow, the 5-open limit and the bot's own route guard apply unchanged) and places the next when that one ends. How many
// orders may wait is the number of open slot rows of the unlock ladder (orderqueue.json lists them; the first opens after
// the Den of Evil, later ones after the Kashya / Charsi / Cain events, or for tally through the ladder's own buy route).
// A queue never overrides safety: the guard still refuses on arrival, and the queue stops placing when the hero dies or
// the Pact / low-life pause is on.

const (
	// OrderQueueID is the first slot row and the gate of the whole feature.
	OrderQueueID = "orderqueue"
	// OrderQueueRoute is GET (viewer) and POST (answer listener): {action: add|cancel|resume|clear}.
	OrderQueueRoute = "/api/orderqueue"

	orderQueueSchema = "watch.orderqueue/1"
)

// Entry states.
const (
	QueueWaiting   = "waiting"
	QueueRunning   = "running"
	QueueDone      = "done"
	QueueFailed    = "failed"
	QueueCancelled = "cancelled"
)

// QueueTable is orderqueue.json.
type QueueTable struct {
	Schema        string   `json:"schema"`
	Doc           string   `json:"doc,omitempty"`
	Slots         []string `json:"slots"`
	MaxUntilLevel int      `json:"maxUntilLevel"`
}

//go:embed orderqueue.json
var shippedQueue []byte

// ShippedQueueTable parses the embedded table.
func ShippedQueueTable() (*QueueTable, error) { return ParseQueueTable(shippedQueue) }

// ParseQueueTable checks the table; errors name the field.
func ParseQueueTable(b []byte) (*QueueTable, error) {
	var t QueueTable
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&t); err != nil {
		return nil, fmt.Errorf("order queue table: %w", err)
	}
	if t.Schema != orderQueueSchema {
		return nil, fmt.Errorf("order queue table: schema %q, want %q", t.Schema, orderQueueSchema)
	}
	if len(t.Slots) == 0 || t.Slots[0] != OrderQueueID {
		return nil, fmt.Errorf("order queue table: the first slot must be %q", OrderQueueID)
	}
	seen := map[string]bool{}
	for _, s := range t.Slots {
		if seen[s] {
			return nil, fmt.Errorf("order queue table: duplicate slot %q", s)
		}
		seen[s] = true
	}
	if t.MaxUntilLevel < 2 {
		return nil, errors.New("order queue table: maxUntilLevel must be 2 or more")
	}
	return &t, nil
}

// QueueEntry is one queued order.
type QueueEntry struct {
	ID         int    `json:"id"`
	Key        string `json:"key,omitempty"`
	Kind       string `json:"kind"`
	Area       string `json:"area,omitempty"`
	Minutes    int    `json:"minutes,omitempty"`
	Forever    bool   `json:"forever,omitempty"`
	UntilLevel int    `json:"untilLevel,omitempty"`
	Approve    bool   `json:"approve,omitempty"` // the player accepted the charter of an above-range order
	Text       string `json:"text"`
	State      string `json:"state"`
	OrderID    int    `json:"orderId,omitempty"`
	Note       string `json:"note,omitempty"`
}

// OrderQueueConfig builds an OrderQueue. Quote nil: no charter check at add time (the placer still decides). Halt nil:
// nothing halts the queue but a death; the bot passes the Pact / low-life pause reason ("" = running).
type OrderQueueConfig struct {
	Table *QueueTable // nil: the shipped table
	Quote func(area string) (currency.Quote, error)
	Halt  func() string
}

// OrderQueue is the sequencer. Safe for concurrent use.
type OrderQueue struct {
	u     *Unlocks
	hub   *Hub
	table *QueueTable
	cfg   OrderQueueConfig

	mu      sync.Mutex
	entries []*QueueEntry
	next    int
	stopped string // why the queue stopped placing ("" = running)
}

// NewOrderQueue checks the ladder: every slot row of the table must exist, an unknown row is an error naming it.
func NewOrderQueue(u *Unlocks, h *Hub, c OrderQueueConfig) (*OrderQueue, error) {
	if h == nil || u == nil {
		return nil, errors.New("watch: the order queue needs a hub and the unlock ladder")
	}
	t := c.Table
	if t == nil {
		var err error
		if t, err = ShippedQueueTable(); err != nil {
			return nil, err
		}
	}
	for _, s := range t.Slots {
		if _, err := u.Engine.Check(s); err != nil {
			return nil, err
		}
	}
	return &OrderQueue{u: u, hub: h, table: t, cfg: c, next: 1}, nil
}

// Capacity is how many orders may wait: the open slot rows, and none while the first row is shut.
func (q *OrderQueue) Capacity() int {
	if !q.u.Engine.Allowed(q.table.Slots[0]) {
		return 0
	}
	n := 0
	for _, s := range q.table.Slots {
		if q.u.Engine.Allowed(s) {
			n++
		}
	}
	return n
}

func (q *OrderQueue) journal(text string) {
	h := q.hub
	h.mu.Lock()
	h.addExtraLocked(extra{kind: "order", text: safeText(text, 400)})
	h.notifyLocked()
	h.mu.Unlock()
}

func (q *OrderQueue) waitingLocked() int {
	n := 0
	for _, e := range q.entries {
		if e.State == QueueWaiting {
			n++
		}
	}
	return n
}

func (q *OrderQueue) runningLocked() *QueueEntry {
	for _, e := range q.entries {
		if e.State == QueueRunning {
			return e
		}
	}
	return nil
}

// Errors from the queue routes.
var (
	ErrQueueLocked = errors.New("watch: the order queue is not unlocked yet")
	ErrQueueFull   = errors.New("watch: every open queue slot is taken")
)

// Add queues an order. key makes a retry harmless: a key already queued returns that entry. An order above the zone
// range needs approve, like POST /api/orders; the charter is paid when the order is placed, not now.
func (q *OrderQueue) Add(kind, area string, minutes int, forever bool, untilLevel int, approve bool, key string) (QueueEntry, bool, error) {
	capN := q.Capacity()
	if capN == 0 {
		return QueueEntry{}, false, ErrQueueLocked
	}
	key = safeText(key, 60)
	q.mu.Lock()
	defer q.mu.Unlock()
	if key != "" {
		for _, e := range q.entries {
			if e.Key == key {
				return *e, true, nil
			}
		}
	}
	if q.waitingLocked() >= capN {
		return QueueEntry{}, false, fmt.Errorf("%w (%d)", ErrQueueFull, capN)
	}
	lvl := q.hub.heroNow().Level
	if untilLevel != 0 {
		switch {
		case kind == OrderHome:
			return QueueEntry{}, false, fmt.Errorf("%w: rest in town takes no level", ErrBadOrder)
		case forever || minutes != 0:
			return QueueEntry{}, false, fmt.Errorf("%w: until-level takes no minutes and no until-you-cancel", ErrBadOrder)
		case untilLevel < 2 || untilLevel > q.table.MaxUntilLevel:
			return QueueEntry{}, false, fmt.Errorf("%w: level must be 2 to %d", ErrBadOrder, q.table.MaxUntilLevel)
		case lvl > 0 && untilLevel <= lvl:
			return QueueEntry{}, false, fmt.Errorf("%w: the hero is already level %d", ErrBadOrder, lvl)
		}
		forever = true // the sequencer ends the order itself at the level
	}
	o, err := validateOrder(kind, area, minutes, forever, q.hub.cfg.Areas)
	if err != nil {
		return QueueEntry{}, false, err
	}
	if untilLevel != 0 {
		o.Text = strings.Replace(o.Text, "until you cancel", fmt.Sprintf("until level %d", untilLevel), 1)
	}
	if o.Kind == OrderGo && q.cfg.Quote != nil {
		qt, err := q.cfg.Quote(o.Area)
		if err != nil {
			return QueueEntry{}, false, err
		}
		if qt.Price > 0 && !approve {
			return QueueEntry{}, false, fmt.Errorf("%w: %s", currency.ErrCharterNeeded, qt.Text)
		}
	}
	e := &QueueEntry{ID: q.next, Key: key, Kind: o.Kind, Area: o.Area, Minutes: o.Minutes, Forever: o.Forever, UntilLevel: untilLevel,
		Approve: approve, Text: o.Text, State: QueueWaiting}
	q.next++
	q.entries = append(q.entries, e)
	q.journal("Queued: " + e.Text)
	return *e, false, nil
}

// Cancel drops a waiting entry, or cancels the running one (the hero's order is cancelled too and the queue goes on).
func (q *OrderQueue) Cancel(id int) error {
	q.mu.Lock()
	var e *QueueEntry
	for _, x := range q.entries {
		if x.ID == id {
			e = x
		}
	}
	if e == nil {
		q.mu.Unlock()
		return fmt.Errorf("%w: queue entry %d", ErrNoOrder, id)
	}
	switch e.State {
	case QueueWaiting:
		e.State = QueueCancelled
		q.journal("Queue entry cancelled: " + e.Text)
		q.mu.Unlock()
		return nil
	case QueueRunning:
		e.State, e.Note = QueueCancelled, "cancelled by you"
		oid := e.OrderID
		q.journal("Queue entry cancelled: " + e.Text)
		q.mu.Unlock()
		if err := q.hub.CancelOrder(oid); err != nil && !errors.Is(err, errOrderClosed) {
			return err
		}
		return nil
	}
	q.mu.Unlock()
	return errOrderClosed
}

// Clear cancels every waiting entry (the running one stays).
func (q *OrderQueue) Clear() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	n := 0
	for _, e := range q.entries {
		if e.State == QueueWaiting {
			e.State = QueueCancelled
			n++
		}
	}
	if n > 0 {
		q.journal(fmt.Sprintf("Queue cleared: %d order%s dropped", n, riskS(n)))
	}
	return n
}

// Resume lets a stopped queue place orders again. It stops again at once if the halt reason still holds.
func (q *OrderQueue) Resume() {
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.stopped != "" {
		q.stopped = ""
		q.journal("Queue resumed")
	}
}

func (q *OrderQueue) stopLocked(why string) {
	if q.stopped == "" {
		q.stopped = why
		q.journal("Queue stopped: " + why)
	}
}

// OnGoalEvent stops the queue when the hero dies (the next order must be given again by a person).
func (q *OrderQueue) OnGoalEvent(ev goals.Event) {
	if ev.Type != goals.EvDeath {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	q.stopLocked("the hero died")
}

func (h *Hub) orderStateOf(id int) (string, string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for _, o := range h.orders {
		if o.ID == id {
			return o.State, o.Note
		}
	}
	return "", ""
}

// Tick advances the sequence (call it from the loop that polls the status, Fun.Settle).
func (q *OrderQueue) Tick() {
	if q.Capacity() == 0 {
		return
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.cfg.Halt != nil {
		if why := q.cfg.Halt(); why != "" {
			q.stopLocked(why)
		}
	}
	lvl := q.hub.heroNow().Level
	if e := q.runningLocked(); e != nil {
		st, note := q.hub.orderStateOf(e.OrderID)
		switch st {
		case OrderQueued, OrderActive:
			if e.UntilLevel > 0 && lvl >= e.UntilLevel {
				e.State, e.Note = QueueDone, fmt.Sprintf("reached level %d", lvl)
				q.journal(fmt.Sprintf("Queue: %s ended, the hero reached level %d", e.Text, lvl))
				q.mu.Unlock()
				_ = q.hub.CancelOrder(e.OrderID) // a second Tick sees the order cancelled but the entry already done
				q.mu.Lock()
			}
			if q.runningLocked() != nil {
				return
			}
		case OrderDone:
			e.State = QueueDone
			q.journal("Queue: " + e.Text + " is done")
		case OrderFailed:
			e.State, e.Note = QueueFailed, note
			q.journal("Queue: " + e.Text + " failed" + noteSuffix(note))
		default: // cancelled outside the queue (the player, "Go now", a proposal): hands off, the queue stops
			e.State, e.Note = QueueCancelled, "cancelled"
			q.stopLocked("the running order was cancelled; press resume to go on")
		}
	}
	for q.stopped == "" && q.runningLocked() == nil && len(q.hub.openIDs()) == 0 {
		var head *QueueEntry
		for _, e := range q.entries {
			if e.State == QueueWaiting {
				head = e
				break
			}
		}
		if head == nil || lvl <= 0 {
			return
		}
		o, _, err := q.hub.place(OrderRequest{Kind: head.Kind, Area: head.Area, Minutes: head.Minutes, Forever: head.Forever}, head.Approve)
		if err != nil {
			head.State, head.Note = QueueFailed, safeText(err.Error(), 200)
			q.journal("Queue: " + head.Text + " was not placed" + noteSuffix(head.Note))
			continue
		}
		head.State, head.OrderID = QueueRunning, o.ID
		q.journal("Queue: next order, " + head.Text)
	}
}

func noteSuffix(n string) string {
	if n == "" {
		return ""
	}
	return " (" + safeText(n, 120) + ")"
}

// OrderQueueSlot is one slot row as the viewer sees it.
type OrderQueueSlot struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Open   bool   `json:"open"`
	Teaser string `json:"teaser,omitempty"`
	Price  int64  `json:"price,omitempty"` // tally price of a slot that can be bought (POST /api/unlocks/buy {id})
}

// OrderQueueView is GET /api/orderqueue. Locked: the teaser of the first slot only.
type OrderQueueView struct {
	Schema   string           `json:"schema"`
	Open     bool             `json:"open"`
	Title    string           `json:"title"`
	Teaser   string           `json:"teaser,omitempty"`
	Capacity int              `json:"capacity"`
	Slots    []OrderQueueSlot `json:"slots,omitempty"`
	Entries  []QueueEntry     `json:"entries"`
	Stopped  string           `json:"stopped,omitempty"`
	Balance  *int64           `json:"balance,omitempty"`
}

// View is the payload.
func (q *OrderQueue) View() OrderQueueView {
	first, _ := q.u.state(q.table.Slots[0])
	v := OrderQueueView{Schema: orderQueueSchema, Open: first.Open, Title: first.Title, Entries: []QueueEntry{}}
	if !first.Open {
		v.Teaser = first.Teaser
		return v
	}
	v.Capacity = q.Capacity()
	for _, id := range q.table.Slots {
		s, _ := q.u.state(id)
		sl := OrderQueueSlot{ID: id, Title: s.Title, Open: s.Open}
		if !s.Open {
			sl.Teaser, sl.Price = s.Teaser, s.Price
		}
		v.Slots = append(v.Slots, sl)
	}
	if b, ok := q.u.Spend.(interface{ Spendable() (int64, error) }); ok {
		if n, err := b.Spendable(); err == nil {
			v.Balance = &n
		}
	}
	q.mu.Lock()
	defer q.mu.Unlock()
	for _, e := range q.entries {
		v.Entries = append(v.Entries, *e)
	}
	v.Stopped = q.stopped
	return v
}

// Handler serves GET /api/orderqueue.
func (q *OrderQueue) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, q.View())
	})
}

type queueBody struct {
	Action  string `json:"action"`
	Kind    string `json:"kind"`
	Area    string `json:"area"`
	Minutes int    `json:"minutes"`
	Until   string `json:"until"` // "" or minutes, "forever", "level"
	Level   int    `json:"level"`
	Approve bool   `json:"approve"`
	Key     string `json:"key"`
	ID      int    `json:"id"`
}

type queueReply struct {
	OK        bool            `json:"ok"`
	Code      string          `json:"code,omitempty"`
	Error     string          `json:"error,omitempty"`
	Entry     *QueueEntry     `json:"entry,omitempty"`
	Duplicate bool            `json:"duplicate,omitempty"`
	Charter   *Charter        `json:"charter,omitempty"`
	Queue     *OrderQueueView `json:"queue,omitempty"`
}

// PostHandler serves POST /api/orderqueue on the answer listener: same origin list, token, JSON-only and small-body rules
// as the other answer routes.
func (q *OrderQueue) PostHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !unlockPostGate(o, w, r) {
			return
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, queueReply{Code: c, Error: msg}) }
		var b queueBody
		if err := decodeSmall(r, &b); err != nil {
			fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields")
			return
		}
		done := func() {
			v := q.View()
			writeJSON(w, 200, queueReply{OK: true, Queue: &v})
		}
		switch b.Action {
		case "add":
			forever, level := false, 0
			switch b.Until {
			case "", "minutes":
			case "forever":
				forever = true
			case "level":
				level = b.Level
				if level == 0 {
					fail(http.StatusBadRequest, "bad-order", "until level needs a level")
					return
				}
			default:
				fail(http.StatusBadRequest, "bad-order", "until must be minutes, forever or level")
				return
			}
			e, dup, err := q.Add(b.Kind, b.Area, b.Minutes, forever, level, b.Approve, b.Key)
			switch {
			case err == nil:
				writeJSON(w, 200, queueReply{OK: true, Entry: &e, Duplicate: dup})
			case errors.Is(err, ErrQueueLocked):
				fail(http.StatusForbidden, "locked", err.Error())
			case errors.Is(err, ErrQueueFull):
				fail(http.StatusConflict, "queue-full", err.Error())
			case errors.Is(err, currency.ErrCharterNeeded):
				rep := queueReply{Code: "charter-needed", Error: err.Error()}
				if q.cfg.Quote != nil {
					if qt, qerr := q.cfg.Quote(b.Area); qerr == nil && qt.Price > 0 {
						rep.Charter = &Charter{Area: qt.Area, Price: qt.Price, Text: qt.Text}
					}
				}
				writeJSON(w, http.StatusPaymentRequired, rep)
			case errors.Is(err, ErrBadOrder):
				fail(http.StatusBadRequest, "bad-order", err.Error())
			default:
				fail(http.StatusInternalServerError, "internal", err.Error())
			}
		case "cancel":
			err := q.Cancel(b.ID)
			switch {
			case err == nil:
				done()
			case errors.Is(err, ErrNoOrder):
				fail(http.StatusNotFound, "not-found", err.Error())
			case errors.Is(err, errOrderClosed):
				fail(http.StatusConflict, "order-state", err.Error())
			default:
				fail(http.StatusInternalServerError, "internal", err.Error())
			}
		case "resume":
			q.Resume()
			done()
		case "clear":
			q.Clear()
			done()
		default:
			fail(http.StatusBadRequest, "bad-request", "action must be add, cancel, resume or clear")
		}
	})
}
