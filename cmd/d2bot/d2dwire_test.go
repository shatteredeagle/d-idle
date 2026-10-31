package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2denv"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/launch"
)

type stalledC struct{ frame int }

func (s *stalledC) Clock(context.Context) (d2dctl.ClockInfo, error) {
	return d2dctl.ClockInfo{}, errors.New("no clock verb")
}
func (s *stalledC) Info(context.Context) (d2dctl.Info, error) {
	return d2dctl.Info{Frame: s.frame}, nil
}

// A d2d that answers ping but whose frame counter stopped is a hang for the monitor (failsafe ladder), not "alive".
func TestD2DHealthStalledBeatIsHung(t *testing.T) {
	c := d2dctl.New("/nonexistent", d2dctl.Options{})
	p := &d2denv.Proc{C: c}
	sc := &stalledC{frame: 10}
	mon := d2dHealth(p, &d2denv.Beat{C: sc}, func() bool { return false }, nil, slog.Default())
	// the process probe needs a live channel: replace it, the heartbeat is what is under test
	mon.P.Alive = func() (int, bool) { return 1, true }
	t0 := time.Now()
	if ev := mon.Step(t0); ev != launch.EventUp {
		t.Fatalf("first step: %v", ev)
	}
	sc.frame = 11
	if ev := mon.Step(t0.Add(time.Second)); ev != launch.EventNone {
		t.Fatalf("advancing frames: %v", ev)
	}
	if ev := mon.Step(t0.Add(time.Minute)); ev != launch.EventNone {
		t.Fatalf("stalled less than HangAfter: %v", ev)
	}
	if ev := mon.Step(t0.Add(time.Second + mon.HangAfter + time.Second)); ev != launch.EventHung {
		t.Fatalf("stalled beat must be a hang: %v", ev)
	}
}

func TestOwnerMarker(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "d2d-i1.sock")
	if ownedSocket(sock) {
		t.Fatal("unmarked socket reported as owned")
	}
	markOwned(sock, 123)
	if !ownedSocket(sock) {
		t.Fatal("marker not written")
	}
	os.Remove(ownerFile(sock))
	if ownedSocket(sock) {
		t.Fatal("marker not removed")
	}
}

// Managed mode never quits (--clean-start) or drives a d2d it did not start: a stranger's socket is refused.
func TestManagedRefusesStrangerSocket(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "stranger.sock")
	srv, err := fake.New(sock, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	for _, clean := range []bool{false, true} {
		o := options{managed: true, cleanStart: clean, data: dir, instance: "x", d2d: d2dOptions{backend: "d2d", sock: sock, bin: "/bin/false", class: 1}}
		err := runD2D(context.Background(), o, slog.New(slog.DiscardHandler))
		if err == nil {
			t.Fatalf("cleanStart=%v: stranger socket accepted", clean)
		} else {
			t.Logf("cleanStart=%v: %v", clean, err)
		}
		if srv.Quit() {
			t.Fatal("a d2d this bot did not start was quit")
		}
	}
}

// The stack's hookups, one by one, over the fake d2d.
func TestBuildD2DStackHookups(t *testing.T) {
	dir := t.TempDir()
	sock := filepath.Join(dir, "d.sock")
	srv, err := fake.New(sock, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	db, err := data.Load(synthTables(t))
	if err != nil {
		t.Fatal(err)
	}
	tune := env.DefaultTuning()
	be, err := d2denv.Build(context.Background(), d2denv.Config{ID: "t", Log: slog.New(slog.DiscardHandler), Tune: &tune, Data: db, Sock: sock})
	if err != nil {
		t.Fatal(err)
	}
	defer be.Close()
	nv := be.Navigator()
	st := buildD2DStack(context.Background(), options{char: "Fake"}, be, nv, db, bot.Config{}, &runnerRef{}, slog.New(slog.DiscardHandler))
	switch {
	case st.task == nil || st.task.ClaimMerc == nil || st.task.Boss == nil || st.task.Farm == nil || st.task.Dialog == nil:
		t.Errorf("task deps incomplete: %+v", st.task)
	case st.leveler == nil || st.gear == nil || st.quester == nil:
		t.Error("leveler, gear and quester must be wired")
	case st.rig.Fighter.Leave == nil:
		t.Error("combat Leave hook missing")
	case nv.Yield == nil || nv.Escape == nil:
		t.Error("nav hooks (Yield, Escape) missing")
	case st.prepare == nil:
		t.Error("prepare missing")
	}
	if st.rig.Fighter.Portal == nil {
		t.Error("d2d portal hook missing (the default needs the inventory flag d2d does not report)")
	} else if err := st.rig.Fighter.Portal(context.Background()); err == nil {
		t.Error("portal cast without a scroll in the pack must fail")
	}
	f, tn, _ := be.Behaviors(st.seams(), nv)
	if _, ok := f.(fighterAdapter); !ok {
		t.Errorf("fighter seam is %T", f)
	}
	if _, ok := tn.(d2dTown); !ok {
		t.Errorf("town seam is %T", tn)
	}
	// merc wrapper goes around it when the knob is on
	tune.Merc.Enabled = 1
	s := st.seams()
	installMercSeam(&s)
	if _, tn, _ = be.Behaviors(s, nv); tn == nil {
		t.Fatal("no town")
	} else if _, ok := tn.(d2dTown); ok {
		t.Error("merc knob on must wrap the town seam")
	}
}

// A missed ping (a long lockstep step on a loaded box) is not an exit; only a silence longer than the grace is.
func TestD2DHealthToleratesMissedPings(t *testing.T) {
	p := &d2denv.Proc{C: d2dctl.New("/nonexistent", d2dctl.Options{})}
	ok := true
	mon := d2dHealthWith(p, &d2denv.Beat{C: &stalledC{}}, nil, nil, func(context.Context) bool { return ok }, 100*time.Millisecond, slog.Default())
	if _, up := mon.P.Alive(); !up {
		t.Fatal("answering d2d reported gone")
	}
	ok = false
	if _, up := mon.P.Alive(); !up {
		t.Fatal("one missed ping must not be an exit")
	}
	time.Sleep(150 * time.Millisecond)
	if _, up := mon.P.Alive(); up {
		t.Fatal("silence beyond the grace must be an exit")
	}
}

// Under lockstep a flat game counter is not a hang (the bot drives the clock): the heartbeat keeps moving.
func TestD2DHealthLockstepNeverHangs(t *testing.T) {
	p := &d2denv.Proc{C: d2dctl.New("/nonexistent", d2dctl.Options{})}
	mon := d2dHealth(p, &d2denv.Beat{C: &stalledC{frame: 10}}, func() bool { return true }, nil, slog.Default())
	mon.P.Alive = func() (int, bool) { return 1, true }
	t0 := time.Now()
	mon.Step(t0)
	for i := 1; i <= 5; i++ {
		time.Sleep(time.Millisecond)
		if ev := mon.Step(t0.Add(time.Duration(i) * time.Minute)); ev != launch.EventNone {
			t.Fatalf("step %d: %v", i, ev)
		}
	}
}

func TestLateEmitForwardsAfterBind(t *testing.T) {
	var l lateEmit
	l.send(goals.Event{Type: goals.EvKill}) // before bind: dropped, no panic
	var got []goals.EventType
	l.bind(func(ev goals.Event) { got = append(got, ev.Type) })
	l.send(goals.Event{Type: goals.EvKill})
	if len(got) != 1 || got[0] != goals.EvKill {
		t.Fatalf("got %v", got)
	}
}

func TestD2DHealthRecentStepIsProofOfLife(t *testing.T) {
	p := &d2denv.Proc{}
	step := time.Now()
	mon := d2dHealthWith(p, &d2denv.Beat{C: &stalledC{}}, nil, func() time.Time { return step }, func(context.Context) bool { return false }, 100*time.Millisecond, slog.Default())
	step = time.Now()
	if _, ok := mon.P.Alive(); !ok {
		t.Fatal("a step that just finished must count as alive while pings starve")
	}
}
