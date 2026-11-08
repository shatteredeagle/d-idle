package watch

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"sync"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

// The unlock ladder on the watch screen (TASK-153; docs/unlock-ladder.md). Unlocks owns the engine's reveal: every
// entry that opens writes one Chronicle line ("story") and one away event, and the return screen lists what opened while
// the viewer was away. Nothing here reads the game and nothing reaches the bot.

const (
	// AwayUnlock is the kind of a reveal record (kept in the Unlocks' own log, not the return screen's); Item holds "Title: text", Type the entry id.
	AwayUnlock = "unlock"
	// UnlocksSectionID is the return screen section id.
	UnlocksSectionID = "unlocks"
	// UnlocksRoute is GET (viewer); UnlocksBuyRoute is POST (answer listener).
	UnlocksRoute    = "/api/unlocks"
	UnlocksBuyRoute = "/api/unlocks/buy"

	// UnlockUIRoute serves the script of the unlocked panels (web/unlockui.js).
	UnlockUIRoute = "/watch/unlockui.js"

	unlocksSchema = "watch.unlocks/1"
)

func unlockUIHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := webFS.ReadFile("web/unlockui.js")
		if err != nil {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", pageCSP)
		w.Header().Set("Cache-Control", "no-store")
		w.Write(b)
	})
}

// Unlocks connects the engine to the Hub, the away log and the tally.
type Unlocks struct {
	Engine *unlock.Engine
	Hub    *Hub
	Spend  unlock.Spender // nil: tally is off, buying answers 503

	mu      sync.Mutex
	errs    []string
	kills   map[string]int // running kills per monster name, in memory (a restart starts again; opening is persisted)
	killSeq uint64         // highest kill Seq counted: a replayed stream never counts twice
	// log holds the reveals, apart from the return screen's event log: an unlock arriving in camp (the Codex settles
	// after the hero is back) must not open a phantom expedition there.
	log *AwayLog
}

func (u *Unlocks) reveals() *AwayLog {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.log == nil {
		var now func() float64
		if u.Hub != nil {
			now = func() float64 { return u.Hub.rel(u.Hub.cfg.Clock.Now()) }
		}
		u.log = NewAwayLog(now)
	}
	return u.log
}

// Observe feeds the engine and reveals what opened.
func (u *Unlocks) Observe(ev unlock.Event) {
	opened, err := u.Engine.Observe(ev)
	if err != nil {
		u.fail(err)
	}
	for _, e := range opened {
		u.reveal(e)
	}
}

// reveal is the moment: a Chronicle line and a reveal record. Never skipped, so nothing unlocks silently.
func (u *Unlocks) reveal(e unlock.Entry) {
	if u.Hub != nil {
		u.Hub.Note("story", fmt.Sprintf("Unlocked: %s. %s", e.Reveal.Title, e.Reveal.Text))
	}
	u.reveals().Add(AwayEvent{Kind: AwayUnlock, Item: safeText(e.Reveal.Title+": "+e.Reveal.Text, 200), Type: e.ID})
}

func (u *Unlocks) fail(err error) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if len(u.errs) < 20 {
		u.errs = append(u.errs, err.Error())
	}
}

// Errors lists what the engine could not do (a failed save, an unknown event), oldest first.
func (u *Unlocks) Errors() []string {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]string(nil), u.errs...)
}

// OnGoalEvent maps the goals events the ladder reads: a level reached and a finished quest.
func (u *Unlocks) OnGoalEvent(ev goals.Event) {
	switch ev.Type {
	case goals.EvKill:
		if n := u.countKill(ev); n > 0 {
			u.Observe(unlock.Event{Type: unlock.EvKills, Name: ev.Monster, N: n})
		}
	case goals.EvLevelUp:
		if ev.Level > 0 {
			u.Observe(unlock.Event{Type: unlock.EvLevel, N: ev.Level})
		}
	case goals.EvQuest:
		if ev.Done && ev.Quest != "" {
			u.Observe(unlock.Event{Type: unlock.EvQuest, Name: ev.Quest})
		}
	}
}

// countKill returns the running count of the monster kind, 0 for an event without a name or one already counted.
func (u *Unlocks) countKill(ev goals.Event) int {
	if ev.Monster == "" {
		return 0
	}
	u.mu.Lock()
	defer u.mu.Unlock()
	if ev.Seq != 0 {
		if ev.Seq <= u.killSeq {
			return 0
		}
		u.killSeq = ev.Seq
	}
	if u.kills == nil {
		u.kills = map[string]int{}
	}
	u.kills[ev.Monster]++
	return u.kills[ev.Monster]
}

// Seen reports a viewer-side moment (trigger kind seen), for example "expedition_card".
func (u *Unlocks) Seen(what string) { u.Observe(unlock.Event{Type: unlock.EvSeen, Name: what}) }

// OnFired maps new Codex entries (decide.Tracker.Observe) to codex_first events.
func (u *Unlocks) OnFired(fired []decide.Fired) {
	for _, f := range fired {
		u.Observe(unlock.Event{Type: unlock.EvCodexFirst, Sub: f.Entry.Kind, ID: f.Entry.ID, Name: f.Entry.Subject})
	}
}

// UnlocksView is GET /api/unlocks. Locked entries carry the teaser, never the text.
type UnlocksView struct {
	Schema  string         `json:"schema"`
	Open    []unlock.State `json:"open"`
	Locked  []unlock.State `json:"locked"`
	Errors  []string       `json:"errors,omitempty"`
	Balance *int64         `json:"balance,omitempty"` // spendable tally, when tally is on
}

// View lists the entries, open first within table order.
func (u *Unlocks) View() UnlocksView {
	v := UnlocksView{Schema: unlocksSchema, Open: []unlock.State{}, Locked: []unlock.State{}, Errors: u.Errors()}
	for _, s := range u.Engine.States("") {
		if s.Open {
			v.Open = append(v.Open, s)
		} else {
			v.Locked = append(v.Locked, s)
		}
	}
	if b, ok := u.Spend.(interface{ Spendable() (int64, error) }); ok {
		if n, err := b.Spendable(); err == nil {
			v.Balance = &n
		}
	}
	return v
}

// Handler serves GET /api/unlocks.
func (u *Unlocks) Handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, u.View())
	})
}

// Section is the return screen's "Unlocked since you left". Hidden when nothing opened in the window.
func (u *Unlocks) Section() ReturnSection {
	return SectionFunc(func(d *ReturnData) *SectionView {
		var lines []string
		for _, e := range u.reveals().Events() {
			if e.Kind == AwayUnlock && e.T >= d.Since {
				lines = append(lines, e.Item)
			}
		}
		if len(lines) == 0 {
			return nil
		}
		return &SectionView{ID: UnlocksSectionID, Title: "Unlocked since you left", Lines: lines}
	})
}

type unlockBuyBody struct {
	ID string `json:"id"`
}

type unlockReply struct {
	OK       bool          `json:"ok"`
	Code     string        `json:"code,omitempty"`
	Error    string        `json:"error,omitempty"`
	Unlocked *unlock.State `json:"unlocked,omitempty"`
}

// BuyHandler is POST /api/unlocks/buy {id}: pays a tally_spend entry from the ledger. Same gate as the other answer
// routes (origin list, token, JSON only, small body). 402 when the tally is short, nothing is booked then.
func (u *Unlocks) BuyHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, unlockReply{Code: c, Error: msg}) }
		switch {
		case r.Method == http.MethodOptions && allowed:
			w.Header().Set("Access-Control-Allow-Methods", "POST")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.WriteHeader(http.StatusNoContent)
			return
		case r.Method != http.MethodPost:
			w.Header().Set("Allow", "POST, OPTIONS")
			fail(http.StatusMethodNotAllowed, "method", "POST only")
			return
		case origin != "" && !allowed:
			fail(http.StatusForbidden, "origin", "origin not allowed")
			return
		case !o.tokenOK(r):
			fail(http.StatusUnauthorized, "token", "unauthorized")
			return
		}
		if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
			fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
			return
		}
		var b unlockBuyBody
		if err := decodeSmall(r, &b); err != nil || b.ID == "" {
			fail(http.StatusBadRequest, "body", `body must be {"id": "<entry>"}`)
			return
		}
		if u.Spend == nil {
			fail(http.StatusServiceUnavailable, "tally-off", TallyOffText)
			return
		}
		e, err := u.Engine.Buy(b.ID, u.Spend)
		switch {
		case err == nil:
			u.reveal(e)
			for _, s := range u.Engine.States("") {
				if s.ID == e.ID {
					writeJSON(w, 200, unlockReply{OK: true, Unlocked: &s})
					return
				}
			}
		case errors.Is(err, unlock.ErrUnknownEntry):
			fail(http.StatusNotFound, "not-found", err.Error())
		case errors.Is(err, unlock.ErrNotBuyable):
			fail(http.StatusBadRequest, "not-for-sale", err.Error())
		case errors.Is(err, unlock.ErrAlreadyOpen):
			fail(http.StatusConflict, "already-open", err.Error())
		case errors.Is(err, currency.ErrInsufficient):
			fail(http.StatusPaymentRequired, "insufficient", err.Error())
		case errors.Is(err, currency.ErrVanilla), errors.Is(err, currency.ErrNeedsOverlay):
			fail(http.StatusServiceUnavailable, "tally-off", TallyOffText)
		default:
			fail(http.StatusInternalServerError, "internal", err.Error())
		}
	})
}

// LedgerSpender books an unlock as a tally spend in the profile's ledger. A key already booked counts as paid
// (the retry of a crash between the spend and the save); a short balance is refused by the ledger, never clamped.
type LedgerSpender struct {
	Ledger    *currency.Ledger
	Currency  string
	Character string
	Run       string
}

func (s LedgerSpender) Spend(key string, price int64, note string) error {
	_, err := s.Ledger.Append(currency.Entry{Kind: currency.KindSpend, Currency: s.Currency, Amount: price, Key: key,
		Upgrade: "unlock", Character: s.Character, Run: s.Run, Note: note})
	if errors.Is(err, currency.ErrDuplicate) {
		return nil
	}
	return err
}

// Spendable is the tally balance the buyer may spend.
func (s LedgerSpender) Spendable() (int64, error) {
	b, err := s.Ledger.Balance(s.Currency)
	return b.Spendable, err
}

// UnlockSpender is the strip's ledger as an unlock.Spender (nil without a wallet or tally).
func (s *TallyStrip) UnlockSpender() unlock.Spender {
	if s == nil || s.Wallet == nil || s.Wallet.Ledger == nil || s.Tally == nil {
		return nil
	}
	return LedgerSpender{Ledger: s.Wallet.Ledger, Currency: s.Tally.Rows.Currency, Character: s.Tally.Character, Run: s.Tally.Run}
}
