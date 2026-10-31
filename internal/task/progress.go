package task

import (
	"fmt"
	"time"

	"d2lod-bot/internal/goals"
)

// Report builds the goals.Progress fields the task layer owns (tracking.md section 3): Steps, Phase, ETASeconds,
// Source and Verified, plus Current/Target/Percent as quests done of quests total. The manager merges the counters
// it keeps itself (kills, deaths, activeSeconds) and overwrites these.
func (c *Campaign) Report(w World, ex *Executor) goals.Progress {
	if ex == nil {
		ex = NewExecutor(nil, nil)
	}
	var p goals.Progress
	plan := c.Plan(w, ex)
	p.Target = len(c.Entries)
	if c.wrongDifficulty(w) {
		// The world's quests belong to another difficulty: counting them would finish this campaign unplayed.
		p.Phase, p.Source = "difficulty", "events"
		for _, e := range c.Entries {
			p.Steps = append(p.Steps, goals.Step{ID: "quest." + e.Quest, Title: e.Title, State: goals.StateQueued})
		}
		return p
	}
	var eta time.Duration
	allMem, anyMem := true, false
	note := func(ev Evidence) {
		if ev >= EvMemory {
			anyMem = true
		} else {
			allMem = false
		}
	}
	activeSeen := false
	for _, e := range c.Entries {
		st := goals.Step{ID: "quest." + e.Quest, Title: e.Title, State: goals.StateQueued}
		qs := w.Quest(e.Quest)
		switch {
		case QuestDone(w, e.Quest):
			st.State = goals.StateDone
			p.Current++
			note(qs.Ev)
		case ex != nil && ex.Finished(e.Quest):
			st.State = goals.StateDone
			p.Current++
			allMem = false // done by assumption, not proof
		case ex != nil && ex.isBlocked(e.Quest):
			st.State = goals.StateFailed
		case plan.Group == e.Quest:
			st.State = goals.StateActive
			activeSeen = true
			note(qs.Ev)
			for _, n := range plan.Nodes {
				if need, ev := n.Step.Need(w); need && !ex.skippedOrAssumed(n.Step.ID) {
					eta += n.Step.Est
					note(ev)
				}
			}
			if s, _, ok := ex.Peek(plan, w); ok {
				p.Phase = s.Phase
			}
		default:
			for _, n := range Handler(e.Quest) {
				if need, _ := n.Step.Need(w); need {
					eta += n.Step.Est
				}
			}
		}
		p.Steps = append(p.Steps, st)
	}
	if !activeSeen && len(plan.Nodes) > 0 { // a level plan: show it as the phase and as one extra step
		p.Phase = "level"
		s := plan.Nodes[0].Step
		p.Steps = append(p.Steps, goals.Step{ID: s.ID, Title: s.Title, State: goals.StateActive, Current: w.Level, Target: lvlTarget(plan)})
		eta += s.Est
	}
	if p.Target > 0 {
		p.Percent = float64(p.Current) * 100 / float64(p.Target)
	}
	p.ETASeconds = int64(eta / time.Second)
	switch {
	case anyMem && allMem:
		p.Source, p.Verified = "memory", true
	case anyMem:
		p.Source = "mixed"
	default:
		p.Source = "events"
	}
	return p
}

func lvlTarget(p Plan) int {
	var n int
	fmt.Sscanf(p.ID, "level.%d", &n)
	return n
}

func (e *Executor) isBlocked(g string) bool { _, b := e.Blocked(g); return b }

func (e *Executor) skippedOrAssumed(id string) bool {
	return e != nil && (e.skipped[id] || e.assumed[id])
}
