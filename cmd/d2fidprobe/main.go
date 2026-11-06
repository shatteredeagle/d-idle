// Command d2fidprobe compares one character of one save, read from game.exe and from d2d, field by field
// (docs/research/fidelity-probe-design.md, docs/probes/fidprobe.md). Read-only on both sides: no input, no
// memory writes, no cheats, d2d lab verbs off. Exit 0 only when no field is a MISMATCH.
//
//	d2fidprobe -list
//	d2fidprobe -scenario 1 -saves DIR -instance i1 -sock SLOT.sock [-out DIR] [-format text|json]
//	d2fidprobe -scenario 1 -a json:a.json -b json:b.json      (offline replay)
//
// The operator loads <role>.d2s from -saves into both sides first (the tool only checks the file exists).
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"d2lod-bot/internal/buildinfo"
	"d2lod-bot/internal/fidprobe"
)

type flags struct {
	instance, sock, tables string
}

func main() {
	buildinfo.Exit("d2fidprobe")
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("d2fidprobe", flag.ContinueOnError)
	fs.SetOutput(stderr)
	var f flags
	list := fs.Bool("list", false, "print the scenario catalogue and exit")
	scen := fs.String("scenario", "", "scenario id or name")
	saves := fs.String("saves", "", "directory holding <role>.d2s files (never in the repo)")
	a := fs.String("a", "gameexe", "side A: gameexe | d2d | json:FILE")
	b := fs.String("b", "d2d", "side B: gameexe | d2d | json:FILE")
	fs.StringVar(&f.instance, "instance", os.Getenv("D2BOT_INSTANCE"), "game.exe instance (i1..i5), default $D2BOT_INSTANCE")
	fs.StringVar(&f.sock, "sock", os.Getenv("D2D_SOCK"), "d2d control socket, default $D2D_SOCK")
	fs.StringVar(&f.tables, "tables", "", "extracted game table cache dir for d2dsrc item rows")
	out := fs.String("out", "", "also write <scenario>.txt and .json here")
	format := fs.String("format", "text", "stdout format: text or json")
	all := fs.Bool("all", false, "list EQUAL cells too")
	reads := fs.Int("reads", 3, "reads per side that must agree")
	gap := fs.Duration("gap", time.Second, "pause between reads")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *list {
		scs, err := fidprobe.Scenarios()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		for _, s := range scs {
			fmt.Fprintf(stdout, "%2d %-28s role=%-16s %-10s slice1=%-5v %v\n", s.ID, s.Name, s.Role, s.Class, s.Slice1, s.Groups)
		}
		return 0
	}
	sc, err := fidprobe.FindScenario(*scen)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if *saves != "" {
		if _, err := os.Stat(filepath.Join(*saves, sc.Role+".d2s")); err != nil {
			fmt.Fprintf(stderr, "d2fidprobe: scenario %s needs %s.d2s in %s: %v\n", sc.Name, sc.Role, *saves, err)
			return 2
		}
	}
	rules, err := fidprobe.DefaultRules()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	sa, err := newSource(*a, "A", &f)
	if err == nil {
		var sb fidprobe.Source
		if sb, err = newSource(*b, "B", &f); err == nil {
			return probe(sc, sa, sb, rules, *reads, *gap, *out, *format, *all, stdout, stderr)
		}
	}
	fmt.Fprintln(stderr, err)
	return 2
}

func probe(sc fidprobe.Scenario, a, b fidprobe.Source, rules *fidprobe.Rules, reads int, gap time.Duration, out, format string, all bool, stdout, stderr io.Writer) int {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	rep, err := fidprobe.Run(ctx, sc, a, b, reads, gap, rules)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	text := rep.Text(all)
	js, err := rep.JSON()
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if format == "json" {
		stdout.Write(append(js, '\n'))
	} else {
		io.WriteString(stdout, text)
	}
	if out != "" {
		if err := os.MkdirAll(out, 0o755); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		base := filepath.Join(out, sc.Name)
		if err := os.WriteFile(base+".txt", []byte(text), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
		if err := os.WriteFile(base+".json", append(js, '\n'), 0o644); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if !rep.Pass() {
		return 3
	}
	return 0
}
