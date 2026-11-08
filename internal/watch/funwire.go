package watch

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/unlock"
)

// The wiring of the fun loop (TASK-127 return screen, TASK-129 tally strip, TASK-130 Codex; docs/return-screen.md,
// docs/tally-strip.md, docs/codex.md "Wiring"). Fun is what the bot builds once and feeds with every goals event;
// ServeOptions.Fun mounts its routes on the two listeners.

// Charter is the price of an order above the zone range, as the order sheet shows it.
type Charter struct {
	Area  string `json:"area"`
	Price int64  `json:"price"`
	Text  string `json:"text"`
}

// OrderPlacer takes an order with the charter sink in front of it; TallyStrip is the shipped one.
type OrderPlacer interface {
	PlaceOrder(r OrderRequest, approve bool) (Order, currency.Quote, error)
}

type placerBox struct{ p OrderPlacer }

// SetOrderPlacer routes POST /api/orders through p (nil: straight to Give, the default).
func (h *Hub) SetOrderPlacer(p OrderPlacer) {
	if p == nil {
		h.placer.Store(nil)
		return
	}
	h.placer.Store(&placerBox{p})
}

// place gives an order, through the charter sink when one is set. The Charter is returned for any priced order.
func (h *Hub) place(r OrderRequest, approve bool) (Order, *Charter, error) {
	b := h.placer.Load()
	if b == nil {
		o, err := h.Give(r)
		return o, nil, err
	}
	o, q, err := b.p.PlaceOrder(r, approve)
	var ch *Charter
	if q.Price > 0 {
		ch = &Charter{Area: q.Area, Price: q.Price, Text: q.Text}
	}
	return o, ch, err
}

// FunConfig builds a Fun. Hub is required; every other part is optional and a missing part leaves its route off
// (the return screen then shows the placeholder of that section).
type FunConfig struct {
	Hub     *Hub
	Visit   *Visit  // nil: in memory
	Speed   float64 // virtual seconds per wall second (Config.Speed)
	Notes   NotesSink
	Strip   *TallyStrip // nil: /api/tally says tally is off
	Codex   *decide.Codex
	Table   *decide.Firsts
	Clock   decide.Clock
	Tracker func(class string) (*decide.Tracker, error) // builds the Tracker once the hero's class is known
	// Policy turns the chores policy sheet on (TASK-128): the table and, for options that set knobs, the tuner's edit.
	Policy *PolicyConfig
	// Proposals turns the zone proposals on (TASK-131): the proposer, the clock and the bot's facts. Intro is filled from
	// the Codex when it is left nil and a Codex is wired.
	Proposals *ProposalConfig
	// Unlock turns the unlock ladder on (TASK-153): the engine of the profile (unlock.Open). nil: no ladder, no routes.
	Unlock *unlock.Engine
	// Hall turns the Hall of Fame and the retire question on (TASK-135). StatePath "" keeps the state next to the ledger.
	Hall *HallConfig
}

// PolicyConfig is what the sheet needs from the process: the table (decide.ShippedPolicy) and the knob path (nil: options
// that set knobs are refused with a clear error, rule-only options work).
type PolicyConfig struct {
	Table *decide.PolicyTable
	Knobs KnobSetter
}

// Fun connects the three pieces to one running bot. Safe for concurrent use.
type Fun struct {
	Hub    *Hub
	Away   *AwayLog
	Return ReturnConfig
	Strip  *TallyStrip
	Policy *PolicySheet // nil: the sheet is off
	// Proposals is nil when zone proposals are off.
	Proposals *ProposalDesk
	// Unlocks is nil when the unlock ladder is off.
	Unlocks *Unlocks
	// Hall is nil when the Hall of Fame is off.
	Hall *HallDesk
	// Chronicle is nil unless the ladder has the Chronicle rows (TASK-154).
	Chronicle *Chronicle
	// Voice is nil unless the ladder has the row "voice" (TASK-155).
	Voice *Voice
	// Pacts mounts the Pact routes (pactapi.go); set it before Serve, nil: no routes.
	Pacts *PactAPI
	// OnView is called for every request that passed the viewer's gate (the presence signal for away safety); nil: none.
	OnView func()

	codex   *decide.Codex
	table   *decide.Firsts
	clock   decide.Clock
	mkTrack func(string) (*decide.Tracker, error)

	mu      sync.Mutex
	tracker *decide.Tracker
	pending []decide.FirstEvent // first-sight edges seen before the hero's class was known
	errs    []string
}

// maxPendingFirsts bounds the edges kept while the class is unknown.
const maxPendingFirsts = 200

// NewFun makes the Fun, registers the return screen sections (tally, codex, quiet) and puts the charter sink in front
// of the order route.
func NewFun(c FunConfig) (*Fun, error) {
	if c.Hub == nil {
		return nil, errors.New("watch: Fun needs a Hub")
	}
	f := &Fun{Hub: c.Hub, Away: c.Hub.AwayLog(), Strip: c.Strip, codex: c.Codex, table: c.Table, clock: c.Clock, mkTrack: c.Tracker}
	if (c.Codex == nil) != (c.Table == nil) || (c.Codex != nil && (c.Clock == nil || c.Tracker == nil)) {
		return nil, errors.New("watch: the Codex needs its file, the table, a clock and a tracker factory together")
	}
	if c.Visit == nil {
		c.Visit = NewVisit("", nil)
	}
	sec := DefaultSections()
	sec.Register("tally", TallySection(c.Strip))
	if c.Codex != nil {
		sec.Register(CodexSectionID, CodexReturnSection(f.CodexSource()))
	}
	sec.Register("quiet", QuietSection(c.Hub))
	if c.Unlock != nil {
		f.Unlocks = &Unlocks{Engine: c.Unlock, Hub: c.Hub, Spend: c.Strip.UnlockSpender()}
		sec.Register(UnlocksSectionID, f.Unlocks.Section())
		if c.Unlock.Has(ChronicleID) || c.Unlock.Has(ChronicleTapeID) {
			h := c.Hub
			ch, err := NewChronicle(f.Unlocks, nil, func() float64 { return h.rel(h.cfg.Clock.Now()) },
				func() []Trip { return f.Return.Return(7 * 86400).Trips })
			if err != nil {
				return nil, err
			}
			f.Chronicle = ch
		}
		if c.Unlock.Has(VoiceID) {
			h := c.Hub
			vo, err := NewVoice(f.Unlocks, c.Hub, nil, func() string { return h.heroNow().Class }, func() float64 { return h.rel(h.cfg.Clock.Now()) })
			if err != nil {
				return nil, err
			}
			f.Voice = vo
		}
	}
	if c.Proposals != nil {
		pc := *c.Proposals
		if pc.Intro == nil && c.Codex != nil {
			pc.Intro = CodexIntro(c.Codex, c.Table)
		}
		if pc.Visit == nil {
			pc.Visit = c.Visit
		}
		pd, err := NewProposalDesk(c.Hub, pc)
		if err != nil {
			return nil, err
		}
		f.Proposals = pd
		sec.Register(ProposalsSectionID, pd.Section())
	}
	if c.Policy != nil {
		h := c.Hub
		ps, err := NewPolicySheet(h.cfg.Store, h.cfg.Cat, c.Policy.Table, func() string { return strings.ToLower(h.heroNow().Class) },
			func() float64 { return h.rel(h.cfg.Clock.Now()) }, f.Away, c.Policy.Knobs)
		if err != nil {
			return nil, err
		}
		f.Policy = ps
		sec.Register(PolicySectionID, ps.Section())
	}
	if c.Hall != nil {
		hc := *c.Hall
		if hc.StatePath == "" && hc.Store != nil && hc.Store.Ledger != nil {
			hc.StatePath = ledgerDirState(hc.Store.Ledger)
		}
		hd, err := NewHallDesk(hc)
		if err != nil {
			return nil, err
		}
		f.Hall = hd
		sec.Register(HallSectionID, hd.Section())
	}
	f.Return = ReturnConfig{Hub: c.Hub, Events: f.Away, Visit: c.Visit, Speed: c.Speed, Sections: sec, Notes: c.Notes}
	if f.Unlocks != nil {
		f.Return.OnRead = func(d ReturnData) { // the first expedition card the viewer is shown
			if len(d.Trips) > 0 {
				f.Unlocks.Seen("expedition_card")
			}
		}
	}
	if c.Strip != nil {
		c.Hub.SetOrderPlacer(c.Strip)
	}
	return f, nil
}

// CodexSource is the Codex view's source, nil without a Codex.
func (f *Fun) CodexSource() CodexSource {
	if f.codex == nil {
		return nil
	}
	return CodexReader{Codex: f.codex, Table: f.table, Clock: f.clock}
}

// OnGoalEvent is the one hook feedHub calls for every goals event: the away log, the tally credits and the firsts.
// Only edges the events carry are derived (kill by name, area change, drop quality, death killer); champion and shrine
// have no reader behind goals.Event (docs/codex.md).
func (f *Fun) OnGoalEvent(ev goals.Event) {
	f.Away.OnGoalEvent(ev)
	if f.Hall != nil {
		f.Hall.OnGoalEvent(ev)
	}
	if f.Strip != nil {
		f.Strip.OnEvent(ev)
	}
	if f.Unlocks != nil {
		f.Unlocks.OnGoalEvent(ev)
	}
	if f.Chronicle != nil {
		f.Chronicle.OnGoalEvent(ev)
	}
	if f.Voice != nil {
		f.Voice.OnGoalEvent(ev)
	}
	if f.codex == nil {
		return
	}
	hero := f.Hub.heroNow()
	fe := decide.FirstEvent{Area: hero.Area, Clvl: hero.Level}
	switch ev.Type {
	case goals.EvKill:
		fe.Kind, fe.Name = "monster", ev.Monster
	case goals.EvAreaChange:
		fe.Kind, fe.Name, fe.Area = "area", ev.Area, ev.Area
	case goals.EvDrop:
		if ev.Item == nil {
			return
		}
		fe.Kind = "drop"
		fe.Name, _ = rankOf(AwayEvent{Item: ev.Item.Name, Quality: ev.Item.Quality, Type: ev.Item.Type})
	case goals.EvDeath:
		fe.Kind, fe.Name = "death", ev.Monster
	default:
		return
	}
	if fe.Name == "" {
		return
	}
	f.observe(fe)
}

// OnWaypoint is called where the bot learns that it lit a waypoint (Runner.SetWaypointHook). id is unique per touch.
func (f *Fun) OnWaypoint(area, id string) {
	if f.Strip != nil {
		f.Strip.Waypoint(area, id)
	}
	if f.codex != nil {
		hero := f.Hub.heroNow()
		f.observe(decide.FirstEvent{Kind: "waypoint", Name: area, Area: area, Clvl: hero.Level})
	}
}

func (f *Fun) fail(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.errs) < 20 {
		f.errs = append(f.errs, err.Error())
	}
}

// Errors lists what the wiring could not do (a Codex row that failed, an unknown class), oldest first.
func (f *Fun) Errors() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.errs...)
}

func (f *Fun) tracking() *decide.Tracker {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.tracker != nil {
		return f.tracker
	}
	class := strings.ToLower(f.Hub.heroNow().Class)
	if class == "" {
		return nil // the status poll has not reported the character yet
	}
	t, err := f.mkTrack(class)
	if err != nil {
		if len(f.errs) < 20 {
			f.errs = append(f.errs, err.Error())
		}
		return nil
	}
	f.tracker = t
	return t
}

func (f *Fun) observe(e decide.FirstEvent) {
	f.mu.Lock()
	if len(f.pending) < maxPendingFirsts {
		f.pending = append(f.pending, e)
	}
	f.mu.Unlock()
	f.flush()
}

// flush gives the queued edges to the tracker once the hero's class is known.
func (f *Fun) flush() {
	t := f.tracking()
	if t == nil {
		return
	}
	f.mu.Lock()
	queue := f.pending
	f.pending = nil
	f.mu.Unlock()
	for _, q := range queue {
		fired, err := t.Observe(q)
		if err != nil {
			f.fail(err)
		}
		if f.Unlocks != nil {
			f.Unlocks.OnFired(fired)
		}
		if f.Chronicle != nil {
			f.Chronicle.OnFired(fired)
		}
		if f.Voice != nil {
			f.Voice.OnFired(fired)
		}
	}
}

// Settle updates Codex entries whose thought closed and delivers the edges that waited for the class (call it from
// the loop that polls the status).
func (f *Fun) Settle() {
	if f.Hall != nil {
		f.Hub.setHeroTitle(f.Hall.Title())
	}
	if f.Proposals != nil {
		f.Proposals.Tick()
	}
	if f.codex == nil {
		return
	}
	f.flush()
	f.mu.Lock()
	t := f.tracker
	f.mu.Unlock()
	if t != nil {
		if err := t.Settle(); err != nil {
			f.fail(err)
		}
	}
}

// Mount adds the routes to the viewer mux and returns the wrapper for the answer listener. Viewer: the return screen
// and the /watch landing (GET only), /api/tally, /api/codex. Answer listener: the return screen's two POST routes.
func (f *Fun) mountView(m *http.ServeMux, live http.Handler) {
	rh := f.Return.ReturnHandler(live)
	m.Handle("/watch", rh)
	m.Handle("/api/thoughts", rh) // a poll of the live screen counts as a look (Visit)
	for _, p := range ReturnRoutes {
		m.Handle(p, rh)
	}
	if f.Strip != nil {
		m.Handle("/api/tally", f.Strip.Handler())
	} else {
		m.Handle("/api/tally", tallyOff())
	}
	if src := f.CodexSource(); src != nil {
		m.Handle(CodexRoute, CodexHandler(src))
	}
	if f.Proposals != nil {
		m.Handle(ProposalsRoute, f.Proposals.Handler())
	}
	if f.Unlocks != nil {
		m.Handle(UnlocksRoute, f.Unlocks.Handler())
		m.Handle(UnlockUIRoute, unlockUIHandler())
	}
	if f.Chronicle != nil {
		m.Handle(ChronicleRoute, f.Chronicle.Handler())
	}
	if f.Voice != nil {
		m.Handle(VoiceRoute, f.Voice.Handler())
	}
	if f.Hall != nil {
		hh := f.Hall.Handler()
		for _, p := range HallRoutes {
			m.Handle(p, hh)
		}
	}
	if f.Pacts != nil {
		for _, p := range PactRoutes {
			m.Handle(p, f.Pacts.Handler())
		}
	}
	if f.Policy != nil {
		ph := f.Policy.PolicyHandler()
		for _, p := range PolicyRoutes {
			m.Handle(p, ph)
		}
	}
}

// owns says whether a ViewRoutes entry is served by mountView instead (the landing redirect and the look tracking).
func (f *Fun) owns(route string) bool { return route == "/watch" || route == "/api/thoughts" }

func (f *Fun) wrapAnswer(base http.Handler, o AnswerOptions) http.Handler {
	m := http.NewServeMux()
	m.Handle("/", base)
	ra := f.Return.ReturnAnswerHandler(o)
	for _, p := range ReturnPostRoutes {
		m.Handle(p, ra)
	}
	if f.Proposals != nil {
		m.Handle(ProposalsPostRoute, f.Proposals.PostHandler(o))
	}
	if f.Unlocks != nil {
		m.Handle(UnlocksBuyRoute, f.Unlocks.BuyHandler(o))
	}
	if f.Voice != nil {
		m.Handle(VoiceMuteRoute, f.Voice.MuteHandler(o))
	}
	if f.Hall != nil {
		m.Handle(HallPostRoute, f.Hall.PostHandler(o))
	}
	if f.Pacts != nil {
		for _, p := range PactPostRoutes {
			m.Handle(p, f.Pacts.PostHandler(o))
		}
	}
	if f.Policy != nil {
		m.Handle(PolicyPostRoute, f.Policy.PolicyAnswerHandler(o))
	}
	return m
}

func tallyOff() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "GET only", http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, 200, &TallyView{Schema: tallySchema, Mode: currency.VanillaMode, Strip: TallyOffText, Recent: []TallyLine{}, Next: []NextBuy{}})
	})
}

func (h *Hub) heroNow() HeroInfo {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.hero
}

// TallySection is the return screen's tally section (replaces the placeholder). nil strip: it keeps the placeholder text.
func TallySection(s *TallyStrip) ReturnSection {
	return SectionFunc(func(*ReturnData) *SectionView {
		v := &SectionView{ID: "tally", Title: "Tally", After: true, Placeholder: "Tally is not switched on."}
		if s == nil {
			return v
		}
		tv := s.View()
		v.Lines = append(v.Lines, tv.Strip)
		if tv.Error != "" {
			v.Lines = append(v.Lines, tv.Error)
		}
		if tv.Last != nil {
			v.Lines = append(v.Lines, fmt.Sprintf("Last trip: %+d (%d earned, %d spent)", tv.Last.Net, tv.Last.Credited, tv.Last.Spent))
			for i, l := range tv.Last.Lines {
				if i == 5 {
					break
				}
				v.Lines = append(v.Lines, fmt.Sprintf("%+d %s", l.Amount, l.Why))
			}
		}
		return v
	})
}

// QuietSection says which questions the hero stopped asking while the player was away (TASK-137). Hidden when none.
func QuietSection(h *Hub) ReturnSection {
	return SectionFunc(func(d *ReturnData) *SectionView {
		var lines []string
		for _, l := range h.Journal(0, "quiet").Lines {
			if l.Kind == "quiet" && l.T >= d.Since {
				lines = append(lines, l.Text)
			}
		}
		if len(lines) == 0 {
			return nil
		}
		return &SectionView{ID: "quiet", Title: "Stopped asking", Lines: lines}
	})
}
