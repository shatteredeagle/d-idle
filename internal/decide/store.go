package decide

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Log operations. The rules are a pure function of the log (spec section 3).
const (
	OpOpen      = "open"      // a thought was opened
	OpAnswer    = "answer"    // the player answered an ask thought (incl. "notsure")
	OpFade      = "fade"      // a thought faded unanswered
	OpConfirm   = "confirm"   // yes or no on a confirm thought
	OpRevoke    = "revoke"    // player revoked one rule
	OpRevokeAll = "revokeall" // player revoked all rules
	OpSacrifice = "sacrifice" // a sacrifice banks one skip for a point
	OpRebirth   = "rebirth"   // hardcore rebirth: new hero, rules/log/quality stay
)

// Entry is one line of the append-only answer log. T is virtual time.
type Entry struct {
	Seq     int           `json:"seq"`
	T       time.Duration `json:"t"`
	Hero    string        `json:"hero,omitempty"`
	Op      string        `json:"op"`
	Thought int           `json:"thought,omitempty"`
	Point   string        `json:"point,omitempty"`
	Key     string        `json:"key,omitempty"`
	Answer  string        `json:"answer,omitempty"`
	Yes     bool          `json:"yes,omitempty"`
	FadesAt time.Duration `json:"fadesAt,omitempty"`
}

// Clock is injectable so fades run on virtual time.
type Clock interface{ Now() time.Duration }

// ManualClock is the virtual clock of tests and soaks.
type ManualClock struct{ T time.Duration }

func (m *ManualClock) Now() time.Duration      { return m.T }
func (m *ManualClock) Advance(d time.Duration) { m.T += d }

// WallClock reports real time since the Unix epoch.
type WallClock struct{}

func (WallClock) Now() time.Duration { return time.Duration(time.Now().UnixNano()) }

var (
	ErrFaded      = errors.New("decide: thought faded")
	ErrNoThought  = errors.New("decide: no such open thought")
	ErrBadAnswer  = errors.New("decide: not an answer of this point")
	ErrNoRule     = errors.New("decide: no such rule")
	ErrWrongKind  = errors.New("decide: thought is of the wrong kind")
	ErrBadProfile = errors.New("decide: bad profile name")
)

type Rule struct {
	Answer    string        `json:"answer"`
	LearnedAt time.Duration `json:"learnedAt"`
	Revoked   bool          `json:"revoked,omitempty"`
	RevokedAt time.Duration `json:"revokedAt,omitempty"`
	// RevokedBy is always "player": no code path revokes a rule by itself.
	RevokedBy string `json:"revokedBy,omitempty"`
}

type Record struct {
	Key        string        `json:"key"`
	Last       string        `json:"last,omitempty"`
	Streak     int           `json:"streak"`
	AnsweredAt time.Duration `json:"answeredAt,omitempty"`
	Rule       *Rule         `json:"rule,omitempty"`
	Pending    string        `json:"pending,omitempty"` // answer waiting for a risky confirm
}

type Thought struct {
	ID       int
	Point    string
	Key      string
	Confirm  bool   // a confirm thought ("from now on I will always X here, OK?")
	Answer   string // for confirm: the answer to learn
	OpenedAt time.Duration
	FadesAt  time.Duration
	Sit      Situation // display only; in memory (not in the answer log), so a reopened thought has none
}

// EventKind names what a subscriber can see.
type EventKind string

const (
	EvThoughtOpened EventKind = "thought-opened"
	EvThoughtFaded  EventKind = "thought-faded"
	EvConfirmOpened EventKind = "confirm-opened"
	EvRuleLearned   EventKind = "rule-learned"
	EvRevoked       EventKind = "revoked"
	// EvLateAnswer: the player answered after the thought faded. The answer is treated as a fade (nothing counts,
	// the streak is not reset); the screen can say "too late, I carried on". Not a log line: the fade is.
	EvLateAnswer EventKind = "late-answer"
)

// Event is what a subscriber sees. Info is set on EvThoughtOpened and EvConfirmOpened so a subscriber need not poll
// OpenThoughts.
type Event struct {
	Kind    EventKind
	Key     string
	Thought int
	Answer  string
	Info    *ThoughtInfo
}

// ThoughtInfo is the public read of one open thought: everything a screen needs to put the question.
type ThoughtInfo struct {
	ID       int
	Point    string
	Key      string
	Question string
	Confirm  bool   // a confirm thought: Answer is the answer that would be learned; Choices is empty (yes or no)
	Answer   string // confirm only
	// Choices are the point's answers; "notsure" is always allowed on an ask thought and is not listed.
	Choices []Answer
	Expires time.Duration // virtual time at which the thought fades
	Opened  time.Duration // virtual time at which it was opened (additive in catalogue v4; a ladder ThoughtView carries it)
	Sit     Situation     // the facts that prompted the question, display only; nil when the call site gave none
}

func (s *Store) info(t *Thought) *ThoughtInfo {
	p, _ := s.cat.Point(t.Point)
	ti := &ThoughtInfo{ID: t.ID, Point: t.Point, Key: t.Key, Confirm: t.Confirm, Answer: t.Answer, Expires: t.FadesAt, Opened: t.OpenedAt, Sit: t.Sit}
	if p != nil {
		ti.Question = p.Question
		if !t.Confirm {
			ti.Choices = append([]Answer(nil), p.Answers...)
		}
	}
	return ti
}

// Thought reads one open thought; false when it is closed, faded or unknown.
func (s *Store) Thought(id int) (ThoughtInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.thoughts[id]
	if t == nil {
		return ThoughtInfo{}, false
	}
	return *s.info(t), true
}

// Decision is what a call site gets: the action now. The hero never waits for the thought.
type Decision struct {
	Answer  string
	Source  string // "rule", "default" or "default-locked" (tier not unlocked)
	Thought *Thought
}

// Store holds the answers of one profile. Safe for concurrent use.
type Store struct {
	mu      sync.Mutex
	cat     *Catalogue
	clk     Clock
	profile string
	path    string // "" = memory only
	hero    string
	unlock  int // highest tier that may open thoughts

	recs      map[string]*Record
	thoughts  map[int]*Thought
	openByKey map[string]int
	nextID    int
	skips     map[string]int
	quality   map[string]int
	log       []Entry
	subs      []func(Event)

	answers, faded int
}

func validProfile(name string) bool {
	if name == "" || len(name) > 64 {
		return false
	}
	for _, r := range name {
		if !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

// New makes a memory-only store. clk may be nil only for Replay.
func New(cat *Catalogue, profile string, clk Clock) (*Store, error) {
	if cat == nil {
		return nil, errors.New("decide: nil catalogue")
	}
	if !validProfile(profile) {
		return nil, fmt.Errorf("%w %q", ErrBadProfile, profile)
	}
	return &Store{
		cat: cat, clk: clk, profile: profile, hero: "hero1", unlock: 2,
		recs: map[string]*Record{}, thoughts: map[int]*Thought{}, openByKey: map[string]int{},
		nextID: 1, skips: map[string]int{}, quality: map[string]int{},
	}, nil
}

func (s *Store) Subscribe(f func(Event)) { s.mu.Lock(); s.subs = append(s.subs, f); s.mu.Unlock() }

// SetUnlockedTier sets the highest tier that opens thoughts; higher points run their default only.
func (s *Store) SetUnlockedTier(t int) { s.mu.Lock(); s.unlock = t; s.mu.Unlock() }

func (s *Store) emit(e Event) {
	for _, f := range s.subs {
		f(e)
	}
}

// commit validates and applies one entry, then appends it and saves. A rejected entry changes nothing.
func (s *Store) commit(e Entry) ([]Event, error) {
	e.Seq = len(s.log) + 1
	e.T = s.clk.Now()
	e.Hero = s.hero
	evs, err := s.apply(e)
	if err != nil {
		return nil, err
	}
	s.log = append(s.log, e)
	if err := s.save(); err != nil {
		return evs, err
	}
	return evs, nil
}

// apply changes state from one log entry. It validates before mutating. Replay uses it too.
func (s *Store) apply(e Entry) ([]Event, error) {
	switch e.Op {
	case OpOpen:
		return s.applyOpen(e)
	case OpAnswer:
		return s.applyAnswer(e)
	case OpFade:
		return s.applyFade(e)
	case OpConfirm:
		return s.applyConfirm(e)
	case OpRevoke:
		return s.applyRevoke(e.Key, e.T)
	case OpRevokeAll:
		var evs []Event
		for _, k := range s.sortedKeys() {
			r, _ := s.applyRevoke(k, e.T)
			evs = append(evs, r...)
		}
		return evs, nil
	case OpSacrifice:
		if _, err := s.cat.Point(e.Point); err != nil {
			return nil, err
		}
		if s.skips[e.Point] < min(s.cat.Config.MaxSkips, 1) { // at most one banked skip per point
			s.skips[e.Point]++
		}
		return nil, nil
	case OpRebirth:
		// Open thoughts belong to the dead hero; rules, streaks, log and quality stay (spec 3).
		var evs []Event
		for _, id := range s.openIDs() {
			evs = append(evs, s.fade(id)...)
		}
		s.hero = e.Hero
		return evs, nil
	}
	return nil, fmt.Errorf("decide: unknown log op %q", e.Op)
}

func (s *Store) sortedKeys() []string {
	ks := make([]string, 0, len(s.recs))
	for k := range s.recs {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func (s *Store) openIDs() []int {
	ids := make([]int, 0, len(s.thoughts))
	for id := range s.thoughts {
		ids = append(ids, id)
	}
	sort.Ints(ids)
	return ids
}

func (s *Store) applyOpen(e Entry) ([]Event, error) {
	ctx, err := ParseKey(e.Key)
	if err != nil {
		return nil, err
	}
	if _, err := s.cat.Point(ctx.Point); err != nil {
		return nil, err
	}
	if e.Thought != s.nextID {
		return nil, fmt.Errorf("decide: log out of order: thought %d, expected %d", e.Thought, s.nextID)
	}
	s.nextID++
	s.thoughts[e.Thought] = &Thought{ID: e.Thought, Point: ctx.Point, Key: e.Key, OpenedAt: e.T, FadesAt: e.FadesAt}
	s.openByKey[e.Key] = e.Thought
	return []Event{{Kind: EvThoughtOpened, Key: e.Key, Thought: e.Thought, Info: s.info(s.thoughts[e.Thought])}}, nil
}

func (s *Store) rec(key string) *Record {
	r := s.recs[key]
	if r == nil {
		r = &Record{Key: key}
		s.recs[key] = r
	}
	return r
}

func (s *Store) close(t *Thought) {
	delete(s.thoughts, t.ID)
	if s.openByKey[t.Key] == t.ID {
		delete(s.openByKey, t.Key)
	}
}

func (s *Store) applyAnswer(e Entry) ([]Event, error) {
	t := s.thoughts[e.Thought]
	if t == nil {
		return nil, fmt.Errorf("%w: %d", ErrNoThought, e.Thought)
	}
	if t.Confirm {
		return nil, ErrWrongKind
	}
	p, _ := s.cat.Point(t.Point)
	if e.Answer != NotSure {
		if _, ok := p.answer(e.Answer); !ok {
			return nil, fmt.Errorf("%w: %q for %s", ErrBadAnswer, e.Answer, p.ID)
		}
	}
	ctx, err := ParseKey(t.Key)
	if err != nil {
		return nil, err
	}
	n, err := s.cat.N(ctx)
	if err != nil {
		return nil, err
	}
	cl, _ := s.cat.RiskClass(ctx)
	s.close(t)
	s.answers++
	if e.Answer == NotSure {
		return nil, nil // uses the default, does not count, does not reset
	}
	r := s.rec(t.Key)
	if r.Pending != "" || (r.Rule != nil && !r.Rule.Revoked) {
		return nil, nil
	}
	r.AnsweredAt = e.T
	// A banked skip is consumed by the next non-risky answer at the point, and never used on a risky one.
	skip := 0
	if cl != Risky {
		skip, s.skips[t.Point] = s.skips[t.Point], 0
	}
	if r.Last == e.Answer {
		r.Streak += 1 + skip
	} else {
		r.Last, r.Streak = e.Answer, 1
	}
	if r.Streak < n {
		return nil, nil
	}
	if cl != Risky {
		r.Rule = &Rule{Answer: e.Answer, LearnedAt: e.T}
		return []Event{{Kind: EvRuleLearned, Key: t.Key, Answer: e.Answer}}, nil
	}
	r.Pending = e.Answer
	id := s.nextID
	s.nextID++
	ct := &Thought{ID: id, Point: t.Point, Key: t.Key, Confirm: true, Answer: e.Answer, OpenedAt: e.T,
		FadesAt: e.T + time.Duration(s.cat.Config.FadeSec)*time.Second}
	s.thoughts[id] = ct
	s.openByKey[t.Key] = id
	return []Event{{Kind: EvConfirmOpened, Key: t.Key, Thought: id, Answer: e.Answer, Info: s.info(ct)}}, nil
}

// fade closes a thought without count or reset; a faded confirm drops the pending answer (rule stays unlearned).
func (s *Store) fade(id int) []Event {
	t := s.thoughts[id]
	if t.Confirm {
		if r := s.recs[t.Key]; r != nil {
			r.Pending = ""
		}
	}
	s.close(t)
	s.faded++
	return []Event{{Kind: EvThoughtFaded, Key: t.Key, Thought: id}}
}

func (s *Store) applyFade(e Entry) ([]Event, error) {
	if s.thoughts[e.Thought] == nil {
		return nil, fmt.Errorf("%w: %d", ErrNoThought, e.Thought)
	}
	return s.fade(e.Thought), nil
}

func (s *Store) applyConfirm(e Entry) ([]Event, error) {
	t := s.thoughts[e.Thought]
	if t == nil {
		return nil, fmt.Errorf("%w: %d", ErrNoThought, e.Thought)
	}
	if !t.Confirm {
		return nil, ErrWrongKind
	}
	r := s.rec(t.Key)
	s.close(t)
	s.answers++
	r.Pending = ""
	if !e.Yes {
		r.Streak, r.Last = 0, ""
		return nil, nil
	}
	r.Rule = &Rule{Answer: t.Answer, LearnedAt: e.T}
	return []Event{{Kind: EvRuleLearned, Key: t.Key, Answer: t.Answer}}, nil
}

func (s *Store) applyRevoke(key string, t time.Duration) ([]Event, error) {
	r := s.recs[key]
	if r == nil || r.Rule == nil || r.Rule.Revoked {
		return nil, fmt.Errorf("%w: %q", ErrNoRule, key)
	}
	r.Rule.Revoked, r.Rule.RevokedAt, r.Rule.RevokedBy = true, t, "player"
	r.Streak, r.Last, r.Pending = 0, "", ""
	return []Event{{Kind: EvRevoked, Key: key, Answer: r.Rule.Answer}}, nil
}

func (s *Store) publish(evs []Event) {
	for _, e := range evs {
		s.emit(e)
	}
}

// Ask is called by the bot at a decision point. A live rule answers at once. Otherwise the default is returned at
// once and a thought opens (unless one is already open for the key, a confirm is pending, or the tier is locked).
func (s *Store) Ask(ctx Context) (Decision, error) {
	return s.ask(ctx, "")
}

// AskWithDefault is Ask for a call site that owns today's behaviour: legacy is the answer the old code would give
// (an answer id of the point). A live rule still wins; with none the caller's legacy answer is returned (Source
// "legacy", or "default-locked" for a locked tier) and a thought opens as in Ask. The catalogue default is not used.
func (s *Store) AskWithDefault(ctx Context, legacy string) (Decision, error) {
	return s.ask(ctx, legacy)
}

func (s *Store) ask(ctx Context, legacy string) (Decision, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, err := s.cat.Point(ctx.Point)
	if err != nil {
		return Decision{}, err
	}
	src := "default"
	var def string
	if legacy != "" {
		if _, ok := p.answer(legacy); !ok {
			return Decision{}, fmt.Errorf("%w: legacy %q for %s", ErrBadAnswer, legacy, p.ID)
		}
		def, src = legacy, "legacy"
	} else if def, err = s.cat.Default(ctx); err != nil {
		return Decision{}, err
	}
	if r := s.recs[ctx.Key]; r != nil && r.Rule != nil && !r.Rule.Revoked {
		return Decision{Answer: r.Rule.Answer, Source: "rule"}, nil
	}
	if p.Tier > s.unlock {
		return Decision{Answer: def, Source: "default-locked"}, nil
	}
	d := Decision{Answer: def, Source: src}
	if id, ok := s.openByKey[ctx.Key]; ok {
		if t := s.thoughts[id]; !t.Confirm {
			c := *t
			d.Thought = &c
		}
		return d, nil
	}
	e := Entry{Op: OpOpen, Thought: s.nextID, Point: ctx.Point, Key: ctx.Key,
		FadesAt: s.clk.Now() + time.Duration(s.cat.Config.FadeSec)*time.Second}
	evs, err := s.commit(e)
	if err != nil {
		return d, err
	}
	s.thoughts[e.Thought].Sit = ctx.Sit
	for i := range evs {
		if evs[i].Info != nil && evs[i].Thought == e.Thought {
			evs[i].Info.Sit = ctx.Sit
		}
	}
	c := *s.thoughts[e.Thought]
	d.Thought = &c
	s.publish(evs)
	return d, nil
}

func (s *Store) resolve(id int, op string, e Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := s.thoughts[id]
	if t == nil {
		return fmt.Errorf("%w: %d", ErrNoThought, id)
	}
	e.Op, e.Thought = op, id
	if s.clk.Now() >= t.FadesAt { // too late: the thought faded, nothing counts
		evs, err := s.commit(Entry{Op: OpFade, Thought: id})
		s.publish(evs)
		if err != nil {
			return err
		}
		s.emit(Event{Kind: EvLateAnswer, Key: t.Key, Thought: id, Answer: e.Answer})
		return ErrFaded
	}
	if op == OpAnswer && t.Confirm || op == OpConfirm && !t.Confirm {
		return ErrWrongKind
	}
	evs, err := s.commit(e)
	s.publish(evs)
	return err
}

// Answer records the player's answer ("notsure" allowed) to an ask thought.
func (s *Store) Answer(thought int, answer string) error {
	return s.resolve(thought, OpAnswer, Entry{Answer: answer})
}

// Confirm answers a confirm thought of a risky point.
func (s *Store) Confirm(thought int, yes bool) error {
	return s.resolve(thought, OpConfirm, Entry{Yes: yes})
}

// Tick fades every open thought whose time is up and returns how many faded.
func (s *Store) Tick() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, id := range s.openIDs() {
		if t := s.thoughts[id]; s.clk.Now() >= t.FadesAt {
			evs, err := s.commit(Entry{Op: OpFade, Thought: id})
			s.publish(evs)
			if err != nil {
				return n, err
			}
			n++
		}
	}
	return n, nil
}

// Revoke is the player's revoke of one rule. Nothing else in the package revokes.
func (s *Store) Revoke(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	evs, err := s.commit(Entry{Op: OpRevoke, Key: key})
	s.publish(evs)
	return err
}

// RevokeAll revokes every live rule.
func (s *Store) RevokeAll() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	evs, err := s.commit(Entry{Op: OpRevokeAll})
	s.publish(evs)
	return err
}

// Sacrifice is the skip hook: the next repeated answer for the point counts one extra (bounded by maxSkips).
func (s *Store) Sacrifice(point string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.commit(Entry{Op: OpSacrifice, Point: point})
	return err
}

// Rebirth starts a new hero on the same profile: rules, streaks, log and quality stay.
func (s *Store) Rebirth(hero string) error {
	if hero == "" {
		return errors.New("decide: rebirth needs a hero name")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.hero = hero
	evs, err := s.commit(Entry{Op: OpRebirth})
	s.publish(evs)
	return err
}

// SetQuality stores an opaque altar-quality level beside the rules (kept through rebirth).
func (s *Store) SetQuality(name string, level int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.quality[name] = level
	return s.save()
}

func (s *Store) Quality(name string) int { s.mu.Lock(); defer s.mu.Unlock(); return s.quality[name] }

// RuleView is one line of the rules list in plain language.
type RuleView struct {
	Key      string `json:"key"`
	Point    string `json:"point"`
	Question string `json:"question"`
	Context  string `json:"context"`
	Answer   string `json:"answer"`
	Label    string `json:"label"`
	Learned  time.Duration
	Revoked  bool `json:"revoked,omitempty"`
}

// Rules lists every learned rule (revoked ones too, marked), sorted by key.
func (s *Store) Rules() []RuleView {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rules()
}

func (s *Store) rules() []RuleView {
	var out []RuleView
	for _, k := range s.sortedKeys() {
		r := s.recs[k]
		if r.Rule == nil {
			continue
		}
		ctx, _ := ParseKey(k)
		p, _ := s.cat.Point(ctx.Point)
		label := r.Rule.Answer
		if a, ok := p.answer(label); ok {
			label = a.Label
		}
		out = append(out, RuleView{Key: k, Point: p.ID, Question: p.Question, Context: ctx.Plain(),
			Answer: r.Rule.Answer, Label: label, Learned: r.Rule.LearnedAt, Revoked: r.Rule.Revoked})
	}
	return out
}

// RulesJSON is the canonical byte form of the live rule state (records sorted by key). Replay compares it.
func (s *Store) RulesJSON() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.rulesJSON()
}

func (s *Store) rulesJSON() ([]byte, error) {
	recs := []*Record{}
	for _, k := range s.sortedKeys() {
		recs = append(recs, s.recs[k])
	}
	return json.MarshalIndent(recs, "", " ")
}

// Record returns a copy of the record for a key, if any.
func (s *Store) Record(key string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	r := s.recs[key]
	if r == nil {
		return Record{}, false
	}
	c := *r
	if r.Rule != nil {
		rc := *r.Rule
		c.Rule = &rc
	}
	return c, true
}

// OpenThoughts lists the thoughts still open, oldest first.
func (s *Store) OpenThoughts() []Thought {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []Thought
	for _, id := range s.openIDs() {
		out = append(out, *s.thoughts[id])
	}
	return out
}

// Log returns a copy of the answer log.
func (s *Store) Log() []Entry {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]Entry(nil), s.log...)
}

// Stats for gate reports: "rules learned: K, answers given: M, thoughts faded: F".
func (s *Store) Stats() (rules, answers, faded int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.recs {
		if r.Rule != nil && !r.Rule.Revoked {
			rules++
		}
	}
	return rules, s.answers, s.faded
}

// Replay applies a log to an empty store of the profile and returns the canonical rules bytes. The same log
// always gives the same bytes; a log naming an unknown point or thought is an error.
func Replay(cat *Catalogue, profile string, script []Entry) ([]byte, error) {
	s, err := New(cat, profile, nil)
	if err != nil {
		return nil, err
	}
	if err := s.replay(script); err != nil {
		return nil, err
	}
	return s.rulesJSON()
}

func (s *Store) replay(script []Entry) error {
	for i, e := range script {
		if e.Hero != "" {
			s.hero = e.Hero
		}
		if _, err := s.apply(e); err != nil {
			return fmt.Errorf("decide: replay entry %d: %w", i+1, err)
		}
		s.log = append(s.log, e)
	}
	return nil
}
