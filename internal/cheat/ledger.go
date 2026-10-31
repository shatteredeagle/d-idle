package cheat

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Entry is one line of the append-only cheat ledger (JSONL). Every cheat attempt is recorded, refused
// and failed ones too (Success false, Error set), so the ledger is also the audit of what was tried.
type Entry struct {
	Time      time.Time      `json:"time"`
	Actor     string         `json:"actor"`
	Instance  string         `json:"instance,omitempty"`
	Character string         `json:"character,omitempty"`
	Kind      Kind           `json:"kind"`
	Params    map[string]any `json:"params,omitempty"`
	Before    map[string]any `json:"before,omitempty"`
	After     map[string]any `json:"after,omitempty"`
	Reason    string         `json:"reason"`
	Success   bool           `json:"success"`
	Error     string         `json:"error,omitempty"`
	Backup    string         `json:"backup,omitempty"` // save-file backup made before an edit
}

// DefaultLedgerPath is $D2CHEAT_LEDGER or <data>/ledger/cheats.jsonl (paths.Layout). The old
// ~/.local/state/d2lod/cheats.jsonl is moved there on first use (see MigrateLegacy).
func DefaultLedgerPath() string {
	if p := os.Getenv("D2CHEAT_LEDGER"); p != "" {
		return p
	}
	l := layout()
	legacyOnce.Do(func() { _ = MigrateLegacy(l) })
	return l.CheatLedger()
}

// Ledger appends to and reads a JSONL file. Appends are single O_APPEND writes (atomic for lines this
// short), so concurrent tools never interleave; the file is never rewritten.
type Ledger struct {
	Path string
	mu   sync.Mutex
}

func NewLedger(path string) *Ledger { return &Ledger{Path: path} }

// Ensure creates the directory and file so a later Append cannot fail for lack of a place to write.
func (l *Ledger) Ensure() error {
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	return f.Close()
}

func (l *Ledger) Append(e Entry) error {
	if e.Reason == "" {
		return errors.New("cheat: ledger entry without a reason")
	}
	b, err := json.Marshal(e)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.Path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(l.Path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

// Filter selects entries; zero fields match everything.
type Filter struct {
	Since, Until time.Time
	Actor        string
	Instance     string
	Character    string
	Kind         Kind
	OnlySuccess  bool
}

func (f Filter) Match(e Entry) bool {
	switch {
	case !f.Since.IsZero() && e.Time.Before(f.Since):
	case !f.Until.IsZero() && e.Time.After(f.Until):
	case f.Actor != "" && e.Actor != f.Actor:
	case f.Instance != "" && e.Instance != f.Instance:
	case f.Character != "" && !strings.EqualFold(e.Character, f.Character):
	case f.Kind != "" && e.Kind != f.Kind:
	case f.OnlySuccess && !e.Success:
	default:
		return true
	}
	return false
}

// List returns the matching entries in file order. A missing file is an empty ledger; a corrupt line
// is an error naming its number (the file is never silently skipped).
func (l *Ledger) List(f Filter) ([]Entry, error) {
	file, err := os.Open(l.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var out []Entry
	sc := bufio.NewScanner(file)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			return out, fmt.Errorf("cheat: ledger %s line %d: %w", l.Path, n, err)
		}
		if f.Match(e) {
			out = append(out, e)
		}
	}
	return out, sc.Err()
}

// Summary counts entries.
type Summary struct {
	Total, Succeeded, Failed int
	First, Last              time.Time
	ByKind                   map[Kind]int
	ByCharacter              map[string]int
	ByActor                  map[string]int
}

func Summarize(es []Entry) Summary {
	s := Summary{ByKind: map[Kind]int{}, ByCharacter: map[string]int{}, ByActor: map[string]int{}}
	for i, e := range es {
		s.Total++
		if e.Success {
			s.Succeeded++
		} else {
			s.Failed++
		}
		if i == 0 || e.Time.Before(s.First) {
			s.First = e.Time
		}
		if e.Time.After(s.Last) {
			s.Last = e.Time
		}
		s.ByKind[e.Kind]++
		s.ByCharacter[e.Character]++
		s.ByActor[e.Actor]++
	}
	return s
}

func (s Summary) String() string {
	keys := func(m map[string]int) string {
		var ks []string
		for k := range m {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		var b []string
		for _, k := range ks {
			b = append(b, fmt.Sprintf("%s=%d", k, m[k]))
		}
		return strings.Join(b, " ")
	}
	kinds := map[string]int{}
	for k, v := range s.ByKind {
		kinds[string(k)] = v
	}
	return fmt.Sprintf("%d entries (%d ok, %d failed)\n  kinds: %s\n  characters: %s\n  actors: %s",
		s.Total, s.Succeeded, s.Failed, keys(kinds), keys(s.ByCharacter), keys(s.ByActor))
}

// Cheated reports whether a successful cheat on character (any character when "") happened at or after
// since. Statistics and the dashboard call it to flag or exclude a run: pass the run's start time.
func (l *Ledger) Cheated(since time.Time, character string) bool {
	es, err := l.List(Filter{Since: since, Character: character, OnlySuccess: true})
	return err == nil && len(es) > 0
}

// Cheated is Ledger.Cheated on the default ledger.
func Cheated(since time.Time) bool { return NewLedger(DefaultLedgerPath()).Cheated(since, "") }

// FormatJSON renders an entry as its ledger line.
func FormatJSON(e Entry) string {
	b, _ := json.Marshal(e)
	return string(b)
}
