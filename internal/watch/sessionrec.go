package watch

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/decide"
)

// The session record (TASK-106, docs/session-record.md): per play session two append-only JSONL files under
// <state>/sessions/<id>/, the human-facing record (session.jsonl), the decision dataset (decisions.jsonl, schema v1 of
// docs/research/decision-dataset.md), and at the end summary.json and summary.md. Nothing here talks to a network, and no
// access token is ever given to this file: free text (the return screen's notes) is scrubbed of the configured secrets.

// SessionSchema is the version of both files.
const SessionSchema = 1

// Vitals is what the recorder samples from the running bot. LifePct is 0..100; XP is the hero's total experience.
type Vitals struct {
	LifePct float64
	XP      int64
	Credits int64 // tally credits so far (any monotone count; the summary reports the difference)
}

// SessionConfig builds a Session. Root, Store, Cat and Clock are required; the rest is optional.
type SessionConfig struct {
	Root             string // <state>/sessions
	Profile          string // store profile name, part of the directory name only
	Commit           string // bot commit or version
	CatalogueVersion int
	Store            *decide.Store
	Cat              *decide.Catalogue
	Clock            decide.Clock // game clock (virtual under lockstep, wall in realtime)
	Now              func() time.Time
	Rand             io.Reader // session id source; crypto/rand when nil
	Identity         func() (class, build string)
	Vitals           func() Vitals
	Viewers          func() int
	Orders           func() []Order
	WhyCounts        func() map[string]int // reason codes started this session (counts only, whyline.go); nil: none
	Secrets          []string              // scrubbed from every free-text field
	WindowMin        int                   // outcome window of the dataset (default 10)
	SampleSec        int                   // period of the "sample" lines of the record (default 300)
	DedupeSec        int                   // repeats of one key and source inside this many game seconds are counted, not logged (default 60)
}

// fireLine is a "fire" line of the dataset; field order is the file's order.
type fireLine struct {
	V         int               `json:"v"`
	Kind      string            `json:"kind"`
	ID        int               `json:"id"`
	Session   string            `json:"session"`
	Commit    string            `json:"commit"`
	TGame     int64             `json:"t_game"`
	TWall     string            `json:"t_wall"`
	Point     string            `json:"point"`
	Ctx       map[string]string `json:"ctx"`
	Class     string            `json:"class"`
	Build     string            `json:"build"`
	Options   []string          `json:"options"`
	Default   string            `json:"default"`
	Answer    string            `json:"answer"`
	By        string            `json:"by"`
	LatencyMS *int64            `json:"latency_ms"`
}

type pendingOutcome struct {
	id        int
	at        time.Duration // game clock
	xp0       int64
	deaths0   int
	minLife   float64
	haveLife  bool
	thoughtID int
}

// Session is one play session. Safe for concurrent use.
type Session struct {
	cfg       SessionConfig
	dir       string
	id        string // directory name: UTC start plus profile
	sid       string // random id of the dataset
	start     time.Time
	gameStart time.Duration // the game clock at the start; lockstep runs it many times faster than the wall
	identSent [2]string     // class and build of the last start or identity line

	mu       sync.Mutex
	rec, dec *os.File
	stopped  bool
	logIdx   int
	fireN    int
	seen     map[int]int           // thought id -> fire id
	openedAt map[int]time.Duration // thought id -> game time it opened
	lastFire map[string]time.Duration
	pending  []*pendingOutcome
	lastPoll time.Time
	lastSamp time.Time
	last     Vitals
	vit0     Vitals
	haveVit0 bool
	viewing  bool
	seenOrd  map[int]bool
	ordTimes []float64
	bytesRec int64
	bytesDec int64
	decKind  map[string]int64
	timeline []TimelineItem

	sum Summary
	lat []int64
}

// TimelineItem is one entry of the first-10-minutes timeline of the summary.
type TimelineItem struct {
	TS    float64 `json:"t_s"`
	Kind  string  `json:"kind"`
	Label string  `json:"label,omitempty"`
}

// Summary is summary.json: the metrics the direction review asked for.
type Summary struct {
	V                     int              `json:"v"`
	Session               string           `json:"session"`
	Commit                string           `json:"commit"`
	Class                 string           `json:"class"`
	Build                 string           `json:"build"`
	Start                 string           `json:"start"`
	End                   string           `json:"end"`
	DurationS             float64          `json:"duration_s"`      // wall seconds
	GameDurationS         float64          `json:"game_duration_s"` // game-clock seconds (virtual under lockstep)
	WallToGame            float64          `json:"game_per_wall"`   // game seconds per wall second (1 in realtime)
	Questions             int              `json:"questions"`
	Answered              int              `json:"answered"`
	Faded                 int              `json:"faded"`
	RulesLearned          int              `json:"rules_learned"`
	RulesRevoked          int              `json:"rules_revoked"`
	QuestionsPer10Min     []int            `json:"questions_per_10min"`
	FiresBy               map[string]int   `json:"fires_by"`
	FiresLogged           int              `json:"fires_logged"`
	FiresSuppressed       int              `json:"fires_suppressed"`
	LatencyMedianMS       *int64           `json:"answer_latency_median_ms"`
	Orders                int              `json:"orders"`
	SecBetweenOrders      *float64         `json:"mean_sec_between_orders"`
	StayedInTabMin        float64          `json:"stayed_in_tab_min"`
	Deaths                int              `json:"deaths"`
	XPPerH                float64          `json:"xp_per_h"` // per GAME hour; xp_per_wall_h is the same per wall hour
	XPPerWallH            float64          `json:"xp_per_wall_h"`
	Credits               int64            `json:"credits"`
	Notes                 string           `json:"notes,omitempty"`
	First10               []TimelineItem   `json:"first_10min"`
	RecordBytes           int64            `json:"record_bytes"`
	DecisionBytes         int64            `json:"decision_bytes"`
	DecisionBytesPerH     float64          `json:"decision_bytes_per_h"` // per GAME hour; the wall figure follows
	DecisionBytesPerWallH float64          `json:"decision_bytes_per_wall_h"`
	DecisionBytesByKind   map[string]int64 `json:"decision_bytes_by_kind"`
	WhyChain              map[string]int   `json:"why_chain,omitempty"` // reason codes started, counts only
	inTab                 time.Duration
	questionsPerBucket    map[int]int
}

// NewSession creates the directory and both files, writes the headers and hooks the store. Call Poll about every
// second and Stop at the end.
func NewSession(cfg SessionConfig) (*Session, error) {
	if cfg.Root == "" || cfg.Store == nil || cfg.Cat == nil || cfg.Clock == nil {
		return nil, errors.New("session: Root, Store, Cat and Clock are required")
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	if cfg.Rand == nil {
		cfg.Rand = rand.Reader
	}
	if cfg.WindowMin <= 0 {
		cfg.WindowMin = 10
	}
	if cfg.SampleSec <= 0 {
		cfg.SampleSec = 300
	}
	if cfg.DedupeSec <= 0 {
		cfg.DedupeSec = 60
	}
	var rb [8]byte
	if _, err := io.ReadFull(cfg.Rand, rb[:]); err != nil {
		return nil, err
	}
	s := &Session{cfg: cfg, sid: hex.EncodeToString(rb[:]), seen: map[int]int{}, openedAt: map[int]time.Duration{},
		lastFire: map[string]time.Duration{}, seenOrd: map[int]bool{}}
	s.start = cfg.Now().UTC()
	s.gameStart = cfg.Clock.Now()
	s.decKind = map[string]int64{}
	base := s.start.Format("20060102T150405Z")
	if cfg.Profile != "" {
		base += "-" + cfg.Profile
	}
	if err := os.MkdirAll(cfg.Root, 0o700); err != nil {
		return nil, err
	}
	for n := 1; ; n++ {
		s.id = base
		if n > 1 {
			s.id = fmt.Sprintf("%s-%d", base, n)
		}
		s.dir = filepath.Join(cfg.Root, s.id)
		err := os.Mkdir(s.dir, 0o700)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrExist) || n > 99 {
			return nil, err
		}
	}
	var err error
	if s.rec, err = os.OpenFile(filepath.Join(s.dir, "session.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err != nil {
		return nil, err
	}
	if s.dec, err = os.OpenFile(filepath.Join(s.dir, "decisions.jsonl"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); err != nil {
		s.rec.Close()
		return nil, err
	}
	class, build := s.identity()
	s.sum = Summary{V: SessionSchema, Session: s.id, Commit: cfg.Commit, Start: s.start.Format(time.RFC3339), FiresBy: map[string]int{},
		questionsPerBucket: map[int]int{}}
	s.writeDec(map[string]any{"v": SessionSchema, "kind": "header", "schema": "decision-dataset", "session": s.sid, "commit": cfg.Commit,
		"catalogue": cfg.CatalogueVersion, "start": s.start.Format(time.RFC3339), "window_min": cfg.WindowMin})
	s.writeRec("start", map[string]any{"session": s.id, "commit": cfg.Commit, "class": class, "build": build, "catalogue": cfg.CatalogueVersion})
	s.identSent = [2]string{class, build}
	cfg.Store.SetAskObserver(s.OnAsk)
	cfg.Store.Subscribe(s.onEvent)
	return s, nil
}

// Dir is the session's directory; ID its name.
func (s *Session) Dir() string { return s.dir }
func (s *Session) ID() string  { return s.id }

func (s *Session) identity() (string, string) {
	if s.cfg.Identity == nil {
		return "", ""
	}
	return s.cfg.Identity()
}

func (s *Session) scrub(t string) string {
	for _, sec := range s.cfg.Secrets {
		if len(sec) >= 4 {
			t = strings.ReplaceAll(t, sec, "[removed]")
		}
	}
	return t
}

// elapsed is seconds since the start by the injected wall clock.
func (s *Session) elapsed() float64 { return s.cfg.Now().Sub(s.start).Seconds() }

// writeRec appends one line to session.jsonl. The caller holds mu or is in NewSession.
func (s *Session) writeRec(kind string, f map[string]any) {
	if f == nil {
		f = map[string]any{}
	}
	f["v"], f["kind"] = SessionSchema, kind
	ts := s.elapsed()
	f["t_s"], f["t_wall"] = round1(ts), s.cfg.Now().UTC().Format(time.RFC3339)
	b, _ := json.Marshal(f)
	n, _ := s.rec.Write(append(b, '\n'))
	s.bytesRec += int64(n)
	if ts < 600 && len(s.timeline) < 40 && kind != "start" {
		label, _ := f["point"].(string)
		s.timeline = append(s.timeline, TimelineItem{TS: round1(ts), Kind: kind, Label: label})
	}
}

func (s *Session) writeDec(v any) {
	b, _ := json.Marshal(v)
	n, _ := s.dec.Write(append(b, '\n'))
	s.bytesDec += int64(n)
	var k struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(b, &k)
	s.decKind[k.Kind] += int64(n)
}

func round1(f float64) float64 { return float64(int64(f*10+0.5)) / 10 }

// gameElapsed is game seconds since the start; the ten-minute buckets and the per-hour figures use it, because under
// lockstep one wall minute holds many game minutes.
func (s *Session) gameElapsed() float64 { return (s.cfg.Clock.Now() - s.gameStart).Seconds() }

func (s *Session) bucket() int { return int(s.gameElapsed() / 600) }

func fromSource(src string) string {
	switch src {
	case "rule":
		return "rule"
	case "answer":
		return "player"
	}
	return "default"
}

// OnAsk is the store's ask observer: one decision-point firing.
func (s *Session) OnAsk(ev decide.AskEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	d, ctx := ev.Decision, ev.Ctx
	if ctx.Fields == nil {
		if c, err := decide.ParseKey(ctx.Key); err == nil {
			ctx.Fields = c.Fields
		}
	}
	by := fromSource(d.Source)
	now := s.cfg.Clock.Now()
	newThought := d.Thought != nil && !d.Thought.Confirm
	if newThought {
		_, dup := s.seen[d.Thought.ID]
		newThought = !dup
	}
	if newThought {
		s.sum.Questions++
		s.sum.questionsPerBucket[s.bucket()]++
	} else {
		k := ctx.Key + "|" + d.Source
		if at, ok := s.lastFire[k]; ok && now-at < time.Duration(s.cfg.DedupeSec)*time.Second {
			s.sum.FiresSuppressed++
			s.sum.FiresBy[by]++
			return
		}
	}
	s.lastFire[ctx.Key+"|"+d.Source] = now
	p, err := s.cfg.Cat.Point(ctx.Point)
	if err != nil {
		return
	}
	var opts []string
	for _, a := range p.Answers {
		opts = append(opts, a.ID)
	}
	def := ev.Legacy
	if def == "" {
		def, _ = s.cfg.Cat.Default(ctx)
	}
	s.fireN++
	class, build := s.identity()
	if ctx.Fields == nil {
		ctx.Fields = map[string]string{}
	}
	s.writeDec(fireLine{V: SessionSchema, Kind: "fire", ID: s.fireN, Session: s.sid, Commit: s.cfg.Commit, TGame: now.Milliseconds(),
		TWall: s.cfg.Now().UTC().Format(time.RFC3339), Point: ctx.Point, Ctx: ctx.Fields, Class: class, Build: build,
		Options: opts, Default: def, Answer: d.Answer, By: by})
	s.sum.FiresLogged++
	s.sum.FiresBy[by]++
	po := &pendingOutcome{id: s.fireN, at: now, xp0: s.last.XP, deaths0: s.sum.Deaths}
	if newThought {
		s.seen[d.Thought.ID] = s.fireN
		s.openedAt[d.Thought.ID] = d.Thought.OpenedAt
		po.thoughtID = d.Thought.ID
		s.writeRec("question", map[string]any{"thought": d.Thought.ID, "point": ctx.Point, "fire": s.fireN})
	}
	if len(s.pending) < 2000 {
		s.pending = append(s.pending, po)
	}
}

// onEvent runs under the store's lock: it must not call the store.
func (s *Session) onEvent(e decide.Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	switch e.Kind {
	case decide.EvRuleLearned:
		s.sum.RulesLearned++
		s.writeRec("rule", map[string]any{"key": e.Key, "answer": e.Answer})
	case decide.EvRevoked:
		s.sum.RulesRevoked++
		s.writeRec("revoke", map[string]any{"key": e.Key})
	}
}

// OnDeath records a hero death (call it from the goal event feed).
func (s *Session) OnDeath() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	s.sum.Deaths++
	s.writeRec("death", nil)
}

// SaveSessionNotes implements NotesSink: the three sentences of the return screen go into the record.
func (s *Session) SaveSessionNotes(text string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return errors.New("session: already stopped")
	}
	t := s.scrub(strings.TrimSpace(text))
	if len(t) > 2000 {
		t = t[:2000]
	}
	if s.sum.Notes != "" {
		t = s.sum.Notes + "\n" + t
	}
	s.sum.Notes = t
	s.writeRec("notes", map[string]any{"text": s.scrub(strings.TrimSpace(text))})
	return nil
}

// Poll reads what the store, the vitals, the viewers and the orders did since the last call, writes the answer and
// outcome lines and the periodic sample. Call it about once a second.
func (s *Session) Poll() {
	log := s.cfg.Store.Log() // before s.mu: the store's event callback takes s.mu under the store's lock
	var v Vitals
	if s.cfg.Vitals != nil {
		v = s.cfg.Vitals()
	}
	viewers := -1
	if s.cfg.Viewers != nil {
		viewers = s.cfg.Viewers()
	}
	var ords []Order
	if s.cfg.Orders != nil {
		ords = s.cfg.Orders()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return
	}
	now := s.cfg.Now()
	for ; s.logIdx < len(log); s.logIdx++ {
		s.onLog(log[s.logIdx])
	}
	if s.cfg.Vitals != nil {
		if !s.haveVit0 {
			s.vit0, s.haveVit0 = v, true
		}
		s.last = v
	}
	if viewers >= 0 {
		if !s.lastPoll.IsZero() && s.viewing {
			s.sum.inTab += now.Sub(s.lastPoll)
		}
		if on := viewers > 0; on != s.viewing {
			s.viewing = on
			s.writeRec("viewers", map[string]any{"open": on, "n": viewers})
		}
	}
	for _, o := range ords {
		if s.seenOrd[o.ID] {
			continue
		}
		s.seenOrd[o.ID] = true
		s.sum.Orders++
		s.ordTimes = append(s.ordTimes, s.elapsed())
		s.writeRec("order", map[string]any{"order": o.ID, "order_kind": o.Kind, "area": o.Area, "minutes": o.Minutes, "forever": o.Forever})
	}
	if s.lastSamp.IsZero() || now.Sub(s.lastSamp) >= time.Duration(s.cfg.SampleSec)*time.Second {
		if !s.lastSamp.IsZero() && s.cfg.Vitals != nil {
			s.writeRec("sample", map[string]any{"life_pct": v.LifePct, "xp": v.XP - s.vit0.XP, "credits": v.Credits - s.vit0.Credits})
		}
		s.lastSamp = now
	}
	s.lastPoll = now
	s.outcomes(s.cfg.Clock.Now(), false)
	if class, build := s.identity(); class != "" && [2]string{class, build} != s.identSent {
		s.identSent = [2]string{class, build}
		s.writeRec("identity", map[string]any{"class": class, "build": build})
	}
}

func (s *Session) onLog(e decide.Entry) {
	fid, known := s.seen[e.Thought]
	lat := func() *int64 {
		o, ok := s.openedAt[e.Thought]
		if !ok || e.T < o {
			return nil
		}
		ms := (e.T - o).Milliseconds()
		return &ms
	}
	switch e.Op {
	case decide.OpPolicy:
		// a sheet row changed (the player, a Pact or the class start); the stance row is the fight stance
		s.writeRec("policy", map[string]any{"row": e.Key, "option": e.Answer})
	case decide.OpAnswer:
		if !known {
			return
		}
		l := lat()
		s.sum.Answered++
		if l != nil {
			s.lat = append(s.lat, *l)
		}
		s.writeDec(map[string]any{"v": SessionSchema, "kind": "answer", "of": fid, "answer": e.Answer, "by": "player", "latency_ms": l})
		s.writeRec("answer", map[string]any{"thought": e.Thought, "answer": e.Answer, "by": "player", "latency_ms": l})
	case decide.OpFade:
		if !known {
			return
		}
		s.sum.Faded++
		s.writeDec(map[string]any{"v": SessionSchema, "kind": "answer", "of": fid, "answer": nil, "by": "timeout", "latency_ms": nil})
		s.writeRec("fade", map[string]any{"thought": e.Thought, "by": "timeout"})
	}
}

// outcomes writes the outcome line of every firing whose window is over (all of them at the stop, marked partial).
func (s *Session) outcomes(now time.Duration, final bool) {
	win := time.Duration(s.cfg.WindowMin) * time.Minute
	keep := s.pending[:0]
	for _, p := range s.pending {
		if s.cfg.Vitals != nil && (!p.haveLife || s.last.LifePct < p.minLife) {
			p.minLife, p.haveLife = s.last.LifePct, true
		}
		el := now - p.at
		if el < win && !final {
			keep = append(keep, p)
			continue
		}
		line := map[string]any{"v": SessionSchema, "kind": "outcome", "of": p.id, "window_min": s.cfg.WindowMin, "deaths": s.sum.Deaths - p.deaths0}
		if s.cfg.Vitals != nil {
			line["life_pct_min"] = p.minLife
			if h := el.Hours(); h > 0 {
				line["xp_per_h"] = float64(int64(float64(s.last.XP-p.xp0)/h + 0.5))
			}
		}
		if el < win {
			line["partial"], line["window_min"] = true, int(el.Minutes())
		}
		s.writeDec(line)
	}
	s.pending = keep
}

// Stop ends the session: a last poll, the partial outcomes, the stop line, summary.json and summary.md. Idempotent.
func (s *Session) Stop() error {
	s.Poll()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.stopped {
		return nil
	}
	now := s.cfg.Now()
	s.outcomes(s.cfg.Clock.Now(), true)
	s.stopped = true
	s.cfg.Store.SetAskObserver(nil)
	s.writeRec("stop", nil)
	sum := s.finish(now)
	err := errors.Join(s.rec.Close(), s.dec.Close())
	b, _ := json.MarshalIndent(sum, "", "  ")
	err = errors.Join(err, os.WriteFile(filepath.Join(s.dir, "summary.json"), append(b, '\n'), 0o600),
		os.WriteFile(filepath.Join(s.dir, "summary.md"), []byte(sum.Markdown()), 0o600))
	return err
}

func (s *Session) finish(now time.Time) Summary {
	sum := s.sum
	sum.End = now.UTC().Format(time.RFC3339)
	sum.Class, sum.Build = s.identity()
	wall := now.Sub(s.start)
	dur := s.cfg.Clock.Now() - s.gameStart
	sum.DurationS = round1(wall.Seconds())
	sum.GameDurationS = round1(dur.Seconds())
	if wall > 0 {
		sum.WallToGame = round1(dur.Seconds() / wall.Seconds())
	}
	n := int(dur/(10*time.Minute)) + 1
	sum.QuestionsPer10Min = make([]int, n)
	for b, c := range s.sum.questionsPerBucket {
		if b < n {
			sum.QuestionsPer10Min[b] = c
		}
	}
	if len(s.lat) > 0 {
		l := slices.Clone(s.lat)
		slices.Sort(l)
		m := l[len(l)/2]
		sum.LatencyMedianMS = &m
	}
	if len(s.ordTimes) > 1 {
		g := (s.ordTimes[len(s.ordTimes)-1] - s.ordTimes[0]) / float64(len(s.ordTimes)-1)
		g = round1(g)
		sum.SecBetweenOrders = &g
	}
	sum.StayedInTabMin = round1(s.sum.inTab.Minutes())
	if h := dur.Hours(); h > 0 {
		if s.haveVit0 {
			sum.XPPerH = float64(int64(float64(s.last.XP-s.vit0.XP)/h + 0.5))
		}
		sum.DecisionBytesPerH = float64(int64(float64(s.bytesDec) / h))
	}
	if h := wall.Hours(); h > 0 {
		if s.haveVit0 {
			sum.XPPerWallH = float64(int64(float64(s.last.XP-s.vit0.XP)/h + 0.5))
		}
		sum.DecisionBytesPerWallH = float64(int64(float64(s.bytesDec) / h))
	}
	sum.DecisionBytesByKind = maps.Clone(s.decKind)
	sum.Credits = s.last.Credits - s.vit0.Credits
	if s.cfg.WhyCounts != nil {
		sum.WhyChain = s.cfg.WhyCounts()
	}
	sum.First10 = s.timeline
	if sum.First10 == nil {
		sum.First10 = []TimelineItem{}
	}
	sum.RecordBytes, sum.DecisionBytes = s.bytesRec, s.bytesDec
	return sum
}

// Markdown is summary.md.
func (m Summary) Markdown() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Play session %s\n\n", m.Session)
	fmt.Fprintf(&b, "- Length: %.1f min wall, %.1f min game (x%.1f), %s to %s, commit %s, %s %s\n", m.DurationS/60, m.GameDurationS/60, m.WallToGame, m.Start, m.End, m.Commit, m.Class, m.Build)
	fmt.Fprintf(&b, "- Questions: %d asked, %d answered, %d faded; per 10 minutes: %v\n", m.Questions, m.Answered, m.Faded, m.QuestionsPer10Min)
	fmt.Fprintf(&b, "- Rules learned: %d (revoked %d); orders given: %d", m.RulesLearned, m.RulesRevoked, m.Orders)
	if m.SecBetweenOrders != nil {
		fmt.Fprintf(&b, ", %.0f s between orders", *m.SecBetweenOrders)
	}
	fmt.Fprintf(&b, "\n- In the tab: %.1f min; deaths: %d; XP per game hour: %.0f (per wall hour %.0f); credits: %d\n", m.StayedInTabMin, m.Deaths, m.XPPerH, m.XPPerWallH, m.Credits)
	if m.LatencyMedianMS != nil {
		fmt.Fprintf(&b, "- Median answer latency: %d ms\n", *m.LatencyMedianMS)
	}
	fmt.Fprintf(&b, "- Decided by: %v (logged %d, repeats not logged %d)\n", m.FiresBy, m.FiresLogged, m.FiresSuppressed)
	fmt.Fprintf(&b, "- Dataset: %d bytes (%.0f per game hour, %.0f per wall hour); record %d bytes\n", m.DecisionBytes, m.DecisionBytesPerH, m.DecisionBytesPerWallH, m.RecordBytes)
	if m.Notes != "" {
		fmt.Fprintf(&b, "\n## Notes\n\n%s\n", m.Notes)
	}
	return b.String()
}
