package event

import "d2lod-bot/internal/goals"

var goalsKind = map[goals.EventType]Kind{
	goals.EvKill: KindKill, goals.EvDrop: KindDrop, goals.EvDeath: KindDeath, goals.EvLevelUp: KindLevelUp,
	goals.EvAreaChange: KindAreaChange, goals.EvPotion: KindPotion, goals.EvError: KindError,
	goals.EvGoalAdded: KindGoalAdded, goals.EvGoalStarted: KindGoalStarted, goals.EvGoalDone: KindGoalDone,
	goals.EvGoalFailed: KindGoalFailed, goals.EvGoalPaused: KindGoalPaused, goals.EvGoalResumed: KindGoalResumed,
	goals.EvGoalSkipped: KindGoalSkipped, goals.EvGoalRemoved: KindGoalRemoved, goals.EvGoalEdited: KindGoalEdited,
	goals.EvGoalPreempted: KindGoalPreempted, goals.EvControl: KindControl, goals.EvInfo: KindInfo,
}

var kindGoals = func() map[Kind]goals.EventType {
	m := map[Kind]goals.EventType{}
	for t, k := range goalsKind {
		m[k] = t
	}
	return m
}()

func fromItem(i *goals.Item) Item {
	if i == nil {
		return Item{}
	}
	return Item{Name: i.Name, Base: i.Base, Type: i.Type, Quality: i.Quality, Qty: i.Qty}
}

// FromGoals converts a goals.Event. Seq and Time are copied (zero from a runner; the Bus stamps them on
// Publish). source names the producer. Unknown goals types become KindInfo with the type in the message.
func FromGoals(g goals.Event, source string) Event {
	e := Event{V: Version, Seq: g.Seq, Time: g.Time, Goal: g.GoalID, Source: source}
	k, ok := goalsKind[g.Type]
	if !ok {
		e.Kind = KindInfo
		e.Data = Message{Message: string(g.Type) + ": " + g.Message}
		return e
	}
	e.Kind = k
	switch k {
	case KindKill:
		e.Data = Kill{Monster: g.Monster, Boss: g.Boss, XP: g.XP, Area: g.AreaID, AreaName: g.Area}
	case KindDrop:
		e.Data = Drop{Item: fromItem(g.Item), Area: g.AreaID, AreaName: g.Area}
	case KindDeath:
		e.Data = Death{Area: g.AreaID, AreaName: g.Area}
	case KindLevelUp:
		e.Data = LevelUp{Level: g.Level}
	case KindAreaChange:
		e.Data = AreaChange{Area: g.AreaID, AreaName: g.Area}
	case KindPotion:
		e.Data = Potion{Name: g.Potion}
	case KindError:
		e.Data = Error{Message: g.Message, Fatal: g.Fatal}
	case KindControl, KindInfo:
		e.Data = Message{Message: g.Message}
	default:
		e.Data = GoalEvent{GoalID: g.GoalID, Message: g.Message}
	}
	return e
}

// ToGoals converts back for consumers that still take goals.Event. ok is false for kinds goals cannot
// express (failsafe, lifecycle, cheat, and the tracking.md kinds).
func ToGoals(e Event) (g goals.Event, ok bool) {
	t, known := kindGoals[e.Kind]
	if !known {
		return goals.Event{}, false
	}
	g = goals.Event{Seq: e.Seq, Time: e.Time, Type: t, GoalID: e.Goal}
	switch d := e.Payload().(type) {
	case Kill:
		g.Monster, g.Boss, g.XP, g.AreaID, g.Area = d.Monster, d.Boss, d.XP, d.Area, d.AreaName
	case Drop:
		g.Item = &goals.Item{Name: d.Item.Name, Base: d.Item.Base, Type: d.Item.Type, Quality: d.Item.Quality, Qty: d.Item.Qty}
		g.AreaID, g.Area = d.Area, d.AreaName
	case Death:
		g.AreaID, g.Area = d.Area, d.AreaName
	case LevelUp:
		g.Level = d.Level
	case AreaChange:
		g.AreaID, g.Area = d.Area, d.AreaName
	case Potion:
		g.Potion = d.Name
	case Error:
		g.Message, g.Fatal = d.Message, d.Fatal
	case Message:
		g.Message = d.Message
	case GoalEvent:
		g.GoalID, g.Message = d.GoalID, d.Message
	}
	return g, true
}
