package sentinel

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Dur is a time.Duration that reads from JSON as "90s" / "5m" or a number of seconds.
type Dur time.Duration

func (d Dur) D() time.Duration { return time.Duration(d) }

func (d *Dur) UnmarshalJSON(b []byte) error {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	switch t := v.(type) {
	case float64:
		*d = Dur(time.Duration(t * float64(time.Second)))
	case string:
		x, err := time.ParseDuration(t)
		if err != nil {
			return err
		}
		*d = Dur(x)
	default:
		return fmt.Errorf("duration: want string or seconds, got %T", v)
	}
	return nil
}

func (d Dur) MarshalJSON() ([]byte, error) { return json.Marshal(time.Duration(d).String()) }

// Thresholds are the durations a condition must hold before its rule fires. Defaults in DefaultConfig.
type Thresholds struct {
	ProcGone     Dur `json:"procGone"`     // no Game.exe
	HelperGone   Dur `json:"helperGone"`   // helper not listening
	WindowGone   Dur `json:"windowGone"`   // game alive, helper sees no window
	Dialog       Dur `json:"dialog"`       // crash / error dialog visible
	Stopped      Dur `json:"stopped"`      // SIGSTOP / state T
	Hung         Dur `json:"hung"`         // IsHungAppWindow
	FrozenQuiet  Dur `json:"frozenQuiet"`  // frame and CPU both unchanged
	DeathOverlay Dur `json:"deathOverlay"` // dead mode or life 0
	EscMenu      Dur `json:"escMenu"`      // Esc menu open
	PanelOpen    Dur `json:"panelOpen"`    // any other panel open
	OutOfGame    Dur `json:"outOfGame"`    // at a menu screen
	LoadingStuck Dur `json:"loadingStuck"` // loading / ambiguous state
	BotSilent    Dur `json:"botSilent"`    // bot health unreachable
	BotStuck     Dur `json:"botStuck"`     // bot reports a stuck reason
	NoProgress   Dur `json:"noProgress"`   // no xp/gold/area/position change while a goal is active
	StepWait     Dur `json:"stepWait"`     // between steps of the no-progress ladder
	ResetAfter   Dur `json:"resetAfter"`   // condition false this long clears attempts and ladder
	IdleReap     Dur `json:"idleReap"`     // idle instance (no lock, no bot, memory unchanged) is stopped
	Settle       Dur `json:"settle"`       // quiet time after any action on an instance
}

// RuleOverride tunes one rule from the JSON config.
type RuleOverride struct {
	Enabled     *bool `json:"enabled,omitempty"`
	For         *Dur  `json:"for,omitempty"`
	Cooldown    *Dur  `json:"cooldown,omitempty"`
	MaxAttempts *int  `json:"maxAttempts,omitempty"`
}

// InstanceConf describes one watched instance.
type InstanceConf struct {
	ID             string   `json:"id"`
	Agent          string   `json:"agent,omitempty"`
	Character      string   `json:"character,omitempty"` // OOG start target; "" disables oog-start and restarts' login
	Class          string   `json:"class,omitempty"`
	Display        string   `json:"display,omitempty"`     // default :100+N (i0 = the desktop, never touched)
	HelperAddr     string   `json:"helperAddr,omitempty"`  // default 127.0.0.1:47012+N
	BotURL         string   `json:"botUrl,omitempty"`      // health endpoint, e.g. http://127.0.0.1:8081/api/v1/bot/health
	BotResetURL    string   `json:"botResetUrl,omitempty"` // POST: ask the bot to go to town
	BotRestartCmd  []string `json:"botRestartCmd,omitempty"`
	ExpectProgress bool     `json:"expectProgress,omitempty"` // treat the instance as having an active goal even without a bot
	Enabled        *bool    `json:"enabled,omitempty"`
}

// Config is the sentinel's JSON configuration. Every field has a default.
type Config struct {
	Interval   Dur                     `json:"interval"`   // observation period per instance
	FrameEvery Dur                     `json:"frameEvery"` // frame capture period
	LogEvery   Dur                     `json:"logEvery"`   // periodic observation record
	Th         Thresholds              `json:"thresholds"`
	Rules      map[string]RuleOverride `json:"rules,omitempty"`
	Instances  []InstanceConf          `json:"instances"`
	// InstancesConf is the instances.conf whose "#character" lines name each instance's character ("" = the env var
	// D2LOD_INSTANCES_CONF, else DefaultInstancesConf).
	InstancesConf string   `json:"instancesConf,omitempty"`
	Keep          []string `json:"keep,omitempty"` // instances the idle reaper never stops (e.g. i5 during drills)
	GameSh        string   `json:"gameSh"`         // DEPRECATED, ignored: restarts use internal/launch (Life), not game.sh
	MuteSh        string   `json:"muteSh"`         // tools/mute-d2.sh, run after launches
	LogPath       string   `json:"logPath"`
	LockWait      Dur      `json:"lockWait"` // wait for a per-instance game lock before a restart
}

// DefaultCharacters maps instance ids to their throwaway characters (docs/multi.md). It is only the fallback: a
// "#character" directive in tools/instances.conf (CharactersFromConf) overrides it, the JSON config overrides both.
var DefaultCharacters = map[string][2]string{
	"i0": {"Testy", "sorceress"}, "i1": {"Navi", "sorceress"}, "i2": {"Combi", "sorceress"},
	"i3": {"Towni", "sorceress"}, "i4": {"Booti", "sorceress"}, "i5": {"Multi", "sorceress"},
}

// DefaultInstancesConf is where the dev tooling keeps the agent -> instance map (cmd/d2watch uses the same path).
const DefaultInstancesConf = "~/code/d2lod-bot/tools/instances.conf"

// CharactersFromConf reads "#character iN Name class" directives out of an instances.conf. They are comments to every
// other consumer of the file (live.sh, game.sh and sentinel.sh skip lines starting with '#'), so the file stays
// valid for them. A missing file gives an empty map.
func CharactersFromConf(path string) map[string][2]string {
	out := map[string][2]string{}
	if strings.HasPrefix(path, "~/") {
		if h, err := os.UserHomeDir(); err == nil {
			path = filepath.Join(h, path[2:])
		}
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return out
	}
	for _, ln := range strings.Split(string(b), "\n") {
		f := strings.Fields(ln)
		if len(f) >= 4 && f[0] == "#character" {
			out[f[1]] = [2]string{f[2], strings.ToLower(f[3])}
		}
	}
	return out
}

// characterFor resolves the character of an instance: instances.conf directive, else the built-in default.
func (c *Config) characterFor(id string, conf map[string][2]string) ([2]string, bool) {
	if d, ok := conf[id]; ok {
		return d, true
	}
	d, ok := DefaultCharacters[id]
	return d, ok
}

func DefaultConfig() Config {
	s := func(n int) Dur { return Dur(time.Duration(n) * time.Second) }
	return Config{
		Interval: s(2), FrameEvery: s(5), LogEvery: s(30), LockWait: s(20),
		Th: Thresholds{
			ProcGone: s(8), HelperGone: s(10), WindowGone: s(45), Dialog: s(3), Stopped: s(10),
			Hung: s(30), FrozenQuiet: s(90), DeathOverlay: s(10), EscMenu: s(20), PanelOpen: s(90),
			OutOfGame: s(30), LoadingStuck: s(150), BotSilent: s(60), BotStuck: s(60),
			NoProgress: s(480), IdleReap: s(1800), StepWait: s(120), ResetAfter: s(30), Settle: s(5),
		},
		GameSh: "game.sh", MuteSh: "",
	}
}

// LoadConfig reads a JSON file over the defaults ("" = defaults only) and fills per-instance defaults.
func LoadConfig(path string) (Config, error) {
	c := DefaultConfig()
	if path != "" {
		b, err := os.ReadFile(path)
		if err != nil {
			return c, err
		}
		if err := json.Unmarshal(b, &c); err != nil {
			return c, fmt.Errorf("%s: %w", path, err)
		}
	}
	return c, nil
}

// WithInstances adds default entries for ids not already configured (and applies character defaults).
func (c *Config) WithInstances(ids []string) {
	have := map[string]*InstanceConf{}
	for i := range c.Instances {
		have[c.Instances[i].ID] = &c.Instances[i]
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := have[id]; !ok {
			c.Instances = append(c.Instances, InstanceConf{ID: id})
			have[id] = &c.Instances[len(c.Instances)-1]
		}
	}
	conf := c.InstancesConf
	if conf == "" {
		if conf = os.Getenv("D2LOD_INSTANCES_CONF"); conf == "" {
			conf = DefaultInstancesConf
		}
	}
	chars := CharactersFromConf(conf)
	for i := range c.Instances {
		in := &c.Instances[i]
		if d, ok := c.characterFor(in.ID, chars); ok {
			if in.Character == "" {
				in.Character = d[0]
			}
			if in.Class == "" {
				in.Class = d[1]
			}
		}
	}
}

// Num is the numeric part of "iN".
func (i InstanceConf) Num() int {
	var n int
	fmt.Sscanf(strings.TrimPrefix(i.ID, "i"), "%d", &n)
	return n
}

func (i InstanceConf) DisplayName() string {
	if i.Display != "" {
		return i.Display
	}
	if n := i.Num(); n > 0 {
		return fmt.Sprintf(":%d", 100+n)
	}
	return ":1"
}

func (i InstanceConf) Helper() string {
	if i.HelperAddr != "" {
		return i.HelperAddr
	}
	return fmt.Sprintf("127.0.0.1:%d", 47012+i.Num())
}

func (i InstanceConf) On() bool { return i.Enabled == nil || *i.Enabled }
