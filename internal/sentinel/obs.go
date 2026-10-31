// Package sentinel is layer 2 of the failsafes (PLAN.md "Failsafes"): an out-of-process watcher that does
// NOT trust the bot. It observes each game instance (process, window frame, game memory, input helper,
// dialogs, optional bot health), runs a table of named rules with durations, cooldowns, attempt limits and
// escalation, and (only when acting is enabled) repairs bad states: death overlay, stuck panels, crashes,
// frozen games, lost helpers, silent bots, no progress. Everything is logged to a JSONL file.
//
// The rule engine (rules.go) is pure: Observation in, Decisions out, clock injected. The real world
// (observe_linux.go, act.go) sits behind the Observer and Actor interfaces so the rules are table-tested.
package sentinel

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Observation is everything the sentinel knows about one instance at one moment. Zero values mean
// "not observed"; the *OK flags say whether the group was read at all.
type Observation struct {
	Time     time.Time `json:"time"`
	Instance string    `json:"instance"`

	// Process.
	ProcAlive bool    `json:"procAlive"`
	PID       int     `json:"pid,omitempty"`
	ProcState string  `json:"procState,omitempty"` // /proc stat state letter: R S D T Z ...
	CPUTicks  uint64  `json:"cpuTicks,omitempty"`  // utime+stime, monotonic while the game runs
	CPUPct    float64 `json:"cpuPct,omitempty"`

	// Window frame (hash of the captured game window; "" when no capture).
	FrameOK   bool   `json:"frameOK"`
	FrameHash string `json:"frameHash,omitempty"`

	// Game memory.
	MemOK    bool   `json:"memOK"`
	MemErr   string `json:"memErr,omitempty"`
	State    string `json:"state,omitempty"` // menu | ingame | loading
	Mode     int    `json:"mode"`
	Life     int    `json:"life"`
	MaxLife  int    `json:"maxLife"`
	X        int    `json:"x"`
	Y        int    `json:"y"`
	Area     int    `json:"area"`
	Level    int    `json:"level"`
	Exp      uint32 `json:"exp"`
	Gold     int    `json:"gold"`
	InTown   bool   `json:"inTown"`
	EscMenu  bool   `json:"escMenu"`
	Panels   []int  `json:"panels,omitempty"` // open UI flag indexes other than the Esc menu
	MenuScr  string `json:"menuScreen,omitempty"`
	CharName string `json:"char,omitempty"`

	// Input helper.
	HelperUp     bool `json:"helperUp"`
	HelperWindow bool `json:"helperWindow"`
	Hung         bool `json:"hung"` // IsHungAppWindow via the helper's `hung`

	// Dialogs: titles of unexpected windows on the instance's own display, and crash-reporter processes.
	Dialogs []string `json:"dialogs,omitempty"`

	// LockHeld: another process (tools/live.sh run, a bot) holds /tmp/d2lod-game-<id>.lock right now.
	LockHeld bool `json:"lockHeld"`

	// Bot health (nil when no bot address is configured for the instance).
	Bot *BotHealth `json:"bot,omitempty"`
}

// Dead reports the death overlay state: the player is in the dead/death mode or has no life.
func (o *Observation) Dead() bool {
	return o.State == "ingame" && (o.Life <= 0 || o.Mode == 0 || o.Mode == 17)
}

// BotHealth is the bot's own view, decoded tolerantly from /api/v1/bot/health or similar (field names vary).
type BotHealth struct {
	Reachable     bool          `json:"reachable"`
	Err           string        `json:"err,omitempty"`
	State         string        `json:"state,omitempty"`
	TimeInState   time.Duration `json:"timeInState,omitempty"`
	SinceProgress time.Duration `json:"sinceProgress,omitempty"` // time since the bot's last progress; -1 unknown
	StuckReason   string        `json:"stuckReason,omitempty"`
	GoalActive    bool          `json:"goalActive"`
	Goal          string        `json:"goal,omitempty"`
}

// ParseBotHealth decodes a health JSON body, accepting several spellings. Unknown shapes still yield
// Reachable=true with whatever was understood.
func ParseBotHealth(b []byte, now time.Time) BotHealth {
	h := BotHealth{Reachable: true, SinceProgress: -1}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		return h
	}
	// Unwrap {"data": {...}} / {"health": {...}}.
	for _, k := range []string{"data", "health", "status"} {
		if sub, ok := m[k].(map[string]any); ok {
			m = sub
			break
		}
	}
	str := func(keys ...string) string {
		for _, k := range keys {
			if v, ok := m[k]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
		}
		return ""
	}
	// dur reads seconds (number), a Go duration string, or an RFC3339/unix timestamp (age = now - t).
	dur := func(keys ...string) (time.Duration, bool) {
		for _, k := range keys {
			v, ok := m[k]
			if !ok || v == nil {
				continue
			}
			switch t := v.(type) {
			case float64:
				if strings.HasSuffix(k, "Ms") || strings.HasSuffix(k, "_ms") {
					return time.Duration(t * float64(time.Millisecond)), true
				}
				if strings.Contains(k, "unix") || strings.HasSuffix(k, "At") || strings.HasSuffix(k, "_at") || strings.HasSuffix(k, "Time") {
					if t > 1e9 {
						return now.Sub(time.Unix(int64(t), 0)), true
					}
				}
				return time.Duration(t * float64(time.Second)), true
			case string:
				if d, err := time.ParseDuration(t); err == nil {
					return d, true
				}
				if ts, err := time.Parse(time.RFC3339, t); err == nil {
					return now.Sub(ts), true
				}
			}
		}
		return 0, false
	}
	h.State = str("state", "State")
	h.StuckReason = str("stuckReason", "stuck_reason", "stuck", "StuckReason")
	h.Goal = str("goal", "Goal", "activeGoal")
	if d, ok := dur("timeInState", "time_in_state", "timeInStateSec", "time_in_state_sec", "stateSec", "stateAge", "sinceStateChange"); ok {
		h.TimeInState = d
	}
	if d, ok := dur("sinceProgress", "since_progress", "lastProgressAge", "last_progress_age", "lastProgressSec", "last_progress_sec", "lastProgress", "last_progress", "lastProgressAt", "last_progress_at", "lastProgressUnix", "last_progress_unix"); ok {
		h.SinceProgress = d
	}
	for _, k := range []string{"goalActive", "goal_active", "active"} {
		if v, ok := m[k].(bool); ok {
			h.GoalActive = v
		}
	}
	if !h.GoalActive && h.Goal != "" {
		h.GoalActive = true
	}
	return h
}

// Action names the repairs the sentinel can perform.
type Action string

const (
	ActEsc          Action = "esc"            // Esc until panels and the Esc menu are closed
	ActRespawn      Action = "respawn"        // Esc on the death overlay, verify life and town
	ActOOGStart     Action = "oog-start"      // launch.Flow.Start for the instance's character
	ActDismiss      Action = "dismiss-dialog" // key Return to the dialog window on the instance's display
	ActCont         Action = "sigcont"        // SIGCONT a stopped game
	ActRestartGame  Action = "restart-game"   // kill, game.sh start, helper, mute, OOG start
	ActRestartHelp  Action = "restart-helper" // game.sh helper
	ActRestartBot   Action = "restart-bot"    // configured command
	ActBotReset     Action = "bot-reset"      // ask the bot to go to town (HTTP)
	ActLeaveRejoin  Action = "leave-rejoin"   // Save and Exit, then OOG start
	ActAlert        Action = "alert"          // nothing to do automatically
	ActStopInstance Action = "stop-instance"  // Save and Exit if in game, then game.sh stop (idle reaper)
	ActEsc3         Action = "esc3"           // three Esc presses (no-progress step 1)
)

// Result is what an action reports.
type Result struct {
	OK       bool          `json:"ok"`
	Verified bool          `json:"verified"` // the post-condition was read back from memory
	Detail   string        `json:"detail,omitempty"`
	Took     time.Duration `json:"took"`
}

func (r Result) String() string {
	return fmt.Sprintf("ok=%v verified=%v %s", r.OK, r.Verified, r.Detail)
}
