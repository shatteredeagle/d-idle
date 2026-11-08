package watch

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/goals"
)

// The tally strip: credits from the bot's observed events, the strip and per-expedition view (GET /api/tally, mounted
// by the caller; docs/tally-strip.md), and the charter sink for an above-range order. Reads and writes go through the
// currency ledger; nothing here keeps a balance of its own.

const tallySchema = "watch.tally/1"

// TallyLine is one movement shown under the strip.
type TallyLine = currency.Line

// ExpeditionDelta is the tally of one town trip (leave town to return).
type ExpeditionDelta struct {
	Open     bool        `json:"open"` // the hero is still out
	Credited int64       `json:"credited"`
	Spent    int64       `json:"spent"`
	Net      int64       `json:"net"`
	Lines    []TallyLine `json:"lines"`
}

// NextBuy is one of the things the tally can buy next.
type NextBuy struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Cost       int64  `json:"cost"`
	Affordable bool   `json:"affordable"`
	Wired      bool   `json:"wired"`
}

// TallyView is GET /api/tally.
type TallyView struct {
	Schema     string           `json:"schema"`
	Enabled    bool             `json:"enabled"`
	Mode       string           `json:"mode"`
	Strip      string           `json:"strip"`
	Balance    int64            `json:"balance"`
	Sealed     int64            `json:"sealed,omitempty"`
	Recent     []TallyLine      `json:"recent"`
	Expedition *ExpeditionDelta `json:"expedition"` // the trip now under way, null in town
	Last       *ExpeditionDelta `json:"last"`       // the trip that just ended, null before the first
	Next       []NextBuy        `json:"next"`
	Error      string           `json:"error,omitempty"`
}

// TallyStrip feeds events to the tally and renders it. IsTown says whether an area name is a town (trip boundaries);
// there is no default because a wrong guess would credit or hide trips silently.
type TallyStrip struct {
	Hub    *Hub
	Tally  *currency.Tally
	Wallet *currency.Wallet
	IsTown func(area string) bool
	Now    func() time.Time

	mu       sync.Mutex
	inField  bool
	start    time.Time
	trips    int
	last     *currency.Window
	firstErr error
}

// NewTallyStrip checks the parts.
func NewTallyStrip(h *Hub, w *currency.Wallet, t *currency.Tally, isTown func(string) bool) (*TallyStrip, error) {
	if h == nil || w == nil || t == nil || isTown == nil {
		return nil, errors.New("watch: tally strip needs a hub, a wallet, a tally and an IsTown function")
	}
	return &TallyStrip{Hub: h, Tally: t, Wallet: w, IsTown: isTown, Now: time.Now}, nil
}

func (s *TallyStrip) fail(err error) {
	if err == nil || errors.Is(err, currency.ErrVanilla) {
		return
	}
	s.mu.Lock()
	if s.firstErr == nil {
		s.firstErr = err
	}
	s.mu.Unlock()
}

func (s *TallyStrip) credit(ev currency.TallyEvent) {
	_, err := s.Tally.Credit(ev)
	s.fail(err)
}

// OnEvent is the credit hook: call it for every goals event (GoalsOrders.OnEvent is the place). Events that earn
// nothing are ignored. The event id is the run plus the event's sequence number, so a replayed journal books nothing twice.
func (s *TallyStrip) OnEvent(ev goals.Event) {
	id := fmt.Sprintf("%s:%d", s.Tally.Run, ev.Seq)
	if ev.Seq == 0 && (ev.Type == goals.EvKill || ev.Type == goals.EvAreaChange || ev.Type == goals.EvQuest) {
		s.fail(fmt.Errorf("tally: a %s event without a sequence number cannot be credited safely", ev.Type))
		return
	}
	switch ev.Type {
	case goals.EvKill:
		if ev.Monster == "" {
			return
		}
		class := "normal"
		if ev.Boss {
			class = "boss"
		}
		s.credit(currency.TallyEvent{Kind: currency.EvKill, ID: id, Subject: ev.Monster, Class: class})
	case goals.EvQuest:
		if ev.Done && ev.Quest != "" {
			s.credit(currency.TallyEvent{Kind: currency.EvQuest, ID: id, Subject: fmt.Sprintf("%s/%d", ev.Quest, ev.Diff)})
		}
	case goals.EvAreaChange:
		if ev.Area == "" {
			return
		}
		if !s.IsTown(ev.Area) {
			s.credit(currency.TallyEvent{Kind: currency.EvArea, ID: id, Subject: ev.Area})
			s.mu.Lock()
			if !s.inField {
				s.inField, s.start = true, s.Now()
			}
			s.mu.Unlock()
			return
		}
		s.mu.Lock()
		was, start := s.inField, s.start
		s.inField = false
		if was {
			s.trips++
		}
		n := s.trips
		s.mu.Unlock()
		if !was {
			return
		}
		s.credit(currency.TallyEvent{Kind: currency.EvTrip, ID: id, Subject: fmt.Sprint(n)})
		s.mu.Lock()
		s.last = &currency.Window{From: start, To: s.Now().Add(time.Second)}
		s.mu.Unlock()
	}
}

// Waypoint credits a lit waypoint. No goals event says this today (U), so the bot calls it where it learns the fact.
func (s *TallyStrip) Waypoint(area, id string) {
	s.credit(currency.TallyEvent{Kind: currency.EvWaypoint, ID: id, Subject: area})
}

func delta(t *currency.Tally, w currency.Window, open bool) (*ExpeditionDelta, error) {
	sum, err := t.SummaryFor(w)
	if err != nil {
		return nil, err
	}
	return &ExpeditionDelta{Open: open, Credited: sum.Credited, Spent: sum.Spent, Net: sum.Net, Lines: sum.Lines}, nil
}

// View builds the strip. It only reads; a ledger that fails its check shows an error and no balance.
func (s *TallyStrip) View() *TallyView {
	w := s.Wallet
	v := &TallyView{Schema: tallySchema, Mode: w.Mode, Recent: []TallyLine{}, Next: []NextBuy{}}
	if w.Mode == "" || w.Mode == currency.VanillaMode || w.Ledger == nil {
		v.Mode = currency.VanillaMode
		v.Strip = "Tally is off in vanilla mode."
		return v
	}
	v.Enabled = true
	s.mu.Lock()
	inField, start, last, ferr := s.inField, s.start, s.last, s.firstErr
	s.mu.Unlock()
	if ferr != nil {
		v.Error = "A tally credit failed: " + ferr.Error()
	}
	all, err := s.Tally.SummaryFor(currency.Window{})
	if err != nil {
		v.Error = "The ledger failed its check, so tally is hidden: " + err.Error()
		v.Strip = "Tally -"
		return v
	}
	v.Balance, v.Sealed = all.Balance, all.Sealed
	for i := len(all.Lines) - 1; i >= 0 && len(v.Recent) < 5; i-- {
		if all.Lines[i].Amount > 0 {
			v.Recent = append(v.Recent, all.Lines[i])
		}
	}
	if inField {
		if v.Expedition, err = delta(s.Tally, currency.Window{From: start}, true); err != nil {
			v.Error = err.Error()
		}
	}
	if last != nil {
		if v.Last, err = delta(s.Tally, *last, false); err != nil {
			v.Error = err.Error()
		}
	}
	rows := s.Tally.Rows
	v.Next = append(v.Next, NextBuy{ID: "charter", Name: "Charter for a trip above your level", Cost: rows.Charter.Base, Affordable: v.Balance >= rows.Charter.Base, Wired: true})
	levels, _ := w.Ledger.Levels()
	for _, u := range w.Reg.Upgrades {
		if lv := levels[u.ID]; u.Currency == rows.Currency && lv < len(u.Costs) {
			v.Next = append(v.Next, NextBuy{ID: u.ID, Name: u.Name, Cost: u.Costs[lv], Affordable: v.Balance >= u.Costs[lv], Wired: u.Wired})
		}
	}
	sort.SliceStable(v.Next, func(i, j int) bool { return v.Next[i].Cost < v.Next[j].Cost })
	if len(v.Next) > 3 {
		v.Next = v.Next[:3]
	}
	v.Strip = fmt.Sprintf("Tally %d", v.Balance)
	if v.Expedition != nil {
		v.Strip += fmt.Sprintf(" (%+d this trip)", v.Expedition.Net)
	} else if v.Last != nil {
		v.Strip += fmt.Sprintf(" (%+d last trip)", v.Last.Net)
	}
	if len(v.Next) > 0 {
		v.Strip += fmt.Sprintf(" · next: %s %d", v.Next[0].Name, v.Next[0].Cost)
	}
	return v
}

// Handler serves View as JSON (GET only). The caller mounts it where /api/currency is served.
func (s *TallyStrip) Handler() http.Handler {
	return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(rw, "GET only", http.StatusMethodNotAllowed)
			return
		}
		rw.Header().Set("Content-Type", "application/json")
		rw.Header().Set("Cache-Control", "no-store")
		_ = json.NewEncoder(rw).Encode(s.View())
	})
}

// Quote prices a trip to area for the hero as the screen knows it.
func (s *TallyStrip) Quote(area string) (currency.Quote, error) {
	s.Hub.mu.Lock()
	level := s.Hub.hero.Level
	s.Hub.mu.Unlock()
	return s.Tally.Rows.Quote(area, level)
}

// PlaceOrder is Give with the charter sink. A "go" order to an area above the hero's range needs approve: without it
// nothing is placed and the error wraps ErrCharterNeeded; the returned quote carries the hero's warning text. With
// approve the order is placed first and the charter is paid after, keyed by the order, so a refused order costs nothing
// and tally never goes negative (a short balance is refused before the order is placed). Other order kinds are free.
func (s *TallyStrip) PlaceOrder(r OrderRequest, approve bool) (Order, currency.Quote, error) {
	if r.Kind != OrderGo {
		o, err := s.Hub.Give(r)
		return o, currency.Quote{}, err
	}
	q, err := s.Quote(r.Area)
	if err != nil {
		return Order{}, q, err
	}
	if q.Price > 0 {
		if !approve {
			return Order{}, q, fmt.Errorf("%w: %s", currency.ErrCharterNeeded, q.Text)
		}
		b, err := s.Wallet.Ledger.Balance(s.Tally.Rows.Currency)
		if err != nil {
			return Order{}, q, err
		}
		if b.Spendable < q.Price {
			return Order{}, q, fmt.Errorf("%w: a charter costs %d, you have %d", currency.ErrInsufficient, q.Price, b.Spendable)
		}
	}
	o, err := s.Hub.Give(r)
	if err != nil {
		return o, q, err
	}
	if _, err := s.Tally.Charter(q, fmt.Sprintf("%s/%d", s.Tally.Run, o.ID)); err != nil {
		_ = s.Hub.CancelOrder(o.ID)
		return o, q, err
	}
	return o, q, nil
}
