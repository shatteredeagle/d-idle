package main

import (
	"strings"
	"testing"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/env"
	"d2lod-bot/internal/resource"
)

func TestResourceKnobsRegistered(t *testing.T) {
	tune := env.DefaultTuning()
	n := 0
	for _, k := range env.Knobs() {
		if !strings.HasPrefix(k.Key, "resource.") {
			continue
		}
		n++
		v, err := bot.GetKnob(&tune, k)
		if err != nil {
			t.Errorf("%s does not resolve: %v", k.Key, err)
			continue
		}
		if v != k.Default {
			t.Errorf("%s: field default %v, knob default %v", k.Key, v, k.Default)
		}
	}
	if n != 54 {
		t.Fatalf("%d resource knobs registered, want 54", n)
	}
}

// A dashboard edit reaches the policy: the tuner writes env.Tuning.Resource and FromEnv reads it.
func TestResourceKnobEditReachesPolicy(t *testing.T) {
	tune := env.DefaultTuning()
	tuner, err := bot.NewTuner(&tune, bot.Config{}, "")
	if err != nil {
		t.Fatal(err)
	}
	snap := tuner.Snapshot()
	if _, ok := snap.Values["resource.restMaxSec"]; !ok {
		t.Fatal("restMaxSec missing from the Tuning view")
	}
	over := map[string]float64{"resource.restMaxSec": 20}
	if err := bot.ApplyOverrides(&tune, over); err != nil {
		t.Fatal(err)
	}
	if got := resource.FromEnv(tune).RestMaxSec; got != 20 {
		t.Fatalf("RestMaxSec %d", got)
	}
	if got, want := resource.FromEnv(env.DefaultTuning()).DrinkLifePct, env.DefaultTuning().Combat.DrinkLifePct; got != want {
		t.Fatalf("potion thresholds are not single-sourced: %d vs %d", got, want)
	}
}

// The clvl 8 Blood Moor sorceress replay, run with the tuning the bot really builds (profile + registered knobs).
func TestBloodMoorReplayWithWiredTuning(t *testing.T) {
	tune, warns, err := (bot.Config{}).BuildTuning()
	if err != nil || len(warns) > 0 {
		t.Fatalf("%v %v", err, warns)
	}
	rt := resource.FromEnv(tune)
	sc := resource.BloodMoorSorc()
	res := resource.Replay(sc, resource.ResourcePolicy(), rt)
	leg := resource.Replay(sc, resource.LegacyPolicy(), rt)
	t.Logf("legacy %+v\nresource %+v", leg, res)
	if res.StarvedSec > leg.StarvedSec/4 || res.Deaths > 0 || res.KillsPerMin < leg.KillsPerMin*0.9 {
		t.Fatalf("wired policy regressed: %+v vs %+v", res, leg)
	}
}
