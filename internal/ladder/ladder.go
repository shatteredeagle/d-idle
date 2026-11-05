// Package ladder is the glue between the bot's code paths and the decision-point store (internal/decide): a wired
// decision point builds its context key from banded observations, asks the store for the action (a learned rule, else
// the untrained default from the shipped rule file, else the catalogue default) and acts on it. The hero never waits
// for a question: a thought that opens is queued, shown to a Sink, and put to an AnswerSource at the next safe moment.
//
// The package is off unless somebody installs a Ladder (Install, driven by the ladder.enabled knob). Nothing here talks
// to a person: answers come from an AnswerSource, which in tests and drills is a scripted file or a seeded profile.
package ladder

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
)

// Hero is what a call site reports about the character. Class comes from the class table of the catalogue, never from a
// literal: an unknown class is an error and the call site keeps its legacy path.
type Hero struct {
	Class      d2.Class
	Act        int // 0-based, as d2.Player.Act
	Diff       d2.Difficulty
	Hardcore   bool
	Gold       int
	AreaClass  string // "town", "field" or "boss"
	Durability int    // percent, lowest worn item; -1 unknown
}

// HeroFrom reads the fields every point shares from a snapshot player. Durability stays -1 and AreaClass is derived.
func HeroFrom(p *d2.Player) Hero {
	h := Hero{Durability: -1, AreaClass: "field"}
	if p == nil {
		return h
	}
	h.Class, h.Act, h.Diff, h.Gold = p.Class, p.Act, p.Difficulty, p.Gold
	if p.InTown() {
		h.AreaClass = "town"
	}
	return h
}

func (h Hero) obs() decide.Obs {
	diff := "normal"
	switch h.Diff {
	case d2.Nightmare:
		diff = "nightmare"
	case d2.Hell:
		diff = "hell"
	}
	hc := "normal"
	if h.Hardcore {
		hc = "hardcore"
	}
	return decide.Obs{Class: strings.ToLower(h.Class.String()),
		Num: map[string]float64{"gold": float64(h.Gold), "durability": float64(h.Durability)},
		Cat: map[string]string{"act": fmt.Sprint(h.Act + 1), "diff": diff, "hc": hc, "areaclass": h.AreaClass}}
}

// ThoughtView is what a Sink and an AnswerSource see of an open thought.
type ThoughtView struct {
	ID       int
	Point    string
	Key      string
	Confirm  bool     // a confirm thought: reply "yes", "no" or "" (look away)
	Choices  []string // answer ids of the point
	Default  string   // the action taken meanwhile
	OpenedAt time.Duration
}

// Reply is the scripted player's answer: "" lets the thought fade, "notsure" or an answer id answers it. Delay is the
// virtual time the player takes.
type Reply struct {
	Answer string
	Delay  time.Duration
}

// AnswerSource decides what the (scripted) player replies. It must not read stdin or the network.
type AnswerSource interface{ Reply(ThoughtView) Reply }

// Sink receives the visible side of the ladder: thoughts opening, answers, fades, rules learned.
type Sink interface {
	Event(kind string, v ThoughtView, detail string)
}

// ClockFunc adapts a virtual-time function (the bot's clock since its start) to decide.Clock.
type ClockFunc func() time.Duration

func (f ClockFunc) Now() time.Duration { return f() }

// Config builds a Ladder. Store, Cat and Clock are required.
type Config struct {
	Cat     *decide.Catalogue
	Store   *decide.Store
	Clock   decide.Clock
	Sink    Sink
	Answers AnswerSource
	// InFight reports a fight in progress: questions are held until it is false (the hero acts on the default now).
	InFight func() bool
}

type pend struct {
	v        ThoughtView
	asked    bool
	answerAt time.Duration
	reply    Reply
}

// Ladder asks the store at the wired points. Safe for concurrent use.
type Ladder struct {
	cfg    Config
	mu     sync.Mutex // call sites; taken before the store's lock
	qmu    sync.Mutex // the queue and counters the store's events fill; never held while calling the store
	q      []*pend
	legacy map[string]string // context key -> the legacy answer last given there (the default a thought shows)
	stat   Stats
}

// Stats is the ladder's run record, next to the store's own counters.
type Stats struct {
	Asked, ByRule, ByLegacy, ByDefault int
	Opened, Answered, Faded, Learned   int
	Faults                             int
	LastFault                          string
}

// New wires the ladder to its store events.
func New(c Config) (*Ladder, error) {
	if c.Cat == nil || c.Store == nil || c.Clock == nil {
		return nil, fmt.Errorf("ladder: catalogue, store and clock are required")
	}
	if c.Sink == nil {
		c.Sink = nopSink{}
	}
	l := &Ladder{cfg: c, legacy: map[string]string{}}
	c.Store.Subscribe(l.onEvent)
	// thoughts a persisted store reopened never produced an event
	for _, t := range c.Store.OpenThoughts() {
		if ti, ok := c.Store.Thought(t.ID); ok {
			l.q = append(l.q, &pend{v: l.view(ti, ti.Expires-time.Duration(c.Cat.Config.FadeSec)*time.Second)})
			l.stat.Opened++
		}
	}
	return l, nil
}

type nopSink struct{}

func (nopSink) Event(string, ThoughtView, string) {}

// onEvent runs inside the store: it queues, counts and shows, and never calls back into the store.
func (l *Ladder) onEvent(e decide.Event) {
	switch e.Kind {
	case decide.EvThoughtOpened, decide.EvConfirmOpened:
		if e.Info == nil {
			return
		}
		l.qmu.Lock()
		l.q = append(l.q, &pend{v: l.view(*e.Info, l.cfg.Clock.Now())})
		l.stat.Opened++
		l.qmu.Unlock()
	case decide.EvThoughtFaded:
		l.qmu.Lock()
		l.stat.Faded++
		l.drop(e.Thought)
		l.qmu.Unlock()
		l.cfg.Sink.Event("faded", ThoughtView{ID: e.Thought, Key: e.Key}, "")
	case decide.EvRuleLearned:
		l.qmu.Lock()
		l.stat.Learned++
		l.qmu.Unlock()
		l.cfg.Sink.Event("learned", ThoughtView{Key: e.Key}, e.Answer)
	}
}

// drop removes a thought from the queue (qmu held).
func (l *Ladder) drop(id int) {
	l.q = slices.DeleteFunc(l.q, func(p *pend) bool { return p.v.ID == id })
}

// Fault is called by a call site when Decide failed: the site then runs its legacy path. Counted, never fatal.
func (l *Ladder) Fault(err error) {
	l.qmu.Lock()
	l.stat.Faults++
	l.stat.LastFault = err.Error()
	l.qmu.Unlock()
}

// Stats returns a copy of the counters.
func (l *Ladder) Stats() Stats {
	l.qmu.Lock()
	defer l.qmu.Unlock()
	return l.stat
}

// Decide builds the context of a point from the hero and extra observations, asks the store and returns the answer id
// to act on now. legacy is the answer the old code gives here (an answer id of the point): a learned rule wins, else
// legacy is acted on, so behaviour equals the old code until a person teaches a rule. legacy "" uses the catalogue
// default instead. Errors name the unknown class, point or field; the caller falls back to its legacy path.
func (l *Ladder) Decide(point string, h Hero, extra decide.Obs, legacy string) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.cfg.Store.Tick(); err != nil {
		return "", err
	}
	ctx, err := l.context(point, h, extra)
	if err != nil {
		return "", err
	}
	l.qmu.Lock()
	if legacy != "" {
		l.legacy[ctx.Key] = legacy // before the ask: the thought it opens shows this default
	}
	l.qmu.Unlock()
	d, err := l.cfg.Store.AskWithDefault(ctx, legacy)
	if err != nil {
		return "", err
	}
	l.qmu.Lock()
	l.stat.Asked++
	switch d.Source {
	case "rule":
		l.stat.ByRule++
	case "legacy":
		l.stat.ByLegacy++
	default:
		l.stat.ByDefault++
	}
	l.qmu.Unlock()
	l.pumpLocked()
	return d.Answer, nil
}

func (l *Ladder) context(point string, h Hero, extra decide.Obs) (decide.Context, error) {
	o := h.obs()
	for k, v := range extra.Num {
		o.Num[k] = v
	}
	for k, v := range extra.Cat {
		o.Cat[k] = v
	}
	return l.cfg.Cat.Build(point, o, 1, nil)
}

// CatalogueDefault is the untrained default of the catalogue at this state: what the store would answer without a
// legacy answer. Drills and parity tests compare it with the legacy code; the call sites do not use it.
func (l *Ladder) CatalogueDefault(point string, h Hero, extra decide.Obs) (string, error) {
	ctx, err := l.context(point, h, extra)
	if err != nil {
		return "", err
	}
	return l.cfg.Cat.Default(ctx)
}

// Pump puts queued thoughts to the answer source and delivers the replies that are due. The bot calls it from its
// loop outside fights; Decide calls it too.
func (l *Ladder) Pump() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pumpLocked()
}

// view turns the store's public read of a thought into what the sink and the answer source see. The default shown is
// the action the hero took meanwhile: the legacy answer given at that key, else the catalogue default.
func (l *Ladder) view(ti decide.ThoughtInfo, opened time.Duration) ThoughtView {
	v := ThoughtView{ID: ti.ID, Point: ti.Point, Key: ti.Key, Confirm: ti.Confirm, OpenedAt: opened}
	for _, a := range ti.Choices {
		v.Choices = append(v.Choices, a.ID)
	}
	if d, ok := l.legacy[ti.Key]; ok {
		v.Default = d
	} else if ctx, err := decide.ParseKey(ti.Key); err == nil {
		v.Default, _ = l.cfg.Cat.Default(ctx)
	}
	return v
}

func (l *Ladder) pumpLocked() {
	if l.cfg.InFight != nil && l.cfg.InFight() {
		return
	}
	now := l.cfg.Clock.Now()
	l.qmu.Lock()
	work := append([]*pend(nil), l.q...)
	l.qmu.Unlock()
	sort.Slice(work, func(i, j int) bool { return work[i].v.ID < work[j].v.ID })
	for _, p := range work {
		if _, open := l.cfg.Store.Thought(p.v.ID); !open { // answered elsewhere or faded
			l.qmu.Lock()
			l.drop(p.v.ID)
			l.qmu.Unlock()
			continue
		}
		if !p.asked {
			p.asked = true
			if !p.v.Confirm {
				l.cfg.Sink.Event("thought", p.v, "")
			}
			if l.cfg.Answers == nil {
				continue
			}
			p.reply = l.cfg.Answers.Reply(p.v)
			p.answerAt = max(now, p.v.OpenedAt) + p.reply.Delay
		}
		if l.cfg.Answers == nil || p.reply.Answer == "" || now < p.answerAt {
			continue
		}
		var err error
		if p.v.Confirm {
			err = l.cfg.Store.Confirm(p.v.ID, p.reply.Answer == "yes")
		} else {
			err = l.cfg.Store.Answer(p.v.ID, p.reply.Answer)
		}
		l.qmu.Lock()
		l.drop(p.v.ID) // a reply the store refused is not retried
		if err == nil {
			l.stat.Answered++
		}
		l.qmu.Unlock()
		if err == nil {
			l.cfg.Sink.Event("answer", p.v, p.reply.Answer)
		}
	}
}

// Open makes a ladder over the shipped catalogue and rule file. dir "" keeps the store in memory; otherwise the
// answers persist in dir (decide.Open, file decide-<profile>.json).
func Open(dir, profile string, clk decide.Clock, sink Sink, src AnswerSource, inFight func() bool) (*Ladder, error) {
	cat, err := decide.Shipped()
	if err != nil {
		return nil, err
	}
	var st *decide.Store
	if dir == "" {
		st, err = decide.New(cat, profile, clk)
	} else {
		st, err = decide.Open(dir, profile, cat, clk)
	}
	if err != nil {
		return nil, err
	}
	return New(Config{Cat: cat, Store: st, Clock: clk, Sink: sink, Answers: src, InFight: inFight})
}

// Store exposes the store for run records (rules, answer log) and for tests.
func (l *Ladder) Store() *decide.Store { return l.cfg.Store }
