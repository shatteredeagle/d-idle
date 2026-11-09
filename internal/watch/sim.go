package watch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
)

// SimClock is the virtual clock of the simulated runner (decide.Clock, safe for concurrent use).
type SimClock struct {
	mu sync.Mutex
	t  time.Duration
}

func (c *SimClock) Now() time.Duration { c.mu.Lock(); defer c.mu.Unlock(); return c.t }
func (c *SimClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.t += d
	c.mu.Unlock()
}

// SimBuild is the data of one simulated hero. Nothing about a class is in the code of the runner: the two builds only
// differ here (build-flexibility rule, AGENTS.md).
type SimBuild struct {
	Class    d2.Class
	Hardcore bool // hardcore makes the retreat point risky: N=5 and a confirm before the rule is learned
	Life     float64
	Danger   string
	Areas    []string
	Level    int // clvl shown on the map
	// TowerFloor is the clvl below which the sim's guard closes the Forgotten Tower (data, like the bot's tuning).
	TowerFloor int
	// MaxLife and MaxMana are the pools the sim reports for the command centre (Life is a percent of MaxLife).
	MaxLife, MaxMana int
}

// SimBuilds are the two builds of the scenarios: a softcore Sorceress and a hardcore Paladin.
var SimBuilds = map[string]SimBuild{
	"sorceress": {Class: d2.Sorceress, Life: 30, Danger: "2", Level: 6, TowerFloor: 16, MaxLife: 120, MaxMana: 140},
	"paladin":   {Class: d2.Paladin, Hardcore: true, Life: 45, Danger: "3", Level: 12, TowerFloor: 14, MaxLife: 260, MaxMana: 60},
}

// pointRetreat is the decision point the sim asks while fighting (the T0 retreat question of the shipped catalogue).
const pointRetreat = "t0.retreat.drink"

// Simulated phases of the hero.
const (
	phIdle      = "idle"
	phWalking   = "walking"
	phFighting  = "fighting"
	phDead      = "dead"
	phRecovery  = "recovering"
	phStalled   = "stalled"
	phGoingHome = "home"
)

// Sim is a fake bot: it drives the real decide store through the real ladder (Decide, Pump), reports activity, orders and
// incidents to the Hub, and runs on a virtual clock the caller steps. Used by the tests and by `d2dview -watch`.
type Sim struct {
	Hub    *Hub
	Ladder *ladder.Ladder
	Store  *decide.Store
	Clock  *SimClock
	Build  SimBuild

	mu       sync.Mutex
	fight    bool
	phase    string
	until    time.Duration
	area     string
	cur      *Order
	curEnd   time.Duration
	queue    []Order
	nextAsk  time.Duration
	nextChat time.Duration
	chatN    int
	herArea  string
	xp       int64
	ticks    int
}

const simTown = "Rogue Encampment"

// Placeholder idle lines: our own words, no game text (docs/contracts/licence-policy.md).
var simChatter = []string{
	"Nothing to do. I am sharpening my gear.",
	"Waiting for an order.",
	"It is quiet in camp.",
}

const askEvery = 30 * time.Second

// NewSim builds a simulated bot of the named build (sorceress or paladin) with an in-memory store. speed is virtual
// seconds per wall second, passed to the page for its countdown.
func NewSim(build string, speed float64) (*Sim, error) {
	b, ok := SimBuilds[build]
	if !ok {
		return nil, fmt.Errorf("watch: unknown simulated build %q", build)
	}
	cat, err := decide.Shipped()
	if err != nil {
		return nil, err
	}
	clk := &SimClock{}
	st, err := decide.New(cat, "sim-"+build, clk)
	if err != nil {
		return nil, err
	}
	s := &Sim{Clock: clk, Store: st, Build: b, phase: phIdle, nextChat: 60 * time.Second}
	b.Areas = OrderableAreas()
	s.Build = b
	s.herArea, s.xp = simTown, 0
	hub, err := New(Config{Store: st, Cat: cat, Clock: clk, Areas: b.Areas, Orders: s, WaitForSink: true, Speed: speed,
		AreaGuard: func(id int) string {
			if id >= 20 && id <= 25 && b.Level < b.TowerFloor {
				return fmt.Sprintf("no mercenary and clvl %d is below %d", b.Level, b.TowerFloor)
			}
			return ""
		}})
	if err != nil {
		return nil, err
	}
	s.Hub = hub
	s.Ladder, err = ladder.New(ladder.Config{Cat: cat, Store: st, Clock: clk, Sink: hub, Answers: hub,
		InFight: func() bool { s.mu.Lock(); defer s.mu.Unlock(); return s.fight }})
	if err != nil {
		return nil, err
	}
	hub.Activity(KindIdle, "waiting for an order")
	s.reportHeroLocked("waiting for an order")
	return s, nil
}

// Order implements OrderSink: the sim queues it and starts it when the hero is free.
func (s *Sim) Order(o Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.queue = append(s.queue, o)
	s.startNextLocked()
	return nil
}

// Cancel implements OrderSink.
func (s *Sim) Cancel(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, o := range s.queue {
		if o.ID == id {
			s.queue = append(s.queue[:i], s.queue[i+1:]...)
			return nil
		}
	}
	if s.cur != nil && s.cur.ID == id {
		s.cur = nil
		s.setLocked(phIdle, KindIdle, "waiting for an order", 0)
		return nil
	}
	return ErrNoOrder
}

func (s *Sim) setLocked(phase, kind, label string, d time.Duration) {
	s.phase = phase
	s.until = s.Clock.Now() + d
	s.Hub.Activity(kind, label)
	s.reportHeroLocked(label)
}

func (s *Sim) reportHeroLocked(label string) {
	id := 0
	if s.herArea == simTown {
		id = 1
	}
	b := s.Build
	s.Hub.SetHero(HeroInfo{Class: b.Class.String(), Level: b.Level, XP: s.xp, Area: s.herArea, AreaID: id, Activity: label,
		Name: "Simmy", Life: int(b.Life * float64(b.MaxLife) / 100), MaxLife: b.MaxLife, Mana: b.MaxMana * 3 / 4, MaxMana: b.MaxMana})
}

func (s *Sim) busyLocked() bool {
	return s.phase == phDead || s.phase == phRecovery || s.phase == phStalled
}

func (s *Sim) startNextLocked() {
	if s.cur != nil || len(s.queue) == 0 || s.busyLocked() {
		return
	}
	o := s.queue[0]
	s.queue = s.queue[1:]
	s.cur = &o
	now := s.Clock.Now()
	s.Hub.OrderState(o.ID, OrderActive, "")
	switch o.Kind {
	case OrderGo:
		s.area = o.Area
		s.curEnd = now + time.Duration(o.Minutes)*time.Minute
		s.setLocked(phWalking, KindProgress, "walking to "+o.Area, 15*time.Second)
	case OrderHere:
		if s.area == "" {
			s.area = s.Build.Areas[0]
		}
		s.curEnd = now + time.Duration(o.Minutes)*time.Minute
		s.fightLocked(now)
	default:
		s.curEnd = now + 10*time.Second
		s.setLocked(phGoingHome, KindProgress, "walking home", 10*time.Second)
	}
}

func (s *Sim) fightLocked(now time.Duration) {
	s.herArea = s.area
	s.setLocked(phFighting, KindProgress, "fighting in "+s.area, 0)
	s.nextAsk = now + askEvery
}

// SetFight marks a fight in progress: the ladder holds its questions until it is over.
func (s *Sim) SetFight(f bool) {
	s.mu.Lock()
	s.fight = f
	s.mu.Unlock()
	if !f {
		s.Ladder.Pump()
	}
}

// Kill is a scripted death: the incident names the rules in force and offers their revoke (ADR 0016 item 23).
func (s *Sim) Kill() {
	var keys []string
	for _, r := range s.Store.Rules() {
		if !r.Revoked {
			keys = append(keys, r.Key)
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	where := s.area
	if where == "" {
		where = "the field"
	}
	msg := fmt.Sprintf("The hero died in %s.", where)
	if len(keys) > 0 {
		msg += " These rules were in force; revoke one if it was a bad habit."
	}
	s.Hub.Incident(msg, keys...)
	s.setLocked(phDead, KindDeath, "dead, waiting to be revived", 20*time.Second)
}

// Stall is a scripted town stall: the hero is stuck for d.
func (s *Sim) Stall(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setLocked(phStalled, KindStall, "stuck in town", d)
}

func (s *Sim) hero() ladder.Hero {
	return ladder.Hero{Class: s.Build.Class, Hardcore: s.Build.Hardcore, Gold: 600, AreaClass: "field", Durability: 25}
}

// Step advances the virtual clock by d in slices of at most 5 s and plays the scenario.
func (s *Sim) Step(d time.Duration) {
	for d > 0 {
		dt := min(d, 5*time.Second)
		d -= dt
		s.Clock.Advance(dt)
		s.tick()
	}
}

func (s *Sim) tick() {
	s.Hub.Tick()
	s.mu.Lock()
	now := s.Clock.Now()
	switch {
	case s.phase == phDead && now >= s.until:
		s.setLocked(phRecovery, KindRecovery, "recovering the body", 20*time.Second)
	case (s.phase == phRecovery || s.phase == phStalled) && now >= s.until:
		switch {
		case s.cur != nil && s.cur.Kind != OrderHome:
			s.fightLocked(now)
		case s.cur != nil:
			s.setLocked(phGoingHome, KindProgress, "walking home", 10*time.Second)
		default:
			s.setLocked(phIdle, KindIdle, "waiting for an order", 0)
			s.startNextLocked()
		}
	case s.phase == phWalking && now >= s.until:
		s.fightLocked(now)
	}
	if s.phase == phFighting {
		s.ticks++
		if s.ticks%2 == 0 {
			s.xp += 12
			s.Hub.NoteKill()
			s.reportHeroLocked("fighting in " + s.herArea)
		}
	}
	if s.cur != nil && !s.busyLocked() && now >= s.curEnd && !s.cur.Forever {
		if s.cur.Kind == OrderHome {
			s.herArea = simTown
		}
		s.Hub.OrderState(s.cur.ID, OrderDone, "")
		s.cur = nil
		s.setLocked(phIdle, KindIdle, "waiting for an order", 0)
		s.startNextLocked()
	}
	ask := s.phase == phFighting && now >= s.nextAsk
	if ask {
		s.nextAsk = now + askEvery
	}
	chat := s.phase == phIdle && now >= s.nextChat
	if chat {
		s.nextChat = now + 120*time.Second
		s.chatN++
	}
	n, area := s.chatN, s.area
	if area == "" {
		area = "the field"
	}
	s.mu.Unlock()
	// The store and the Hub are called without the Sim lock: their callbacks never call the Sim.
	if ask {
		extra := decide.Obs{Num: map[string]float64{"life": s.Build.Life},
			Cat: map[string]string{"packdanger": s.Build.Danger, "kit": "OK", "held": "hp"},
			Sit: decide.Situation{"life": fmt.Sprint(int(s.Build.Life)), "area": area, "potions": "4", "hostiles": "3"}}
		if _, err := s.Ladder.Decide(pointRetreat, s.hero(), extra, ""); err != nil {
			s.Ladder.Fault(err)
		}
	} else {
		s.Ladder.Pump()
	}
	if chat {
		s.Hub.Say(simChatter[n%len(simChatter)])
	}
}

// Run steps the sim in real time: every period of wall time advances speed times as much virtual time, until ctx ends.
func (s *Sim) Run(ctx context.Context, speed float64, period time.Duration) {
	t := time.NewTicker(period)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.Step(time.Duration(float64(period) * speed))
		}
	}
}
