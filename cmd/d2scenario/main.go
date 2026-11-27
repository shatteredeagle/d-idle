// Command d2scenario is the Proving Ground tool (docs/process/proving-ground.md): list, validate, plan (the dry run),
// setup, eval, coverage, report, determinism and gate-check work offline; run executes scenarios on a lockstep d2d lab slot of
// the builder and writes results as JSON. The scenario files are data under tools/provingground/scenarios.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"d2lod-bot/internal/scenario"
)

const defaultDir = "tools/provingground/scenarios"

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(2)
	}
	cmd, args := os.Args[1], os.Args[2:]
	var err error
	switch cmd {
	case "list":
		err = cmdList(args, os.Stdout)
	case "validate":
		err = cmdValidate(args, os.Stdout)
	case "plan":
		err = cmdPlan(args, os.Stdout)
	case "setup":
		err = cmdSetup(args, os.Stdout)
	case "eval":
		err = cmdEval(args, os.Stdout)
	case "coverage":
		err = cmdCoverage(args, os.Stdout)
	case "provenance":
		err = cmdProvenance(args, os.Stdout)
	case "incident":
		err = cmdIncident(args, os.Stdout)
	case "arbiter-replay":
		err = cmdArbiterReplay(args, os.Stdout)
	case "report":
		err = cmdReport(args, os.Stdout)
	case "determinism":
		err = cmdDeterminism(args, os.Stdout)
	case "determinism-run":
		err = cmdDeterminismRun(args, os.Stdout)
	case "ledger":
		err = cmdLedger(args, os.Stdout)
	case "gate-check":
		err = cmdGateCheck(args, os.Stdout)
	case "run":
		err = cmdRun(args, os.Stdout)
	case "soak":
		err = cmdSoak(args, os.Stdout)
	case "exec-one":
		err = cmdExecOne(args)
	case "-h", "--help", "help":
		usage(os.Stdout)
		return
	default:
		usage(os.Stderr)
		os.Exit(2)
	}
	if err != nil {
		if e, ok := err.(exitErr); ok {
			fmt.Fprintln(os.Stdout, e.msg)
			os.Exit(e.code)
		}
		fmt.Fprintln(os.Stderr, "d2scenario:", err)
		os.Exit(1)
	}
}

// exitErr carries a verdict exit code (3 = not green) without an error line on stderr.
type exitErr struct {
	code int
	msg  string
}

func (e exitErr) Error() string { return e.msg }

func usage(w io.Writer) {
	fmt.Fprint(w, `usage: d2scenario COMMAND [flags]
  list        [-dir D] [-tier T]                list scenarios
  validate    [-dir D] [-board B]               check files, ids and coverage references
  plan        [-tier fast|full|nightly] [-changed F,F] [-id ID,..] [-class C] [-seed N] [-json]
                                                DRY RUN: the runs, their setup verbs and a wall estimate; starts nothing
  setup       ID -class C -seed N               print the declared SETUP (verbs, board record, provenance tags)
  eval        ID -state state.json              evaluate a scenario's assertions on a state file (offline)
  coverage    [-dir D] [-board B] [-json]       behaviour to scenario map, UNCOVERED list
  provenance  [-dir D]                          rows for docs/probes/test-characters.md (every scenario's CHEATED/SETUP tags)
  report      results.json                      one-screen report
  arbiter-replay [-strict] [-follow] [-session LOG] FILE.json..  score the arbiter against the chain by outcome: the recorded cases, or every tick of a session log (tools/provingground/arbiter-outcomes, report-only)
  incident    [-class C] [-seed N] [-out DIR] FILE.json..   build replay rows from incident files the bot wrote at a death (report-only tier)
  determinism-run -n N -out DIR [run flags]      run the selection N times (journals on) and compare every run with the first
  ledger status-doc|page-json|check -in L.jsonl the results ledger: generated status page, board /pg data, line check (lines are written by exec-one)
  determinism a.json b.json                     same seed twice: same verdict (and state digest)
  gate-check  results.json                      exit 3 unless GREEN (no RED, no BLOCKED)
  soak        -faults SET -duration D [-id ID,..] [-class C] [-seed N] [-dry-run] [-strict]   run rows under injected faults, report invariant hits and survival (lab only)
  run         [plan flags] [-out DIR] [-parallel N] [-dry-run]   run on the builder (long lane), see docs/process/proving-ground.md
`)
}

func loadAll(dir string) ([]scenario.Scenario, error) {
	scs, err := scenario.LoadDir(dir)
	if err != nil {
		return nil, err
	}
	if len(scs) == 0 {
		return nil, fmt.Errorf("no scenarios in %s", dir)
	}
	return scs, nil
}

func splitList(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func findScenario(scs []scenario.Scenario, id string) (scenario.Scenario, error) {
	for _, s := range scs {
		if s.ID == id {
			return s, nil
		}
	}
	return scenario.Scenario{}, fmt.Errorf("no scenario %q", id)
}

func readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}

func readResults(path string) ([]scenario.Result, error) {
	var f struct {
		Version string            `json:"version"`
		Results []scenario.Result `json:"results"`
	}
	if err := readJSON(path, &f); err != nil {
		return nil, err
	}
	if f.Version != scenario.ResultVersion {
		return nil, fmt.Errorf("%s: version %q, want %q", path, f.Version, scenario.ResultVersion)
	}
	return f.Results, nil
}

func newFlags(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	return fs
}

// parseArgs lets a leading positional (a scenario id) come before the flags.
func parseArgs(fs *flag.FlagSet, args []string) (pos string, err error) {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		pos, args = args[0], args[1:]
	}
	return pos, fs.Parse(args)
}

// flagSet reports whether the flag was given on the command line.
func flagSet(fs *flag.FlagSet, name string) bool {
	set := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == name {
			set = true
		}
	})
	return set
}
