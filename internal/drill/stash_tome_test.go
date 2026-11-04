package drill

import (
	"context"
	"testing"
	"time"
)

// stashGame is a lab whose chores withdraw like the fixed bot (or, with buggy, like the old needWithdraw).
type stashGame struct {
	fakeGame
	purse, keep, kit int
	rejuv            bool
	buggy            bool
	belt             int
	trips, rounds    int
}

func (s *stashGame) TView(context.Context) (TownView, error) {
	p := map[string]int{}
	if s.rejuv {
		p["rv1"] = 1
	}
	return TownView{InTown: true, Gold: s.purse, Stash: s.stash, Belt: s.belt, Pack: p}, nil
}
func (s *stashGame) SetGold(_ context.Context, r *Run, n int) error {
	s.purse = n
	_, err := r.Cheat(context.Background(), "debug stat 14", "gold")
	return err
}
func (s *stashGame) SetStashGold(_ context.Context, r *Run, n int) error {
	s.stash = n
	_, err := r.Cheat(context.Background(), "debug stat 15", "stash")
	return err
}
func (s *stashGame) DrainBelt(context.Context) error { s.belt = 0; return nil }
func (s *stashGame) GiveGround(_ context.Context, r *Run, code string) error {
	s.rejuv = true
	_, err := r.Cheat(context.Background(), "debug give "+code, "rejuvenation for the drill")
	return err
}
func (s *stashGame) PickupRaw(context.Context) (int, error) { return 1, nil }
func (s *stashGame) KeepGold() int                          { return s.keep }
func (s *stashGame) KitCost() int                           { return s.kit }
func (s *stashGame) RejuvCode() string                      { return "rv1" }
func (s *stashGame) Chores(_ context.Context, plan string) ChoreInfo {
	need := s.purse < s.keep && (s.buggy || s.purse < s.kit) && !s.rejuv
	if need && s.stash > 0 {
		w := s.keep - s.purse
		if w > s.stash {
			w = s.stash
		}
		s.purse, s.stash = s.purse+w, s.stash-w
	}
	if s.purse >= 20 {
		s.purse -= 20
		s.belt += 4
		return ChoreInfo{Plan: plan, Spent: 20, Bought: map[string]int{"hp1": 4}}
	}
	return ChoreInfo{Plan: plan}
}

// tripGame adds the Runner loop of scenario 3.
type tripGame struct{ *stashGame }

func (s tripGame) RunnerRounds(context.Context, int) (int, int, error) { return s.trips, s.rounds, nil }

func runStash(s *stashGame, id string) Outcome {
	s.fakeGame.GameLab = s
	s.w = &fakeWorld{}
	return runStashOn(s, tripGame{s}, id)
}

func runStashOn(s *stashGame, l GameLab, id string) Outcome {
	for _, d := range DStash(l, time.Minute) {
		if d.ID == id {
			if d.Do == nil {
				return Outcome{Verdict: Blocked, Reason: "needs"}
			}
			return d.Do(context.Background(), &Run{W: s.w, Drill: id})
		}
	}
	panic("no drill " + id)
}

func TestStashDrills(t *testing.T) {
	newG := func() *stashGame { return &stashGame{keep: 250, kit: 150} }
	if o := runStash(newG(), "DS.1"); o.Verdict != Pass {
		t.Fatalf("DS.1 %v %s", o.Verdict, o.Reason)
	}
	if o := runStash(newG(), "DS.2"); o.Verdict != Pass {
		t.Fatalf("DS.2 %v %s", o.Verdict, o.Reason)
	}
	if o := runStash(newG(), "DS.4"); o.Verdict != Pass {
		t.Fatalf("DS.4 %v %s", o.Verdict, o.Reason)
	}
	// the old needWithdraw fired for a purse that paid the kit
	if o := runStash(&stashGame{keep: 700, kit: 150, buggy: true}, "DS.1"); o.Verdict != Fail {
		t.Fatalf("DS.1 passed with the old withdrawal: %v", o.Verdict)
	}
	if o := runStash(newG(), "DS.3"); o.Verdict != Pass {
		t.Fatalf("DS.3 %v %s", o.Verdict, o.Reason)
	}
	if o := runStash(&stashGame{keep: 250, kit: 150, trips: 3, rounds: 3}, "DS.3"); o.Verdict != Fail {
		t.Fatalf("DS.3 latch missing passed: %v", o.Verdict)
	}
}

func TestStashBlockedWithoutLab(t *testing.T) {
	if d := DStash(&fakeGame{}, time.Minute); len(d) != 1 || len(d[0].Needs) == 0 {
		t.Fatal("expected one blocked drill")
	}
	d := DStash(&stashGame{}, time.Minute) // StashLab without TripLab
	if len(d) != 4 || len(d[2].Needs) == 0 || d[2].Do != nil {
		t.Fatal("DS.3 must be BLOCKED without a TripLab")
	}
}

type tomeGame struct {
	fakeGame
	charges    int
	dropAt     int // look number at which the tome reads empty, 0 never
	looks      int
	log        []string
	logOnDrop  string
	worldReady bool
}

func (g *tomeGame) PView(context.Context) (PortalView, error) {
	g.looks++
	q := g.charges
	if g.dropAt > 0 && g.looks > g.dropAt {
		q = 0
		if g.logOnDrop != "" {
			g.log = []string{g.logOnDrop}
		}
	}
	return PortalView{Tome: q}, nil
}
func (g *tomeGame) Tick(context.Context, int) error { return nil }
func (g *tomeGame) SetGold(_ context.Context, r *Run, n int) error {
	_, err := r.Cheat(context.Background(), "debug stat 14", "gold")
	return err
}
func (g *tomeGame) GiveGround(_ context.Context, r *Run, c string) error {
	_, err := r.Cheat(context.Background(), "debug give "+c, "junk")
	return err
}
func (g *tomeGame) Goto(context.Context, *Run, int) error { return nil }
func (g *tomeGame) Chores(context.Context, string) ChoreInfo {
	return ChoreInfo{}
}
func (g *tomeGame) JunkCode() string      { return "cap" }
func (g *tomeGame) LogMark() int          { return 0 }
func (g *tomeGame) LogSince(int) []string { return g.log }

func runTome(g *tomeGame, id string) Outcome {
	g.fakeGame.GameLab = g
	g.w = &fakeWorld{}
	for _, d := range DTome(g, time.Minute) {
		if d.ID == id {
			if d.Do == nil {
				return Outcome{Verdict: Blocked, Reason: "needs " + d.Needs[0]}
			}
			return d.Do(context.Background(), &Run{W: g.w, Drill: id})
		}
	}
	panic("no drill")
}

func TestTomeDrill(t *testing.T) {
	if o := runTome(&tomeGame{charges: 20}, "TM.a"); o.Verdict != Pass {
		t.Fatalf("%v %s", o.Verdict, o.Reason)
	}
	if o := runTome(&tomeGame{charges: 5}, "TM.a"); o.Verdict != Blocked {
		t.Fatalf("short tome: %v", o.Verdict)
	}
	// a qty dropout with no warning is the fork case
	if o := runTome(&tomeGame{charges: 20, dropAt: 10}, "TM.a"); o.Verdict != Fail {
		t.Fatalf("dropout passed: %v", o.Verdict)
	}
	o := runTome(&tomeGame{charges: 20, dropAt: 10, logOnDrop: "WARN dump items failed: x"}, "TM.a")
	if o.Verdict != Fail || o.Reason[:11] != "empty read " {
		t.Fatalf("%v %s", o.Verdict, o.Reason)
	}
	for _, id := range []string{"TM.b", "TM.c"} {
		if o := runTome(&tomeGame{charges: 20}, id); o.Verdict != Blocked {
			t.Fatalf("%s %v", id, o.Verdict)
		}
	}
}
