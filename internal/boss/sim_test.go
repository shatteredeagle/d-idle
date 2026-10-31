package boss

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"
)

// sim wraps an objFake with a step cap (a spinning loop fails the test instead of hanging it), a fault injector
// (the n-th action returns failErr, once) and a counter of town portal use. Fake time advances only in Sleep,
// Attack and ClearAround, so a loop that never advances time trips the cap.
type sim struct {
	*objFake
	steps, cap int
	acts       int // action calls (GoTo, Attack, ClearAround, UseObject): the fault injector counts these
	failAt     int
	failErr    error
	spun       bool
	onAct      func(n int) // runs at the start of every action
}

func newSim(o *objFake) *sim { return &sim{objFake: o, cap: 20000} }

func (s *sim) tick() error {
	s.steps++
	if s.steps > s.cap {
		s.spun = true
		return context.Canceled
	}
	return nil
}

func (s *sim) act() error {
	if err := s.tick(); err != nil {
		return err
	}
	s.acts++
	if s.onAct != nil {
		s.onAct(s.acts)
	}
	if s.failAt > 0 && s.acts == s.failAt {
		return s.failErr
	}
	return nil
}

func (s *sim) View(ctx context.Context) (View, error) {
	if err := s.tick(); err != nil && s.spun {
		return View{}, err
	}
	return s.objFake.View(ctx)
}
func (s *sim) GoTo(ctx context.Context, p Pt) error {
	if err := s.act(); err != nil {
		return err
	}
	return s.objFake.GoTo(ctx, p)
}
func (s *sim) Attack(ctx context.Context, id uint32, d time.Duration) error {
	if err := s.act(); err != nil {
		return err
	}
	if s.attackErr != nil {
		return s.attackErr // a failing Attack returns at once and costs no game time
	}
	return s.objFake.Attack(ctx, id, d)
}
func (s *sim) ClearAround(ctx context.Context, p Pt, r int, b time.Duration) error {
	if err := s.act(); err != nil {
		return err
	}
	return s.objFake.ClearAround(ctx, p, r, b)
}
func (s *sim) UseObject(ctx context.Context, c int, at Pt) error {
	if err := s.act(); err != nil {
		return err
	}
	return s.objFake.UseObject(ctx, c, at)
}
func (s *sim) Sleep(ctx context.Context, d time.Duration) error {
	if err := s.tick(); err != nil && s.spun {
		return err
	}
	return s.objFake.Sleep(ctx, d)
}

func (s *sim) openCount() int {
	n := 0
	for _, c := range []int{SealVizier1, SealVizier2, SealSeis, SealInfector1, SealInfector2} {
		if s.modes[c] != 0 {
			n++
		}
	}
	return n
}

func (s *sim) noSpin(t *testing.T) {
	t.Helper()
	if s.spun {
		t.Fatalf("loop spun past %d steps, log tail %v", s.cap, s.log[max(0, len(s.log)-8):])
	}
}

func TestSealSimHappyPathBounded(t *testing.T) {
	s := newSim(chaos(true, false, true))
	t0 := s.Now()
	res, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if err != nil || !res.Diablo || s.openCount() != 5 || len(res.Bosses) != 3 {
		t.Fatalf("res %+v err %v", res, err)
	}
	if el := s.Now().Sub(t0); el > 15*time.Minute {
		t.Fatalf("took %v of game time", el)
	}
	if s.townCalls != 0 || s.did("town") != 0 {
		t.Fatal("the seal phase must never open a town portal")
	}
}

func TestSealSimSealThatDoesNotOpen(t *testing.T) {
	for _, stuck := range []int{SealVizier1, SealVizier2, SealSeis, SealInfector1, SealInfector2} {
		o := chaos(true, true, true)
		s := newSim(o)
		inner := o.onUse
		o.onUse = func(o *objFake, class int) {
			inner(o, class)
			if class == stuck {
				o.modes[class] = 0 // the click lands but the seal stays closed
			}
		}
		_, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
		s.noSpin(t)
		if !errors.Is(err, ErrSealStuck) {
			t.Fatalf("seal %d: want ErrSealStuck, got %v", stuck, err)
		}
		n := 0
		for _, c := range o.uses {
			if c == stuck {
				n++
			}
		}
		if n != 5 {
			t.Fatalf("seal %d: %d clicks, want exactly Tries=5", stuck, n)
		}
		if s.townCalls != 0 {
			t.Fatal("town portal used")
		}
	}
}

func TestSealSimBossNeverSpawnsStillOpensAndFindsDiablo(t *testing.T) {
	o := chaos(true, true, true)
	s := newSim(o)
	inner := o.onUse
	o.onUse = func(o *objFake, class int) {
		inner(o, class)
		for _, m := range o.mons {
			if m.UniqueNo >= uniqueInfector && m.UniqueNo <= uniqueVizier {
				m.area = 0
			}
		}
	}
	t0 := s.Now()
	res, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if err != nil || !res.Diablo || len(res.Bosses) != 0 || len(res.Notes) < 3 {
		t.Fatalf("res %+v err %v", res, err)
	}
	if el := s.Now().Sub(t0); el > 5*time.Minute {
		t.Fatalf("waiting for three phantom bosses took %v", el)
	}
}

// A boss that outlives its fight budget no longer aborts the run: the other seals open, the boss gets one retry at
// the end, and Diablo (which only needs the seals) is still reached.
func TestSealSimUnkillableBossIsRecordedAndRunContinues(t *testing.T) {
	o := chaos(true, true, true)
	s := newSim(o)
	inner := o.onUse
	o.onUse = func(o *objFake, class int) {
		inner(o, class)
		for _, m := range o.mons {
			if m.UniqueNo == uniqueVizier {
				m.hp = 1 << 30
			}
		}
	}
	t0 := s.Now()
	res, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if err != nil || !res.Diablo || s.openCount() != 5 {
		t.Fatalf("res %+v err %v", res, err)
	}
	if len(res.Unkilled) != 1 || res.Unkilled[0] != "Grand Vizier of Chaos" || len(res.Bosses) != 2 {
		t.Fatalf("unkilled %v bosses %v", res.Unkilled, res.Bosses)
	}
	if el := s.Now().Sub(t0); el > 15*time.Minute {
		t.Fatalf("boss budget overrun: %v", el)
	}
}

func TestSealSimStubbornBossDiesOnRetry(t *testing.T) {
	o := chaos(true, true, true)
	s := newSim(o)
	inner := o.onUse
	t0 := s.Now()
	o.onUse = func(o *objFake, class int) {
		inner(o, class)
		for _, m := range o.mons {
			if m.UniqueNo == uniqueVizier {
				m.hp = 1 << 30
			}
		}
	}
	s.onAct = func(int) { // the boss becomes killable once the first fight budget is spent
		if s.Now().Sub(t0) > 4*time.Minute {
			for _, m := range o.mons {
				if m.UniqueNo == uniqueVizier && m.hp > 1000 {
					m.hp = 30
				}
			}
		}
	}
	res, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if err != nil || !res.Diablo || len(res.Unkilled) != 0 || len(res.Bosses) != 3 {
		t.Fatalf("res %+v err %v", res, err)
	}
}

// Diablo never showing up after a stubborn boss is the only failure.
func TestSealSimDiabloUnreachableFailsRun(t *testing.T) {
	o := chaos(true, true, true)
	s := newSim(o)
	inner := o.onUse
	o.onUse = func(o *objFake, class int) {
		inner(o, class)
		for _, m := range o.mons {
			if m.UniqueNo == uniqueVizier {
				m.hp = 1 << 30
			}
			if m.Class == ClassDiablo {
				m.area = 0
			}
		}
	}
	_, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if !errors.Is(err, ErrBossNotFound) {
		t.Fatalf("want ErrBossNotFound, got %v", err)
	}
}

// Resumed from inside a seal room, the walk goes back to the star by the room path, not via the entrance route.
func TestSealSimResumeFromSealRoomSkipsEntranceRoute(t *testing.T) {
	o := chaos(true, true, true)
	o.pos = Pt{7665, 5276} // vizier A room node
	s := newSim(o)
	res, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if err != nil || !res.Diablo || s.openCount() != 5 {
		t.Fatalf("res %+v err %v", res, err)
	}
	for _, n := range EntranceToStar[:len(EntranceToStar)-1] {
		if o.did(fmt.Sprintf("goto %d,%d", n.X, n.Y)) != 0 {
			t.Fatalf("walked entrance node %v", n)
		}
	}
	if got := o.log[0]; got != "goto 7665,5276" {
		t.Fatalf("first step %q, want the nearest node", got)
	}
}

func TestSealSimResumeFromEntranceMidRoute(t *testing.T) {
	o := chaos(true, true, true)
	o.pos = Pt{7777, 5408} // an entrance route node
	s := newSim(o)
	if _, err := OpenSeals(context.Background(), s, s, s, SealOpts{}); err != nil {
		t.Fatal(err)
	}
	if o.did("goto 7794,5517") != 0 || o.did("goto 7777,5408") == 0 {
		t.Fatalf("should resume at the nearest entrance node: %v", o.log[:3])
	}
}

// Death is noticed on the next walk step even when every action keeps succeeding.
func TestSealSimDeathDuringWalkStopsAtOnce(t *testing.T) {
	o := chaos(true, true, true)
	o.pos = Pt{7790, 5544}
	s := newSim(o)
	s.onAct = func(n int) {
		if n == 3 {
			o.life = 0
		}
	}
	_, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if !errors.Is(err, ErrDead) {
		t.Fatalf("want ErrDead, got %v", err)
	}
	if s.acts > 3 {
		t.Fatalf("%d actions, want the walk to stop right after the death", s.acts)
	}
}

// A non-fatal Attack error that returns at once must not turn the boss loop into a busy spin.
func TestSealSimInstantAttackErrorDoesNotSpin(t *testing.T) {
	o := chaos(true, true, true)
	s := newSim(o)
	o.attackErr = errors.New("no line of sight")
	res, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if err != nil || len(res.Unkilled) != 3 {
		t.Fatalf("bosses that cannot be hit are recorded, not fatal: %+v %v", res, err)
	}
}

func TestSealSimDeathAnywhereIsErrDead(t *testing.T) {
	probe := newSim(chaos(true, true, true))
	if _, err := OpenSeals(context.Background(), probe, probe, probe, SealOpts{}); err != nil {
		t.Fatal(err)
	}
	total := probe.acts
	for n := 1; n <= total; n++ {
		s := newSim(chaos(true, true, true))
		s.failAt, s.failErr = n, ErrDead
		_, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
		s.noSpin(t)
		if !errors.Is(err, ErrDead) {
			t.Fatalf("death at step %d: want ErrDead, got %v", n, err)
		}
	}
}

// The character dies (life 0) while Actions keep returning plain errors: the executor must still report ErrDead and
// not ErrSealStuck (which would be retried as if the seal were broken).
func TestSealSimDeadPlayerIsNotSealStuck(t *testing.T) {
	o := chaos(true, true, true)
	s := newSim(o)
	o.life = 0
	o.useErr = errors.New("cannot interact while dead")
	_, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
	s.noSpin(t)
	if !errors.Is(err, ErrDead) {
		t.Fatalf("want ErrDead, got %v", err)
	}
}

// Restart at every step: the bot is killed (ErrChicken, ErrDead or plain context cancel) at step n, the world keeps
// its seal state, and a fresh OpenSeals must finish with all five seals open and no seal clicked twice.
func TestSealSimRestartAtEveryStep(t *testing.T) {
	probe := newSim(chaos(true, false, true))
	if _, err := OpenSeals(context.Background(), probe, probe, probe, SealOpts{}); err != nil {
		t.Fatal(err)
	}
	total := probe.acts
	for _, fail := range []error{ErrChicken, ErrDead, ErrEscaped} {
		for n := 1; n <= total; n++ {
			o := chaos(true, false, true)
			s := newSim(o)
			s.failAt, s.failErr = n, fail
			if _, err := OpenSeals(context.Background(), s, s, s, SealOpts{}); !errors.Is(err, fail) {
				t.Fatalf("%v at %d: first run err %v", fail, n, err)
			}
			s.failAt, s.acts, s.steps = 0, 0, 0
			res, err := OpenSeals(context.Background(), s, s, s, SealOpts{})
			s.noSpin(t)
			if err != nil || !res.Diablo || s.openCount() != 5 {
				t.Fatalf("%v at %d: restart res %+v err %v", fail, n, res, err)
			}
			seen := map[int]int{}
			for _, c := range o.uses {
				seen[c]++
				if seen[c] > 1 {
					t.Fatalf("%v at %d: seal %d clicked twice: %v", fail, n, c, o.uses)
				}
			}
		}
	}
}

func TestSealSimRunDiabloNoTownPortalWithoutTP(t *testing.T) {
	o := chaos(true, true, true)
	o.hasTP = false
	o.level = 40
	o.lootN = 1
	s := newSim(o)
	res, err := RunDiablo(context.Background(), s, s, s, SealOpts{}, Opts{})
	s.noSpin(t)
	if err != nil || !res.Killed {
		t.Fatalf("res %+v err %v", res, err)
	}
	if o.did("town") != 0 {
		t.Fatalf("town used without a portal: %v", o.log)
	}
}

func TestSealSimRunDiabloDeathMidFight(t *testing.T) {
	o := chaos(true, true, true)
	o.level = 40
	s := newSim(o)
	o.onAttack = func(f *fake, m *fmon) {
		if m.Class == ClassDiablo {
			f.life = 0
		}
	}
	_, err := RunDiablo(context.Background(), s, s, s, SealOpts{}, Opts{})
	s.noSpin(t)
	if !errors.Is(err, ErrDead) {
		t.Fatalf("want ErrDead, got %v", err)
	}
}

// ---- Baal waves ----

func TestWaveSimBoundedWhenWavesNeverEnd(t *testing.T) {
	// every dead monster is replaced: the loop must end at WaveOpts.Timeout, not run forever
	o := throne([][]int{{101}})
	s := newSim(o)
	s.cap = 200000
	o.onAttack = func(f *fake, m *fmon) {
		if m.Class != ClassBaalThrone && m.hp <= 30 {
			f.addBoss(Mon{Class: 101, UniqueNo: -1, Pos: Pt{15090, 5040}}, AreaThrone, 40)
		}
	}
	t0 := s.Now()
	_, err := RunWaves(context.Background(), s, s, WaveOpts{Timeout: 5 * time.Minute})
	s.noSpin(t)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
	if el := s.Now().Sub(t0); el > 6*time.Minute {
		t.Fatalf("overran the timeout: %v", el)
	}
	if s.townCalls != 0 {
		t.Fatal("town portal used in the throne")
	}
}

func TestWaveSimBaalNeverLeaves(t *testing.T) {
	o := throne([][]int{{101}, {ClassLister}})
	s := newSim(o)
	o.onAttack = nil // Baal never leaves
	for _, m := range o.mons {
		if m.Class != ClassBaalThrone {
			m.Mode = 12
		}
	}
	o.addBoss(Mon{Class: ClassLister, UniqueNo: -1, Pos: Pt{15090, 5040}}, AreaThrone, 30)
	t0 := s.Now()
	_, err := RunWaves(context.Background(), s, s, WaveOpts{})
	s.noSpin(t)
	_ = t0
	if err == nil || errors.Is(err, ErrDead) {
		t.Fatalf("want a timeout-style failure, got %v", err)
	}
}

func TestWaveSimBaalNeverSpawnsEndsFast(t *testing.T) {
	o := newObjFake()
	o.level, o.area, o.potions = 34, AreaThrone, 8
	s := newSim(o)
	t0 := s.Now()
	res, err := RunWaves(context.Background(), s, s, WaveOpts{})
	s.noSpin(t)
	if err != nil || !res.BaalGone {
		t.Fatalf("res %+v err %v", res, err)
	}
	if el := s.Now().Sub(t0); el > 30*time.Second {
		t.Fatalf("took %v", el)
	}
}

func TestWaveSimInstantAttackErrorDoesNotSpin(t *testing.T) {
	o := throne([][]int{{101}})
	s := newSim(o)
	o.attackErr = errors.New("no line of sight")
	_, err := RunWaves(context.Background(), s, s, WaveOpts{Timeout: time.Minute})
	s.noSpin(t)
	if !errors.Is(err, ErrTimeout) {
		t.Fatalf("want ErrTimeout, got %v", err)
	}
}

func TestWaveSimDeathAtEveryStep(t *testing.T) {
	mk := func() *sim {
		return newSim(throne([][]int{{101, 102}, {103}, {ClassCouncilA5}, {104}, {ClassLister}}))
	}
	probe := mk()
	if _, err := RunWaves(context.Background(), probe, probe, WaveOpts{}); err != nil {
		t.Fatal(err)
	}
	for _, fail := range []error{ErrDead, ErrChicken, ErrEscaped} {
		for n := 1; n <= probe.acts; n++ {
			s := mk()
			s.failAt, s.failErr = n, fail
			_, err := RunWaves(context.Background(), s, s, WaveOpts{})
			s.noSpin(t)
			if !errors.Is(err, fail) {
				t.Fatalf("%v at step %d: got %v", fail, n, err)
			}
			if s.townCalls != 0 {
				t.Fatalf("town portal used at step %d", n)
			}
		}
	}
}

func TestWaveSimRunBaalRestartFromThroneAndChamber(t *testing.T) {
	mk := func() *sim {
		o := throne([][]int{{101}, {ClassLister}})
		o.presets = []Preset{{Kind: PresetObject, ID: objBaalPortal, Pos: Pt{15090, 5010}}}
		o.onUse = func(o *objFake, class int) {
			if class == objBaalPortal {
				o.area = AreaWorldstoneChamb
				o.addBoss(Mon{Class: ClassBaal, UniqueNo: -1, Pos: Pt{15134, 5900}}, AreaWorldstoneChamb, 90)
			}
		}
		o.lootN = 1
		return newSim(o)
	}
	probe := mk()
	if _, err := RunBaal(context.Background(), probe, probe, probe, WaveOpts{}, Opts{}); err != nil {
		t.Fatal(err)
	}
	for n := 1; n <= probe.acts; n++ {
		s := mk()
		s.failAt, s.failErr = n, ErrChicken
		_, err := RunBaal(context.Background(), s, s, s, WaveOpts{}, Opts{})
		if !errors.Is(err, ErrChicken) {
			t.Fatalf("step %d first run: %v", n, err)
		}
		s.failAt, s.acts, s.steps = 0, 0, 0
		res, err := RunBaal(context.Background(), s, s, s, WaveOpts{}, Opts{})
		s.noSpin(t)
		if err != nil || !res.Killed {
			t.Fatalf("step %d restart: res %+v err %v", n, res, err)
		}
	}
}

func TestWaveSimPortalNeverOpens(t *testing.T) {
	o := throne([][]int{{101}})
	o.presets = []Preset{{Kind: PresetObject, ID: objBaalPortal, Pos: Pt{15090, 5010}}}
	s := newSim(o) // UseObject succeeds but the area never changes
	_, err := RunBaal(context.Background(), s, s, s, WaveOpts{}, Opts{})
	s.noSpin(t)
	if !errors.Is(err, ErrLeftArena) {
		t.Fatalf("want ErrLeftArena, got %v", err)
	}
	if len(o.uses) != 1 {
		t.Fatalf("portal clicked %d times", len(o.uses))
	}
}
