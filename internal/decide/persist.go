package decide

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// FileVersion is the on-disk format version; a newer file is refused (ErrFutureVersion).
// Version 4: keys follow catalogue version 4; older files (1, 2, 3) are migrated on open (Catalogue.MigrateKey).
const FileVersion = 4

var (
	ErrFutureVersion = errors.New("decide: profile file was written by a newer version")
	ErrCorrupt       = errors.New("decide: corrupt profile file")
)

// fileState is the per-profile file (beside the goals state file). The log is the truth; Rules is a human-readable
// snapshot that load ignores (rules are recomputed by replay).
type fileState struct {
	Version int            `json:"version"`
	Profile string         `json:"profile"`
	Hero    string         `json:"hero"`
	Quality map[string]int `json:"quality,omitempty"`
	Log     []Entry        `json:"log"`
	Rules   []RuleView     `json:"rules"`
}

// Path is the profile file inside dir.
func Path(dir, profile string) string { return filepath.Join(dir, "decide-"+profile+".json") }

// Open loads (or creates) the profile in dir. Every later change is saved atomically.
func Open(dir, profile string, cat *Catalogue, clk Clock) (*Store, error) {
	s, err := New(cat, profile, clk)
	if err != nil {
		return nil, err
	}
	s.path = Path(dir, profile)
	data, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return nil, err
	}
	var raw struct {
		Version int `json:"version"`
	}
	if err := json.Unmarshal(data, &raw); err != nil || raw.Version < 1 {
		return nil, fmt.Errorf("%w: %s", ErrCorrupt, s.path)
	}
	if raw.Version > FileVersion {
		return nil, fmt.Errorf("%w (file v%d, this build v%d)", ErrFutureVersion, raw.Version, FileVersion)
	}
	var fs fileState
	if err := json.Unmarshal(data, &fs); err != nil || fs.Profile != profile {
		return nil, fmt.Errorf("%w: %s", ErrCorrupt, s.path)
	}
	if raw.Version < FileVersion { // an older file: move its keys to the current catalogue shape (see Migration)
		for i := range fs.Log {
			if fs.Log[i].Key != "" {
				fs.Log[i].Key = cat.MigrateKey(fs.Log[i].Key)
			}
		}
	}
	if err := s.replay(fs.Log); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrCorrupt, err)
	}
	if fs.Hero != "" {
		s.hero = fs.Hero
	}
	for k, v := range fs.Quality {
		s.quality[k] = v
	}
	return s, nil
}

func (s *Store) save() error {
	if s.path == "" {
		return nil
	}
	rules := s.rules()
	if rules == nil {
		rules = []RuleView{}
	}
	data, err := json.MarshalIndent(fileState{Version: FileVersion, Profile: s.profile, Hero: s.hero,
		Quality: s.quality, Log: s.log, Rules: rules}, "", " ")
	if err != nil {
		return err
	}
	return writeAtomic(s.path, data)
}

// writeAtomic: temp file, fsync, keep the previous file as .bak, rename (same pattern as internal/goals/store.go).
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	if _, err := f.Write(data); err != nil {
		f.Close()
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, path+".bak"); err != nil {
			return err
		}
	}
	if err := os.Rename(tmp, path); err != nil {
		return err
	}
	if d, err := os.Open(dir); err == nil {
		d.Sync()
		d.Close()
	}
	return nil
}
