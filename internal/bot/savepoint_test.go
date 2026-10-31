package bot

import (
	"context"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// spRig is a character in town, in the game since the clock's start, ready for a save point.
func spRig(t *testing.T) (*rig, control) {
	t.Helper()
	g := newRig(t)
	g.r.d.Cycler = fakeCycler{g.w}
	g.w.set(func(w *world) { w.state, w.area, w.life, w.maxL = d2.StateInGame, 1, 100, 100 })
	g.observeAt(g.clock.Now()) // game entry
	return g, control{gen: g.r.gen}
}

// menuObserver makes the menu leg of NewGame visible to the observer: fakeCycler holds it for 5 ms only, which a loaded
// machine can let pass unseen, and then the new entry is never noticed (savedExp stays stale).
type menuObserver struct {
	fakeCycler
	g *rig
}

func (m menuObserver) NewGame(ctx context.Context, d string) error {
	err := m.fakeCycler.NewGame(ctx, d)
	m.g.w.set(func(w *world) { w.state = d2.StateMenu })
	m.g.observeAt(m.g.clock.Now())
	m.g.w.set(func(w *world) { w.state = d2.StateInGame })
	return err
}

func spSnap(g *rig) *d2.Snapshot { s, _ := g.w.snapshot(); return s }

func TestSavePointDueAfterInterval(t *testing.T) {
	g, c := spRig(t)
	if ok, why := g.r.savePointDue(spSnap(g), c, g.clock.Now().Add(19*time.Minute)); ok || why != "too soon" {
		t.Fatalf("19 min: %v %q", ok, why)
	}
	if ok, why := g.r.savePointDue(spSnap(g), c, g.clock.Now().Add(21*time.Minute)); !ok {
		t.Fatalf("21 min: %q", why)
	}
}

func TestSavePointOffAndGuards(t *testing.T) {
	late := func(g *rig) time.Time { return g.clock.Now().Add(time.Hour) }
	cases := []struct {
		name, why string
		setup     func(g *rig)
	}{
		{"off", "off", func(g *rig) { g.e.Tune.Bot.SavePointMin = 0 }},
		{"field", "not in town", func(g *rig) { g.w.set(func(w *world) { w.area = 2 }) }},
		{"trip", "trip in progress", func(g *rig) { g.r.loop.trip = &trip{} }},
		{"low life", "low life", func(g *rig) { g.w.set(func(w *world) { w.life = 40 }) }},
		{"backoff", "backoff", func(g *rig) { g.r.loop.saveNext = late(g).Add(time.Minute) }},
		{"quest step", "mid quest step", func(g *rig) { g.r.loop.task.pseudo = map[string]int{"staff": 1} }},
		{"recent death", "recent death", func(g *rig) {
			g.r.obMu.Lock()
			g.r.ob.lastDeathAt = late(g).Add(-time.Minute)
			g.r.obMu.Unlock()
		}},
	}
	for _, tc := range cases {
		g, c := spRig(t)
		tc.setup(g)
		if tc.name == "quest step" {
			c.spec.camp = &task.Campaign{}
		}
		ok, why := g.r.savePointDue(spSnap(g), c, late(g))
		if ok || why != tc.why {
			t.Errorf("%s: due=%v why=%q, want %q", tc.name, ok, why, tc.why)
		}
	}
	g, c := spRig(t)
	g.r.loop.rec = &recovery{}
	if ok, why := g.r.savePointDue(spSnap(g), c, late(g)); ok || !strings.Contains(why, "death") {
		t.Errorf("recovery: %v %q", ok, why)
	}
	// an old death does not block
	g, c = spRig(t)
	g.r.obMu.Lock()
	g.r.ob.lastDeathAt = g.clock.Now().Add(-time.Hour)
	g.r.obMu.Unlock()
	if ok, why := g.r.savePointDue(spSnap(g), c, late(g)); !ok {
		t.Errorf("old death blocked: %q", why)
	}
}

func TestSavePointStepReentersAndIsPlanned(t *testing.T) {
	g, c := spRig(t)
	g.r.d.Cycler = menuObserver{fakeCycler{g.w}, g}
	g.clock.Sleep(25 * time.Minute)
	g.w.set(func(w *world) { w.exp = 5000 })
	if !g.r.savePointStep(context.Background(), spSnap(g), c) {
		t.Fatal("no save point")
	}
	if g.w.count("newgame") != 1 {
		t.Fatalf("log: %s", g.w.log())
	}
	g.observeAt(g.clock.Now()) // the observer sees the new entry
	g.r.obMu.Lock()
	games, planned, risk := g.r.ob.newGames, g.r.ob.plannedGames, g.r.ob.xpAtRiskLocked(nil)
	g.r.obMu.Unlock()
	if planned != 1 || games < 1 {
		t.Fatalf("games %d planned %d", games, planned)
	}
	if risk != 0 {
		t.Fatalf("xp at risk %d after the save point, want 0", risk)
	}
	if len(g.r.loop.reentries) != 0 {
		t.Fatal("a planned re-entry must not count toward the re-entry pause")
	}
	// just saved: not due again
	if ok, _ := g.r.savePointDue(spSnap(g), c, g.clock.Now().Add(time.Minute)); ok {
		t.Fatal("due again right after a save point")
	}
}

func TestSavePointFailureBacksOffWithoutLadder(t *testing.T) {
	g, c := spRig(t)
	g.r.d.Cycler = failCycler{}
	g.clock.Sleep(25 * time.Minute)
	if !g.r.savePointStep(context.Background(), spSnap(g), c) {
		t.Fatal("not attempted")
	}
	if g.r.loop.failures != 0 || g.r.loop.stuckStage != 0 {
		t.Fatalf("failsafe involved: failures %d stage %d", g.r.loop.failures, g.r.loop.stuckStage)
	}
	if ok, why := g.r.savePointDue(spSnap(g), c, g.clock.Now()); ok || why != "backoff" {
		t.Fatalf("%v %q", ok, why)
	}
}

type failCycler struct{}

func (failCycler) EnterGame(context.Context, string) error { return errBoom }
func (failCycler) NewGame(context.Context, string) error   { return errBoom }

// Any other game entry (recycle, farm NewGame, fresh-game leveling, crash re-entry) is a save too: the timer restarts.
func TestOtherGameEntriesCountAsSavePoints(t *testing.T) {
	g, c := spRig(t)
	g.clock.Sleep(30 * time.Minute)
	g.w.set(func(w *world) { w.state = d2.StateMenu })
	g.observeAt(g.clock.Now())
	g.w.set(func(w *world) { w.state = d2.StateInGame })
	g.observeAt(g.clock.Now())
	if ok, why := g.r.savePointDue(spSnap(g), c, g.clock.Now().Add(time.Minute)); ok || why != "too soon" {
		t.Fatalf("%v %q", ok, why)
	}
}
