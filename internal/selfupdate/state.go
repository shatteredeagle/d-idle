package selfupdate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State is /data/didle/state.json (design 3.1). The boot shim and the updater both write it, so every write is
// atomic (temp file, fsync, rename).
type State struct {
	Current      string    `json:"current"`
	Previous     string    `json:"previous"`
	Pending      string    `json:"pending"`
	Good         string    `json:"good"`
	Bad          []string  `json:"bad"`
	SeqFloor     int64     `json:"seq_floor"`
	VersionFloor string    `json:"version_floor"` // monotonic: nothing below it is accepted again
	Boots        int       `json:"boots"`
	LastCheck    time.Time `json:"last_check"`
	ETag         string    `json:"etag"`
}

// RollbackMarker is /data/didle/rollback.json: why a release was left and what runs now. The watch screen shows it
// once ("Update 0.4.0 failed and was rolled back: <reason>").
type RollbackMarker struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Reason string    `json:"reason"`
	At     time.Time `json:"at"`
	Seen   bool      `json:"seen"`
}

// Layout names the paths below the update root (default /data/didle).
type Layout struct{ Root string }

func (l Layout) StateFile() string       { return filepath.Join(l.Root, "state.json") }
func (l Layout) MarkerFile() string      { return filepath.Join(l.Root, "rollback.json") }
func (l Layout) Releases() string        { return filepath.Join(l.Root, "releases") }
func (l Layout) Release(v string) string { return filepath.Join(l.Releases(), v) }
func (l Layout) Current() string         { return filepath.Join(l.Root, "current") }
func (l Layout) Staging() string         { return filepath.Join(l.Root, "staging") }
func (l Layout) Inbox() string           { return filepath.Join(l.Root, "inbox") }

// LoadState reads the state; a missing file is the zero state.
func (l Layout) LoadState() (State, error) {
	var s State
	b, err := os.ReadFile(l.StateFile())
	if errors.Is(err, os.ErrNotExist) {
		return s, nil
	}
	if err != nil {
		return s, err
	}
	if err := json.Unmarshal(b, &s); err != nil {
		return State{}, fmt.Errorf("selfupdate: %s: %w", l.StateFile(), err)
	}
	return s, nil
}

// SaveState writes the state atomically.
func (l Layout) SaveState(s State) error { return l.writeJSON(l.StateFile(), s) }

// WriteMarker records a rollback.
func (l Layout) WriteMarker(m RollbackMarker) error { return l.writeJSON(l.MarkerFile(), m) }

// ReadMarker returns the marker, or nil when there is none.
func (l Layout) ReadMarker() (*RollbackMarker, error) {
	b, err := os.ReadFile(l.MarkerFile())
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var m RollbackMarker
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (l Layout) writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}

// IsBad reports whether v is on the local bad list.
func (s State) IsBad(v string) bool {
	for _, b := range s.Bad {
		if CompareVersions(b, v) == 0 {
			return true
		}
	}
	return false
}

// Rollback points `current` back at `to` ("" = the baked base: the link is removed), clears the pending state, puts
// `from` on the bad list (it is not retried; a newer version clears the need) and writes the marker. The boot shim and
// the engine share it.
func (l Layout) Rollback(from, to, reason string, at time.Time) error {
	if to != "" {
		if err := l.SwapCurrent(to); err != nil {
			return err
		}
	} else if err := os.Remove(l.Current()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	st, err := l.LoadState()
	if err != nil {
		return err
	}
	st.Current, st.Previous, st.Pending, st.Boots = to, "", "", 0
	if !st.IsBad(from) {
		st.Bad = append(st.Bad, from)
	}
	if err := l.SaveState(st); err != nil {
		return err
	}
	return l.WriteMarker(RollbackMarker{From: from, To: to, Reason: reason, At: at})
}
