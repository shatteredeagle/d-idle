package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/d2dctl"
	"d2lod-bot/internal/scenario"
	"d2lod-bot/internal/scenario/hero"
)

// smokeRows are the four prep paths the smoke tier runs on the fake d2d (review 2026-11-28 section 1.2): a legacy banked
// town start, a banked points scenario, a played hero with the `attack` row, and a field start.
var smokeRows = []scenario.Run{
	{ID: "town.resupply", Class: "Paladin", Seed: 11},
	{ID: "points.bank-by-default", Class: "Sorceress", Seed: 3},
	{ID: "combat.point-spell-hits", Class: "Sorceress", Seed: 11},
	{ID: "cain.scroll.field", Class: "Paladin", Seed: 11},
}

// smokeProv is the fixed provenance of the "can this exit be recorded" check: a real run fills it from its binaries.
var smokeProv = scenario.Provenance{BotCommit: "smoke", D2DSHA256: "smoke", Host: "smoke"}

// cmdSmoke is the cheap tier: no game, no slot, seconds. Stage A is static over every scenario and class (the setup
// builds, the T0 self-check, the pack holds the declared items, a BLOCKED exit is a valid ledger line); stage B runs the
// real prep half of exec-one (prepHero: setup verbs, point spending, prepSelfCheck) for smokeRows on a fake d2d. Any
// failure is exit 1; success prints one marker line a merge step can require.
func cmdSmoke(args []string, w io.Writer) error {
	fs := newFlags("smoke")
	dir := fs.String("dir", defaultDir, "scenario directory")
	if _, err := parseArgs(fs, args); err != nil {
		return err
	}
	scs, err := loadAll(*dir)
	if err != nil {
		return err
	}
	var bad []string
	for _, s := range scs {
		for _, c := range s.Matrix.Classes {
			if err := smokeStatic(s, c); err != nil {
				bad = append(bad, err.Error())
			}
		}
	}
	prepped, lines := 0, 0
	for _, r := range smokeRows {
		s, err := findScenario(scs, r.ID)
		if err != nil {
			bad = append(bad, "prep: "+err.Error())
			continue
		}
		res := smokePrep(s, r, &smokeGame{})
		if res.Status == scenario.Blocked {
			bad = append(bad, fmt.Sprintf("prep %s: BLOCKED %s", r.Key(), res.Reason))
			continue
		}
		prepped++
		if _, err := smokeLine(res); err != nil {
			bad = append(bad, fmt.Sprintf("prep %s: %v", r.Key(), err))
			continue
		}
		lines++
	}
	if len(bad) > 0 {
		return fmt.Errorf("smoke FAILED (%d):\n  %s", len(bad), strings.Join(bad, "\n  "))
	}
	fmt.Fprintf(w, "smoke: static %d scenarios, prep %d, ledger %d, blocked 0\n", len(scs), prepped, lines)
	return nil
}

// smokeStatic is stage A for one scenario and class.
func smokeStatic(s scenario.Scenario, class string) error {
	su, err := scenario.BuildSetup(s, class, s.Matrix.Seeds[0])
	if err != nil {
		return err
	}
	if err := staticCheck(s, class, su, hero.NameTables()); err != nil {
		return err
	}
	res := blocked("smoke: every exit is recordable")
	res.Run, res.Tags = scenario.Run{ID: s.ID, Class: class, Seed: s.Matrix.Seeds[0]}, su.Tags
	if _, err := smokeLine(res); err != nil {
		return fmt.Errorf("scenario %s (%s): a BLOCKED exit is not recordable: %w", s.ID, class, err)
	}
	return nil
}

func smokeLine(res scenario.Result) (scenario.LedgerRecord, error) {
	p := smokeProv
	res.Prov = &p
	return scenario.NewLedgerRecord(res, scenario.LedgerCtx{Job: "smoke", Started: time.Unix(0, 0)})
}

// smokePrep runs the prep half of one run on the fake d2d g (its zero value with the flags set; the class and the
// socket are filled here). The result is BLOCKED with the prep's reason, or carries the setup tags.
func smokePrep(s scenario.Scenario, r scenario.Run, g *smokeGame) scenario.Result {
	fail := func(why string, a ...any) scenario.Result {
		res := blocked(why, a...)
		res.Run = r
		return res
	}
	su, err := scenario.BuildSetup(s, r.Class, r.Seed)
	if err != nil {
		return fail("%v", err)
	}
	if len(su.Unsupported) > 0 {
		return fail("setup not buildable: %s", strings.Join(su.Unsupported, "; "))
	}
	if len(su.Seeds) > 0 {
		return fail("smoke: the fake d2d has no item seeding; pick a row without seeds")
	}
	live, err := startSmokeGame(r.Class)
	if err != nil {
		return fail("fake d2d: %v", err)
	}
	defer live.Close()
	live.NoHands = g.NoHands
	work, err := os.MkdirTemp("", "pgsmoke-work")
	if err != nil {
		return fail("%v", err)
	}
	defer os.RemoveAll(work)
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	c := d2dctl.New(live.Path(), d2dctl.Options{})
	defer c.Close()
	_, why := prepHero(ctx, c, s, r, su, hero.NameTables(), prepPaths{ledger: filepath.Join(work, "setup-ledger.jsonl"), data: work, state: work,
		hero: scenario.HeroName(s.ID, r.Class, r.Seed)})
	if why != "" {
		return fail("%s", why)
	}
	return scenario.Result{Run: r, Status: scenario.Green, Tags: su.Tags}
}
