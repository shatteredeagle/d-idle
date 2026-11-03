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
	if got := w.ReviveStep(ctx, townSnap(5000, true)); got != "merc revived" || rev.calls != 1 {
		t.Fatalf("got %q calls %d", got, rev.calls)
	}
	if st := w.Status(); st["deathsLastHour"] != 1 || st["revivesLastHour"] != 1 {
		t.Errorf("status %+v", st)
	}
	if !strings.Contains(buf.String(), "merc: revived price=750 level=10") || !strings.Contains(buf.String(), "merc: died") {
		t.Errorf("log:\n%s", buf.String())
	}
	// poor: below 2 x 750
	clk.t = clk.t.Add(10 * time.Second)
	if got := w.ReviveStep(ctx, townSnap(1000, true)); !strings.Contains(got, "gold") || rev.calls != 1 {
		t.Errorf("poor: %q", got)
	}
	// no kit
	if got := w.ReviveStep(ctx, townSnap(9000, false)); !strings.Contains(got, "kit") {
		t.Errorf("kit: %q", got)
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
	// price 400 + 150% of the 310 kit = 865
	if got := w.HireStep(context.Background(), lutSnap(864)); !strings.Contains(got, "gold") || len(h.hired) != 0 || len(h.stripped) != 0 {
		t.Fatalf("poor: %q hired %v stripped %v", got, h.hired, h.stripped)
	}
	if h.closed != 1 {
		t.Error("window left open")
	}
	if got := w.HireStep(context.Background(), lutSnap(865)); got != "merc hired" {
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
	if !strings.Contains(buf.String(), "merc: gold farm on need=1215 gold=300 clvl=11") {
		t.Errorf("log:\n%s", buf)
	}
	if w.GoldFarm(field(5000, 11)) {
		t.Error("rich enough: no farm")
	}
	if w.GoldFarm(field(300, 16)) {
		t.Error("clvl 16 is past the cap")
	}
	rd.set(merc.MercState{Known: true, Owned: true, Alive: true})
	w.curAt = time.Time{}
	if w.GoldFarm(field(300, 11)) {
		t.Error("living merc: no farm")
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
