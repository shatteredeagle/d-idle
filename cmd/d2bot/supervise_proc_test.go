package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
)

// The test binary doubles as the fake child: with D2BOT_FAKECHILD=<dir> it appends its args and relevant env to
// <dir>/log, then exits with the next code of <dir>/codes (comma list, the last repeats), or blocks until SIGTERM for "hang".
func TestMain(m *testing.M) {
	if dir := os.Getenv("D2BOT_FAKECHILD"); dir != "" {
		os.Exit(fakeChild(dir))
	}
	os.Exit(m.Run())
}

func fakeChild(dir string) int {
	f, _ := os.OpenFile(filepath.Join(dir, "log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	n := 0
	if b, err := os.ReadFile(filepath.Join(dir, "log")); err == nil {
		n = strings.Count(string(b), "\n")
	}
	fmt.Fprintf(f, "args=%s goalsenv=%q crashenv=%q\n", strings.Join(os.Args[1:], " "), os.Getenv("D2BOT_GOALS"), os.Getenv(testCrashEnv))
	f.Close()
	b, _ := os.ReadFile(filepath.Join(dir, "codes"))
	codes := strings.Split(strings.TrimSpace(string(b)), ",")
	c := codes[min(n, len(codes)-1)]
	if c == "hang" {
		ch := make(chan os.Signal, 1)
		signalNotify(ch)
		<-ch
		return 0
	}
	v, _ := strconv.Atoi(c)
	return v
}

func fakeSup(t *testing.T, codes string, max int, args []string, goals bool) (*supervisor, string) {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "codes"), []byte(codes), 0o644)
	t.Setenv("D2BOT_FAKECHILD", dir)
	t.Setenv(testCrashEnv, "30")
	o := options{superviseMax: max}
	if goals {
		o.goalSpec = "campaign"
	}
	s := newSupervisor(o, slog.New(slog.NewTextHandler(io.Discard, nil)), args, os.Args[0])
	return s, dir
}

func lines(t *testing.T, dir string) []string {
	b, _ := os.ReadFile(filepath.Join(dir, "log"))
	return strings.Split(strings.TrimSpace(string(b)), "\n")
}

func TestSuperviseCrashDrillWithFakeChild(t *testing.T) {
	s, dir := fakeSup(t, "2,2,0", 10, []string{"--managed", "--goals", "campaign", "--supervise"}, true)
	var mu sync.Mutex
	var waits []time.Duration
	s.sleep = func(_ context.Context, d time.Duration) { mu.Lock(); waits = append(waits, d); mu.Unlock() }
	if c := s.loop(context.Background()); c != 0 {
		t.Fatalf("code %d", c)
	}
	ls := lines(t, dir)
	if len(ls) != 3 {
		t.Fatalf("children: %v", ls)
	}
	if strings.Contains(ls[0], "--clean-start") || !strings.Contains(ls[0], "--goals campaign") || !strings.Contains(ls[0], `crashenv="30"`) {
		t.Fatalf("first start: %s", ls[0])
	}
	for _, l := range ls[1:] { // clean start (the child does the one game cleanup) each restart, no goals, no second crash
		if !strings.Contains(l, "--clean-start") || strings.Contains(l, "--goals") || !strings.Contains(l, "--autostart") ||
			!strings.Contains(l, `goalsenv=""`) || !strings.Contains(l, `crashenv=""`) {
			t.Fatalf("restart: %s", l)
		}
	}
	if !slices.Equal(waits, []time.Duration{s.base, 2 * s.base}) {
		t.Fatalf("backoff %v", waits)
	}
}

func TestSuperviseFakeChildGivesUpAtCap(t *testing.T) {
	s, dir := fakeSup(t, "2", 3, []string{"--managed"}, false)
	s.sleep = func(context.Context, time.Duration) {}
	if c := s.loop(context.Background()); c != 2 {
		t.Fatalf("code %d", c)
	}
	if n := len(lines(t, dir)); n != 4 { // first run + 3 restarts, then give up
		t.Fatalf("children %d", n)
	}
}

func TestSuperviseFakeChildExit3NotRestarted(t *testing.T) {
	s, dir := fakeSup(t, "3", 10, []string{"--managed"}, false)
	if c := s.loop(context.Background()); c != exitFatal || len(lines(t, dir)) != 1 {
		t.Fatalf("code %d", c)
	}
}

func TestSuperviseSIGTERMStopsChild(t *testing.T) {
	s, dir := fakeSup(t, "hang", 10, []string{"--managed"}, false)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan int, 1)
	go func() { done <- s.loop(ctx) }()
	for i := 0; i < 200 && len(lines(t, dir)) < 1; i++ {
		time.Sleep(25 * time.Millisecond)
	}
	time.Sleep(200 * time.Millisecond) // let the child reach its signal handler
	cancel()                           // what SIGTERM to the supervisor does: runChild forwards SIGTERM to the child
	select {
	case c := <-done:
		if c != 0 {
			t.Fatalf("code %d", c)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("supervisor did not stop")
	}
	if n := len(lines(t, dir)); n != 1 {
		t.Fatalf("restarted after shutdown: %d", n)
	}
}

func TestTestCrashDelayNeedsD2BotTest(t *testing.T) {
	env := func(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
	if _, ok := testCrashDelay(env(map[string]string{testCrashEnv: "5"})); ok {
		t.Fatal("honoured without D2BOT_TEST=1")
	}
	if d, ok := testCrashDelay(env(map[string]string{"D2BOT_TEST": "1", testCrashEnv: "5"})); !ok || d != 5*time.Second {
		t.Fatal(d, ok)
	}
	if _, ok := testCrashDelay(env(map[string]string{"D2BOT_TEST": "1", testCrashEnv: "x"})); ok {
		t.Fatal("garbage honoured")
	}
	if got := childEnv([]string{testCrashEnv + "=5", "D2BOT_TEST=1"}, true); slices.Contains(got, testCrashEnv+"=5") || !slices.Contains(got, "D2BOT_TEST=1") {
		t.Fatal(got)
	}
	_ = syscall.SIGTERM
}

func signalNotify(ch chan os.Signal) { signal.Notify(ch, syscall.SIGTERM) }

func TestChildArgsDropsTestCrashOnRestart(t *testing.T) {
	in := []string{"--managed", "--test-crash-after", "30"}
	if got := childArgs(in, true, 0, false); !slices.Equal(got, []string{"--managed", "--clean-start"}) {
		t.Fatal(got)
	}
	if got := childArgs(in, false, 0, false); !slices.Equal(got, in) {
		t.Fatal(got)
	}
}
