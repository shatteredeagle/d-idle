package goals

import "time"

// Runner is what plays the game. The real bot (wave 2) and sim.Runner implement it.
//
// Contract:
//   - Start, Pause, Resume, Stop and Status must return quickly (well under 100 ms) and never call the
//     Manager; the Manager calls them while holding its lock. Slow work belongs in the runner's own goroutine.
//   - Start(g) begins working on g.Type/g.Params, replacing any goal already running. Only the goal's
//     parameters matter: progress and completion are decided by the manager from the events below.
//   - Pause freezes play at the next safe moment (not mid-fight in a dangerous spot); Resume continues.
//     Stop abandons the current goal and leaves the character safe (town if possible). All are idempotent.
//   - Events returns one long-lived channel. The runner emits Kill, Drop, Death, LevelUp, AreaChange, Potion
//     and Error events as they happen, never blocks on it (drop the oldest if full), and leaves Time and Seq zero
//     (the manager stamps them). Kill events carry Monster, Boss and XP; Drop events carry Item.
//   - Status returns a consistent snapshot of the character and what the bot is doing.
type Runner interface {
	Start(g Goal) error
	Pause() error
	Resume() error
	Stop() error
	Status() Status
	Events() <-chan Event
}

// ProgressReporter is optionally implemented by a Runner that tracks a goal itself (the bot's campaign goals): the
// manager merges the report into the active goal's Progress on every tick and ends the goal when it is complete.
type ProgressReporter interface {
	Progress() (Progress, bool)
}

// Configurer is optionally implemented by a Runner that wants the user's settings.
// The manager calls Configure at startup and whenever settings change.
type Configurer interface {
	Configure(Settings)
}

type RunnerState string

const (
	RunnerIdle    RunnerState = "idle"    // no goal
	RunnerRunning RunnerState = "running" // playing
	RunnerPaused  RunnerState = "paused"
	RunnerStopped RunnerState = "stopped"
	RunnerError   RunnerState = "error" // needs attention; Message says why
)

type Point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type Character struct {
	Name       string `json:"name"`
	Class      string `json:"class"`
	Level      int    `json:"level"`
	XP         int64  `json:"xp"`
	XPNext     int64  `json:"xpNext"` // experience needed for the next level; 0 if unknown
	Life       int    `json:"life"`
	MaxLife    int    `json:"maxLife"`
	Mana       int    `json:"mana"`
	MaxMana    int    `json:"maxMana"`
	Area       string `json:"area"`
	AreaID     int    `json:"areaId"`
	Position   Point  `json:"position"`
	Difficulty string `json:"difficulty,omitempty"`
}

type Status struct {
	State     RunnerState `json:"state"`
	Activity  string      `json:"activity"` // short human text: "Fighting", "Walking to Cold Plains", "Selling in town"
	GoalID    string      `json:"goalId,omitempty"`
	Message   string      `json:"message,omitempty"`
	Character Character   `json:"character"`
	Updated   time.Time   `json:"updated"`
}

type EventType string

const (
	EvKill       EventType = "kill"
	EvDrop       EventType = "drop"
	EvDeath      EventType = "death"
	EvLevelUp    EventType = "level_up"
	EvAreaChange EventType = "area_change"
	EvPotion     EventType = "potion"
	EvError      EventType = "error"
	EvQuest      EventType = "quest" // a quest changed stage; Quest, Stage, Done, Diff say which

	// Emitted by the manager itself.
	EvGoalAdded     EventType = "goal_added"
	EvGoalStarted   EventType = "goal_started"
	EvGoalDone      EventType = "goal_done"
	EvGoalFailed    EventType = "goal_failed"
	EvGoalPaused    EventType = "goal_paused"
	EvGoalResumed   EventType = "goal_resumed"
	EvGoalSkipped   EventType = "goal_skipped"
	EvGoalRemoved   EventType = "goal_removed"
	EvGoalEdited    EventType = "goal_edited"
	EvControl       EventType = "control" // start/pause/stop of the whole bot
	EvInfo          EventType = "info"
	EvGoalPreempted EventType = "goal_preempted"
)

// Quality values used in Item.Quality and ItemFilter.Quality.
var Qualities = []string{"normal", "magic", "rare", "set", "unique", "crafted"}

type Item struct {
	Name    string `json:"name"`
	Base    string `json:"base,omitempty"`
	Type    string `json:"type,omitempty"`
	Quality string `json:"quality,omitempty"`
	Qty     int    `json:"qty,omitempty"` // stack size; 0 means 1
}

func (i Item) qty() int {
	if i.Qty <= 0 {
		return 1
	}
	return i.Qty
}

type Event struct {
	Seq     uint64    `json:"seq"`
	Time    time.Time `json:"time"`
	Type    EventType `json:"type"`
	Message string    `json:"message,omitempty"`
	GoalID  string    `json:"goalId,omitempty"`
	Monster string    `json:"monster,omitempty"`
	Boss    bool      `json:"boss,omitempty"`
	XP      int64     `json:"xp,omitempty"`
	Item    *Item     `json:"item,omitempty"`
	Level   int       `json:"level,omitempty"`
	Area    string    `json:"area,omitempty"`
	AreaID  int       `json:"areaId,omitempty"`
	Potion  string    `json:"potion,omitempty"`
	Fatal   bool      `json:"fatal,omitempty"` // Error only: the bot cannot continue
	Hold    bool      `json:"hold,omitempty"`  // Error only: the active goal cannot run yet; hold it until the next level up
	// Quest events: Quest is the quest id, Stage its new stage (task.Stage value), Done true once it is complete.
	// Hold errors: Unlock names the unlock flag the goal waits for (e.g. anya_portal); the matching Done quest event
	// resumes it. Diff is the difficulty (0 normal, 1 nightmare, 2 hell) both refer to.
	Quest  string `json:"quest,omitempty"`
	Stage  int    `json:"stage,omitempty"`
	Done   bool   `json:"done,omitempty"`
	Unlock string `json:"unlock,omitempty"`
	Diff   int    `json:"diff,omitempty"`
}

func (e *Event) describe() string {
	switch e.Type {
	case EvKill:
		s := "Killed " + e.Monster
		if e.Boss {
			s = "Defeated " + e.Monster
		}
		if e.XP > 0 {
			s += " (+" + itoa(e.XP) + " xp)"
		}
		return s
	case EvDrop:
		if e.Item != nil {
			q := e.Item.Quality
			if q != "" && q != "normal" {
				q += " "
			} else {
				q = ""
			}
			return "Found " + q + e.Item.Name
		}
		return "Found an item"
	case EvDeath:
		return "Died" + inArea(e.Area)
	case EvLevelUp:
		return "Reached level " + itoa(int64(e.Level))
	case EvAreaChange:
		return "Entered " + e.Area
	case EvPotion:
		return "Used " + e.Potion
	case EvError:
		return "Error"
	case EvQuest:
		if e.Done {
			return "Quest done: " + e.Quest
		}
		return "Quest progress: " + e.Quest
	}
	return string(e.Type)
}

func inArea(a string) string {
	if a == "" {
		return ""
	}
	return " in " + a
}
