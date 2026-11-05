package watch

import (
	"errors"
	"testing"
	"time"

	"d2lod-bot/internal/goals"
)

type fakeQueue struct {
	added   []goals.Goal
	removed []string
}

func (q *fakeQueue) Add(g goals.Goal, pos int) (goals.Goal, error) {
	g.ID = "g" + string(rune('1'+len(q.added)))
	q.added = append(q.added, g)
	return g, nil
}
func (q *fakeQueue) Remove(id string) error {
	if id == "" {
		return errors.New("no id")
	}
	q.removed = append(q.removed, id)
	return nil
}

// TestGoalsOrdersAgainstAFakeBot: orders from the answer route reach the goal queue, goal events come back as order
// states, a death posts an incident that names the rules and revokes nothing.
func TestGoalsOrdersAgainstAFakeBot(t *testing.T) {
	r := newRig(t, "paladin")
	q := &fakeQueue{}
	b := &GoalsOrders{Q: q, Hub: r.sim.Hub, Here: func() string { return "Cold Plains" },
		Rules: func() []string {
			var ks []string
			for _, x := range r.sim.Store.Rules() {
				ks = append(ks, x.Key)
			}
			return ks
		}}
	r.sim.Hub.cfg.Orders = b // the real bot's sink instead of the sim's
	r.must("/api/orders", map[string]any{"kind": "go", "area": "Dark Wood", "minutes": 15})
	r.must("/api/orders", map[string]any{"kind": "fight", "minutes": 5})
	if len(q.added) != 2 || q.added[0].Params.Area != "Dark Wood" || q.added[0].Params.UntilMins != 15 || q.added[1].Params.Area != "Cold Plains" {
		t.Fatalf("goals: %+v", q.added)
	}
	b.OnGoalEvent(goals.Event{Type: goals.EvGoalStarted, GoalID: "g1"})
	var ov OrdersView
	r.get("/api/orders", &ov)
	if ov.Orders[0].State != OrderActive || ov.Orders[1].State != OrderQueued {
		t.Fatalf("after start: %+v", ov.Orders)
	}
	b.OnGoalEvent(goals.Event{Type: goals.EvGoalDone, GoalID: "g1"})
	b.OnGoalEvent(goals.Event{Type: goals.EvGoalFailed, GoalID: "g2", Message: "area unknown"})
	r.get("/api/orders", &ov)
	if ov.Orders[0].State != OrderDone || ov.Orders[1].State != OrderFailed || ov.Orders[1].Note != "area unknown" {
		t.Fatalf("after events: %+v", ov.Orders)
	}
	r.must("/api/orders", map[string]any{"kind": "home"})
	r.must("/api/orders/cancel", map[string]any{"id": 3})
	if len(q.removed) != 1 || q.removed[0] != "g3" {
		t.Errorf("cancel did not reach the queue: %v", q.removed)
	}
	// Goals the screen did not order are ignored.
	b.OnGoalEvent(goals.Event{Type: goals.EvGoalDone, GoalID: "other"})
}

// A hardcore Paladin learns a rule, then the bot reports a death: the incident names the rule, the rule stays.
func TestDeathEventFromTheGoalsManager(t *testing.T) {
	r := newRig(t, "paladin")
	b := &GoalsOrders{Hub: r.sim.Hub, Rules: func() []string {
		var ks []string
		for _, x := range r.sim.Store.Rules() {
			ks = append(ks, x.Key)
		}
		return ks
	}}
	r.fight()
	answerUntilRule(r, "portal")
	r.must("/api/confirm", map[string]any{"thought": r.thoughts().Thoughts[0].ID, "yes": true})
	r.sim.Step(time.Second)
	b.OnGoalEvent(goals.Event{Type: goals.EvDeath, Area: "Dark Wood"})
	var v ThoughtsView
	r.get("/api/thoughts", &v)
	last := v.Feed[len(v.Feed)-1]
	if last.Kind != "incident" || len(last.Rules) != 1 || !last.Rules[0].Live {
		t.Fatalf("incident: %+v", last)
	}
	if r.rules().Live != 1 {
		t.Error("the death revoked a rule")
	}
	if u := r.uptime(); u.State != KindDeath || u.Deaths != 1 {
		t.Errorf("uptime after death: %+v", u)
	}
}

func TestActivityFromStatus(t *testing.T) {
	for _, c := range []struct {
		st   goals.Status
		kind string
	}{
		{goals.Status{State: goals.RunnerRunning, Activity: "Fighting"}, KindProgress},
		{goals.Status{State: goals.RunnerIdle}, KindIdle},
		{goals.Status{State: goals.RunnerPaused}, KindIdle},
		{goals.Status{State: goals.RunnerError, Message: "stuck"}, KindStall},
	} {
		if k, _ := ActivityFromStatus(c.st); k != c.kind {
			t.Errorf("%+v -> %s", c.st, k)
		}
	}
}
