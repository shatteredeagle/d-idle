package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/d2dsoakstat"
)

// outputDrain is how long a command may keep running after the virtual clock reached its target, to flush its output.
const outputDrain = time.Second

// runCfg is one replicated soak: the same command once per seed.
type runCfg struct {
	Seeds   []int
	Hours   float64 // virtual hours per seed
	Out     string  // output directory, one seed-N subdirectory per seed
	Cmd     string  // shell command per seed; its stdout and stderr are captured as bot.log (the minute lines)
	Clock   string  // "runner": this tool steps d2d (the command must not use the socket); "bot": the command steps, we only read the clock
	Chunk   int     // runner clock: world ticks per step call (1500 = one virtual minute)
	Setup   string  // optional devctl script (d2dctl script syntax) run before the command: character spec
	MaxWall time.Duration
	Class   int // d2d --start-class index (UI order, docs/d2d-fidelity.md): the coverage line names it
	Log     func(string, ...any)
}

// clockLine is one line of clock.jsonl.
type clockLine struct {
	VMin   int     `json:"vmin"`
	GameMS int     `json:"gameMs"`
	Tick   int     `json:"tick"`
	WallS  float64 `json:"wallS"`
}

// runResult is run.json of one seed: the extracted metrics plus how the run ended.
type runResult struct {
	d2dsoakstat.Run
	Seed    int     `json:"seed"`
	Clock   string  `json:"clock"`
	GameMS  int     `json:"gameMs"`
	WallS   float64 `json:"wallS"`
	End     string  `json:"end"` // hours reached | cmd exited | wall limit | fake
	CmdErr  string  `json:"cmdErr,omitempty"`
	Speedup float64 `json:"speedup,omitempty"` // virtual seconds per wall second
}

func runSeed(ctx context.Context, cfg runCfg, lab Lab, seed int) (runResult, error) {
	dir := filepath.Join(cfg.Out, "seed-"+strconv.Itoa(seed))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return runResult{}, err
	}
	res := runResult{Seed: seed, Clock: cfg.Clock}
	t0 := time.Now()
	sock, err := lab.Start(ctx, seed, filepath.Join(dir, "d2d.log"))
	if err != nil {
		return res, err
	}
	defer func() {
		if e := lab.Stop(); e != nil {
			cfg.Log("seed %d: STOP PROBLEM: %v", seed, e)
		}
	}()
	c := d2dctl.New(sock, d2dctl.Options{Timeout: 30 * time.Second})
	defer func() { c.Close() }()
	if cfg.Setup != "" {
		f, err := os.Open(cfg.Setup)
		if err != nil {
			return res, err
		}
		_, err = d2dctl.RunScript(ctx, c, f)
		f.Close()
		if err != nil {
			return res, fmt.Errorf("setup script: %w", err)
		}
	}
	botLog, err := os.Create(filepath.Join(dir, "bot.log"))
	if err != nil {
		return res, err
	}
	defer botLog.Close()
	clockF, err := os.Create(filepath.Join(dir, "clock.jsonl"))
	if err != nil {
		return res, err
	}
	defer clockF.Close()
	writeClock := func(vmin int, ci d2dctl.ClockInfo) {
		b, _ := json.Marshal(clockLine{vmin, ci.GameMS, ci.Tick, time.Since(t0).Seconds()})
		clockF.Write(append(b, '\n'))
	}

	var (
		cmd     *exec.Cmd
		cmdDone = make(chan error, 1)
	)
	if cfg.Cmd != "" {
		if cfg.Clock == "bot" {
			c.Close() // a single control client: free the socket for the command
		}
		cmd = exec.CommandContext(ctx, "sh", "-c", cfg.Cmd)
		cmd.Env = append(os.Environ(), "D2DSOAK_SEED="+strconv.Itoa(seed), "D2DSOAK_SOCK="+sock, "D2DSOAK_DIR="+dir,
			"D2DSOAK_HOURS="+strconv.FormatFloat(cfg.Hours, 'f', -1, 64), "D2D_SOCK="+sock)
		cmd.Stdout, cmd.Stderr = botLog, botLog
		cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
		cmd.Cancel = func() error { return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM) }
		cmd.WaitDelay = 5 * time.Second
		if err := cmd.Start(); err != nil {
			return res, err
		}
		go func() { cmdDone <- cmd.Wait() }()
	}
	killCmd := func() {
		if cmd != nil && cmd.Process != nil {
			syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
			select {
			case <-cmdDone:
			case <-time.After(5 * time.Second):
				syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
				<-cmdDone
			}
			cmd = nil
		}
	}

	deadline := time.Now().Add(cfg.MaxWall)
	var ci d2dctl.ClockInfo
	switch cfg.Clock {
	case "runner":
		target := int(cfg.Hours * 3600 * 25) // 25 ticks per virtual second
		for vmin := 1; ; vmin++ {
			select {
			case e := <-cmdDone:
				cmd, res.End = nil, "cmd exited"
				if e != nil {
					res.CmdErr = e.Error()
				}
			default:
			}
			if res.End != "" {
				break
			}
			sctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
			ci, err = c.Step(sctx, cfg.Chunk)
			cancel()
			if err != nil {
				killCmd()
				return res, fmt.Errorf("step: %w", err)
			}
			writeClock(vmin, ci)
			if ci.Tick >= target {
				res.End = "hours reached"
				break
			}
			if time.Now().After(deadline) {
				res.End = "wall limit"
				break
			}
		}
	default: // bot: wait for the command, then read the clock it left behind
		if cmd == nil {
			return res, fmt.Errorf("--clock bot needs --cmd")
		}
		select {
		case e := <-cmdDone:
			cmd, res.End = nil, "cmd exited"
			if e != nil {
				res.CmdErr = e.Error()
			}
		case <-time.After(time.Until(deadline)):
			res.End = "wall limit"
		}
	}
	if cmd != nil && res.End == "hours reached" {
		// The command's output is the metrics source: when the clock finishes first (the fake lab steps in milliseconds)
		// give a just-started command a moment to write before it is stopped.
		select {
		case e := <-cmdDone:
			cmd = nil
			if e != nil {
				res.CmdErr = e.Error()
			}
		case <-time.After(outputDrain):
		}
	}
	killCmd()
	if cfg.Clock == "bot" {
		c = d2dctl.New(sock, d2dctl.Options{Timeout: 30 * time.Second}) // the command held the one control connection, ours was closed
	}
	cctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if ci, err = c.Clock(cctx); err != nil {
		return res, fmt.Errorf("final clock read: %w", err)
	}
	writeClock(-1, ci)
	res.GameMS = ci.GameMS
	res.WallS = time.Since(t0).Seconds()
	if res.WallS > 0 {
		res.Speedup = float64(ci.GameMS) / 1000 / res.WallS
	}
	res, err = finish(res, filepath.Join(dir, "bot.log"), float64(ci.GameMS)/3.6e6, cfg.Hours)
	res.Cover = &d2dsoakstat.Cover{Class: className(cfg.Class), Difficulty: "normal"} // a fresh d2d start character is Normal
	if err != nil {
		return res, err
	}
	markEarlyEnd(&res, cfg.Hours)
	return res, writeJSON(filepath.Join(dir, "run.json"), res)
}

// finish extracts the metrics from a bot log. hours is the d2d virtual clock (0 = unknown: the minute lines are then
// counted as one virtual minute each, which only holds if the backend paces the bot's minute ticker by the d2d clock, U).
func finish(res runResult, logPath string, hours, want float64) (runResult, error) {
	f, err := os.Open(logPath)
	if err != nil {
		return res, err
	}
	defer f.Close()
	ms, dropped, err := d2dsoakstat.ParseMinutesCount(f)
	if err != nil {
		return res, err
	}
	if hours <= 0 && len(ms) > 1 {
		hours = float64(len(ms)-1) / 60
	}
	res.Run = d2dsoakstat.Extract("seed-"+strconv.Itoa(res.Seed), ms, d2dsoakstat.Options{Hours: hours})
	res.NoteDropped(dropped)
	res.Run.Seed = res.Seed
	if res.Inv, err = scanInvariants(logPath, d2dsoakstat.ExitInfo{WantHours: want, End: res.End, CmdErr: res.CmdErr}); err != nil {
		return res, err
	}
	// An empty or crashed seed has no exposure: Hours stays 0 and Invalid says why (the d2d clock alone is not exposure).
	if res.Hours == 0 && res.Invalid == "" {
		res.Hours = hours
	}
	return res, nil
}

// markEarlyEnd flags a seed that did not run the requested length: a failed command or a clock more than 5 % short
// (cmd exited early, wall limit) is not a clean full-length run. res.Hours stays the covered virtual time.
func markEarlyEnd(res *runResult, want float64) {
	if res.Invalid != "" || res.End == "fake" {
		return
	}
	switch {
	case res.CmdErr != "":
		res.Invalid = fmt.Sprintf("command failed (%s) after %.2f of %.2f h", res.CmdErr, float64(res.GameMS)/3.6e6, want)
	case res.End != "hours reached" && float64(res.GameMS)/3.6e6 < 0.95*want:
		res.Invalid = fmt.Sprintf("%s after %.2f of %.2f h", res.End, float64(res.GameMS)/3.6e6, want)
	}
}

// scanInvariants streams a bot log through the invariant scanner; a supervisor.log next to it (hidden soaks) adds the
// supervisor's launches and how the bot ended.
func scanInvariants(logPath string, x d2dsoakstat.ExitInfo) (*d2dsoakstat.Invariants, error) {
	if f, err := os.Open(filepath.Join(filepath.Dir(logPath), "supervisor.log")); err == nil {
		d2dsoakstat.ScanSupervisor(f, &x)
		f.Close()
	}
	f, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	inv, err := d2dsoakstat.ScanInvariants(f, x, d2dsoakstat.DefaultInvLimits())
	return &inv, err
}

var classNames = map[int]string{0: "barbarian", 1: "necromancer", 2: "paladin", 3: "amazon", 4: "sorceress", 5: "druid", 6: "assassin"}

func className(i int) string {
	if n, ok := classNames[i]; ok {
		return n
	}
	return fmt.Sprintf("class#%d", i)
}

func writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func splitInts(s string) ([]int, error) {
	var out []int
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if a, b, ok := strings.Cut(p, "-"); ok { // 1-5
			lo, e1 := strconv.Atoi(a)
			hi, e2 := strconv.Atoi(b)
			if e1 != nil || e2 != nil || hi < lo || hi-lo > 999 {
				return nil, fmt.Errorf("bad seed range %q", p)
			}
			for i := lo; i <= hi; i++ {
				out = append(out, i)
			}
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("bad seed %q", p)
		}
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("no seeds")
	}
	seen := map[int]bool{}
	for _, n := range out {
		if seen[n] {
			return nil, fmt.Errorf("duplicate seed %d (one seed is deterministic, repeating it adds no sample)", n)
		}
		seen[n] = true
	}
	return out, nil
}
