// Package sim is a deterministic simulated goals.Runner for demos and tests. It plays no game: it invents
// kills, drops, deaths and level-ups from a seeded random source, one virtual second per step.
// Numbers are illustrative, not D2 data.
package sim

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"d2lod-bot/internal/goals"
)

type Options struct {
	Seed  int64
	Name  string // default "Testy"
	Class string // default "Sorceress"
	Level int    // starting level, default 1
	// Clock stamps Status.Updated. Default: a virtual clock that advances with Advance.
	Clock func() time.Time
	// FailStart, if set, is consulted by Start; a non-nil error makes Start fail.
	FailStart func(goals.Goal) error
	// DeathRate multiplies the chance of dying when low on life (default 1).
	DeathRate float64
}

type phase int

const (
	phWalk phase = iota
	phFight
	phTown
	phDead
	phRest
)

const townName = "Rogue Encampment"

type Runner struct {
	opt Options
	rng *rand.Rand
	ch  chan goals.Event

	mu       sync.Mutex
	state    goals.RunnerState
	goal     *goals.Goal
	ch1      goals.Character
	xp       int64
	phase    phase
	left     int // seconds left in a timed phase
	sinceTwn int
	trash    int // trash kills since the last boss
	virtual  time.Time
	rem      time.Duration
	settings goals.Settings
	activity string
	bossName string
	areaGoal string
	belt     int // potions left
}

func New(opt Options) *Runner {
	if opt.Name == "" {
		opt.Name = "Testy"
	}
	if opt.Class == "" {
		opt.Class = "Sorceress"
	}
	if opt.Level < 1 {
		opt.Level = 1
	}
	if opt.DeathRate == 0 {
		opt.DeathRate = 1
	}
	r := &Runner{
		opt: opt, rng: rand.New(rand.NewSource(opt.Seed)), ch: make(chan goals.Event, 4096),
		state: goals.RunnerIdle, virtual: time.Date(2026, 1, 1, 8, 0, 0, 0, time.UTC),
		settings: goals.DefaultSettings(), phase: phTown, activity: "Standing in town", belt: 4,
	}
	r.ch1 = goals.Character{Name: opt.Name, Class: opt.Class, Level: opt.Level, Area: townName, AreaID: 1, Difficulty: "normal",
		Position: goals.Point{X: 5100, Y: 5040}}
	r.recalc(true)
	return r
}

func (r *Runner) Configure(s goals.Settings) { r.mu.Lock(); r.settings = s; r.mu.Unlock() }

func (r *Runner) Events() <-chan goals.Event { return r.ch }

func (r *Runner) Start(g goals.Goal) error {
	if r.opt.FailStart != nil {
		if err := r.opt.FailStart(g); err != nil {
			return err
		}
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.goal = &g
	r.state = goals.RunnerRunning
	r.trash = 0
	r.bossName, r.areaGoal = "", ""
	if g.Params.Difficulty != "" {
		r.ch1.Difficulty = g.Params.Difficulty
	}
	if g.Type == goals.Idle {
		r.enter(townName, 1)
		r.phase, r.activity = phRest, "Resting in town"
		return nil
	}
	r.phase, r.left, r.activity = phWalk, 3, "Walking to "+r.targetArea()
	return nil
}

func (r *Runner) Pause() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == goals.RunnerRunning {
		r.state = goals.RunnerPaused
	}
	return nil
}

func (r *Runner) Resume() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.state == goals.RunnerPaused {
		r.state = goals.RunnerRunning
	}
	return nil
}

func (r *Runner) Stop() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.goal == nil && r.state == goals.RunnerIdle {
		return nil
	}
	r.goal, r.state = nil, goals.RunnerIdle
	r.enter(townName, 1)
	r.phase, r.activity = phTown, "Standing in town"
	r.ch1.Life, r.ch1.Mana = r.ch1.MaxLife, r.ch1.MaxMana
	return nil
}

func (r *Runner) Status() goals.Status {
	r.mu.Lock()
	defer r.mu.Unlock()
	st := goals.Status{State: r.state, Activity: r.activity, Character: r.ch1}
	st.Character.XP = r.xp
	if r.goal != nil {
		st.GoalID = r.goal.ID
	}
	st.Updated = r.virtual
	if r.opt.Clock != nil {
		st.Updated = r.opt.Clock()
	}
	return st
}

// Run advances the simulation in real time until ctx is done. speed 1 means one virtual second per second.
func (r *Runner) Run(ctx context.Context, speed float64) {
	const every = 100 * time.Millisecond
	t := time.NewTicker(every)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			r.Advance(time.Duration(float64(every) * speed))
		}
	}
}

// Advance runs whole virtual seconds; the remainder is carried to the next call.
func (r *Runner) Advance(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rem += d
	for r.rem >= time.Second {
		r.rem -= time.Second
		r.virtual = r.virtual.Add(time.Second)
		r.step()
	}
}

func (r *Runner) emit(ev goals.Event) {
	for {
		select {
		case r.ch <- ev:
			return
		default:
			select { // full: drop the oldest
			case <-r.ch:
			default:
			}
		}
	}
}

func need(level int) int64 { return int64(60 * math.Pow(float64(level), 2.1)) }

func (r *Runner) recalc(full bool) {
	c := &r.ch1
	c.MaxLife = 40 + 18*c.Level
	c.MaxMana = 30 + 14*c.Level
	c.XPNext = need(c.Level)
	if full {
		c.Life, c.Mana = c.MaxLife, c.MaxMana
	}
}

func (r *Runner) enter(area string, id int) {
	if r.ch1.Area == area {
		return
	}
	r.ch1.Area, r.ch1.AreaID = area, id
	r.emit(goals.Event{Type: goals.EvAreaChange, Area: area, AreaID: id})
}

var levelLadder = []struct {
	upTo int
	area string
	id   int
}{
	{4, "Blood Moor", 2}, {8, "Cold Plains", 3}, {12, "Stony Field", 4}, {16, "Dark Wood", 5},
	{20, "Black Marsh", 6}, {25, "Burial Grounds", 17}, {30, "Cave Level 2", 13}, {35, "Spider Forest", 0},
	{45, "Flayer Jungle", 0}, {60, "Durance of Hate Level 3", 102}, {80, "Chaos Sanctuary", 108}, {100, "Worldstone Chamber", 132},
}

// targetArea resolves the goal to a concrete area name (and a boss for boss goals).
func (r *Runner) targetArea() string {
	g := r.goal
	if g == nil {
		return townName
	}
	if r.areaGoal != "" && g.Type != goals.ReachLevel {
		return r.areaGoal
	}
	switch g.Type {
	case goals.ReachLevel:
		for _, l := range levelLadder {
			if r.ch1.Level < l.upTo {
				return l.area
			}
		}
		return "Worldstone Chamber"
	case goals.KillBoss:
		r.bossName = g.Params.Boss
		r.areaGoal = bossArea(g.Params.Boss, "Pit Level 1")
	case goals.CollectItem:
		r.areaGoal = "Pit Level 1"
	case goals.FarmArea:
		r.areaGoal = g.Params.Area
		if k, ok := strings.CutPrefix(g.Params.Area, "run:"); ok {
			for _, run := range goals.Catalog().Runs {
				if run.Key == g.Params.Area && run.Boss != "" {
					r.bossName = run.Boss
					r.areaGoal = bossArea(run.Boss, k)
				}
			}
		}
		if r.areaGoal == "" {
			r.areaGoal = fmt.Sprintf("Area %d", g.Params.AreaID)
		}
	}
	return r.areaGoal
}

func bossArea(boss, def string) string {
	for _, b := range goals.Catalog().Bosses {
		if strings.EqualFold(b.Name, boss) {
			return b.Area
		}
	}
	return def
}

var monsters = map[string][]string{
	"Blood Moor":     {"Fallen", "Zombie", "Quill Rat", "Corrupt Rogue"},
	"Cold Plains":    {"Fallen", "Dark Hunter", "Skeleton", "Carver"},
	"Stony Field":    {"Rock Worm", "Fallen Shaman", "Wild Dog", "Zombie"},
	"Burial Grounds": {"Zombie", "Skeleton Archer", "Ghoul", "Blood Raven's Pack"},
	"Pit Level 1":    {"Fallen", "Dark Hunter", "Skeleton", "Unraveler"},
}
var generic = []string{"Fallen", "Skeleton", "Zombie", "Dark Hunter", "Carver"}

func (r *Runner) pickMonster() string {
	m := monsters[r.ch1.Area]
	if m == nil {
		m = generic
	}
	return m[r.rng.Intn(len(m))]
}

func (r *Runner) step() {
	if r.state != goals.RunnerRunning || r.goal == nil {
		return
	}
	c := &r.ch1
	// small drift so the position on the card moves
	c.Position.X += r.rng.Intn(9) - 4
	c.Position.Y += r.rng.Intn(9) - 4
	if r.phase != phFight && r.phase != phDead {
		c.Life = min(c.MaxLife, c.Life+c.MaxLife/10+1)
		c.Mana = min(c.MaxMana, c.Mana+c.MaxMana/10+1)
	}
	switch r.phase {
	case phRest:
		r.activity = "Resting in town"
	case phDead:
		r.left--
		if r.left <= 0 {
			c.Life, c.Mana = c.MaxLife, c.MaxMana
			r.belt = 6
			r.enter(townName, 1)
			r.phase, r.left, r.activity = phTown, 4, "Recovering in town"
		}
	case phTown:
		r.left--
		if r.left <= 0 {
			r.phase, r.left, r.activity = phWalk, 3, "Walking to "+r.targetArea()
		}
	case phWalk:
		r.left--
		if r.left <= 0 {
			area := r.targetArea()
			id := 0
			for _, a := range goals.Catalog().Areas {
				if a.Name == area {
					id = a.ID
				}
			}
			r.enter(area, id)
			r.phase, r.activity = phFight, "Fighting in "+area
		}
	case phFight:
		r.fight()
	}
}

func (r *Runner) fight() {
	c := &r.ch1
	hurt := r.rng.Intn(c.MaxLife/6 + 1)
	if r.rng.Float64() < 0.4 {
		c.Life -= hurt
	} else {
		c.Life = min(c.MaxLife, c.Life+c.MaxLife/50)
	}
	if r.rng.Float64() < 0.004*r.opt.DeathRate { // ambush: a unique pack, sometimes lethal
		c.Life -= c.MaxLife * (40 + r.rng.Intn(80)) / 100
	}
	c.Mana = max(0, c.Mana-r.rng.Intn(c.MaxMana/15+1))
	if c.Mana < c.MaxMana/5 {
		c.Mana = c.MaxMana / 2
	}
	if r.belt > 0 && c.Life*100 < r.settings.PotionLifePct*c.MaxLife {
		r.belt--
		pot := "Healing Potion"
		if c.Level > 20 {
			pot = "Greater Healing Potion"
		}
		r.emit(goals.Event{Type: goals.EvPotion, Potion: pot})
		c.Life = min(c.MaxLife, c.Life+c.MaxLife*4/10)
	}
	if c.Life <= 0 || (c.Life*100 < c.MaxLife*8 && r.rng.Float64() < 0.2*r.opt.DeathRate) {
		c.Life = 0
		r.emit(goals.Event{Type: goals.EvDeath, Area: c.Area})
		r.phase, r.left, r.activity = phDead, 6, "Dead, waiting to respawn"
		return
	}
	r.activity = "Fighting in " + c.Area
	if r.rng.Float64() < 0.5 {
		r.kill()
	}
	if r.sinceTwn >= 60 {
		r.sinceTwn, r.belt = 0, 6
		r.enter(townName, 1)
		r.phase, r.left, r.activity = phTown, 5, "Selling and restocking in town"
		return
	}
	if r.rng.Float64() < 0.06 {
		r.phase, r.left, r.activity = phWalk, 2, "Moving to the next pack"
	}
}

func (r *Runner) kill() {
	c := &r.ch1
	r.sinceTwn++
	xp := int64(12 + 6*c.Level)
	if r.bossName != "" {
		r.trash++
		if r.trash >= 25 {
			r.trash = 0
			r.gain(xp * 25)
			r.emit(goals.Event{Type: goals.EvKill, Monster: r.bossName, Boss: true, XP: xp * 25})
			for i := 0; i < 3; i++ {
				r.maybeDrop(1)
			}
			return
		}
	}
	r.gain(xp)
	r.emit(goals.Event{Type: goals.EvKill, Monster: r.pickMonster(), XP: xp})
	rate := 0.12
	if r.goal.Type == goals.CollectItem {
		rate = 0.25
	}
	r.maybeDrop(rate)
}

func (r *Runner) gain(xp int64) {
	r.xp += xp
	for r.ch1.Level < 99 && r.xp >= r.ch1.XPNext {
		r.xp -= r.ch1.XPNext
		r.ch1.Level++
		r.recalc(true)
		r.emit(goals.Event{Type: goals.EvLevelUp, Level: r.ch1.Level})
	}
}

var runes = []string{"El", "Eld", "Tir", "Nef", "Eth", "Ith", "Tal", "Ral", "Ort", "Thul", "Amn", "Sol", "Shael", "Dol", "Hel", "Io", "Lum", "Ko", "Fal", "Lem"}
var uniques = []string{"Goldskin", "Hotspur", "Tarnhelm", "Stormshield", "Rixot's Keen", "Skin of the Vipermagi", "The Gnasher"}
var sets = []string{"Cleglaw's Tooth", "Tancred's Skull", "Angelic Mantle", "Sander's Folly"}
var bases = []string{"Cap", "Leather Boots", "Belt", "Ring", "Amulet", "Short Sword", "Buckler", "Gloves"}
var prefixes = []string{"Sturdy", "Jagged", "Fine", "Shimmering", "Cobalt", "Crimson"}
var suffixes = []string{"of Quickness", "of the Whale", "of Haste", "of Vita", "of Fortune"}

func (r *Runner) maybeDrop(rate float64) {
	if r.rng.Float64() >= rate {
		return
	}
	it := r.genItem()
	if g := r.goal; g != nil && g.Type == goals.CollectItem && r.rng.Float64() < 0.15 {
		it = r.matching(g.Params.Item)
	}
	r.emit(goals.Event{Type: goals.EvDrop, Item: &it})
}

func (r *Runner) genItem() goals.Item {
	x := r.rng.Float64()
	switch {
	case x < 0.06:
		return goals.Item{Name: runes[r.rng.Intn(len(runes))] + " Rune", Type: "rune", Quality: "normal"}
	case x < 0.10:
		return goals.Item{Name: "Chipped Amethyst", Type: "gem", Quality: "normal"}
	case x < 0.12:
		return goals.Item{Name: uniques[r.rng.Intn(len(uniques))], Type: "armor", Quality: "unique"}
	case x < 0.14:
		return goals.Item{Name: sets[r.rng.Intn(len(sets))], Type: "armor", Quality: "set"}
	case x < 0.26:
		b := bases[r.rng.Intn(len(bases))]
		return goals.Item{Name: b, Base: b, Type: strings.ToLower(b), Quality: "rare"}
	case x < 0.55:
		b := bases[r.rng.Intn(len(bases))]
		n := prefixes[r.rng.Intn(len(prefixes))] + " " + b + " " + suffixes[r.rng.Intn(len(suffixes))]
		return goals.Item{Name: n, Base: b, Type: strings.ToLower(b), Quality: "magic"}
	}
	b := bases[r.rng.Intn(len(bases))]
	return goals.Item{Name: b, Base: b, Type: strings.ToLower(b), Quality: "normal"}
}

// matching makes an item that satisfies f, so collect goals can finish in a demo.
func (r *Runner) matching(f goals.ItemFilter) goals.Item {
	it := r.genItem()
	if f.Type == "rune" {
		it = goals.Item{Name: runes[r.rng.Intn(len(runes))] + " Rune", Type: "rune", Quality: "normal"}
	}
	if f.Name != "" {
		it.Name = f.Name
	}
	if f.Type != "" {
		it.Type = f.Type
	}
	if f.Quality != "" && f.Quality != "any" {
		it.Quality = f.Quality
	}
	return it
}
