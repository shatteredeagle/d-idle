package telemetry

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// Event kinds.
const (
	KindKill     = "kill"
	KindDrop     = "drop"
	KindDeath    = "death"
	KindLevelUp  = "levelup"
	KindArea     = "area"
	KindError    = "error"
	KindPotion   = "potion"
	KindActivity = "activity"
	KindRun      = "run"
	KindInfo     = "info"
)

// Monster modes that mean "dying" (0) and "dead" (12).
// UNVERIFIED for 1.14d beyond D2BS constants; see docs/d2-1.14d-memory.md 2.5.
const (
	ModeDeath = 0
	ModeDead  = 12
)

type Monster struct {
	ID       uint32
	TxtNo    int
	Name     string
	Mode     int
	X, Y     int
	Friendly bool // merc, pets, town NPCs: never counted as kills
}

type GroundItem struct {
	ID      uint32
	Name    string
	Quality string
	X, Y    int
}

type Belt struct{ Health, Mana, Rejuv int }

// Frame is one observation of the game. The bot builds it from internal/d2.
type Frame struct {
	InGame   bool
	InTown   bool
	Area     int
	Level    int
	Exp      uint32
	Gold     int
	StashGld int
	Dead     bool
	Monsters []Monster
	Ground   []GroundItem
	Belt     Belt
}

type Event struct {
	Seq   uint64    `json:"seq"`
	Time  time.Time `json:"time"`
	Kind  string    `json:"kind"`
	Text  string    `json:"text"`
	Area  int       `json:"area"`
	Value float64   `json:"value,omitempty"`
	Extra string    `json:"extra,omitempty"` // quality for drops
}

type Segment struct {
	Start  time.Time  `json:"start"`
	End    *time.Time `json:"end,omitempty"`
	State  string     `json:"state"`
	Detail string     `json:"detail,omitempty"`
}

type AreaStat struct {
	Area   int           `json:"area"`
	Visits int           `json:"visits"`
	Time   time.Duration `json:"-"`
	Secs   float64       `json:"secs"`
	Kills  int           `json:"kills"`
	Drops  int           `json:"drops"`
	XP     float64       `json:"xp"`
	Gold   float64       `json:"gold"`
}

type Bucket struct {
	T     time.Time `json:"t"` // start of the minute
	XP    float64   `json:"xp"`
	Kills float64   `json:"kills"`
	Gold  float64   `json:"gold"`
}

type Config struct {
	// ExpForLevel returns the total experience needed to reach level lvl.
	ExpForLevel func(lvl int) (uint32, bool)
	EventCap    int
	AreaName    func(id int) string // optional, for event text
}

type Goal struct {
	Name     string  `json:"name"`
	Progress float64 `json:"progress"` // 0..1
	Detail   string  `json:"detail,omitempty"`
}

type Tracker struct {
	mu    sync.Mutex
	clock Clock
	cfg   Config

	started   bool
	startTime time.Time
	frames    uint64

	area    int
	areaAt  time.Time
	level   int
	exp     uint32
	gold    int
	dead    bool
	belt    Belt
	monst   map[uint32]*mstate
	ground  map[uint32]struct{}
	last    Frame
	hasLast bool

	xp, kills, golds, drops *Rate
	deaths, potions         int
	levelUps                int
	potionBy                [3]int

	areas   map[int]*AreaStat
	events  *Ring[Event]
	dropLog *Ring[Event]
	segs    *Ring[Segment]
	buckets *Ring[Bucket]
	seq     uint64

	activity, activityDetail string
	goal                     Goal
	runName                  string
	runStart                 time.Time
	runs                     *Ring[time.Duration]
	runCount                 int
	runTotal                 time.Duration

	subs    []func(Event)
	pending []Event
}

type mstate struct {
	mode int
	seen uint64
}

func New(clock Clock, cfg Config) *Tracker {
	if cfg.EventCap <= 0 {
		cfg.EventCap = 1000
	}
	mk := func() *Rate { return NewRate(4096, time.Second) }
	return &Tracker{
		clock: clock, cfg: cfg,
		monst: map[uint32]*mstate{}, ground: map[uint32]struct{}{},
		xp: mk(), kills: mk(), golds: mk(), drops: mk(),
		areas:   map[int]*AreaStat{},
		events:  NewRing[Event](cfg.EventCap),
		dropLog: NewRing[Event](200),
		segs:    NewRing[Segment](500),
		buckets: NewRing[Bucket](24 * 60),
		runs:    NewRing[time.Duration](200),
	}
}

// Now returns the tracker's clock time.
func (t *Tracker) Now() time.Time { return t.clock.Now() }

// OnEvent registers a callback invoked (outside the lock) for each new event.
func (t *Tracker) OnEvent(f func(Event)) {
	t.mu.Lock()
	t.subs = append(t.subs, f)
	t.mu.Unlock()
}

func (t *Tracker) emit(kind, text string, value float64, extra string) {
	t.seq++
	ev := Event{Seq: t.seq, Time: t.clock.Now(), Kind: kind, Text: text, Area: t.area, Value: value, Extra: extra}
	t.events.Push(ev)
	if kind == KindDrop {
		t.dropLog.Push(ev)
	}
	t.pending = append(t.pending, ev)
}

func (t *Tracker) flush() {
	evs, subs := t.pending, t.subs
	t.pending = nil
	if len(evs) == 0 || len(subs) == 0 {
		return
	}
	t.mu.Unlock()
	for _, e := range evs {
		for _, f := range subs {
			f(e)
		}
	}
	t.mu.Lock()
}

func (t *Tracker) areaStat(id int) *AreaStat {
	a := t.areas[id]
	if a == nil {
		a = &AreaStat{Area: id}
		t.areas[id] = a
	}
	return a
}

func (t *Tracker) bucket(now time.Time) *Bucket {
	m := now.Truncate(time.Minute)
	if b := t.buckets.Newest(); b != nil && b.T.Equal(m) {
		return b
	}
	t.buckets.Push(Bucket{T: m})
	return t.buckets.Newest()
}

func (t *Tracker) leaveArea(now time.Time) {
	if t.area != 0 {
		t.areaStat(t.area).Time += now.Sub(t.areaAt)
	}
}

// Observe feeds one game frame. Frames with InGame=false are ignored except
// that they force a re-baseline on the next in-game frame.
func (t *Tracker) Observe(f Frame) {
	t.mu.Lock()
	defer t.mu.Unlock()
	defer t.flush()
	now := t.clock.Now()
	if !t.started {
		t.started, t.startTime = true, now
	}
	if !f.InGame {
		if t.hasLast {
			t.leaveArea(now)
			t.area, t.hasLast = 0, false
			t.monst = map[uint32]*mstate{}
			t.ground = map[uint32]struct{}{}
		}
		return
	}
	t.frames++
	first := !t.hasLast
	t.last, t.hasLast = f, true
	b := t.bucket(now)

	if first || f.Area != t.area {
		if !first {
			t.leaveArea(now)
		}
		t.area, t.areaAt = f.Area, now
		t.areaStat(f.Area).Visits++
		t.monst = map[uint32]*mstate{}
		t.ground = map[uint32]struct{}{}
		name := fmt.Sprintf("area %d", f.Area)
		if t.cfg.AreaName != nil {
			name = t.cfg.AreaName(f.Area)
		}
		t.emit(KindArea, "entered "+name, float64(f.Area), "")
		first = true // silent baseline for units
	}
	as := t.areaStat(f.Area)

	// experience / level
	if t.level != 0 && f.Exp > t.exp {
		d := float64(f.Exp - t.exp)
		t.xp.Add(now, d)
		b.XP += d
		as.XP += d
	} else {
		t.xp.Touch(now)
	}
	if t.level != 0 && f.Level > t.level {
		t.levelUps += f.Level - t.level
		t.emit(KindLevelUp, fmt.Sprintf("reached level %d", f.Level), float64(f.Level), "")
	}
	t.level, t.exp = f.Level, f.Exp

	// gold: only increases of carried+stash count as earned
	g := f.Gold + f.StashGld
	if t.gold != 0 && g > t.gold {
		d := float64(g - t.gold)
		t.golds.Add(now, d)
		b.Gold += d
		as.Gold += d
	} else {
		t.golds.Touch(now)
	}
	t.gold = g

	// death
	if f.Dead && !t.dead {
		t.deaths++
		t.emit(KindDeath, "the character died", 0, "")
	}
	t.dead = f.Dead

	// kills
	for _, m := range f.Monsters {
		if m.Friendly {
			continue
		}
		dead := m.Mode == ModeDeath || m.Mode == ModeDead
		old := t.monst[m.ID]
		if old == nil {
			t.monst[m.ID] = &mstate{mode: m.Mode, seen: t.frames}
			continue
		}
		oldDead := old.mode == ModeDeath || old.mode == ModeDead
		if dead && !oldDead {
			t.kills.Add(now, 1)
			b.Kills++
			as.Kills++
			name := m.Name
			if name == "" {
				name = fmt.Sprintf("monster #%d", m.TxtNo)
			}
			t.emit(KindKill, "killed "+name, float64(m.TxtNo), "")
		}
		old.mode, old.seen = m.Mode, t.frames
	}
	if len(t.monst) > 2048 || t.frames%64 == 0 {
		for id, s := range t.monst {
			if t.frames-s.seen > 100 {
				delete(t.monst, id)
			}
		}
	}

	// drops
	if len(t.ground) > 4096 {
		t.ground = map[uint32]struct{}{}
	}
	for _, it := range f.Ground {
		if _, ok := t.ground[it.ID]; ok {
			continue
		}
		t.ground[it.ID] = struct{}{}
		if first {
			continue
		}
		t.drops.Add(now, 1)
		as.Drops++
		name := it.Name
		if name == "" {
			name = "item"
		}
		t.emit(KindDrop, name+" dropped", 0, it.Quality)
	}

	// potions
	if !first && !f.InTown {
		use := func(prev, cur, idx int, label string) {
			for i := cur; i < prev; i++ {
				t.potions++
				t.potionBy[idx]++
				t.emit(KindPotion, "used "+label+" potion", 0, "")
			}
		}
		use(t.belt.Health, f.Belt.Health, 0, "health")
		use(t.belt.Mana, f.Belt.Mana, 1, "mana")
		use(t.belt.Rejuv, f.Belt.Rejuv, 2, "rejuvenation")
	}
	t.belt = f.Belt
}

// SetActivity records what the bot is doing for the timeline bar.
func (t *Tracker) SetActivity(state, detail string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if state == t.activity && detail == t.activityDetail {
		return
	}
	now := t.clock.Now()
	if n := t.segs.Newest(); n != nil && n.End == nil {
		e := now
		n.End = &e
	}
	t.activity, t.activityDetail = state, detail
	t.segs.Push(Segment{Start: now, State: state, Detail: detail})
	t.emit(KindActivity, state+detailSuffix(detail), 0, "")
	t.flush()
}

func detailSuffix(d string) string {
	if d == "" {
		return ""
	}
	return ": " + d
}

func (t *Tracker) SetGoal(g Goal) { t.mu.Lock(); t.goal = g; t.mu.Unlock() }

// Error records a bot error in the event log.
func (t *Tracker) Error(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.emit(KindError, text, 0, "")
	t.flush()
}

// Info records a free-form note.
func (t *Tracker) Info(text string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.emit(KindInfo, text, 0, "")
	t.flush()
}

// RunStart / RunEnd bracket one run (for example "Blood Moor").
func (t *Tracker) RunStart(name string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.runName, t.runStart = name, t.clock.Now()
	t.emit(KindRun, "run started: "+name, 0, "")
	t.flush()
}

func (t *Tracker) RunEnd() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.runStart.IsZero() {
		return
	}
	d := t.clock.Now().Sub(t.runStart)
	t.runs.Push(d)
	t.runCount++
	t.runTotal += d
	t.emit(KindRun, fmt.Sprintf("run finished: %s (%s)", t.runName, d.Round(time.Second)), d.Seconds(), "")
	t.runStart = time.Time{}
	t.flush()
}

// RateSet is one window's rates.
type RateSet struct {
	Window  string  `json:"window"`
	Secs    float64 `json:"secs"` // effective seconds covered
	Partial bool    `json:"partial"`
	XP      float64 `json:"xp"`
	Kills   float64 `json:"kills"`
	Gold    float64 `json:"gold"`
	Drops   float64 `json:"drops"`
}

type Overview struct {
	Activity     string    `json:"activity"`
	Detail       string    `json:"detail,omitempty"`
	Goal         Goal      `json:"goal"`
	Area         int       `json:"area"`
	InTown       bool      `json:"inTown"`
	Level        int       `json:"level"`
	Exp          uint32    `json:"exp"`
	ExpThisLevel uint32    `json:"expThisLevel"`
	ExpNext      uint32    `json:"expNext"`
	ETASecs      float64   `json:"etaSecs"` // -1 unknown
	Rates        []RateSet `json:"rates"`
	TotalXP      float64   `json:"totalXP"`
	TotalKills   int       `json:"totalKills"`
	TotalGold    float64   `json:"totalGold"`
	TotalDrops   int       `json:"totalDrops"`
	Deaths       int       `json:"deaths"`
	LevelUps     int       `json:"levelUps"`
	Potions      int       `json:"potions"`
	PotionsBy    [3]int    `json:"potionsBy"` // health, mana, rejuv
	Runs         int       `json:"runs"`
	AvgRunSecs   float64   `json:"avgRunSecs"`
	CurRunSecs   float64   `json:"curRunSecs"`
	CurRun       string    `json:"curRun,omitempty"`
	SessionSecs  float64   `json:"sessionSecs"`
	InGame       bool      `json:"inGame"`
}

func (t *Tracker) rateSet(label string, now time.Time, w time.Duration) RateSet {
	rs := RateSet{Window: label}
	var p bool
	if w == 0 { // session
		rs.XP = t.xp.SessionPerHour(now)
		rs.Kills = t.kills.SessionPerHour(now)
		rs.Gold = t.golds.SessionPerHour(now)
		rs.Drops = t.drops.SessionPerHour(now)
		rs.Secs = now.Sub(t.startTime).Seconds()
		return rs
	}
	rs.XP, p = t.xp.PerHour(now, w)
	rs.Kills, _ = t.kills.PerHour(now, w)
	rs.Gold, _ = t.golds.PerHour(now, w)
	rs.Drops, _ = t.drops.PerHour(now, w)
	rs.Partial = p
	rs.Secs = w.Seconds()
	if p {
		rs.Secs = now.Sub(t.startTime).Seconds()
	}
	return rs
}

func (t *Tracker) Overview() Overview {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.clock.Now()
	o := Overview{
		Activity: t.activity, Detail: t.activityDetail, Goal: t.goal,
		Area: t.area, InTown: t.last.InTown, Level: t.level, Exp: t.exp, ETASecs: -1,
		TotalXP: t.xp.Total(), TotalKills: int(t.kills.Total()), TotalGold: t.golds.Total(),
		TotalDrops: int(t.drops.Total()), Deaths: t.deaths, LevelUps: t.levelUps,
		Potions: t.potions, PotionsBy: t.potionBy, Runs: t.runCount, InGame: t.hasLast,
		CurRun: t.runName,
	}
	if t.runCount > 0 {
		o.AvgRunSecs = t.runTotal.Seconds() / float64(t.runCount)
	}
	if !t.runStart.IsZero() {
		o.CurRunSecs = now.Sub(t.runStart).Seconds()
	}
	if !t.started {
		o.Rates = []RateSet{{Window: "5 min"}, {Window: "1 h"}, {Window: "session"}}
		return o
	}
	o.SessionSecs = now.Sub(t.startTime).Seconds()
	o.Rates = []RateSet{
		t.rateSet("5 min", now, 5*time.Minute),
		t.rateSet("1 h", now, time.Hour),
		t.rateSet("session", now, 0),
	}
	if t.cfg.ExpForLevel != nil && t.level > 0 {
		if cur, ok := t.cfg.ExpForLevel(t.level); ok {
			o.ExpThisLevel = cur
		}
		if next, ok := t.cfg.ExpForLevel(t.level + 1); ok {
			o.ExpNext = next
			rate := o.Rates[0].XP
			if rate <= 0 {
				rate = o.Rates[1].XP
			}
			if rate <= 0 {
				rate = o.Rates[2].XP
			}
			if rate > 0 && next > t.exp {
				o.ETASecs = float64(next-t.exp) / rate * 3600
			}
		}
	}
	return o
}

// Events returns up to n newest events whose kind is in kinds (all if empty).
func (t *Tracker) Events(n int, kinds ...string) []Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	all := t.events.Slice()
	var out []Event
	for i := len(all) - 1; i >= 0 && len(out) < n; i-- {
		if len(kinds) == 0 || contains(kinds, all[i].Kind) {
			out = append(out, all[i])
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func (t *Tracker) Drops(n int) []Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.dropLog.Last(n)
}

// Segments returns the activity timeline, oldest first, with the open segment last.
func (t *Tracker) Segments(n int) []Segment {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.segs.Last(n)
}

// Areas returns per-area statistics sorted by time spent, descending.
func (t *Tracker) Areas() []AreaStat {
	t.mu.Lock()
	defer t.mu.Unlock()
	now := t.clock.Now()
	out := make([]AreaStat, 0, len(t.areas))
	for _, a := range t.areas {
		c := *a
		if c.Area == t.area && t.hasLast {
			c.Time += now.Sub(t.areaAt)
		}
		c.Secs = c.Time.Seconds()
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time > out[j].Time })
	return out
}

type Point struct {
	T     time.Time `json:"t"`
	XP    float64   `json:"xp"`    // per hour, trailing window
	Kills float64   `json:"kills"` // per hour
	Gold  float64   `json:"gold"`
}

// Series returns per-minute points where each value is the per-hour rate over
// the trailing `smooth` minutes (clamped at the start of data).
func (t *Tracker) Series(smooth int) []Point {
	t.mu.Lock()
	defer t.mu.Unlock()
	bs := t.buckets.Slice()
	out := make([]Point, len(bs))
	for i, b := range bs {
		lo := i - smooth + 1
		if lo < 0 {
			lo = 0
		}
		var xp, k, g float64
		for _, x := range bs[lo : i+1] {
			xp += x.XP
			k += x.Kills
			g += x.Gold
		}
		mins := float64(i - lo + 1)
		out[i] = Point{T: b.T, XP: xp / mins * 60, Kills: k / mins * 60, Gold: g / mins * 60}
	}
	return out
}

// Frame returns the last observed frame.
func (t *Tracker) Frame() (Frame, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.last, t.hasLast
}
