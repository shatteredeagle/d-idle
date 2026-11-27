package bot

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// Relaunch reasons: every Save and Exit that ends the running game (and, on the d2d backend, the process) logs one INFO
// line "relaunch reason=<code> cause=<what>" and counts into the minute line (relaunches, relaunchBy). The codes are the
// whole list; a new trigger adds a code here.
const (
	RelaunchChicken    = "chicken"          // a chicken carried out: combat left the game at the chicken line
	RelaunchLeave      = "leave"            // combat left the game without a chicken decision (fled too long, merc down)
	RelaunchNewGame    = "newgame"          // an unplanned fresh game (farm, difficulty switch, rung 3, drought; cause=from file:line)
	RelaunchSavePoint  = "savepoint"        // a planned save point
	RelaunchRecycle    = "recycle"          // the recycler between goals (docs/soak.md)
	RelaunchFailsafe   = "failsafe-restart" // failsafe ladder rung 4
	RelaunchGameLost   = "gamelost"         // the game process was gone (crash or exit the bot did not plan)
	chickenLeaveWindow = 30 * time.Second   // a Leave this soon after a carried-out chicken is that chicken
)

// relaunchLog counts relaunches by reason since the process started.
type relaunchLog struct {
	mu        sync.Mutex
	by        map[string]int
	total     int
	chickenAt time.Time
	chickenKV string
}

// noteRelaunch logs and counts one relaunch.
func (r *Runner) noteRelaunch(reason, cause string) {
	rl := &r.relaunch
	rl.mu.Lock()
	if rl.by == nil {
		rl.by = map[string]int{}
	}
	rl.by[reason]++
	rl.total++
	rl.mu.Unlock()
	r.e.Log.Info("relaunch", "reason", reason, "cause", cause)
}

// noteChicken remembers a carried-out chicken (life and area) so the Leave that follows is told as a chicken.
func (r *Runner) noteChicken(cause string) {
	rl := &r.relaunch
	rl.mu.Lock()
	rl.chickenAt, rl.chickenKV = r.e.Clock.Now(), cause
	rl.mu.Unlock()
}

// NoteLeave is called by the wiring each time combat's Leave hook runs (Save and Exit of the running game).
func (r *Runner) NoteLeave() {
	rl := &r.relaunch
	rl.mu.Lock()
	chick := !rl.chickenAt.IsZero() && r.e.Clock.Now().Sub(rl.chickenAt) <= chickenLeaveWindow
	cause := rl.chickenKV
	rl.chickenAt = time.Time{}
	rl.mu.Unlock()
	if chick {
		r.noteRelaunch(RelaunchChicken, cause)
		return
	}
	r.noteRelaunch(RelaunchLeave, "combat abort without a chicken decision")
}

// relaunchKV is the minute line part: the total and the count per reason ("chicken=2,newgame=3").
func (r *Runner) relaunchKV() []any {
	rl := &r.relaunch
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.total == 0 {
		return nil
	}
	var parts []string
	for k, n := range rl.by {
		parts = append(parts, fmt.Sprintf("%s=%d", k, n))
	}
	sort.Strings(parts)
	return []any{"relaunches", rl.total, "relaunchBy", strings.Join(parts, ",")}
}
