// Package launch owns everything between "the user has installers" and "a game window is up":
// Proton runner discovery, prefix/install/patch steps, launching the game and the input helper as
// children of this process (child subreaper, so /proc/<pid>/mem is readable with no capability),
// process health supervision, and the out-of-game menu flow.
//
// Linux only, standard library only.
package launch

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"

	"d2lod-bot/internal/paths"
)

// Config is everything the launcher needs. Zero fields are filled by Resolve.
type Config struct {
	// Root is the app's data directory for the Wine side (prefix, helper, logs). Default
	// paths.Layout.GameRoot("i0"): <data>/prefixes/i0, or the dev-era ~/Games/d2lod while that still exists
	// (env D2_ROOT overrides, same as tools/d2-prefix.sh).
	Root string
	// PrefixDir is STEAM_COMPAT_DATA_PATH; the Wine prefix is PrefixDir/pfx. Default Root/proton.
	PrefixDir string
	// GameDir is the host path of the Diablo II install. Default PrefixDir/pfx/drive_c/Program Files (x86)/Diablo II.
	GameDir string
	// SteamRoot is passed as STEAM_COMPAT_CLIENT_INSTALL_PATH. Proton only needs the directory to exist.
	// Default ~/.local/share/Steam.
	SteamRoot string
	// CompatToolDirs are searched for Proton runners. Default: SteamRoot/compatibilitytools.d,
	// ~/.steam/root/compatibilitytools.d, /usr/share/steam/compatibilitytools.d.
	CompatToolDirs []string
	// RunnerName pins a runner directory name (default "GE-Proton11-11", the VERIFIED one). If it is not
	// installed the newest GE-Proton found is used. Env PROTON (path to the proton script) overrides both.
	RunnerName string
	// ProtonScript is the resolved path of the `proton` script (set by Resolve).
	ProtonScript string
	// HelperExe is the host path of the Windows build of cmd/d2input. Default Root/helper/d2input.exe.
	HelperExe string
	// HelperAddr is the helper's TCP address. Default 127.0.0.1:47012.
	HelperAddr string
	// LogDir receives PROTON_LOG output and child stdout/stderr. Default Root/logs.
	LogDir string
	// GameArgs are passed to Diablo II.exe. Default ["-w"] (windowed, VERIFIED) plus "-ns" when Mute.
	GameArgs []string
	// Mute starts the game without sound (`-ns`). Defaults to true (see MuteOff); every game window we
	// launch must start muted. Also run tools/mute-d2.sh after a launch to mute the PipeWire stream.
	// A Go bool cannot default to true, so the zero value means muted and MuteOff disables it.
	MuteOff bool
	// Instance is the bot instance id ("i1"...). When set, every child gets D2BOT_INSTANCE=<id> (Wine
	// processes inherit it, so mem.FindGamesFor can tell the games apart) and Find only sees this
	// instance's processes. Empty: the original single game. See instance.go.
	Instance string
	// Sandbox switches and extends the mount allowlist the Wine processes run in (sandbox.go, docs/sandbox.md).
	Sandbox SandboxConfig
	// ExtraEnv is appended to the environment of every Proton child (KEY=VALUE).
	ExtraEnv []string
}

// Defaults returns a Config with environment overrides applied but nothing resolved or checked.
func Defaults() Config {
	c := Config{Root: os.Getenv("D2_ROOT")}
	if c.Root == "" {
		c.Root = layout().GameRoot("i0")
	}
	return c
}

// Paths returns the data-dir paths of the configured instance (i0 when Instance is empty) with SaveDir
// pointing at this Config's real character folder inside the prefix. ok is false for an invalid id.
func (c Config) Paths(l paths.Layout) (paths.Instance, bool) {
	id := c.Instance
	if id == "" {
		id = "i0"
	}
	ip, err := l.Instance(id)
	return ip.WithSaveDir(c.SaveDir()), err == nil
}

// Resolve fills defaults and finds the Proton script. It does not create anything on disk.
func (c *Config) Resolve() error {
	home, _ := os.UserHomeDir()
	if c.Root == "" {
		c.Root = Defaults().Root
	}
	if c.PrefixDir == "" {
		c.PrefixDir = filepath.Join(c.Root, "proton")
	}
	if c.GameDir == "" {
		c.GameDir = filepath.Join(c.PrefixDir, "pfx", "drive_c", "Program Files (x86)", "Diablo II")
	}
	if c.SteamRoot == "" {
		c.SteamRoot = filepath.Join(home, ".local", "share", "Steam")
	}
	if len(c.CompatToolDirs) == 0 {
		c.CompatToolDirs = []string{
			filepath.Join(c.SteamRoot, "compatibilitytools.d"),
			filepath.Join(home, ".steam", "root", "compatibilitytools.d"),
			"/usr/share/steam/compatibilitytools.d",
		}
	}
	if c.RunnerName == "" {
		c.RunnerName = "GE-Proton11-11"
	}
	if c.HelperExe == "" {
		c.HelperExe = os.Getenv("D2INPUT_EXE") // test builds of the helper, without replacing the shared exe
	}
	if c.HelperExe == "" {
		c.HelperExe = filepath.Join(c.Root, "helper", "d2input.exe")
	}
	if c.HelperAddr == "" {
		c.HelperAddr = "127.0.0.1:47012"
	}
	if c.LogDir == "" {
		c.LogDir = filepath.Join(c.Root, "logs")
	}
	if c.GameArgs == nil {
		c.GameArgs = []string{"-w"}
		if !c.MuteOff {
			c.GameArgs = append(c.GameArgs, "-ns")
		}
	}
	if c.ProtonScript == "" {
		if p := os.Getenv("PROTON"); p != "" {
			c.ProtonScript = p
		} else {
			r, err := PickRunner(FindRunners(c.CompatToolDirs), c.RunnerName)
			if err != nil {
				return err
			}
			c.ProtonScript = r.Script
		}
	}
	if _, err := os.Stat(c.ProtonScript); err != nil {
		return fmt.Errorf("proton script: %w", err)
	}
	return nil
}

// WinPath maps a host path to a Windows path through the Z: drive (Wine maps / to Z:).
func WinPath(host string) string {
	abs, err := filepath.Abs(host)
	if err != nil {
		abs = host
	}
	out := []byte("Z:" + abs)
	for i := range out {
		if out[i] == '/' {
			out[i] = '\\'
		}
	}
	return string(out)
}

// GameExe returns the host path of Diablo II.exe (the launcher stub that execs Game.exe).
func (c Config) GameExe() string { return filepath.Join(c.GameDir, "Diablo II.exe") }

// RealGameExe returns the host path of Game.exe, the process that holds the memory we read.
func (c Config) RealGameExe() string { return filepath.Join(c.GameDir, "Game.exe") }

// Runner is one installed Proton build.
type Runner struct {
	Name    string // directory name, e.g. GE-Proton11-11
	Dir     string
	Script  string // Dir/proton
	Version [2]int // parsed from GE-ProtonM-N, zero if unparseable
}

var geRe = regexp.MustCompile(`^GE-Proton(\d+)-(\d+)`)

// FindRunners lists directories under dirs that contain a `proton` script, newest GE-Proton first.
// Names that do not parse (e.g. "Proton-CachyOS Latest") sort after the parseable ones.
func FindRunners(dirs []string) []Runner {
	var out []Runner
	seen := map[string]bool{}
	for _, d := range dirs {
		ents, err := os.ReadDir(d)
		if err != nil {
			continue
		}
		for _, e := range ents {
			dir := filepath.Join(d, e.Name())
			script := filepath.Join(dir, "proton")
			if st, err := os.Stat(script); err != nil || st.IsDir() {
				continue
			}
			real, err := filepath.EvalSymlinks(dir)
			if err != nil || seen[real] {
				continue
			}
			seen[real] = true
			r := Runner{Name: e.Name(), Dir: dir, Script: script}
			if m := geRe.FindStringSubmatch(e.Name()); m != nil {
				r.Version[0], _ = strconv.Atoi(m[1])
				r.Version[1], _ = strconv.Atoi(m[2])
			}
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].Version, out[j].Version
		if a != b {
			return a[0] > b[0] || a[0] == b[0] && a[1] > b[1]
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// PickRunner returns the runner called want, else the newest GE-Proton.
func PickRunner(rs []Runner, want string) (Runner, error) {
	for _, r := range rs {
		if r.Name == want {
			return r, nil
		}
	}
	for _, r := range rs {
		if r.Version != [2]int{} {
			return r, nil
		}
	}
	return Runner{}, errors.New("no GE-Proton runner found in compatibilitytools.d (install one, e.g. with ProtonUp-Qt)")
}
