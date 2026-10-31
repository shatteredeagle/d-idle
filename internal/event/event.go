// Package event is the one typed, versioned event model for the bot: behaviors, the observer, the manager,
// the sentinel and the launcher publish to a Bus; the API/SSE stream, telemetry and logs subscribe.
//
// Wire format: a flat JSON envelope {v,seq,time,instance,run,game,goal,kind,source,data}; data is the typed
// payload for the kind (see Payload). Unknown kinds round-trip as raw JSON, so a newer producer never breaks
// an older reader of the same schema version. Bumping Version means a breaking envelope change; readers
// reject a larger V with ErrFutureVersion.
//
// Package event deliberately knows nothing about goal types: goal.* events only transport ids and text.
// adapter.go imports internal/goals for the transition period; delete it when goals consumes the bus directly.
package event

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Version is the envelope schema version written by this package.
const Version = 1

var ErrFutureVersion = errors.New("event: schema version newer than this build understands")

type Kind string

const (
	KindKill       Kind = "kill"
	KindDrop       Kind = "drop"
	KindDeath      Kind = "death"
	KindLevelUp    Kind = "level_up"
	KindAreaChange Kind = "area_change"
	KindPotion     Kind = "potion"
	KindError      Kind = "error"
	KindFailsafe   Kind = "failsafe"
	KindLifecycle  Kind = "lifecycle"
	KindCheat      Kind = "cheat"
	KindActivity   Kind = "activity"
	KindControl    Kind = "control" // start/pause/stop of the whole bot
	KindInfo       Kind = "info"

	KindGoalAdded     Kind = "goal.added"
	KindGoalStarted   Kind = "goal.started"
	KindGoalDone      Kind = "goal.done"
	KindGoalFailed    Kind = "goal.failed"
	KindGoalPaused    Kind = "goal.paused"
	KindGoalResumed   Kind = "goal.resumed"
	KindGoalSkipped   Kind = "goal.skipped"
	KindGoalRemoved   Kind = "goal.removed"
	KindGoalEdited    Kind = "goal.edited"
	KindGoalPreempted Kind = "goal.preempted"

	// Proposed in docs/goals/tracking.md section 3; transported, not interpreted, here.
	KindQuestStep         Kind = "quest_step"
	KindQuestDone         Kind = "quest_done"
	KindWaypointActivated Kind = "waypoint_activated"
	KindDifficultyChange  Kind = "difficulty_change"
	KindActChange         Kind = "act_change"
	KindItemAcquired      Kind = "item_acquired"
	KindItemEquipped      Kind = "item_equipped"
	KindPointsSpent       Kind = "points_spent"
	KindGold              Kind = "gold"
	KindTownTrip          Kind = "town_trip"
	KindChicken           Kind = "chicken"
	KindCorpseRun         Kind = "corpse_run"
)

// Sources used in Event.Source.
const (
	SrcObserver = "observer"
	SrcCombat   = "combat"
	SrcLoot     = "loot"
	SrcTown     = "town"
	SrcNav      = "nav"
	SrcSentinel = "sentinel"
	SrcManager  = "manager"
	SrcLaunch   = "launch"
	SrcCheat    = "cheat"
)

// DerivedKinds are derived once, from snapshot deltas, by the observer (design-gaps D2). Behaviors must not
// publish them; they publish only intent (chicken, town_trip, ...). Bus.Publish does not enforce this; the
// table is for the producers' lint test and for dedupe in the migration.
var DerivedKinds = map[Kind]bool{
	KindKill: true, KindDrop: true, KindDeath: true, KindLevelUp: true, KindAreaChange: true, KindPotion: true,
}

// Event is the envelope. Seq is assigned by the Bus (per bus, monotonic from 1).
type Event struct {
	V        int       `json:"v"`
	Seq      uint64    `json:"seq"`
	Time     time.Time `json:"time"`
	Instance string    `json:"instance,omitempty"`
	Run      string    `json:"run,omitempty"`  // runner session id (process start)
	Game     int       `json:"game,omitempty"` // in-game session counter (each EnterGame)
	Goal     string    `json:"goal,omitempty"` // goal id the event belongs to, if any
	Kind     Kind      `json:"kind"`
	Source   string    `json:"source,omitempty"`
	Data     any       `json:"data,omitempty"` // typed payload for Kind; json.RawMessage for unknown kinds
}

type Item struct {
	Name    string `json:"name"`
	Base    string `json:"base,omitempty"`
	Type    string `json:"type,omitempty"`
	Quality string `json:"quality,omitempty"`
	Qty     int    `json:"qty,omitempty"` // 0 means 1
}

type Kill struct {
	Monster  string `json:"monster"`
	ClassID  int    `json:"classId,omitempty"`
	Boss     bool   `json:"boss,omitempty"`
	XP       int64  `json:"xp,omitempty"`
	Area     int    `json:"area,omitempty"`
	AreaName string `json:"areaName,omitempty"`
}

type Drop struct {
	Item     Item   `json:"item"`
	Area     int    `json:"area,omitempty"`
	AreaName string `json:"areaName,omitempty"`
}

type Death struct {
	Area       int      `json:"area,omitempty"`
	AreaName   string   `json:"areaName,omitempty"`
	Pos        [2]int   `json:"pos"`
	Killer     string   `json:"killer,omitempty"`
	Affixes    []string `json:"affixes,omitempty"`
	LifeCurve  []int    `json:"lifeCurve,omitempty"`
	Difficulty int      `json:"difficulty,omitempty"`
	CharLevel  int      `json:"charLevel,omitempty"`
}

type LevelUp struct {
	Level int `json:"level"`
}

type AreaChange struct {
	Area     int    `json:"area"`
	AreaName string `json:"areaName,omitempty"`
}

type Potion struct {
	Name string `json:"name"`
}

type Error struct {
	Message string `json:"message"`
	Fatal   bool   `json:"fatal,omitempty"` // the bot cannot continue
}

type Failsafe struct {
	Layer  string `json:"layer"`
	Rule   string `json:"rule"`
	Step   int    `json:"step,omitempty"`
	Action string `json:"action,omitempty"`
	Result string `json:"result,omitempty"`
}

// Lifecycle covers the game process and the runner: State is e.g. "starting","running","exited","restarting".
type Lifecycle struct {
	State   string `json:"state"`
	Reason  string `json:"reason,omitempty"`
	Message string `json:"message,omitempty"`
}

// Cheat mirrors a cheat-ledger entry (internal/cheat); the ledger stays the source of truth.
type Cheat struct {
	Cheat  string `json:"cheat"`
	Reason string `json:"reason"`
	Detail string `json:"detail,omitempty"`
}

// Activity is a coarse "what is the bot doing" change (telemetry segments).
type Activity struct {
	State  string `json:"state"`
	Detail string `json:"detail,omitempty"`
}

// Message is the payload of control and info events.
type Message struct {
	Message string `json:"message"`
}

// GoalEvent is the payload of every goal.* kind. Goal types and params are not carried; fetch the goal by id.
type GoalEvent struct {
	GoalID  string `json:"goalId"`
	Message string `json:"message,omitempty"`
}

type QuestStep struct {
	Quest string `json:"quest"`
	Step  string `json:"step"`
	State string `json:"state"`
}

type QuestDone struct {
	Quest      string `json:"quest"`
	Difficulty int    `json:"difficulty"`
}

type WaypointActivated struct {
	Area int `json:"area"`
}

type DifficultyChange struct {
	Difficulty int `json:"difficulty"`
}

type ActChange struct {
	Act int `json:"act"`
}

type ItemAcquired struct {
	Item  Item   `json:"item"`
	Where string `json:"where,omitempty"`
}

type ItemEquipped struct {
	Item Item   `json:"item"`
	Slot string `json:"slot"`
}

type PointsSpent struct {
	Kind  string `json:"kind"` // "stat" | "skill"
	Name  string `json:"name"`
	Level int    `json:"level,omitempty"`
}

type Gold struct {
	Delta  int64  `json:"delta"`
	Reason string `json:"reason"` // loot, sell, buy, repair
}

type TownTrip struct {
	Reason string `json:"reason"`
}

// Empty is the payload of kinds that carry nothing (chicken, corpse_run have only the envelope).
type Empty struct{}

// payloadType returns a new pointer to the payload struct for k, or nil for unknown kinds.
func payloadType(k Kind) any {
	switch k {
	case KindKill:
		return new(Kill)
	case KindDrop:
		return new(Drop)
	case KindDeath:
		return new(Death)
	case KindLevelUp:
		return new(LevelUp)
	case KindAreaChange:
		return new(AreaChange)
	case KindPotion:
		return new(Potion)
	case KindError:
		return new(Error)
	case KindFailsafe:
		return new(Failsafe)
	case KindLifecycle:
		return new(Lifecycle)
	case KindCheat:
		return new(Cheat)
	case KindActivity:
		return new(Activity)
	case KindControl, KindInfo:
		return new(Message)
	case KindGoalAdded, KindGoalStarted, KindGoalDone, KindGoalFailed, KindGoalPaused, KindGoalResumed,
		KindGoalSkipped, KindGoalRemoved, KindGoalEdited, KindGoalPreempted:
		return new(GoalEvent)
	case KindQuestStep:
		return new(QuestStep)
	case KindQuestDone:
		return new(QuestDone)
	case KindWaypointActivated:
		return new(WaypointActivated)
	case KindDifficultyChange:
		return new(DifficultyChange)
	case KindActChange:
		return new(ActChange)
	case KindItemAcquired:
		return new(ItemAcquired)
	case KindItemEquipped:
		return new(ItemEquipped)
	case KindPointsSpent:
		return new(PointsSpent)
	case KindGold:
		return new(Gold)
	case KindTownTrip:
		return new(TownTrip)
	case KindChicken, KindCorpseRun:
		return new(Empty)
	}
	return nil
}

// New builds an event with the current schema version; the Bus fills Seq, Time and Instance.
func New(kind Kind, source string, data any) Event {
	return Event{V: Version, Kind: kind, Source: source, Data: data}
}

// Payload returns the typed payload as a value (Kill, not *Kill), or nil if absent or of an unknown kind.
func (e Event) Payload() any {
	switch d := e.Data.(type) {
	case nil, json.RawMessage:
		return nil
	default:
		return deref(d)
	}
}

func deref(p any) any {
	switch d := p.(type) {
	case *Kill:
		return *d
	case *Drop:
		return *d
	case *Death:
		return *d
	case *LevelUp:
		return *d
	case *AreaChange:
		return *d
	case *Potion:
		return *d
	case *Error:
		return *d
	case *Failsafe:
		return *d
	case *Lifecycle:
		return *d
	case *Cheat:
		return *d
	case *Activity:
		return *d
	case *Message:
		return *d
	case *GoalEvent:
		return *d
	case *QuestStep:
		return *d
	case *QuestDone:
		return *d
	case *WaypointActivated:
		return *d
	case *DifficultyChange:
		return *d
	case *ActChange:
		return *d
	case *ItemAcquired:
		return *d
	case *ItemEquipped:
		return *d
	case *PointsSpent:
		return *d
	case *Gold:
		return *d
	case *TownTrip:
		return *d
	case *Empty:
		return *d
	}
	return p
}

// As reports whether the payload is a T (value or pointer) and returns it by value.
func As[T any](e Event) (T, bool) {
	if v, ok := e.Data.(T); ok {
		return v, true
	}
	if v, ok := e.Data.(*T); ok && v != nil {
		return *v, true
	}
	var zero T
	return zero, false
}

type wire Event // no methods: avoids recursion

type wireIn struct {
	wire
	Data json.RawMessage `json:"data"`
}

// UnmarshalJSON decodes the envelope and, for known kinds, the typed payload (stored as a value, so
// As[Kill](e) works on decoded events). Unknown kinds keep Data as json.RawMessage.
func (e *Event) UnmarshalJSON(b []byte) error {
	var w wireIn
	if err := json.Unmarshal(b, &w); err != nil {
		return err
	}
	if w.V > Version {
		return fmt.Errorf("%w: v=%d, supported %d", ErrFutureVersion, w.V, Version)
	}
	*e = Event(w.wire)
	if e.V == 0 {
		e.V = 1
	}
	e.Data = nil
	if len(w.Data) == 0 || string(w.Data) == "null" {
		return nil
	}
	p := payloadType(e.Kind)
	if p == nil {
		e.Data = w.Data
		return nil
	}
	if err := json.Unmarshal(w.Data, p); err != nil {
		return fmt.Errorf("event %s: %w", e.Kind, err)
	}
	e.Data = deref(p)
	return nil
}
