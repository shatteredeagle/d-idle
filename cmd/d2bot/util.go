package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/leveler"
	"d2lod-bot/internal/paths"
	"d2lod-bot/internal/task"
)

func sprintf(f string, a ...any) string { return fmt.Sprintf(f, a...) }

// logEvents writes every goal-manager event to the log.
func logEvents(ctx context.Context, log *slog.Logger, m *goals.Manager) {
	ch, cancel := m.Subscribe()
	defer cancel()
	for {
		select {
		case <-ctx.Done():
			return
		case ev, ok := <-ch:
			if !ok {
				return
			}
			attrs := []any{"type", ev.Type, "goal", ev.GoalID}
			if ev.Monster != "" {
				attrs = append(attrs, "monster", ev.Monster)
			}
			if ev.Area != "" {
				attrs = append(attrs, "area", ev.Area)
			}
			lvl := slog.LevelInfo
			switch ev.Type {
			case goals.EvError, goals.EvDeath:
				lvl = slog.LevelWarn
			case goals.EvKill, goals.EvPotion:
				lvl = slog.LevelDebug
			}
			log.Log(ctx, lvl, ev.Message, attrs...)
		}
	}
}

// enqueueGoals parses "farm:Blood Moor;reach:3;idle;boss:Andariel" and adds the goals to the manager.
func enqueueGoals(m *goals.Manager, spec string) error {
	seen := map[string]int{}
	for _, part := range splitSemi(spec) {
		kind, arg, _ := strings.Cut(part, ":")
		var g goals.Goal
		switch kind {
		case "farm":
			g = goals.Goal{Type: goals.FarmArea, Params: goals.Params{Area: arg}}
		case "reach":
			n, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("goal %q: %w", part, err)
			}
			g = goals.Goal{Type: goals.ReachLevel, Params: goals.Params{Level: n}}
		case "boss":
			g = goals.Goal{Type: goals.KillBoss, Params: goals.Params{Boss: arg, Count: 1}}
		case "idle":
			g = goals.Goal{Type: goals.Idle}
		case "campaign":
			// Chaining the next difficulty is on for campaigns started from the command line; "campaign:T:nochain" opts out.
			tmpl, opt, _ := strings.Cut(arg, ":")
			g = goals.Goal{Type: goals.Campaign, Params: goals.Params{Template: tmpl, Chain: opt != "nochain"}}
		case "runs":
			var err error
			if g, err = goals.ParseRuns(part); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unknown goal %q (use farm:AREA, reach:LEVEL, boss:NAME, campaign[:TEMPLATE], runs:NAME:N, idle)", part)
		}
		// One key per --goals entry (plus its occurrence, so a deliberate repeat inside one string still works):
		// editing the string adds only the new entries, an unchanged one is never queued again, whatever its state.
		seen[part]++
		sum := sha256.Sum256([]byte(fmt.Sprintf("%s#%d", part, seen[part])))
		_, added, err := m.AddSeeded(g, hex.EncodeToString(sum[:6]))
		if err != nil {
			return fmt.Errorf("goal %q: %w", part, err)
		}
		if !added {
			slog.Info("goal from --goals already present, not queued again", "goal", part)
		}
	}
	return nil
}

func splitSemi(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ";") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// logSummary writes the totals of the run and the death report to the log.
func logSummary(log *slog.Logger, m *goals.Manager, r *bot.Runner) {
	s := m.StatsSummary()
	log.Info("summary", "kills", s.Totals.Kills, "bossKills", s.Totals.BossKills, "drops", s.Totals.Drops, "deaths", s.Totals.Deaths,
		"potions", s.Totals.Potions, "levelUps", s.Totals.LevelUps, "errors", s.Totals.Errors, "xp", s.Totals.XP,
		"activeSeconds", s.Totals.ActiveSeconds, "xpPerHour", s.XPPerHour)
	rep := r.DeathReport()
	log.Info("death report", "total", rep.Total, "lastHour", rep.LastHour, "goldLost", rep.TotalGold, "itemsLost", rep.TotalItems, "byMonster", rep.ByMonster, "byArea", rep.ByArea)
	for _, d := range rep.Records {
		log.Info("death", "area", d.AreaName, "killer", d.Killer, "level", d.Level, "corpseValue", d.CorpseValue, "retrieved", d.Retrieved, "why", d.Why)
	}
	st := r.Status()
	log.Info("final status", "state", st.State, "activity", st.Activity, "level", st.Character.Level, "area", st.Character.Area)
}

// instancePaths returns the per-instance file locations (paths.Layout) and moves state files written by older
// builds (<data>/state/instances/<id>/) to their new place when the new one does not exist yet.
func instancePaths(o options, log *slog.Logger) (paths.Instance, error) {
	l := o.layout
	if l.Root == "" { // options built without parseFlags (tests)
		l = paths.Layout{Root: o.data}
	}
	ip, err := l.Instance(o.instance)
	if err != nil {
		return ip, err
	}
	old := filepath.Join(o.data, "state", "instances", o.instance)
	for _, f := range []struct{ from, to string }{
		{filepath.Join(old, "goals.json"), ip.Goals()}, {filepath.Join(old, "goals.json.bak"), ip.Goals() + ".bak"},
		{filepath.Join(old, "deaths.json"), ip.Deaths()},
	} {
		if err := moveIfAbsent(f.from, f.to); err != nil {
			log.Warn("legacy state not moved", "from", f.from, "to", f.to, "err", err)
		}
	}
	return ip, nil
}

// moveIfAbsent renames from to to when from exists and to does not.
func moveIfAbsent(from, to string) error {
	if _, err := os.Stat(from); err != nil {
		return nil
	}
	if _, err := os.Stat(to); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	return os.Rename(from, to)
}

// taskChain hands the difficulty rules of internal/task to the goal manager (which cannot import it).
// snap gives the newest snapshot (the runner's LastSnapshot, may be nil); its resists feed the resist gate.
func taskChain(snap func() *d2.Snapshot) *goals.Chain {
	ready := func(next string, clvl int) (bool, string) {
		var raw *d2.Resists
		if s := snap(); s != nil && s.Player != nil {
			r := leveler.RawResists(s.Player)
			raw = &r
		}
		return task.ReadyForNext(next, clvl, raw)
	}
	return &goals.Chain{Next: task.NextDifficulty, Ready: ready, Floor: task.EntryLevel}
}

// hasGoal reports whether the manager already holds a goal of the same type and parameters, in any state.
func hasGoal(m *goals.Manager, g goals.Goal) bool {
	if g.Normalize() != nil {
		return false // Add reports the error
	}
	for _, e := range m.Goals() {
		if e.Type == g.Type && reflect.DeepEqual(e.Params, g.Params) {
			return true
		}
	}
	return false
}
