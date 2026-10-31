package bot

import (
	"errors"
	"fmt"

	"d2lod-bot/internal/launch"
)

// State is the bot's operating state (the state machine in docs/bot.md).
type State string

const (
	StateIdle        State = "idle"        // no goal, or resting
	StateInTown      State = "in_town"     // working the goal from town: planning, finishing, resting
	StateTraveling   State = "traveling"   // walking, taking exits and waypoints to the target area
	StateClearing    State = "clearing"    // exploring, fighting and looting in the target area
	StateResupplying State = "resupplying" // town trip: portal home, chores, portal back
	StateRecovering  State = "recovering"  // after death: respawn, heal, corpse
	StatePaused      State = "paused"
	StateError       State = "error" // waiting after an error that needs a new goal, a retry timer or an operator
)

// Reasons an action was cancelled. They are context causes: the guard or a control call cancels the running action
// with one of them and the main loop reacts after the action returns.
var (
	errNewGoal     = errors.New("bot: goal changed")
	errStop        = errors.New("bot: stop requested")
	errPause       = errors.New("bot: pause requested")
	errStuck       = errors.New("bot: no progress")
	errDied        = errors.New("bot: character died")
	errLeftGame    = errors.New("bot: left the game")
	errTripDue     = errors.New("bot: town trip due")
	errReplan      = errors.New("bot: level changed, re-plan")
	errStepTimeout = errors.New("bot: action timed out")
	errGuardPanic  = errors.New("bot: guard panicked, action cancelled")
	errPoints      = errors.New("bot: unspent points, calm moment")
)

// safetyInterrupt cancels an action so the main loop can carry out the fighter's decision.
type safetyInterrupt struct{ Action SafetyAction }

func (e safetyInterrupt) Error() string { return "bot: safety " + e.Action.String() }

// gameGone means the game process exited, crashed or hung.
type gameGone struct{ Event launch.Event }

func (e gameGone) Error() string { return fmt.Sprintf("bot: game %s", e.Event) }
