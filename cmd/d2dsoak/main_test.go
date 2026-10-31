//go:build unix

package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dctl/fake"
	"d2lod-bot/internal/d2dsoakstat"
)

// fakeLab stands in for the d2d process: the d2dctl fake server answers step and clock (virtual clock = ticks * 40 ms).
type fakeLab struct {
	srv     *fake.Server
	cleanup func()
	stopped bool
}

func (l *fakeLab) Start(ctx context.Context, seed int, logPath string) (string, error) {
	sock, cleanup, err := fake.TempSocket()
	if err != nil {
		return "", err
	}
	l.cleanup = cleanup
	l.srv, err = fake.New(sock, fake.Sample(), fake.Options{})
	return sock, err
}

func (l *fakeLab) Stop() error { l.stopped = true; l.srv.Close(); l.cleanup(); return nil }

func testCfg(t *testing.T, hours float64, cmd string) runCfg {
	return runCfg{Hours: hours, Out: t.TempDir(), Cmd: cmd, Clock: "runner", Chunk: 1500, MaxWall: time.Minute,
		Log: func(f string, a ...any) { t.Logf(f, a...) }}
}

// Dry run: no command, the runner steps the virtual clock for 3 virtual minutes, reads it and stops the lab.
func TestRunnerClockOnly(t *testing.T) {
	cfg := testCfg(t, 0.05, "")
	lab := &fakeLab{}
	res, err := runSeed(context.Background(), cfg, lab, 7)
	if err != nil {
		t.Fatal(err)
	}
	if !lab.stopped {
		t.Error("lab not stopped")
	}
	if res.End != "hours reached" || res.GameMS != 180000 {
		t.Errorf("end %q game_ms %d, want hours reached / 180000 (3 virtual minutes)", res.End, res.GameMS)
	}
	lines, _ := os.ReadFile(filepath.Join(cfg.Out, "seed-7", "clock.jsonl"))
	if n := strings.Count(string(lines), "\n"); n != 4 { // 3 chunks + the final read
		t.Errorf("clock.jsonl has %d lines:\n%s", n, lines)
	}
	if _, err := os.Stat(filepath.Join(cfg.Out, "seed-7", "run.json")); err != nil {
		t.Error(err)
	}
}

// A command that prints real minute lines: they become bot.log and the metrics, with the d2d clock as the run length.
func TestRunnerWithCommand(t *testing.T) {
	fixture, _ := filepath.Abs("../../internal/d2dsoakstat/testdata/a18_minutes.jsonl")
	cfg := testCfg(t, 2, "cat "+fixture+"; sleep 60")
	res, err := runSeed(context.Background(), cfg, &fakeLab{}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if res.Minutes != 124 || res.Hours < 1.99 || res.Hours > 2.01 {
		t.Errorf("minutes %d hours %.3f (end %q)", res.Minutes, res.Hours, res.End)
	}
	// the clock says 2 h, the 124 lines are spread over it: XP rate = 18965 / 2 h
	if x := res.Metrics["xp_per_hour"]; x < 9400 || x > 9560 {
		t.Errorf("xp/h = %.0f", x)
	}
}

func TestBotClockMode(t *testing.T) {
	cfg := testCfg(t, 1, "echo hello")
	cfg.Clock = "bot"
	res, err := runSeed(context.Background(), cfg, &fakeLab{}, 2)
	if err != nil {
		t.Fatal(err)
	}
	if res.End != "cmd exited" || res.Minutes != 0 {
		t.Errorf("end %q minutes %d", res.End, res.Minutes)
	}
}

func TestSplitInts(t *testing.T) {
	got, err := splitInts("1,3-5, 9")
	if err != nil || len(got) != 5 || got[1] != 3 || got[3] != 5 || got[4] != 9 {
		t.Errorf("got %v, %v", got, err)
	}
	if _, err := splitInts("5-2"); err == nil {
		t.Error("descending range must fail")
	}
}

func TestFakePipeline(t *testing.T) {
	cfg := testCfg(t, 2, "")
	res, err := runFake(cfg, 1, 3)
	if err != nil || res.Minutes != 121 || res.Metrics["xp_per_hour"] < 5000 {
		t.Fatalf("%v %+v", err, res.Run)
	}
	runs, err := loadRuns([]string{filepath.Join(cfg.Out, "seed-3")}, 0)
	if err != nil || len(runs) != 1 || runs[0].Minutes != 121 {
		t.Fatalf("loadRuns: %v %v", runs, err)
	}
}

func TestSplitIntsDuplicates(t *testing.T) {
	for _, s := range []string{"1,1,1,1,1", "1-3,2"} {
		if _, err := splitInts(s); err == nil {
			t.Errorf("%q must be rejected", s)
		}
	}
}

func TestVerdictExit(t *testing.T) {
	cases := []struct {
		overall string
		zero    bool
		want    int
	}{{"PASS", false, 0}, {"FAIL", false, 3}, {"INCONCLUSIVE", false, 2}, {"FAIL", true, 0}, {"INCONCLUSIVE", true, 0}}
	for _, c := range cases {
		if got := verdictExit(c.overall, c.zero); got != c.want {
			t.Errorf("%s zero=%v: %d, want %d", c.overall, c.zero, got, c.want)
		}
	}
}

// -json used to return before the exit code: a FAIL or INCONCLUSIVE verdict printed JSON and exited 0.
func TestCompareGateExitCodesJSON(t *testing.T) {
	dir := t.TempDir()
	bl := filepath.Join(dir, "bl.json")
	os.WriteFile(bl, []byte(`{"rules":{"xp_per_hour":{"gate":true,"rel":0.2}},"minSeeds":5,"minHoursPerSeed":2,"minTotalHours":10,
	"runs":[{"name":"a","hours":2,"metrics":{"xp_per_hour":1000}},{"name":"b","hours":2,"metrics":{"xp_per_hour":1000}},{"name":"c","hours":2,"metrics":{"xp_per_hour":1000}}]}`), 0o644)
	mk := func(n int, xp float64) []d2dsoakstat.Run {
		var rs []d2dsoakstat.Run
		for i := 0; i < n; i++ {
			rs = append(rs, d2dsoakstat.Run{Name: "s", Hours: 2, Metrics: map[string]float64{"xp_per_hour": xp}})
		}
		return rs
	}
	save := filepath.Join(dir, "v.txt")
	code := func(err error) int {
		var ee exitError
		if errors.As(err, &ee) {
			return ee.Code
		}
		return 0
	}
	for _, c := range []struct {
		runs []d2dsoakstat.Run
		want int
	}{{mk(5, 1000), 0}, {mk(5, 300), 3}, {mk(2, 1000), 2}} {
		for _, js := range []bool{false, true} {
			if got := code(compareGate(c.runs, bl, js, save, "v1", false)); got != c.want {
				t.Errorf("json=%v want %d got %d", js, c.want, got)
			}
		}
	}
	if _, err := os.Stat(save); err != nil {
		t.Error("verdict.txt must be saved with -json too")
	}
	if err := compareGate(mk(5, 300), bl, true, "", "v1", true); err != nil {
		t.Errorf("-exit-zero: %v", err)
	}
}

func TestEarlyEndAndEmptySeedAreInvalid(t *testing.T) {
	res := runResult{End: "cmd exited", CmdErr: "exit status 1", GameMS: 600000}
	markEarlyEnd(&res, 2)
	if res.Invalid == "" {
		t.Error("failed command not flagged")
	}
	res = runResult{End: "wall limit", GameMS: 3600000}
	markEarlyEnd(&res, 2)
	if res.Invalid == "" {
		t.Error("half-length wall-limit seed not flagged")
	}
	res = runResult{End: "hours reached", GameMS: 7200000}
	markEarlyEnd(&res, 2)
	if res.Invalid != "" {
		t.Errorf("clean run flagged: %s", res.Invalid)
	}
	// crashed at start: no minute lines, the d2d clock says 2 h, still no exposure
	log := filepath.Join(t.TempDir(), "bot.log")
	os.WriteFile(log, []byte("panic\n"), 0o644)
	res, err := finish(runResult{}, log, 2)
	if err != nil || res.Hours != 0 || res.Invalid == "" {
		t.Errorf("empty seed: %v %+v", err, res.Run)
	}
}

func TestLoadRunsRejectsDuplicatePaths(t *testing.T) {
	fixture := "../../internal/d2dsoakstat/testdata/a18_minutes.jsonl"
	if _, err := loadRuns([]string{fixture, fixture}, 0); err == nil {
		t.Error("duplicate path must be rejected")
	}
}
