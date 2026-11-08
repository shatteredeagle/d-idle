package main

import (
	"testing"

	"d2lod-bot/internal/bot"
)

// TASK-183: the player preset reaches the tuning only with the ladder on, and a key the config sets wins.
func TestPlayerTuningPreset(t *testing.T) {
	for _, class := range []string{"optimal", "relaxed"} {
		cfg := bot.Config{Profile: class}
		off, _, err := withPlayerTuning(cfg, ladderOptions{}).BuildTuning()
		if err != nil {
			t.Fatal(err)
		}
		if off.Resource.RecoveryEndOnKit != 0 {
			t.Fatalf("%s: gate/soak defaults changed without the ladder: %+v", class, off.Resource)
		}
		cfg.Tuning = map[string]float64{"resource.recoveryStallMin": 15}
		on, _, err := withPlayerTuning(cfg, ladderOptions{on: true}).BuildTuning()
		if err != nil {
			t.Fatal(err)
		}
		r := on.Resource
		if r.RecoveryEndOnKit != 1 || r.RecoveryStallGold != 50 || r.PovertySellMin != 8 || r.RecoveryStallMin != 15 {
			t.Fatalf("%s: player preset wrong (config key must win): %+v", class, r)
		}
		if cfg.Tuning["resource.recoveryEndOnKit"] != 0 {
			t.Fatalf("%s: the caller's config map was changed", class)
		}
	}
}
