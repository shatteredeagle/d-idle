// Command d2doctor runs the setup preflight checks.
//
//	d2doctor [--json] [--data DIR] [--game DIR] [--slots N] [--addr HOST:PORT] [--pre-install] [--quick] [--instance iN] [--bundle OUT.tgz]
//
// Exit codes: 0 all pass, 1 warnings, 2 at least one failure, 3 usage or internal error.
package main

import (
	"context"
	"d2lod-bot/internal/buildinfo"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/setup"
)

func main() {
	buildinfo.Exit("d2doctor")
	setup.MaybeProbe()
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr, setup.NewEnv()))
}

func run(args []string, out, errw io.Writer, env *setup.Env) int {
	fs := flag.NewFlagSet("d2doctor", flag.ContinueOnError)
	fs.SetOutput(errw)
	var o setup.Options
	asJSON := fs.Bool("json", false, "machine-readable output")
	fs.StringVar(&o.DataDir, "data", "", "data dir (default $D2BOT_DATA or the standard location)")
	fs.StringVar(&o.GameDir, "game", "", "game install dir (default: base prefix under the data dir)")
	fs.StringVar(&o.Instance, "instance", "", "game instance to read the character from (default $D2BOT_INSTANCE)")
	fs.IntVar(&o.Slots, "slots", 1, "planned instances")
	fs.StringVar(&o.UIAddr, "addr", "", "web UI address to test")
	bundle := fs.String("bundle", "", "also write a support bundle (tar.gz: doctor JSON, log tails, version, redacted config) to this file")
	pre := fs.Bool("pre-install", false, "the game is not installed yet: missing game files warn instead of fail")
	fs.BoolVar(&o.Quick, "quick", false, "skip slow checks")
	if err := fs.Parse(args); err != nil {
		return 3
	}
	if o.DataDir == "" {
		l, err := paths.FromEnv()
		if err != nil {
			fmt.Fprintln(errw, err)
			return 3
		}
		o.DataDir = l.Root
	}
	o.GameExpected = !*pre
	rep := setup.NewReport(setup.RunChecks(context.Background(), env, o, setup.Checks()))
	if *bundle != "" {
		l := paths.Layout{Root: o.DataDir, ConfigFile: os.Getenv(paths.EnvConfig)}
		if err := writeBundle(*bundle, rep, l, time.Now()); err != nil {
			fmt.Fprintln(errw, "bundle:", err)
			return 3
		}
		fmt.Fprintln(errw, "bundle written:", *bundle)
	}
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
		return rep.Exit
	}
	for _, r := range rep.Results {
		fmt.Fprintf(out, "%-5s %-18s %s\n", label(r.Status), r.ID, r.Detail)
		if r.Fix != "" && (r.Status == setup.Fail || r.Status == setup.Warn) {
			fmt.Fprintf(out, "      fix: %s\n", r.Fix)
		}
	}
	fmt.Fprintf(out, "\n%d pass, %d warn, %d fail, %d skipped\n", rep.Counts.Pass, rep.Counts.Warn, rep.Counts.Fail, rep.Counts.Skip)
	return rep.Exit
}

func label(s setup.Status) string {
	switch s {
	case setup.Pass:
		return "PASS"
	case setup.Warn:
		return "WARN"
	case setup.Fail:
		return "FAIL"
	}
	return "SKIP"
}
