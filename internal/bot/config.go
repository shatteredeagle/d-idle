package bot

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"d2lod-bot/internal/env"
)

// Config is the bot core's configuration (the "bot" section of D2BOT_CONFIG, see docs/bot.md). All durations are
// seconds so the file stays plain JSON. Zero values take the defaults of WithDefaults.
type Config struct {
	// Profile names a built-in handicap profile (Profiles); "" is optimal.
	Profile string `json:"profile,omitempty"`
	// Custom defines an ad-hoc profile instead of a built-in one.
	Custom *Profile `json:"custom,omitempty"`
	// Tuning overrides single knobs by key after the profile ("bot.clearRadius": 20). Values are clamped to the knob range.
	Tuning map[string]float64 `json:"tuning,omitempty"`
	// PlanDir overrides the embedded plan data (areas.json, runs.json, progression-*.json), file by file.
	PlanDir string `json:"planDir,omitempty"`
	// Acts the bot may travel in (1-based). Act changes need quests, which the bot cannot do yet, so the default is [1].
	// The act of the town the character stands in is always allowed.
	Acts []int `json:"acts,omitempty"`
	// Autostart starts playing the saved goals at startup (otherwise the bot idles until POST /control/start). Same as --autostart.
	Autostart bool `json:"autostart,omitempty"`
	// Hardcore disables corpse retrieval; hardcore death handling is not designed yet (PLAN.md: later, as a prestige mechanic).
	Hardcore bool `json:"hardcore,omitempty"`

	StuckAfterSec       int `json:"stuckAfterSec,omitempty"`       // no progress for this long = stuck (default 120)
	TravelTimeoutSec    int `json:"travelTimeoutSec,omitempty"`    // one travel action (default 300)
	SweepTimeoutSec     int `json:"sweepTimeoutSec,omitempty"`     // one clear/explore action (default 240)
	TownTimeoutSec      int `json:"townTimeoutSec,omitempty"`      // one town action (default 180)
	StopTimeoutSec      int `json:"stopTimeoutSec,omitempty"`      // finishing in town on Stop (default 60)
	RecoverTimeoutSec   int `json:"recoverTimeoutSec,omitempty"`   // waiting for the respawn after death (default 60)
	MenuGraceSec        int `json:"menuGraceSec,omitempty"`        // in the menus this long = left the game (default 8)
	ResupplyCooldownSec int `json:"resupplyCooldownSec,omitempty"` // minimum gap between non-emergency town trips (default 240)
	ErrorRetrySec       int `json:"errorRetrySec,omitempty"`       // leave the Error state and retry after this long (default 300)
	GlobalStuckSec      int `json:"globalStuckSec,omitempty"`      // no progress at all, whatever the state, for this long (default 600)
	// ClearIdleSec: in the clearing state, no progress at all (no step, kill, damage, item, xp) for this long is a stall,
	// caught well before StuckAfterSec because a fight or an exploration always produces one of those (default 60; an
	// explicit StateStuckSec["clearing"] below it wins). Docs: docs/stallfatal.md.
	ClearIdleSec int `json:"clearIdleSec,omitempty"`
	// MaxFailsafeRestarts bounds game restarts by the failsafe ladder without real progress in between (default 3);
	// after that the bot pauses itself visibly instead of ending the run.
	MaxFailsafeRestarts int `json:"maxFailsafeRestarts,omitempty"`
	// StateStuckSec overrides StuckAfterSec per state name (recovering, resupplying, traveling, clearing, in_town).
	StateStuckSec   map[string]int `json:"stateStuckSec,omitempty"`
	MaxStepFailures int            `json:"maxStepFailures,omitempty"` // consecutive failed actions before escalating (default 5)
	// EmptyAreaSec: a sweep that has not killed anything for this long in a leveling/recovery area (not a quest area)
	// calls the area cleared, so the planner picks the next area or starts a fresh game (monsters respawn only there).
	// 0 = off, the default everywhere: 300 s never fired in 8 d2d runs and 150 s fired 16 times without a kill gain (docs/killfix.md).
	EmptyAreaSec int `json:"emptyAreaSec,omitempty"`
	// MouthGuard (d2d only): combat does not walk to points inside a cave mouth's trigger area (nav.NearMouth). Off by default.
	MouthGuard   bool `json:"mouthGuard,omitempty"`
	DangerBanSec int  `json:"dangerBanSec,omitempty"` // how long the spiral breaker keeps the bot out of an area (0 = the bot.dangerBanMin knob, default 20 min)

	// DeathLogPath persists the death log (per-area and per-monster counters, heat map). Empty keeps it in memory.
	DeathLogPath string `json:"deathLogPath,omitempty"`

	// IgnoreTownAdvice stops the bot from asking Townsman.NeedsTrip (its own potion and inventory checks stay). Use it when the
	// Townsman misjudges, for example a brand-new character that never has the mana potions it wants.
	IgnoreTownAdvice bool `json:"ignoreTownAdvice,omitempty"`

	// LevelerBuild selects the stat/skill table: "fallback" (Lightning, the default, no respec needed) or "primary" (Blizzard,
	// assumes a respec at clvl 26 the bot cannot do yet). NoLeveler leaves points and gear alone (docs/leveler.md).
	LevelerBuild string `json:"levelerBuild,omitempty"`
	NoLeveler    bool   `json:"noLeveler,omitempty"`

	// DisableObserve names event kinds ("kill", "drop", "potion") the bot must not derive from snapshots because a
	// behavior package already emits them through env.Emit (avoids double counting).
	DisableObserve []string `json:"disableObserve,omitempty"`
}

// WithDefaults returns c with zero fields filled.
func (c Config) WithDefaults() Config {
	def := func(v *int, d int) {
		if *v <= 0 {
			*v = d
		}
	}
	def(&c.StuckAfterSec, 120)
	def(&c.TravelTimeoutSec, 300)
	def(&c.SweepTimeoutSec, 240)
	def(&c.TownTimeoutSec, 180)
	def(&c.StopTimeoutSec, 60)
	def(&c.RecoverTimeoutSec, 60)
	def(&c.MenuGraceSec, 8)
	def(&c.ResupplyCooldownSec, 240)
	def(&c.ErrorRetrySec, 300)
	def(&c.MaxStepFailures, 5)
	def(&c.GlobalStuckSec, 600)
	def(&c.ClearIdleSec, 60)
	def(&c.MaxFailsafeRestarts, 3)
	if c.StateStuckSec == nil {
		c.StateStuckSec = map[string]int{"recovering": 180, "resupplying": 240, "traveling": 240, "in_town": 300}
	}
	if len(c.Acts) == 0 {
		c.Acts = []int{1}
	}
	return c
}

func sec(n int) time.Duration { return time.Duration(n) * time.Second }

func (c Config) observes(kind string) bool {
	for _, k := range c.DisableObserve {
		if strings.EqualFold(k, kind) {
			return false
		}
	}
	return true
}

// ResolveProfile returns the effective profile (Custom wins over Profile).
func (c Config) ResolveProfile() (Profile, error) {
	if c.Custom != nil {
		p := *c.Custom
		if p.Name == "" {
			p.Name = "custom"
		}
		return p, nil
	}
	return ProfileByName(c.Profile)
}

// LoadConfig reads a JSON file. The bot section may be the whole file or the "bot" object of a larger config;
// unknown fields are rejected so typos do not silently fall back to defaults. An empty path or a missing file
// gives the defaults.
func LoadConfig(path string) (Config, error) {
	if path == "" {
		return Config{}.WithDefaults(), nil
	}
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Config{}.WithDefaults(), nil
	}
	if err != nil {
		return Config{}, err
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(b, &top); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	if sect, ok := top["bot"]; ok {
		b = sect
	}
	return ParseConfig(b)
}

// ParseConfig decodes the bot section strictly.
func ParseConfig(b []byte) (Config, error) {
	var c Config
	dec := json.NewDecoder(strings.NewReader(string(b)))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&c); err != nil {
		return Config{}, fmt.Errorf("bot config: %w", err)
	}
	return c.WithDefaults(), nil
}

// BuildTuning makes the Tuning for an instance: defaults, then the profile, then explicit knob overrides.
//
// Knobs whose key does not resolve to a Tuning field (a naming mismatch in some domain) are skipped and returned as
// warnings, so one domain's typo never stops the bot; the knob stays at its default.
func (c Config) BuildTuning() (t env.Tuning, warnings []string, err error) {
	t = env.DefaultTuning()
	p, err := c.ResolveProfile()
	if err != nil {
		return t, nil, err
	}
	warnings = p.Apply(&t)
	return t, warnings, ApplyOverrides(&t, c.Tuning)
}
