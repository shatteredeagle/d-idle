package goals

import (
	"fmt"
	"time"
)

// Chain wires the difficulty chaining of campaign goals. It is injected because internal/task imports this package
// (task.NextDifficulty, task.ReadyForNext and task.EntryLevel are the real implementations, wired in cmd/d2bot).
type Chain struct {
	Next  func(cur string, baalDone bool) string // "" when there is nothing after cur
	Ready func(next string, clvl int) (bool, string)
	Floor func(next string) int // clvl at which Ready turns true
}

func (c *Chain) usable() bool { return c != nil && c.Next != nil && c.Ready != nil && c.Floor != nil }

// chainTemplate is the full-difficulty campaign that follows a finished one, as a difficulty and template.
func chainDifficulty(template string) (cur string, ok bool) {
	switch template {
	case CampaignNormalAll:
		return "normal", true
	case CampaignNightmareAll:
		return "nightmare", true
	}
	return "", false // hell_all is the end; the single-act templates do not finish a difficulty
}

func templateOf(difficulty string) string {
	switch difficulty {
	case "nightmare":
		return CampaignNightmareAll
	case "hell":
		return CampaignHellAll
	}
	return ""
}

// chainLocked enqueues the follow-up of a completed campaign goal: the next difficulty's campaign, preceded by a
// reach_level goal while the character is below that difficulty's entry floor. It is idempotent: the ids it creates
// are stored in g.NextIDs, and a goal that has any is never chained again (not even after the user deleted them).
func (m *Manager) chainLocked(g *Goal, now time.Time) {
	if g.Type != Campaign || g.State != StateDone || !g.Params.Chain || len(g.NextIDs) > 0 || !m.opt.Chain.usable() {
		return
	}
	cur, ok := chainDifficulty(g.Params.Template)
	if !ok {
		return
	}
	next := m.opt.Chain.Next(cur, true)
	tmpl := templateOf(next)
	if tmpl == "" {
		return
	}
	var ids []string
	note := ""
	lvl := m.opt.Runner.Status().Character.Level
	if ok, why := m.opt.Chain.Ready(next, lvl); !ok {
		floor := max(m.opt.Chain.Floor(next), lvl+1) // refused with the level met (resist gate): farm one more level
		lg, err := m.addLocked(Goal{Type: ReachLevel, Priority: g.Priority,
			Params: Params{Level: floor, Difficulty: cur}}, now)
		if err != nil {
			m.chainFail(g, err)
			return
		}
		ids = append(ids, lg.ID)
		note = fmt.Sprintf("%s: clvl %d, %s; queued %s", next, lvl, why, lg.ID)
	}
	cg, err := m.addLocked(Goal{Type: Campaign, Priority: g.Priority,
		Params: Params{Template: tmpl, Cain: g.Params.Cain, Chain: true}}, now)
	if err != nil {
		m.chainFail(g, err)
		return
	}
	g.NextIDs = append(ids, cg.ID)
	if note == "" {
		note = "chained: " + cg.ID + " " + tmpl + " queued"
	} else {
		note += ", then " + cg.ID + " " + tmpl
	}
	g.Note = note
	m.emitLocked(Event{Type: EvInfo, GoalID: g.ID, Message: "Next difficulty: " + note})
}

func (m *Manager) chainFail(g *Goal, err error) {
	g.Note = "chain failed: " + err.Error()
	m.opt.Logf("chain after %s: %v", g.ID, err)
}

// resumeChains runs after a restart: a campaign that finished but crashed before its follow-up was saved gets it now.
func (m *Manager) resumeChains(now time.Time) {
	for _, g := range append([]*Goal(nil), m.goals...) {
		m.chainLocked(g, now)
	}
}
