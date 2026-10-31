package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"
)

// superviseProcess is --supervise: it runs this same binary as a child (without --supervise, with --clean-start on every
// restart) and starts it again when it exits non-zero, with a growing wait and a cap of restarts per hour. A clean exit
// (0: --run-for reached, SIGTERM shutdown) ends supervision. SIGTERM to the supervisor is forwarded to the child, which
// shuts down gracefully (walks home, saves, stops the game with --stop-game).
func superviseProcess(ctx context.Context, o options, log *slog.Logger, args []string) int {
	self, err := os.Executable()
	if err != nil {
		log.Error("supervise: cannot find own binary", "err", err)
		return 2
	}
	return newSupervisor(o, log, args, self).loop(ctx)
}

// newSupervisor wires the restart policy to real child processes of self (a test points self at a fake child).
func newSupervisor(o options, log *slog.Logger, args []string, self string) *supervisor {
	start := time.Now()
	return &supervisor{
		log: log, max: o.superviseMax, window: time.Hour, minRun: 10 * time.Minute,
		base: 5 * time.Second, ceil: 2 * time.Minute, now: time.Now,
		run: func(ctx context.Context, attempt int) int {
			var left time.Duration
			if o.runFor > 0 {
				if left = o.runFor - time.Since(start); left < time.Second {
					return 0
				}
			}
			return runChild(ctx, self, childArgs(args, attempt > 0, left, o.goalSpec != ""), childEnv(os.Environ(), attempt > 0), log)
		},
		sleep: func(ctx context.Context, d time.Duration) {
			select {
			case <-ctx.Done():
			case <-time.After(d):
			}
		},
	}
}

// supervisor is the restart policy, separated from process handling so it can be tested.
type supervisor struct {
	log    *slog.Logger
	max    int           // restarts allowed within window
	window time.Duration //
	minRun time.Duration // a child that lived this long resets the backoff
	base   time.Duration
	ceil   time.Duration
	now    func() time.Time
	run    func(ctx context.Context, attempt int) int // runs one child, returns its exit code
	sleep  func(ctx context.Context, d time.Duration)
}

func (s *supervisor) loop(ctx context.Context) int {
	var restarts []time.Time
	wait := s.base
	for attempt := 0; ; attempt++ {
		t0 := s.now()
		code := s.run(ctx, attempt)
		if code == exitFatal {
			s.log.Error("supervise: d2bot ended with a deliberate fatal, not restarting", "attempt", attempt)
			return code
		}
		if code == 0 || ctx.Err() != nil {
			s.log.Info("supervise: child ended, not restarting", "code", code, "attempt", attempt)
			return code
		}
		if s.now().Sub(t0) >= s.minRun {
			wait = s.base
		}
		keep := restarts[:0]
		for _, t := range restarts {
			if s.now().Sub(t) < s.window {
				keep = append(keep, t)
			}
		}
		restarts = append(keep, s.now())
		if len(restarts) > s.max {
			s.log.Error("supervise: too many restarts, giving up", "max", s.max, "window", s.window, "code", code)
			return code
		}
		s.log.Error("supervise: d2bot exited, restarting", "code", code, "restart", attempt+1, "in", wait)
		s.sleep(ctx, wait)
		wait = min(wait*2, s.ceil)
	}
}

// Exit codes of d2bot. A deliberate fatal (corrupt character, unusable config: restarting cannot help) is not restarted;
// everything else non-zero (1 error, 2 parse error or Go panic, 4 panic storm, dead goroutine, stuck loop or heap guard, 128+n signal) is.
const (
	exitFatal = 3
	exitStorm = 4
)

// exitError makes main exit with a specific code.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// superviseMarker is set in the child's environment: a child never becomes a supervisor, whatever D2BOT_SUPERVISE says.
const superviseMarker = "D2BOT_SUPERVISED"

// childEnv is the child's environment: no D2BOT_SUPERVISE (H3), the marker set, and on a restart no D2BOT_GOALS (H2:
// the goals are in the goals file already).
func childEnv(env []string, restart bool) []string {
	var out []string
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, "D2BOT_SUPERVISE="), strings.HasPrefix(kv, superviseMarker+"="):
			continue
		case restart && (strings.HasPrefix(kv, "D2BOT_GOALS=") || strings.HasPrefix(kv, testCrashEnv+"=")): // the drill crashes once
			continue
		}
		out = append(out, kv)
	}
	return append(out, superviseMarker+"=1")
}

// childArgs strips --supervise* and --run-for from args, re-adds --run-for with what is left, and on a restart adds
// --clean-start, drops --goals (they are in the goals file; passing them again would queue a second campaign) and keeps
// playing with --autostart when goals were given.
func childArgs(args []string, restart bool, runFor time.Duration, hadGoals bool) []string {
	var out []string
	for i := 0; i < len(args); i++ {
		name := strings.TrimLeft(args[i], "-")
		if k := strings.IndexByte(name, '='); k >= 0 {
			name = name[:k]
		}
		hasVal := !strings.Contains(args[i], "=")
		switch name {
		case "supervise", "clean-start":
			continue
		case "goals":
			if restart {
				if hasVal {
					i++
				}
				continue
			}
		case "test-crash-after":
			if restart {
				if hasVal {
					i++
				}
				continue
			}
		case "supervise-max", "run-for":
			if hasVal {
				i++
			}
			continue
		}
		out = append(out, args[i])
	}
	if runFor > 0 {
		out = append(out, "--run-for", runFor.Round(time.Second).String())
	}
	if restart {
		if hadGoals && !slices.Contains(out, "--autostart") {
			out = append(out, "--autostart")
		}
		out = append(out, "--clean-start")
	}
	return out
}

// runChild runs one child to its end and returns its exit code (128+signal when killed). When ctx ends it forwards SIGTERM
// and kills the child after the shutdown grace.
func runChild(ctx context.Context, self string, args, env []string, log *slog.Logger) int {
	cmd := exec.Command(self, args...)
	cmd.Env = env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = nil, os.Stdout, os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
	if err := startFromLockedThread(cmd); err != nil {
		log.Error("supervise: cannot start the child", "err", err)
		return 127
	}
	log.Info("supervise: child started", "pid", cmd.Process.Pid)
	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	var err error
	select {
	case err = <-done:
	case <-ctx.Done():
		_ = cmd.Process.Signal(syscall.SIGTERM)
		select {
		case err = <-done:
		case <-time.After(100 * time.Second): // above the child's own worst case (bot.ShutdownWorstCase)
			_ = cmd.Process.Kill()
			err = <-done
		}
	}
	if err == nil {
		return 0
	}
	var ee *exec.ExitError
	if errors.As(err, &ee) {
		if ws, ok := ee.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
			return 128 + int(ws.Signal())
		}
		return ee.ExitCode()
	}
	log.Error("supervise: wait failed", "err", fmt.Sprint(err))
	return 1
}

// underFleet reports whether d2fleet started this d2bot (ProcBots sets D2BOT_FLEET=1; the parent name is a second check).
func underFleet() bool {
	if os.Getenv("D2BOT_FLEET") == "1" {
		return true
	}
	b, err := os.ReadFile(fmt.Sprintf("/proc/%d/comm", os.Getppid()))
	return err == nil && strings.TrimSpace(string(b)) == "d2fleet"
}

// Pdeathsig fires when the *thread* that forked the child dies, and Go retires threads freely, so children are started
// from one locked thread that lives as long as the process (same pattern as launch.Spawner). A SIGKILLed supervisor
// then takes its d2bot (and with --stop-game its game) down with it instead of orphaning them.
var (
	spawnOnce sync.Once
	spawnReqs = make(chan spawnReq)
)

type spawnReq struct {
	cmd  *exec.Cmd
	done chan error
}

func startFromLockedThread(cmd *exec.Cmd) error {
	spawnOnce.Do(func() {
		go func() {
			runtime.LockOSThread() // never unlocked: the thread dies with the process
			for r := range spawnReqs {
				r.done <- r.cmd.Start()
			}
		}()
	})
	done := make(chan error, 1)
	spawnReqs <- spawnReq{cmd, done}
	return <-done
}
