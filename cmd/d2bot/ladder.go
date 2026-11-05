package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/status"
	"d2lod-bot/internal/watch"
)

// The decision ladder and the player screen on the real bot (docs/t0-slice.md "How to run it"). Both are off by default:
// without -ladder nothing in this file runs, no directory is made and no port is opened.

type ladderOptions struct {
	on      bool
	profile string // "" = a person answers through the watch screen; NAME or holdout:SEED = a scripted player
	watch   bool
	store   string // "" = a store per D2BOT_RUN when a run is named, else the shared one; "shared" = the shared one; else a named store
	// -watch: the read-only viewer and the SEPARATE answer listener (same semantics as d2dview -watch).
	viewAddr, answerAddr, origins string
}

func ladderFlags(fs *flag.FlagSet, l *ladderOptions) {
	fs.BoolVar(&l.on, "ladder", envBool("D2BOT_LADDER"), "turn the decision ladder on (knob ladder.enabled 1): the T0/T1 points ask a store whose untrained answer is the legacy one; state under <instance state>/ladder (env D2BOT_LADDER; default off)")
	fs.StringVar(&l.profile, "ladder-profile", envOr("D2BOT_LADDER_PROFILE", ""), "with --ladder: a scripted player for drills (cautious, bold, inconsistent, holdout:SEED); default none, a person answers on the watch screen")
	fs.StringVar(&l.store, "ladder-store", envOr("D2BOT_LADDER_STORE", ""), "with --ladder: the store under <instance state>/ladder/runs/NAME; default the run named by D2BOT_RUN (so drills are independent), or the shared store when no run is named; \"shared\" forces the shared store (learning carries on across sessions)")
	fs.BoolVar(&l.watch, "watch", envBool("D2BOT_WATCH"), "with --ladder: serve the player screen (/watch) from this process (env D2BOT_WATCH; default off)")
	fs.StringVar(&l.viewAddr, "watch-addr", envOr("D2BOT_WATCH_ADDR", "127.0.0.1:8891"), "with --watch: listen address of the read-only viewer; non-loopback requires --token")
	fs.StringVar(&l.answerAddr, "watch-answer-addr", envOr("D2BOT_WATCH_ANSWER_ADDR", "127.0.0.1:8892"), "with --watch: listen address of the SEPARATE answer route (answer, confirm, revoke, orders); non-loopback requires --token")
	fs.StringVar(&l.origins, "watch-origin", envOr("D2BOT_WATCH_ORIGIN", ""), "with --watch: comma list of page origins the answer route accepts (default: the viewer's port on localhost)")
}

func (l ladderOptions) validate(sim bool) error {
	if !l.on {
		switch {
		case l.watch:
			return errors.New("--watch needs --ladder: the screen shows the ladder's thoughts")
		case l.profile != "":
			return errors.New("--ladder-profile needs --ladder")
		}
		return nil
	}
	if sim {
		return errors.New("--ladder needs the real bot runner: use --attach or --managed, not --sim")
	}
	if l.profile == "" && !l.watch {
		return errors.New("--ladder needs someone to answer: add --watch (a person) or --ladder-profile (a scripted player)")
	}
	if l.profile != "" {
		if _, _, err := l.source(); err != nil {
			return err
		}
	}
	return nil
}

// source parses -ladder-profile into the decide profile name and seed.
func (l ladderOptions) source() (name string, seed int64, err error) {
	name = l.profile
	if rest, ok := strings.CutPrefix(name, "holdout:"); ok {
		if seed, err = strconv.ParseInt(rest, 10, 64); err != nil {
			return "", 0, fmt.Errorf("--ladder-profile holdout:SEED needs a whole number, got %q", rest)
		}
		name = "holdout"
	}
	if _, err := decide.NewProfile(name, seed); err != nil {
		return "", 0, fmt.Errorf("--ladder-profile: %w", err)
	}
	return name, seed, nil
}

// storeProfile is the store's profile (file name) for this run: the scripted player or "human".
func (l ladderOptions) storeProfile() string {
	if l.profile == "" {
		return "human"
	}
	return strings.ReplaceAll(l.profile, ":", "-")
}

// ladderRig is what startLadder built; nil fields when -watch is off.
type ladderRig struct {
	Ladder *ladder.Ladder
	Hub    *watch.Hub
	Served watch.Served
}

// enableFunc is bot.(*Runner).EnableLadderWith (a seam for tests).
type enableFunc func(stateDir, profile string, wire bot.LadderWire) (*ladder.Ladder, error)

// startLadder installs the ladder on the runner (call after the goal manager exists and before the runner starts),
// attaches the watch Hub as the thought sink and answer source when -watch is on, and feeds the Hub from the goal
// manager's events and the runner's status. here names the area the hero stands in, for "fight here".
func startLadder(ctx context.Context, l ladderOptions, token, stateDir string, enable enableFunc, m *goals.Manager, status func() goals.Status, here func() string) (*ladderRig, error) {
	if !l.on {
		return nil, nil
	}
	rig := &ladderRig{}
	var orders *watch.GoalsOrders
	wire := func(st *decide.Store, cat *decide.Catalogue, clk decide.Clock) (ladder.Sink, ladder.AnswerSource, error) {
		var src ladder.AnswerSource
		if l.profile != "" {
			name, seed, err := l.source()
			if err != nil {
				return nil, nil, err
			}
			if src, err = ladder.NewProfileSource(name, seed); err != nil {
				return nil, nil, err
			}
		}
		if !l.watch {
			return nil, src, nil
		}
		orders = &watch.GoalsOrders{Q: m, Here: here, Rules: func() []string { return liveRuleKeys(st) }}
		hub, err := watch.New(watch.Config{Store: st, Cat: cat, Clock: clk, Areas: watch.OrderableAreas(), Orders: orders, WaitForSink: true})
		if err != nil {
			return nil, nil, err
		}
		orders.Hub = hub
		rig.Hub = hub
		if src == nil {
			src = hub // a person answers: the Hub never answers by itself
		}
		return hub, src, nil
	}
	lad, err := enable(stateDir, l.storeProfile(), wire)
	if err != nil {
		return nil, err
	}
	if lad == nil {
		return nil, errors.New("--ladder: the ladder.enabled knob is off")
	}
	rig.Ladder = lad
	if rig.Hub == nil {
		return rig, nil
	}
	served, err := rig.Hub.Serve(ctx, watch.ServeOptions{ViewAddr: l.viewAddr, AnswerAddr: l.answerAddr, Token: token, Origins: l.origins})
	if err != nil {
		ladder.Install(nil)
		return nil, err
	}
	rig.Served = served
	go rig.Hub.RunTicker(ctx, time.Second)
	go feedHub(ctx, rig.Hub, orders, m, status)
	return rig, nil
}

func liveRuleKeys(st *decide.Store) []string {
	var keys []string
	for _, r := range st.Rules() {
		keys = append(keys, r.Key)
	}
	return keys
}

// feedHub forwards the goal manager's events (orders, deaths, kills, area changes) and a status poll to the Hub: the
// uptime source (docs/research/watch-screen-design.md, 3.5).
func feedHub(ctx context.Context, hub *watch.Hub, o *watch.GoalsOrders, m *goals.Manager, status func() goals.Status) {
	ch, cancel := m.Subscribe()
	defer cancel()
	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			o.OnGoalEvent(ev)
		case <-tick.C:
			if status != nil {
				o.OnStatus(status())
			}
		}
	}
}

// ladderEnv switches the knob when the flag asks for it; the knob stays what the config file says otherwise.
func ladderEnv(l ladderOptions, t *env.Tuning) {
	if l.on {
		t.Ladder.Enabled = 1
	}
}

// ladderPost is ladderEnv as a Tuner hook: a saved tuning file or a viewer edit rebuilds the whole tuning and must not
// turn the flag off. Nil without the flag, so the config file and tuning.json stay authoritative.
func ladderPost(l ladderOptions) func(*env.Tuning) {
	if !l.on {
		return nil
	}
	return func(t *env.Tuning) { ladderEnv(l, t) }
}

// hereFunc names the area of the runner's last snapshot ("" when unknown).
func hereFunc(r *bot.Runner) func() string {
	return func() string {
		s := r.LastSnapshot()
		if s == nil || s.Player == nil {
			return ""
		}
		if c := r.Catalog(); c != nil {
			if n := c.AreaName(s.Player.Area); n != "" {
				return n
			}
		}
		return status.AreaName(s.Player.Area)
	}
}

// ladderDir is the decision store's directory, made only when -ladder is on. The shared store is <state>/ladder and
// carries learning across sessions; a named run (--ladder-store, else D2BOT_RUN) gets <state>/ladder/runs/NAME, so
// two drills on one instance do not see each other's thoughts and rules (t0drill Result 2, defect 7).
func ladderDir(stateDir string, l ladderOptions) (string, error) {
	shared := filepath.Join(stateDir, "ladder")
	name := l.store
	if !l.on {
		return shared, nil
	}
	if name == "" {
		name = os.Getenv("D2BOT_RUN")
	}
	if name == "" || name == "shared" {
		return shared, nil
	}
	if strings.ContainsAny(name, `/\:`) || strings.Contains(name, "..") || name[0] == '.' {
		return "", fmt.Errorf("--ladder-store / D2BOT_RUN %q is not a plain name", name)
	}
	return filepath.Join(shared, "runs", name), nil
}

// logLadder prints the one startup line.
func logLadder(log *slog.Logger, l ladderOptions, rig *ladderRig, dir string) {
	if rig == nil {
		return
	}
	args := []any{"profile", l.storeProfile(), "dir", dir}
	if rig.Hub != nil {
		args = append(args, "view", rig.Served.View.String(), "answer", rig.Served.Answer.String())
	}
	log.Info("ladder on", args...)
}
