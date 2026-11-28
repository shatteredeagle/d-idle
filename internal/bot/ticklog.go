package bot

import (
	"time"

	"d2lod-bot/internal/arbiter"
	"d2lod-bot/internal/d2"
)

// tickLogGap is the least bot-clock time between two tick lines. It was the knob bot.tickLogSec (value 1 on the
// sandboxes); always on since card grad1 (docs/probes/graduations-2026-11-28.md).
const tickLogGap = time.Second

// tickLog writes the recorded-session line (docs/design/ticklog-t4.md): the facts the arbiter saw this tick and the
// chain's hook family, at most one line per tickLogGap. internal/arbiter.Replay reads it (msg "tick"); once a log
// carries tick lines the replay ignores its minute lines. Log only, never acts.
func (r *Runner) tickLog(snap *d2.Snapshot, v arbiter.View, acted bool) {
	defer func() { _ = recover() }()
	now := r.e.Clock.Now()
	a := &r.arb
	a.mu.Lock()
	due := a.tickAt.IsZero() || now.Sub(a.tickAt) >= tickLogGap
	if due {
		a.tickAt = now
	}
	a.mu.Unlock()
	if !due {
		return
	}
	chain := ""
	if acted {
		chain = chainFamily(r.currentWhy())
	}
	p := snap.Player
	r.obMu.Lock()
	kills := r.lifetimeKills(p.Name) // the minute line's counter, so progress means the same in both lines
	r.obMu.Unlock()
	r.e.Log.Info("tick", "area", v.Area, "kills", kills, "xp", p.Exp, "chain", chain,
		"objectiveArea", v.ObjectiveArea, "facts", v.Facts, "vtMs", now.UnixMilli())
}
