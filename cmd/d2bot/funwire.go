package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/watch"
)

// The fun loop on the real bot (docs/return-screen.md, docs/tally-strip.md, docs/codex.md "Wiring"): the return screen,
// the tally strip with its charter sink and the Act 1 Codex, all behind -watch. With a scripted player (--ladder-profile)
// or without -watch nothing here runs and the catalogue stays the shipped one.

// useFirsts adds the Codex point to the runner's ladder catalogue when a person plays through the watch screen. Call it
// before startLadder.
func useFirsts(l ladderOptions, r *bot.Runner) error {
	if !l.on || !l.watch || l.profile != "" {
		return nil
	}
	tbl, err := decide.ShippedFirsts()
	if err != nil {
		return err
	}
	r.SetLadderFirsts(tbl)
	return nil
}

// useFun hands the waypoint edge of the runner to the fun loop. Call it after startLadder; no rig or no Fun: nothing.
func useFun(rig *ladderRig, r *bot.Runner, run string) {
	if rig == nil || rig.Fun == nil {
		return
	}
	r.SetWaypointHook(func(id int, name string) { rig.Fun.OnWaypoint(name, fmt.Sprintf("%s:wp:%d", run, id)) })
}

// townNames are the five towns by the names the bot prints in area_change events (U beyond Act 1: the tally strip
// only needs the names to tell a trip's end).
func townNames() map[string]bool {
	out := map[string]bool{}
	for act := 0; act < 5; act++ {
		out[status.AreaName(data.Town(act))] = true
	}
	return out
}

// runID is unique per process, so the tally's event ids (run plus the manager's sequence number, which restarts with
// the process) never collide with an earlier process's ledger keys.
func runID() string {
	name := os.Getenv("D2BOT_RUN")
	if name == "" {
		name = "run"
	}
	return fmt.Sprintf("%s-%d", name, time.Now().Unix())
}

// newFun builds the fun loop for a running ladder: the tally strip when the profile has a d.idle wallet, the Codex when
// the catalogue carries the firsts point (useFirsts), and the return screen over both.
func newFun(l ladderOptions, stateDir string, hub *watch.Hub, wallet *currency.Wallet, st *decide.Store, cat *decide.Catalogue, clk decide.Clock) (*watch.Fun, error) {
	run := runID()
	cfg := watch.FunConfig{Hub: hub, Speed: 1}
	if stateDir != "" {
		cfg.Visit = watch.NewVisit(filepath.Join(stateDir, "visit.json"), nil)
	}
	if wallet != nil && wallet.Ledger != nil {
		tally, err := currency.NewTally(wallet.Ledger, currency.DefaultTallyRows(), l.storeProfile(), run)
		if err != nil {
			return nil, err
		}
		towns := townNames()
		if cfg.Strip, err = watch.NewTallyStrip(hub, wallet, tally, func(a string) bool { return towns[a] }); err != nil {
			return nil, err
		}
	}
	if _, err := cat.Point(decide.FirstsPoint); err == nil {
		tbl, err := decide.ShippedFirsts()
		if err != nil {
			return nil, err
		}
		if cfg.Codex, err = openCodex(stateDir, l.storeProfile()); err != nil {
			return nil, err
		}
		cfg.Table, cfg.Clock = tbl, clk
		cfg.Tracker = func(class string) (*decide.Tracker, error) {
			return decide.NewTracker(tbl, cat, st, cfg.Codex, clk, decide.Who{Class: class, Diff: "normal"})
		}
	}
	pt, err := decide.ShippedPolicy(cat)
	if err != nil {
		return nil, err
	}
	cfg.Policy = &watch.PolicyConfig{Table: pt, Knobs: l.knobs}
	f, err := watch.NewFun(cfg)
	if err != nil {
		return nil, err
	}
	if err := f.Policy.Reapply(); err != nil {
		return nil, fmt.Errorf("policy sheet: %w", err)
	}
	return f, nil
}

// knobSetter is the policy sheet's knob path: the tuner's own edit (range checks, all or nothing, saved to tuning.json).
func knobSetter(t *bot.Tuner) watch.KnobSetter {
	return func(over map[string]*float64) error { return t.Edit(api.TuneEdit{Overrides: over}) }
}

func openCodex(stateDir, profile string) (*decide.Codex, error) {
	if stateDir == "" {
		return decide.NewCodex(profile), nil
	}
	return decide.OpenCodex(stateDir, profile)
}
