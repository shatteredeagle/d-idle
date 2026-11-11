package main

import (
	"path/filepath"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/campaign"
	"d2lod-bot/internal/readiness"
	"d2lod-bot/internal/task"
	"d2lod-bot/internal/telemetry"
	"d2lod-bot/internal/watch"
)

// journeyIdentifyCain is the feature a done chapter opens for the "Identify all (Cain)" command.
const journeyIdentifyCain = "identify_cain"

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
	// AskWait, FarmBudget and DefaultGoAt come from the campaign.journey* knobs (bot.Runner.SetJourney)
	jc.Pace = journeyPace(func() (int, uint32, bool) {
		if s := ref.snap(); s != nil && s.Player != nil {
			return s.Player.Level, s.Player.Exp, true
		}
		return 0, 0, false
	}, time.Now)
	return jc, nil
}

// journeyPace is the level meter of a farm step: the share of the experience from the current level to the target, and
// the time left at the rate seen since the target was set (after two minutes of samples; U: a short window is noisy).
func journeyPace(hero func() (lvl int, exp uint32, ok bool), now func() time.Time) func(int) (float64, time.Duration, bool) {
	var target int
	var t0 time.Time
	var exp0 uint32
	return func(to int) (float64, time.Duration, bool) {
		lvl, exp, ok := hero()
		lo, ok1 := telemetry.DefaultExpForLevel(lvl)
		hi, ok2 := telemetry.DefaultExpForLevel(to)
		if !ok || !ok1 || !ok2 || hi <= lo || exp < lo {
			return 0, 0, false
		}
		if to != target || exp < exp0 {
			target, t0, exp0 = to, now(), exp
		}
		pct := min(1, float64(exp-lo)/float64(hi-lo))
		var eta time.Duration
		if dt := now().Sub(t0); dt >= 2*time.Minute && exp > exp0 && hi > exp {
			eta = time.Duration(float64(hi-exp) / (float64(exp-exp0) / dt.Seconds()) * float64(time.Second))
		}
		return pct, eta, true
	}
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

// bindJourney mounts GET /api/campaign and POST /api/campaign/intent on the watch screen (docs/campaign.md section 6).
// A chapter done also writes the Chronicle and feed line and tells the unlock ladder (watch.Fun.ChapterDone).
func bindJourney(f *watch.Fun, jc *campaign.Controller) {
	if f == nil || jc == nil {
		return
	}
	api := watch.NewCampaignAPI()
	api.Bind(jc)
	f.Campaign = api
	sp := jc.Spine()
	jc.AddOnDone(func(_ string, ids []string) {
		for _, id := range ids {
			if ch := sp.Chapter(id); ch != nil {
				f.ChapterDone(ch.ID, ch.Title, ch.Reward)
			}
		}
	})
}
