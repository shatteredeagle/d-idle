package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func main() { os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr)) }

func run(args []string, in io.Reader, out, errw io.Writer) int {
	fs := flag.NewFlagSet("d2firsthour", flag.ContinueOnError)
	fs.SetOutput(errw)
	logf := fs.String("log", "-", "bot log (slog text or JSON, e.g. `docker logs` output); - = stdin")
	class := fs.String("class", "", "class label of the hero (required)")
	startS := fs.String("start", "", "RFC3339 start; default the first `d2bot start` line of the log")
	table := fs.String("table", "docs/probes/firsthour.md", "table file to append to")
	dry := fs.Bool("dry-run", false, "print the row, write nothing")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *class == "" {
		fmt.Fprintln(errw, "d2firsthour: -class is required")
		return 2
	}
	var start time.Time
	if *startS != "" {
		t, err := time.Parse(time.RFC3339, *startS)
		if err != nil {
			fmt.Fprintln(errw, "d2firsthour: -start:", err)
			return 2
		}
		start = t
	}
	r := in
	if *logf != "-" {
		f, err := os.Open(*logf)
		if err != nil {
			fmt.Fprintln(errw, "d2firsthour:", err)
			return 2
		}
		defer f.Close()
		r = f
	}
	lines, err := parse(r)
	if err != nil {
		fmt.Fprintln(errw, "d2firsthour:", err)
		return 2
	}
	res, err := Compute(lines, *class, start, time.Hour, 90*time.Second)
	if err != nil {
		fmt.Fprintln(errw, "d2firsthour: FAIL:", err)
		return 1
	}
	if *dry {
		fmt.Fprint(out, res.Row())
		return 0
	}
	if _, err := os.Stat(*table); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(*table), 0o755); err != nil {
			fmt.Fprintln(errw, "d2firsthour:", err)
			return 2
		}
		if err := os.WriteFile(*table, []byte(Header), 0o644); err != nil {
			fmt.Fprintln(errw, "d2firsthour:", err)
			return 2
		}
	}
	f, err := os.OpenFile(*table, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		fmt.Fprintln(errw, "d2firsthour:", err)
		return 2
	}
	defer f.Close()
	if _, err := f.WriteString(res.Row()); err != nil {
		fmt.Fprintln(errw, "d2firsthour:", err)
		return 2
	}
	fmt.Fprint(out, res.Row())
	return 0
}
