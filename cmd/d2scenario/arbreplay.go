package main

import (
	"fmt"
	"io"
	"os"
	"slices"

	"d2lod-bot/internal/arbiter"
)

const defaultOutcomes = "tools/provingground/arbiter-outcomes"

// cmdArbiterReplay scores the arbiter against the chain by OUTCOME (docs/arbiter.md, exit rule by outcome):
//
//	d2scenario arbiter-replay [-strict] FILE.json...
//	d2scenario arbiter-replay [-strict] [-follow] -session LOG.jsonl FILE.json...
//
// Each file holds decision points (an incident pre-state or a recorded divergence) and the data rules that say which
// invariant a choice would hit. Hard and soft come from invariants.json. Report-only tier: the exit code is 0 unless
// -strict is given and a family's verdict fails. With -session the cases are not scored: the rules of all files are
// applied to every tick of a recorded session (the bot's tick lines, knob bot.tickLogSec), chain against arbiter
// (arbiter.ScoreSession); -follow lets the simulated hero obey the arbiter as in d2arbiter -follow.
func cmdArbiterReplay(args []string, w io.Writer) error {
	fs := newFlags("arbiter-replay")
	strict := fs.Bool("strict", false, "exit 1 when a family has cases and does not pass")
	session := fs.String("session", "", "a recorded bot log (JSON lines with tick lines) to score tick by tick")
	follow := fs.Bool("follow", false, "with -session: the simulated hero obeys the arbiter")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		return fmt.Errorf("arbiter-replay: give at least one outcome file (see %s)", defaultOutcomes)
	}
	pol := loadInvariantPolicy()
	hard := func(n string) bool { return slices.Contains(pol.Hard, n) }
	var all []arbiter.CaseScore
	var files []arbiter.OutcomeFile
	for _, p := range fs.Args() {
		f, err := arbiter.LoadOutcomes(p)
		if err != nil {
			return err
		}
		files = append(files, f)
	}
	if *session != "" {
		lf, err := os.Open(*session)
		if err != nil {
			return err
		}
		defer lf.Close()
		res, err := arbiter.Replay(lf, arbiter.DefaultConfig(), arbiter.ShadowConfig{LogGapSec: 60}, *follow)
		if err != nil {
			return fmt.Errorf("arbiter-replay: %s: %w", *session, err)
		}
		all = arbiter.MergeRules(files...).ScoreSession(res.Points)
		fmt.Fprintf(w, "session %s: ticks %d, scored %d, logged area changes %d, arbiter area changes %d (follow=%v)\n", *session, res.Ticks, len(all), res.AreaChanges, res.FollowMoves, *follow)
		files = nil
	}
	for _, f := range files {
		for _, c := range f.Cases {
			s := f.Score(c, arbiter.DefaultConfig())
			all = append(all, s)
			fmt.Fprintf(w, "%-40s chain=%-9s hits=%v | arbiter=%-9s hits=%v | %s\n", s.ID, s.Chain, s.ChainHits, s.Arbiter, s.ArbiterHits, s.ArbiterIntent)
		}
	}
	failed := false
	fmt.Fprintln(w, "verdicts (hard, soft hits chain vs arbiter):")
	for _, v := range arbiter.Judge(all, hard) {
		// A session family without a hit on either side is no evidence either way; it does not fail -strict.
		quiet := *session != "" && v.ChainHard+v.ArbHard+v.ChainSoft+v.ArbSoft == 0
		if quiet {
			v.Note = "no hit on either side: no evidence"
		}
		fmt.Fprintf(w, "  %-9s cases=%d hard %d vs %d soft %d vs %d pass=%v (%s)\n", v.Family, v.Cases, v.ChainHard, v.ArbHard, v.ChainSoft, v.ArbSoft, v.Pass, v.Note)
		failed = failed || !v.Pass && !quiet
	}
	if *strict && failed {
		return exitErr{code: 1, msg: "arbiter-replay: a family does not pass"}
	}
	return nil
}
