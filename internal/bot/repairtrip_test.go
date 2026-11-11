package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goldfarm"
	"d2lod-bot/internal/repair"
)

// Both classes run every scenario: the repair trip reads no class (the world only changes the player's class).
var tripClasses = []d2.Class{d2.Sorceress, d2.Paladin}

type gearBox struct{ dur map[string]int }

func (b *gearBox) cond() []repair.Cond {
	var out []repair.Cond
	for slot, d := range b.dur {
		if c, ok := repair.Classify(slot, slot, d, 100, 30); ok {
			out = append(out, c)
		}
	}
	return out
}
func (b *gearBox) full() {
	for k := range b.dur {
		b.dur[k] = 100
	}
}

// repairTown is the fake town of a repair trip: chores run the repair like the real town does (Repaired on success,
// Failed or Short otherwise), counted.
type repairTown struct {
	*fakeTown
	rc     *repair.Controller
	gear   *gearBox
	mode   string // "pay" (default), "fail", "short"
	chores int
	totown int
	back   int
	now    func() time.Time
}

func (t *repairTown) ToTown(ctx context.Context) error {
	t.totown++
	return t.fakeTown.ToTown(ctx)
}
func (t *repairTown) BackToField(ctx context.Context, l int) error {
	t.back++
	return t.fakeTown.BackToField(ctx, l)
}
func (t *repairTown) Chores(ctx context.Context, p ChorePlan) (ChoreReport, error) {
	t.chores++
	rep, err := t.fakeTown.Chores(ctx, p)
	if t.rc.Requested() {
		switch t.mode {
		case "fail":
			t.rc.Failed("repair not confirmed")
		case "short":
			t.rc.Short(120, 260)
		default:
			t.gear.full()
			t.rc.Repaired(t.now())
		}
	}
	return rep, err
}

type tripRig struct {
	*rig
	rc   *repair.Controller
	town *repairTown
	gear *gearBox
}

func newTripRig(t *testing.T, class d2.Class, gold, estimate int) *tripRig {
	t.Helper()
	g := campaignRig(t, &fakeQuests{}, 10)
	g.w.set(func(w *world) {
		w.area, w.level, w.class, w.gold0 = 2, 10, class, gold
		w.items = append(w.items, d2.Unit{ID: 900, Type: d2.TypeItem, Class: 5, Item: &d2.Item{Where: d2.ContInventory}}) // a town portal scroll
	})
	g.w.belt(101, 102, 103, 104)
	g.r.poll()
	gear := &gearBox{dur: map[string]int{"rhand": 40, "torso": 70}}
	rc, err := repair.New(repair.Config{
		Gear: gear.cond, Estimate: func() int { return estimate }, Now: g.clock.Now,
		Gold: func() int { s := g.r.latest(); return s.Player.Gold },
	})
	if err != nil {
		t.Fatal(err)
	}
	rt := &repairTown{fakeTown: g.town, rc: rc, gear: gear, now: g.clock.Now}
	g.r.d.Town = rt
	g.r.SetRepairTrip(rc)
	return &tripRig{rig: g, rc: rc, town: rt, gear: gear}
}

// step is one goal step as far as the repair trip is concerned (goal.go order).
func (g *tripRig) step() bool {
	s := g.r.poll()
	c := levelGoalControl()
	if g.r.loop.trip != nil {
		return g.r.advanceTrip(context.Background(), s, c)
	}
	return g.r.repairTripStep(context.Background(), s, c)
}

func (g *tripRig) steps(n int) {
	for i := 0; i < n; i++ {
		g.step()
	}
}

// ---- the pure rule

func TestSafeMomentRule(t *testing.T) {
	cfg := repair.Trip{SafeWaitSecs: 60, PortalClearTiles: 10, ElitePotionMin: 2}
	cases := []struct {
		name string
		in   safeIn
		want safeAct
	}{
		{"quiet", safeIn{}, safeGo},
		{"a pack in reach is finished first", safeIn{Fight: 3, Close: 2, WaitedS: 10}, safeWait},
		{"after the bound the hero steps away", safeIn{Fight: 3, Close: 2, WaitedS: 61}, safeRetreat},
		{"far pack, nothing close, after the bound still waits", safeIn{Fight: 3, WaitedS: 90}, safeWait},
		{"a boss is killed first with life and escape", safeIn{Fight: 2, Close: 2, Elite: true, Potions: 3, Escape: true, LifePct: 80, WaitedS: 90}, safeWait},
		{"a boss without potions: the bound applies", safeIn{Fight: 2, Close: 2, Elite: true, Potions: 0, Escape: true, LifePct: 80, WaitedS: 90}, safeRetreat},
		{"a boss at low life: the bound applies", safeIn{Fight: 2, Close: 2, Elite: true, Potions: 3, Escape: true, LifePct: 30, WaitedS: 90}, safeRetreat},
		{"the boss exception is bounded too", safeIn{Fight: 2, Close: 2, Elite: true, Potions: 3, Escape: true, LifePct: 80, WaitedS: 181}, safeRetreat},
		{"something close but no fight", safeIn{Close: 1, WaitedS: 5}, safeWait},
		{"retreat steps are bounded", safeIn{Fight: 2, Close: 2, WaitedS: 90, Retreats: repairRetreatMax}, safeWait},
		{"a quest step that cannot be interrupted", safeIn{NoBreak: true, Reason: "x"}, safeWait},
	}
	for _, c := range cases {
		if got, _ := repairSafeMoment(c.in, cfg); got != c.want {
			t.Errorf("%s: %d, want %d", c.name, got, c.want)
		}
	}
}

// ---- the trip

func TestRepairTripQuietFieldBothClasses(t *testing.T) {
	for _, cl := range tripClasses {
		g := newTripRig(t, cl, 400, 200)
		if _, err := g.rc.Request(); err != nil {
			t.Fatal(err)
		}
		g.steps(12)
		if g.town.totown != 1 || g.town.chores != 1 || g.town.back != 1 {
			t.Fatalf("%v: totown %d chores %d back %d (%s)", cl, g.town.totown, g.town.chores, g.town.back, g.w.log())
		}
		if !strings.Contains(g.w.log(), "pickup") {
			t.Errorf("%v: no loot pass before leaving: %s", cl, g.w.log())
		}
		q := g.rc.View().Request
		if g.rc.Requested() || q.State != repair.StateDone || q.DurBefore != 40 || q.DurAfter != 100 || q.Trips != 1 {
			t.Errorf("%v: %+v", cl, q)
		}
		if g.r.loop.trip != nil {
			t.Errorf("%v: trip left behind", cl)
		}
	}
}

// Never mid-fight: a pack in reach holds the trip; the Why/phase say so; once it is dead the trip goes.
func TestRepairTripWaitsForThePackThenGoes(t *testing.T) {
	for _, cl := range tripClasses {
		g := newTripRig(t, cl, 400, 200)
		g.w.addMonster(1, 10, 6, 0)
		g.w.addMonster(2, 10, 12, 4)
		g.rc.Request()
		g.steps(3)
		if g.town.totown != 0 {
			t.Fatalf("%v: left in the middle of a fight", cl)
		}
		q, _ := g.rc.Active()
		if q.Phase != repair.PhaseWaiting || !strings.Contains(q.Note, "finishing the pack first") {
			t.Fatalf("%v: %+v", cl, q)
		}
		if code, _, ok := g.r.repairWhy(g.r.latest()); !ok || code != WhyRepairWait {
			t.Errorf("%v: why %q %v", cl, code, ok)
		}
		g.w.set(func(w *world) { w.mons = nil })
		g.steps(12)
		if g.town.totown != 1 || g.rc.Requested() {
			t.Errorf("%v: totown %d requested %v", cl, g.town.totown, g.rc.Requested())
		}
	}
}

// A monster within the portal radius also stops the cast at the last moment (it came while the loot pass ran).
func TestRepairTripRefusesToCastNearAHostile(t *testing.T) {
	g := newTripRig(t, d2.Paladin, 400, 200)
	g.rc.Request()
	g.r.loop.repairLooted = true
	g.w.addMonster(1, 10, 8, 0)
	s := g.r.poll()
	g.r.loop.trip = &trip{plan: g.r.fullChores("repair trip"), returnTo: 2, repair: true}
	if g.r.advanceTrip(context.Background(), s, levelGoalControl()); g.town.totown != 0 || g.r.loop.trip != nil {
		t.Fatalf("totown %d trip %v", g.town.totown, g.r.loop.trip)
	}
	if q, _ := g.rc.Active(); q.Phase != repair.PhaseWaiting || q.Trips != 0 {
		t.Errorf("%+v", q)
	}
}

// After the bound with a pack still close the hero steps away (the fighter's retreat) before anything else.
func TestRepairTripRetreatsAfterTheBound(t *testing.T) {
	g := newTripRig(t, d2.Sorceress, 400, 200)
	g.w.addMonster(1, 10, 6, 0)
	g.rc.Request()
	g.step()
	g.clock.Sleep(61 * time.Second)
	g.step()
	if g.w.count("act crowd-retreat") == 0 || g.town.totown != 0 {
		t.Fatalf("retreat steps %d totown %d (%s)", g.w.count("act crowd-retreat"), g.town.totown, g.w.log())
	}
}

// In town: no portal, straight to the smith, no walk back.
func TestRepairTripInTownGoesStraightToTheSmith(t *testing.T) {
	for _, cl := range tripClasses {
		g := newTripRig(t, cl, 400, 200)
		g.w.set(func(w *world) { w.area = 1 })
		g.rc.Request()
		g.steps(6)
		if g.town.totown != 0 || g.town.chores != 1 || g.town.back != 0 || g.rc.Requested() {
			t.Errorf("%v: totown %d chores %d back %d requested %v", cl, g.town.totown, g.town.chores, g.town.back, g.rc.Requested())
		}
	}
}

// The smith fails: one more round, then the request stays open and queued with the reason; no second trip inside the
// cooldown (no portal spam), a second after it, and two failed trips end the request as failed.
func TestRepairTripSmithFailureKeepsTheRequestAndGuardsTheLoop(t *testing.T) {
	for _, cl := range tripClasses {
		g := newTripRig(t, cl, 400, 200)
		g.town.mode = "fail"
		g.rc.Request()
		g.steps(14)
		if g.town.chores != 2 { // the round and its one retry
			t.Fatalf("%v: chores %d", cl, g.town.chores)
		}
		q, ok := g.rc.Active()
		if !ok || q.Phase != repair.PhaseQueued || q.Trips != 1 || !strings.Contains(q.Note, "repair not confirmed") {
			t.Fatalf("%v: %+v", cl, q)
		}
		trips := g.town.totown
		g.clock.Sleep(5 * time.Minute)
		g.steps(10)
		if g.town.totown != trips {
			t.Errorf("%v: a second trip inside the cooldown", cl)
		}
		g.clock.Sleep(20 * time.Minute)
		g.steps(20)
		if g.town.totown != trips+1 {
			t.Fatalf("%v: no second trip after the cooldown: %d", cl, g.town.totown)
		}
		if g.rc.Requested() || g.rc.View().Request.State != repair.StateFailed {
			t.Errorf("%v: %+v", cl, g.rc.View().Request)
		}
	}
}

// No way home: the trip is given up after the failures and the request waits queued, one trip counted.
func TestRepairTripNoWayHome(t *testing.T) {
	g := newTripRig(t, d2.Paladin, 400, 200)
	g.rig.town.toTownErr = errBoom
	g.rc.Request()
	g.steps(10)
	q, _ := g.rc.Active()
	if g.r.loop.trip != nil || q.Phase != repair.PhaseQueued || q.Trips != 1 || g.town.totown > 3 {
		t.Errorf("trip %v %+v totown %d", g.r.loop.trip, q, g.town.totown)
	}
}

// The smith asks for more than the pack has: the request becomes a farm leg, then (still short) is cancelled.
func TestRepairTripShortAtTheSmithFarmsOnceThenCancels(t *testing.T) {
	g := newTripRig(t, d2.Sorceress, 400, 200)
	g.town.mode = "short"
	gf, _ := goldfarm.New("", nil)
	g.r.SetGoldFarm(gf, func() int { return 200 })
	g.rc.Request()
	g.steps(12)
	q, ok := g.rc.Active()
	if !ok || q.Kind != repair.KindFarm || q.Phase != repair.PhaseFarming || q.FarmLegs != 1 {
		t.Fatalf("after the first short: %+v", q)
	}
	if g.town.chores != 1 {
		t.Errorf("a short smith must not be retried: %d", g.town.chores)
	}
	g.w.set(func(w *world) { w.gold0 = 5000 }) // the leg's target is met
	g.clock.Sleep(25 * time.Minute)
	g.steps(14)
	if q := g.rc.View().Request; q.State != repair.StateCancel || !q.Auto || !strings.Contains(q.Note, "still short") {
		t.Fatalf("second short must cancel with a reason: %+v", q)
	}
	if gf.On() {
		t.Error("the farm order outlived the request")
	}
}

// Farm until repair: the farm order runs for the request with the repair x 1.3 target, then the trip, then the order
// ends itself and the plan goes on.
func TestFarmUntilRepairThenTripThenOrderEnds(t *testing.T) {
	for _, cl := range tripClasses {
		g := newTripRig(t, cl, 60, 200)
		gf, _ := goldfarm.New("", nil)
		g.r.SetGoldFarm(gf, func() int { return 200 })
		q, err := g.rc.Request()
		if err != nil || q.Kind != repair.KindFarm {
			t.Fatalf("%v: %+v %v", cl, q, err)
		}
		gf.StartForRepair() // what the adapter does
		g.step()
		have, want := g.r.repairFarmFigures(g.r.latest())
		if have != 60 || want < 260+1 {
			t.Fatalf("%v: have %d want %d (repair x1.3 + kit + margin)", cl, have, want)
		}
		if q, _ := g.rc.Active(); q.Phase != repair.PhaseFarming || q.Have != 60 || q.Target != want {
			t.Fatalf("%v: %+v", cl, q)
		}
		if g.town.totown != 0 || !gf.On() || !gf.ForRepair() {
			t.Fatalf("%v: left early", cl)
		}
		g.w.set(func(w *world) { w.gold0 = want + 10 })
		g.steps(14)
		if g.town.totown != 1 || g.town.chores != 1 || g.rc.Requested() {
			t.Fatalf("%v: totown %d chores %d requested %v", cl, g.town.totown, g.town.chores, g.rc.Requested())
		}
		if gf.On() {
			t.Errorf("%v: the farm order must end with the repair", cl)
		}
	}
}

// A Farm gold order the player gave is not ended by a repair request finishing.
func TestPlayerFarmOrderSurvivesRepair(t *testing.T) {
	g := newTripRig(t, d2.Paladin, 400, 200)
	gf, _ := goldfarm.New("", nil)
	g.r.SetGoldFarm(gf, func() int { return 200 })
	gf.Start()
	g.rc.Request()
	g.steps(12)
	if !gf.On() || gf.ForRepair() {
		t.Errorf("on %v forRepair %v", gf.On(), gf.ForRepair())
	}
}

// An emergency trip replaces a repair trip in progress: the request starts over, the trip does not count.
func TestRepairTripInterruptedStartsOver(t *testing.T) {
	g := newTripRig(t, d2.Sorceress, 400, 200)
	g.rc.Request()
	g.steps(2)
	g.rc.BeginTrip(g.clock.Now())
	g.rc.SetPhase(repair.PhaseRepairing, "")
	g.r.loop.trip = nil // replaced and finished
	g.steps(12)
	q := g.rc.View().Request
	if q.State != repair.StateDone || q.Trips != 1 {
		t.Errorf("the interrupted trip must not count and the next one must finish: %+v", q)
	}
}
