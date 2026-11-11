package main

import (
	"os"

	"d2lod-bot/internal/bot"
)

// gearDecisionsKey is the one knob behind the player-only gear policy (docs/gear-policy.md): 0 = the player decides
// equipment (the default), 1 = the bot does (legacy behaviour).
const gearDecisionsKey = "resource.gearDecisions"

// withGearPolicy lays gear.decisions = bot under cfg when the process is a measured or drill run (D2BOT_RUN is set) or
// D2BOT_GEAR=bot says so, so gates, soaks and drills keep the old equip, swap and buy paths. An explicit tuning key of
// the config wins. A copy: cfg.Tuning is not changed.
func withGearPolicy(cfg bot.Config) bot.Config {
	if _, set := cfg.Tuning[gearDecisionsKey]; set {
		return cfg
	}
	if os.Getenv("D2BOT_RUN") == "" && os.Getenv("D2BOT_GEAR") != "bot" {
		return cfg
	}
	m := make(map[string]float64, len(cfg.Tuning)+1)
	for k, v := range cfg.Tuning {
		m[k] = v
	}
	m[gearDecisionsKey] = 1
	cfg.Tuning = m
	return cfg
}
