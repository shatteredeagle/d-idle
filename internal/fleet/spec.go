// Package fleet is the supervisor of one bot HOST (a container or a machine): N instance slots that are
// added and removed at runtime through an app call, never through Docker (deploy/container/README.md
// Decision, PLAN.md item 7, docs/archive/review/design-gaps.md D8). Each slot is one game instance with its own
// Wine prefix, Xvfb display and d2input helper port (docs/multi.md). The package decides WHETHER and WHERE
// an instance may run (admission, allocation, leases) and keeps it running (supervised restart); the game
// mechanics sit behind Runtime, implemented over launch.Life (LifeRuntime) and faked in tests.
package fleet

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"d2lod-bot/internal/bot"
	"d2lod-bot/internal/goals"
	"d2lod-bot/internal/party"
)

// Spec is what the caller asks for.
type Spec struct {
	ID         string     `json:"id,omitempty"` // "i1".."i9"; empty = allocate the lowest free one
	Character  string     `json:"character"`
	Class      string     `json:"class,omitempty"`   // used only when the character has to be created
	Build      string     `json:"build,omitempty"`   // build/skill plan key for the bot (free text here)
	Profile    string     `json:"profile,omitempty"` // handicap profile name (bot.ProfileByName), "" = optimal
	Goals      string     `json:"goals,omitempty"`   // goal presets "campaign:T,levelto:N,farm:AREA" (GoalsArg)
	Role       party.Role `json:"role,omitempty"`    // default solo
	Difficulty string     `json:"difficulty,omitempty"`
	Hardcore   bool       `json:"hardcore,omitempty"`
}

// Alloc is the host resources derived from the slot number (V docs/multi.md: helper 47012+N, display :10N).
type Alloc struct {
	ID         string `json:"id"`
	Num        int    `json:"num"`
	HelperPort int    `json:"helperPort"`
	Display    string `json:"display"`
	Prefix     string `json:"prefix"` // Wine prefix directory (own wineserver; the game dir is shared)
}

// AllocFor computes the allocation of slot number n (1..9). prefix maps an id to its prefix directory.
func AllocFor(n int, prefix func(string) string) Alloc {
	id := fmt.Sprintf("i%d", n)
	a := Alloc{ID: id, Num: n, HelperPort: 47012 + n, Display: fmt.Sprintf(":%d", 100+n)}
	if prefix != nil {
		a.Prefix = prefix(id)
	}
	return a
}

var (
	idRe    = regexp.MustCompile(`^i[1-9]$`)
	charRe  = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9]{1,14}$`)
	classes = map[string]bool{"amazon": true, "sorceress": true, "necromancer": true, "paladin": true, "barbarian": true, "druid": true, "assassin": true}
	diffs   = map[string]bool{"normal": true, "nightmare": true, "hell": true}
)

// Validate normalises the spec (role, class case) and reports the first problem.
func (s *Spec) Validate() error {
	if !charRe.MatchString(s.Character) {
		return errors.New("character must be 2 to 15 letters or digits, starting with a letter")
	}
	if s.ID != "" && !idRe.MatchString(s.ID) {
		return fmt.Errorf("bad slot id %q (i1 to i9; i0 is the desktop game)", s.ID)
	}
	s.Class = strings.ToLower(s.Class)
	if s.Class != "" && !classes[s.Class] {
		return fmt.Errorf("unknown class %q", s.Class)
	}
	if _, err := bot.ProfileByName(s.Profile); err != nil {
		return err
	}
	if _, err := s.GoalsArg(); err != nil {
		return err
	}
	r, err := party.Parse(string(s.Role))
	if err != nil {
		return err
	}
	s.Role = r
	if s.Difficulty != "" && !diffs[s.Difficulty] {
		return fmt.Errorf("unknown difficulty %q", s.Difficulty)
	}
	return nil
}

type State string

const (
	Starting   State = "starting"
	Running    State = "running"
	Restarting State = "restarting" // game died, waiting out the backoff
	Draining   State = "draining"   // being removed
	Failed     State = "failed"     // gave up after MaxRestarts; Remove still works, Restart revives
)

// View is the public state of one slot.
type View struct {
	ID        string     `json:"id"`
	Spec      Spec       `json:"spec"`
	Alloc     Alloc      `json:"alloc"`
	State     State      `json:"state"`
	Restarts  int        `json:"restarts"`
	LastError string     `json:"lastError,omitempty"`
	Since     time.Time  `json:"since"`
	LeasedBy  string     `json:"leasedBy,omitempty"`
	Bot       *BotStatus `json:"bot,omitempty"` // the slot's d2bot process (nil without Config.Bots)
}

// HostStatus is the whole host: capacity checks plus all slots.
type HostStatus struct {
	Capacity Capacity `json:"capacity"`
	Slots    []View   `json:"slots"`
}

var areaRe = regexp.MustCompile(`^[A-Za-z0-9_.-]{1,40}$`)

// GoalsArg translates the goal presets of the wizard (comma separated "campaign:TEMPLATE", "levelto:N",
// "farm:AREA") into the value of d2bot --goals (";" separated, cmd/d2bot enqueueGoals): campaign:T stays,
// levelto:N becomes reach:N, farm:AREA becomes farm:AREA with "_" as a space. Empty Goals gives "" (d2bot then
// runs its saved goals). The farm area name is not checked against the area table here (U).
func (s Spec) GoalsArg() (string, error) {
	var out []string
	for _, part := range strings.Split(s.Goals, ",") {
		if part = strings.TrimSpace(part); part == "" {
			continue
		}
		kind, val, _ := strings.Cut(part, ":")
		switch kind {
		case "campaign":
			if !slices.Contains(goals.CampaignTemplates, val) {
				return "", fmt.Errorf("goals: unknown campaign template %q (have %s)", val, strings.Join(goals.CampaignTemplates, ", "))
			}
			out = append(out, "campaign:"+val)
		case "levelto":
			n, err := strconv.Atoi(val)
			if err != nil || n < 2 || n > 99 {
				return "", fmt.Errorf("goals: levelto needs a level from 2 to 99, got %q", val)
			}
			out = append(out, "reach:"+val)
		case "farm":
			if !areaRe.MatchString(val) {
				return "", fmt.Errorf("goals: farm area %q: letters, digits and _ . - only", val)
			}
			out = append(out, "farm:"+strings.ReplaceAll(val, "_", " "))
		default:
			return "", fmt.Errorf("goals: unknown preset %q (use campaign:T, levelto:N, farm:AREA)", part)
		}
	}
	return strings.Join(out, ";"), nil
}

// BotArgs are the d2bot flags this spec adds (--goals, and --autostart with it). The handicap profile is a
// config key of d2bot, not a flag (bot.Config.Profile), so it is not here.
func (s Spec) BotArgs() ([]string, error) {
	g, err := s.GoalsArg()
	if err != nil || g == "" {
		return nil, err
	}
	return []string{"--goals", g, "--autostart"}, nil
}
