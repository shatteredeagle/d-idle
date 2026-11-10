package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/currency"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/unlock"
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
func newFun(l ladderOptions, stateDir string, hub *watch.Hub, wallet *currency.Wallet, st *decide.Store, cat *decide.Catalogue, clk decide.Clock, notes watch.NotesSink, status func() goals.Status, halt func() string) (*watch.Fun, error) {
	run := runID()
	cfg := watch.FunConfig{Hub: hub, Speed: 1, Notes: notes}
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
	var err error
	if cfg.Unlock, err = openUnlocks(stateDir, l.storeProfile()); err != nil {
		return nil, err
	}
	if cfg.Proposals, err = newProposals(l, stateDir, clk); err != nil {
		return nil, err
	}
	pt, err := decide.ShippedPolicy(cat)
	if err != nil {
		return nil, err
	}
	cfg.Policy = &watch.PolicyConfig{Table: pt, Knobs: l.knobs}
	if cfg.Hall, err = newHallConfig(wallet, l.storeProfile(), status); err != nil {
		return nil, err
	}
	var fun *watch.Fun // the risk facts read the proposal desk of the Fun built below
	if cfg.Risk, err = newRiskConfig(l, func() string { return nextZone(fun) }); err != nil {
		return nil, err
	}
	cfg.OrderQueue = &watch.OrderQueueConfig{Halt: halt}
	if cfg.Strip != nil {
		cfg.OrderQueue.Quote = cfg.Strip.Quote
	}
	if m := l.monsterTables(); m != nil {
		cfg.Monsters = m
		if stateDir != "" {
			cfg.BestiaryPath = filepath.Join(stateDir, "bestiary-"+l.storeProfile()+".json")
		}
	} else {
		slog.Info("bestiary: no monster table, cards carry the flavour only")
	}
	f, err := watch.NewFun(cfg)
	if err != nil {
		return nil, err
	}
	fun = f
	for _, e := range f.Errors() {
		slog.Warn("fun: " + e)
	}
	if os.Getenv("D2BOT_RUN") != "" {
		// A named run (gate, soak, drill) keeps today's knobs: the class start of the stance row is for the player's game.
		f.Policy.SetStarts(func(string) (map[string]string, error) { return nil, nil })
	}
	if err := f.Policy.Reapply(); err != nil {
		return nil, fmt.Errorf("policy sheet: %w", err)
	}
	return f, nil
}

// newProposals builds the zone proposer's config (docs/proposezone.md): the shipped table, the tally zone table as the
// one range source, a state file next to the other ladder state, the knob values for the repeat rule. nil (no error)
// when the backend gave no facts reader or a scripted player answers.
func newProposals(l ladderOptions, stateDir string, clk decide.Clock) (*watch.ProposalConfig, error) {
	if l.proposalFacts == nil || l.profile != "" {
		return nil, nil
	}
	tbl, err := decide.ShippedProposeTable()
	if err != nil {
		return nil, err
	}
	ranges := tallyRanges()
	path := ""
	if stateDir != "" {
		path = filepath.Join(stateDir, "proposals-"+l.storeProfile()+".json")
	}
	pr, err := decide.NewProposer(tbl, ranges, clk, path)
	if err != nil {
		return nil, err
	}
	pr.SetRepeatPolicy(time.Duration(l.repeatCooldownMin)*time.Minute, l.quietAfter)
	return &watch.ProposalConfig{Proposer: pr, Clock: clk, Facts: l.proposalFacts, AwayMin: l.awayMin}, nil
}

// tallyRanges is the one zone range source: the tally zone table.
func tallyRanges() decide.RangeFn {
	rows := currency.DefaultTallyRows()
	return func(area string) (int, int, bool) {
		for _, z := range rows.Zones {
			if strings.EqualFold(z.Area, strings.TrimSpace(area)) {
				return z.Min, z.Max, true
			}
		}
		return 0, 0, false
	}
}

// monsterTables is the monster table accessor for the bestiary: nil without tables (a nil *data.DB must not become a
// non-nil interface).
func (l ladderOptions) monsterTables() watch.MonsterTables {
	if l.monsters == nil {
		return nil
	}
	return l.monsters
}

// newRiskConfig builds the risk meter's config (docs/unlock-risk.md): the proposer table and the tally zone ranges as the
// shared sources and the runner's facts. next names the zone the hero proposes next ("" = the area it stands in). nil (no
// error) when the backend gave no facts reader or a scripted player answers.
func newRiskConfig(l ladderOptions, next func() string) (*watch.RiskConfig, error) {
	if l.riskFacts == nil || l.profile != "" {
		return nil, nil
	}
	pt, err := decide.ShippedProposeTable()
	if err != nil {
		return nil, err
	}
	return &watch.RiskConfig{Propose: pt, Ranges: tallyRanges(), Facts: func(zone string) watch.RiskFacts {
		if zone == "" {
			zone = next()
		}
		return l.riskFacts(zone)
	}}, nil
}

// nextZone is the zone of the first pending proposal, "" without one.
func nextZone(f *watch.Fun) string {
	if f == nil || f.Proposals == nil {
		return ""
	}
	if p := f.Proposals.View().Pending; len(p) > 0 {
		return p[0].To
	}
	return ""
}

// knobSetter is the policy sheet's knob path: the tuner's own edit (range checks, all or nothing, saved to tuning.json).
func knobSetter(t *bot.Tuner) watch.KnobSetter {
	return func(over map[string]*float64) error { return t.Edit(api.TuneEdit{Overrides: over}) }
}

// openUnlocks opens the unlock ladder of the profile (docs/unlock-ladder.md); without a state dir it lives in memory.
func openUnlocks(stateDir, profile string) (*unlock.Engine, error) {
	lad, err := unlock.Shipped()
	if err != nil {
		return nil, err
	}
	return unlock.Open(stateDir, profile, lad, nil)
}

func openCodex(stateDir, profile string) (*decide.Codex, error) {
	if stateDir == "" {
		return decide.NewCodex(profile), nil
	}
	return decide.OpenCodex(stateDir, profile)
}

// currencyDirFor is the per-profile account ledger directory under the profile's state dir (the same for every
// backend, so the tally survives a backend change).
func currencyDirFor(stateDir string) string {
	return envOr("D2BOT_CURRENCY_DIR", filepath.Join(stateDir, "currency"))
}

// stopSelf is the bot's graceful shutdown: SIGTERM to this process, the path `docker stop` and the updater take (hero
// saved, d2d quit in managed mode; docs/tester-live.md). A seam for tests.
var stopSelf = func() error { return signalSelfTerm() }

// newHallConfig wires the Hall of Fame (docs/hall-of-fame.md "Wiring"): the ledger of the account, the shipped title
// table, the hero's facts from the status poll and the shutdown path. nil (no error) without an account ledger or a
// status source: the Hall needs both, and the rest of the fun loop does not.
func newHallConfig(wallet *currency.Wallet, profile string, status func() goals.Status) (*watch.HallConfig, error) {
	if wallet == nil || wallet.Ledger == nil || status == nil {
		return nil, nil
	}
	titles, err := watch.ShippedHallTitles()
	if err != nil {
		return nil, err
	}
	start := time.Now()
	return &watch.HallConfig{
		Store: &watch.HallStore{Ledger: wallet.Ledger, Titles: titles},
		Stop:  func() error { return stopSelf() },
		Facts: func() watch.HallFacts {
			c := status().Character
			f := watch.HallFacts{Name: c.Name, Class: strings.ToLower(c.Class), Level: c.Level, WallSecs: time.Since(start).Seconds()}
			if c.Name != "" {
				f.ID = profile + ":" + c.Name // U: no creation stamp in the status, so a recreated name is the same hero
			}
			// Permanent stays false: no event says a death is final (hardcore play does not exist yet, docs/pact-away.md).
			return f
		},
	}, nil
}
