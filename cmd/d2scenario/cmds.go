package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"d2lod-bot/internal/scenario"
)

func cmdList(args []string, w io.Writer) error {
	fs := newFlags("list")
	dir := fs.String("dir", defaultDir, "scenario directory")
	tier := fs.String("tier", "", "fast, full or nightly (includes lower tiers)")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	for _, s := range scenario.Select(scs, scenario.Selector{Tier: *tier}) {
		st := "active"
		if s.Pending() {
			st = "PENDING"
		}
		fmt.Fprintf(w, "%-30s %-9s %-6s %-9s %s\n", s.ID, s.Kind, s.Tier, st, s.Title)
	}
	return nil
}

func loadBehaviours(dir, board string) ([]scenario.Behaviour, error) {
	bs, err := scenario.LoadBehaviours(dir + "/behaviours.json")
	if err != nil {
		return nil, err
	}
	cs, err := scenario.ChunkBehaviours(board)
	if err != nil {
		return nil, err
	}
	return append(bs, cs...), nil
}

func hasBehaviour(bs []scenario.Behaviour, id string) bool {
	for _, b := range bs {
		if b.ID == id {
			return true
		}
	}
	return false
}

func selectorFrom(tier, changed, ids, class string, seed int64, seedSet bool) scenario.Selector {
	sel := scenario.Selector{Tier: tier, Class: class, Changed: splitList(changed), IDs: splitList(ids)}
	if seedSet {
		sel.Seed = &seed
	}
	return sel
}

func cmdPlan(args []string, w io.Writer) error {
	fs := newFlags("plan")
	dir := fs.String("dir", defaultDir, "scenario directory")
	tier := fs.String("tier", "fast", "fast, full or nightly")
	changed := fs.String("changed", "", "comma list of changed files (merge gate selection)")
	ids := fs.String("id", "", "comma list of ids or id prefixes ending in '.'")
	class := fs.String("class", "", "one class")
	seed := fs.Int64("seed", 0, "one seed")
	parallel := fs.Int("parallel", 2, "lab slots at once (at most 4)")
	asJSON := fs.Bool("json", false, "machine-readable plan")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	seedSet := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "seed" {
			seedSet = true
		}
	})
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	if *ids != "" && !flagSet(fs, "tier") {
		*tier = scenario.TierNightly // naming scenarios selects them whatever their tier
	}
	sel := selectorFrom(*tier, *changed, *ids, *class, *seed, seedSet)
	picked := scenario.Select(scs, sel)
	runs, err := scenario.Expand(picked, sel)
	if err != nil {
		return err
	}
	type planRun struct {
		scenario.Run
		Setup scenario.Setup `json:"setup"`
		Needs []string       `json:"needs,omitempty"`
		Build string         `json:"build"`
	}
	var out []planRun
	by := map[string]scenario.Scenario{}
	for _, s := range picked {
		by[s.ID] = s
	}
	for _, r := range runs {
		su, err := scenario.BuildSetup(by[r.ID], r.Class, r.Seed)
		if err != nil {
			return err
		}
		out = append(out, planRun{Run: r, Setup: su, Needs: by[r.ID].Needs, Build: by[r.ID].Build})
	}
	est := scenario.Estimate(picked, runs, *parallel)
	if *asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		return enc.Encode(map[string]any{"tier": *tier, "runs": out, "wall_min_upper_bound": est, "parallel": *parallel})
	}
	pend := 0
	for _, s := range picked {
		if s.Pending() {
			pend++
		}
	}
	fmt.Fprintf(w, "DRY RUN tier=%s: %d runs of %d scenarios (%d pending, never run), wall upper bound %d min at %d slots\n", *tier, len(out), len(picked)-pend, pend, est, *parallel)
	for _, r := range out {
		fmt.Fprintf(w, "  %-40s build=%s tags=%s verbs=%d needs=%v\n", r.Key(), r.Build, strings.Join(r.Setup.Tags, ","), len(r.Setup.Verbs), r.Needs)
	}
	return nil
}

func cmdSetup(args []string, w io.Writer) error {
	fs := newFlags("setup")
	dir := fs.String("dir", defaultDir, "scenario directory")
	class := fs.String("class", "Paladin", "class")
	seed := fs.Int64("seed", 11, "seed")
	id, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	s, err := findScenario(scs, id)
	if err != nil {
		return err
	}
	su, err := scenario.BuildSetup(s, *class, *seed)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", " ")
	return enc.Encode(map[string]any{"hero": scenario.HeroName(s.ID, *class, *seed), "setup": su, "provenance": s.Provenance})
}

func cmdEval(args []string, w io.Writer) error {
	fs := newFlags("eval")
	dir := fs.String("dir", defaultDir, "scenario directory")
	state := fs.String("state", "", "state file (didle.scenario-state/1)")
	id, err := parseArgs(fs, args)
	if err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	s, err := findScenario(scs, id)
	if err != nil {
		return err
	}
	var st scenario.State
	if err := readJSON(*state, &st); err != nil {
		return err
	}
	rs := scenario.Evaluate(s.Assert, st)
	status, reason := scenario.Verdict(rs)
	for _, r := range rs {
		fmt.Fprintf(w, "  %-18s ok=%-5v want %s | got %s\n", r.Kind, r.OK, r.Want, r.Got)
	}
	fmt.Fprintf(w, "%s %s\n", status, reason)
	if status != scenario.Green {
		return exitErr{3, ""}
	}
	return nil
}

func cmdCoverage(args []string, w io.Writer) error {
	fs := newFlags("coverage")
	dir := fs.String("dir", defaultDir, "scenario directory")
	board := fs.String("board", "data/questboard", "quest board data directory")
	asJSON := fs.Bool("json", false, "JSON rows")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	bs, err := loadBehaviours(*dir, *board)
	if err != nil {
		return err
	}
	rows, problems := scenario.Coverage(bs, scs)
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	if *asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", " ")
		return enc.Encode(rows)
	}
	scenario.CoverageReport(w, rows)
	return nil
}

func cmdReport(args []string, w io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: report results.json")
	}
	rs, err := readResults(args[0])
	if err != nil {
		return err
	}
	scenario.Report(w, rs)
	return nil
}

func cmdDeterminism(args []string, w io.Writer) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: determinism a.json b.json")
	}
	a, err := readResults(args[0])
	if err != nil {
		return err
	}
	b, err := readResults(args[1])
	if err != nil {
		return err
	}
	ok, vd, dd, missing := scenario.CheckDeterminism(a, b)
	for _, d := range vd {
		fmt.Fprintf(w, "DIFFERS (hard) %s: %s vs %s (%s)\n", d.Key, d.A, d.B, d.Detail)
	}
	for _, d := range dd {
		fmt.Fprintf(w, "drift (state digest, not a failure) %s: %s vs %s\n", d.Key, d.A, d.B)
	}
	for _, m := range missing {
		fmt.Fprintf(w, "missing in one execution: %s\n", m)
	}
	if !ok {
		return exitErr{3, "NOT DETERMINISTIC (a flaky scenario is a failed scenario)"}
	}
	fmt.Fprintf(w, "deterministic: %d runs, same verdicts and assertion digests (%d state-digest drifts)\n", len(a), len(dd))
	return nil
}

func cmdGateCheck(args []string, w io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: gate-check results.json")
	}
	rs, err := readResults(args[0])
	if err != nil {
		return err
	}
	scenario.Report(w, rs)
	if !scenario.Gate(rs) {
		return exitErr{3, "GATE: NOT GREEN"}
	}
	fmt.Fprintln(w, "GATE: GREEN")
	return nil
}

// cmdProvenance prints one table row per scenario in the format of docs/probes/test-characters.md, with the date and
// card left for the person who pastes them (a run that created heroes pastes them once per batch).
func cmdProvenance(args []string, w io.Writer) error {
	fs := newFlags("provenance")
	dir := fs.String("dir", defaultDir, "scenario directory")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	fmt.Fprintln(w, "| date | hero | class | card | how | setup writes (CHEATED / IMPOSSIBLE tags) |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|")
	for _, s := range scs {
		if s.Pending() {
			continue
		}
		su, err := scenario.BuildSetup(s, s.Matrix.Classes[0], s.Matrix.Seeds[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(w, "| DATE | Pg<scenario><class><seed> (fresh per run) | %s | provingground (%s) | `d2scenario run -id %s` on a lockstep lab slot, lab verbs for SETUP only, then the bot on a d2d without them | %s: %s |\n",
			strings.Join(s.Matrix.Classes, ", "), s.ID, s.ID, strings.Join(su.Tags, ", "), s.Provenance)
	}
	return nil
}
