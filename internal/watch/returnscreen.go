package watch

// The return screen (TASK-127, ADR 0018, docs/return-screen.md): one server-rendered page that tells what happened
// while the player was away. BuildReturn is pure (journal lines + away events in, ReturnData out), so a recorded journal
// is a golden test; the Hub glue and the handlers sit below it.

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
)

const returnSchema = "watch.return/1"

// AwayAfter is the absence after which /watch lands on the return screen (fun-ideas 3.6; V user decision D Q-U5).
const AwayAfter = 20 * time.Minute

// AwayEvent is one event of the bot's stream kept for the return screen (the journal has no kills, drops, levels or
// potions). T is in the journal's time base (Hub seconds). Kind is one of the Away* constants.
type AwayEvent struct {
	T       float64 `json:"t"`
	Kind    string  `json:"kind"`
	Monster string  `json:"monster,omitempty"`
	Boss    bool    `json:"boss,omitempty"`
	XP      int64   `json:"xp,omitempty"`
	Item    string  `json:"item,omitempty"`
	Quality string  `json:"quality,omitempty"` // normal, magic, rare, set, unique, crafted
	Type    string  `json:"type,omitempty"`    // "rune" ranks as a rune drop
	Level   int     `json:"level,omitempty"`
	Area    string  `json:"area,omitempty"`
	AreaID  int     `json:"areaId,omitempty"`
	Potion  string  `json:"potion,omitempty"`
	Quest   string  `json:"quest,omitempty"`
	Done    bool    `json:"done,omitempty"`
	LifePct int     `json:"lifePct,omitempty"` // AwayClose only; a source that knows the minimum life of a fight reports it here
	Gold    int64   `json:"gold,omitempty"`    // AwayGold only: gold gained (U: no emitter yet)
	// AwayClose only (card previewfeed): one finished fight. Potions drunk, seconds, damage taken, and how it ended
	// ("chicken", "portal" or empty). Wall is unix seconds, so a restart can place a spooled event on the new time base.
	Potions int    `json:"potions,omitempty"`
	Sec     int    `json:"sec,omitempty"`
	Damage  int    `json:"damage,omitempty"`
	Left    string `json:"left,omitempty"`
	Wall    int64  `json:"wall,omitempty"`
}

// Away event kinds: the goals manager's own (kill, drop, death, level_up, area_change, potion, quest) plus two that no
// emitter sends yet (gold); close comes from the fight feed (awayspool.go); the page uses them when present.
const (
	AwayKill   = "kill"
	AwayDrop   = "drop"
	AwayDeath  = "death"
	AwayLevel  = "level_up"
	AwayArea   = "area_change"
	AwayPotion = "potion"
	AwayQuest  = "quest"
	AwayClose  = "close"
	AwayGold   = "gold"
)

// AwayLog keeps the last events for the return screen. Safe for concurrent use. Feed it from the loop that already
// reads Manager.Subscribe (cmd/d2bot feedHub: log.OnGoalEvent(ev)).
type AwayLog struct {
	mu  sync.Mutex
	now func() float64
	max int
	evs []AwayEvent

	spool *os.File // previewfeed: close and drop events survive a restart (awayspool.go)
	wall  func() time.Time
}

// NewAwayLog makes a log; now gives the journal's time base (Hub.AwayLog does that for a Hub).
func NewAwayLog(now func() float64) *AwayLog { return &AwayLog{now: now, max: 20000} }

// AwayLog is a log on this Hub's time base.
func (h *Hub) AwayLog() *AwayLog {
	return NewAwayLog(func() float64 { return h.rel(h.cfg.Clock.Now()) })
}

// Add appends an event; T 0 is stamped with now.
func (l *AwayLog) Add(e AwayEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if e.T == 0 && l.now != nil {
		e.T = l.now()
	}
	l.evs = append(l.evs, e)
	l.spoolLocked(e)
	if len(l.evs) > l.max {
		l.evs = append([]AwayEvent(nil), l.evs[len(l.evs)-l.max:]...)
	}
}

// OnGoalEvent keeps the goals events the page uses and drops the rest.
func (l *AwayLog) OnGoalEvent(ev goals.Event) {
	switch ev.Type {
	case goals.EvKill, goals.EvDrop, goals.EvDeath, goals.EvLevelUp, goals.EvAreaChange, goals.EvPotion, goals.EvQuest:
	default:
		return
	}
	e := AwayEvent{Kind: string(ev.Type), Monster: safeText(ev.Monster, 60), Boss: ev.Boss, XP: ev.XP, Level: ev.Level,
		Area: safeText(ev.Area, 60), AreaID: ev.AreaID, Potion: safeText(ev.Potion, 40), Quest: safeText(ev.Quest, 40), Done: ev.Done}
	if ev.Item != nil {
		e.Item, e.Quality, e.Type = safeText(ev.Item.Name, 80), ev.Item.Quality, ev.Item.Type
	}
	l.Add(e)
}

// Events returns a copy, oldest first.
func (l *AwayLog) Events() []AwayEvent {
	if l == nil {
		return nil
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]AwayEvent(nil), l.evs...)
}

// ReadAwayEvents reads JSON lines (one AwayEvent per line): the fixture and soak-log format.
func ReadAwayEvents(r io.Reader) ([]AwayEvent, error) {
	var out []AwayEvent
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e AwayEvent
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return nil, fmt.Errorf("watch: away events line %d: %w", n, err)
		}
		out = append(out, e)
	}
	return out, sc.Err()
}

// Visit remembers when the player last looked at the live screen, to tell an absence. In memory, optionally kept in a
// small JSON file so a bot restart does not forget it.
type Visit struct {
	mu     sync.Mutex
	last   time.Time
	landed time.Time // when /watch last redirected to the return screen; a redirect is due once per absence
	path   string
	now    func() time.Time
}

// NewVisit makes a tracker; path "" keeps it in memory only. A missing or unreadable file means "never seen".
func NewVisit(path string, now func() time.Time) *Visit {
	if now == nil {
		now = time.Now
	}
	v := &Visit{path: path, now: now}
	if b, err := os.ReadFile(path); path != "" && err == nil {
		var f struct {
			Last   time.Time `json:"lastSeen"`
			Landed time.Time `json:"landed"`
		}
		if json.Unmarshal(b, &f) == nil {
			v.last, v.landed = f.Last, f.Landed
		}
	}
	return v
}

// Touch records a look now.
func (v *Visit) Touch() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.last = v.now()
	v.save()
}

func (v *Visit) save() {
	if v.path != "" {
		b, _ := json.Marshal(map[string]time.Time{"lastSeen": v.last, "landed": v.landed})
		os.WriteFile(v.path, b, 0o600)
	}
}

// due says whether /watch should land on the return screen now: the last look is older than AwayAfter and no landing
// happened since that look. Due marks the landing, so the same absence redirects once; a new look and a new absence
// make it due again.
func (v *Visit) due() bool {
	if v == nil {
		return false
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.last.IsZero() || v.now().Sub(v.last) <= AwayAfter || v.landed.After(v.last) {
		return false
	}
	v.landed = v.now()
	v.save()
	return true
}

// Away is how long ago the last look was; ok is false when there was none.
func (v *Visit) Away() (d time.Duration, ok bool) {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.last.IsZero() {
		return 0, false
	}
	return v.now().Sub(v.last), true
}

// ---- the model -------------------------------------------------------------------------------------------------

// Moment is one of the five watch moments (closest call, boss, best drop, level, the stats line that moved).
type Moment struct {
	Kind  string `json:"kind"` // close, boss, drop, level, stats
	Title string `json:"title"`
	Text  string `json:"text"`
}

// Trip is one expedition card: the events between leaving town and coming back.
type Trip struct {
	N        int      `json:"n"` // 1 = the oldest in the window
	Zones    []string `json:"zones"`
	Start    float64  `json:"start"`
	End      float64  `json:"end"`
	Kills    int      `json:"kills"`
	XP       int64    `json:"xp"`
	Gold     int64    `json:"gold"` // 0 until a gold event exists (U)
	Drops    []string `json:"drops"`
	Potions  int      `json:"potions"`
	Deaths   []string `json:"deaths"` // cause per death
	Closest  int      `json:"closest,omitempty"`
	Level    int      `json:"level,omitempty"` // level at the end of the trip, 0 unknown
	Bosses   []string `json:"bosses"`
	Open     bool     `json:"open"` // the hero is still out on it
	Unsure   bool     `json:"unsure,omitempty"`
	Headline string   `json:"headline"`
}

// Choice is one answer of a decision point.
type Choice struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}

// MissedGroup is the questions that faded unanswered in the window, one group per decision point.
type MissedGroup struct {
	Point    string   `json:"point"`
	Question string   `json:"question"`
	Count    int      `json:"count"`
	Contexts []string `json:"contexts"` // up to three distinct situations, plain language
	Choices  []Choice `json:"choices"`
	// OpenAsk and OpenRule are the still open questions and "make this a rule?" confirms of the same point: what the two
	// buttons can act on (a faded question cannot be answered afterwards; the store refuses it).
	OpenAsk  int `json:"openAsk"`
	OpenRule int `json:"openRule"`
}

// SectionView is what a registered ReturnSection contributes.
type SectionView struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Lines       []string `json:"lines,omitempty"`
	Placeholder string   `json:"placeholder,omitempty"` // shown when Lines is empty
	After       bool     `json:"after,omitempty"`       // after the missed questions instead of before
	// Buttons post a policy change to /api/policy (the sheet's 'change it back'); none for the other sections.
	Buttons []SectionButton `json:"buttons,omitempty"`
}

// SectionButton is one button of a section: a policy row and the option it sets.
type SectionButton struct {
	Label  string `json:"label"`
	Row    string `json:"row"`
	Option string `json:"option"`
	// Act is the page action: "policy" (the default, POST /api/policy), "hall" (POST /api/hall/answer) or "proposal" (POST /api/proposals/answer, Row is
	// the proposal id).
	Act string `json:"act,omitempty"`
}

// ReturnSection is the plug-in point of the page (cards 129 tally, 131 proposals, 130 policy effects). Build reads the
// finished data and returns nil to hide the section.
type ReturnSection interface {
	Build(d *ReturnData) *SectionView
}

// SectionFunc adapts a function.
type SectionFunc func(d *ReturnData) *SectionView

func (f SectionFunc) Build(d *ReturnData) *SectionView { return f(d) }

// SectionRegistry is an ordered list of sections by id. Register replaces a section of the same id (a card replaces
// the placeholder in place), else appends.
type SectionRegistry struct {
	ids  []string
	secs map[string]ReturnSection
}

func (r *SectionRegistry) Register(id string, s ReturnSection) {
	if r.secs == nil {
		r.secs = map[string]ReturnSection{}
	}
	if _, ok := r.secs[id]; !ok {
		r.ids = append(r.ids, id)
	}
	r.secs[id] = s
}

// DefaultSections holds the placeholders: proposals (card 131 proposezone) and tally (card 129 tallystrip).
func DefaultSections() *SectionRegistry {
	r := &SectionRegistry{}
	r.Register("proposals", SectionFunc(func(*ReturnData) *SectionView {
		return &SectionView{ID: "proposals", Title: "Waiting for your yes", Placeholder: "Nothing is waiting. The hero keeps farming where it is."}
	}))
	r.Register("tally", SectionFunc(func(*ReturnData) *SectionView {
		return &SectionView{ID: "tally", Title: "Tally", After: true, Placeholder: "Tally is not switched on yet."}
	}))
	return r
}

// ReturnData is GET /api/return and the input of the HTML.
type ReturnData struct {
	Schema   string        `json:"schema"`
	Since    float64       `json:"since"`
	Now      float64       `json:"now"`
	Class    string        `json:"class,omitempty"`
	Headline string        `json:"headline"`
	Levels   int           `json:"levelsGained"`
	Deaths   []string      `json:"deaths"`
	Drops    []string      `json:"drops"` // worth noting, best first
	Areas    []string      `json:"areas"` // entered in the window, in order, towns left out
	Quests   []string      `json:"quests"`
	Moments  []Moment      `json:"moments"`
	Trips    []Trip        `json:"trips"` // newest first
	Missed   []MissedGroup `json:"missed"`
	Sections []SectionView `json:"sections"`
	// Gaps names what the page could not show because no event carries it (honest list, U items of docs/return-screen.md).
	Gaps []string `json:"gaps"`
}

// ReturnInput is everything BuildReturn reads.
type ReturnInput struct {
	Journal []JournalLine
	Events  []AwayEvent
	Since   float64
	Now     float64
	Hero    HeroInfo
	Cat     *decide.Catalogue // names the missed points and their answers; nil: points are shown by id only
	// OpenAsk and OpenRule count the open questions and confirms per point.
	OpenAsk, OpenRule map[string]int
	Sections          *SectionRegistry // nil: DefaultSections
}

// dropRank orders what is worth noting. Data: a new quality is one row. Runes rank by item type, not by quality.
var dropRank = map[string]int{"unique": 5, "set": 4, "rune": 3, "rare": 2, "crafted": 2}

func rankOf(e AwayEvent) (string, int) {
	if strings.EqualFold(e.Type, "rune") || strings.HasSuffix(e.Item, " Rune") {
		return "rune", dropRank["rune"]
	}
	q := strings.ToLower(e.Quality)
	return q, dropRank[q]
}

// lifeRE finds the life percent the card texts print ("Life fell to 30%", "life is 12%", "at life 8%").
func lifeRE(s string) int {
	low := strings.ToLower(s)
	for _, p := range []string{"life fell to ", "life is ", "at life ", "life at "} {
		i := strings.Index(low, p)
		if i < 0 {
			continue
		}
		n, any := 0, false
		for _, c := range low[i+len(p):] {
			if c < '0' || c > '9' {
				break
			}
			n, any = n*10+int(c-'0'), true
		}
		if any {
			return n
		}
	}
	return -1
}

func pointOf(key string) string {
	p, _, _ := strings.Cut(key, "|")
	return p
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("1 %s", one)
	}
	return fmt.Sprintf("%d %s", n, many)
}

func fmtDur(sec float64) string {
	m := int(sec/60 + 0.5)
	switch {
	case m < 1:
		return "under a minute"
	case m < 60:
		return fmt.Sprintf("%d min", m)
	}
	return fmt.Sprintf("%d h %02d min", m/60, m%60)
}

func deathCause(e AwayEvent) string {
	switch {
	case e.Monster != "" && e.Area != "":
		return "killed by " + e.Monster + " in " + e.Area
	case e.Monster != "":
		return "killed by " + e.Monster
	case e.Area != "":
		return "died in " + e.Area + " (cause not reported)"
	}
	return "died (cause not reported)"
}

// BuildReturn groups the window [Since, Now] into the page's data. It never fails: a source that is missing leaves its
// part empty and the part is named in Gaps.
func BuildReturn(in ReturnInput) ReturnData {
	d := ReturnData{Schema: returnSchema, Since: in.Since, Now: in.Now, Class: in.Hero.Class,
		Deaths: []string{}, Drops: []string{}, Areas: []string{}, Quests: []string{}, Moments: []Moment{}, Trips: []Trip{},
		Missed: []MissedGroup{}, Sections: []SectionView{}, Gaps: []string{}}
	var evs []AwayEvent
	for _, e := range in.Events {
		if e.T >= in.Since && e.T <= in.Now {
			evs = append(evs, e)
		}
	}
	sort.SliceStable(evs, func(i, j int) bool { return evs[i].T < evs[j].T })

	trips, bestDrop, boss, closeLife, closeWhere := groupTrips(evs, &d)

	// Closest call: the lowest life percent of a non-fatal fight, from a close event or the journal's fight questions
	// (their card text prints the life). U: not every fight opens a question, so this is the lowest the page can see.
	closeAt := -1
	if closeLife > 0 {
		closeAt = closeLife
	}
	for _, l := range in.Journal {
		if l.T < in.Since || l.T > in.Now || l.Kind != "question" || !strings.HasPrefix(l.Key, "t0.") {
			continue
		}
		if p := lifeRE(l.Text); p > 0 && (closeAt < 0 || p < closeAt) {
			closeAt, closeWhere = p, firstClause(l.Text)
		}
	}
	missed := map[string]*MissedGroup{}
	var order []string
	for _, l := range in.Journal {
		if l.T < in.Since || l.T > in.Now {
			continue
		}
		switch {
		case l.Kind == "missed" && l.Key != "":
			p := pointOf(l.Key)
			g := missed[p]
			if g == nil {
				g = &MissedGroup{Point: p, Question: p, Choices: []Choice{}, Contexts: []string{}}
				if in.Cat != nil {
					if pt, err := in.Cat.Point(p); err == nil {
						g.Question = pt.Question
						for _, a := range pt.Answers {
							g.Choices = append(g.Choices, Choice{ID: a.ID, Label: a.Label})
						}
					}
				}
				g.OpenAsk, g.OpenRule = in.OpenAsk[p], in.OpenRule[p]
				missed[p] = g
				order = append(order, p)
			}
			g.Count++
			if c := PlainContext(l.Key); c != "" && len(g.Contexts) < 3 && !contains(g.Contexts, c) {
				g.Contexts = append(g.Contexts, c)
			}
		case l.Kind == "incident" && strings.HasPrefix(l.Text, "The hero died"):
			if len(d.Deaths) < journalDeathsMax && !hasDeathEvent(evs) {
				d.Deaths = append(d.Deaths, strings.TrimSuffix(strings.TrimPrefix(l.Text, "The hero "), ".")+" (cause not reported)")
			}
		}
	}
	sort.SliceStable(order, func(i, j int) bool { return missed[order[i]].Count > missed[order[j]].Count })
	for _, p := range order {
		d.Missed = append(d.Missed, *missed[p])
	}

	// Levels, drops, headline.
	startLvl, endLvl := 0, in.Hero.Level
	for _, e := range evs {
		if e.Kind == AwayLevel && e.Level > 0 {
			d.Levels++
			if startLvl == 0 {
				startLvl = e.Level - 1
			}
			endLvl = e.Level
		}
	}
	var totalXP int64
	kills := 0
	potions := 0
	for _, t := range trips {
		totalXP += t.XP
		kills += t.Kills
		potions += t.Potions
	}
	for i := len(trips) - 1; i >= 0; i-- {
		d.Trips = append(d.Trips, trips[i])
	}
	if len(d.Trips) > 0 {
		d.Trips[0].Open = d.Trips[0].End >= in.Now-1 && !endedInTown(evs)
	}
	for _, e := range evs {
		switch e.Kind {
		case AwayQuest:
			if e.Done {
				d.Quests = append(d.Quests, e.Quest)
			}
		case AwayDrop:
			if _, r := rankOf(e); r >= dropRank["rare"] {
				d.Drops = append(d.Drops, fmt.Sprintf("%s (%s)", e.Item, dropLabel(e)))
			}
		}
	}
	sort.SliceStable(d.Drops, func(i, j int) bool { return dropOrder(d.Drops[i]) > dropOrder(d.Drops[j]) })
	if len(d.Drops) > 8 {
		d.Drops = d.Drops[:8]
	}

	away := in.Now - in.Since
	where := in.Hero.Area
	who := "Your hero"
	if in.Hero.Class != "" {
		who = "Your " + in.Hero.Class
	}
	h := fmt.Sprintf("Away %s. %s", fmtDur(away), who)
	if endLvl > 0 {
		h += fmt.Sprintf(" is level %d", endLvl)
		if where != "" {
			h += " in " + where
		}
	} else if where != "" {
		h += " is in " + where
	} else {
		h += " is still going"
	}
	nd := len(d.Deaths)
	if n := countKind(evs, AwayDeath); n > nd {
		nd = n
	}
	h += fmt.Sprintf(". %s, %s.", plural(len(trips), "expedition", "expeditions"), plural(nd, "death", "deaths"))
	d.Headline = h

	// The five moments, each only when it happened.
	if closeAt > 0 {
		d.Moments = append(d.Moments, Moment{"close", "Closest call", closeWhere})
	}
	if boss != "" {
		d.Moments = append(d.Moments, Moment{"boss", "Named boss", boss})
	}
	if bestDrop != "" {
		d.Moments = append(d.Moments, Moment{"drop", "Best drop", bestDrop})
	}
	if d.Levels > 0 {
		txt := fmt.Sprintf("Level %d to %d.", startLvl, endLvl)
		if len(d.Quests) > 0 {
			txt += " Quests done: " + strings.Join(d.Quests, ", ") + "."
		}
		d.Moments = append(d.Moments, Moment{"level", "Level up", txt})
	}
	if kills+nd > 0 || totalXP > 0 {
		d.Moments = append(d.Moments, Moment{"stats", "What moved", fmt.Sprintf("%s killed, %d XP, %s, %s drunk, %s entered.",
			plural(kills, "monster", "monsters"), totalXP, plural(nd, "death", "deaths"), plural(potions, "potion", "potions"), plural(len(d.Areas), "area", "areas"))})
	}

	// Gaps: what no event provides today. U items are listed in docs/return-screen.md.
	if len(in.Events) == 0 {
		d.Gaps = append(d.Gaps, "No event stream was recorded: expeditions, drops, levels and kills are empty. Feed AwayLog.OnGoalEvent.")
	}
	d.Gaps = append(d.Gaps, "Gold, waypoints lit and supply cost per trip have no event yet (shown as not reported).")

	reg := in.Sections
	if reg == nil {
		reg = DefaultSections()
	}
	for _, id := range reg.ids {
		if sv := reg.secs[id].Build(&d); sv != nil {
			if sv.ID == "" {
				sv.ID = id
			}
			d.Sections = append(d.Sections, *sv)
		}
	}
	return d
}

const journalDeathsMax = 20

func hasDeathEvent(evs []AwayEvent) bool { return countKind(evs, AwayDeath) > 0 }

func countKind(evs []AwayEvent, k string) int {
	n := 0
	for _, e := range evs {
		if e.Kind == k {
			n++
		}
	}
	return n
}

// endedInTown says whether the last area change went to a town.
func endedInTown(evs []AwayEvent) bool {
	for i := len(evs) - 1; i >= 0; i-- {
		if evs[i].Kind == AwayArea {
			return data.IsTown(evs[i].AreaID)
		}
	}
	return false
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func firstClause(t string) string {
	if i := strings.Index(t, " When: "); i >= 0 {
		t = t[i+len(" When: "):]
	}
	if i := strings.Index(t, " Meanwhile"); i >= 0 {
		t = t[:i]
	}
	return t
}

func dropLabel(e AwayEvent) string {
	if q, _ := rankOf(e); q != "" {
		return q
	}
	return "item"
}

func dropOrder(s string) int {
	for q, r := range dropRank {
		if strings.HasSuffix(s, "("+q+")") {
			return r
		}
	}
	return 0
}

// groupTrips cuts the events at town: an area change into a town ends the trip, the next change out of it starts one.
// Trip boundaries are U until traced on a real soak (docs/return-screen.md): the rule is "area_change to a town id".
func groupTrips(evs []AwayEvent, d *ReturnData) (trips []Trip, bestDrop, boss string, closeLife int, closeWhere string) {
	var cur *Trip
	bestRank := -1
	seen := map[string]bool{}
	open := func(t float64) {
		cur = &Trip{N: len(trips) + 1, Start: t, End: t, Zones: []string{}, Drops: []string{}, Deaths: []string{}, Bosses: []string{}}
	}
	closeTrip := func(t float64) {
		if cur == nil {
			return
		}
		cur.End = t
		cur.Headline = tripHeadline(cur)
		trips = append(trips, *cur)
		cur = nil
	}
	for _, e := range evs {
		if e.Kind == AwayArea && data.IsTown(e.AreaID) {
			closeTrip(e.T)
			continue
		}
		if cur == nil {
			open(e.T)
		}
		cur.End = e.T
		switch e.Kind {
		case AwayArea:
			if e.Area != "" && !contains(cur.Zones, e.Area) {
				cur.Zones = append(cur.Zones, e.Area)
			}
			if e.Area != "" && !seen[e.Area] {
				seen[e.Area] = true
				d.Areas = append(d.Areas, e.Area)
			}
		case AwayKill:
			cur.Kills++
			cur.XP += e.XP
			if e.Boss && e.Monster != "" {
				cur.Bosses = append(cur.Bosses, e.Monster)
				boss = "Killed " + e.Monster + "."
			}
		case AwayDrop:
			if q, r := rankOf(e); r >= dropRank["rare"] {
				cur.Drops = append(cur.Drops, fmt.Sprintf("%s (%s)", e.Item, firstNonEmpty(q, "item")))
				if r > bestRank {
					bestRank, bestDrop = r, fmt.Sprintf("%s (%s).", e.Item, firstNonEmpty(q, "item"))
				}
			}
		case AwayPotion:
			cur.Potions++
		case AwayDeath:
			c := deathCause(e)
			cur.Deaths = append(cur.Deaths, c)
			d.Deaths = append(d.Deaths, c)
		case AwayLevel:
			cur.Level = e.Level
		case AwayGold:
			cur.Gold += e.Gold
		case AwayClose:
			if e.LifePct > 0 && (closeLife == 0 || e.LifePct < closeLife) {
				closeLife, closeWhere = e.LifePct, fmt.Sprintf("Life fell to %d%%", e.LifePct)
				if e.Area != "" {
					closeWhere += " in " + e.Area
				}
				closeWhere += "."
				cur.Closest = e.LifePct
			}
		}
	}
	if cur != nil {
		cur.Unsure = len(cur.Zones) == 0
		cur.Headline = tripHeadline(cur)
		trips = append(trips, *cur)
	}
	return
}

func firstNonEmpty(a ...string) string {
	for _, s := range a {
		if s != "" {
			return s
		}
	}
	return ""
}

func tripHeadline(t *Trip) string {
	z := strings.Join(t.Zones, ", ")
	if z == "" {
		z = "an unknown area"
	}
	return fmt.Sprintf("%s for %s", z, fmtDur(t.End-t.Start))
}

// ---- Hub glue --------------------------------------------------------------------------------------------------

// ReturnConfig connects the page to the running bot. Hub is required; the rest is optional.
type ReturnConfig struct {
	Hub    *Hub
	Events *AwayLog
	Visit  *Visit
	// Speed is virtual seconds per wall second (Config.Speed); maps a wall absence into the Hub's time base.
	Speed    float64
	Sections *SectionRegistry
	// Notes receives the three sentences of the session record (TASK-106). nil: the box says no record is connected.
	Notes NotesSink
	// OnRead is called with the data of every served HTML return page (a card the viewer is shown); nil: none.
	OnRead func(ReturnData)
}

// NotesSink is what the session record (TASK-106) implements.
type NotesSink interface{ SaveSessionNotes(text string) error }

// Return builds the page data for the window since the last visit (or since `since` seconds when > 0).
func (c ReturnConfig) Return(sinceOverride float64) ReturnData {
	h := c.Hub
	now := h.rel(h.cfg.Clock.Now())
	since := 0.0
	if sinceOverride > 0 {
		since = max(0, now-sinceOverride)
	} else if d, ok := c.Visit.awayOK(); ok {
		sp := c.Speed
		if sp <= 0 {
			sp = 1
		}
		since = max(0, now-d.Seconds()*sp)
	}
	in := ReturnInput{Journal: h.Journal(0, "").Lines, Events: c.Events.Events(), Since: since, Now: now,
		Cat: h.cfg.Cat, Sections: c.Sections, OpenAsk: map[string]int{}, OpenRule: map[string]int{}}
	h.mu.Lock()
	in.Hero = h.hero
	h.mu.Unlock()
	for _, t := range h.cfg.Store.OpenThoughts() {
		if t.Confirm {
			in.OpenRule[t.Point]++
		} else {
			in.OpenAsk[t.Point]++
		}
	}
	return BuildReturn(in)
}

func (v *Visit) awayOK() (time.Duration, bool) {
	if v == nil {
		return 0, false
	}
	return v.Away()
}

// Absent says whether the last look is older than AwayAfter. No tracker or no earlier look: false (a first start is not
// a return).
func (c ReturnConfig) Absent() bool {
	d, ok := c.Visit.awayOK()
	return ok && d > AwayAfter
}

// AnswerKind answers every open question of one decision point with the same answer ("answer once for all of them").
// It reuses Hub.Answer, so each answer counts like a tap on the thought (the store's streak and rule learning apply).
// It returns how many were answered; none open is ErrUnknownThought.
func (h *Hub) AnswerKind(point, answer string) (int, error) {
	return h.eachOpen(point, false, func(id int) error { return h.Answer(id, answer) })
}

// ConfirmKind says yes or no to every open "make this a rule?" of one decision point.
func (h *Hub) ConfirmKind(point string, yes bool) (int, error) {
	return h.eachOpen(point, true, func(id int) error { return h.Confirm(id, yes) })
}

func (h *Hub) eachOpen(point string, confirm bool, f func(id int) error) (int, error) {
	n := 0
	for _, t := range h.cfg.Store.OpenThoughts() {
		if t.Point != point || t.Confirm != confirm {
			continue
		}
		if err := f(t.ID); err != nil {
			if n > 0 {
				return n, err
			}
			return 0, err
		}
		n++
	}
	if n == 0 {
		return 0, fmt.Errorf("%w: no open question for %q", ErrUnknownThought, safeText(point, 60))
	}
	return n, nil
}

// ReturnRoutes are the GET routes ReturnHandler serves. ADD THEM to the viewer's mux (serve.go / ViewRoutes; outside
// the TASK-127 owns, see docs/return-screen.md "Wiring").
var ReturnRoutes = []string{"/watch/return", "/api/return", "/watch/return.js", "/watch/return.css"}

// ReturnPostRoutes are the POST routes of ReturnAnswerHandler (mount on the answer listener).
var ReturnPostRoutes = []string{"/api/return/answer-kind", "/api/return/notes"}

const returnCSP = "default-src 'none'; script-src 'self'; style-src 'self'; img-src 'self' data:; connect-src 'self' http: https:; base-uri 'none'; form-action 'none'; frame-ancestors 'none'"

// ReturnHandler serves the page, its JSON and its two small assets, read-only. With wrap = true it also handles
// /watch itself: after an absence over AwayAfter it redirects to the return screen (once; /watch?live=1 or the link on
// the page goes on to the live screen and counts as a look), else it falls through to next.
func (c ReturnConfig) ReturnHandler(next http.Handler) http.Handler {
	m := http.NewServeMux()
	hdr := func(w http.ResponseWriter, ctype string) {
		w.Header().Set("Content-Type", ctype)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", returnCSP)
	}
	since := func(r *http.Request) float64 {
		var s float64
		if h := r.URL.Query().Get("hours"); h != "" {
			fmt.Sscanf(h, "%f", &s)
			s *= 3600
		}
		if s < 0 || s > 7*86400 {
			s = 0
		}
		return s
	}
	m.HandleFunc("/watch/return", func(w http.ResponseWriter, r *http.Request) {
		hdr(w, "text/html; charset=utf-8")
		d := c.Return(since(r))
		if c.OnRead != nil && r.Method == http.MethodGet {
			c.OnRead(d)
		}
		w.Write(RenderReturn(d, c.Notes != nil))
	})
	m.HandleFunc("/api/return", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, c.Return(since(r))) })
	m.HandleFunc("/watch/return.js", func(w http.ResponseWriter, r *http.Request) {
		hdr(w, "text/javascript; charset=utf-8")
		io.WriteString(w, returnJS)
	})
	m.HandleFunc("/watch/return.css", func(w http.ResponseWriter, r *http.Request) {
		hdr(w, "text/css; charset=utf-8")
		io.WriteString(w, returnCSS)
	})
	m.HandleFunc("/watch", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("live") != "" {
			c.Visit.touch()
		} else if c.Visit.due() {
			http.Redirect(w, r, "/watch/return", http.StatusFound)
			return
		}
		if next == nil {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/thoughts" {
			c.Visit.touch() // the open live page polls this: a look in progress
		}
		if next == nil {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
	return readOnly(m)
}

func (v *Visit) touch() {
	if v != nil {
		v.Touch()
	}
}

type returnBody struct {
	Point  string `json:"point"`
	Answer string `json:"answer"`
	Yes    *bool  `json:"yes"`
	Text   string `json:"text"`
}

// ReturnAnswerHandler is the write side of the page: POST /api/return/answer-kind ({point, answer} answers every open
// question of the point; {point, yes} answers every open "make this a rule?") and /api/return/notes ({text}). Same
// gate as AnswerHandler (origin list, token, JSON only); mount it on the answer listener.
func (c ReturnConfig) ReturnAnswerHandler(o AnswerOptions) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		allowed := origin != "" && containsStr(o.Origins, origin)
		if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Vary", "Origin")
		}
		fail := func(code int, c, msg string) { writeJSON(w, code, reply{Code: c, Error: msg}) }
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
		case !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json"):
			fail(http.StatusUnsupportedMediaType, "content-type", "send application/json")
			return
		}
		var b returnBody
		dec := json.NewDecoder(io.LimitReader(r.Body, maxBody+1))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&b); err != nil || dec.More() {
			fail(http.StatusBadRequest, "body", "body must be one small JSON object with known fields")
			return
		}
		switch r.URL.Path {
		case "/api/return/answer-kind":
			var n int
			var err error
			switch {
			case b.Yes != nil && b.Answer == "":
				n, err = c.Hub.ConfirmKind(b.Point, *b.Yes)
			case b.Answer != "" && b.Yes == nil:
				n, err = c.Hub.AnswerKind(b.Point, b.Answer)
			default:
				fail(http.StatusBadRequest, "body", "send answer or yes, not both")
				return
			}
			if err != nil {
				code, rep := status(err)
				if n > 0 {
					rep.Revoked = n // reused as "done before the error"
				}
				writeJSON(w, code, rep)
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true, "answered": n})
		case "/api/return/notes":
			if c.Notes == nil {
				fail(http.StatusNotImplemented, "no-record", "no session record is connected")
				return
			}
			t := safeText(b.Text, 900)
			if t == "" {
				fail(http.StatusBadRequest, "body", "text is empty")
				return
			}
			if err := c.Notes.SaveSessionNotes(t); err != nil {
				fail(http.StatusInternalServerError, "internal", errors.New("could not save the notes").Error())
				return
			}
			writeJSON(w, 200, map[string]any{"ok": true})
		default:
			fail(http.StatusNotFound, "route", "no such route")
		}
	})
}

// ---- the page --------------------------------------------------------------------------------------------------

var returnTmpl = template.Must(template.New("return").Funcs(template.FuncMap{"dur": fmtDur, "clock": func(sec float64) string {
	s := int(sec)
	return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
}}).Parse(`<!DOCTYPE html>
<html lang="en"><head><meta charset="utf-8"><title>d.idle - while you were away</title>
<meta name="viewport" content="width=device-width,initial-scale=1"><meta name="color-scheme" content="light dark">
<link rel="icon" href="data:,"><link rel="stylesheet" href="/watch/app.css"><link rel="stylesheet" href="/watch/return.css">
</head><body>
<header class="top"><div class="brand"><span class="brand-mark" aria-hidden="true"></span><h1>d.idle</h1></div>
<a class="btn primary" href="/watch?live=1">Go to the live screen</a></header>
<main class="ret" id="return">
<section class="card" aria-labelledby="r-head"><h2 id="r-head">While you were away</h2><p class="headline">{{.Headline}}</p></section>
{{if .Moments}}<section class="card" aria-labelledby="r-moments"><h2 id="r-moments">The moments</h2><ul class="moments">
{{range .Moments}}<li class="moment {{.Kind}}"><b>{{.Title}}</b> <span>{{.Text}}</span></li>{{end}}</ul></section>{{end}}
{{if .Deaths}}<section class="card" aria-labelledby="r-deaths"><h2 id="r-deaths">Deaths</h2><ul>{{range .Deaths}}<li>{{.}}</li>{{end}}</ul></section>{{end}}
{{if .Drops}}<section class="card" aria-labelledby="r-drops"><h2 id="r-drops">Drops worth noting</h2><ul>{{range .Drops}}<li>{{.}}</li>{{end}}</ul></section>{{end}}
{{if .Areas}}<section class="card" aria-labelledby="r-areas"><h2 id="r-areas">Areas entered</h2><p>{{range $i, $a := .Areas}}{{if $i}}, {{end}}{{$a}}{{end}}</p>
<p class="hint">Waypoints lit: not reported yet (no event).</p></section>{{end}}
<section class="card" aria-labelledby="r-trips"><h2 id="r-trips">Expeditions</h2>
{{if .Trips}}<div class="trips">{{range .Trips}}<article class="trip" aria-label="Expedition {{.N}}">
<h3>#{{.N}} {{.Headline}}{{if .Open}} <span class="badge">still out</span>{{end}}{{if .Unsure}} <span class="badge">start not seen</span>{{end}}</h3>
<p class="hint">{{clock .Start}} to {{clock .End}}</p>
<dl><dt>Kills</dt><dd>{{.Kills}}</dd><dt>XP</dt><dd>{{.XP}}</dd><dt>Gold</dt><dd>{{if .Gold}}{{.Gold}}{{else}}not reported{{end}}</dd>
<dt>Potions drunk</dt><dd>{{.Potions}}</dd><dt>Supplies cost</dt><dd>not reported</dd></dl>
{{if .Bosses}}<p>Bosses: {{range $i, $b := .Bosses}}{{if $i}}, {{end}}{{$b}}{{end}}</p>{{end}}
{{if .Drops}}<p>Kept: {{range $i, $b := .Drops}}{{if $i}}, {{end}}{{$b}}{{end}}</p>{{end}}
{{range .Deaths}}<p class="bad">Died: {{.}}</p>{{end}}
</article>{{end}}</div>{{else}}<p class="empty">No expedition in this window.</p>{{end}}</section>
{{range .Sections}}{{if not .After}}{{template "sec" .}}{{end}}{{end}}
<section class="card" aria-labelledby="r-missed"><h2 id="r-missed">Questions I missed</h2>
{{if .Missed}}{{range .Missed}}<article class="missed" data-point="{{.Point}}"><h3>{{.Question}} <span class="badge">{{.Count}}x</span></h3>
{{range .Contexts}}<p class="hint">{{.}}</p>{{end}}
{{if .OpenAsk}}<p>Answer once for all {{.OpenAsk}} still open:</p><div class="row">{{$p := .Point}}{{range .Choices}}<button type="button" class="btn small" data-act="answer" data-point="{{$p}}" data-answer="{{.ID}}">{{.Label}}</button>{{end}}</div>
{{else}}<p class="hint">Faded questions cannot be answered afterwards. I will ask again and carry on with my default until then.</p>{{end}}
{{if .OpenRule}}<p>Make this a rule ({{.OpenRule}} waiting):</p><div class="row"><button type="button" class="btn small primary" data-act="rule" data-point="{{.Point}}" data-yes="1">Yes, make it a rule</button><button type="button" class="btn small" data-act="rule" data-point="{{.Point}}" data-yes="0">No</button></div>{{end}}
</article>{{end}}{{else}}<p class="empty">Nothing was missed.</p>{{end}}</section>
{{range .Sections}}{{if .After}}{{template "sec" .}}{{end}}{{end}}
<section class="card" aria-labelledby="r-notes"><h2 id="r-notes">Three sentences about this session</h2>
<textarea id="notes" rows="4" maxlength="900" aria-label="Three sentences about this session"></textarea>
<div class="row"><button type="button" class="btn small" id="notes-save" data-act="notes" {{if not .Notes}}disabled{{end}}>Save</button><span class="hint" id="notes-state">{{if not .Notes}}No session record is connected yet.{{end}}</span></div></section>
{{if .Gaps}}<section class="card" aria-labelledby="r-gaps"><h2 id="r-gaps">What this page cannot see yet</h2><ul class="hint">{{range .Gaps}}<li>{{.}}</li>{{end}}</ul></section>{{end}}
</main>
<p id="msg" class="hint" role="status"></p>
<script src="/watch/return.js"></script>
</body></html>
{{define "sec"}}<section class="card" data-section="{{.ID}}" aria-label="{{.Title}}"><h2>{{.Title}}</h2>
{{if .Lines}}<ul>{{range .Lines}}<li>{{.}}</li>{{end}}</ul>{{else}}<p class="empty">{{.Placeholder}}</p>{{end}}{{if .Buttons}}<div class="row">{{range .Buttons}}<button type="button" class="btn small" data-act="{{if .Act}}{{.Act}}{{else}}policy{{end}}" data-row="{{.Row}}" data-option="{{.Option}}">{{.Label}}</button>{{end}}</div>{{end}}</section>{{end}}`))

type returnPage struct {
	ReturnData
	Notes bool
}

// RenderReturn is the page as server-rendered HTML; notes says whether the three-sentence box can save.
func RenderReturn(d ReturnData, notes bool) []byte {
	var b strings.Builder
	if err := returnTmpl.Execute(&b, returnPage{d, notes}); err != nil {
		return []byte("render error")
	}
	return []byte(b.String())
}

// main.ret must reset the global main grid (12 columns from 900 px up in app.css), or every card becomes a narrow column.
const returnCSS = `main.ret{display:block;grid-template-columns:none;max-width:46rem;margin:0 auto;padding:12px}.ret .card{min-width:0;overflow-wrap:anywhere}.ret .card{margin-bottom:12px;padding:12px 14px}
.headline{font-size:1.1rem;margin:0}.moments{list-style:none;margin:0;padding:0}.moment{padding:6px 0;border-bottom:1px solid var(--line)}
.trip,.missed{border:1px solid var(--line);border-radius:8px;padding:8px 12px;margin-bottom:8px}.trip h3,.missed h3{margin:0;font-size:.95rem}
.trip dl{display:grid;grid-template-columns:auto 1fr;gap:2px 12px;margin:6px 0}.trip dt{color:var(--muted)}.trip dd{margin:0}
.bad{color:var(--bad)}.row{display:flex;flex-wrap:wrap;gap:6px;margin:6px 0}textarea{width:100%;background:var(--surface);border:1px solid var(--line);border-radius:7px;padding:8px}`

// returnJS posts the three kinds of write to the answer route (another port, like app.js). All text via textContent.
const returnJS = `(function(){'use strict';
var msg=document.getElementById('msg');
function say(t){msg.textContent=t||'';}
function base(){return fetch('/api/config',{cache:'no-store'}).then(function(r){return r.json();}).then(function(c){
 if(c.answerBase)return c.answerBase;return location.protocol+'//'+location.hostname+':'+c.answerPort;});}
function post(path,body){return base().then(function(b){return fetch(b+path,{method:'POST',credentials:'include',
 headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});}).then(function(r){return r.json();})
 .then(function(j){say(j.ok?'Done'+(j.answered?' ('+j.answered+' answered)':''):(j.error||'failed'));if(j.ok&&(j.answered||j.changed))setTimeout(function(){location.reload();},600);})
 .catch(function(){say('the answer route is not reachable');});}
document.addEventListener('click',function(e){var b=e.target.closest('button[data-act]');if(!b||b.disabled)return;
 var a=b.dataset.act;
 if(a==='answer')post('/api/return/answer-kind',{point:b.dataset.point,answer:b.dataset.answer});
 else if(a==='rule')post('/api/return/answer-kind',{point:b.dataset.point,yes:b.dataset.yes==='1'});
 else if(a==='policy')post('/api/policy',{row:b.dataset.row,option:b.dataset.option});
 else if(a==='hall')post('/api/hall/answer',{answer:b.dataset.option});
 else if(a==='proposal')post('/api/proposals/answer',{id:parseInt(b.dataset.row,10),answer:b.dataset.option});
 else if(a==='notes')post('/api/return/notes',{text:document.getElementById('notes').value});});
})();`
