// Command d2shadow digests the shadow lines of recorded bot logs (JSON lines) per family: admission, planner, upkeep
// and loot (docs/shadowlog.md). Offline and read only. Usage: d2shadow [-json] [-strict] [-rows N] LOGFILE...
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"d2lod-bot/internal/shadowlog"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, out, errw io.Writer) int {
	fs := flag.NewFlagSet("d2shadow", flag.ContinueOnError)
	fs.SetOutput(errw)
	asJSON := fs.Bool("json", false, "print the report as JSON")
	strict := fs.Bool("strict", false, "exit 1 when a family's verdict is review or worse")
	rows := fs.Int("rows", 10, "disagreement rows printed per family (text output; 0 = all)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() == 0 {
		fmt.Fprintln(errw, "usage: d2shadow [-json] [-strict] [-rows N] LOGFILE...")
		return 2
	}
	var ins []io.Reader
	for _, p := range fs.Args() {
		f, err := os.Open(p)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
		defer f.Close()
		ins = append(ins, f)
	}
	rep, err := shadowlog.Read(ins...)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 2
	}
	if *asJSON {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		_ = enc.Encode(rep)
	} else {
		text(out, rep, *rows)
	}
	if *strict && rep.Failing() {
		return 1
	}
	return 0
}

func text(w io.Writer, rep shadowlog.Report, max int) {
	fmt.Fprintf(w, "log: %d line(s), %d not JSON, %s to %s\n", rep.Lines, rep.NotJSON, orDash(rep.First), orDash(rep.Last))
	for _, d := range rep.Families {
		fmt.Fprintf(w, "\n%s (%s): %s; lines %d, differ %d, unparsed %d\n  %s\n", d.Family, d.Knob, d.Verdict, d.Lines, d.Differ, d.Unparsed, d.Why)
		for _, r := range d.Totals {
			fmt.Fprintf(w, "  total %-40s %d\n", r.Key, r.Count)
		}
		for i, r := range d.Rows {
			if max > 0 && i == max {
				fmt.Fprintf(w, "  ... %d more row(s), -rows 0 prints all\n", len(d.Rows)-max)
				break
			}
			fmt.Fprintf(w, "  %4d  %s", r.Count, r.Key)
			if r.Example != "" {
				fmt.Fprintf(w, "  [%s]", r.Example)
			}
			fmt.Fprintln(w)
		}
	}
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
