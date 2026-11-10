package main

import (
	"log/slog"
	"net/http"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/data"
	"d2lod-bot/internal/readiness"
)

// wireReadiness builds the read-only readiness model (docs/readiness.md): the area table derived from the game tables,
// the shipped check rows, the runner's hero snapshot. It also hands the evaluator to the runner for the risk meter's
// resist gap. nil without game tables or on a table error (logged): the endpoint is then simply absent.
func wireReadiness(runner *bot.Runner, db *data.DB) http.Handler {
	if runner == nil || db == nil {
		return nil
	}
	at, err := readiness.BuildAreaTable(db)
	if err != nil {
		slog.Warn("readiness: no area table", "err", err)
		return nil
	}
	ct, err := readiness.ShippedChecks()
	if err != nil {
		slog.Warn("readiness: bad check table", "err", err)
		return nil
	}
	ev, err := readiness.New(ct, at)
	if err != nil {
		slog.Warn("readiness: " + err.Error())
		return nil
	}
	runner.SetReadiness(ev)
	return readiness.Handler(ev, runner.ReadinessHero)
}
