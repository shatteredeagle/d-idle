// Package ops is the operator control of the quest board: a local unix socket through which someone who can already
// reach the container (docker exec) presses the same board verbs the viewer buttons press, without the viewer's
// passcode. Off by default, quest verbs only, every mutating verb audited and rate limited. docs/operator.md.
package ops

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/questboard"
)

// SockName is the socket file inside the instance state directory.
const SockName = "operator.sock"

const (
	MaxWhy       = 200
	PerQuestHour = 6
	PerTotalHour = 30
	RetryCool    = 60 * time.Second
)

// The allowlist. Nothing else is dispatched, whatever the request says. mode and dismiss are board verbs too, but the
// operator does not get them: they change what the player sees or how the bot plays, not a quest.
var mutating = map[string]bool{"start": true, "retry": true, "cancel": true, "handin": true, "auto": true}

// orderVerbs are the ADR 0030 amendment of 2026-11-25: they place the orders the viewer already has (go, fight, home)
// or park a quest; they never touch the hero directly. why is the read verb of the decision ring.
var orderVerbs = map[string]bool{"hold": true, "home": true, "go": true, "park": true}

// DifficultyVerb is the owner's go for the next difficulty (user decision 2026-11-27: the hop is opt-in). The bot
// refuses it with a reason unless the hero is ready; it never hops alone.
const DifficultyVerb = "go-difficulty"

func allowed(v string) bool {
	return mutating[v] || orderVerbs[v] || v == "status" || v == "why" || v == DifficultyVerb
}

// Park limits: a quest is set aside for 1 to MaxParkHours hours (the operator's own wake time, kept in process memory).
const (
	MaxParkHours  = 24
	DefaultGoMins = 60
	ordersBucket  = "_orders" // the rate-limit bucket of the order verbs (they also count against the 30 per hour)
)

// Orders places the T0 orders (go, fight, home) the way the viewer's order box does: the hero's own guards still decide
// when the walk starts. Kind is "go", "fight" or "home"; replace cancels every open order first.
type Orders interface {
	Give(kind, area string, minutes int, replace bool) (text string, err error)
}

// DifficultyGo records the owner's go for a difficulty hop ("nightmare" or "hell") or refuses with the reason.
type DifficultyGo interface {
	RequestDifficultyGo(target string) (text string, err error)
}

// Desk is what the operator verbs need from the board (questboard.Desk; the same calls the HTTP handlers make).
type Desk interface {
	View() questboard.View
	Start(quest, chunk string) questboard.Reply
	StartAnyway(quest, chunk string) questboard.Reply
	HandIn(quest, chunk string) questboard.Reply
	Cancel(quest, chunk string) questboard.Reply
	Retry(quest, chunk string) questboard.Reply
	SetAuto(quest string, on bool) questboard.Reply
	Record(questboard.AuditEvent)
}

// Request is one line of JSON on the socket.
type Request struct {
	Verb  string `json:"verb"`
	Quest string `json:"quest,omitempty"`
	Chunk string `json:"chunk,omitempty"`
	On    *bool  `json:"on,omitempty"`
	// Area and Minutes belong to go; Hours to park.
	Area    string `json:"area,omitempty"`
	Minutes int    `json:"minutes,omitempty"`
	Hours   int    `json:"hours,omitempty"`
	// Target belongs to go-difficulty: "nightmare" or "hell".
	Target string `json:"target,omitempty"`
	Why    string `json:"why,omitempty"`
	// TryAnyway turns start into the owner's try-anyway start: one attempt past a readiness refusal (docs/operator.md).
	TryAnyway bool `json:"try_anyway,omitempty"`
}

// Response is the one line answered. Reply is the desk's own answer of a mutating verb; Board answers status.
type Response struct {
	OK     bool              `json:"ok"`
	Code   string            `json:"code,omitempty"`
	Reason string            `json:"reason,omitempty"`
	Reply  *questboard.Reply `json:"reply,omitempty"`
	Board  *Board            `json:"board,omitempty"`
	// Explain answers why: the bot's decision ring and standing refusals (bot.ExplainView), read only.
	Explain any `json:"explain,omitempty"`
}

// Board is the status view: one row per chunk of the difficulty.
type Board struct {
	Enabled    bool   `json:"enabled"`
	Difficulty string `json:"difficulty,omitempty"`
	Why        string `json:"why,omitempty"`
	Rows       []Row  `json:"rows"`
	// Parked lists the quests the operator set aside, with the wake time (RFC 3339).
	Parked []Parked `json:"parked,omitempty"`
}

// Parked is one quest set aside by the park verb.
type Parked struct {
	Quest string `json:"quest"`
	Until string `json:"until"`
}

// Row is a chunk with its state, its park reason (Code and Text) and its step while it runs.
type Row struct {
	Quest string `json:"quest"`
	Chunk string `json:"chunk"`
	State string `json:"state"`
	Step  string `json:"step,omitempty"`
	// Progress counts the changes of state or step the server has seen for this chunk since it started; a wait that
	// times out with a moved counter is slow progress, not a stall.
	Progress int    `json:"progress,omitempty"`
	Code     string `json:"code,omitempty"`
	Text     string `json:"text,omitempty"`
	// WhyNot is the chunk's "Why not?" row (readiness word, floor, need, wake, last refusal); `status --why` prints it.
	WhyNot *questboard.WhyNotView `json:"whyNot,omitempty"`
}

func boardOf(v questboard.View) *Board {
	b := &Board{Enabled: v.Enabled, Difficulty: v.Difficulty, Why: v.Reason, Rows: []Row{}}
	runs := []*questboard.RunView{v.Running, v.Awaiting, v.Blocked}
	for _, q := range v.Quests {
		for _, c := range q.Chunks {
			r := Row{Quest: q.ID, Chunk: c.ID, State: string(c.State), WhyNot: c.WhyNot}
			if c.Reason != nil {
				r.Code, r.Text = c.Reason.Code, c.Reason.Text
			}
			for _, rv := range runs {
				if rv == nil || rv.Quest != q.ID || rv.Chunk != c.ID {
					continue
				}
				if rv.Step != "" {
					r.Step = rv.Step
				}
				if r.Code == "" && rv.Reason != nil {
					r.Code, r.Text = rv.Reason.Code, rv.Reason.Text
				}
			}
			b.Rows = append(b.Rows, r)
		}
	}
	return b
}

// Enabled says whether the operator socket may exist: only with D2BOT_OPERATOR=1 and never in a measured run.
func Enabled(getenv func(string) string) (bool, string) {
	if getenv("D2BOT_OPERATOR") != "1" {
		return false, "operator control is off (D2BOT_OPERATOR is not 1)"
	}
	if getenv("D2BOT_RUN") != "" {
		return false, "operator control is off in a measured run (D2BOT_RUN is set)"
	}
	return true, ""
}

// Config wires a Server. Getenv and Now default to the process; Logf may be nil.
type Config struct {
	Desk       func() Desk
	Orders     func() Orders       // nil: the order verbs answer "off"
	Explain    func() any          // nil: why answers "off"
	Difficulty func() DifficultyGo // nil: go-difficulty answers "off"
	Getenv     func(string) string
	Now        func() time.Time
	Logf       func(msg string, kv ...any)
}

// Server dispatches requests. Safe for concurrent use.
type Server struct {
	cfg  Config
	mu   sync.Mutex
	all  []time.Time
	per  map[string][]time.Time
	last map[string]time.Time // last retry per quest
	prog map[string]*progress
	park map[string]time.Time // quest -> wake time of the park verb
}

func New(c Config) *Server {
	if c.Now == nil {
		c.Now = time.Now
	}
	if c.Getenv == nil {
		c.Getenv = func(string) string { return "" }
	}
	if c.Logf == nil {
		c.Logf = func(string, ...any) {}
	}
	return &Server{cfg: c, per: map[string][]time.Time{}, last: map[string]time.Time{}, prog: map[string]*progress{}, park: map[string]time.Time{}}
}

func no(code, reason string) Response { return Response{Code: code, Reason: reason} }

func safeID(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '_' || c == '-') {
			return false
		}
	}
	return true
}

func cleanWhy(s string) (string, bool) {
	s = strings.TrimSpace(s)
	if s == "" || len(s) > MaxWhy {
		return "", false
	}
	for _, c := range s {
		if c < ' ' || c == 0x7f {
			return "", false
		}
	}
	return s, true
}

// admit applies the rate limits and books the verb on success.
func (s *Server) admit(verb, quest string) *Response {
	now := s.cfg.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	prune := func(l []time.Time) []time.Time {
		i := 0
		for i < len(l) && now.Sub(l[i]) >= time.Hour {
			i++
		}
		return l[i:]
	}
	s.all = prune(s.all)
	s.per[quest] = prune(s.per[quest])
	if t, ok := s.last[quest]; ok && (verb == "retry" || verb == "start") && now.Sub(t) < RetryCool {
		r := no("cooldown", "a retry of this quest was pressed less than a minute ago; wait "+(RetryCool-now.Sub(t)).Round(time.Second).String())
		return &r
	}
	if len(s.per[quest]) >= PerQuestHour {
		r := no("rate_limited", "this quest already had 6 operator verbs in the last hour")
		return &r
	}
	if len(s.all) >= PerTotalHour {
		r := no("rate_limited", "30 operator verbs in the last hour is the limit for the whole board")
		return &r
	}
	s.all = append(s.all, now)
	s.per[quest] = append(s.per[quest], now)
	if verb == "retry" {
		s.last[quest] = now
	}
	return nil
}

// Handle runs one request. Every refusal before the desk is logged, not written to the audit ring (a runaway loop
// must not push real events out of it); everything the desk ran is audited with remote "operator".
func (s *Server) Handle(rq Request) Response {
	rsp := s.handle(rq)
	if !rsp.OK && rsp.Reply == nil {
		s.cfg.Logf("operator verb refused", "verb", rq.Verb, "quest", rq.Quest, "code", rsp.Code)
	}
	return rsp
}

func (s *Server) handle(rq Request) Response {
	if !allowed(rq.Verb) {
		return no("verb", "only start, retry, cancel, handin, auto, hold, home, go, park, go-difficulty, status and why are served here")
	}
	if rq.Verb == "why" {
		if s.cfg.Explain == nil {
			return no("off", "the decision ring is not available yet")
		}
		return Response{OK: true, Explain: s.cfg.Explain()}
	}
	var d Desk
	if s.cfg.Desk != nil {
		d = s.cfg.Desk()
	}
	if d == nil {
		return no("off", "the quest board is not available yet")
	}
	if rq.Verb == "status" {
		b := boardOf(d.View())
		s.track(b)
		b.Parked = s.parkedList()
		return Response{OK: true, Board: b}
	}
	if s.cfg.Getenv("D2BOT_RUN") != "" {
		return no("measured_run", "a measured run never takes operator verbs (D2BOT_RUN is set)")
	}
	if orderVerbs[rq.Verb] {
		return s.order(d, rq)
	}
	if rq.Verb == DifficultyVerb {
		return s.goDifficulty(d, rq)
	}
	if !safeID(rq.Quest) {
		return no("quest", "send a quest id (a-z, 0-9, _ and -, at most 64)")
	}
	if rq.Chunk != "" && !safeID(rq.Chunk) {
		return no("chunk", "the chunk id has characters a chunk never has")
	}
	why, ok := cleanWhy(rq.Why)
	if !ok {
		return no("why", "every mutating verb needs a reason: one line, 1 to 200 characters")
	}
	if rq.TryAnyway && rq.Verb != "start" {
		return no("body", "try_anyway belongs to the start verb only")
	}
	if rq.Verb == "auto" && rq.On == nil {
		return no("body", "auto needs on or off")
	}
	if (rq.Verb == "start" || rq.Verb == "retry") && s.parkedNow(rq.Quest) != "" {
		return no("parked", rq.Quest+" is parked by the operator until "+s.parkedNow(rq.Quest))
	}
	if rq.Verb == "retry" {
		if r := notParked(d.View(), rq.Quest, rq.Chunk); r != nil {
			return *r
		}
	}
	if r := s.admit(rq.Verb, rq.Quest); r != nil {
		return *r
	}
	var rep questboard.Reply
	switch rq.Verb {
	case "start":
		if rq.TryAnyway {
			rep = d.StartAnyway(rq.Quest, rq.Chunk)
		} else {
			rep = d.Start(rq.Quest, rq.Chunk)
		}
	case "handin":
		rep = d.HandIn(rq.Quest, rq.Chunk)
	case "cancel":
		rep = d.Cancel(rq.Quest, rq.Chunk)
	case "retry":
		rep = d.Retry(rq.Quest, rq.Chunk)
	case "auto":
		rep = d.SetAuto(rq.Quest, *rq.On)
	}
	st := rep.Status
	if st == 0 {
		st = 200
	}
	path, auditWhy := "operator/"+rq.Verb, why
	if rq.TryAnyway {
		// the override is its own audit path and carries the refusal it skipped
		path, auditWhy = "operator/try-anyway", "override: try anyway: "+why
		if rep.Override != "" {
			auditWhy += " (refused: " + rep.Override + ")"
		}
		s.cfg.Logf("override: try anyway", "quest", rq.Quest, "chunk", rq.Chunk, "ok", rep.OK, "code", rep.Code, "refused", rep.Override, "why", why)
	}
	d.Record(questboard.AuditEvent{Action: rq.Verb, Quest: rq.Quest, Chunk: rq.Chunk, Path: path,
		Remote: "operator", Agent: "ops-cli", Why: auditWhy, OK: rep.OK, Already: rep.Already, Status: st, Code: rep.Code, Reason: rep.Reason})
	s.cfg.Logf("operator verb", "verb", rq.Verb, "quest", rq.Quest, "chunk", rq.Chunk, "ok", rep.OK, "code", rep.Code)
	return Response{OK: rep.OK, Code: rep.Code, Reason: rep.Reason, Reply: &rep}
}

type progress struct {
	sig string
	n   int
}

// track numbers the state and step changes of each chunk across status calls.
func (s *Server) track(b *Board) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range b.Rows {
		r := &b.Rows[i]
		k := r.Quest + "/" + r.Chunk
		p := s.prog[k]
		if p == nil {
			p = &progress{sig: r.State + "|" + r.Step}
			s.prog[k] = p
		}
		if sig := r.State + "|" + r.Step; sig != p.sig {
			p.sig, p.n = sig, p.n+1
		}
		r.Progress = p.n
	}
}

// notParked refuses a retry unless a matching chunk of the quest is blocked: retry clears a park, and on a chunk that
// runs it would disturb the run (it put a running chunk into "trying again" live).
func notParked(v questboard.View, quest, chunk string) *Response {
	state := ""
	for _, q := range v.Quests {
		if q.ID != quest {
			continue
		}
		for _, c := range q.Chunks {
			if chunk != "" && c.ID != chunk {
				continue
			}
			if c.State == questboard.Blocked {
				return nil
			}
			if state == "" || c.State == questboard.InProgress || c.State == questboard.Accepted {
				state = string(c.State)
			}
		}
	}
	if state == "" {
		state = "not on the board"
	}
	r := no("not_parked", "retry only clears a park; "+quest+" is "+state+", not blocked")
	return &r
}

func (s *Server) parkedNow(quest string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.park[quest]
	if !ok || !s.cfg.Now().Before(t) {
		delete(s.park, quest)
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func (s *Server) parkedList() []Parked {
	var out []Parked
	s.mu.Lock()
	qs := make([]string, 0, len(s.park))
	for q := range s.park {
		qs = append(qs, q)
	}
	s.mu.Unlock()
	sort.Strings(qs)
	for _, q := range qs {
		if u := s.parkedNow(q); u != "" {
			out = append(out, Parked{Quest: q, Until: u})
		}
	}
	return out
}

// order serves hold, home, go and park. hold stops what the hero does and sends it to rest in town (a home order that
// replaces the open orders); home queues the same order behind them; go places a go order for Minutes (default 60);
// park stops the quest's running chunk and refuses start and retry of it until the wake time. Each needs a reason, is
// rate limited with the board's budget and is an audit line with remote "operator".
func (s *Server) order(d Desk, rq Request) Response {
	why, ok := cleanWhy(rq.Why)
	if !ok {
		return no("why", "every mutating verb needs a reason: one line, 1 to 200 characters")
	}
	bucket := ordersBucket
	if rq.Verb == "park" {
		if !safeID(rq.Quest) {
			return no("quest", "send a quest id (a-z, 0-9, _ and -, at most 64)")
		}
		if rq.Hours < 1 || rq.Hours > MaxParkHours {
			return no("hours", "park takes 1 to 24 hours")
		}
		bucket = rq.Quest
	} else if s.cfg.Orders == nil || s.cfg.Orders() == nil {
		return no("off", "orders are not available yet")
	}
	if r := s.admit(rq.Verb, bucket); r != nil {
		return *r
	}
	var rep questboard.Reply
	auditWhy := why
	switch rq.Verb {
	case "park":
		rep = d.Cancel(rq.Quest, "")
		until := s.cfg.Now().Add(time.Duration(rq.Hours) * time.Hour)
		s.mu.Lock()
		s.park[rq.Quest] = until
		s.mu.Unlock()
		rep.OK = true
		rep.Reason = rq.Quest + " parked until " + until.UTC().Format(time.RFC3339) + ". " + rep.Reason
		auditWhy = fmt.Sprintf("park %dh: %s", rq.Hours, why)
	default:
		kind, area, mins, replace := "home", "", 0, rq.Verb == "hold"
		if rq.Verb == "go" {
			kind, area, mins = "go", rq.Area, rq.Minutes
			if mins == 0 {
				mins = DefaultGoMins
			}
		}
		text, err := s.cfg.Orders().Give(kind, area, mins, replace)
		rep.OK, rep.Reason = err == nil, text
		if err != nil {
			rep.Code, rep.Reason = "order_refused", err.Error()
		}
		auditWhy = rq.Verb + ": " + why
	}
	st := 200
	if !rep.OK {
		st = 409
	}
	d.Record(questboard.AuditEvent{Action: rq.Verb, Quest: rq.Quest, Path: "operator/" + rq.Verb, Remote: "operator",
		Agent: "ops-cli", Why: auditWhy, OK: rep.OK, Status: st, Code: rep.Code, Reason: rep.Reason})
	s.cfg.Logf("operator verb", "verb", rq.Verb, "quest", rq.Quest, "area", rq.Area, "ok", rep.OK, "code", rep.Code)
	return Response{OK: rep.OK, Code: rep.Code, Reason: rep.Reason, Reply: &rep}
}

// goDifficulty serves go-difficulty: needs a reason, is rate limited with the board's budget and audited with remote
// "operator". The bot refuses it (order_refused) with the missing requirements unless the hero is ready.
func (s *Server) goDifficulty(d Desk, rq Request) Response {
	why, ok := cleanWhy(rq.Why)
	if !ok {
		return no("why", "every mutating verb needs a reason: one line, 1 to 200 characters")
	}
	if rq.Target != "nightmare" && rq.Target != "hell" {
		return no("target", "go-difficulty takes nightmare or hell")
	}
	var g DifficultyGo
	if s.cfg.Difficulty != nil {
		g = s.cfg.Difficulty()
	}
	if g == nil {
		return no("off", "the difficulty hop is not available yet")
	}
	if r := s.admit(rq.Verb, "_difficulty"); r != nil {
		return *r
	}
	text, err := g.RequestDifficultyGo(rq.Target)
	rep := questboard.Reply{OK: err == nil, Reason: text}
	st := 200
	if err != nil {
		rep.Code, rep.Reason, st = "order_refused", err.Error(), 409
	}
	d.Record(questboard.AuditEvent{Action: rq.Verb, Quest: rq.Target, Path: "operator/" + rq.Verb, Remote: "operator",
		Agent: "ops-cli", Why: rq.Verb + " " + rq.Target + ": " + why, OK: rep.OK, Status: st, Code: rep.Code, Reason: rep.Reason})
	s.cfg.Logf("operator verb", "verb", rq.Verb, "target", rq.Target, "ok", rep.OK, "code", rep.Code)
	return Response{OK: rep.OK, Code: rep.Code, Reason: rep.Reason, Reply: &rep}
}
