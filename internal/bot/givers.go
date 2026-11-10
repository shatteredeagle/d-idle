package bot

import (
	"os"
	"strings"
	"sync"

	"d2lod-bot/internal/task"
)

// withQuestGivers opts a campaign into town visits to quest givers (docs/quest-givers.md, internal/task/givers.go). On by
// default for the player's game; off in a named run (D2BOT_RUN: gates and soaks keep today's plan) and with
// D2BOT_QUEST_GIVERS=0. The live part is U.
func (r *Runner) withQuestGivers(c *task.Campaign) *task.Campaign {
	if os.Getenv("D2BOT_RUN") != "" || os.Getenv("D2BOT_QUEST_GIVERS") == "0" {
		return c
	}
	gt, err := task.ShippedGivers()
	if err != nil {
		r.e.Log.Warn("quest givers: table error, visits stay off", "err", err)
		return c
	}
	var once sync.Once
	return c.WithQuestGivers(gt, func() string {
		if s := r.latest(); s != nil && s.Player != nil {
			return strings.ToLower(s.Player.Class.String())
		}
		return ""
	}, func(err error) {
		once.Do(func() { r.e.Log.Warn("quest givers: no visit", "err", err) })
	})
}
