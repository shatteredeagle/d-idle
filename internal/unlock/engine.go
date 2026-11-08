package unlock

import (
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

// Event types the engine understands. Counts (kills, retire) carry the running total, so a replayed or repeated event
// can never count twice; Codex firsts, quests and levels are idempotent by nature.
const (
	EvCodexFirst = "codex_first" // Sub: Codex kind, ID: Codex id, Name: the subject met
	EvQuest      = "quest"       // Name: quest name (a finished quest)
	EvLevel      = "level"       // N: character level
	EvKills      = "kills"       // Name: monster kind, N: kills of it so far
	EvRetire     = "retire"      // N: heroes retired so far
	EvQuestStage = "quest_stage" // Name: quest id, N: stage reached (1 started, 2 cleared, 3 done)
	EvSeen       = "seen"        // Name: something the viewer did or saw ("expedition_card"); idempotent by nature
)

// Event is one observation from a system that already exists.
type Event struct {
	Type string
	Sub  string
	ID   string
	Name string
	N    int
}

// How an entry was opened.
const (
	ByTrigger = "trigger"
	ByBuying  = "bought"
)

// Opened is the persisted record of one open entry.
type Opened struct {
	ID  string `json:"id"`
	How string `json:"how"`
	At  string `json:"at"`
}

// FileVersion is the on-disk version of unlocks-<profile>.json.
const FileVersion = 1

type file struct {
	Version int      `json:"version"`
	Profile string   `json:"profile"`
	Opened  []Opened `json:"opened"`
	Met     []string `json:"met,omitempty"` // "<sub>/<name>" of every Codex first seen
}

// Path is the per-profile file, next to the ledger and the Codex.
func Path(dir, profile string) string { return filepath.Join(dir, "unlocks-"+profile+".json") }

// Spender pays for a tally_spend entry. Key is stable per profile and entry, so a retry after a crash books once
// (a Spender treats an already booked key as paid). A short balance returns an error that wraps the ledger's
// ErrInsufficient; the engine never clamps or goes negative.
type Spender interface {
	Spend(key string, price int64, note string) error
}

// Engine holds the open entries of one profile. Safe for concurrent use. An entry never relocks.
type Engine struct {
	l       *Ladder
	path    string
	profile string
	now     func() time.Time

	mu     sync.Mutex
	opened map[string]Opened
	order  []string
	met    map[string]bool
}

// Open loads (or starts) the profile's file in dir. dir "" keeps everything in memory (tests, no state dir).
func Open(dir, profile string, l *Ladder, now func() time.Time) (*Engine, error) {
	if l == nil {
		return nil, errors.New("unlock: no ladder")
	}
	if now == nil {
		now = time.Now
	}
	e := &Engine{l: l, profile: profile, now: now, opened: map[string]Opened{}, met: map[string]bool{}}
	if dir == "" {
		return e, nil
	}
	e.path = Path(dir, profile)
	b, err := os.ReadFile(e.path)
	if errors.Is(err, os.ErrNotExist) {
		return e, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(b, &f); err != nil {
		return nil, fmt.Errorf("unlock: %s: %w", e.path, err)
	}
	if f.Version != FileVersion {
		return nil, fmt.Errorf("unlock: %s: version %d, want %d", e.path, f.Version, FileVersion)
	}
	for _, o := range f.Opened { // an id the ladder no longer has stays on file and is never reported
		if _, dup := e.opened[o.ID]; !dup {
			e.opened[o.ID] = o
			e.order = append(e.order, o.ID)
		}
	}
	for _, m := range f.Met {
		e.met[m] = true
	}
	return e, nil
}

// Profile is the profile the engine was opened for; Dir is its state directory ("" in memory). Features that keep
// their own small state file (the hero voice) put it next to the engine's.
func (e *Engine) Profile() string { return e.profile }

func (e *Engine) Dir() string {
	if e.path == "" {
		return ""
	}
	return filepath.Dir(e.path)
}

func (e *Engine) saveLocked() error {
	if e.path == "" {
		return nil
	}
	f := file{Version: FileVersion, Profile: e.profile}
	for _, id := range e.order {
		f.Opened = append(f.Opened, e.opened[id])
	}
	for m := range e.met {
		f.Met = append(f.Met, m)
	}
	sort.Strings(f.Met)
	b, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return err
	}
	tmp := e.path + ".tmp"
	if err := os.WriteFile(tmp, append(b, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, e.path)
}

func (e *Engine) openLocked(id, how string) {
	e.opened[id] = Opened{ID: id, How: how, At: e.now().UTC().Format(time.RFC3339)}
	e.order = append(e.order, id)
}

func (e *Engine) rollbackLocked(ids []string, metKey string, hadMet bool) {
	for _, id := range ids {
		delete(e.opened, id)
		e.order = e.order[:len(e.order)-1]
	}
	if metKey != "" && !hadMet {
		delete(e.met, metKey)
	}
}

// Observe feeds one event and returns the entries it opened (oldest table row first), each exactly once. An event of
// an unknown type is an error naming it. A failed save opens nothing and returns the error; the event can be sent again.
func (e *Engine) Observe(ev Event) ([]Entry, error) {
	switch ev.Type {
	case EvCodexFirst, EvQuest, EvLevel, EvKills, EvRetire, EvSeen, EvQuestStage:
	default:
		return nil, fmt.Errorf("%w %q", ErrUnknownEvent, ev.Type)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	metKey, hadMet := "", false
	if ev.Type == EvCodexFirst && ev.Name != "" {
		metKey = strings.ToLower(ev.Sub + "/" + ev.Name)
		hadMet = e.met[metKey]
		e.met[metKey] = true
	}
	var out []Entry
	var ids []string
	for _, en := range e.l.Entries {
		if _, open := e.opened[en.ID]; open {
			continue
		}
		for _, t := range append([]Trigger{en.Trigger}, en.Alt...) {
			if t.matches(ev) {
				e.openLocked(en.ID, ByTrigger)
				out = append(out, en)
				ids = append(ids, en.ID)
				break
			}
		}
	}
	if len(out) == 0 && (metKey == "" || hadMet) {
		return nil, nil
	}
	if err := e.saveLocked(); err != nil {
		e.rollbackLocked(ids, metKey, hadMet)
		return nil, err
	}
	return out, nil
}

func (t Trigger) matches(ev Event) bool {
	switch t.Kind {
	case CodexFirst:
		if ev.Type != EvCodexFirst || !strings.EqualFold(t.str("sub"), ev.Sub) {
			return false
		}
		ids := t.strs("ids")
		if len(ids) == 0 {
			return true
		}
		for _, id := range ids {
			if strings.EqualFold(id, ev.ID) {
				return true
			}
		}
		return false
	case Quest:
		return ev.Type == EvQuest && strings.EqualFold(t.str("quest"), ev.Name)
	case Level:
		n, _ := t.num("min")
		return ev.Type == EvLevel && ev.N >= n
	case QuestStage:
		n, _ := t.num("min")
		return ev.Type == EvQuestStage && strings.EqualFold(t.str("quest"), ev.Name) && ev.N >= n
	case Seen:
		return ev.Type == EvSeen && strings.EqualFold(t.str("what"), ev.Name)
	case RetireCount:
		n, _ := t.num("min")
		return ev.Type == EvRetire && ev.N >= n
	case KillsOfKind:
		n, _ := t.num("min")
		k := t.str("kind")
		return ev.Type == EvKills && (k == "*" || strings.EqualFold(k, ev.Name)) && ev.N >= n
	}
	return false
}

// Allowed says whether the entry is open. An unknown id is closed; Check says why.
func (e *Engine) Allowed(id string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	_, ok := e.opened[id]
	return ok && e.known(id)
}

func (e *Engine) known(id string) bool { _, err := e.l.Entry(id); return err == nil }

// Has says whether the ladder has a row with that id (a feature without its row stays off).
func (e *Engine) Has(id string) bool { return e.known(id) }

// Check is Allowed with an error naming an unknown entry.
func (e *Engine) Check(id string) (bool, error) {
	if _, err := e.l.Entry(id); err != nil {
		return false, err
	}
	return e.Allowed(id), nil
}

// SenseTier says whether the senses entry of tier n is open (false for a tier the ladder has no row for).
func (e *Engine) SenseTier(n int) bool {
	for _, en := range e.l.Entries {
		if en.Group == "senses" && en.Tier == n {
			return e.Allowed(en.ID)
		}
	}
	return false
}

// Met says whether a Codex first of that kind and name has been seen (monster class firsts, for per-class senses).
func (e *Engine) Met(sub, name string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.met[strings.ToLower(sub+"/"+name)]
}

// State is one entry as the watcher may see it. Text is empty while the entry is locked (no spoilers); a locked
// entry shows its teaser and, for sale, its price.
type State struct {
	ID     string `json:"id"`
	Group  string `json:"group"`
	Tier   int    `json:"tier,omitempty"`
	Open   bool   `json:"open"`
	Title  string `json:"title"`
	Text   string `json:"text,omitempty"`
	Teaser string `json:"teaser,omitempty"`
	Price  int64  `json:"price,omitempty"`
	How    string `json:"how,omitempty"`
	At     string `json:"at,omitempty"`
}

// States lists every entry in table order; group "" means all.
func (e *Engine) States(group string) []State {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := []State{}
	for _, en := range e.l.Entries {
		if group != "" && en.Group != group {
			continue
		}
		s := State{ID: en.ID, Group: en.Group, Tier: en.Tier, Title: en.Reveal.Title}
		if o, ok := e.opened[en.ID]; ok {
			s.Open, s.Text, s.How, s.At = true, en.Reveal.Text, o.How, o.At
		} else {
			s.Teaser, s.Price = en.Reveal.Teaser, en.Price
		}
		out = append(out, s)
	}
	return out
}

// Pending lists the locked entries that can be bought, in table order.
func (e *Engine) Pending() []State {
	var out []State
	for _, s := range e.States("") {
		if !s.Open && s.Price > 0 {
			out = append(out, s)
		}
	}
	return out
}

// Buy pays for a tally_spend entry and opens it. Open already: ErrAlreadyOpen. Not for sale: ErrNotBuyable. A short
// balance is refused by the Spender and nothing opens. The key is stable, so a crash between the spend and the save is
// repaired by buying again.
func (e *Engine) Buy(id string, sp Spender) (Entry, error) {
	en, err := e.l.Entry(id)
	if err != nil {
		return Entry{}, err
	}
	if en.Price <= 0 {
		return Entry{}, fmt.Errorf("%w: %q", ErrNotBuyable, id)
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, open := e.opened[id]; open {
		return en, fmt.Errorf("%w: %q", ErrAlreadyOpen, id)
	}
	if err := sp.Spend("unlock/"+e.profile+"/"+id, en.Price, "unlock "+en.Reveal.Title); err != nil {
		return Entry{}, err
	}
	e.openLocked(id, ByBuying)
	if err := e.saveLocked(); err != nil {
		e.rollbackLocked([]string{id}, "", false)
		return Entry{}, err
	}
	return en, nil
}
