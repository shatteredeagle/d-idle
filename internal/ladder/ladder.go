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

// Config builds a Ladder. Store and Cat are required; Rules defaults to the shipped rule file.
type Config struct {
	Cat     *decide.Catalogue
	Store   *decide.Store
	Rules   *decide.RuleFile
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
	cfg  Config
	mu   sync.Mutex // call sites
	qmu  sync.Mutex // the queue the store's events fill
	q    []*pend
	stat Stats
}

// Stats is the ladder's run record, next to the store's own counters.
type Stats struct {
	Asked, ByRule, ByDefault, ByFile int
	Opened, Answered, Faded, Learned int
	Faults                           int
	LastFault                        string
}

// New wires the ladder to its store events.
func New(c Config) (*Ladder, error) {
	if c.Cat == nil || c.Store == nil || c.Clock == nil {
		return nil, fmt.Errorf("ladder: catalogue, store and clock are required")
	}
	if c.Rules == nil {
		f, err := decide.ShippedRuleFile()
		if err != nil {
			return nil, err
		}
		c.Rules = f
	}
	if c.Sink == nil {
		c.Sink = nopSink{}
	}
	l := &Ladder{cfg: c}
	c.Store.Subscribe(l.onEvent)
	return l, nil
}

type nopSink struct{}

func (nopSink) Event(string, ThoughtView, string) {}

// onEvent runs inside the store: it counts and shows, and never calls back into the store.
func (l *Ladder) onEvent(e decide.Event) {
	switch e.Kind {
	case decide.EvThoughtFaded:
		l.qmu.Lock()
		l.stat.Faded++
		l.qmu.Unlock()
		l.cfg.Sink.Event("faded", ThoughtView{ID: e.Thought, Key: e.Key}, "")
	case decide.EvRuleLearned:
		l.qmu.Lock()
		l.stat.Learned++
		l.qmu.Unlock()
		l.cfg.Sink.Event("learned", ThoughtView{Key: e.Key}, e.Answer)
	}
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
// to act on now. Layers: learned rule, shipped rule file, catalogue default. Errors name the unknown class, point or
// field; the caller falls back to its legacy path.
func (l *Ladder) Decide(point string, h Hero, extra decide.Obs) (string, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if _, err := l.cfg.Store.Tick(); err != nil {
		return "", err
	}
	o := h.obs()
	for k, v := range extra.Num {
		o.Num[k] = v
	}
	for k, v := range extra.Cat {
		o.Cat[k] = v
	}
	ctx, err := l.cfg.Cat.Build(point, o, 1, nil)
	if err != nil {
		return "", err
	}
	d, err := l.cfg.Store.Ask(ctx)
	if err != nil {
		return "", err
	}
	ans := d.Answer
	l.qmu.Lock()
	l.stat.Asked++
	switch d.Source {
	case "rule":
		l.stat.ByRule++
	default:
		if a, ok := l.cfg.Rules.Lookup(ctx); ok {
			ans = a
			l.stat.ByFile++
		} else {
			l.stat.ByDefault++
		}
	}
	l.qmu.Unlock()
	l.pumpLocked()
	return ans, nil
}

// Pump puts queued thoughts to the answer source and delivers the replies that are due. The bot calls it from its
// loop outside fights; Decide calls it too.
func (l *Ladder) Pump() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.pumpLocked()
}

func (l *Ladder) view(t decide.Thought) ThoughtView {
	v := ThoughtView{ID: t.ID, Point: t.Point, Key: t.Key, Confirm: t.Confirm, OpenedAt: t.OpenedAt}
	if p, err := l.cfg.Cat.Point(t.Point); err == nil {
		for _, a := range p.Answers {
			v.Choices = append(v.Choices, a.ID)
		}
	}
	if ctx, err := decide.ParseKey(t.Key); err == nil {
		v.Default, _ = l.cfg.Cat.Default(ctx)
		if a, ok := l.cfg.Rules.Lookup(ctx); ok {
			v.Default = a
		}
	}
	return v
}

func (l *Ladder) pumpLocked() {
	if l.cfg.InFight != nil && l.cfg.InFight() {
		return
	}
	now := l.cfg.Clock.Now()
	work := l.sync()
	for _, p := range work {
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
		if err == nil {
			l.qmu.Lock()
			l.stat.Answered++
			l.qmu.Unlock()
			l.cfg.Sink.Event("answer", p.v, p.reply.Answer)
		}
	}
}

// sync reconciles the queue with the store's open thoughts: new ones are queued, closed ones (answered, faded) leave.
func (l *Ladder) sync() []*pend {
	open := l.cfg.Store.OpenThoughts()
	l.qmu.Lock()
	defer l.qmu.Unlock()
	have := map[int]*pend{}
	for _, p := range l.q {
		have[p.v.ID] = p
	}
	l.q = l.q[:0]
	for _, t := range open {
		p := have[t.ID]
		if p == nil {
			p = &pend{v: l.view(t)}
			l.stat.Opened++
		}
		l.q = append(l.q, p)
	}
	sort.Slice(l.q, func(i, j int) bool { return l.q[i].v.ID < l.q[j].v.ID })
	return append([]*pend(nil), l.q...)
}
