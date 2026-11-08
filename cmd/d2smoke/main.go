// Command d2smoke judges the log of a smoke run (tools/smoke.sh, docs/smoke.md).
//
//	d2smoke judge -class C -log FILE [-commit H] [-session DIR] [-secret-file F] [-writes N] [-rules F] [-record DIR] [-profile P]
//	d2smoke info  -class C -field index|build|minutes [-rules F]
//
// Exit codes: 0 pass, 1 fail, 2 usage or input error.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/smoke"
)

func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }

func run(args []string, out, errw *os.File) int {
	if len(args) == 0 {
		fmt.Fprintln(errw, "usage: d2smoke judge|info ...")
		return 2
	}
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.SetOutput(errw)
	rules := fs.String("rules", "data/smoke/rules.json", "rules file")
	class := fs.String("class", "", "class name")
	logf := fs.String("log", "", "bot log")
	commit := fs.String("commit", "", "bot commit")
	session := fs.String("session", "", "session record directory (empty: no ladder)")
	secretFile := fs.String("secret-file", "", "file with one secret per line to look for in the session files")
	writes := fs.Int("writes", -1, "writes after start from d2cheat ledger --since-run (-1 unknown)")
	minutes := fs.Int("minutes", 0, "virtual minutes judged (default from the rules)")
	record := fs.String("record", "", "write smoke-<stamp>-<class>.md into this directory (a single dash prints it after a marker line)")
	profile := fs.String("profile", "", "profile name for the record")
	field := fs.String("field", "", "info: index|build|minutes")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	r, err := smoke.LoadRules(*rules)
	if err != nil {
		fmt.Fprintln(errw, err)
		return 2
	}
	switch args[0] {
	case "info":
		c, err := r.Class(*class)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
		switch *field {
		case "index":
			fmt.Fprintln(out, c.D2DIndex)
		case "build":
			fmt.Fprintln(out, c.Build)
		case "minutes":
			fmt.Fprintln(out, r.Minutes)
		default:
			fmt.Fprintln(errw, "info: -field index|build|minutes")
			return 2
		}
		return 0
	case "judge":
		f, err := os.Open(*logf)
		if err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
		defer f.Close()
		var secrets []string
		if *secretFile != "" {
			b, err := os.ReadFile(*secretFile)
			if err != nil {
				fmt.Fprintln(errw, err)
				return 2
			}
			for _, s := range strings.Split(string(b), "\n") {
				if s = strings.TrimSpace(s); s != "" {
					secrets = append(secrets, s)
				}
			}
		}
		res, err := smoke.Judge(r, smoke.Input{Class: *class, Commit: *commit, Log: f, SessionDir: *session, Secrets: secrets, Writes: *writes, Minutes: *minutes})
		if err != nil {
			fmt.Fprintln(errw, err)
			return 2
		}
		fmt.Fprintln(out, res.Line())
		if *record != "" {
			stamp := time.Now().UTC().Format("20060102T150405Z")
			if *record == "-" { // tools/smoke.sh files the record on the calling side
				fmt.Fprintf(out, "--- RECORD smoke-%s-%s.md ---\n%s", stamp, res.Class, res.Record(stamp, *profile))
				if !res.Pass {
					return 1
				}
				return 0
			}
			p := filepath.Join(*record, "smoke-"+stamp+"-"+res.Class+".md")
			if err := os.WriteFile(p, []byte(res.Record(stamp, *profile)), 0o644); err != nil {
				fmt.Fprintln(errw, err)
				return 2
			}
			fmt.Fprintln(errw, "record:", p)
		}
		if !res.Pass {
			return 1
		}
		return 0
	}
	fmt.Fprintln(errw, "usage: d2smoke judge|info ...")
	return 2
}
