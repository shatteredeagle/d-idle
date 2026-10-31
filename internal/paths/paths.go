// Package paths is the one place that knows where d2lod-bot keeps its files (findings F16, design-gaps
// D4/D11, PLAN constraint 2: no hard-coded $HOME or host paths). Only legacy.go reads or moves files.
//
// Root resolution (FromEnv): $D2BOT_DATA, else $XDG_DATA_HOME/d2bot, else ~/.local/share/d2bot.
// The config file is $D2BOT_CONFIG, else <Root>/config.json. Runtime files (pids, sockets, locks) live in
// $XDG_RUNTIME_DIR/d2bot when that is set, else <Root>/run, so they work in containers without a user session.
//
//	What                       Path (Root = data dir)
//	config                     $D2BOT_CONFIG or <Root>/config.json
//	instance dir               <Root>/instances/<id>/
//	  runtime layer            <Root>/instances/<id>/runtime.json
//	  goals / deaths / routes  <Root>/instances/<id>/{goals,deaths,routes}.json
//	  state dir                <Root>/instances/<id>/state/
//	  save-file backups        <Root>/instances/<id>/saves-backup/   (snapshot rotation, internal/saves)
//	  logs                     <Root>/instances/<id>/logs/
//	  fixtures                 <Root>/instances/<id>/fixtures/
//	  live character saves     <Root>/instances/<id>/saves/          (U: overridable, the real folder is in the prefix)
//	extracted data tables      <Root>/cache/data/<schema>-<stamp>/
//	wine prefix                <Root>/prefixes/<id>/                 (never on NTFS)
//	shared game dir            <Root>/game/                          (read-only, hardlink source)
//	cheat ledger               <Root>/ledger/cheats.jsonl
//	global logs                <Root>/logs/
//	runtime dir                see above; <runtime>/lock-<id> (game lock), <runtime>/<id>.pid, <runtime>/<id>.sock
//
// Legacy locations (read as fallback, moved on first start; see legacy.go): ~/.local/state/d2lod/{cheats,sentinel}.jsonl,
// /tmp/d2lod-game[-<id>].lock (still held together with the new lock), ~/Games/d2lod[-inst/<id>] (kept in place),
// $D2_COLD_DIR/d2lod/backups (kept; old ledger entries point there), ~/.cache/d2lod-cheat-data (data tables).
// Also: sentinel log <Root>/logs/sentinel.jsonl, cheat backups <Root>/ledger/backups/.
//
// Instance ids are validated (Instance returns an error for anything but [A-Za-z0-9._-], no leading dot) so
// they can never escape the tree.
package paths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Environment variables read by FromEnv.
const (
	EnvData   = "D2BOT_DATA"
	EnvConfig = "D2BOT_CONFIG"
)

// Layout is the data-directory layout. The zero value is not usable; set Root (or use FromEnv).
type Layout struct {
	Root       string // data dir
	ConfigFile string // overrides <Root>/config.json when set
	RuntimeDir string // overrides the runtime dir when set
	Home       string // home dir for legacy locations (tests); empty = os.UserHomeDir
}

// FromEnv resolves the layout from the environment (see package doc). It fails only if no home
// directory can be determined and nothing is set.
func FromEnv() (Layout, error) { return fromEnv(os.Getenv, os.UserHomeDir) }

func fromEnv(getenv func(string) string, home func() (string, error)) (Layout, error) {
	l := Layout{ConfigFile: getenv(EnvConfig)}
	switch {
	case getenv(EnvData) != "":
		l.Root = getenv(EnvData)
	case getenv("XDG_DATA_HOME") != "":
		l.Root = filepath.Join(getenv("XDG_DATA_HOME"), "d2bot")
	default:
		h, err := home()
		if err != nil {
			return Layout{}, fmt.Errorf("paths: no %s, no XDG_DATA_HOME and no home dir: %w", EnvData, err)
		}
		l.Root = filepath.Join(h, ".local", "share", "d2bot")
	}
	if x := getenv("XDG_RUNTIME_DIR"); x != "" {
		l.RuntimeDir = filepath.Join(x, "d2bot")
	}
	return l, nil
}

func (l Layout) Config() string {
	if l.ConfigFile != "" {
		return l.ConfigFile
	}
	return filepath.Join(l.Root, "config.json")
}
func (l Layout) Run() string {
	if l.RuntimeDir != "" {
		return l.RuntimeDir
	}
	return filepath.Join(l.Root, "run")
}
func (l Layout) InstancesDir() string { return filepath.Join(l.Root, "instances") }
func (l Layout) CacheData() string    { return filepath.Join(l.Root, "cache", "data") }

// DataTables is the extraction directory for one data schema and source stamp.
func (l Layout) DataTables(schema, stamp string) string {
	return filepath.Join(l.CacheData(), schema+"-"+stamp)
}
func (l Layout) GameDir() string     { return filepath.Join(l.Root, "game") }
func (l Layout) Logs() string        { return filepath.Join(l.Root, "logs") }
func (l Layout) CheatLedger() string { return filepath.Join(l.Root, "ledger", "cheats.jsonl") }
func (l Layout) PrefixesDir() string { return filepath.Join(l.Root, "prefixes") }

var idRe = regexp.MustCompile(`^[A-Za-z0-9_][A-Za-z0-9._-]*$`)

// ErrBadInstance is returned for an instance id that is not a plain name.
var ErrBadInstance = errors.New("paths: bad instance id")

// Instance returns the per-instance paths. The id is "i0", "i1", ... or any plain name.
func (l Layout) Instance(id string) (Instance, error) {
	if !idRe.MatchString(id) || len(id) > 64 {
		return Instance{}, fmt.Errorf("%w: %q", ErrBadInstance, id)
	}
	return Instance{ID: id, l: l}, nil
}

// MustInstance is Instance for ids known to be valid (constants, tests); it panics otherwise.
func (l Layout) MustInstance(id string) Instance {
	i, err := l.Instance(id)
	if err != nil {
		panic(err)
	}
	return i
}

// Instance is the path set of one game instance.
type Instance struct {
	ID string
	l  Layout

	// SaveFolder overrides SaveDir (the real character folder lives inside the Wine prefix, see
	// internal/launch Config.SaveDir). U: the default below is only a placeholder for tests and fresh setups.
	SaveFolder string
}

// WithSaveDir returns a copy whose SaveDir is dir.
func (i Instance) WithSaveDir(dir string) Instance { i.SaveFolder = dir; return i }

func (i Instance) Dir() string { return filepath.Join(i.l.InstancesDir(), i.ID) }
func (i Instance) join(p ...string) string {
	return filepath.Join(append([]string{i.Dir()}, p...)...)
}
func (i Instance) Runtime() string  { return i.join("runtime.json") }
func (i Instance) Goals() string    { return i.join("goals.json") }
func (i Instance) Deaths() string   { return i.join("deaths.json") }
func (i Instance) Routes() string   { return i.join("routes.json") }
func (i Instance) State() string    { return i.join("state") }
func (i Instance) Logs() string     { return i.join("logs") }
func (i Instance) Fixtures() string { return i.join("fixtures") }
func (i Instance) Prefix() string   { return filepath.Join(i.l.PrefixesDir(), i.ID) }

// GameLock is the per-instance lock file serialising runs that drive the game (was /tmp/d2lod-game-<id>.lock).
func (i Instance) GameLock() string { return filepath.Join(i.l.Run(), "game-"+i.ID+".lock") }
func (i Instance) Pid() string      { return filepath.Join(i.l.Run(), i.ID+".pid") }
func (i Instance) Socket() string   { return filepath.Join(i.l.Run(), i.ID+".sock") }

// SaveDir and BackupDir satisfy internal/saves.Paths (structurally; no import in either direction).
func (i Instance) SaveDir() string {
	if i.SaveFolder != "" {
		return i.SaveFolder
	}
	return i.join("saves")
}
func (i Instance) BackupDir() string { return i.join("saves-backup") }

// MkdirAll creates the instance directories and the runtime dir (0o700 for the runtime dir).
func (i Instance) MkdirAll() error {
	for _, d := range []string{i.State(), i.Logs(), i.Fixtures(), i.BackupDir()} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}
	return os.MkdirAll(i.l.Run(), 0o700)
}
