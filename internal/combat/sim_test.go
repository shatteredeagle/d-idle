package combat

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"math"
	"strconv"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/input"
)

// sim is a tiny world: monsters walk toward the player and hit when adjacent, casts damage the monster nearest to
// the click, potions heal. Time only moves when the fighter sleeps or walks, so the tests are instant.
type simMon struct {
	id       uint32
	class    int
	x, y     float64
	hp, maxH float64
	dmg      float64 // per second while adjacent
	speed    float64 // subtiles per second
	immune   bool    // takes no damage from casts
	mode     int
	flags    uint8
}

type sim struct {
	mu          sync.Mutex
	t           time.Time
	px, py      float64
	life, maxL  float64
	mana, maxM  float64
	exp         uint32
	mons        []*simMon
	belt        []d2.Unit // potion units
	castDmg     float64
	castCost    float64
	manaRegen   float64
	healAmount  float64
	walkSpeed   float64
	casts       int
	melee       int
	meleeDmg    float64
	drinks      map[string]int
	moves       []input.Point
	left        bool
	portalOpen  bool
	states      []int // player state ids
	noTP        bool  // no town portal scroll in the pack
	events      []goals.Event
	ranged      bool
	skills      []d2.Skill
	keysPressed []string
	castKeys    []string // keys of CastAt, in order
	dead        bool
	right       int
	left_       int
	onCast      func()
	noPath      func(p input.Point) bool // tests: points the mover cannot reach
}

func newSim() *sim {
	return &sim{
		t: time.Unix(1_000_000, 0), px: 100, py: 100, life: 100, maxL: 100, mana: 40, maxM: 40,
		castDmg: 12, meleeDmg: 6, castCost: 3, manaRegen: 1.5, healAmount: 40, walkSpeed: 10, drinks: map[string]int{},
		skills: []d2.Skill{{ID: 36, Level: 0, Bonus: 1}},
	}
}

func (s *sim) Now() time.Time { s.mu.Lock(); defer s.mu.Unlock(); return s.t }

func (s *sim) Sleep(d time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance(d)
}

// advance runs the world for d in 50 ms steps.
func (s *sim) advance(d time.Duration) {
	for d > 0 && !s.dead {
		step := 50 * time.Millisecond
		if d < step {
			step = d
		}
		d -= step
		s.t = s.t.Add(step)
		dt := step.Seconds()
		s.mana = math.Min(s.maxM, s.mana+s.manaRegen*dt)
		for _, m := range s.mons {
			if m.mode != d2.MonModeStand {
				continue
			}
			dx, dy := s.px-m.x, s.py-m.y
			dist := math.Hypot(dx, dy)
			if dist > 2.5 {
				m.x += dx / dist * m.speed * dt
				m.y += dy / dist * m.speed * dt
			} else {
				s.life -= m.dmg * dt
			}
		}
		if s.life <= 0 {
			s.life, s.dead = 0, true
		}
	}
}

func (s *sim) snapshot() (*d2.Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	mode := 1
	if s.dead {
		mode = 0
	}
	snap := &d2.Snapshot{
		State: d2.StateInGame,
		Player: &d2.Player{ID: 1, X: int(math.Round(s.px)), Y: int(math.Round(s.py)), Area: 2, Level: 1, Mode: mode,
			Life: int(math.Round(s.life)), MaxLife: int(s.maxL), Mana: int(s.mana), MaxMana: int(s.maxM), Exp: s.exp, Skills: s.skills,
			RightSkill: 36, LeftSkill: s.left_},
		Inventory: &d2.Inventory{Items: append([]d2.Unit(nil), s.belt...)},
		Viewport:  d2.Point{X: int(16*(s.px-s.py) - 400), Y: int(8*(s.px+s.py) - 284)},
	}
	for _, st := range s.states {
		snap.Player.States[st>>5] |= 1 << (uint(st) & 31)
	}
	if !s.noTP {
		snap.Inventory.Items = append(snap.Inventory.Items, d2.Unit{Type: d2.TypeItem, Class: itTSC, ID: 77,
			Item: &d2.Item{Where: d2.ContInventory}})
	}
	for _, m := range s.mons {
		u := mon(m.id, m.class, int(math.Round(m.x)), int(math.Round(m.y)))
		u.Mode = m.mode
		u.Monster.LifePct = int(math.Round(m.hp / m.maxH * 100))
		u.Monster.Flags = m.flags
		snap.Monsters = append(snap.Monsters, u)
	}
	return snap, nil
}

// ---- input.Controller ----
func (s *sim) MoveMouse(x, y int) error { return nil }
func (s *sim) Click(b input.Button, x, y int, m ...input.Mod) error {
	if b != input.Left {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance(400 * time.Millisecond)
	s.melee++
	for _, m := range s.mons {
		if m.mode == d2.MonModeStand && !m.immune && math.Hypot(m.x-s.px, m.y-s.py) < 7 {
			m.hp -= s.meleeDmg
			if m.hp <= 0 {
				m.mode = d2.MonModeDead
				s.exp += 10
			}
			break
		}
	}
	return nil
}
func (s *sim) Hold(string) error                        { return nil }
func (s *sim) Release(string) error                     { return nil }
func (s *sim) Pos() (input.Point, error)                { return input.Point{X: int(s.px), Y: int(s.py)}, nil }
func (s *sim) ClickWorld(input.Point) error             { return nil }
func (s *sim) MoveTo(input.Point, input.MoveOpts) error { return nil }
func (s *sim) SelectSkill(string) error                 { return nil }
func (s *sim) Interact(input.Target) error              { return nil }
func (s *sim) CloseAll() error                          { return nil }

func (s *sim) Key(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keysPressed = append(s.keysPressed, name)
	col, err := strconv.Atoi(name)
	if err != nil || col < 1 || col > 4 {
		return nil
	}
	for i, u := range s.belt {
		if u.Item.GX != col-1 {
			continue
		}
		switch u.Class {
		case itHP1, itHP3:
			s.life = math.Min(s.maxL, s.life+s.healAmount)
			s.drinks["healing"]++
		case itMP1:
			s.mana = s.maxM
			s.drinks["mana"]++
		case itRV:
			s.life, s.mana = s.maxL, s.maxM
			s.drinks["rejuv"]++
		}
		s.belt = append(s.belt[:i], s.belt[i+1:]...)
		break
	}
	return nil
}

func (s *sim) CastAt(key string, w input.Point) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.advance(300 * time.Millisecond) // cast animation
	s.castKeys = append(s.castKeys, key)
	if s.mana < s.castCost || s.dead {
		return nil
	}
	s.mana -= s.castCost
	s.casts++
	if s.onCast != nil {
		s.onCast()
	}
	var best *simMon
	bd := 6.0
	for _, m := range s.mons {
		if m.mode != d2.MonModeStand {
			continue
		}
		if d := math.Hypot(m.x-float64(w.X), m.y-float64(w.Y)); d < bd {
			best, bd = m, d
		}
	}
	if best != nil && !best.immune {
		best.hp -= s.castDmg
		if best.hp <= 0 {
			best.mode = d2.MonModeDead
			s.exp += 10
		}
	}
	return nil
}

// ---- Mover ----
func (s *sim) GoTo(ctx context.Context, p input.Point) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.moves = append(s.moves, p)
	d := math.Hypot(float64(p.X)-s.px, float64(p.Y)-s.py)
	s.advance(time.Duration(d / s.walkSpeed * float64(time.Second)))
	if !s.dead {
		s.px, s.py = float64(p.X), float64(p.Y)
	}
	return nil
}
func (s *sim) GoToUnit(ctx context.Context, id uint32) error { return nil }
func (s *sim) PathTo(p input.Point) ([]input.Point, error) {
	if s.noPath != nil && s.noPath(p) {
		return nil, errors.New("no path")
	}
	return []input.Point{p}, nil
}

// harness wires a Fighter to a sim.
type harness struct {
	s      *sim
	f      *Fighter
	events []goals.Event
	cancel context.CancelFunc
}

func newHarness(t testing.TB, mods ...func(*env.Tuning)) *harness {
	if t != nil {
		t.Helper()
	}
	s := newSim()
	h := &harness{s: s}
	tune := env.DefaultTuning()
	for _, m := range mods {
		m(&tune)
	}
	e := &env.Env{
		ID: "test", Log: slog.New(slog.NewTextHandler(io.Discard, nil)), In: s, Tune: &tune, Clock: s,
		Emit: func(ev goals.Event) { h.events = append(h.events, ev) },
	}
	f := New(e, s)
	f.SetCatalog(testCatalog())
	f.snapFn = s.snapshot
	f.SetHotkeys(map[int]string{36: "f8", 38: "f4", 42: "f1"})
	f.Leave = func(ctx context.Context) error { s.left = true; return nil }
	f.Portal = func(ctx context.Context) error { s.portalOpen = true; return nil }
	h.f = f
	return h
}

func (s *sim) addMon(id uint32, class int, x, y, hp, dmg, speed float64) *simMon {
	m := &simMon{id: id, class: class, x: x, y: y, hp: hp, maxH: hp, dmg: dmg, speed: speed, mode: d2.MonModeStand}
	s.mons = append(s.mons, m)
	return m
}

func (s *sim) stock(class, col, n int) {
	for i := 0; i < n; i++ {
		u := potion(class, col)
		u.ID = uint32(1000 + len(s.belt))
		s.belt = append(s.belt, u)
	}
}

func (h *harness) kinds(typ goals.EventType) []goals.Event {
	var out []goals.Event
	for _, e := range h.events {
		if e.Type == typ {
			out = append(out, e)
		}
	}
	return out
}

func TestClearKillsAPack(t *testing.T) {
	h := newHarness(t)
	h.s.stock(itHP1, 0, 2)
	for i := 0; i < 3; i++ {
		h.s.addMon(uint32(10+i), clsFallen, 130+float64(i)*3, 105, 30, 5, 5)
	}
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kills != 3 || h.s.dead {
		t.Fatalf("result %v dead=%v", res, h.s.dead)
	}
	if res.XP != 30 {
		t.Fatalf("xp attribution = %d, want 30", res.XP)
	}
	if k := h.kinds(goals.EvKill); len(k) != 3 || k[0].Monster != "fallen1" || k[0].XP != 10 {
		t.Fatalf("kill events: %+v", k)
	}
	if h.s.casts < 6 {
		t.Fatalf("30 hp at 12 per cast needs >= 3 casts each, got %d", h.s.casts)
	}
}

func TestClearEmptyReturnsAtOnce(t *testing.T) {
	h := newHarness(t)
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 0 || res.TimedOut {
		t.Fatalf("%v %v", res, err)
	}
}

func TestClearRadiusLeavesFarMonsters(t *testing.T) {
	h := newHarness(t)
	h.s.addMon(10, clsFallen, 100+90, 100, 30, 5, 0) // far and asleep
	res, err := h.f.Clear(context.Background(), ClearOpts{Radius: 30})
	if err != nil || res.Kills != 0 || h.s.casts != 0 {
		t.Fatalf("far monster must be left alone: %v %v casts=%d", res, err, h.s.casts)
	}
}

func TestPotionsAndEvents(t *testing.T) {
	h := newHarness(t)
	h.s.stock(itHP1, 0, 3)
	h.s.stock(itMP1, 2, 2)
	h.s.mana = 4
	h.s.addMon(10, clsFallen, 120, 100, 90, 22, 6)
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil {
		t.Fatalf("%v %v", res, err)
	}
	if h.s.dead || res.Kills != 1 {
		t.Fatalf("expected a survived kill: %v dead=%v", res, h.s.dead)
	}
	if h.s.drinks["healing"] == 0 && h.s.drinks["mana"] == 0 {
		t.Fatal("no potion was drunk")
	}
	if len(h.kinds(goals.EvPotion)) != res.Potions || res.Potions == 0 {
		t.Fatalf("potion events %d vs %d", len(h.kinds(goals.EvPotion)), res.Potions)
	}
}

func TestChickenWhenNothingHelps(t *testing.T) {
	h := newHarness(t)
	h.s.life = 50
	h.s.addMon(10, clsFallen, 104, 100, 500, 5, 6)
	_, err := h.f.Clear(context.Background(), ClearOpts{})
	// no potions, a portal scroll exists: the first answer is the town portal, as soon as life < drink%
	if !errors.Is(err, ErrTownPortal) || !h.s.portalOpen {
		t.Fatalf("want town portal, got %v portal=%v", err, h.s.portalOpen)
	}
	h2 := newHarness(t)
	h2.s.life = 24
	h2.s.addMon(10, clsFallen, 104, 100, 500, 1, 6)
	_, err = h2.f.Clear(context.Background(), ClearOpts{})
	if !errors.Is(err, ErrChicken) || !h2.s.left {
		t.Fatalf("want chicken, got %v left=%v", err, h2.s.left)
	}
}

func TestRejuvAtChickenLine(t *testing.T) {
	h := newHarness(t)
	h.s.life = 22
	h.s.stock(itRV, 3, 1)
	h.s.addMon(10, clsFallen, 112, 100, 24, 1, 4)
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || h.s.drinks["rejuv"] != 1 || res.Kills != 1 {
		t.Fatalf("%v %v drinks=%v", res, err, h.s.drinks)
	}
}

func TestImmuneIsSkipped(t *testing.T) {
	h := newHarness(t)
	h.s.addMon(10, clsAllImmun, 115, 100, 50, 0, 0)
	h.s.addMon(11, clsFallen, 120, 100, 24, 0, 0)
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Kills != 1 || res.Skipped != 1 || res.TimedOut {
		t.Fatalf("the immune monster is skipped, the other killed: %v", res)
	}
}

func TestNoProgressGivesUp(t *testing.T) {
	h := newHarness(t)
	m := h.s.addMon(10, clsFallen, 118, 100, 50, 0, 0)
	m.immune = true // monstats says nothing, but nothing we cast works
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Skipped != 1 || res.TimedOut {
		t.Fatalf("no-progress watchdog: %v %v", res, err)
	}
}

func TestKillBoss(t *testing.T) {
	h := newHarness(t)
	h.s.castDmg = 30
	h.s.mana, h.s.maxM = 200, 200
	h.s.addMon(10, clsBoss, 125, 100, 300, 3, 4)
	if err := h.f.Kill(context.Background(), 10); err != nil {
		t.Fatal(err)
	}
	k := h.kinds(goals.EvKill)
	if len(k) != 1 || !k[0].Boss {
		t.Fatalf("boss kill event: %+v", k)
	}
	if err := h.f.Kill(context.Background(), 999); !errors.Is(err, ErrTargetGone) {
		t.Fatalf("missing target: %v", err)
	}
}

func TestDeadIsReported(t *testing.T) {
	h := newHarness(t, func(tu *env.Tuning) { tu.Combat.ChickenPct = 15 })
	h.s.stock(itHP1, 0, 1)
	h.s.life = 30
	h.s.addMon(10, clsFallen, 104, 100, 500, 500, 6)
	_, err := h.f.Clear(context.Background(), ClearOpts{})
	if !errors.Is(err, ErrDead) && !errors.Is(err, ErrChicken) {
		t.Fatalf("one-shot kill: %v", err)
	}
}

func TestOutOfManaIsBounded(t *testing.T) {
	h := newHarness(t)
	h.s.mana, h.s.manaRegen, h.s.meleeDmg = 0, 0, 0 // a staff that does nothing
	h.s.addMon(10, clsFallen, 115, 100, 50, 0, 0)
	res, err := h.f.Clear(context.Background(), ClearOpts{Timeout: 30 * time.Second})
	if err != nil || !res.TimedOut || res.Kills != 0 {
		t.Fatalf("must give up at the deadline: %v %v", res, err)
	}
}

func TestContextCancel(t *testing.T) {
	h := newHarness(t)
	h.s.mana, h.s.manaRegen = 0, 0
	h.s.addMon(10, clsFallen, 115, 100, 50, 0, 0)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := h.f.Clear(ctx, ClearOpts{}); !errors.Is(err, context.Canceled) {
		t.Fatalf("%v", err)
	}
}

func TestKitingBacksOff(t *testing.T) {
	h := newHarness(t)
	h.s.life = 60
	h.s.stock(itHP1, 0, 4)
	h.s.addMon(10, clsFallen, 106, 100, 40, 3, 3) // slow melee monster right next to us
	res, err := h.f.Clear(context.Background(), ClearOpts{})
	if err != nil || res.Kills != 1 {
		t.Fatalf("%v %v", res, err)
	}
	if len(h.s.moves) == 0 {
		t.Fatal("expected a retreat step")
	}
	first := h.s.moves[0]
	if first.X <= 100 && first.X >= 106 || math.Hypot(float64(first.X-106), float64(first.Y-100)) <= 6 {
		t.Fatalf("first move should go away from the monster, got %v", first)
	}
}

func TestStaticFieldOpensOnElites(t *testing.T) {
	h := newHarness(t)
	h.f.SetHotkeys(map[int]string{36: "f8", 42: "f1"})
	h.s.skills = []d2.Skill{{ID: 36, Bonus: 1}, {ID: 42, Level: 20}}
	h.s.mana, h.s.maxM = 200, 200
	m := h.s.addMon(10, clsFallen, 112, 100, 400, 0, 0)
	m.flags = 1 << 2 // champion
	h.f.tune()
	cast := 0
	h.f.e.In = &keyCounter{Controller: h.s, keys: map[string]int{}, n: &cast}
	h.f.Clear(context.Background(), ClearOpts{Timeout: 20 * time.Second})
	kc := h.f.e.In.(*keyCounter)
	if kc.keys["f1"] != env.DefaultCombatTuning().StaticCasts {
		t.Fatalf("static field casts = %d, want %d (%v)", kc.keys["f1"], env.DefaultCombatTuning().StaticCasts, kc.keys)
	}
}

type keyCounter struct {
	input.Controller
	keys map[string]int
	n    *int
}

func (k *keyCounter) CastAt(key string, w input.Point) error {
	k.keys[key]++
	return k.Controller.CastAt(key, w)
}

// ---- handicap: a lowered knob makes the bot slower, not suicidal ----

type outcome struct {
	elapsed time.Duration
	dead    bool
	err     error
	kills   int
}

func runScenario(seed int64, mod func(*env.Tuning)) outcome {
	h := newHarness(nil, mod)
	h.f.SetSeed(seed)
	h.s.stock(itHP1, 0, 3)
	h.s.stock(itMP1, 2, 1)
	h.s.addMon(10, clsShaman, 150, 100, 30, 4, 3)
	for i := 0; i < 4; i++ {
		h.s.addMon(uint32(11+i), clsFallen, 128+float64(i)*5, 98+float64(i)*3, 30, 4, 3)
	}
	res, err := h.f.Clear(context.Background(), ClearOpts{Radius: 80, Timeout: 10 * time.Minute})
	return outcome{elapsed: res.Elapsed, dead: h.s.dead, err: err, kills: res.Kills}
}

func TestHandicapSlowerNotSuicidal(t *testing.T) {
	type knob struct {
		name string
		mods []func(*env.Tuning)
	}
	knobs := []knob{
		{"targetQuality", []func(*env.Tuning){
			func(tu *env.Tuning) { tu.Combat.TargetQuality = 100 },
			func(tu *env.Tuning) { tu.Combat.TargetQuality = 60 },
			func(tu *env.Tuning) { tu.Combat.TargetQuality = 20 },
		}},
		{"reactionMs", []func(*env.Tuning){
			func(tu *env.Tuning) { tu.Combat.ReactionMs = 40 },
			func(tu *env.Tuning) { tu.Combat.ReactionMs = 300 },
			func(tu *env.Tuning) { tu.Combat.ReactionMs = 700 },
		}},
		{"pollMs", []func(*env.Tuning){
			func(tu *env.Tuning) { tu.Combat.PollMs = 30 },
			func(tu *env.Tuning) { tu.Combat.PollMs = 150 },
			func(tu *env.Tuning) { tu.Combat.PollMs = 400 },
		}},
		{"castGapMs", []func(*env.Tuning){
			func(tu *env.Tuning) { tu.Combat.CastGapMs = 250 },
			func(tu *env.Tuning) { tu.Combat.CastGapMs = 600 },
			func(tu *env.Tuning) { tu.Combat.CastGapMs = 1200 },
		}},
		{"potionGapMs", []func(*env.Tuning){
			func(tu *env.Tuning) { tu.Combat.PotionGapMs = 900 },
			func(tu *env.Tuning) { tu.Combat.PotionGapMs = 3000 },
		}},
		{"drinkLifePct", []func(*env.Tuning){
			func(tu *env.Tuning) { tu.Combat.DrinkLifePct = 80 },
			func(tu *env.Tuning) { tu.Combat.DrinkLifePct = 35 },
		}},
	}
	for _, k := range knobs {
		var mean []time.Duration
		for _, mod := range k.mods {
			var sum time.Duration
			const seeds = 12
			for seed := int64(1); seed <= seeds; seed++ {
				o := runScenario(seed, mod)
				if o.dead || (o.err != nil && !errors.Is(o.err, ErrChicken) && !errors.Is(o.err, ErrTownPortal)) {
					t.Errorf("%s: seed %d died or failed: dead=%v err=%v", k.name, seed, o.dead, o.err)
				}
				if o.err == nil && o.kills != 5 {
					t.Errorf("%s: seed %d killed %d of 5", k.name, seed, o.kills)
				}
				sum += o.elapsed
			}
			mean = append(mean, sum/seeds)
		}
		t.Logf("%s: mean clear time by setting %v", k.name, mean)
		if k.name == "drinkLifePct" || k.name == "potionGapMs" || k.name == "reactionMs" {
			continue // survival knobs: only the "not suicidal" half applies
		}
		if mean[len(mean)-1] < mean[0]*105/100 {
			t.Errorf("%s: lowering the knob should make the clear at least 5%% slower: %v", k.name, mean)
		}
	}
}

// Each knob at its worst allowed value, alone, and then every knob at half way, against a pack that hurts: the bot
// may chicken or portal out, but it must not die. All knobs at their worst at once is NOT promised (the plan's
// handicap profiles must not stack every knob at its limit; documented in docs/combat.md).
func TestWorstAllowedSettingsDoNotDie(t *testing.T) {
	mods := map[string]func(c *env.CombatTuning){
		"quality": func(c *env.CombatTuning) { c.TargetQuality = 20 },
		"react":   func(c *env.CombatTuning) { c.ReactionMs = 700 },
		"poll":    func(c *env.CombatTuning) { c.PollMs = 400 },
		"castgap": func(c *env.CombatTuning) { c.CastGapMs = 1200 },
		"potgap":  func(c *env.CombatTuning) { c.PotionGapMs, c.RejuvGapMs = 3000, 1500 },
		"thr":     func(c *env.CombatTuning) { c.DrinkLifePct, c.DrinkRejuvPct, c.ChickenPct = 35, 20, 15 },
		"nokite":  func(c *env.CombatTuning) { c.KiteEnabled = 0 },
		"half": func(c *env.CombatTuning) {
			c.TargetQuality, c.ReactionMs, c.PollMs, c.CastGapMs = 60, 350, 200, 700
			c.PotionGapMs, c.RejuvGapMs, c.DrinkLifePct, c.ChickenPct = 2000, 900, 45, 20
		},
	}
	for name, m := range mods {
		for seed := int64(1); seed <= 20; seed++ {
			h := newHarness(t, func(tu *env.Tuning) { m(&tu.Combat) })
			h.f.SetSeed(seed)
			h.s.stock(itHP1, 0, 4)
			h.s.stock(itMP1, 2, 2)
			h.s.stock(itRV, 3, 1)
			for i := 0; i < 5; i++ {
				h.s.addMon(uint32(10+i), clsFallen, 118+float64(i)*3, 96+float64(i)*2, 36, 5, 4)
			}
			_, err := h.f.Clear(context.Background(), ClearOpts{Radius: 80, Timeout: 10 * time.Minute})
			if h.s.dead || errors.Is(err, ErrDead) {
				t.Fatalf("%s seed %d: died (err=%v)", name, seed, err)
			}
		}
	}
}

// G1 attempt 13 (V, soak log 23:25): at 52-67% life with a pack of melee monsters around, the old rule kited on every
// tick below KiteLifePct and the character never cast. A pack is fought, not fled; kiting is for one monster that hurts.
func TestPackIsNotKited(t *testing.T) {
	h := newHarness(t)
	h.s.life = 60
	h.s.mana, h.s.maxM = 200, 200
	h.s.stock(itHP1, 0, 4)
	for i := 0; i < 4; i++ {
		h.s.addMon(uint32(10+i), clsFallen, float64(104+i), float64(100+i), 40, 3, 3)
	}
	var buf bytes.Buffer
	h.f.e.Log = slog.New(slog.NewTextHandler(&buf, nil))
	h.f.Clear(context.Background(), ClearOpts{Timeout: 10 * time.Second})
	if h.s.casts == 0 {
		t.Fatal("the pack was never attacked")
	}
	if n := len(h.s.moves); n > 1 { // one kite is fine once a single monster is left
		t.Fatalf("moved %d times against a pack: %v\n%s", n, h.s.moves, buf.String())
	}
}

// Blood Raven resists every element by half, so the Static Field opener is off for her by default (StaticCastsBR 0) and
// still works on other bosses.
func TestStaticFieldSkippedOnBloodRaven(t *testing.T) {
	h := newHarness(t)
	h.f.SetHotkeys(map[int]string{36: "f8", 42: "f1"})
	p := &d2.Player{Skills: []d2.Skill{{ID: 36, Bonus: 1}, {ID: 42, Level: 20}}, Mana: 200, MaxMana: 200}
	br := Target{Boss: true, Elite: true, Dist: 5, LifePct: 100, Mon: &data.Monster{Key: "bloodraven"}}
	other := Target{Boss: true, Elite: true, Dist: 5, LifePct: 100, Mon: &data.Monster{Key: "andariel"}}
	if _, ok := h.f.staticOpener(p, &other, &track{}); !ok {
		t.Fatal("static field should open a normal boss fight in this setup")
	}
	if _, ok := h.f.staticOpener(p, &br, &track{}); ok {
		t.Fatal("static field opened on Blood Raven with staticCastsBR 0")
	}
}
