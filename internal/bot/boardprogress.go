package bot

import (
	"strings"
	"time"

	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/questboard"
)

// The campaign goal's progress under the quest board. With quests.board on the task executor never runs a step, so its
// report (setTaskReport, the only source of Runner.Progress) was never refreshed and the goal read 0 of 6 for the whole
// life of a hero whose quests the board had finished (sandbox logs 2026-11-27, problem 7). The board is the authority now:
// the count is the task layer's own report over the world, with every quest the board shows done counted done.

const boardReportEvery = 5 * time.Second

// boardProgress marks the steps of rep whose quest the board shows done, recounts Current and Percent and says that the
// report now rests on the board's evidence. done holds task ids (taskID of the board quest). It never lowers a count.
func boardProgress(rep goals.Progress, done map[string]bool) goals.Progress {
	steps := make([]goals.Step, len(rep.Steps))
	copy(steps, rep.Steps)
	rep.Steps = steps
	for i, s := range rep.Steps {
		id, ok := strings.CutPrefix(s.ID, "quest.")
		if ok && s.State != goals.StateDone && done[id] {
			rep.Steps[i].State = goals.StateDone
			rep.Current++
		}
	}
	if rep.Target > 0 {
		rep.Percent = float64(rep.Current) * 100 / float64(rep.Target)
	}
	return rep
}

// reportBoardProgress refreshes the campaign goal's progress from the world and the board (loop thread, rate limited).
func (r *Runner) reportBoardProgress(c control) {
	if c.spec.camp == nil || !r.questBoardOn() {
		return
	}
	rt := r.qbRT()
	if rt == nil {
		return
	}
	now := r.e.Clock.Now()
	if now.Sub(rt.reportAt) < boardReportEvery {
		return
	}
	rt.reportAt = now
	w, _, _ := rt.worldLoop()
	st, err := rt.exec.State()
	if err != nil {
		return
	}
	done := map[string]bool{}
	for _, v := range st.Views {
		if v.State != questboard.Done {
			continue
		}
		if q, err := rt.board.Quest(v.ID); err == nil {
			done[taskID(q)] = true
		}
	}
	r.setTaskReport(c.gen, boardProgress(c.spec.camp.Report(w, nil), done))
}
