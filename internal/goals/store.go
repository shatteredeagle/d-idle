package goals

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// FileVersion is the on-disk format version. Older versions are migrated on load; a newer one is refused
// (ErrFutureVersion) so an old binary never overwrites data it cannot understand.
const FileVersion = 1

var ErrFutureVersion = errors.New("goals: state file was written by a newer version")

type fileState struct {
	Version  int       `json:"version"`
	SavedAt  time.Time `json:"savedAt"`
	NextID   int64     `json:"nextId"`
	Mode     Mode      `json:"mode"`
	Goals    []*Goal   `json:"goals"`
	Settings Settings  `json:"settings"`
	Stats    *Stats    `json:"stats"`
	// Locks and QuestSeen survive restarts: a held goal keeps waiting for its unlock, and a quest event already seen
	// (same or lower stage) is not recorded again.
	Locks     map[string]lock `json:"locks,omitempty"`
	QuestSeen map[string]int  `json:"questSeen,omitempty"`
	// PrunedSeeds are the seed keys of pruned goals (oldest first, at most maxPrunedSeeds), so AddSeeded does not queue
	// a pruned --goals entry again after a restart. Additive field: no version bump.
	PrunedSeeds []string `json:"prunedSeeds,omitempty"`
}

// lock is why a goal was held: the level it was held at and, for a quest unlock, the flag and difficulty.
type lock struct {
	Level  int    `json:"level"`
	Unlock string `json:"unlock,omitempty"`
	Diff   int    `json:"diff,omitempty"`
}

// migrations[v] upgrades a document of version v to v+1.
var migrations = map[int]func(map[string]json.RawMessage) error{}

func decodeState(data []byte) (*fileState, error) {
	return decodeStateAt(data, FileVersion, migrations)
}

func decodeStateAt(data []byte, current int, migrations map[int]func(map[string]json.RawMessage) error) (*fileState, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("corrupt: %w", err)
	}
	var ver int
	if err := json.Unmarshal(raw["version"], &ver); err != nil || ver < 1 {
		return nil, errors.New("corrupt: missing version")
	}
	if ver > current {
		return nil, fmt.Errorf("%w (file v%d, this build v%d)", ErrFutureVersion, ver, current)
	}
	for ; ver < current; ver++ {
		mig := migrations[ver]
		if mig == nil {
			return nil, fmt.Errorf("corrupt: no migration from v%d", ver)
		}
		if err := mig(raw); err != nil {
			return nil, fmt.Errorf("migrate v%d: %w", ver, err)
		}
		raw["version"], _ = json.Marshal(ver + 1)
	}
	b, _ := json.Marshal(raw)
	var st fileState
	if err := json.Unmarshal(b, &st); err != nil {
		return nil, fmt.Errorf("corrupt: %w", err)
	}
	return &st, nil
}

// writeAtomic replaces path so that a crash at any point leaves either the old or the new complete file
// (possibly as path.bak; loadState looks there too). Order: write+fsync tmp, move old to .bak, rename tmp.
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

// loadState reads path, falling back to path.bak. A corrupt main file is moved aside to
// path.corrupt-<unix> (the newest three are kept) and a warning is returned. A missing file is not an
// error: it returns (nil, "", nil). ErrFutureVersion is returned as an error and nothing is moved.
func loadState(path string, now time.Time) (*fileState, string, error) {
	var warnings []string
	tried := false
	for i, p := range []string{path, path + ".bak"} {
		data, err := os.ReadFile(p)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, "", err
		}
		tried = true
		st, err := decodeState(data)
		if errors.Is(err, ErrFutureVersion) {
			return nil, "", err
		}
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s: %v", filepath.Base(p), err))
			if i == 0 {
				os.Rename(p, fmt.Sprintf("%s.corrupt-%d", path, now.Unix()))
				pruneCorrupt(path)
			}
			continue
		}
		if i == 1 {
			warnings = append(warnings, "restored from backup file")
		}
		return st, strings.Join(warnings, "; "), nil
	}
	if tried {
		return nil, "state file unreadable, starting empty: " + strings.Join(warnings, "; "), nil
	}
	return nil, "", nil
}

func pruneCorrupt(path string) {
	m, _ := filepath.Glob(path + ".corrupt-*")
	sort.Strings(m)
	for len(m) > 3 {
		os.Remove(m[0])
		m = m[1:]
	}
}
