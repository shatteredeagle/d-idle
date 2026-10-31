package main

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"sync"

	"d2lod-bot/internal/api"
	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/goals/sim"
	"d2lod-bot/internal/server"
)

// runSim serves the dashboard against the simulated runner: no game, no input, no data files needed.
func runSim(ctx context.Context, o options, log *slog.Logger) error {
	runner := sim.New(sim.Options{Seed: o.simSeed})
	ip, err := instancePaths(o, log)
	if err != nil {
		return err
	}
	m, err := goals.New(goals.Options{Runner: runner, Path: ip.Goals(), Logf: func(f string, a ...any) { log.Info("goals", "msg", sprintf(f, a...)) }})
	if err != nil {
		return err
	}
	if len(m.Goals()) == 0 {
		for _, g := range []goals.Goal{
			{Type: goals.ReachLevel, Params: goals.Params{Level: 10}},
			{Type: goals.FarmArea, Params: goals.Params{Area: "Cold Plains", UntilKills: 150}},
		} {
			if _, err := m.Add(g, -1); err != nil {
				return err
			}
		}
	}
	m.Start()
	tune := env.DefaultTuning() // not read by the simulator, but the Tuning tab has something to show
	tuner, err := bot.NewTuner(&tune, bot.Config{}, filepath.Join(ip.State(), "tuning.json"))
	if err != nil {
		return err
	}
	inst := &server.Instance{ID: instanceID(o), Title: "simulated", Manager: m,
		Mounts: map[string]http.Handler{"knobs": api.KnobsHandler(tuner), "upgrades": api.UpgradesHandler(tuner)}}
	reg := newRegistry()

	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); m.Run(ctx) }()
	go func() { defer wg.Done(); runner.Run(ctx, o.simSpeed) }()
	go logEvents(ctx, log, m)
	err = serve(ctx, log, o, inst, reg, "d2bot (simulated runner)")
	wg.Wait()
	return err
}
