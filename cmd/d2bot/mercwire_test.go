package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/merc"
	"d2lod-bot/internal/nav"
	"d2lod-bot/internal/task"
)

type fakeReader struct {
	mu sync.Mutex
	st merc.MercState
	n  int
}

func (f *fakeReader) State() (merc.MercState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.n++
	return f.st, nil
}
func (f *fakeReader) Offers() ([]merc.HireOffer, error) { return nil, errors.New("none") }
func (f *fakeReader) set(st merc.MercState)             { f.mu.Lock(); f.st = st; f.mu.Unlock() }

type fakeReviver struct {
	calls int
	paid  int
	err   error
}

func (f *fakeReviver) Revive(context.Context) (int, error) { f.calls++; return f.paid, f.err }

type fakeCat struct{ bot.Catalog }

func (fakeCat) Item(class int) (bot.ItemMeta, bool) {
	switch class {
	case 1:
		return bot.ItemMeta{Name: "Scroll of Town Portal"}, true
	case 2:
		return bot.ItemMeta{Name: "Healing Potion", Potion: bot.PotionHealing}, true
	}
	return bot.ItemMeta{}, false
}

type clock struct{ t time.Time }

func (c *clock) now() time.Time { return c.t }

func testWire(t *testing.T, rd merc.Reader, rev Reviver) (*mercWire, *bytes.Buffer, *clock) {
	t.Helper()
	var buf bytes.Buffer
	dt := env.DefaultTuning()
	tune := &dt
	tune.Merc.Enabled = 1
	tune.Resource.GearDecisions = 1 // legacy: the older tests strip a geared merc; the player-only gate has its own test
	e := &env.Env{Tune: tune, Log: slog.New(slog.NewTextHandler(&buf, nil))}
	w := newMercWire(e, rd, rev, fakeCat{})
	c := &clock{time.Unix(1000, 0)}
	w.now = c.now
	return w, &buf, c
}

func townSnap(gold int, kit bool) *d2.Snapshot {
	s := &d2.Snapshot{Player: &d2.Player{Area: 1, Gold: gold, Level: 10, Class: d2.Sorceress}, Inventory: &d2.Inventory{}}
	if kit {
		s.Inventory.Items = []d2.Unit{
			{Class: 1, Item: &d2.Item{Where: d2.ContInventory}},
			{Class: 2, Item: &d2.Item{Where: d2.ContBelt}},
		}
	}
	return s
}

func TestKnobOffLeavesSeamsAndHookAlone(t *testing.T) {
	task.MercAlive = nil
	var s d2denv.Seams
	installMercSeam(&s)
	dt := env.DefaultTuning()
	tune := &dt
	if tune.Merc.Enabled != 1 {
		t.Fatal("merc.enabled must default to 1")
	}
	tune.Merc.Enabled = 0
	e := &env.Env{Tune: tune, Log: slog.Default()}
	got := s.Town(e, (*nav.Navigator)(nil))
	if _, wrapped := got.(*mercTown); wrapped {
		t.Error("wrapped with the knob off")
	}
	if task.MercAlive != nil {
		t.Error("hook installed with the knob off")
	}
	// the inner town passes through untouched
	marker := &struct{ bot.Townsman }{}
	s2 := d2denv.Seams{Town: func(*env.Env, *nav.Navigator) bot.Townsman { return marker }}
	installMercSeam(&s2)
	if s2.Town(e, nil) != bot.Townsman(marker) {
		t.Error("inner town not returned as is")
	}
	// knob on but not a d2d env: still nothing
	tune.Merc.Enabled = 1
	if _, wrapped := s2.Town(e, nil).(*mercTown); wrapped {
		t.Error("wrapped a non-d2d env")
	}
}

func TestReviveFlow(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 100, Level: 10, FreeCells: -1}}
	rev := &fakeReviver{paid: 750}
	w, buf, clk := testWire(t, rd, rev)
	ctx := context.Background()

	if w.ReviveStep(ctx, townSnap(5000, true)) != "" || rev.calls != 0 {
		t.Fatal("alive merc must not be revived")
	}
	// dies: the transition is counted once
	clk.t = clk.t.Add(10 * time.Second)
	rd.set(merc.MercState{Known: true, Owned: true, Alive: false, Level: 10, FreeCells: -1})
	if got := w.ReviveStep(ctx, townSnap(5000, false)); got != "merc revived" || rev.calls != 1 { // no kit: gold only (V read)
		t.Fatalf("got %q calls %d", got, rev.calls)
	}
	if st := w.Status(); st["deathsLastHour"] != 1 || st["revivesLastHour"] != 1 {
		t.Errorf("status %+v", st)
	}
	if !strings.Contains(buf.String(), "merc: revived price=750 level=10") || !strings.Contains(buf.String(), "merc: died") {
		t.Errorf("log:\n%s", buf.String())
	}
	// poor: below the price 750 plus the revive reserve cap
	clk.t = clk.t.Add(10 * time.Second)
	if got := w.ReviveStep(ctx, townSnap(700, true)); !strings.Contains(got, "gold") || rev.calls != 1 {
		t.Errorf("poor: %q", got)
	}
	rd.set(merc.MercState{Known: true, Owned: true, Alive: false, Level: 10, FreeCells: -1})
	// survival first: before the shop, with no potion carried, the revive waits
	if got := w.ReviveFirst(ctx, townSnap(9000, false)); got != "" {
		t.Errorf("revive first without potions: %q", got)
	}
	// field: no step outside town
	s := townSnap(9000, true)
	s.Player.Area = 2
	if w.ReviveStep(ctx, s) != "" {
		t.Error("revive outside town")
	}
	// the reviver fails: no revive counted
	rev.err = errors.New("no line")
	clk.t = clk.t.Add(time.Minute)
	rd.set(merc.MercState{Known: true, Owned: true, Alive: false, Level: 10})
	if got := w.ReviveStep(ctx, townSnap(9000, true)); got != "merc revive failed" {
		t.Errorf("failed: %q", got)
	}
	// backoff: no second attempt within the gap
	calls := rev.calls
	if got := w.ReviveStep(ctx, townSnap(9000, true)); !strings.Contains(got, "backoff") || rev.calls != calls {
		t.Errorf("backoff: %q", got)
	}
	clk.t = clk.t.Add(mercSkipGap + time.Second)
	if st := w.Status(); st["revivesLastHour"] != 1 {
		t.Errorf("a failed revive was counted: %+v", st)
	}
	// unknown merc (no reader answer): nothing
	rd.set(merc.MercState{})
	clk.t = clk.t.Add(time.Minute)
	if w.ReviveStep(ctx, townSnap(9000, true)) != "" {
		t.Error("unknown state acted")
	}
}

func TestHookAndTTL(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Alive: true}}
	w, _, clk := testWire(t, rd, nil)
	h := w.AliveHook()
	if a, k := h(); !a || !k {
		t.Error("alive owned merc must read alive+known")
	}
	h()
	h()
	if rd.n != 1 {
		t.Errorf("reads %d, want 1 inside the TTL", rd.n)
	}
	clk.t = clk.t.Add(mercStateTTL + time.Second)
	rd.set(merc.MercState{Known: true, Owned: true, Alive: false})
	if a, k := h(); a || !k {
		t.Error("dead merc must read known, not alive")
	}
	clk.t = clk.t.Add(mercStateTTL + time.Second)
	rd.set(merc.MercState{Known: true})
	if _, k := h(); k {
		t.Error("no merc is unknown to the guard hook")
	}
}

func TestSidecarLineOncePerMinute(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 90, Level: 8, Class: merc.ClassRogue}}
	w, buf, clk := testWire(t, rd, nil)
	for i := 0; i < 5; i++ {
		w.Tick()
		clk.t = clk.t.Add(10 * time.Second)
	}
	if n := strings.Count(buf.String(), "merc: seen class=271"); n != 1 {
		t.Errorf("%d lines in 50 s:\n%s", n, buf.String())
	}
	clk.t = clk.t.Add(time.Minute)
	w.Tick()
	if n := strings.Count(buf.String(), "merc: seen"); n != 2 {
		t.Errorf("%d lines after a minute", n)
	}
	rd.set(merc.MercState{Known: true})
	clk.t = clk.t.Add(2 * time.Minute)
	before := buf.Len()
	w.Tick()
	if buf.Len() != before {
		t.Error("a line without a merc")
	}
}

type fakeClaimer struct {
	granted bool
	err     error
	calls   int
}

func (f *fakeClaimer) ClaimFreeRogue(context.Context) (bool, error) {
	f.calls++
	return f.granted, f.err
}

func TestClaimDep(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	ctx := context.Background()
	none := &fakeReader{st: merc.MercState{Known: true}}
	if err := claimDep(nil, none, nil, log)(ctx); !errors.Is(err, task.ErrClaimUnsupported) {
		t.Errorf("nil claimer: %v", err)
	}
	c := &fakeClaimer{granted: true}
	seen := &seqReader{seq: []merc.MercState{{Known: true}, {Known: true}, {Known: true, Owned: true, Alive: true}}}
	if err := claimDep(c, seen, nil, log)(ctx); err != nil || c.calls != 1 || !strings.Contains(buf.String(), "merc: claimed free rogue") ||
		!strings.Contains(buf.String(), "merc: claim verified") {
		t.Errorf("grant: %v %d\n%s", err, c.calls, buf.String())
	}
	// a merc already owned: policy declines, the claimer is not even called
	owned := &fakeReader{st: merc.MercState{Known: true, Owned: true, Alive: true}}
	c = &fakeClaimer{}
	if err := claimDep(c, owned, nil, log)(ctx); err != nil || c.calls != 0 {
		t.Errorf("owned: %v %d", err, c.calls)
	}
	// Nightmare: no
	if err := claimDep(c, none, func() merc.Difficulty { return merc.Nightmare }, log)(ctx); err != nil || c.calls != 0 {
		t.Errorf("nightmare: %v %d", err, c.calls)
	}
	// unreadable state still claims; errors pass through
	c = &fakeClaimer{err: errors.New("kashya missing")}
	if err := claimDep(c, nil, nil, log)(ctx); err == nil || c.calls != 1 {
		t.Errorf("error: %v %d", err, c.calls)
	}
}

// seqReader answers the reads in order, then repeats the last one.
type seqReader struct {
	seq []merc.MercState
	i   int
}

func (f *seqReader) State() (merc.MercState, error) {
	st := f.seq[min(f.i, len(f.seq)-1)]
	f.i++
	return st, nil
}
func (f *seqReader) Offers() ([]merc.HireOffer, error) { return nil, errors.New("none") }

// TestClaimDepNotSeen: Gate 2 logged "claimed" and the merc was gone; now the step errors (retry) until the limit.
func TestClaimDepNotSeen(t *testing.T) {
	claimPause = time.Millisecond
	var buf bytes.Buffer
	log := slog.New(slog.NewTextHandler(&buf, nil))
	gone := &fakeReader{st: merc.MercState{Known: true}}
	c := &fakeClaimer{granted: true}
	dep := claimDep(c, gone, nil, log)
	for i := 1; i < claimFailLimit; i++ {
		if err := dep(context.Background()); !errors.Is(err, errClaimUnseen) {
			t.Fatalf("try %d: %v", i, err)
		}
	}
	if err := dep(context.Background()); err != nil {
		t.Errorf("at the limit the step stops retrying: %v", err)
	}
	if !strings.Contains(buf.String(), "merc: claim not seen after 4 tries") || strings.Contains(buf.String(), "claim verified") {
		t.Errorf("log:\n%s", buf.String())
	}
}

// TestGuardMercHook: the dump keeps the merc while the unit table lost it; errors and a missing section are unknown.
func TestGuardMercHook(t *testing.T) {
	clk := &clock{t: time.Unix(1000, 0)}
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Alive: true}}
	h := guardMercHook(rd, clk.now)
	if a, k := h(); !a || !k {
		t.Errorf("owned alive: %v %v", a, k)
	}
	rd.set(merc.MercState{Known: true})
	h()
	if rd.n != 1 {
		t.Errorf("read within the TTL: %d reads", rd.n)
	}
	clk.t = clk.t.Add(mercStateTTL)
	if a, k := h(); a || k {
		t.Errorf("no merc is unknown (the unit table decides): %v %v", a, k)
	}
	rd.set(merc.MercState{Known: true, Owned: true})
	clk.t = clk.t.Add(mercStateTTL)
	if a, k := h(); a || !k {
		t.Errorf("owned dead: %v %v", a, k)
	}
}

func TestMercMount(t *testing.T) {
	latestMerc.Store(nil)
	rec := httptest.NewRecorder()
	mercMount().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	if !strings.Contains(rec.Body.String(), `"enabled":false`) {
		t.Errorf("off: %s", rec.Body)
	}
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Alive: true, LifePct: 55, Level: 7}}
	w, _, _ := testWire(t, rd, nil)
	latestMerc.Store(w)
	defer latestMerc.Store(nil)
	rec = httptest.NewRecorder()
	mercMount().ServeHTTP(rec, httptest.NewRequest("GET", "/", nil))
	for _, want := range []string{`"enabled":true`, `"alive":true`, `"lifePct":55`, `"owned":true`} {
		if !strings.Contains(rec.Body.String(), want) {
			t.Errorf("missing %s in %s", want, rec.Body)
		}
	}
}

type fakeHirer struct {
	rows     []merc.HireOffer
	winErr   error
	paid     int
	hired    []int
	stripped []merc.Slot
	closed   int
}

func (f *fakeHirer) Window(context.Context) ([]merc.HireOffer, error) { return f.rows, f.winErr }
func (f *fakeHirer) Hire(_ context.Context, row int) (int, error) {
	f.hired = append(f.hired, row)
	return f.paid, nil
}
func (f *fakeHirer) Unequip(_ context.Context, s merc.Slot) error {
	f.stripped = append(f.stripped, s)
	return nil
}
func (f *fakeHirer) Close(context.Context) error { f.closed++; return nil }

func lutSnap(gold int) *d2.Snapshot {
	s := townSnap(gold, true)
	s.Player.Area, s.Player.Act = areaLutGholein, 1
	return s
}

var desertRows = []merc.HireOffer{
	{Row: 0, Name: "Pol", Class: merc.ClassDesert, Level: 9, Aura: "Prayer", Price: 360},
	{Row: 1, Name: "Han", Class: merc.ClassDesert, Level: 9, Aura: "Defiance", Price: 400},
	{Row: 2, Name: "Ash", Class: merc.ClassDesert, Level: 9, Aura: "Blessed Aim", Price: 300},
}

func rogueWithGear() merc.MercState {
	return merc.MercState{Known: true, Owned: true, Alive: true, Class: merc.ClassRogue, Level: 12, FreeCells: 30,
		Gear: [merc.NumSlots]*merc.Item{merc.SlotHelm: {Name: "Cap", W: 2, H: 2}, merc.SlotWeapon: {Name: "Bow", W: 2, H: 3}}}
}

func TestHireStepSwapsRogueForDefiance(t *testing.T) {
	rd := &fakeReader{st: rogueWithGear()}
	w, buf, _ := testWire(t, rd, nil)
	h := &fakeHirer{rows: desertRows, paid: 400}
	w.hirer = h
	if got := w.HireStep(context.Background(), lutSnap(5000)); got != "merc hired" {
		t.Fatalf("got %q\n%s", got, buf)
	}
	if len(h.hired) != 1 || h.hired[0] != 1 {
		t.Errorf("hired rows %v, want the Defiance row 1", h.hired)
	}
	if len(h.stripped) != 2 || h.stripped[0] != merc.SlotWeapon {
		t.Errorf("strip before the hire: %v", h.stripped)
	}
	out := buf.String()
	for _, want := range []string{"merc: hire window row=1 name=Han class=338 level=9 aura=Defiance price=400 gold=5000 clvl=10", "merc: strip slot=weapon", "merc: hired class=338 aura=Defiance price=400"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in\n%s", want, out)
		}
	}
	if strings.Index(out, "strip slot") > strings.Index(out, "merc: hired") {
		t.Error("the strip must come before the hire")
	}
	// the visit is final: no second swap, window not read again
	if w.HireStep(context.Background(), lutSnap(5000)) != "" || len(h.hired) != 1 {
		t.Error("second Lut Gholein visit acted")
	}
}

func TestHireStepPoorKeepsRogueAndVisitOpen(t *testing.T) {
	rd := &fakeReader{st: rogueWithGear()}
	w, buf, _ := testWire(t, rd, nil)
	h := &fakeHirer{rows: desertRows, paid: 400}
	w.hirer = h
	// price 400 + 100% of the 310 kit = 710 (reserve was 150%, 865; card mercrevive)
	if got := w.HireStep(context.Background(), lutSnap(709)); !strings.Contains(got, "gold") || len(h.hired) != 0 || len(h.stripped) != 0 {
		t.Fatalf("poor: %q hired %v stripped %v", got, h.hired, h.stripped)
	}
	if h.closed != 1 {
		t.Error("window left open")
	}
	if got := w.HireStep(context.Background(), lutSnap(710)); got != "merc hired" {
		t.Fatalf("affordable later: %q\n%s", got, buf)
	}
	if n := strings.Count(buf.String(), "merc: hire window row="); n != 3 {
		t.Errorf("hire window lines %d, want one per row once", n)
	}
}

func TestHireStepGuards(t *testing.T) {
	rd := &fakeReader{st: rogueWithGear()}
	w, _, _ := testWire(t, rd, nil)
	if w.HireStep(context.Background(), lutSnap(5000)) != "" {
		t.Error("no hirer must do nothing")
	}
	h := &fakeHirer{rows: desertRows}
	w.hirer = h
	if w.HireStep(context.Background(), townSnap(5000, true)) != "" || h.closed != 0 {
		t.Error("outside Lut Gholein")
	}
	h.winErr = errors.New("no window")
	if got := w.HireStep(context.Background(), lutSnap(5000)); !strings.Contains(got, "unreadable") {
		t.Errorf("window error: %q", got)
	}
	w.knobs.ReplaceAtAct2 = false
	h.winErr = nil
	if got := w.HireStep(context.Background(), lutSnap(5000)); !strings.Contains(got, "knob off") || len(h.hired) != 0 {
		t.Errorf("knob off: %q", got)
	}
}

func TestHireStepNoMercHires(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true}}
	w, _, _ := testWire(t, rd, nil)
	h := &fakeHirer{rows: desertRows, paid: 400}
	w.hirer = h
	if got := w.HireStep(context.Background(), lutSnap(5000)); got != "merc hired" || len(h.stripped) != 0 {
		t.Errorf("%q stripped %v", got, h.stripped)
	}
}

func TestReviveCapTwoPerHour(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Level: 10}}
	rev := &fakeReviver{paid: 750}
	w, _, clk := testWire(t, rd, rev)
	for i := 0; i < 3; i++ {
		clk.t = clk.t.Add(10 * time.Minute)
		rd.set(merc.MercState{Known: true, Owned: true, Level: 10})
		w.ReviveStep(context.Background(), townSnap(9000, true))
	}
	if rev.calls != 2 {
		t.Errorf("revives %d, want the cap of 2 an hour", rev.calls)
	}
	clk.t = clk.t.Add(time.Hour)
	rd.set(merc.MercState{Known: true, Owned: true, Level: 10})
	w.ReviveStep(context.Background(), townSnap(9000, true))
	if rev.calls != 3 {
		t.Errorf("revives %d after the hour", rev.calls)
	}
}

func TestReviveLogsEstimateAndReadPrice(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Level: 10}}
	rev := &fakeReviver{paid: 600} // the dialog price differs from the 750 formula
	w, buf, _ := testWire(t, rd, rev)
	w.ReviveStep(context.Background(), townSnap(9000, true))
	if !strings.Contains(buf.String(), "merc: revived price=600 level=10 estimate=750") {
		t.Errorf("log:\n%s", buf)
	}
}

func TestGoldFarmFollowsMerc(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Level: 10}} // dead
	w, buf, _ := testWire(t, rd, nil)
	field := func(gold, lvl int) *d2.Snapshot {
		s := townSnap(gold, true)
		s.Player.Level = lvl
		return s
	}
	if !w.GoldFarm(field(300, 11)) {
		t.Error("dead merc and short gold must farm")
	}
	if !strings.Contains(buf.String(), "merc: gold farm on need=750 gold=300 clvl=11") {
		t.Errorf("log:\n%s", buf)
	}
	if w.GoldFarm(field(5000, 11)) {
		t.Error("rich enough: no farm")
	}
	if w.GoldFarm(field(300, 17)) { // cap 17 = no-merc Andariel floor (job mercrevive2)
		t.Error("clvl 17 is past the cap")
	}
	rd.set(merc.MercState{Known: true, Owned: true, Alive: true})
	w.curAt = time.Time{}
	if w.GoldFarm(field(300, 11)) {
		t.Error("living merc: no farm")
	}
}

func TestGoldFarmBudgetLog(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Level: 10}}
	w, buf, _ := testWire(t, rd, nil)
	clock := time.Date(2026, 11, 4, 12, 0, 0, 0, time.UTC)
	w.now = func() time.Time { return clock }
	snap := townSnap(300, true)
	snap.Player.Level = 11
	if !w.GoldFarm(snap) {
		t.Fatal("farm must arm")
	}
	clock = clock.Add(41 * time.Minute)
	w.curAt = time.Time{}
	if w.GoldFarm(snap) {
		t.Error("budget spent: farm off")
	}
	if !strings.Contains(buf.String(), "merc: gold farm budget minutes=40 cooldown=60") {
		t.Errorf("log:\n%s", buf)
	}
	clock = clock.Add(30 * time.Minute)
	w.curAt = time.Time{}
	if w.GoldFarm(snap) {
		t.Error("cooldown: still off")
	}
}

// ---- card mercwire: wait, first-visit flag on disk, field hooks ----

type snapReader struct {
	d2.StateReader
	snaps []*d2.Snapshot // one per call, the last one repeats
	n     int
}

func (r *snapReader) Snapshot() (*d2.Snapshot, error) {
	s := r.snaps[min(r.n, len(r.snaps)-1)]
	r.n++
	return s, nil
}

type mwClock struct{ slept time.Duration }

func (c *mwClock) Now() time.Time        { return time.Unix(5000, 0).Add(c.slept) }
func (c *mwClock) Sleep(d time.Duration) { c.slept += d }

func fieldSnap(mercAt *d2.Unit) *d2.Snapshot {
	s := &d2.Snapshot{Player: &d2.Player{Area: 3, Act: 0, X: 100, Y: 100, Level: 10}}
	if mercAt != nil {
		s.Monsters = []d2.Unit{*mercAt}
	}
	return s
}

func waitWire(t *testing.T, st merc.MercState, snaps ...*d2.Snapshot) (*mercWire, *bytes.Buffer, *mwClock) {
	t.Helper()
	w, buf, _ := testWire(t, &fakeReader{st: st}, nil)
	ck := &mwClock{}
	w.e.Clock = ck
	w.e.Read = &snapReader{snaps: snaps}
	return w, buf, ck
}

var aliveRogue = merc.MercState{Known: true, Owned: true, Alive: true, Class: merc.ClassRogue}

func TestWaitForMercEndsWhenSheArrives(t *testing.T) {
	far := &d2.Unit{Class: merc.ClassRogue, X: 130, Y: 100, Mode: d2.MonModeStand}
	near := &d2.Unit{Class: merc.ClassRogue, X: 104, Y: 101, Mode: d2.MonModeStand}
	w, buf, ck := waitWire(t, aliveRogue, fieldSnap(nil), fieldSnap(far), fieldSnap(near))
	w.WaitForMerc(context.Background(), "portal")
	if ck.slept != 2*mercWaitStep {
		t.Errorf("waited %v, want two steps", ck.slept)
	}
	out := buf.String()
	if !strings.Contains(out, "merc: waiting for the merc why=portal") || !strings.Contains(out, "merc: wait over waited_ms=1000 near=true") {
		t.Errorf("log:\n%s", out)
	}
}

func TestWaitForMercIsBounded(t *testing.T) {
	w, buf, ck := waitWire(t, aliveRogue, fieldSnap(nil))
	w.WaitForMerc(context.Background(), "waypoint")
	if ck.slept != mercWaitMax {
		t.Errorf("waited %v, want the bound %v", ck.slept, mercWaitMax)
	}
	if !strings.Contains(buf.String(), "wait over waited_ms=6000 near=false") {
		t.Errorf("log:\n%s", buf)
	}
}

func TestWaitForMercSkips(t *testing.T) {
	town := fieldSnap(nil)
	town.Player.Area = 1
	for name, c := range map[string]struct {
		st   merc.MercState
		snap *d2.Snapshot
	}{
		"dead":    {merc.MercState{Known: true, Owned: true}, fieldSnap(nil)},
		"none":    {merc.MercState{Known: true}, fieldSnap(nil)},
		"unknown": {merc.MercState{}, fieldSnap(nil)},
		"town":    {aliveRogue, town},
	} {
		w, buf, ck := waitWire(t, c.st, c.snap)
		w.WaitForMerc(context.Background(), "portal")
		if ck.slept != 0 || buf.Len() != 0 {
			t.Errorf("%s: waited %v, log %q", name, ck.slept, buf)
		}
	}
}

func TestWaitHookOnlyWithTheWiring(t *testing.T) {
	old := bot.MercWait
	defer func() { bot.MercWait = old }()
	bot.MercWait = nil
	var s d2denv.Seams
	installMercSeam(&s)
	if bot.MercWait != nil {
		t.Error("hook set by the seam installer alone (knob off path)")
	}
}

func TestFirstVisitFlagSurvivesRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "merc.json")
	old := mercStatePath
	mercStatePath = path
	defer func() { mercStatePath = old }()
	snap := func(name string) *d2.Snapshot { s := lutSnap(5000); s.Player.Name = name; return s }

	rd := &fakeReader{st: rogueWithGear()}
	w, _, _ := testWire(t, rd, nil)
	h := &fakeHirer{rows: desertRows, paid: 400}
	w.hirer = h
	if got := w.HireStep(context.Background(), snap("Fresha")); got != "merc hired" {
		t.Fatalf("first visit: %q", got)
	}
	// a new bot process: new wiring, same file
	w2, _, _ := testWire(t, &fakeReader{st: rogueWithGear()}, nil)
	h2 := &fakeHirer{rows: desertRows, paid: 400}
	w2.hirer = h2
	if got := w2.HireStep(context.Background(), snap("Fresha")); got != "" || h2.closed != 0 || len(h2.hired) != 0 {
		t.Errorf("after a restart the visit was repeated: %q hired %v", got, h2.hired)
	}
	// per character: another name still has its first visit
	if got := w2.HireStep(context.Background(), snap("Soaky")); got != "merc hired" {
		t.Errorf("other character: %q", got)
	}
	// per difficulty: Nightmare is a new first visit
	nm := snap("Fresha")
	nm.Player.Difficulty = d2.Nightmare
	if got := w2.HireStep(context.Background(), nm); got != "merc hired" {
		t.Errorf("nightmare: %q", got)
	}
	b, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(b), `"Fresha":[0,1]`) {
		t.Errorf("file %q err %v", b, err)
	}
}

func TestFirstVisitDeclineIsPersistedButGoldDeclineIsNot(t *testing.T) {
	path := filepath.Join(t.TempDir(), "merc.json")
	old := mercStatePath
	mercStatePath = path
	defer func() { mercStatePath = old }()
	w, _, _ := testWire(t, &fakeReader{st: rogueWithGear()}, nil)
	h := &fakeHirer{rows: desertRows, paid: 400}
	w.hirer = h
	poor := lutSnap(100)
	poor.Player.Name = "Fresha"
	w.HireStep(context.Background(), poor)
	if _, err := os.Stat(path); err == nil {
		t.Error("a decline for gold left the visit open, it must not be written")
	}
	// a corrupt file is no reason to stop
	os.WriteFile(path, []byte("{nope"), 0o644)
	w2, _, _ := testWire(t, &fakeReader{st: rogueWithGear()}, nil)
	w2.hirer = &fakeHirer{rows: desertRows, paid: 400}
	rich := lutSnap(5000)
	rich.Player.Name = "Fresha"
	if got := w2.HireStep(context.Background(), rich); got != "merc hired" {
		t.Errorf("corrupt state file: %q", got)
	}
}

func TestFighterSeamGetsTheMercStateOnlyWithTheKnob(t *testing.T) {
	cb := &combat.Fighter{}
	s := d2denv.Seams{Fighter: func(*env.Env, *nav.Navigator) bot.Fighter { return fighterAdapter{cb} }}
	installMercSeam(&s)
	dt := env.DefaultTuning()
	tune := &dt
	e := &env.Env{Tune: tune, Log: slog.Default()}
	tune.Merc.Enabled = 0
	s.Fighter(e, nil)
	if cb.MercState != nil {
		t.Error("field rules installed with the knob off")
	}
	tune.Merc.Enabled = 1
	latestMerc.Store(nil)
	s.Fighter(e, nil)
	if cb.MercState == nil {
		t.Fatal("field rules not installed with the knob on")
	}
	if _, err := cb.MercState(); err == nil {
		t.Error("state answered without a running wiring")
	}
	w, _, _ := testWire(t, &fakeReader{st: aliveRogue}, nil)
	latestMerc.Store(w)
	defer latestMerc.Store(nil)
	if st, err := cb.MercState(); err != nil || !st.Alive {
		t.Errorf("state %+v err %v", st, err)
	}
}

// g2e F5: the revive runs before the shop, on pack gold, with a healing potion carried; an empty belt shops first (mercrevive).
func TestReviveFirstBeforeShop(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Level: 10}}
	rev := &fakeReviver{paid: 750}
	w, buf, clk := testWire(t, rd, rev)
	ctx := context.Background()
	if got := w.ReviveFirst(ctx, townSnap(300, true)); got != "" || rev.calls != 0 || buf.Len() != 0 {
		t.Fatalf("poor purse: %q calls %d log %q", got, rev.calls, buf)
	}
	if got := w.ReviveFirst(ctx, townSnap(9000, true)); got != "merc revived" || rev.calls != 1 {
		t.Fatalf("funded, no kit yet: %q calls %d", got, rev.calls)
	}
	clk.t = clk.t.Add(time.Minute)
	rd.set(merc.MercState{Known: true, Owned: true, Alive: true, Level: 10})
	if w.ReviveStep(ctx, townSnap(100, true)) != "" || rev.calls != 1 {
		t.Fatal("an alive merc is not revived again after the shop")
	}
}

// g2e F5: the refusal is written at WARN, once per town visit.
func TestReviveSkipLoggedOncePerVisit(t *testing.T) {
	rd := &fakeReader{st: merc.MercState{Known: true, Owned: true, Level: 10}}
	w, buf, clk := testWire(t, rd, &fakeReviver{paid: 750})
	ctx := context.Background()
	w.ReviveStep(ctx, townSnap(100, true))
	w.ReviveStep(ctx, townSnap(100, true))
	if n := strings.Count(buf.String(), "level=WARN msg=\"merc: revive skipped (gold)"); n != 1 {
		t.Fatalf("%d warn lines in one visit:\n%s", n, buf)
	}
	clk.t = clk.t.Add(5 * time.Minute)
	w.ReviveStep(ctx, townSnap(100, true))
	if n := strings.Count(buf.String(), "level=WARN msg=\"merc: revive skipped (gold)"); n != 2 {
		t.Fatalf("%d warn lines after a new visit:\n%s", n, buf)
	}
}

type fakeBank struct {
	asked []int
	err   error
}

func (f *fakeBank) WithdrawGold(_ context.Context, n int) error {
	f.asked = append(f.asked, n)
	return f.err
}

// g2f seed 17: price 1920, pack held at the keep line (1100) and the rest in the stash. The bar counts both, the revive
// pays from the pack, so the shortfall comes out of the stash first. Two classes: the rule is class-free.
func TestReviveFetchesTheStashPart(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		dead := merc.MercState{Known: true, Owned: true, Level: 16}
		rev := &fakeReviver{paid: 1900}
		w, _, _ := testWire(t, &fakeReader{st: dead}, rev)
		bank := &fakeBank{}
		w.bank = bank
		snap := func(pack, stash int) *d2.Snapshot {
			s := townSnap(pack, true)
			s.Player.Class, s.Player.StashGold = class, stash
			return s
		}
		ctx := context.Background()
		// 1500 in all against a price of 1920 plus the margin: it waits and moves nothing
		if got := w.ReviveStep(ctx, snap(1100, 400)); got != "merc revive skipped: gold" || len(bank.asked) != 0 || rev.calls != 0 {
			t.Fatalf("%v poor: %q asked %v calls %d", class, got, bank.asked, rev.calls)
		}
		// 1930 in all: 820 comes out of the stash, then the revive runs
		if got := w.ReviveStep(ctx, snap(1100, 830)); got != "merc revived" || len(bank.asked) != 1 || bank.asked[0] != 820 || rev.calls != 1 {
			t.Fatalf("%v funded: %q asked %v calls %d", class, got, bank.asked, rev.calls)
		}
	}
}

// Without a bank only the pack counts before the shop, as before; with one the stash counts too.
func TestReviveFirstCountsTheStashOnlyWithABank(t *testing.T) {
	dead := merc.MercState{Known: true, Owned: true, Level: 16}
	s := townSnap(1100, true)
	s.Player.StashGold = 900
	rev := &fakeReviver{paid: 1900}
	w, _, _ := testWire(t, &fakeReader{st: dead}, rev)
	if got := w.ReviveFirst(context.Background(), s); got != "" || rev.calls != 0 {
		t.Fatalf("no bank: %q calls %d", got, rev.calls)
	}
	bank := &fakeBank{}
	w.bank = bank
	if got := w.ReviveFirst(context.Background(), s); got != "merc revived" || rev.calls != 1 || len(bank.asked) != 1 || bank.asked[0] != 820 {
		t.Fatalf("bank: %q calls %d asked %v", got, rev.calls, bank.asked)
	}
}

// A failed withdrawal is logged and the revive is still tried.
func TestReviveWithdrawFailureStillTries(t *testing.T) {
	dead := merc.MercState{Known: true, Owned: true, Level: 16}
	rev := &fakeReviver{err: errors.New("purse")}
	w, buf, _ := testWire(t, &fakeReader{st: dead}, rev)
	w.bank = &fakeBank{err: errors.New("stash closed")}
	s := townSnap(1100, true)
	s.Player.StashGold = 900
	if got := w.ReviveStep(context.Background(), s); got != "merc revive failed" || rev.calls != 1 || !strings.Contains(buf.String(), "stash withdrawal for the revive failed") {
		t.Fatalf("%q calls %d log %s", got, rev.calls, buf)
	}
}

type tpTown struct {
	bot.Townsman
	tp, bought int
}

func (f *tpTown) TPCount(*d2.Snapshot) int { return f.tp }
func (f *tpTown) TakeTPBought() int        { return f.bought }

// g2f: the runner finds tp and tpBuy by a type assertion on its Townsman; the merc wrapper hid them (tp 0 on every line).
func TestMercTownForwardsTPDiag(t *testing.T) {
	var d bot.Townsman = &mercTown{Townsman: &tpTown{tp: 3, bought: 2}}
	diag, ok := d.(bot.TPDiag)
	if !ok || diag.TPCount(nil) != 3 || diag.TakeTPBought() != 2 {
		t.Fatalf("forwarding broken: %v", ok)
	}
	d = &mercTown{Townsman: nil}
	if diag := d.(bot.TPDiag); diag.TPCount(nil) != 0 || diag.TakeTPBought() != 0 {
		t.Error("an inner town without the diagnostics must read 0")
	}
}

type probeReviver struct {
	fakeReviver
	quote, probes int
	err           error
}

func (p *probeReviver) Probe(context.Context) (int, error) { p.probes++; return p.quote, p.err }

// g2f seed 11: the formula said 480 for a merc whose menu quoted 120. A purse short of the estimate reads the quote
// once and revives when it covers the real price; the quote is kept while the merc stays dead.
func TestReviveReadsTheRealQuote(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		dead := merc.MercState{Known: true, Owned: true, Level: 16} // estimate 1920
		rev := &probeReviver{quote: 1270}
		rev.paid = 1270
		w, buf, clk := testWire(t, &fakeReader{st: dead}, rev)
		snap := func(gold int) *d2.Snapshot {
			s := townSnap(gold, true)
			s.Player.Class = class
			return s
		}
		ctx := context.Background()
		if got := w.ReviveStep(ctx, snap(1000)); got != "merc revive skipped: gold" || rev.probes != 1 || rev.calls != 0 {
			t.Fatalf("%v poor: %q probes %d calls %d", class, got, rev.probes, rev.calls)
		}
		if !strings.Contains(buf.String(), "need=1270") {
			t.Errorf("the refusal must carry the quote bar:\n%s", buf)
		}
		clk.t = clk.t.Add(5 * time.Minute)
		if got := w.ReviveStep(ctx, snap(1285)); got != "merc revived" || rev.probes != 1 || rev.calls != 1 {
			t.Fatalf("%v funded: %q probes %d calls %d", class, got, rev.probes, rev.calls)
		}
		if w.quote != 0 {
			t.Error("the quote must be dropped after the revive")
		}
	}
}

// A hopeless purse does not walk to the NPC, and an unreadable quote falls back to the estimate.
func TestReviveProbeSkippedOrFailed(t *testing.T) {
	dead := merc.MercState{Known: true, Owned: true, Level: 16}
	rev := &probeReviver{quote: 1270}
	w, _, _ := testWire(t, &fakeReader{st: dead}, rev)
	if got := w.ReviveStep(context.Background(), townSnap(50, true)); got != "merc revive skipped: gold" || rev.probes != 0 {
		t.Fatalf("poor: %q probes %d", got, rev.probes)
	}
	rev2 := &probeReviver{err: errors.New("no hireling")}
	w2, buf, _ := testWire(t, &fakeReader{st: dead}, rev2)
	if got := w2.ReviveStep(context.Background(), townSnap(1500, true)); got != "merc revive skipped: gold" || rev2.probes != 1 || !strings.Contains(buf.String(), "need=1920") {
		t.Fatalf("failed probe: %q probes %d\n%s", got, rev2.probes, buf)
	}
}

// SME rule (user 2026-11-06): the revive keeps back 0 gold below clvl 13 and 50 from clvl 13, whatever the kit costs.
// Price 1920 (merc clvl 16). Two classes: the band is class-free.
func TestReviveReserveByClvl(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, tc := range []struct {
			clvl, gold int
			revive     bool
		}{{12, 1930, true}, {12, 1919, false}, {14, 1930, false}, {14, 1975, true}, {13, 1969, false}} {
			rev := &fakeReviver{paid: 1920}
			w, buf, _ := testWire(t, &fakeReader{st: merc.MercState{Known: true, Owned: true, Level: 16}}, rev)
			s := townSnap(tc.gold, true)
			s.Player.Class, s.Player.Level = class, tc.clvl
			got := w.ReviveStep(context.Background(), s)
			if (got == "merc revived") != tc.revive || (rev.calls == 1) != tc.revive {
				t.Errorf("%v clvl %d gold %d: %q calls %d", class, tc.clvl, tc.gold, got, rev.calls)
			}
			if tc.clvl == 14 && !tc.revive && !strings.Contains(buf.String(), "need=1970") {
				t.Errorf("clvl 14 refusal must show need=1970:\n%s", buf)
			}
		}
	}
}

// A town that records the order of its chores and the revive, and lets a chore change the purse.
type orderTown struct {
	bot.Townsman
	order *[]string
	plans []bot.ChorePlan
	sold  func()
}

func (o *orderTown) Chores(_ context.Context, p bot.ChorePlan) (bot.ChoreReport, error) {
	o.plans = append(o.plans, p)
	switch {
	case p.Sell && !p.Potions:
		*o.order = append(*o.order, "sell")
		if o.sold != nil {
			o.sold()
		}
	default:
		*o.order = append(*o.order, "shop")
	}
	return bot.ChoreReport{}, nil
}

type orderReviver struct {
	fakeReviver
	order *[]string
	after func()
}

func (r *orderReviver) Revive(ctx context.Context) (int, error) {
	*r.order = append(*r.order, "revive")
	if r.after != nil {
		defer r.after()
	}
	return r.fakeReviver.Revive(ctx)
}

type mercSnapReader struct {
	d2.StateReader
	s *d2.Snapshot
}

func (r *mercSnapReader) Snapshot() (*d2.Snapshot, error) { return r.s, nil }

// The revive comes before the potion purchase in the same visit; when the purse is short the junk is sold first and the
// revive asked again, still before any potion is bought. Quote unreadable: nothing is bought for the merc and the next
// visit tries again.
func TestReviveBeforePotionsSellsJunkFirst(t *testing.T) {
	dead := merc.MercState{Known: true, Owned: true, Level: 10} // 750, clvl 12 reserve 0
	var order []string
	rev := &orderReviver{order: &order}
	rev.paid = 750
	rdr := &fakeReader{st: dead}
	rev.after = func() { rdr.set(merc.MercState{Known: true, Owned: true, Alive: true, Level: 10}) }
	w, _, _ := testWire(t, rdr, rev)
	snap := townSnap(400, true)
	snap.Player.Level = 12
	w.e.Read = &mercSnapReader{s: snap}
	town := &orderTown{order: &order, sold: func() { snap.Player.Gold = 800 }}
	mt := &mercTown{Townsman: town, w: w}
	rep, err := mt.Chores(context.Background(), bot.ChorePlan{Sell: true, Potions: true, Repair: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(order, ","); got != "sell,revive,shop" {
		t.Errorf("order %s, want sell,revive,shop", got)
	}
	if len(rep.Done) == 0 || rep.Done[0] != "merc revived" {
		t.Errorf("report %v", rep.Done)
	}
	// a funded purse revives first, no extra sell round
	order = nil
	rev2 := &orderReviver{order: &order}
	rev2.paid = 750
	rdr2 := &fakeReader{st: dead}
	rev2.after = func() { rdr2.set(merc.MercState{Known: true, Owned: true, Alive: true, Level: 10}) }
	w2, _, _ := testWire(t, rdr2, rev2)
	snap2 := townSnap(900, true)
	snap2.Player.Level = 12
	w2.e.Read = &mercSnapReader{s: snap2}
	mt2 := &mercTown{Townsman: &orderTown{order: &order}, w: w2}
	if _, err := mt2.Chores(context.Background(), bot.ChorePlan{Sell: true, Potions: true}); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(order, ","); got != "revive,shop" {
		t.Errorf("funded order %s, want revive,shop", got)
	}
}

func TestReviveQuoteUnreadableLogsOncePerVisitAndRetries(t *testing.T) {
	dead := merc.MercState{Known: true, Owned: true, Level: 16} // estimate 1920
	rev := &probeReviver{err: errors.New("no hireling")}
	w, buf, clk := testWire(t, &fakeReader{st: dead}, rev)
	ctx := context.Background()
	w.ReviveFirst(ctx, townSnap(1500, true))
	w.ReviveStep(ctx, townSnap(1500, true))
	if n := strings.Count(buf.String(), "revive quote unreadable"); n != 1 || rev.probes != 2 {
		t.Fatalf("unreadable lines %d probes %d (one line per visit, probe each time):\n%s", n, rev.probes, buf)
	}
	clk.t = clk.t.Add(5 * time.Minute)
	w.ReviveStep(ctx, townSnap(1500, true))
	if n := strings.Count(buf.String(), "revive quote unreadable"); n != 2 || rev.probes != 3 {
		t.Fatalf("next visit: lines %d probes %d:\n%s", n, rev.probes, buf)
	}
}

// An old tuning file that names the retired cap still loads; the new band knobs can be set.
func TestRetiredReviveReserveMaxStillLoads(t *testing.T) {
	tune := env.DefaultTuning()
	if err := bot.ApplyOverrides(&tune, map[string]float64{"merc.reviveReserveMax": 10, "merc.reviveReserveHigh": 80}); err != nil {
		t.Fatal(err)
	}
	if tune.Merc.ReviveReserveHigh != 80 {
		t.Errorf("high band %d", tune.Merc.ReviveReserveHigh)
	}
}

// g2g M1: the hourly revive cap and death window count the env clock, which runs about 4x wall time under lockstep.
func TestMercWindowsFollowTheEnvClock(t *testing.T) {
	dt := env.DefaultTuning()
	ck := &mwClock{}
	e := &env.Env{Tune: &dt, Clock: ck, Log: slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil))}
	w := newMercWire(e, &fakeReader{}, nil, fakeCat{})
	w.revives = []time.Time{w.now(), w.now()}
	w.deaths = []time.Time{w.now()}
	ck.slept = 59 * time.Minute // 59 game minutes; the wall clock has moved a few milliseconds
	s := townSnap(5000, true)
	if c := mercCtx(w, s); c.RevivesLastHour != 2 || c.DeathsLastHour != 1 {
		t.Fatalf("inside the game hour: %+v", c)
	}
	ck.slept = 61 * time.Minute
	if c := mercCtx(w, s); c.RevivesLastHour != 0 || c.DeathsLastHour != 0 {
		t.Fatalf("after the game hour the cap must release: %+v", c)
	}
	e.Clock = nil
	if d := time.Since(w.now()); d < 0 || d > time.Minute {
		t.Errorf("no env clock must fall back to wall time, off by %v", d)
	}
}

// g2g M2: after one real quote every consumer uses it (farm bar, revive bar), not the level formula; a new merc level
// or an old read goes back to the probe.
func TestQuoteReplacesTheFormulaEverywhere(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		dead := merc.MercState{Known: true, Owned: true, Level: 16} // formula 1920
		rev := &probeReviver{quote: 120}
		rev.paid = 120
		rd := &fakeReader{st: dead}
		w, buf, clk := testWire(t, rd, rev)
		snap := func(gold int) *d2.Snapshot {
			s := townSnap(gold, true)
			s.Player.Class, s.Player.Level = class, 16
			return s
		}
		if !w.GoldFarm(snap(100)) || !strings.Contains(buf.String(), "need=1970") {
			t.Fatalf("%v: before any quote the formula arms the farm:\n%s", class, buf)
		}
		ctx := context.Background()
		if got := w.ReviveStep(ctx, snap(100)); got != "merc revive skipped: gold" || rev.probes != 1 {
			t.Fatalf("%v: %q probes %d", class, got, rev.probes)
		}
		if got := w.ReviveStep(ctx, snap(300)); got != "merc revived" {
			t.Fatalf("%v: %q", class, got)
		}
		// dies again at the same level: no new probe, the farm bar is the quote plus the reserve
		rd.set(dead)
		w.curAt = time.Time{}
		clk.t = clk.t.Add(10 * time.Minute)
		if w.GoldFarm(snap(300)) {
			t.Errorf("%v: 300 gold covers the quote 120, no farm", class)
		}
		if got := w.ReviveStep(ctx, snap(300)); got != "merc revived" || rev.probes != 1 {
			t.Fatalf("%v: second revive %q probes %d", class, got, rev.probes)
		}
		// a new level makes the old quote void, as does age
		rd.set(merc.MercState{Known: true, Owned: true, Level: 17})
		w.curAt = time.Time{}
		w.mu.Lock()
		q := w.quoteFor(17)
		w.mu.Unlock()
		if q != 0 {
			t.Errorf("%v: quote for a new level = %d", class, q)
		}
		w.mu.Lock()
		w.last.level = 16
		fresh := w.quoteFor(16)
		clk.t = clk.t.Add(5 * time.Hour)
		stale := w.quoteFor(16)
		w.mu.Unlock()
		if fresh != 120 || stale != 0 {
			t.Errorf("%v: fresh %d stale %d", class, fresh, stale)
		}
	}
}

type manaCat struct{ fakeCat }

func (c manaCat) Item(class int) (bot.ItemMeta, bool) {
	if class == 3 {
		return bot.ItemMeta{Name: "Mana Potion", Potion: bot.PotionMana}, true
	}
	return c.fakeCat.Item(class)
}

// TASK-174: the hire kit follows the belt plan: a mixed belt still needs healing, an all-mana belt hires without it.
func TestKitFollowsBeltPlan(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for _, c := range []struct {
			name     string
			plan     int
			wantPots bool
			wantFull bool
			withHeal bool
		}{
			{"default, no healing", 0, false, false, false},
			{"balanced, no healing", 1, false, false, false},
			{"all mana, no healing", 3, true, true, false},
			{"all healing, no healing", 2, false, false, false},
			{"balanced, healing", 1, true, true, true},
		} {
			w, _, _ := testWire(t, nil, nil)
			w.cat = manaCat{}
			w.e.Tune.Resource.BeltPlan = c.plan
			s := townSnap(0, false)
			s.Player.Class = class
			s.Inventory.Items = []d2.Unit{{Class: 1, Item: &d2.Item{Where: d2.ContInventory}}}
			for i := 0; i < w.e.Tune.Resource.KitMinMP; i++ {
				s.Inventory.Items = append(s.Inventory.Items, d2.Unit{Class: 3, Item: &d2.Item{Where: d2.ContBelt}})
			}
			for i := 0; c.withHeal && i < w.e.Tune.Resource.KitMinHP; i++ {
				s.Inventory.Items = append(s.Inventory.Items, d2.Unit{Class: 2, Item: &d2.Item{Where: d2.ContBelt}})
			}
			if tp, pots := w.kit(s); !tp || pots != c.wantPots {
				t.Errorf("%v %s: kit tp %v potions %v, want potions %v", class, c.name, tp, pots, c.wantPots)
			}
			if got := w.kitFull(s); got != c.wantFull {
				t.Errorf("%v %s: kitFull %v, want %v", class, c.name, got, c.wantFull)
			}
		}
	}
}

// Under the player-only gear policy the bot never strips the merc: a replacing hire is refused while he wears gear
// (or while what he wears is unread), and goes ahead once the player has taken it off. Two hero classes, two mercs.
func TestHireStepRefusesWhileMercWearsGear(t *testing.T) {
	desertDef := merc.MercState{Known: true, Owned: true, Alive: true, Class: merc.ClassDesert, Aura: "Prayer", Level: 12, FreeCells: 30, GearKnown: true,
		Gear: [merc.NumSlots]*merc.Item{merc.SlotWeapon: {Name: "Spear", W: 2, H: 4}}}
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		for name, st := range map[string]merc.MercState{"rogue worn": withKnown(rogueWithGear()), "desert worn": desertDef, "rogue unread": rogueWithGear()} {
			rd := &fakeReader{st: st}
			w, buf, _ := testWire(t, rd, nil)
			w.e.Tune.Resource.GearDecisions = 0 // the default: the player decides
			h := &fakeHirer{rows: desertRows, paid: 400}
			w.hirer = h
			s := lutSnap(5000)
			s.Player.Class = class
			got := w.HireStep(context.Background(), s)
			if !strings.Contains(got, "unequip your merc first") || len(h.hired) != 0 || len(h.stripped) != 0 {
				t.Fatalf("%v %s: %q hired %v stripped %v", class, name, got, h.hired, h.stripped)
			}
			if h.closed != 1 {
				t.Errorf("%v %s: window left open", class, name)
			}
			// asked again while still geared: one log line, window not read again
			if again := w.HireStep(context.Background(), s); again != "" || h.closed != 1 {
				t.Errorf("%v %s: second visit %q closed %d", class, name, again, h.closed)
			}
			if n := strings.Count(buf.String(), "replace skipped"); n != 1 {
				t.Errorf("%v %s: %d refusal lines\n%s", class, name, n, buf)
			}
			// the player takes it off: the hire goes ahead, nothing stripped by the bot
			bare := st
			bare.Gear, bare.GearKnown = [merc.NumSlots]*merc.Item{}, true
			rd.set(bare)
			w.curAt = time.Time{} // drop the cached state
			if got := w.HireStep(context.Background(), s); got != "merc hired" || len(h.stripped) != 0 {
				t.Fatalf("%v %s: after unequip %q stripped %v\n%s", class, name, got, h.stripped, buf)
			}
		}
	}
}

func withKnown(s merc.MercState) merc.MercState { s.GearKnown = true; return s }

// mercrevive B1: an empty belt shops first (survival), then the revive runs on gold alone, for a caster and a melee class.
func TestEmptyBeltShopsBeforeTheRevive(t *testing.T) {
	for _, class := range []d2.Class{d2.Sorceress, d2.Paladin} {
		dead := merc.MercState{Known: true, Owned: true, Level: 10} // 750
		var order []string
		rev := &orderReviver{order: &order}
		rev.paid = 750
		rdr := &fakeReader{st: dead}
		rev.after = func() { rdr.set(merc.MercState{Known: true, Owned: true, Alive: true, Level: 10}) }
		w, _, _ := testWire(t, rdr, rev)
		snap := townSnap(1500, false) // no potion, no portal
		snap.Player.Class = class
		w.e.Read = &mercSnapReader{s: snap}
		mt := &mercTown{Townsman: &orderTown{order: &order}, w: w}
		if _, err := mt.Chores(context.Background(), bot.ChorePlan{Potions: true}); err != nil {
			t.Fatal(err)
		}
		if got := strings.Join(order, ","); got != "shop,revive" {
			t.Errorf("%v: order %s, want shop,revive", class, got)
		}
	}
}
