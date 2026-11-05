package watch

import (
	"fmt"
	"sync"

	"d2lod-bot/internal/goals"
)

// This file is the seam to the real bot (docs/research/watch-screen-design.md, "Bot integration"): orders become goals
// of internal/goals, goal events and runner status feed the Hub's activity timeline. It imports only the goals types;
// *goals.Manager satisfies GoalQueue as it is.

// GoalQueue is the part of *goals.Manager the orders need.
type GoalQueue interface {
	Add(g goals.Goal, pos int) (goals.Goal, error)
	Remove(id string) error
}

// GoalsOrders is an OrderSink over a goal queue. Here names the area the hero stands in (for "fight here"); Rules
// returns the keys of the live rules for the death incident (nil: none named).
type GoalsOrders struct {
	Q     GoalQueue
	Hub   *Hub
	Here  func() string
	Rules func() []string

	mu     sync.Mutex
	byGoal map[string]int // goal id -> order id
	byOrd  map[int]string
	lost   string // "" playing; KindDeath after a death; KindRecovery after the respawn, until the first kill
}

func (g *GoalsOrders) Order(o Order) error {
	here := ""
	if g.Here != nil {
		here = g.Here()
	}
	goal, err := OrderToGoal(o, here)
	if err != nil {
		return err
	}
	// pos 0: a player's order goes to the front of the queue (U: the manager sorts by priority, which OrderToGoal sets).
	added, err := g.Q.Add(goal, 0)
	if err != nil {
		return err
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.byGoal == nil {
		g.byGoal, g.byOrd = map[string]int{}, map[int]string{}
	}
	g.byGoal[added.ID], g.byOrd[o.ID] = o.ID, added.ID
	return nil
}

func (g *GoalsOrders) Cancel(id int) error {
	g.mu.Lock()
	gid, ok := g.byOrd[id]
	g.mu.Unlock()
	if !ok {
		return ErrNoOrder
	}
	return g.Q.Remove(gid)
}

// OnGoalEvent maps the goals manager's events to the screen: goal state changes update the orders box, a death starts a
// death period and posts the incident (the rules in force are named, never revoked), an area change or a kill ends a
// recovery. Call it from the loop that reads Manager.Subscribe.
func (g *GoalsOrders) OnGoalEvent(ev goals.Event) {
	g.mu.Lock()
	id, mine := g.byGoal[ev.GoalID]
	g.mu.Unlock()
	switch ev.Type {
	case goals.EvGoalStarted:
		if mine {
			g.Hub.OrderState(id, OrderActive, "")
		}
	case goals.EvGoalDone:
		if mine {
			g.Hub.OrderState(id, OrderDone, "")
		}
	case goals.EvGoalFailed, goals.EvGoalSkipped:
		if mine {
			g.Hub.OrderState(id, OrderFailed, ev.Message)
		}
	case goals.EvGoalRemoved:
		if mine {
			g.Hub.OrderState(id, OrderCancelled, "")
		}
	case goals.EvDeath:
		var keys []string
		if g.Rules != nil {
			keys = g.Rules()
		}
		msg := "The hero died."
		if ev.Area != "" {
			msg = fmt.Sprintf("The hero died in %s.", safeText(ev.Area, 60))
		}
		if len(keys) > 0 {
			msg += " These rules were in force; revoke one if it was a bad habit."
		}
		g.Hub.Incident(msg, keys...)
		g.mu.Lock()
		g.lost = KindDeath
		g.mu.Unlock()
		g.Hub.Activity(KindDeath, "dead")
	case goals.EvAreaChange:
		// The first area change after a death is the respawn: recovery lasts until the hero kills something again.
		g.mu.Lock()
		first := g.lost == KindDeath
		if first {
			g.lost = KindRecovery
		}
		g.mu.Unlock()
		if first {
			g.Hub.Activity(KindRecovery, "back after dying")
		}
	case goals.EvKill:
		g.mu.Lock()
		was := g.lost
		g.lost = ""
		g.mu.Unlock()
		if was != "" {
			g.Hub.Activity(KindProgress, "fighting")
		}
	case goals.EvError:
		g.Hub.Activity(KindStall, ev.Message)
	}
}

// OnStatus feeds the runner's status (poll it every few seconds): it sets the activity unless a death or its recovery
// is still being counted, which only a kill ends.
func (g *GoalsOrders) OnStatus(st goals.Status) {
	g.mu.Lock()
	lost := g.lost
	g.mu.Unlock()
	if lost != "" {
		return
	}
	kind, label := ActivityFromStatus(st)
	g.Hub.Activity(kind, label)
}

// ActivityFromStatus maps the runner's status to an activity kind and label. U: the town-stall rule is a placeholder
// (a running hero whose activity text names town waiting); the bot should report stalls explicitly with Hub.Activity.
func ActivityFromStatus(st goals.Status) (kind, label string) {
	label = st.Activity
	switch st.State {
	case goals.RunnerRunning:
		return KindProgress, label
	case goals.RunnerError:
		return KindStall, st.Message
	default:
		return KindIdle, label
	}
}
