package main

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"d2lod-bot/internal/scenario"
)

// cmdNightly prints the night's run set from tools/provingground/nightly.json (card pgnightly). It starts nothing: the
// scheduler job in the ops repo runs the printed commands inside the window.
//
//	nightly check                         validate the plan file and its stage tiers against tiers.json
//	nightly plan [-day D] [-format text|json|cmds] [-quarantined a,b]
func cmdNightly(args []string, w io.Writer) error {
	if len(args) == 0 || (args[0] != "check" && args[0] != "plan") {
		return fmt.Errorf("usage: nightly check|plan [-file F] [-dir D] [-day YYYY-MM-DD] [-format text|json|cmds] [-quarantined id,id]")
	}
	sub := args[0]
	fs := newFlags("nightly " + sub)
	file := fs.String("file", "tools/provingground/nightly.json", "nightly plan")
	dir := fs.String("dir", defaultDir, "scenario directory")
	dayS := fs.String("day", "", "night of this date, EST (default today)")
	format := fs.String("format", "text", "text, json, or cmds (run.sh argument lines)")
	quar := fs.String("quarantined", "", "scenario ids to leave out")
	if _, err := parseArgs(fs, args[1:]); err != nil {
		return err
	}
	p, err := scenario.LoadNightly(*file)
	if err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	tiers, err := scenario.LoadTiers(*dir + "/tiers.json")
	if err != nil {
		return err
	}
	day := time.Now().UTC().Add(-5 * time.Hour)
	if *dayS != "" {
		if day, err = time.Parse("2006-01-02", *dayS); err != nil {
			return fmt.Errorf("-day: %w", err)
		}
	}
	q := map[string]bool{}
	for _, id := range splitList(*quar) {
		q[id] = true
	}
	runs, notes := p.Expand(scs, tiers, q, day)
	if sub == "check" {
		if len(runs) == 0 {
			return fmt.Errorf("the plan selects no run on %s", day.Format("2006-01-02"))
		}
		fmt.Fprintf(w, "nightly: ok, %d runs in %d jobs on %s (%d notes)\n", len(runs), len(p.Jobs), day.Format("2006-01-02"), len(notes))
		return nil
	}
	switch *format {
	case "json":
		return json.NewEncoder(w).Encode(struct {
			Day   string                `json:"day"`
			Runs  []scenario.PlannedRun `json:"runs"`
			Notes []string              `json:"notes,omitempty"`
		}{day.Format("2006-01-02"), runs, notes})
	case "cmds":
		for _, l := range runLines(runs) {
			fmt.Fprintln(w, l)
		}
	case "text":
		for _, r := range runs {
			fmt.Fprintf(w, "%-8s %-22s %-34s %-12s %d %s\n", r.Job, r.Stage, r.Scenario, r.Class, r.Seed, strings.Repeat("p", min(r.Pair, 1)))
		}
		for _, n := range notes {
			fmt.Fprintln(w, "note:", n)
		}
	default:
		return fmt.Errorf("-format %q", *format)
	}
	return nil
}

// runLines groups the runs into `<job> <run.sh arguments>` lines: one line per job, class and seed for the tier stages
// (the scenario ids joined, `-id` takes a list) and one determinism-run line per pair.
func runLines(runs []scenario.PlannedRun) []string {
	type key struct {
		job, class string
		seed       int64
	}
	var order []key
	ids := map[key][]string{}
	var out []string
	for _, r := range runs {
		if r.Pair == 2 {
			continue // the pair is one determinism-run -n 2
		}
		if r.Pair == 1 {
			out = append(out, fmt.Sprintf("%s determinism-run -n 2 -id %s -class %s -seed %d", r.Job, r.Scenario, r.Class, r.Seed))
			continue
		}
		k := key{r.Job, r.Class, r.Seed}
		if _, ok := ids[k]; !ok {
			order = append(order, k)
		}
		ids[k] = append(ids[k], r.Scenario)
	}
	var tier []string
	for _, k := range order {
		tier = append(tier, fmt.Sprintf("%s -id %s -class %s -seed %d", k.job, strings.Join(ids[k], ","), k.class, k.seed))
	}
	return append(tier, out...)
}
