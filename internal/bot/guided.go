package bot

import (
	"os"
	"strings"
	"sync"
	"time"

	journey "d2lod-bot/internal/campaign"
	"d2lod-bot/internal/decide"
	"d2lod-bot/internal/ladder"
	"d2lod-bot/internal/task"
)

// PointJourneyFloor is the decision point of the floor question (internal/decide catalogue): the return-screen queue
// shows it next to the Journey panel's own Go now / Not yet.
const PointJourneyFloor = "t1.journey.floor"

// journeyAskGap paces the ladder question per chapter (the ladder's repeat policy also applies).
const journeyAskGap = time.Minute

// SetJourney gives the runner the guided journey (docs/campaign.md): Normal Act 1 campaigns follow the spine while the
// hero's mode is Guided. The campaign.journey* knobs set the veto window, the farm budget and the default policy. Call
// before the runner starts; nil (the default) keeps the floor planner alone.
func (r *Runner) SetJourney(c *journey.Controller) {
	r.journey = c
	if c == nil || r.e.Tune == nil {
		return
	}
	t := r.e.Tune.Campaign
	c.AskWait = time.Duration(t.JourneyAskWaitMin) * time.Minute
	c.FarmBudget = time.Duration(t.JourneyFarmBudgetMin) * time.Minute
	if t.JourneyGoAtFloor != 0 {
		c.DefaultGoAt = journey.GoFloor
	}
}

// journeyGuide is the guide for a campaign goal, nil without a journey or in a named run (D2BOT_RUN: gates, soaks and
// drills keep the floor planner so measured runs stay comparable).
func (r *Runner) journeyGuide() *task.Guide {
	if r.journey == nil || os.Getenv("D2BOT_RUN") != "" {
		return nil
	}
	g := task.NewGuide(r.journey, func() string {
		if s := r.latest(); s != nil && s.Player != nil {
			return strings.ToLower(s.Player.Class.String())
		}
		return ""
	})
	base := g.OnDecision
	var mu sync.Mutex
	asked := map[string]time.Time{}
	g.OnDecision = func(d journey.Decision, p journey.Progress, o journey.Opts) {
		base(d, p, o)
		if d.Action != journey.ActAsk {
			return
		}
		mu.Lock()
		due := o.Now.Sub(asked[d.Chapter]) >= journeyAskGap
		if due {
			asked[d.Chapter] = o.Now
		}
		mu.Unlock()
		if due {
			r.askJourneyFloor(d.Chapter)
		}
	}
	return g
}

// askJourneyFloor puts the floor question in the ladder's queue (the return screen). The legacy answer is "level", so
// the ladder never sends the hero by itself: only a player's go (or a rule taught from one) is taken; no answer is
// handled by the journey's own veto window (campaign.journeyAskWaitMin).
func (r *Runner) askJourneyFloor(chapter string) {
	l := ladder.Active()
	s := r.latest()
	if l == nil || s == nil || s.Player == nil {
		return
	}
	o := decide.Obs{Cat: map[string]string{"chapter": chapter}, Sit: decide.Situation{"chapter": chapter}}
	a, err := l.Decide(PointJourneyFloor, ladder.HeroFrom(s.Player), o, "level")
	if err != nil {
		r.e.Log.Warn("journey: floor question not asked", "chapter", chapter, "err", err)
		return
	}
	if a == "go" {
		yes := true
		if err := r.journey.Apply(journey.Request{Chapter: chapter, Answer: &yes}); err != nil {
			r.e.Log.Warn("journey: answer not saved", "chapter", chapter, "err", err)
		}
	}
}
