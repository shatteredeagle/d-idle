package questboard

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// The executor on the run machine (knob quests.boardSM, card bsm4). exec.go calls these only when Options.SM is set;
// with the knob at 0 none of this runs and the old paths are byte for byte the same.

// smOverlay carries what derive found while it laid the stored runs and latches over the evidence: latches to write
// (a chunk the board ran reads done), regressions (exact evidence went back), both applied in one save by smFlush.
type smOverlay struct {
	m        SM
	diff     string
	latch    []smLatch
	regress  []Ref
	regTiers map[Ref]Tier
}

type smLatch struct {
	ref    Ref
	tier   Tier
	source string
}

func (o *smOverlay) dirty() bool { return len(o.latch)+len(o.regress) > 0 }

// smLoad loads the machine of this difficulty with the import, the repair and the head selection done.
func (e *Exec) smLoad(diff int) (*smOverlay, error) {
	e.smReleaseRetry()
	m, err := e.st.LoadSM(e.smDone())
	if err != nil {
		return nil, err
	}
	return &smOverlay{m: m, diff: Difficulties[diff], regTiers: map[Ref]Tier{}}, nil
}

// smDone is the evidence the v1 import reads: a chunk that reads done now.
func (e *Exec) smDone() DoneFn {
	return func(quest, chunk string, diff int) (Tier, bool) {
		q, err := e.b.Quest(quest)
		if err != nil {
			return 0, false
		}
		c, err := q.ChunkByID(chunk)
		if err != nil {
			return 0, false
		}
		cf := e.env.Chunk(q, c)
		return cf.Tier, cf.Done
	}
}

// apply lays the stored state over one chunk's fact. A latched chunk reads done when the evidence is silent (R1: a
// proxy that went away, a restart); exact denial regresses it (R3); a chunk that reads done under a live run gets its
// latch (3.6); a parked run shows its park, so it survives a restart (Q5).
func (o *smOverlay) apply(q Quest, c Chunk, cf ChunkFact, rs *runState) ChunkFact {
	ref := Ref{q.ID, c.ID}
	run := o.m.find(q.ID, c.ID, o.diff)
	switch l := o.m.latch(q.ID, c.ID, o.diff); {
	case l != nil && !cf.Done && cf.Denied:
		o.regress = append(o.regress, ref)
		o.regTiers[ref] = TierMemory
	case l != nil && !cf.Done:
		cf.Done, cf.Tier = true, l.Tier
	case l != nil && cf.Done && cf.Tier > l.Tier:
		o.latch = append(o.latch, smLatch{ref, cf.Tier, "evidence"}) // a latch only rises (R2)
	case l == nil && cf.Done && run != nil:
		o.latch = append(o.latch, smLatch{ref, cf.Tier, "evidence"})
	}
	if run != nil && run.Status == RunParked && cf.Blocked == "" && (rs == nil || rs.blocked == "") {
		cf.Blocked = smParkText(q, *run)
	}
	return cf
}

// smParkText is the sentence of a stored park (the code table builds it; the counters are the stored ones).
func smParkText(q Quest, r Run) string {
	n := r.Failures
	if r.Code == "died_too_often" {
		n = r.Deaths
	}
	if r.Code == "hold_timeout" {
		return holdTimeoutText(r.Hold, r.HoldWhy, r.HoldMin)
	}
	return Sentence(r.Code, map[string]string{"quest": q.Title, "n": fmt.Sprint(n), "why": "the step kept failing"})
}

// holdBudget is the minutes the hold's code may wait (its row of the reason table).
func holdBudget(code string) int {
	info, _ := ReasonOf(code)
	return info.BudgetMin
}

// holdTimeoutText is the sentence of a hold_timeout park: the minutes it waited and the hold's own words with the producer's
// reason filled in (V live 2026-11-24, tools_of_the_trade/malus: "Still waiting after 0 minutes: {hold}", the counters and
// the hold of the run were not carried into the stored park). A park stored before the hold was kept says so without a gap.
func holdTimeoutText(hold, why string, minutes int) string {
	if hold == "" {
		return "Held too long without getting ready. Retry to train again."
	}
	h := Sentence(hold, map[string]string{"why": why})
	if strings.Contains(h, "{") && why != "" {
		h = why // a hold whose sentence needs other arguments: the producer's words say it
	}
	return Sentence("hold_timeout", map[string]string{"n": fmt.Sprint(minutes), "hold": h})
}

// smFlush writes the latches and regressions derive found in one save.
func (e *Exec) smFlush(o *smOverlay, diff int) error {
	d := Difficulties[diff]
	return e.st.UpdateSM(e.smDone(), func(m *SM) error {
		for _, l := range o.latch {
			m.NoteDone(e.b, d, l.ref.Quest, l.ref.Chunk, l.tier, e.o.Now(), l.source)
		}
		for _, r := range o.regress {
			if parked, cleared := m.Regress(e.b, d, r.Quest, r.Chunk, o.regTiers[r]); cleared {
				e.o.Log("quest board: regressed "+r.String(), "quest", r.Quest, "chunk", r.Chunk, "parked", parked != nil)
			}
		}
		return nil
	})
}

// smAccept is Accept on the machine: one rule for every state (D1 the chain, D5 one quest, I7 idempotent).
func (e *Exec) smAccept(vs []QuestView, diff int, quest, chunk string, ev *AutoEvent) error {
	d := Difficulties[diff]
	var changed []Ref
	var already bool
	err := e.st.UpdateSM(e.smDone(), func(m *SM) error {
		// Earlier parts that already read done (by hand) have no run and need none: latch them quietly.
		if anc, aerr := ancestors(e.b, quest, chunk); aerr == nil {
			for _, a := range anc {
				if _, cv, ok := chunkView(vs, quest, a); ok && cv.State == Done && m.find(quest, a, d) == nil && !m.Latched(quest, a, d) {
					m.NoteDone(e.b, d, quest, a, cv.Tier, e.o.Now(), "evidence")
				}
			}
		}
		startable := func(r Ref) error {
			_, cv, ok := chunkView(vs, r.Quest, r.Chunk)
			switch {
			case !ok:
				return fmt.Errorf("unknown chunk")
			case r.Chunk == chunk && ev == nil && (afterOnly(cv) || cv.State == Awaiting):
				return nil // Awaiting without a run: the game counts the field work done (C5), the run starts at the hand-in
			case r.Chunk != chunk && (cv.State == Ready || cv.State == Locked && strings.HasPrefix(cv.Why, "after ")):
				return nil
			}
			return e.canStart(vs, r.Quest, r.Chunk)
		}
		var serr error
		changed, already, serr = m.Start(e.b, d, quest, chunk, startBy(ev), e.o.Now(), startable)
		for _, r := range changed {
			if run := m.find(r.Quest, r.Chunk, d); run != nil {
				run.Mode = e.o.Mode
			}
		}
		return serr
	})
	var busy ErrBusy
	switch {
	case errors.As(err, &busy):
		run := busy.Quest
		if m, lerr := e.st.LoadSM(nil); lerr == nil {
			if live := m.Live(e.b, d); len(live) > 0 {
				run = live[0].Quest + "/" + live[0].Chunk
			}
		}
		return fmt.Errorf("questboard: %s is already accepted; finish or stop it first (one chunk at a time)", run)
	case errors.Is(err, ErrDone):
		return fmt.Errorf("questboard: %s is already done", ChunkKey(quest, chunk))
	case err != nil:
		var ns ErrNotStartable
		if errors.As(err, &ns) && ns.Part.Chunk != chunk {
			return fmt.Errorf("questboard: start %q first: %v", ns.Part.Chunk, ns.Err)
		}
		if errors.As(err, &ns) {
			return ns.Err
		}
		return err
	}
	if already {
		return nil
	}
	for _, r := range changed {
		delete(e.rs, ChunkKey(r.Quest, r.Chunk)) // a fresh start: no old counters or park
	}
	if ev != nil {
		if aerr := e.st.AddAutoEvent(*ev); aerr != nil {
			e.o.Log("quest board: auto event not saved", "err", aerr)
		}
	}
	e.lastChanged = changed
	last := changed[len(changed)-1]
	if len(changed) > 1 {
		var pre []string
		for _, r := range changed[:len(changed)-1] {
			pre = append(pre, r.Chunk)
		}
		e.o.Log("quest board: accepted "+last.String()+" (after: "+strings.Join(pre, ", ")+")", "quest", quest, "chunk", last.Chunk)
	} else {
		e.o.Log("quest board: accepted "+last.String(), "quest", quest, "chunk", last.Chunk)
	}
	return nil
}

func startBy(ev *AutoEvent) string {
	if ev != nil {
		return "auto"
	}
	return "player"
}

// smCancel is Cancel on the machine: the run and every run that waits for it (D2). Callers hold e.mu.
func (e *Exec) smCancel(q Quest, c Chunk) error {
	d := Difficulties[e.env.Diff()]
	var changed []Ref
	if err := e.st.UpdateSM(e.smDone(), func(m *SM) error {
		changed, _ = m.Stop(e.b, d, q.ID, c.ID)
		return nil
	}); err != nil {
		return err
	}
	for _, r := range changed {
		delete(e.rs, ChunkKey(r.Quest, r.Chunk))
	}
	e.lastChanged = changed
	e.act.Stop(q, c)
	return nil
}

// smPark stores a park the executor just decided (the counters stay with it, Q5). No-op without the machine.
func (e *Exec) smPark(quest, chunk, code string, rs *runState) {
	if e.boardLegacy() {
		return
	}
	d := Difficulties[e.env.Diff()]
	err := e.st.UpdateSM(e.smDone(), func(m *SM) error {
		if _, err := m.Apply(e.b, d, quest, chunk, EvCap, code); err != nil {
			return err
		}
		if r := m.find(quest, chunk, d); r != nil {
			r.Failures, r.Deaths = rs.failures, rs.deaths
			r.Release = e.o.Release
			if code == "hold_timeout" {
				r.Hold, r.HoldWhy, r.HoldMin = rs.hold, rs.holdWhy, holdBudget(rs.hold)
			}
		}
		return nil
	})
	if err != nil {
		e.o.Log("quest board: park not saved", "chunk", ChunkKey(quest, chunk), "err", err)
	}
}

// smReleaseRetry runs once per Exec: a gave_up park (the step kept failing) stored by another bot release is retried
// once, because the failure may be the old release's bug. The park keeps its history (RetriesAfterRelease) and, if the
// step fails again, comes back stored with this release, so it cannot loop. Parks of other codes stay as the player
// decided (Q5).
func (e *Exec) smReleaseRetry() {
	if e.relDone || e.boardLegacy() || e.o.Release == "" {
		return
	}
	e.relDone = true
	var cleared []Ref
	var olds []string
	err := e.st.UpdateSM(e.smDone(), func(m *SM) error {
		for _, d := range Difficulties {
			for _, r := range m.Live(e.b, d) {
				if r.Status != RunParked || r.Code != "gave_up" || r.Release == e.o.Release {
					continue
				}
				old, n := r.Release, r.RetriesAfterRelease+1
				if _, _, err := m.Retry(e.b, d, r.Quest, r.Chunk, false, e.o.Now()); err != nil {
					continue
				}
				if live := m.find(r.Quest, r.Chunk, d); live != nil {
					live.Release, live.RetriesAfterRelease = e.o.Release, n
					cleared = append(cleared, Ref{r.Quest, r.Chunk})
					olds = append(olds, old)
				}
			}
		}
		return nil
	})
	if err != nil {
		e.o.Log("quest board: release retry not saved", "err", err)
		return
	}
	u, _ := e.act.(unparker)
	for i, ref := range cleared {
		e.o.Log(fmt.Sprintf("quest board: %s park cleared after release %s -> %s: retrying once", ChunkKey(ref.Quest, ref.Chunk), orNone(olds[i]), e.o.Release), "chunk", ChunkKey(ref.Quest, ref.Chunk))
		if u == nil {
			continue
		}
		if q, err := e.b.Quest(ref.Quest); err == nil {
			if c, err := q.ChunkByID(ref.Chunk); err == nil {
				u.Unpark(q, c)
			}
		}
	}
}

func orNone(s string) string {
	if s == "" {
		return "(unrecorded)"
	}
	return s
}

// unparker is implemented by the bot's actions: it lifts the task executor's block of the chunk's group, which Retry
// has to clear too or the chunk is parked again on the next slice (appendix A.4 row 1).
type unparker interface{ Unpark(q Quest, c Chunk) }

// smRetry is Retry on the machine plus the task executor's group.
func (e *Exec) smRetry(quest, chunk string) {
	d := Difficulties[e.env.Diff()]
	var changed []Ref
	if err := e.st.UpdateSM(e.smDone(), func(m *SM) error {
		var err error
		changed, _, err = m.Retry(e.b, d, quest, chunk, false, e.o.Now())
		return err
	}); err != nil && !strings.Contains(err.Error(), "has no run") {
		e.o.Log("quest board: retry not saved", "chunk", ChunkKey(quest, chunk), "err", err)
	}
	e.lastChanged = changed
	if u, ok := e.act.(unparker); ok {
		if q, err := e.b.Quest(quest); err == nil {
			if c, err := q.ChunkByID(chunk); err == nil {
				u.Unpark(q, c)
			}
		}
	}
}

// smRetryAll clears every stored park (the player's Reset).
func (e *Exec) smRetryAll() {
	d := Difficulties[e.env.Diff()]
	_ = e.st.UpdateSM(e.smDone(), func(m *SM) error {
		for _, r := range m.Live(e.b, d) {
			if r.Status == RunParked && r.Code != "game_failed" {
				m.Retry(e.b, d, r.Quest, r.Chunk, false, e.o.Now())
			}
		}
		return nil
	})
}

// holdLocked records a hold (a guard refused for a reason that is no failure). The first refusal starts the clock; a
// hold older than the budget of its code parks the run with hold_timeout (I10, Q10). Callers hold e.mu.
func (e *Exec) holdLocked(d Decision, rs *runState, res StepResult) {
	info, ok := ReasonOf(res.Code)
	if !ok || info.Kind != KindHold {
		// a code outside the closed list is a bug of the producer: count it as the failure it used to be
		rs.failures++
		rs.lastWhy = res.Why
		return
	}
	now := e.o.Now()
	if rs.hold != res.Code {
		rs.hold, rs.holdFrom = res.Code, now
	}
	rs.holdWhy = res.Why
	rs.started = true
	if info.BudgetMin > 0 && now.Sub(rs.holdFrom) >= time.Duration(info.BudgetMin)*time.Minute {
		rs.blocked = holdTimeoutText(res.Code, res.Why, info.BudgetMin)
		e.o.Log("quest board: parked "+ChunkKey(d.Quest, d.Chunk), "chunk", ChunkKey(d.Quest, d.Chunk), "why", rs.blocked, "hold", res.Code)
		e.smPark(d.Quest, d.Chunk, "hold_timeout", rs)
	}
}

// smAutoFinish closes the live run of a chunk whose view reads done or skipped although the chunk's own check does not
// (the quest word says done for the whole quest). Without it that run stays live for ever, nothing can stop it from
// the board (a done row has no Stop) and D5 blocks every other quest. It latches the chunk with source auto-finish and
// reports whether it changed the file.
func (e *Exec) smAutoFinish(vs []QuestView, diff int) bool {
	d := Difficulties[diff]
	var stale []Ref
	var tiers []Tier
	m, err := e.st.LoadSM(nil)
	if err != nil {
		return false
	}
	for _, r := range m.Live(e.b, d) {
		if _, cv, ok := chunkView(vs, r.Quest, r.Chunk); ok && (cv.State == Done || cv.State == Skipped) {
			stale = append(stale, Ref{r.Quest, r.Chunk})
			tiers = append(tiers, cv.Tier)
		}
	}
	if len(stale) == 0 {
		return false
	}
	err = e.st.UpdateSM(nil, func(m *SM) error {
		for i, r := range stale {
			m.NoteDone(e.b, d, r.Quest, r.Chunk, tiers[i], e.o.Now(), "auto-finish")
			e.o.Log("quest board: auto-finish "+r.String()+": the quest reads done, the stale run was closed", "quest", r.Quest, "chunk", r.Chunk)
		}
		return nil
	})
	return err == nil
}

// boardLegacy is the one switch between the old derive-every-tick board and the run machine (knob quests.boardSM 0).
// Removal of the old path deletes every branch that reads it (docs/questboard.md, "boardSM legacy removal").
func (e *Exec) boardLegacy() bool { return !e.o.SM }
