package bot

import (
	"os"
	"strings"

	journey "d2lod-bot/internal/campaign"
	"d2lod-bot/internal/task"
)

// SetJourney gives the runner the guided journey (docs/campaign.md): Normal Act 1 campaigns follow the spine while the
// hero's mode is Guided. Call before the runner starts; nil (the default) keeps the floor planner alone.
func (r *Runner) SetJourney(c *journey.Controller) { r.journey = c }

// journeyGuide is the guide for a campaign goal, nil without a journey or in a named run (D2BOT_RUN: gates, soaks and
// drills keep the floor planner so measured runs stay comparable).
func (r *Runner) journeyGuide() *task.Guide {
	if r.journey == nil || os.Getenv("D2BOT_RUN") != "" {
		return nil
	}
	return task.NewGuide(r.journey, func() string {
		if s := r.latest(); s != nil && s.Player != nil {
			return strings.ToLower(s.Player.Class.String())
		}
		return ""
	})
}
