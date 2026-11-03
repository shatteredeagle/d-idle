package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/combat"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/input"
)

type portalReader struct {
	d2.StateReader
	snap *d2.Snapshot
}

func (r portalReader) Snapshot() (*d2.Snapshot, error)    { return r.snap, nil }
func (r portalReader) UnitStats(uint32) (d2.Stats, error) { return d2.Stats{}, nil } // qty absent: empty tome

// F4: with an empty tome and no scroll the portal verb reports no source without a click (In is nil: a click panics)
// and the caller's fallback treats it as no portal at once.
func TestD2DPortalEmptyTomeNoSourceNoClick(t *testing.T) {
	cat := combat.MapCatalog{Items: map[int]combat.ItemClass{1: {Portal: true, Tome: true}}}
	s := &d2.Snapshot{Player: &d2.Player{}, Inventory: &d2.Inventory{Items: []d2.Unit{
		{ID: 5, Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContInventory}},
	}}}
	e := &env.Env{Read: portalReader{snap: s}, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	err := d2dPortal(e, nil, cat)(t.Context())
	if !errors.Is(err, combat.ErrNoPortal) {
		t.Fatalf("err %v, want ErrNoPortal", err)
	}
}

// ---- FIXPORTAL2: a scripted d2d world (cast, busy drop, cut cast) ----

type portalWorld struct {
	d2.StateReader
	input.Controller
	now     time.Time
	mode    int
	busyFor time.Duration // mode stays busy until this much clock has passed
	scrolls []uint32      // loose scroll ids in the pack
	// onClick decides what a click does to the world (nil: spend the scroll and open the portal)
	onClick func(w *portalWorld, id uint32)
	portal  bool
	clicks  []uint32
	itemID  uint32
	keys    int
}

func (w *portalWorld) Now() time.Time { return w.now }
func (w *portalWorld) Sleep(d time.Duration) {
	w.now = w.now.Add(d)
	if w.busyFor > 0 {
		if w.busyFor -= d; w.busyFor <= 0 {
			w.busyFor, w.mode = 0, 1
		}
	}
}
func (w *portalWorld) Key(string) error                   { w.keys++; return nil }
func (w *portalWorld) MoveMouse(x, y int) error           { return nil }
func (w *portalWorld) UnitStats(uint32) (d2.Stats, error) { return d2.Stats{}, nil }
func (w *portalWorld) Click(_ input.Button, x, y int, _ ...input.Mod) error {
	w.clicks = append(w.clicks, w.itemID)
	if w.onClick != nil {
		w.onClick(w, w.itemID)
		return nil
	}
	w.spend(w.itemID)
	w.portal = true
	return nil
}
func (w *portalWorld) spend(id uint32) {
	for i, s := range w.scrolls {
		if s == id {
			w.scrolls = append(w.scrolls[:i:i], w.scrolls[i+1:]...)
			return
		}
	}
}
func (w *portalWorld) Snapshot() (*d2.Snapshot, error) {
	inv := &d2.Inventory{}
	for _, id := range w.scrolls {
		inv.Items = append(inv.Items, d2.Unit{ID: id, Addr: id, Type: d2.TypeItem, Class: 1, Item: &d2.Item{Where: d2.ContInventory}})
	}
	s := &d2.Snapshot{Player: &d2.Player{Mode: w.mode, X: 100, Y: 100}, Inventory: inv}
	if w.portal {
		s.Objects = []d2.Unit{{Class: 59, X: 101, Y: 100}}
	}
	return s, nil
}

func newPortalWorld(scrolls ...uint32) (*portalWorld, *env.Env, func(context.Context) error) {
	w := &portalWorld{scrolls: scrolls, mode: 1, now: time.Unix(0, 0)}
	e := &env.Env{Read: w, In: w, Clock: w, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	cat := combat.MapCatalog{Items: map[int]combat.ItemClass{1: {Portal: true}}}
	itemAt := func(_ context.Context, id int) (int, int, error) { w.itemID = uint32(id); return 10, 10, nil }
	return w, e, d2dPortalWith(e, itemAt, cat)
}

// FIXPORTAL2 (1): an already-cancelled context never clicks and the error names the cause.
func TestD2DPortalDeadContextNoClick(t *testing.T) {
	w, _, run := newPortalWorld(7)
	ctx, cancel := context.WithCancelCause(t.Context())
	cancel(errors.New("step deadline"))
	err := run(ctx)
	if err == nil || !strings.Contains(err.Error(), "cancelled") || !strings.Contains(err.Error(), "step deadline") {
		t.Fatalf("err %v, want a cancelled error with the cause", err)
	}
	if len(w.clicks) != 0 || w.keys != 0 {
		t.Fatalf("clicks %v keys %d, want none", w.clicks, w.keys)
	}
}

// FIXPORTAL2 (2): a player in hit recovery gets no click until idle.
func TestD2DPortalBusyWaitsForIdle(t *testing.T) {
	w, _, run := newPortalWorld(7)
	w.mode, w.busyFor = 4, 600*time.Millisecond // GH
	var busyAtClick int
	w.onClick = func(w *portalWorld, id uint32) {
		busyAtClick = w.mode
		w.spend(id)
		w.portal = true
	}
	if err := run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(w.clicks) != 1 || busyAtClick != 1 {
		t.Fatalf("clicks %v mode at click %d, want one click while idle", w.clicks, busyAtClick)
	}
}

// FIXPORTAL2 (2): a player that stays busy past the bounded wait is never clicked, and the error says busy.
func TestD2DPortalStaysBusyNoClick(t *testing.T) {
	w, _, run := newPortalWorld(7)
	w.mode = 7 // attacking forever
	err := run(t.Context())
	if err == nil || !strings.Contains(err.Error(), "busy") {
		t.Fatalf("err %v, want busy", err)
	}
	if len(w.clicks) != 0 {
		t.Fatalf("clicks %v, want none", w.clicks)
	}
}

// FIXPORTAL2 (3): a cast cut by a hit spends the scroll without a portal: the verb does not call that success and the
// retry takes the next scroll.
func TestD2DPortalCutCastRetriesNextSource(t *testing.T) {
	w, _, run := newPortalWorld(7, 8)
	w.onClick = func(w *portalWorld, id uint32) {
		w.spend(id)
		w.portal = id == 8 // the first cast is cut, the second completes
	}
	if err := run(t.Context()); err != nil {
		t.Fatal(err)
	}
	if len(w.clicks) != 2 || w.clicks[0] != 7 || w.clicks[1] != 8 {
		t.Fatalf("clicks %v, want scroll 7 then scroll 8", w.clicks)
	}
}

// FIXPORTAL2 (4): the cut cast spent the last source: the re-read finds nothing and no further click is made.
func TestD2DPortalEmptyAfterRereadNoRetry(t *testing.T) {
	w, _, run := newPortalWorld(7)
	w.onClick = func(w *portalWorld, id uint32) { w.spend(id) }
	err := run(t.Context())
	if !errors.Is(err, combat.ErrNoPortal) {
		t.Fatalf("err %v, want ErrNoPortal", err)
	}
	if len(w.clicks) != 1 {
		t.Fatalf("clicks %v, want exactly one", w.clicks)
	}
}

// BELTFIX3 (3), Gate 1 20261102 BELT A: a safety drink cancelled the cast after the click. The error says the source may
// be spent (not "nothing clicked"), and the next call looks for the portal of the cut cast before it reads another charge.
func TestPortalCutAfterClickReportsSpent(t *testing.T) {
	w, _, run := newPortalWorld(7, 8)
	ctx, cancel := context.WithCancelCause(t.Context())
	w.onClick = func(w *portalWorld, id uint32) { w.spend(id); cancel(errors.New("bot: safety drink-life")) }
	err := run(ctx)
	if err == nil || !strings.Contains(err.Error(), "source maybe spent") || strings.Contains(err.Error(), "nothing clicked") {
		t.Fatalf("err %v, want a cut-cast error", err)
	}
	if len(w.clicks) != 1 {
		t.Fatalf("clicks %v, want one", w.clicks)
	}
	w.onClick = nil
	w.portal = true // the cut cast's portal opens after all
	if err := run(t.Context()); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(w.clicks) != 1 || len(w.scrolls) != 1 {
		t.Fatalf("clicks %v scrolls %v: the second call must reuse the open portal, not cast again", w.clicks, w.scrolls)
	}
}

// codefix F8: a call with an already dead context must not forget the cut cast; the next live call still finds its portal.
func TestPortalCutMemorySurvivesDeadContext(t *testing.T) {
	w, _, run := newPortalWorld(7, 8)
	ctx, cancel := context.WithCancelCause(t.Context())
	w.onClick = func(w *portalWorld, id uint32) { w.spend(id); cancel(errors.New("bot: safety drink-life")) }
	_ = run(ctx)
	w.onClick = nil
	dead, stop := context.WithCancel(t.Context())
	stop()
	_ = run(dead)
	w.portal = true
	if err := run(t.Context()); err != nil {
		t.Fatalf("live call: %v", err)
	}
	if len(w.clicks) != 1 {
		t.Fatalf("clicks %v: the live call cast again", w.clicks)
	}
}
