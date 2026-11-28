package bot

import (
	"fmt"
	"sync"
	"time"

	"d2lod-bot/internal/admit"
	"d2lod-bot/internal/d2"
	"d2lod-bot/internal/task"
)

// Wiring of internal/admit (card admit1, step 2). The old judges stay in control while bot.admission is 0: the function runs
// beside them (it is pure and cheap), a disagreement is logged once per distinct case as `admission shadow`, and the counters
// and the last verdict go to `ops status --why`. With bot.admission 1 the function decides the four entry rows (early road,
// poverty, kit floor, lean) at areaGuardWhy and the quest-walk kit at the town exit (fillEarly). Providers that are not rows
// (route guard, Barracks, tower, boss floors) are untouched: areaGuardWhy asks them first.

// AdmissionExplain is the admission line of `ops status --why`: the last verdict and how often the function agreed with the old judge.
type AdmissionExplain struct {
	Mode   string `json:"mode"` // "shadow" (old judge decides) or "control" (admit decides)
	Agree  int    `json:"agree"`
	Differ int    `json:"differ"`
	// The cells behind the two counters. Only AgreeRefused is evidence that a row fired and both sides refused; a window
	// with AgreeRefused 0 exercised no row. OldOnly and NewOnly are the mixed cells (one side refused, the other let through).
	AgreeRefused int       `json:"agreeRefused,omitempty"`
	OldOnly      int       `json:"oldOnly,omitempty"`
	NewOnly      int       `json:"newOnly,omitempty"`
	Words        int       `json:"words,omitempty"` // both refused, with different words
	Last         string    `json:"last,omitempty"`  // area, purpose and the words of the last non-open verdict
	Row          string    `json:"row,omitempty"`
	Need         string    `json:"need,omitempty"`
	Wake         string    `json:"wake,omitempty"`
	Seen         time.Time `json:"seen,omitempty"`
}

type admitState struct {
	mu     sync.Mutex
	st     AdmissionExplain
	logged map[string]bool
}

const admitLogMax = 64

// admitView fills the function's View from the snapshot and the knobs. EarlyRoute 0 turns every row off, as it does for the old judges.
func (r *Runner) admitView(s *d2.Snapshot) admit.View {
	p := s.Player
	in := r.resInput(s)
	fields := r.fieldsOfAct(s)
	v := admit.View{
		Archetype: "melee",
		Normal:    p.Difficulty == d2.Normal && r.e.Tune.Route.EarlyRoute != 0,
		Level:     p.Level, Life: p.Life, MaxLife: p.MaxLife,
		Healing: healingOf(in), HasTP: in.HasTP, Gold: p.Gold,
		InTown: p.InTown(), Area: p.Area, Fields: fields,
		Try:        task.TryActive(),
		IsTown:     r.d.Plan.IsTown,
		MLvl:       func(a int) int { return r.areaMLvl(a, p.Difficulty) },
		PotionGold: r.e.Tune.Resource.KitPotionGold, TPGold: r.e.Tune.Resource.KitTPGold,
	}
	if p.Class.Caster() {
		v.Archetype = "caster"
	}
	if len(fields) > 0 {
		v.FirstField = fields[0]
	}
	return v
}

// rowsEnterWhy is the old path of the four row judges, in their old order.
func (r *Runner) rowsEnterWhy(s *d2.Snapshot, a int) string {
	if why := r.earlyRouteWhy(s, a); why != "" {
		return why
	}
	if why := r.povertyAreaWhy(s, a); why != "" {
		return why
	}
	if why := r.kitFloorWhy(s, a); why != "" {
		return why
	}
	return r.leanEntryWhy(s, a)
}

// admitEnter is the entry rows of areaGuardWhy: the old judges, shadowed by admit.MayEnter, or decided by it under bot.admission.
func (r *Runner) admitEnter(s *d2.Snapshot, a int) string {
	old := r.rowsEnterWhy(s, a)
	if s == nil || s.Player == nil {
		return old
	}
	p := admit.Level
	if r.corpseRun.Load() {
		p = admit.CorpseRun
	}
	nv, err := admit.MayEnter(r.admitView(s), a, p)
	if err != nil {
		return old // unreadable rows: the old judges stay in charge
	}
	return r.admitDecide(a, p, old, nv, old)
}

// admitLeave is the quest-walk kit at the town exit (questWalkCheck). Only the quest-walk row is taken from the function: its lean
// life line at the exit has no old judge, so it is logged (as a difference) and never acted on yet.
func (r *Runner) admitLeave(s *d2.Snapshot) (why string, kitOnly bool) {
	old, oldKit := r.questWalkCheck(s)
	if s == nil || s.Player == nil {
		return old, oldKit
	}
	p := admit.QuestWalk
	if r.corpseRun.Load() {
		p = admit.CorpseRun
	}
	nv, err := admit.MayLeaveTown(r.admitView(s), p)
	if err != nil {
		return old, oldKit
	}
	if nv.Row != "quest-walk" {
		nv.Refused = false // the lean exit line is for the shadow log only
		if nv.Row != "" {
			r.admitNote(0, p, old, nv, true)
		}
		nv = admit.Allow
	}
	got := r.admitDecide(0, p, old, nv, old)
	if r.e.Tune.Bot.Admission != 0 {
		return got, nv.KitOnly
	}
	return old, oldKit
}

// admitDecide notes the pair and returns the words to act on: the function's under bot.admission, else fallback.
func (r *Runner) admitDecide(a int, p admit.Purpose, old string, nv admit.Verdict, fallback string) string {
	r.admitNote(a, p, old, nv, false)
	if r.e.Tune.Bot.Admission != 0 {
		return nv.Why
	}
	return fallback
}

// admitNote counts agreement (same words, or both open), logs a disagreement once per case and keeps the last refusal for ops.
func (r *Runner) admitNote(a int, p admit.Purpose, old string, nv admit.Verdict, extra bool) {
	x := &r.admit
	x.mu.Lock()
	defer x.mu.Unlock()
	x.st.Mode = "shadow"
	if r.e.Tune.Bot.Admission != 0 {
		x.st.Mode = "control"
	}
	if nv.Refused {
		x.st.Last = fmt.Sprintf("area %d %s: %s", a, p, nv.Why)
		x.st.Row, x.st.Need, x.st.Wake, x.st.Seen = nv.Row, nv.Need, nv.Wake, r.e.Clock.Now()
	}
	if !extra && old == nv.Why {
		x.st.Agree++
		if old != "" {
			x.st.AgreeRefused++
		}
		return
	}
	x.st.Differ++
	cell := "words" // both refuse, for different reasons
	switch {
	case !extra && old != "" && nv.Refused:
		x.st.Words++
	case extra || old == "" && nv.Refused:
		cell = "new-only"
		x.st.NewOnly++
	case old != "" && !nv.Refused:
		cell = "old-only"
		x.st.OldOnly++
	}
	key := fmt.Sprintf("%d|%s|%s|%s", a, p, old, nv.Why)
	if x.logged[key] || len(x.logged) >= admitLogMax {
		return
	}
	if x.logged == nil {
		x.logged = map[string]bool{}
	}
	x.logged[key] = true
	r.e.Log.Info("admission shadow", "differ", cell, "area", a, "purpose", string(p), "old", old, "new", nv.Why, "row", nv.Row, "need", nv.Need)
}

// admissionExplain is the admission line of Explain.
func (r *Runner) admissionExplain() AdmissionExplain {
	x := &r.admit
	x.mu.Lock()
	defer x.mu.Unlock()
	st := x.st
	if st.Mode == "" {
		st.Mode = "shadow"
		if r.e.Tune.Bot.Admission != 0 {
			st.Mode = "control"
		}
	}
	return st
}
