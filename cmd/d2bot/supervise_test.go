package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"d2lod-bot/internal/bot"
)

func testSup(codes []int, max int) (*supervisor, *[]int, *time.Time) {
	now := time.Unix(1000, 0)
	var attempts []int
	s := &supervisor{
		log: slog.New(slog.NewTextHandler(io.Discard, nil)), max: max, window: time.Hour, minRun: 10 * time.Minute,
		base: 5 * time.Second, ceil: 20 * time.Second, now: func() time.Time { return now },
		run: func(ctx context.Context, a int) int {
			attempts = append(attempts, a)
			now = now.Add(time.Second)
			return codes[min(a, len(codes)-1)]
		},
		sleep: func(ctx context.Context, d time.Duration) { now = now.Add(d) },
	}
	return s, &attempts, &now
}

func TestSuperviseRestartsUntilClean(t *testing.T) {
	s, att, _ := testSup([]int{2, 1, 0}, 10)
	if c := s.loop(context.Background()); c != 0 || len(*att) != 3 {
		t.Fatalf("code %d attempts %v", c, *att)
	}
}

func TestSuperviseCap(t *testing.T) {
	s, att, _ := testSup([]int{2}, 3)
	if c := s.loop(context.Background()); c != 2 || len(*att) != 4 {
		t.Fatalf("code %d attempts %v", c, *att)
	}
}

func TestSuperviseStopsOnCancel(t *testing.T) {
	s, att, _ := testSup([]int{1}, 10)
	ctx, cancel := context.WithCancel(context.Background())
	s.run = func(ctx context.Context, a int) int { cancel(); return 1 }
	if c := s.loop(ctx); c != 1 || len(*att) != 0 {
		t.Fatalf("code %d", c)
	}
}

func TestChildArgs(t *testing.T) {
	in := []string{"--managed", "--supervise", "--run-for", "5h", "--supervise-max=3", "--char", "Soaky", "--clean-start"}
	got := childArgs(in, true, 90*time.Minute, false)
	want := []string{"--managed", "--char", "Soaky", "--run-for", "1h30m0s", "--clean-start"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	if got := childArgs(in, false, 0, false); !reflect.DeepEqual(got, []string{"--managed", "--char", "Soaky"}) {
		t.Fatalf("%v", got)
	}
}

func TestChildArgsRestartDropsGoals(t *testing.T) {
	in := []string{"--managed", "--goals", "campaign", "--autostart", "--supervise"}
	got := childArgs(in, true, 0, true)
	if want := []string{"--managed", "--autostart", "--clean-start"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	got = childArgs([]string{"--managed", "--goals=campaign"}, true, 0, true)
	if want := []string{"--managed", "--autostart", "--clean-start"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	if got := childArgs(in, false, 0, true); !reflect.DeepEqual(got, []string{"--managed", "--goals", "campaign", "--autostart"}) {
		t.Fatalf("first start keeps goals: %v", got)
	}
}

func TestChildEnv(t *testing.T) {
	env := []string{"A=1", "D2BOT_SUPERVISE=1", "D2BOT_GOALS=campaign", "D2BOT_SUPERVISED=0"}
	got := childEnv(env, false)
	if want := []string{"A=1", "D2BOT_GOALS=campaign", "D2BOT_SUPERVISED=1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("%v", got)
	}
	if got := childEnv(env, true); !reflect.DeepEqual(got, []string{"A=1", "D2BOT_SUPERVISED=1"}) {
		t.Fatalf("%v", got)
	}
}

func TestSuperviseDoesNotRestartDeliberateFatal(t *testing.T) {
	s, att, _ := testSup([]int{exitFatal}, 10)
	if c := s.loop(context.Background()); c != exitFatal || len(*att) != 1 {
		t.Fatalf("code %d attempts %v", c, *att)
	}
	s, att, _ = testSup([]int{exitStorm, 0}, 10)
	if c := s.loop(context.Background()); c != 0 || len(*att) != 2 {
		t.Fatalf("a panic storm must be restarted: code %d attempts %v", c, *att)
	}
}

func TestHaltError(t *testing.T) {
	var ee *exitError
	if !errors.As(haltError(errors.New("corrupt")), &ee) || ee.code != exitFatal {
		t.Fatal("fatal must be exitFatal")
	}
	if !errors.As(haltError(bot.ErrPanicStorm), &ee) || ee.code != exitStorm {
		t.Fatal("storm must be exitStorm")
	}
	if !errors.As(haltError(nil), &ee) || ee.code != exitStorm {
		t.Fatal("nil must be a crash")
	}
}

func TestGoSafeReportsGiveUp(t *testing.T) {
	goBackoff = time.Microsecond
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	if goSafe(context.Background(), log, "t", func() { panic("x") }) {
		t.Fatal("must report false after giving up")
	}
	if !goSafe(context.Background(), log, "t", func() {}) {
		t.Fatal("a normal return is true")
	}
}
