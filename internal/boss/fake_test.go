package boss

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// fake is a scripted world: one value implements World and Actions, time moves only in Sleep and Attack.
type fake struct {
	now       time.Time
	area      int
	pos       Pt
	life, max int
	potions   int
	antidotes int
	poisoned  bool
	hasTP     bool
	level     int
	diff      int
	mons      []*fmon
	dangers   []Danger
	presets   []Preset
	log       []string
	loots     []LootSpec
	lootN     int
	exploreFn func(f *fake) // reveals monsters
	onAttack  func(f *fake, m *fmon)
	exitFail  map[int]bool
	attackErr error
	arriveFn  func(f *fake, area int)
	townCalls int
	drinkErr  error
	antiErr   error
	lootSeq   []int // per Loot call, then lootN
	townErr   error
	presetErr error
}

type fmon struct {
	Mon
	area     int
	vis      int // visible only within this distance of the player, 0 = always
	hp       int
	dmg      int // damage to the player per slice when within 6
	approach int // subtiles moved toward the player per slice
}

func newFake() *fake {
	return &fake{now: time.Unix(1000, 0), area: 1, pos: Pt{100, 100}, life: 100, max: 100, potions: 3, hasTP: true, level: 30}
}

func (f *fake) logf(s string, a ...any) { f.log = append(f.log, fmt.Sprintf(s, a...)) }
func (f *fake) did(s string) int {
	n := 0
	for _, l := range f.log {
		if len(l) >= len(s) && l[:len(s)] == s {
			n++
		}
	}
	return n
}

func (f *fake) addBoss(m Mon, area, hp int) *fmon {
	m.Hostile, m.LifePct, m.Mode = true, 100, 2
	if m.ID == 0 {
		m.ID = uint32(len(f.mons) + 1)
	}
	fm := &fmon{Mon: m, area: area, hp: hp}
	f.mons = append(f.mons, fm)
	return fm
}

func (f *fake) View(ctx context.Context) (View, error) {
	if err := ctx.Err(); err != nil {
		return View{}, err
	}
	v := View{Area: f.area, Difficulty: f.diff, Level: f.level, Pos: f.pos, Life: f.life, Max: f.max, Poisoned: f.poisoned,
		Antidotes: f.antidotes, Potions: f.potions, HasTP: f.hasTP, Dead: f.life <= 0, Dangers: f.dangers}
	for _, m := range f.mons {
		if m.area != f.area {
			continue
		}
		if m.vis > 0 && f.pos.Dist(m.Pos) > m.vis {
			continue
		}
		v.Monsters = append(v.Monsters, m.Mon)
	}
	return v, nil
}

func (f *fake) Presets(ctx context.Context) ([]Preset, error) { return f.presets, f.presetErr }
func (f *fake) Now() time.Time                                { return f.now }
func (f *fake) Sleep(ctx context.Context, d time.Duration) error {
	f.now = f.now.Add(d)
	return ctx.Err()
}

func (f *fake) arrive(area int) {
	f.area = area
	if f.arriveFn != nil {
		f.arriveFn(f, area)
	}
}

func (f *fake) Travel(ctx context.Context, area int) error {
	f.logf("travel %d", area)
	f.arrive(area)
	return nil
}

func (f *fake) ExitTo(ctx context.Context, area int) error {
	f.logf("exit %d", area)
	if f.exitFail[area] {
		return errors.New("no exit")
	}
	f.arrive(area)
	return nil
}

func (f *fake) GoTo(ctx context.Context, p Pt) error {
	f.logf("goto %d,%d", p.X, p.Y)
	f.pos = p
	return nil
}
func (f *fake) Explore(ctx context.Context, b time.Duration, stop func() bool) error {
	f.logf("explore")
	if f.exploreFn != nil {
		f.exploreFn(f)
	}
	f.now = f.now.Add(b / 3)
	stop()
	return nil
}

func (f *fake) Attack(ctx context.Context, id uint32, slice time.Duration) error {
	f.logf("attack %d", id)
	f.now = f.now.Add(slice)
	if f.attackErr != nil {
		return f.attackErr
	}
	for _, m := range f.mons {
		if m.ID != id || !m.Alive() {
			continue
		}
		if f.onAttack != nil {
			f.onAttack(f, m)
		}
		m.hp -= 30
		m.LifePct = max0(m.hp * 100 / 100)
		if m.hp <= 0 {
			m.Mode, m.LifePct = 0, 0
		}
		if m.approach > 0 && m.Pos.Dist(f.pos) > 3 {
			step := m.approach
			if d := m.Pos.Dist(f.pos) - 2; d < step {
				step = d
			}
			m.Pos = awayPoint(m.Pos, f.pos, -step)
		}
		if m.dmg > 0 && m.Pos.Dist(f.pos) <= 6 && m.Alive() {
			f.life -= m.dmg
		}
	}
	return nil
}

func max0(a int) int {
	if a < 0 {
		return 0
	}
	return a
}

func (f *fake) ClearAround(ctx context.Context, p Pt, r int, b time.Duration) error {
	f.logf("clear %d", r)
	f.now = f.now.Add(b / 2)
	for _, m := range f.mons {
		if m.area == f.area && !(m.Class == 267 || m.Class == 156 || m.Class == 402 || m.Name != "") && m.Pos.Dist(p) <= r {
			m.Mode = 12
		}
	}
	return nil
}

func (f *fake) Antidote(ctx context.Context) error {
	f.logf("antidote")
	if f.antiErr != nil {
		return f.antiErr
	}
	f.antidotes--
	f.poisoned = false
	return nil
}

func (f *fake) DrinkHealing(ctx context.Context) error {
	f.logf("drink")
	if f.drinkErr != nil {
		return f.drinkErr
	}
	f.potions--
	f.life = f.max
	return nil
}

func (f *fake) Loot(ctx context.Context, s LootSpec) (int, error) {
	f.logf("loot")
	f.loots = append(f.loots, s)
	if len(f.lootSeq) > 0 {
		n := f.lootSeq[0]
		f.lootSeq = f.lootSeq[1:]
		return n, nil
	}
	return f.lootN, nil
}

func (f *fake) ToTown(ctx context.Context) error {
	f.logf("town")
	if f.townErr != nil {
		return f.townErr
	}
	f.townCalls++
	f.area = 1
	return nil
}
func (f *fake) Chicken(ctx context.Context) error { f.logf("chicken"); return nil }
