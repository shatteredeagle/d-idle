// Command gatemark judges one merge gate run from its `go test -json` stream (card mgonejob, internal/gatemark).
//
//	go list ./... | wc -l                                      # the package count the gate must reach
//	go test -json -short ./... | gatemark run -want N -commit SHA [-job ID] [-record FILE] [-ledger FILE]
//	gatemark quarantine -ledger FILE                           # flaky.txt rows of tests that failed twice without reproducing
//
// run prints the marker line ("gate ran: ..." or "gate NOT RUN: ...") and the failing tests. Exit codes: 0 green,
// 1 a test or package failed, 2 usage or unreadable input, 3 the gate did not run (no package reported, or fewer or
// more than -want). Standard library only; it starts nothing and reads only what it is given.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"d2lod-bot/internal/gatemark"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr, time.Now))
}

// record is the file kept for a failed gate (design 4.7): the summary with failures and tail, and what the caller knows
// about the run.
type record struct {
	At     string `json:"at"`
	Commit string `json:"commit"`
	Job    string `json:"job,omitempty"`
	Load   string `json:"load,omitempty"`
	Marker string `json:"marker"`
	gatemark.Summary
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer, now func() time.Time) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: gatemark run|quarantine [flags]")
		return 2
	}
	fs := flag.NewFlagSet("gatemark "+args[0], flag.ContinueOnError)
	fs.SetOutput(stderr)
	ledger := fs.String("ledger", "", "flake ledger file (JSONL, appended by run, read by quarantine)")
	switch args[0] {
	case "run":
		want := fs.Int("want", 0, "package count of `go list` for the gate's pattern (required)")
		commit := fs.String("commit", "", "commit the gate ran on (required)")
		job := fs.String("job", "", "builder job id")
		load := fs.String("load", "", "builder load at start and end, as the caller measured it")
		rec := fs.String("record", "", "write the run record (failing tests, tail) to this file when the gate is not green")
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		if *want <= 0 || *commit == "" {
			fmt.Fprintln(stderr, "gatemark run: -want and -commit are required (a marker without a package count proves nothing)")
			return 2
		}
		s, err := gatemark.Parse(stdin, *want)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		at := now().UTC().Format(time.RFC3339)
		fmt.Fprintln(stdout, s.Marker())
		for _, f := range s.Failures {
			fmt.Fprintf(stdout, "FAIL %s %s\n", f.Package, f.Test)
		}
		if *rec != "" && !s.Green() {
			b, _ := json.MarshalIndent(record{At: at, Commit: *commit, Job: *job, Load: *load, Marker: s.Marker(), Summary: s}, "", " ")
			if err := os.WriteFile(*rec, append(b, '\n'), 0o644); err != nil {
				fmt.Fprintln(stderr, "gatemark:", err)
				return 2
			}
		}
		// A run that did not run says nothing about the commit: it is not a ledger line.
		if *ledger != "" && s.Ran() {
			f, err := os.OpenFile(*ledger, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
			if err == nil {
				err = gatemark.Append(f, s.Lines(at, *commit, *job))
				if cerr := f.Close(); err == nil {
					err = cerr
				}
			}
			if err != nil {
				fmt.Fprintln(stderr, "gatemark:", err)
				return 2
			}
		}
		switch {
		case !s.Ran():
			return 3
		case !s.Green():
			return 1
		}
		return 0
	case "quarantine":
		if fs.Parse(args[1:]) != nil {
			return 2
		}
		f, err := os.Open(*ledger)
		if err != nil {
			fmt.Fprintln(stderr, "gatemark:", err)
			return 2
		}
		defer f.Close()
		lines, err := gatemark.ReadLedger(f)
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
		for _, fl := range gatemark.Flakes(lines) {
			fmt.Fprintln(stdout, fl.QuarantineRow())
		}
		return 0
	}
	fmt.Fprintf(stderr, "gatemark: unknown command %q\n", args[0])
	return 2
}
