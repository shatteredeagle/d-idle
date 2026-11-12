package main

import (
	"path/filepath"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/readiness"
	"d2lod-bot/internal/task"
)

// Guided journey timing (docs/campaign.md section 4; U: judgment, not yet knobs). journeyAskWait is the veto window of
// the floor question (no answer = yes), journeyFarmBudget the level-step time after which the hero offers to go at the floor.
const (
	journeyAskWait    = 10 * time.Minute
	journeyFarmBudget = 30 * time.Minute
)

// newJourney builds the guided-journey controller with its per-hero state in stateDir (docs/campaign.md); "" keeps it
// in memory. The hero key is the character name of the newest snapshot.
func newJourney(stateDir string, ref *runnerRef) (*campaign.Controller, error) {
	ct, err := readiness.ShippedChecks()
	if err != nil {
		return nil, err
	}
	checks := map[string]bool{}
	for _, c := range ct.Checks {
		checks[c.ID] = true
	}
	sp, err := campaign.Shipped(campaign.Validator{Quest: func(q string) bool { return task.Handler(q) != nil }, Check: func(id string) bool { return checks[id] }})
	if err != nil {
		return nil, err
	}
	path := ""
	if stateDir != "" {
		path = filepath.Join(stateDir, campaign.StateFile)
	}
	jc, err := campaign.NewController(sp, path, func() string {
		if s := ref.snap(); s != nil && s.Player != nil && s.Player.Name != "" {
			return s.Player.Name
		}
		return "-"
	}, time.Now)
	if err != nil {
		return nil, err
	}
	jc.AskWait, jc.FarmBudget = journeyAskWait, journeyFarmBudget
	return jc, nil
}

// wireJourney gives the runner the guided journey; a broken spine or state file is logged and the floor planner stays.
func wireJourney(stateDir string, ref *runnerRef, r *bot.Runner, logf func(msg string, kv ...any)) {
	if ref == nil || r == nil {
		return
	}
	jc, err := newJourney(stateDir, ref)
	if err != nil {
		logf("journey off", "err", err)
		return
	}
	jc.OnError = func(err error) { logf("journey state not saved", "err", err) }
	jc.OnDone = func(hero string, ids []string) { logf("journey: chapter done", "hero", hero, "chapters", ids) }
	ref.jc = jc
	r.SetJourney(jc)
}
