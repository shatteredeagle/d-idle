package combat

import (
	"testing"
	"time"

	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/env"
)

func playerWith(life, maxLife int, states ...int) *d2.Player {
	p := &d2.Player{Life: life, MaxLife: maxLife}
	for _, s := range states {
		p.States[s>>5] |= 1 << (uint(s) & 31)
	}
	return p
}

func TestCurseDangerAndRetreat(t *testing.T) {
	ids := d2.DefaultCurseIDs
	if got := CurseDanger(playerWith(100, 100), DangerNone); got != DangerNone {
		t.Fatalf("no curse: %d", got)
	}
	for _, id := range []int{ids.Amplify, ids.Decrepify, ids.LowerResist, ids.Weaken, ids.IronMaiden} {
		if got := CurseDanger(playerWith(100, 100, id), DangerNone); got != DangerElite {
			t.Fatalf("state %d: %d", id, got)
		}
	}
	if got := CurseDanger(playerWith(100, 100, ids.Amplify), DangerDeadly); got != DangerDeadly {
		t.Fatalf("must not lower: %d", got)
	}
	if !CurseRetreat(playerWith(50, 100, ids.Amplify), 60) || CurseRetreat(playerWith(70, 100, ids.Amplify), 60) {
		t.Fatal("amplify threshold")
	}
	if CurseRetreat(playerWith(10, 100, ids.Weaken), 60) || CurseRetreat(playerWith(10, 100, ids.Amplify), 0) || CurseRetreat(nil, 60) {
		t.Fatal("only amplify/iron maiden, knob 0 off, nil safe")
	}
}

func TestResolveCurseIDs(t *testing.T) {
	ids := d2.ResolveCurseIDs(func(n string) (int, bool) { return 77, n == "decrepify" })
	if ids.Decrepify != 77 || ids.Amplify != d2.DefaultCurseIDs.Amplify {
		t.Fatalf("%+v", ids)
	}
}

func TestSidestepMissiles(t *testing.T) {
	missileKinds = func(c int) d2.MissileKind {
		if c == 1 {
			return d2.MissileFireWall
		}
		return d2.MissileOther
	}
	defer func() { missileKinds = nil }()
	f, s := newDodgeFighter(1)
	snap := &d2.Snapshot{Player: &d2.Player{ID: 1, X: 100, Y: 100, Life: 100, MaxLife: 100}, Missiles: []d2.Unit{
		{ID: 9, Class: 1, OwnerID: 50, OwnerType: 1, X: 102, Y: 100}, // fire wall next to us
		{ID: 10, Class: 2, OwnerID: 50, OwnerType: 1, X: 98, Y: 100}, // harmless kind
		{ID: 11, Class: 1, OwnerID: 1, X: 100, Y: 102},               // our own
	}}
	n := 0
	moved, err := f.sidestepMissiles(t.Context(), snap, &n)
	if err != nil || !moved || n != 1 || len(s.moves) != 1 {
		t.Fatalf("moved=%v err=%v n=%d moves=%v", moved, err, n, s.moves)
	}
	if s.moves[0].X >= 100 {
		t.Fatalf("should step away from x=102: %v", s.moves[0])
	}
	// gap and cap
	f.lastDodge = f.now()
	if m, _ := f.sidestepMissiles(t.Context(), snap, &n); m {
		t.Fatal("second dodge inside the gap")
	}
	f2, _ := newDodgeFighter(0)
	n = 0
	if m, _ := f2.sidestepMissiles(t.Context(), snap, &n); m {
		t.Fatal("knob off must not dodge")
	}
	f3, _ := newDodgeFighter(1)
	n = maxMissileDodges
	if m, _ := f3.sidestepMissiles(t.Context(), snap, &n); m {
		t.Fatal("cap")
	}
	missileKinds = nil
	f4, _ := newDodgeFighter(1)
	n = 0
	if m, _ := f4.sidestepMissiles(t.Context(), snap, &n); m {
		t.Fatal("unknown kinds must not dodge")
	}
}

func newDodgeFighter(knob int) (*Fighter, *sim) {
	h := newHarness(nil, func(t *env.Tuning) { t.Combat.MissileDodge = knob })
	return h.f, h.s
}

func TestFightTelemetry(t *testing.T) {
	var ft FightTelemetry
	ft.Area, ft.Diff, ft.Danger = 3, 1, 2
	for _, l := range []int{100, 80, 90, 40, 60} {
		ft.Sample(&d2.Player{Life: l, MaxLife: 100})
	}
	ft.Dodges = 2
	got := ft.Finish(ClearResult{Kills: 7, Potions: 2, Elapsed: 12 * time.Second})
	want := "fight area=3 diff=1 danger=2 kills=7 damage=70 minlife=40 potions=2 dodges=2 sec=12 casts=0 kites=0 approaches=0 novas=0 forced=0 castsPerSec=0.00"
	if got != want {
		t.Fatalf("\n got %s\nwant %s", got, want)
	}
}
