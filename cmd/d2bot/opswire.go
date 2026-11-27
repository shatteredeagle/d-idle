package main

import (
	"os"
	"sync/atomic"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/ops"
	"d2lod-bot/internal/questboard"
	"d2lod-bot/internal/watch"
)

// startOperator opens the operator socket next to the quest plan files when D2BOT_OPERATOR=1 (never in a measured
// run, never by default). It reuses the desk the viewer routes use; failures only log. docs/operator.md.
func startOperator(stateDir string, d *questboard.Desk, r *bot.Runner, logf func(msg string, kv ...any)) {
	if d == nil || stateDir == "" {
		return
	}
	opsRunner.Store(r)
	_, err := ops.Start(stateDir, func() ops.Desk { return d }, os.Getenv, logf, func(c *ops.Config) {
		c.Orders = func() ops.Orders {
			if h := opsHub.Load(); h != nil {
				return hubOrders{h}
			}
			return nil
		}
		c.Explain = explainNow
		c.Difficulty = func() ops.DifficultyGo {
			if r := opsRunner.Load(); r != nil {
				return r
			}
			return nil
		}
	})
	if err != nil {
		logf("operator control failed to start", "err", err)
	}
}

// The hub is made before the runner and the runner before the hub's routes, so the operator socket and /api/explain
// find each other through these two late-bound pointers (set once by the wiring, read lock-free).
var (
	opsHub    atomic.Pointer[watch.Hub]
	opsRunner atomic.Pointer[bot.Runner]
)

// explainNow is the decision ring of the runner, or nil before it exists (both the socket and /api/explain read it).
func explainNow() any {
	if r := opsRunner.Load(); r != nil {
		v := r.Explain()
		if f := opsKnobs.Load(); f != nil {
			v.Knobs = (*f)()
		}
		return v
	}
	return nil
}

// opsKnobs returns the live knob values; the backend wiring sets it from the tuner (nil: the view has no knobs).
var opsKnobs atomic.Pointer[func() map[string]float64]

func setOpsKnobs(t *bot.Tuner) {
	f := func() map[string]float64 { return t.Snapshot().Values }
	opsKnobs.Store(&f)
}

// hubOrders places the operator's orders through the same Hub.Give the viewer's order box uses.
type hubOrders struct{ h *watch.Hub }

func (o hubOrders) Give(kind, area string, minutes int, replace bool) (string, error) {
	ord, err := o.h.Give(watch.OrderRequest{Kind: kind, Area: area, Minutes: minutes, Replace: replace})
	return ord.Text, err
}
