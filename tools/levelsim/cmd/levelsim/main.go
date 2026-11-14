// Command levelsim simulates Act 1 Normal leveling strategies (docs/levelsim.md).
//
//	go run ./tools/levelsim/cmd/levelsim -calibrate
//	go run ./tools/levelsim/cmd/levelsim -optimise -mode plan -horizon 60,120,240
//	go run ./tools/levelsim/cmd/levelsim -sens -horizon 60,120
//	go run ./tools/levelsim/cmd/levelsim -strategy my.json -horizon 120
//	go run ./tools/levelsim/cmd/levelsim -extract bot.log -run-id 5
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/tools/levelsim"
)

func main() {
	var (
		horizon   = flag.String("horizon", "60,120,240", "comma list of horizons in minutes")
		calibrate = flag.Bool("calibrate", false, "replay the logged runs, fit the supply parameters, print the errors")
		optimise  = flag.Bool("optimise", false, "beam search the best strategy per horizon and compare it with today's plan")
		sens      = flag.Bool("sens", false, "sensitivity table (kill rate, overhead, restart cost, XP, supply, Den refarm)")
		strategy  = flag.String("strategy", "", "JSON policy file to simulate (see docs/levelsim.md)")
		extract   = flag.String("extract", "", "bot log to reduce to a runs.json entry (printed)")
		runID     = flag.Int("run-id", 0, "run id for -extract")
		start     = flag.String("start", "", "RFC3339 time of the hero's first moment for -extract (default: first log line)")
		modeF     = flag.String("mode", "plan", "danger floors: plan (today's entry levels), strict (boss minimum levels), bold, or all")
		width     = flag.Int("width", 60, "beam width per class")
		reachH    = flag.Float64("reach", 480, "horizon of the time-to-clvl search in minutes (0 = skip)")
		out       = flag.String("out", "", "directory for best-strategies.json")
		runs      = flag.String("runs", "", "extra runs.json to add to the embedded runs")
		useFit    = flag.Bool("fitted", false, "use freshly fitted supply parameters instead of the defaults")
		denRefarm = flag.Bool("den-refarm", false, "allow farming the Den again after the quest")
	)
	flag.Parse()
	d, err := levelsim.Load()
	die(err)
	if *runs != "" {
		b, err := os.ReadFile(*runs)
		die(err)
		var rf struct {
			Runs []levelsim.Run `json:"runs"`
		}
		die(json.Unmarshal(b, &rf))
		d.Runs = append(d.Runs, rf.Runs...)
	}
	hs := floats(*horizon)
	p := levelsim.DefaultParams()
	if *extract != "" {
		f, err := os.Open(*extract)
		die(err)
		defer f.Close()
		var t0 time.Time
		if *start != "" {
			t0, err = time.Parse(time.RFC3339, *start)
			die(err)
		}
		r, err := levelsim.Extract(f, *runID, "", t0, 60)
		die(err)
		b, _ := json.MarshalIndent(r, "", " ")
		fmt.Println(string(b))
		return
	}
	if *calibrate {
		calib(d, p)
	}
	if *useFit {
		p = levelsim.Calibrate(d, p, d.Runs).Best.P
	}
	if *strategy != "" {
		b, err := os.ReadFile(*strategy)
		die(err)
		var pol levelsim.Policy
		die(json.Unmarshal(b, &pol))
		e := levelsim.NewEnv(d, p, levelsim.Mode(*modeF))
		e.AllowDenRefarm = *denRefarm
		for _, h := range hs {
			r := e.RunPolicy(pol, h)
			fmt.Printf("%s @%g min: %.0f xp, clvl %d, %.0f kills, %d games\n%s", pol.Name, h, r.End.XP, r.Clvl(), r.End.TotalK, r.End.Games+1, e.FormatBands(r.Steps))
			fmt.Printf("  clvl by minute: %s\n", levelsim.FormatTrajectory(r.Steps, h, 10))
			for _, v := range r.Violations {
				fmt.Println("  below floor:", v)
			}
		}
	}
	modes := []levelsim.Mode{levelsim.Mode(*modeF)}
	if *modeF == "all" {
		modes = []levelsim.Mode{levelsim.ModePlan, levelsim.ModeStrict, levelsim.ModeBold}
	}
	sum := levelsim.Summary{Schema: "levelsim summary/1", Params: p,
		Note: "xp and clvl are model predictions (docs/levelsim.md); danger floors are U; plan_today is the intended plan with no bot failure modes"}
	if *optimise || *out != "" {
		for _, m := range modes {
			ms := levelsim.Summarise(d, p, m, hs, *width, *reachH, *width/2, *denRefarm)
			sum.Modes = append(sum.Modes, ms)
			fmt.Print(levelsim.FormatMode(levelsim.NewEnv(d, p, m), ms))
		}
	}
	if *sens {
		m := modes[0]
		rows := levelsim.Sensitivity(d, p, m, hs, *width)
		sum.Sens = rows
		fmt.Printf("sensitivity, danger mode %s\n%s", m, levelsim.FormatSens(rows, hs))
	}
	if *out != "" {
		die(os.MkdirAll(*out, 0o755))
		b, _ := json.MarshalIndent(sum, "", " ")
		die(os.WriteFile(filepath.Join(*out, "best-strategies.json"), append(b, '\n'), 0o644))
	}
	if !*calibrate && !*optimise && !*sens && *strategy == "" && *out == "" {
		flag.Usage()
	}
}

func calib(d *levelsim.Data, p levelsim.Params) {
	fmt.Println("== replay with the default parameters ==")
	fmt.Print(levelsim.CalibReport(d, p))
	fs := levelsim.Calibrate(d, p, d.Runs)
	fmt.Printf("\n== grid fit on all runs (%d points) ==\nbest: decay %.2f floor %.2f yield %.2f loss %.4f; %d points within 25%% of the best loss span decay %.2f..%.2f floor %.2f..%.2f yield %.2f..%.2f\n",
		fs.GridPoints, fs.Best.P.Decay, fs.Best.P.Floor, fs.Best.P.Yield, fs.Best.Loss, fs.Near,
		fs.Lo.Decay, fs.Hi.Decay, fs.Lo.Floor, fs.Hi.Floor, fs.Lo.Yield, fs.Hi.Yield)
	fmt.Print(levelsim.CalibReport(d, fs.Best.P))
	fmt.Println("\n== leave one run out (fit on the others, predict the held-out run) ==")
	for i, r := range d.Runs {
		if len(r.Arrivals) == 0 {
			continue
		}
		var rest []levelsim.Run
		for j, o := range d.Runs {
			if j != i && len(o.Arrivals) > 0 {
				rest = append(rest, o)
			}
		}
		if len(rest) == 0 {
			continue
		}
		f := levelsim.Calibrate(d, p, rest)
		rp := levelsim.NewEnv(d, f.Best.P, levelsim.ModeStrict).ReplayRun(r)
		fmt.Printf("run %d held out: fit on the rest (decay %.2f floor %.2f yield %.2f) predicts %.0f xp against %.0f observed (%+.1f%%), kills %+.1f%%\n",
			r.ID, f.Best.P.Decay, f.Best.P.Floor, f.Best.P.Yield, rp.PredXP, rp.ObsXP, 100*rp.XPErr, 100*rp.KillsErr)
	}
}

func floats(s string) []float64 {
	var out []float64
	for _, f := range strings.Split(s, ",") {
		v, err := strconv.ParseFloat(strings.TrimSpace(f), 64)
		die(err)
		out = append(out, v)
	}
	return out
}

func die(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "levelsim:", err)
		os.Exit(1)
	}
}
