package watch

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/decide"
)

// Schema names. Every JSON body carries "schema": "<name>/<version>". A change that removes or renames a field bumps
// the version; adding a field does not.
const (
	thoughtsSchema = "watch.thoughts/1"
	journalSchema  = "watch.journal/1"
	rulesSchema    = "watch.rules/1"
	ordersSchema   = "watch.orders/1"
	whySchema      = "watch.why/1"
	configSchema   = "watch.config/1"
)

// ChoiceView is one fixed answer button.
type ChoiceView struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Does  string `json:"does,omitempty"` // what the hero does on this answer, in plain words
}

// RuleRef names a rule in force in an incident or journal line, with the revoke target.
type RuleRef struct {
	Key     string `json:"key"`
	Context string `json:"context"`
	Label   string `json:"label"`
	Live    bool   `json:"live"`
}

// ThoughtItem is one open question or confirm in the feed.
type ThoughtItem struct {
	ID           int          `json:"id"`
	Kind         string       `json:"kind"` // "question" or "confirm"
	Point        string       `json:"point"`
	Question     string       `json:"question"`
	Headline     string       `json:"headline"`            // what happened, with the real values
	Specific     bool         `json:"specific"`            // headline carries the event's values
	IfNothing    string       `json:"ifNothing,omitempty"` // what the hero does on no answer, and the fade time
	WhyNow       string       `json:"whyNow,omitempty"`
	Rule         string       `json:"rule,omitempty"` // footnote: the learning key in words
	Key          string       `json:"key"`
	Context      string       `json:"context"`
	Fields       []FieldView  `json:"fields"`
	Urgency      string       `json:"urgency,omitempty"`
	Choices      []ChoiceView `json:"choices"` // fixed buttons; "notsure" is last on a question, yes/no on a confirm
	Meanwhile    string       `json:"meanwhile,omitempty"`
	Learn        string       `json:"learn,omitempty"` // confirm only: what the rule would be
	OpenedAt     float64      `json:"openedAt"`
	ExpiresAt    float64      `json:"expiresAt"`
	RemainingSec float64      `json:"remainingSec"`
}

// FeedItem is a non-question line of the feed (idle dialogue, story, incident, order change).
type FeedItem struct {
	ID    string    `json:"id"`
	T     float64   `json:"t"`
	Kind  string    `json:"kind"`
	Text  string    `json:"text"`
	Rules []RuleRef `json:"rules,omitempty"`
}

// ThoughtsView is GET /api/thoughts.
type ThoughtsView struct {
	Schema   string        `json:"schema"`
	Seq      uint64        `json:"seq"`
	Now      float64       `json:"now"`
	FadeSec  int           `json:"fadeSec"`
	Speed    float64       `json:"speed"`
	Thoughts []ThoughtItem `json:"thoughts"`
	Feed     []FeedItem    `json:"feed"`
}

func (h *Hub) Thoughts() ThoughtsView {
	now := h.cfg.Clock.Now()
	open := h.cfg.Store.OpenThoughts() // store first, Hub lock second
	infos := map[int]decide.ThoughtInfo{}
	for _, t := range open {
		if info, ok := h.cfg.Store.Thought(t.ID); ok {
			infos[t.ID] = info
		}
	}
	sn := h.snap()
	v := ThoughtsView{Schema: thoughtsSchema, Seq: sn.seq, Now: h.rel(now), FadeSec: h.cfg.Cat.Config.FadeSec, Speed: h.cfg.Speed,
		Thoughts: []ThoughtItem{}, Feed: []FeedItem{}}
	for _, t := range open {
		info, ok := infos[t.ID]
		if !ok || info.Expires <= now { // faded but not yet recorded: gone from the feed, a GET never writes
			continue
		}
		var rec *thoughtRec
		if r, ok := sn.recs[t.ID]; ok {
			rec = &r
		}
		if rec != nil && !rec.shown {
			continue
		}
		it := ThoughtItem{ID: info.ID, Kind: "question", Point: info.Point, Question: info.Question, Key: info.Key,
			Context: PlainContext(info.Key), Fields: Fields(info.Key), Urgency: UrgencyText(info.Key), Choices: []ChoiceView{},
			OpenedAt: h.rel(t.OpenedAt), ExpiresAt: h.rel(info.Expires), RemainingSec: secs(info.Expires - now)}
		def := ""
		if rec != nil && rec.meanwhile != "" {
			it.Meanwhile = "Meanwhile I: " + h.pointAnswerLabel(info.Point, rec.meanwhile)
			def = rec.meanwhile
		}
		card := h.BuildCard(info, def)
		it.Headline, it.Specific, it.IfNothing, it.WhyNow, it.Rule = card.Headline, card.Specific, card.IfNothing, card.WhyNow, card.Rule
		if info.Confirm {
			it.Kind = "confirm"
			it.Headline, it.Specific, it.WhyNow = "", false, ""
			it.IfNothing = "If you do nothing: no rule is made and the hero keeps asking."
			it.Learn = fmt.Sprintf("From now on I will always \"%s\" here (%s). OK?", h.pointAnswerLabel(info.Point, info.Answer), it.Context)
			it.Choices = []ChoiceView{{"yes", "Yes, always", ""}, {"no", "No", ""}}
		} else {
			for _, c := range info.Choices {
				it.Choices = append(it.Choices, ChoiceView{c.ID, c.Label, card.Does[c.ID]})
			}
			it.Choices = append(it.Choices, ChoiceView{decide.NotSure, "Not sure, you decide", "let the hero decide this time; nothing is learned"})
		}
		v.Thoughts = append(v.Thoughts, it)
	}
	var feed []FeedItem
	for _, x := range sn.extras {
		switch x.kind {
		case "idle", "story", "incident", "order", "learned", "late":
			feed = append(feed, FeedItem{ID: x.id, T: h.rel(x.t), Kind: x.kind, Text: x.text, Rules: h.refs(x.rules)})
		}
	}
	if len(feed) > h.cfg.FeedKeep {
		feed = feed[len(feed)-h.cfg.FeedKeep:]
	}
	if feed != nil {
		v.Feed = feed
	}
	return v
}

// refs builds rule refs from keys. It reads the store, so the caller must not hold the Hub lock.
func (h *Hub) refs(keys []string) []RuleRef {
	var out []RuleRef
	for _, k := range keys {
		r, ok := h.cfg.Store.Record(k)
		ref := RuleRef{Key: k, Context: PlainContext(k)}
		if ok && r.Rule != nil {
			ref.Label, ref.Live = h.answerLabel(k, r.Rule.Answer), !r.Rule.Revoked
		}
		out = append(out, ref)
	}
	return out
}

// JournalLine is one line of the permanent history.
type JournalLine struct {
	ID      string    `json:"id"`
	T       float64   `json:"t"`
	Kind    string    `json:"kind"`
	Text    string    `json:"text"`
	Thought int       `json:"thought,omitempty"`
	Key     string    `json:"key,omitempty"`
	Why     bool      `json:"why,omitempty"` // the "why did you do that?" link is offered
	Rules   []RuleRef `json:"rules,omitempty"`
}

// JournalView is GET /api/journal.
type JournalView struct {
	Schema string        `json:"schema"`
	Seq    uint64        `json:"seq"`
	Now    float64       `json:"now"`
	Total  int           `json:"total"`
	Lines  []JournalLine `json:"lines"` // oldest first, the last `limit`
}

var kindRank = map[string]int{"question": 0, "answer": 1, "notsure": 1, "confirm-yes": 1, "confirm-no": 1, "missed": 1,
	"revoked": 1, "revoked-all": 1, "rebirth": 1, "learned": 2, "late": 2}

// Journal builds the journal from the store's answer log (the part that persists) and the Hub's own lines.
func (h *Hub) Journal(limit int, kind string) JournalView {
	log := h.cfg.Store.Log()
	sn := h.snap()
	keyOf := map[int]string{}
	for _, e := range log {
		if e.Op == decide.OpOpen {
			keyOf[e.Thought] = e.Key
		}
	}
	for id, r := range sn.recs {
		if _, ok := keyOf[id]; !ok {
			keyOf[id] = r.info.Key
		}
	}
	question := func(id int, key string) string {
		if r, ok := sn.recs[id]; ok {
			return r.info.Question
		}
		return h.questionFor(key)
	}
	type row struct {
		l    JournalLine
		t    time.Duration
		rank int
		ord  int
	}
	var rows []row
	add := func(t time.Duration, l JournalLine) {
		l.T = h.rel(t)
		rows = append(rows, row{l, t, kindRank[l.Kind], len(rows)})
	}
	for _, e := range log {
		id := fmt.Sprintf("L%d", e.Seq)
		key := e.Key
		if key == "" {
			key = keyOf[e.Thought]
		}
		q, ctx := question(e.Thought, key), PlainContext(key)
		when := "" // " When: ..." the event behind the question, if the call site gave its facts
		if r, ok := sn.recs[e.Thought]; ok {
			if w := situationLine(r.info); w != "" {
				when = " When: " + w
			}
		}
		switch e.Op {
		case decide.OpOpen:
			txt := fmt.Sprintf("Question: %s (%s).%s", q, ctx, when)
			if r, ok := sn.recs[e.Thought]; ok && r.meanwhile != "" {
				txt += " Meanwhile I: " + h.pointAnswerLabel(r.info.Point, r.meanwhile) + "."
			}
			add(e.T, JournalLine{ID: id, Kind: "question", Text: txt, Thought: e.Thought, Key: key, Why: true})
		case decide.OpAnswer:
			if e.Answer == decide.NotSure {
				add(e.T, JournalLine{ID: id, Kind: "notsure", Thought: e.Thought, Key: key, Why: true,
					Text: fmt.Sprintf("You said not sure: %s I used my default.%s", q, when)})
			} else {
				add(e.T, JournalLine{ID: id, Kind: "answer", Thought: e.Thought, Key: key, Why: true,
					Text: fmt.Sprintf("You answered \"%s\": %s%s", h.answerLabel(key, e.Answer), q, when)})
			}
		case decide.OpConfirm:
			t := "You confirmed the rule."
			k := "confirm-yes"
			if !e.Yes {
				t, k = "You said no to the rule; the streak starts again.", "confirm-no"
			}
			if key != "" {
				t += " (" + ctx + ")"
			}
			add(e.T, JournalLine{ID: id, Kind: k, Thought: e.Thought, Key: key, Why: key != "", Text: t})
		case decide.OpFade:
			txt := fmt.Sprintf("Missed: %s (%s). It faded and I carried on with my default; nothing was counted.%s", q, ctx, when)
			add(e.T, JournalLine{ID: id, Kind: "missed", Thought: e.Thought, Key: key, Why: key != "", Text: txt})
		case decide.OpRevoke:
			add(e.T, JournalLine{ID: id, Kind: "revoked", Key: e.Key, Why: true,
				Text: fmt.Sprintf("You revoked a rule (%s). I will ask again.", PlainContext(e.Key))})
		case decide.OpRevokeAll:
			add(e.T, JournalLine{ID: id, Kind: "revoked-all", Text: "You revoked every rule. I will ask again."})
		case decide.OpRebirth:
			add(e.T, JournalLine{ID: id, Kind: "rebirth", Text: "A new hero (" + safeText(e.Hero, 40) + ") started; the rules stayed."})
		}
	}
	for _, x := range sn.extras {
		l := JournalLine{ID: x.id, Kind: x.kind, Text: x.text, Thought: x.thought, Key: x.key, Why: x.key != "", Rules: h.refs(x.rules)}
		add(x.t, l)
	}
	sort.SliceStable(rows, func(i, j int) bool {
		a, b := rows[i], rows[j]
		if a.t != b.t {
			return a.t < b.t
		}
		if a.rank != b.rank {
			return a.rank < b.rank
		}
		return a.ord < b.ord
	})
	v := JournalView{Schema: journalSchema, Seq: sn.seq, Now: h.rel(h.cfg.Clock.Now()), Lines: []JournalLine{}}
	for _, r := range rows {
		if kind == "" || r.l.Kind == kind {
			v.Lines = append(v.Lines, r.l)
		}
	}
	v.Total = len(v.Lines)
	if limit > 0 && len(v.Lines) > limit {
		v.Lines = v.Lines[len(v.Lines)-limit:]
	}
	return v
}

// RuleItem is one learned rule in plain language.
type RuleItem struct {
	Key       string      `json:"key"`
	Point     string      `json:"point"`
	Question  string      `json:"question"`
	Context   string      `json:"context"`
	Fields    []FieldView `json:"fields"`
	Answer    string      `json:"answer"`
	Label     string      `json:"label"`
	Risk      string      `json:"risk"`
	Text      string      `json:"text"`
	LearnedAt float64     `json:"learnedAt"`
	Revoked   bool        `json:"revoked"`
	RevokedAt float64     `json:"revokedAt,omitempty"`
}

// RulesView is GET /api/rules.
type RulesView struct {
	Schema string     `json:"schema"`
	Seq    uint64     `json:"seq"`
	Now    float64    `json:"now"`
	Live   int        `json:"live"`
	Rules  []RuleItem `json:"rules"`
}

func (h *Hub) Rules() RulesView {
	rv := h.cfg.Store.Rules()
	items := make([]RuleItem, 0, len(rv))
	live := 0
	for _, r := range rv {
		it := RuleItem{Key: r.Key, Point: r.Point, Question: r.Question, Context: PlainContext(r.Key), Fields: Fields(r.Key),
			Answer: r.Answer, Label: r.Label, LearnedAt: h.rel(r.Learned), Revoked: r.Revoked}
		if ctx, err := decide.ParseKey(r.Key); err == nil {
			it.Risk, _ = h.cfg.Cat.RiskClass(ctx)
		}
		it.Text = fmt.Sprintf("When asked \"%s\" (%s), I %s.", r.Question, it.Context, r.Label)
		if rec, ok := h.cfg.Store.Record(r.Key); ok && rec.Rule != nil && rec.Rule.Revoked {
			it.RevokedAt = h.rel(rec.Rule.RevokedAt)
		}
		if !r.Revoked {
			live++
		}
		items = append(items, it)
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	return RulesView{Schema: rulesSchema, Seq: h.seq, Now: h.rel(h.cfg.Clock.Now()), Live: live, Rules: items}
}

// WhyView is GET /api/why: why the hero did what it did at a context.
type WhyView struct {
	Schema   string      `json:"schema"`
	Key      string      `json:"key"`
	Question string      `json:"question"`
	Context  string      `json:"context"`
	Fields   []FieldView `json:"fields"`
	Source   string      `json:"source"` // "rule", "revoked", "default"
	Answer   string      `json:"answer"`
	Label    string      `json:"label"`
	Streak   int         `json:"streak"`
	Needed   int         `json:"needed"`
	Risk     string      `json:"risk"`
	Text     string      `json:"text"`
	Revoke   bool        `json:"revoke"` // a live rule exists: the page offers revoke
}

// Why explains the hero's behaviour at a context key from the store's record.
func (h *Hub) Why(key string) (WhyView, error) {
	ctx, err := decide.ParseKey(key)
	if err != nil {
		return WhyView{}, fmt.Errorf("%w: %v", ErrNoRule, err)
	}
	p, err := h.cfg.Cat.Point(ctx.Point)
	if err != nil {
		return WhyView{}, fmt.Errorf("%w: %v", ErrNoRule, err)
	}
	n, err := h.cfg.Cat.N(ctx)
	if err != nil {
		return WhyView{}, err
	}
	risk, _ := h.cfg.Cat.RiskClass(ctx)
	v := WhyView{Schema: whySchema, Key: key, Question: p.Question, Context: PlainContext(key), Fields: Fields(key), Needed: n, Risk: risk}
	rec, _ := h.cfg.Store.Record(key)
	v.Streak = rec.Streak
	label := func(a string) string { return h.pointAnswerLabel(p.ID, a) }
	switch {
	case rec.Rule != nil && !rec.Rule.Revoked:
		v.Source, v.Answer, v.Label, v.Revoke = "rule", rec.Rule.Answer, label(rec.Rule.Answer), true
		v.Text = fmt.Sprintf("I did \"%s\" because of a rule you taught me: you gave this answer %d times in a row when asked \"%s\" (%s). "+
			"A rule never revokes itself; revoke it if it is wrong.", v.Label, n, p.Question, v.Context)
	default:
		def, _ := h.cfg.Cat.Default(ctx)
		v.Answer, v.Label = def, label(def)
		if rec.Rule != nil {
			v.Source = "revoked"
			v.Text = fmt.Sprintf("You revoked the rule here, so I use my default \"%s\" and ask again (%s).", v.Label, v.Context)
		} else {
			v.Source = "default"
			v.Text = fmt.Sprintf("No rule yet, so I used my default \"%s\" (%s).", v.Label, v.Context)
			if rec.Streak > 0 {
				v.Text += fmt.Sprintf(" You have given \"%s\" %d of %d times in a row; at %d it becomes a rule.", label(rec.Last), rec.Streak, n, n)
			} else {
				v.Text += fmt.Sprintf(" The same answer %d times in a row teaches me a rule.", n)
			}
		}
	}
	return v, nil
}

// WhyLine resolves a journal line id to its key and explains it.
func (h *Hub) WhyLine(line string) (WhyView, error) {
	for _, l := range h.Journal(0, "").Lines {
		if l.ID == line && l.Key != "" {
			return h.Why(l.Key)
		}
	}
	return WhyView{}, fmt.Errorf("%w: no explanation for line %q", ErrNoRule, safeText(line, 20))
}

// OrdersView is GET /api/orders.
type OrdersView struct {
	Schema     string   `json:"schema"`
	Seq        uint64   `json:"seq"`
	Now        float64  `json:"now"`
	Orders     []Order  `json:"orders"`
	Areas      []string `json:"areas"`
	Kinds      []string `json:"kinds"`
	MaxMinutes int      `json:"maxMinutes"`
}

func (h *Hub) OrdersView() OrdersView {
	h.mu.Lock()
	defer h.mu.Unlock()
	v := OrdersView{Schema: ordersSchema, Seq: h.seq, Now: h.rel(h.cfg.Clock.Now()), Orders: []Order{},
		Areas: append([]string{}, h.cfg.Areas...), Kinds: []string{OrderGo, OrderHere, OrderHome}, MaxMinutes: MaxOrderMinutes}
	for _, o := range h.orders {
		v.Orders = append(v.Orders, *o)
	}
	return v
}

// Uptime is GET /api/uptime.
func (h *Hub) Uptime() UptimeView {
	now := h.cfg.Clock.Now()
	h.mu.Lock()
	defer h.mu.Unlock()
	v := h.up.view(now, h.cfg.WindowSec)
	v.Seq, v.Now = h.seq, h.rel(now)
	return v
}

// ConfigView is GET /api/config: what the page needs to find the answer route.
type ConfigView struct {
	Schema     string `json:"schema"`
	AnswerBase string `json:"answerBase,omitempty"`
	AnswerPort int    `json:"answerPort,omitempty"`
	FadeSec    int    `json:"fadeSec"`
	FrameURL   string `json:"frameUrl,omitempty"` // the viewer's frame stream, empty when there is no game
}

func (h *Hub) Config() ConfigView {
	return ConfigView{Schema: configSchema, AnswerBase: strings.TrimRight(h.cfg.AnswerBase, "/"), AnswerPort: h.cfg.AnswerPort,
		FadeSec: h.cfg.Cat.Config.FadeSec, FrameURL: h.cfg.FrameURL}
}
