// Command d2dsoak is the harness of the d2d fidelity gate (docs/d2d-soak.md): run the same bot command on the private
// lockstep d2d for several seeds and virtual hours, extract the soak metrics of each run, aggregate over seeds and
// compare with the original-game baselines (tools/d2dsoak/baselines.json).
//
//	d2dsoak run     -seeds 1-8 -hours 2 -out DIR [-cmd 'bot --backend d2d ...'] [-clock runner|bot] [-fake]
//	d2dsoak extract [-hours H] LOG...                  metrics of minute-line logs (original-game soak logs too)
//	d2dsoak compare [-baselines FILE] [-json] RUN...   RUN = run.json, seed dir, or minute log
//	d2dsoak baseline NAME|LOG|SOURCE|NOTE...           print baseline runs as JSON (build baselines.json)
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"d2lod-bot/internal/d2dsoakstat"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch os.Args[1] {
	case "run":
		err = cmdRun(os.Args[2:])
	case "extract":
		err = cmdExtract(os.Args[2:])
	case "compare":
		err = cmdCompare(os.Args[2:])
	case "baseline":
		err = cmdBaseline(os.Args[2:])
	default:
		usage()
	}
	if err != nil {
		var ee exitError
		if errors.As(err, &ee) {
			if ee.Msg != "" {
				fmt.Fprintln(os.Stderr, "d2dsoak:", ee.Msg)
			}
			os.Exit(ee.Code)
		}
		fmt.Fprintln(os.Stderr, "d2dsoak:", err)
		os.Exit(ExitError)
	}
}

// Exit codes (docs/d2d-soak.md "Exit codes"). Only PASS is 0.
const (
	ExitPass         = d2dsoakstat.ExitPass
	ExitError        = 1 // I/O, bad input, seeds that failed to run
	ExitInconclusive = d2dsoakstat.ExitInconclusive
	ExitFail         = d2dsoakstat.ExitFail // FAIL and NO-DATA
	ExitUsage        = 64
)

// exitError makes main exit with Code (Msg printed to stderr when set).
type exitError struct {
	Code int
	Msg  string
}

func (e exitError) Error() string { return fmt.Sprintf("exit %d: %s", e.Code, e.Msg) }

// verdictExit maps a verdict to its exit code; zero is -exit-zero (old behaviour: always 0 after printing).
func verdictExit(overall string, zero bool) int {
	if zero {
		return ExitPass
	}
	return d2dsoakstat.ExitCode(overall)
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: d2dsoak run|extract|compare|baseline ... (see the package comment and docs/d2d-soak.md)")
	os.Exit(ExitUsage)
}

func cmdRun(args []string) error {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	home, _ := os.UserHomeDir()
	lab := filepath.Join(home, "d2d-lab")
	seeds := fs.String("seeds", "1-3", "seed list, e.g. 1,2,5 or 1-8")
	var cfg runCfg
	fs.Float64Var(&cfg.Hours, "hours", 2, "virtual hours per seed")
	fs.StringVar(&cfg.Out, "out", "", "output directory (required)")
	fs.StringVar(&cfg.Cmd, "cmd", "", "shell command run per seed (env D2DSOAK_SEED, D2DSOAK_SOCK, D2DSOAK_DIR, D2DSOAK_HOURS); stdout+stderr become bot.log")
	fs.StringVar(&cfg.Clock, "clock", "runner", "runner: this tool steps d2d (command must not use the socket); bot: the command steps, we read the clock at the end")
	fs.IntVar(&cfg.Chunk, "chunk", 1500, "runner clock: world ticks per step call (1500 = 1 virtual minute)")
	fs.StringVar(&cfg.Setup, "setup", "", "devctl script run before the command (character spec, d2dprobe script syntax)")
	fs.DurationVar(&cfg.MaxWall, "max-wall", 3*time.Hour, "wall limit per seed")
	bin := fs.String("d2d", filepath.Join(lab, "fork/src/build/apps/d2d/d2d"), "private lockstep fork binary")
	data := fs.String("data", filepath.Join(lab, "data"), "MPQ directory (read-only copy)")
	dhome := fs.String("home", filepath.Join(lab, "soak/home"), "private XDG dirs and socket directory")
	class := fs.Int("class", 4, "start class index (4 = Sorceress, docs/d2d-fidelity.md)")
	name := fs.String("name", "Soaky", "start character name")
	toggles := fs.String("toggle", "autoloot=off,autoparty=off", "d2d --toggle value (the original needs a click for gold)")
	maxLoad := fs.Float64("max-load", 45, "refuse to start above this 1-minute load average (0 = off)")
	baselines := fs.String("baselines", "", "baselines.json: compare after the run")
	exitZero := fs.Bool("exit-zero", false, "exit 0 whatever the verdict (old behaviour for scripts that read verdict.txt)")
	holdout := fs.String("holdout", d2dsoakstat.DefaultHoldout, "holdout seeds, reported as their own group (docs/geninv.md); empty = no holdout group")
	fake := fs.Bool("fake", false, "synthetic minute lines, no d2d (pipeline test, numbers mean nothing)")
	fakeScale := fs.Float64("fake-scale", 1, "fake mode: rate factor versus attempt 18")
	fs.Parse(args)
	if cfg.Out == "" {
		return fmt.Errorf("-out is required")
	}
	if cfg.Clock != "runner" && cfg.Clock != "bot" {
		return fmt.Errorf("-clock must be runner or bot")
	}
	sl, err := splitInts(*seeds)
	if err != nil {
		return err
	}
	hs, err := d2dsoakstat.ParseSeedSet(*holdout)
	if err != nil {
		return err
	}
	cfg.Class = *class
	cfg.Log = func(f string, a ...any) { fmt.Fprintf(os.Stderr, time.Now().Format("15:04:05 ")+f+"\n", a...) }
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var runs []d2dsoakstat.Run
	failed := 0
	for _, seed := range sl {
		var res runResult
		var err error
		if *fake {
			res, err = runFake(cfg, *fakeScale, seed)
		} else {
			pl := &procLab{Bin: *bin, Data: *data, Home: *dhome, Toggles: *toggles, Class: *class, Name: *name, MaxLoad: *maxLoad, MinFreeGB: 15}
			res, err = runSeed(ctx, cfg, pl, seed)
		}
		if err != nil {
			cfg.Log("seed %d failed: %v", seed, err)
			failed++
			if ctx.Err() != nil {
				break
			}
			continue
		}
		cfg.Log("seed %d: %s, %.2f virtual h in %.0f s wall (x%.0f), %d minute lines, xp/h %.0f", seed, res.End, res.Hours, res.WallS,
			res.Speedup, res.Minutes, res.Metrics["xp_per_hour"])
		runs = append(runs, res.Run)
	}
	if len(runs) == 0 {
		if *baselines != "" { // NO-DATA is a verdict (exit 3), not an I/O error
			return compareGate(nil, *baselines, false, filepath.Join(cfg.Out, "verdict.txt"), "v1", *exitZero, hs)
		}
		return fmt.Errorf("no seed completed")
	}
	agg := d2dsoakstat.Aggregate(runs)
	if err := writeJSON(filepath.Join(cfg.Out, "aggregate.json"), map[string]any{"runs": runs, "aggregate": agg}); err != nil {
		return err
	}
	fmt.Print(aggText(runs, agg))
	if *baselines == "" { // with baselines the verdict report carries the invariants
		fmt.Print(d2dsoakstat.BuildInvReport(runs, hs).Text())
	}
	var verr error
	if *baselines != "" {
		verr = compareGate(runs, *baselines, false, filepath.Join(cfg.Out, "verdict.txt"), "v1", *exitZero, hs)
	}
	if failed > 0 && !*exitZero {
		var ee exitError
		if !errors.As(verr, &ee) {
			return exitError{ExitError, fmt.Sprintf("%d of %d seeds failed to run", failed, len(sl))}
		}
	}
	return verr
}

func cmdExtract(args []string) error {
	fs := flag.NewFlagSet("extract", flag.ExitOnError)
	hours := fs.Float64("hours", 0, "fixed run length in hours (0 = from the minute timestamps)")
	asJSON := fs.Bool("json", false, "print JSON")
	holdout := fs.String("holdout", d2dsoakstat.DefaultHoldout, "holdout seeds, reported as their own group (docs/geninv.md); empty = no holdout group")
	fs.Parse(args)
	hs, err := d2dsoakstat.ParseSeedSet(*holdout)
	if err != nil {
		return err
	}
	runs, err := loadRuns(fs.Args(), *hours)
	if err != nil {
		return err
	}
	inv := d2dsoakstat.BuildInvReport(runs, hs)
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]any{"runs": runs, "aggregate": d2dsoakstat.Aggregate(runs), "invariants": inv})
	}
	fmt.Print(aggText(runs, d2dsoakstat.Aggregate(runs)))
	fmt.Print(inv.Text())
	printKillTimes(fs.Args(), 5)
	return nil
}

func cmdCompare(args []string) error {
	fs := flag.NewFlagSet("compare", flag.ExitOnError)
	bl := fs.String("baselines", "tools/d2dsoak/baselines.json", "baselines file")
	asJSON := fs.Bool("json", false, "print JSON")
	hours := fs.Float64("hours", 0, "fixed run length for raw minute logs")
	exitZero := fs.Bool("exit-zero", false, "exit 0 whatever the verdict")
	holdout := fs.String("holdout", d2dsoakstat.DefaultHoldout, "holdout seeds, reported as their own group (docs/geninv.md); empty = no holdout group")
	gate := fs.String("gate", "v1", "gate: v1 (20 % on four metrics) or v2 (Poisson deaths + proxies, docs/d2d-fidelity.md)")
	fs.Parse(args)
	if *gate != "v1" && *gate != "v2" {
		return fmt.Errorf("unknown -gate %q", *gate)
	}
	runs, err := loadRuns(fs.Args(), *hours)
	if err != nil {
		return err
	}
	hs, err := d2dsoakstat.ParseSeedSet(*holdout)
	if err != nil {
		return err
	}
	return compareGate(runs, *bl, *asJSON, "", *gate, *exitZero, hs)
}

// compareGate prints the verdict (text or JSON), saves the text when save is set and returns the exit code as an
// exitError (nil for PASS or exitZero), the same for text and JSON output.
func compareGate(runs []d2dsoakstat.Run, blPath string, asJSON bool, save, gate string, exitZero bool, holdout map[int]bool) error {
	f, err := os.Open(blPath)
	if err != nil {
		return err
	}
	defer f.Close()
	b, err := d2dsoakstat.ReadBaselines(f)
	if err != nil {
		return err
	}
	v := d2dsoakstat.Compare(runs, b)
	if gate == "v2" {
		v = d2dsoakstat.CompareV2(runs, b)
	}
	inv := d2dsoakstat.BuildInvReport(runs, holdout)
	v.Inv = &inv
	if save != "" {
		os.WriteFile(save, []byte(v.Text()), 0o644)
	}
	if asJSON {
		if err := json.NewEncoder(os.Stdout).Encode(v); err != nil {
			return err
		}
	} else {
		fmt.Print(v.Text())
	}
	if code := verdictExit(v.Overall, exitZero); code != ExitPass {
		return exitError{Code: code}
	}
	return nil
}

func cmdBaseline(args []string) error {
	var runs []d2dsoakstat.BaselineRun
	for _, a := range args {
		p := strings.SplitN(a, "|", 4)
		if len(p) < 3 {
			return fmt.Errorf("want NAME|LOG|SOURCE[|NOTE], got %q", a)
		}
		rs, err := loadRuns([]string{p[1]}, 0)
		if err != nil {
			return err
		}
		r := rs[0]
		r.Inv = nil // baselines carry metrics only
		r.Name = p[0]
		br := d2dsoakstat.BaselineRun{Run: r, Source: p[2]}
		if len(p) == 4 {
			br.Note = p[3]
		}
		runs = append(runs, br)
	}
	return json.NewEncoder(os.Stdout).Encode(runs)
}

// loadRuns reads each path as a run.json, a seed directory (run.json, else bot.log) or a minute log.
func loadRuns(paths []string, hours float64) ([]d2dsoakstat.Run, error) {
	if len(paths) == 0 {
		return nil, fmt.Errorf("no input")
	}
	var runs []d2dsoakstat.Run
	seen := map[string]bool{}
	for _, p := range paths {
		if c := filepath.Clean(p); seen[c] {
			return nil, fmt.Errorf("%s given twice (the same run would count as two seeds)", p)
		} else {
			seen[c] = true
		}
		if st, err := os.Stat(p); err == nil && st.IsDir() {
			if _, err := os.Stat(filepath.Join(p, "run.json")); err == nil {
				p = filepath.Join(p, "run.json")
			} else {
				p = filepath.Join(p, "bot.log")
			}
		}
		if strings.HasSuffix(p, "run.json") {
			b, err := os.ReadFile(p)
			if err != nil {
				return nil, err
			}
			var r runResult
			if err := json.Unmarshal(b, &r); err != nil {
				return nil, fmt.Errorf("%s: %w", p, err)
			}
			if r.Minutes < 2 && r.Invalid == "" {
				r.Invalid = fmt.Sprintf("%d minute line(s), need 2", r.Minutes)
			}
			r.Run.Seed = r.Seed
			if r.Inv == nil { // an older run.json: read the counters from the bot.log next to it
				if inv, err := scanInvariants(filepath.Join(filepath.Dir(p), "bot.log"), d2dsoakstat.ExitInfo{WantHours: hours, End: r.End, CmdErr: r.CmdErr}); err == nil {
					r.Inv = inv
				}
			}
			runs = append(runs, r.Run)
			continue
		}
		f, err := os.Open(p)
		if err != nil {
			return nil, err
		}
		ms, dropped, err := d2dsoakstat.ParseMinutesCount(f)
		f.Close()
		if err != nil {
			return nil, err
		}
		r := d2dsoakstat.Extract(filepath.Base(filepath.Dir(p))+"/"+filepath.Base(p), ms, d2dsoakstat.Options{Hours: hours})
		r.NoteDropped(dropped)
		if inv, err := scanInvariants(p, d2dsoakstat.ExitInfo{WantHours: hours}); err == nil {
			r.Inv = inv
		}
		runs = append(runs, r)
	}
	return runs, nil
}

func aggText(runs []d2dsoakstat.Run, agg map[string]d2dsoakstat.Stat) string {
	var b strings.Builder
	fmt.Fprintf(&b, "%-26s %6s %7s %7s %9s %8s %8s %6s\n", "run", "hours", "min", "kills", "xp/h", "deaths", "kills/h", "clvl+")
	for _, r := range runs {
		fmt.Fprintf(&b, "%-26s %6.2f %7d %7d %9.0f %8d %8.0f %6.0f\n", r.Name, r.Hours, r.Minutes, r.Kills, r.Metrics["xp_per_hour"], r.Deaths, r.Metrics["kills_per_hour"], r.Metrics["clvl_gain"])
	}
	fmt.Fprintf(&b, "\n%-22s %12s %10s %12s %12s %5s\n", "metric (all runs)", "mean", "sd", "min", "max", "n")
	keys := d2dsoakstat.MetricKeys
	for _, k := range keys {
		if s, ok := agg[k]; ok {
			fmt.Fprintf(&b, "%-22s %12.4g %10.4g %12.4g %12.4g %5d  %s\n", k, s.Mean, s.SD, s.Min, s.Max, s.N, d2dsoakstat.MetricUnit[k])
		}
	}
	states := map[string]float64{}
	n := 0
	for _, r := range runs {
		if len(r.States) > 0 {
			n++
		}
		for k, v := range r.States {
			states[k] += v
		}
	}
	if n > 0 {
		var ks []string
		for k := range states {
			ks = append(ks, k)
		}
		sort.Slice(ks, func(i, j int) bool { return states[ks[i]] > states[ks[j]] })
		b.WriteString("time in state (mean over runs):")
		for _, k := range ks {
			fmt.Fprintf(&b, " %s %.0f%%", k, 100*states[k]/float64(n))
		}
		b.WriteByte('\n')
	}
	return b.String()
}
