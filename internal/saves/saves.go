// Package saves protects the character files of an offline instance and reads progression state out of
// them: integrity check (Verify), rotating snapshots with restore (Snapshot, Restore, List) and the
// quest / waypoint / difficulty / map-id view (Progress).
//
// Tags used here, as everywhere in this repo: V = verified on a real file (named in the comment),
// S = from a published format description or recollection, U = unverified guess. Nothing in this
// package was verified in a running game; "V" only ever means "read from a real .d2s copy".
//
// Only the d2s header, quest, waypoint and stat sections are interpreted (internal/cheat/d2s); the item
// section is never parsed, so a snapshot is a byte copy.
package saves

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"
)

// Paths is the stub for WP-STATE (internal/paths, not built yet): where one instance keeps its save
// folder and where its snapshots go. Replace DirPaths with the paths.Layout adapter when it lands.
type Paths interface {
	SaveDir() string   // folder with <name>.d2s/.key/.map/.ma*
	BackupDir() string // snapshots go to <BackupDir>/<name>/<stamp>/
}

// DirPaths is the trivial Paths.
type DirPaths struct{ Save, Backup string }

func (d DirPaths) SaveDir() string   { return d.Save }
func (d DirPaths) BackupDir() string { return d.Backup }

// GameState tells whether the instance's game is currently inside a game (character file may be
// written at any moment) as opposed to at a menu or not running. Supplied by the caller (mem/env/launch).
type GameState interface {
	InGame() (bool, error)
}

// GameFunc adapts a function to GameState.
type GameFunc func() (bool, error)

func (f GameFunc) InGame() (bool, error) { return f() }

var (
	ErrInGame      = errors.New("saves: the game is in a game; the character file may change while copying")
	ErrNoGameState = errors.New("saves: no GameState given and AtMenu not set; refusing (fail closed)")
	ErrNoFiles     = errors.New("saves: no character files found")
	ErrBadName     = errors.New("saves: bad character name")
)

// Options control Snapshot and Restore.
type Options struct {
	Game   GameState // nil is allowed only with AtMenu
	AtMenu bool      // the caller asserts the game is at a menu or not running
	Keep   int       // newest snapshots kept (default 10)
	Days   int       // plus the newest snapshot of each of the last Days UTC days (default 14)
}

func (o Options) keep() int {
	if o.Keep <= 0 {
		return 10
	}
	return o.Keep
}

func (o Options) days() int {
	if o.Days <= 0 {
		return 14
	}
	return o.Days
}

// gate refuses unless it is safe to touch the character files.
func (o Options) gate() error {
	if o.AtMenu {
		return nil
	}
	if o.Game == nil {
		return ErrNoGameState
	}
	in, err := o.Game.InGame()
	if err != nil {
		return fmt.Errorf("saves: cannot tell whether the game is in a game: %w", err)
	}
	if in {
		return ErrInGame
	}
	return nil
}

func checkName(name string) error {
	if name == "" || name != filepath.Base(name) || strings.ContainsAny(name, `/\`) || strings.HasPrefix(name, ".") {
		return fmt.Errorf("%w: %q", ErrBadName, name)
	}
	return nil
}

// isCharFile reports whether file (a base name) belongs to character name: name.d2s, .key, .map,
// .ma0.. (automap per difficulty; V: Testy and Botty have .d2s .key .ma0 .map).
func isCharFile(file, name string) bool {
	if !strings.HasPrefix(file, name+".") {
		return false
	}
	ext := file[len(name)+1:]
	switch {
	case ext == "d2s" || ext == "key" || ext == "map":
		return true
	case len(ext) == 3 && ext[:2] == "ma" && ext[2] >= '0' && ext[2] <= '9':
		return true
	}
	return false
}
