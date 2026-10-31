package main

import (
	"encoding/json"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

// fakeMinutes writes a synthetic bot log in the real minute-line format, so the extractor, aggregation and comparison
// can be exercised without d2d. The model is a toy (rates of attempt 18 times scale, Poisson deaths); it says nothing
// about d2d and its numbers must never be quoted as a result.
func fakeMinutes(path string, seed int, hours, scale float64) error {
	rng := rand.New(rand.NewPCG(uint64(seed), 0xd2d))
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	at := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	xp, kills, deaths, potions, gold, clvl := 143323.0, 1747, 0, 0, 16, 14
	for i := 0; i <= int(hours*60); i++ {
		state, rstate := "farming", "clearing"
		if rng.Float64() < 0.2 {
			state, rstate = "town chores", "resupplying"
		} else {
			noise := 0.6 + 0.8*rng.Float64()
			xp += 190 * scale * noise
			kills += int(math.Round(12 * scale * noise))
			if rng.Float64() < 0.002*scale { // about 0.1 deaths/h
				deaths++
			}
			if rng.Float64() < 0.5 {
				potions++
			}
		}
		gold += int(rng.Float64() * 20)
		enc.Encode(map[string]any{"time": at.Add(time.Duration(i) * time.Minute).Format(time.RFC3339Nano), "level": "INFO",
			"msg": "minute", "inst": "fake", "area": 4, "clvl": clvl, "xp": uint64(xp), "kills": kills, "deaths": deaths,
			"potions": 4, "potionsUsed": potions, "gold": gold, "restarts": 0, "rung": 0, "goal": "campaign",
			"step": "Quest: fighting (Level up to 15)", "state": state, "rstate": rstate})
	}
	return nil
}

// runFake runs the whole pipeline on synthetic logs: no d2d, no lock, no clock.
func runFake(cfg runCfg, scale float64, seed int) (runResult, error) {
	dir := filepath.Join(cfg.Out, "seed-"+strconv.Itoa(seed))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return runResult{}, err
	}
	p := filepath.Join(dir, "bot.log")
	if err := fakeMinutes(p, seed, cfg.Hours, scale); err != nil {
		return runResult{}, err
	}
	res, err := finish(runResult{Seed: seed, Clock: "fake", End: "fake"}, p, cfg.Hours, cfg.Hours)
	if err != nil {
		return res, err
	}
	return res, writeJSON(filepath.Join(dir, "run.json"), res)
}
