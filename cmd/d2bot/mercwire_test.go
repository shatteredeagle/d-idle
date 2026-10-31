package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
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
	tune := &dt // merc.enabled defaults to 0
	if tune.Merc.Enabled != 0 {
		t.Fatal("merc.enabled must default to 0")
	}
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
	if err := claimDep(c, none, nil, log)(ctx); err != nil || c.calls != 1 || !strings.Contains(buf.String(), "merc: claimed free rogue") {
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
