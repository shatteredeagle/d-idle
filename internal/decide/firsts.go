package decide

import (
	_ "embed"
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

// The Act 1 firsts (TASK-130, ADR 0018 item 5, docs/codex.md). A first is the hero's first meeting with a monster
// kind, an area, a waypoint, a shrine, a drop kind, a champion pack or a death. Each one fills a Codex entry and, when
// the table gives a question, raises ONE short thought through the decision-point store with a default and a visible
// consequence. The table is data (firsts_act1.json); no monster, area or class name is Go.

//go:embed firsts_act1.json
var shippedFirsts []byte

// FirstsPoint is the catalogue point every first asks; its context key is the "first" field (the first's id), so the
// store dedupes per first and a rule is never learned (a key is answered once).
const FirstsPoint = "t0.first"

// First is one row of the firsts table.
type First struct {
	ID    string   `json:"id"`
	Kind  string   `json:"kind"`
	Match []string `json:"match"`
	Area  string   `json:"area,omitempty"`
	Title string   `json:"title"`
	// Question empty means a quiet first: the Codex is filled, no thought opens.
	Question       string            `json:"question,omitempty"`
	Yes            string            `json:"yes,omitempty"`
	No             string            `json:"no,omitempty"`
	Default        string            `json:"default,omitempty"`
	DefaultByClass map[string]string `json:"defaultByClass,omitempty"`
	Teaches        string            `json:"teaches,omitempty"`
	Consequence    string            `json:"consequence,omitempty"`
	Tip            string            `json:"tip"`
	TipTag         string            `json:"tipTag"`
}

// Firsts is the loaded table.
type Firsts struct {
	Version int     `json:"version"`
	Group   string  `json:"group"`
	Doc     string  `json:"doc"`
	Field   Field   `json:"field"`
	Point   Point   `json:"point"`
	Firsts  []First `json:"firsts"`
}

var firstKinds = []string{"monster", "area", "waypoint", "shrine", "drop", "champion", "death"}

// ShippedFirsts loads the embedded Act 1 table.
func ShippedFirsts() (*Firsts, error) { return ParseFirsts(shippedFirsts) }

// ParseFirsts decodes and validates a firsts document.
func ParseFirsts(data []byte) (*Firsts, error) {
	var f Firsts
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("decide: firsts: %w", err)
	}
	seen := map[string]bool{}
	for _, r := range f.Firsts {
		if badToken(r.ID) {
			return nil, fmt.Errorf("decide: firsts: bad id %q", r.ID)
		}
		if seen[r.ID] {
			return nil, fmt.Errorf("decide: firsts: duplicate id %q", r.ID)
		}
		seen[r.ID] = true
		if !contains(firstKinds, r.Kind) {
			return nil, fmt.Errorf("decide: firsts: %q has unknown kind %q", r.ID, r.Kind)
		}
		if len(r.Match) == 0 || r.Title == "" || r.Tip == "" || !contains([]string{"V", "S", "U"}, r.TipTag) {
			return nil, fmt.Errorf("decide: firsts: %q needs match, title, tip and a V/S/U tip tag", r.ID)
		}
		if r.Question != "" {
			if r.Yes == "" || r.No == "" || (r.Default != "yes" && r.Default != "no") || r.Consequence == "" || r.Teaches == "" {
				return nil, fmt.Errorf("decide: firsts: %q: a question needs yes, no, default yes|no, teaches and consequence", r.ID)
			}
			for cl, d := range r.DefaultByClass {
				if d != "yes" && d != "no" {
					return nil, fmt.Errorf("decide: firsts: %q: defaultByClass[%s] %q is not yes or no", r.ID, cl, d)
				}
			}
		}
	}
	if len(f.Field.Values) != len(f.Firsts) {
		return nil, fmt.Errorf("decide: firsts: field lists %d values for %d firsts", len(f.Field.Values), len(f.Firsts))
	}
	for i, r := range f.Firsts {
		if f.Field.Values[i] != r.ID {
			return nil, fmt.Errorf("decide: firsts: field value %d is %q, the table has %q", i, f.Field.Values[i], r.ID)
		}
	}
	return &f, nil
}

// WithFirsts returns a copy of the catalogue that also carries the "first" field and the t0.first point. The shipped
// catalogue is unchanged, so its point count and parity tables stay what they were; the runner that wants the firsts
// builds its store from this copy. A catalogue that already has either is an error.
func (c *Catalogue) WithFirsts(f *Firsts) (*Catalogue, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}
	var n Catalogue
	if err := json.Unmarshal(b, &n); err != nil {
		return nil, err
	}
	if _, dup := n.Fields["first"]; dup {
		return nil, errors.New(`decide: catalogue already has the field "first"`)
	}
	n.Fields["first"] = f.Field
	n.Points = append(n.Points, f.Point)
	if err := n.validate(); err != nil {
		return nil, err
	}
	return &n, nil
}

// FirstEvent is what a reader reports. Kind is a table kind. Name is the monster, area or waypoint name, the shrine
// name, the drop kind (rune, unique, set, rare), the champion's mod name (one event per mod, or one with no mods),
// or the killer's name. Area is the area name; Mods are the champion's mods; Clvl is display only.
type FirstEvent struct {
	Kind string
	Name string
	Area string
	Mods []string
	Clvl int
}

// matches reports whether the event is this first's trigger.
func (r First) matches(e FirstEvent) bool {
	if r.Kind != e.Kind {
		return false
	}
	if r.Area != "" && !strings.EqualFold(r.Area, e.Area) && !strings.EqualFold(r.Area, e.Name) {
		return false
	}
	names := append([]string{e.Name}, e.Mods...)
	for _, m := range r.Match {
		if m == "*" {
			return true
		}
		for _, n := range names {
			if strings.EqualFold(m, n) {
				return true
			}
		}
	}
	return false
}

// Entry of the Codex: one first the hero has met.
type CodexEntry struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Title    string `json:"title"`
	Subject  string `json:"subject"`
	Area     string `json:"area,omitempty"`
	Clvl     int    `json:"clvl,omitempty"`
	FirstSec int64  `json:"firstSec"` // virtual seconds since the run started
	// Decided is the answer id in force ("" for a quiet first); Source says who chose: "default" (the table, on
	// the player's behalf), "answered" (the player), "quiet" (no question).
	Decided string `json:"decided,omitempty"`
	Label   string `json:"label,omitempty"`
	Source  string `json:"source"`
	Open    bool   `json:"open,omitempty"` // the thought is still open: the player can still change the default
	Key     string `json:"key,omitempty"`
	Tip     string `json:"tip"`
	TipTag  string `json:"tipTag"`
	Taught  string `json:"taught,omitempty"`
}

// CodexFileVersion is the on-disk version of the codex file.
const CodexFileVersion = 1

// Codex is the per-profile record of firsts. Persistent: every change is saved atomically.
type Codex struct {
	mu      sync.Mutex
	path    string
	profile string
	order   []string
	byID    map[string]*CodexEntry
}

type codexFile struct {
	Version int          `json:"version"`
	Profile string       `json:"profile"`
	Entries []CodexEntry `json:"entries"`
}

// NewCodex is an in-memory codex (tests).
func NewCodex(profile string) *Codex { return &Codex{profile: profile, byID: map[string]*CodexEntry{}} }

// CodexPath is the codex file of a profile inside dir.
func CodexPath(dir, profile string) string { return filepath.Join(dir, "codex-"+profile+".json") }

// OpenCodex loads (or creates) the codex of a profile in dir.
func OpenCodex(dir, profile string) (*Codex, error) {
	if !validProfile(profile) {
		return nil, fmt.Errorf("%w: %q", ErrBadProfile, profile)
	}
	c := NewCodex(profile)
	c.path = CodexPath(dir, profile)
	data, err := os.ReadFile(c.path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return nil, err
	}
	var f codexFile
	if err := json.Unmarshal(data, &f); err != nil || f.Profile != profile || f.Version < 1 {
		return nil, fmt.Errorf("%w: %s", ErrCorrupt, c.path)
	}
	if f.Version > CodexFileVersion {
		return nil, fmt.Errorf("%w (codex v%d, this build v%d)", ErrFutureVersion, f.Version, CodexFileVersion)
	}
	for _, e := range f.Entries {
		e := e
		c.order = append(c.order, e.ID)
		c.byID[e.ID] = &e
	}
	return c, nil
}

func (c *Codex) saveLocked() error {
	if c.path == "" {
		return nil
	}
	f := codexFile{Version: CodexFileVersion, Profile: c.profile}
	for _, id := range c.order {
		f.Entries = append(f.Entries, *c.byID[id])
	}
	b, err := json.MarshalIndent(f, "", " ")
	if err != nil {
		return err
	}
	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, c.path)
}

// Has reports whether the first is already in the Codex.
func (c *Codex) Has(id string) bool { c.mu.Lock(); defer c.mu.Unlock(); return c.byID[id] != nil }

// Entry returns a copy of one entry.
func (c *Codex) Entry(id string) (CodexEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if e := c.byID[id]; e != nil {
		return *e, true
	}
	return CodexEntry{}, false
}

// Decided is the answer in force for a first ("" if it is not in the Codex or is quiet): the consumer of a
// rule the first taught reads it here.
func (c *Codex) Decided(id string) string {
	if e, ok := c.Entry(id); ok {
		return e.Decided
	}
	return ""
}

// Entries lists the entries, oldest first.
func (c *Codex) Entries() []CodexEntry {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]CodexEntry, 0, len(c.order))
	for _, id := range c.order {
		out = append(out, *c.byID[id])
	}
	return out
}

func (c *Codex) add(e CodexEntry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.byID[e.ID] != nil {
		return nil
	}
	c.order = append(c.order, e.ID)
	c.byID[e.ID] = &e
	return c.saveLocked()
}

// CodexSummary is the read the return screen and /api/codex show: counts, and the entries within the window.
type CodexSummary struct {
	Total   int            `json:"total"`  // firsts in the table
	Filled  int            `json:"filled"` // entries in the Codex
	Asked   int            `json:"asked"`  // entries that opened a question
	Window  string         `json:"window,omitempty"`
	Recent  []CodexEntry   `json:"recent"`
	ByKind  map[string]int `json:"byKind"`
	Missing []string       `json:"missing,omitempty"` // titles not met yet, table order
}

// Summary describes the Codex against the table. window 0 lists every entry as recent; otherwise those first seen in
// the last window of virtual time up to now.
func (c *Codex) Summary(t *Firsts, now, window time.Duration) CodexSummary {
	c.mu.Lock()
	defer c.mu.Unlock()
	s := CodexSummary{Total: len(t.Firsts), Recent: []CodexEntry{}, ByKind: map[string]int{}}
	if window > 0 {
		s.Window = window.String()
	}
	for _, id := range c.order {
		e := c.byID[id]
		s.Filled++
		s.ByKind[e.Kind]++
		if e.Source != "quiet" {
			s.Asked++
		}
		if window <= 0 || time.Duration(e.FirstSec)*time.Second >= now-window {
			s.Recent = append(s.Recent, *e)
		}
	}
	sort.SliceStable(s.Recent, func(i, j int) bool { return s.Recent[i].FirstSec > s.Recent[j].FirstSec })
	for _, r := range t.Firsts {
		if c.byID[r.ID] == nil {
			s.Missing = append(s.Missing, r.Title)
		}
	}
	return s
}

// Who is the hero data a tracker needs for the context key: the catalogue's class table key, difficulty and hardcore.
type Who struct {
	Class string
	Diff  string // normal, nightmare, hell
	HC    bool
}

// Tracker turns reader events into Codex entries and thoughts. One per run.
type Tracker struct {
	tbl   *Firsts
	cat   *Catalogue
	store *Store
	codex *Codex
	clk   Clock
	who   Who
}

// NewTracker checks that the catalogue carries the firsts point and that the class is in its class table and in every
// defaultByClass of the table; an unknown class is an error naming it, never a fallback.
func NewTracker(tbl *Firsts, cat *Catalogue, store *Store, codex *Codex, clk Clock, who Who) (*Tracker, error) {
	if _, err := cat.Point(FirstsPoint); err != nil {
		return nil, fmt.Errorf("decide: the catalogue has no firsts point (use Catalogue.WithFirsts): %w", err)
	}
	if _, ok := cat.Classes[who.Class]; !ok {
		return nil, fmt.Errorf("decide: unknown class %q", who.Class)
	}
	for _, r := range tbl.Firsts {
		for cl := range r.DefaultByClass {
			if _, ok := cat.Classes[cl]; !ok {
				return nil, fmt.Errorf("decide: firsts: %q: defaultByClass names unknown class %q", r.ID, cl)
			}
		}
	}
	return &Tracker{tbl: tbl, cat: cat, store: store, codex: codex, clk: clk, who: who}, nil
}

// Fired is one first that just filled the Codex.
type Fired struct {
	ID       string
	Entry    CodexEntry
	Decision *Decision // nil for a quiet first
}

func (t *Tracker) defaultFor(r First) string {
	if d, ok := r.DefaultByClass[t.who.Class]; ok {
		return d
	}
	return r.Default
}

func (t *Tracker) obs(r First, e FirstEvent) Obs {
	hc := "normal"
	if t.who.HC {
		hc = "hardcore"
	}
	sit := Situation{"first": r.ID, "kind": r.Kind, "title": r.Title, "subject": e.Name, "question": r.Question,
		"yes": r.Yes, "no": r.No, "teaches": r.Teaches, "consequence": r.Consequence, "tip": r.Tip, "tiptag": r.TipTag}
	if e.Area != "" {
		sit["area"] = e.Area
	}
	if e.Clvl > 0 {
		sit["herolevel"] = fmt.Sprint(e.Clvl)
	}
	return Obs{Class: t.who.Class, Cat: map[string]string{"act": "1", "diff": t.who.Diff, "hc": hc, "first": r.ID}, Sit: sit}
}

// Observe feeds one reader event. The first row in table order that the event triggers claims it: it fills the Codex once and
// opens its question; an event seen again (a replay, a reload) fires nothing. An error names the unknown class, field or value.
func (t *Tracker) Observe(e FirstEvent) ([]Fired, error) {
	if !contains(firstKinds, e.Kind) {
		return nil, fmt.Errorf("decide: firsts: unknown event kind %q", e.Kind)
	}
	var out []Fired
	for _, r := range t.tbl.Firsts {
		if !r.matches(e) {
			continue
		}
		if t.codex.Has(r.ID) {
			break // the event is already claimed by this row
		}
		ent := CodexEntry{ID: r.ID, Kind: r.Kind, Title: r.Title, Subject: e.Name, Area: e.Area, Clvl: e.Clvl,
			FirstSec: int64(t.clk.Now() / time.Second), Tip: r.Tip, TipTag: r.TipTag, Source: "quiet"}
		f := Fired{ID: r.ID}
		if r.Question != "" {
			ctx, err := t.cat.Build(FirstsPoint, t.obs(r, e), 1, nil)
			if err != nil {
				return out, err
			}
			d, err := t.store.AskWithDefault(ctx, t.defaultFor(r))
			if err != nil {
				return out, err
			}
			ent.Key, ent.Decided, ent.Source, ent.Taught = ctx.Key, d.Answer, "default", r.Teaches
			ent.Open = d.Thought != nil
			ent.Label = r.label(d.Answer)
			f.Decision = &d
		}
		f.Entry = ent
		if err := t.codex.add(ent); err != nil {
			return out, err
		}
		out = append(out, f)
		break // one first per event: the first matching row in table order
	}
	return out, nil
}

func (r First) label(answer string) string {
	if answer == "yes" {
		return r.Yes
	}
	return r.No
}

// Settle updates the entries whose thought was open: a recorded answer wins (source "answered"), a thought that is
// gone without one faded and the default stays. Call it from the bot loop after Tick, or before reading the Codex.
func (t *Tracker) Settle() error {
	open := map[string]bool{}
	for _, th := range t.store.OpenThoughts() {
		open[th.Key] = true
	}
	t.codex.mu.Lock()
	defer t.codex.mu.Unlock()
	changed := false
	for _, id := range t.codex.order {
		e := t.codex.byID[id]
		if !e.Open || open[e.Key] {
			continue
		}
		e.Open = false
		if rec, ok := t.store.Record(e.Key); ok && rec.Last != "" {
			e.Decided, e.Source = rec.Last, "answered"
			for _, r := range t.tbl.Firsts {
				if r.ID == e.ID {
					e.Label = r.label(rec.Last)
				}
			}
		}
		changed = true
	}
	if changed {
		return t.codex.saveLocked()
	}
	return nil
}
