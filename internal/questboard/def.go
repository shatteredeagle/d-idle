// Package questboard is the quest board model (docs/design/quest-board-2026-11-16.md, slice S1): quest
// definitions as data, the nine states, the derive-from-evidence function and the plan-record store. It is
// additive and unused until S3: nothing in the bot imports it yet.
//
// Tags used here: V verified (read in this tree or by a test), S secondary, U unverified, I design choice.
package questboard

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Schema is the definitions file schema.
const Schema = "quests.def/1"

// Difficulties are the keys of every per-difficulty column, in game order (the index is the save's).
var Difficulties = [3]string{"normal", "nm", "hell"}

// DiffIndex returns the index of a difficulty key, or an error naming it.
func DiffIndex(d string) (int, error) {
	for i, k := range Difficulties {
		if k == d {
			return i, nil
		}
	}
	return 0, fmt.Errorf("questboard: unknown difficulty %q (want normal, nm or hell)", d)
}

// Closed vocabularies; a value outside them is a validation error naming it.
var (
	kinds     = set("story", "optional", "unlock")
	risks     = set("routine", "boss", "hazard")
	autoModes = set("allowed", "never")
	endsKinds = set("handin", "item", "auto")
)

func set(v ...string) map[string]bool {
	m := map[string]bool{}
	for _, s := range v {
		m[s] = true
	}
	return m
}

// Giver is an NPC and the dialog pick (the rows of internal/talk/tables.go).
type Giver struct {
	NPC  string `json:"npc"`
	Pick string `json:"pick,omitempty"`
}

// Item is a quest item and where it comes from. Pickit must take it (V docs/progression/quests.md, act 4).
type Item struct {
	Name string `json:"name"`
	From string `json:"from,omitempty"`
}

// Reward is what lands at hand-in. Repeats lists the difficulties in which it is paid again.
type Reward struct {
	Kind    string   `json:"kind"`
	Amount  int      `json:"amount,omitempty"`
	Item    string   `json:"item,omitempty"`
	Repeats []string `json:"repeats,omitempty"`
}

// Chunk is the unit of Start. Order: After lists chunks that must be done first; empty means any order.
// Requires lists quest ids that must be done (T1/T2 only) before the chunk can be offered.
type Chunk struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	After    []string `json:"after,omitempty"`
	Requires []string `json:"requires,omitempty"`
	Ends     string   `json:"ends"` // handin | item | auto
	Giver    *Giver   `json:"handin,omitempty"`
	Steps    []string `json:"steps,omitempty"`
	Boss     string   `json:"boss,omitempty"`
	Minutes  [2]int   `json:"minutes"`
	Risk     string   `json:"risk,omitempty"`
	Meta     Meta     `json:"meta,omitempty"`

	// The handler mapping (slice S7). Handler is the task-layer group the chunk runs (default: the quest's task
	// id); Nodes are the step suffixes of that group the chunk runs in the field (a quest cut into chunks over one
	// group must list them, a group of its own runs all but its hand-in); HandInStep is the suffix of the hand-in
	// step ("reward" when empty). Unsupported is the reason no handler can run the chunk yet: the offer shows it
	// and has no Start (never a dead button).
	Handler     string   `json:"handler,omitempty"`
	Nodes       []string `json:"nodes,omitempty"`
	HandInStep  string   `json:"handInStep,omitempty"`
	Unsupported string   `json:"unsupported,omitempty"`
	// NoGold says the chunk buys nothing on its way (no shop, no gamble): the purse recovery that refills gold and kit
	// does not hold its start (cainfix3 F3). Default false: the recovery waits, as before.
	NoGold bool `json:"noGold,omitempty"`
}

// ChunkByID returns the chunk of q with that id; an unknown id is an error naming it.
func (q Quest) ChunkByID(id string) (Chunk, error) {
	for _, c := range q.Chunks {
		if c.ID == id {
			return c, nil
		}
	}
	return Chunk{}, fmt.Errorf("questboard: quest %q has no chunk %q", q.ID, id)
}

// HandlerRef says which task-layer steps a chunk runs. All: every step of Group except the hand-in step; else
// only Field. HandIn is the suffix of the hand-in step.
type HandlerRef struct {
	Group  string
	All    bool
	Field  []string
	HandIn string
}

// HandlerOf resolves the handler of chunk c of q. The task id of the quest is the default group.
func (q Quest) HandlerOf(c Chunk) HandlerRef {
	g := c.Handler
	if g == "" {
		g = q.TaskID
	}
	if g == "" {
		g = q.ID
	}
	h := c.HandInStep
	if h == "" {
		h = "reward"
	}
	// Over a shared group (several chunks, no group of its own) only the listed steps belong to the chunk.
	shared := len(q.Chunks) > 1 && c.Handler == ""
	return HandlerRef{Group: g, All: !shared && len(c.Nodes) == 0, Field: c.Nodes, HandIn: h}
}

// Meta is the meta currency column per difficulty (placeholders, U: tuned later, never a number in Go).
type Meta map[string]int

// Quest is one row of the definitions.
type Quest struct {
	ID       string   `json:"id"`     // the save quest id (internal/saves questTable)
	TaskID   string   `json:"taskId"` // the task-layer id (internal/task/world.go), empty when equal
	Title    string   `json:"title"`
	Blurb    string   `json:"blurb"`
	Act      int      `json:"act"`
	Diffs    []string `json:"difficulties"`
	Kind     string   `json:"kind"`
	Requires []string `json:"requires,omitempty"`
	// RequiresCleared lists quests whose objective (Fact.Cleared, any tier) is enough: the prerequisite that this quest
	// itself completes. The Seven Tombs needs the Horadric Staff made; only placing it in the orifice sets the Staff's
	// done bit (S d2d SevenTombsQuest::join, FUN_0059d4f0: record 10 bit 0 means placed), so requires would never open.
	RequiresCleared []string          `json:"requiresCleared,omitempty"`
	SoftAfter       []string          `json:"softAfter,omitempty"`
	Accept          *Giver            `json:"accept,omitempty"`
	Chunks          []Chunk           `json:"chunks"`
	Items           []Item            `json:"items,omitempty"`
	Boss            string            `json:"boss,omitempty"`
	Rewards         []Reward          `json:"rewards,omitempty"`
	Meta            Meta              `json:"meta,omitempty"`
	Minutes         [2]int            `json:"minutes"`
	Risk            string            `json:"risk"`
	AutoStart       string            `json:"autoStart"`
	LegacyFlor      map[string]int    `json:"legacyFloor,omitempty"` // clvl by class family, transitional (design 1.1)
	Tags            map[string]string `json:"tags,omitempty"`        // field -> V|S|U|SRC|EXP, with the source
}

// File is one act's definitions file.
type File struct {
	Schema string  `json:"schema"`
	Act    int     `json:"act"`
	Source string  `json:"source"`
	Quests []Quest `json:"quests"`
}

// Families maps a class to its family (melee, caster, summoner), from data (no class literal in Go).
type Families struct {
	Schema   string            `json:"schema"`
	Families map[string]string `json:"families"`
	Tags     map[string]string `json:"tags,omitempty"`
}

// Family returns the family of class; an unknown class is an error naming it.
func (f Families) Family(class string) (string, error) {
	fam, ok := f.Families[strings.ToLower(class)]
	if !ok {
		return "", fmt.Errorf("questboard: unknown class %q (known: %s)", class, strings.Join(sortedKeys(f.Families), ", "))
	}
	return fam, nil
}

func sortedKeys[V any](m map[string]V) []string {
	k := make([]string, 0, len(m))
	for s := range m {
		k = append(k, s)
	}
	sort.Strings(k)
	return k
}

// Board is the validated set of all acts.
type Board struct {
	Families Families
	quests   []Quest
	byID     map[string]int
}

// Parse decodes act files and the families table and validates the whole set. Unknown fields are errors.
func Parse(families []byte, acts ...[]byte) (*Board, error) {
	b := &Board{byID: map[string]int{}}
	if err := strict(families, &b.Families); err != nil {
		return nil, fmt.Errorf("questboard: families: %w", err)
	}
	if b.Families.Schema != "quests.families/1" {
		return nil, fmt.Errorf("questboard: families schema %q, want quests.families/1", b.Families.Schema)
	}
	for i, raw := range acts {
		var f File
		if err := strict(raw, &f); err != nil {
			return nil, fmt.Errorf("questboard: act file %d: %w", i, err)
		}
		if f.Schema != Schema {
			return nil, fmt.Errorf("questboard: act file %d schema %q, want %s", i, f.Schema, Schema)
		}
		if f.Act < 1 || f.Act > 5 {
			return nil, fmt.Errorf("questboard: act file %d names unknown act %d (want 1 to 5)", i, f.Act)
		}
		for _, q := range f.Quests {
			if q.Act != f.Act {
				return nil, fmt.Errorf("questboard: quest %q is act %d in the act %d file", q.ID, q.Act, f.Act)
			}
			if _, dup := b.byID[q.ID]; dup {
				return nil, fmt.Errorf("questboard: duplicate quest %q", q.ID)
			}
			b.byID[q.ID] = len(b.quests)
			b.quests = append(b.quests, q)
		}
	}
	return b, b.validate()
}

func strict(raw []byte, v any) error {
	d := json.NewDecoder(strings.NewReader(string(raw)))
	d.DisallowUnknownFields()
	return d.Decode(v)
}

// Quests lists all quests in act then file order.
func (b *Board) Quests() []Quest { return append([]Quest(nil), b.quests...) }

// Quest looks a quest up; an unknown id is an error naming it.
func (b *Board) Quest(id string) (Quest, error) {
	i, ok := b.byID[id]
	if !ok {
		return Quest{}, fmt.Errorf("questboard: unknown quest %q", id)
	}
	return b.quests[i], nil
}

// Act lists the quests of one act; an act outside 1 to 5 is an error.
func (b *Board) Act(act int) ([]Quest, error) {
	if act < 1 || act > 5 {
		return nil, fmt.Errorf("questboard: unknown act %d (want 1 to 5)", act)
	}
	var out []Quest
	for _, q := range b.quests {
		if q.Act == act {
			out = append(out, q)
		}
	}
	return out, nil
}

// Chunk returns one chunk of a quest; unknown ids are errors naming them.
func (b *Board) Chunk(quest, chunk string) (Chunk, error) {
	q, err := b.Quest(quest)
	if err != nil {
		return Chunk{}, err
	}
	for _, c := range q.Chunks {
		if c.ID == chunk {
			return c, nil
		}
	}
	return Chunk{}, fmt.Errorf("questboard: quest %q has no chunk %q", quest, chunk)
}

// Exists reports whether the quest exists in difficulty d (index); false means policy skips it there.
func (q Quest) Exists(d int) bool {
	for _, k := range q.Diffs {
		if k == Difficulties[d] {
			return true
		}
	}
	return false
}

// Floor returns the transitional clvl floor for a class (design 1.1 legacyFloor, shown beside the margin
// word). ok is false when the quest has none for the family. An unknown class is an error naming it.
func (b *Board) Floor(q Quest, class string) (int, bool, error) {
	fam, err := b.Families.Family(class)
	if err != nil {
		return 0, false, err
	}
	n, ok := q.LegacyFlor[fam]
	return n, ok, nil
}

func (b *Board) validate() error {
	for _, q := range b.quests {
		if err := b.validateQuest(q); err != nil {
			return err
		}
	}
	return b.requiresCycle()
}

// requiresCycle rejects a circular hard-prerequisite chain between quests.
func (b *Board) requiresCycle() error {
	state := map[string]int{}
	var visit func(id string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("questboard: quest prerequisites are circular at %q", id)
		case 2:
			return nil
		}
		state[id] = 1
		q := b.quests[b.byID[id]]
		for _, r := range append(append([]string{}, q.Requires...), q.RequiresCleared...) {
			if err := visit(r); err != nil {
				return err
			}
		}
		state[id] = 2
		return nil
	}
	for _, q := range b.quests {
		if err := visit(q.ID); err != nil {
			return err
		}
	}
	return nil
}

func (b *Board) validateQuest(q Quest) error {
	at := fmt.Sprintf("quest %q", q.ID)
	if q.ID == "" || q.Title == "" {
		return fmt.Errorf("questboard: a quest needs an id and a title (%q)", q.ID)
	}
	if !kinds[q.Kind] {
		return fmt.Errorf("questboard: %s: unknown kind %q", at, q.Kind)
	}
	if !risks[q.Risk] {
		return fmt.Errorf("questboard: %s: unknown risk %q", at, q.Risk)
	}
	if !autoModes[q.AutoStart] {
		return fmt.Errorf("questboard: %s: unknown autoStart %q", at, q.AutoStart)
	}
	// A hazard is never auto-started until the user opts in with eyes open (design 1.1).
	if q.Risk == "hazard" && q.AutoStart != "never" {
		return fmt.Errorf("questboard: %s: risk hazard needs autoStart never", at)
	}
	if len(q.Diffs) == 0 {
		return fmt.Errorf("questboard: %s: no difficulties", at)
	}
	for _, d := range q.Diffs {
		if _, err := DiffIndex(d); err != nil {
			return fmt.Errorf("%s: %w", at, err)
		}
	}
	if err := checkMinutes(at, q.Minutes); err != nil {
		return err
	}
	if err := b.checkMeta(at, q.Meta); err != nil {
		return err
	}
	for _, id := range append(append(append([]string{}, q.Requires...), q.RequiresCleared...), q.SoftAfter...) {
		if _, ok := b.byID[id]; !ok {
			return fmt.Errorf("questboard: %s: requires unknown quest %q", at, id)
		}
		if id == q.ID {
			return fmt.Errorf("questboard: %s requires itself", at)
		}
	}
	for fam := range q.LegacyFlor {
		if _, ok := familySet(b.Families)[fam]; !ok {
			return fmt.Errorf("questboard: %s: legacyFloor names unknown family %q", at, fam)
		}
	}
	for _, r := range q.Rewards {
		for _, d := range r.Repeats {
			if _, err := DiffIndex(d); err != nil {
				return fmt.Errorf("%s: reward %q: %w", at, r.Kind, err)
			}
		}
	}
	if len(q.Chunks) == 0 {
		return fmt.Errorf("questboard: %s: no chunks", at)
	}
	ids := map[string]bool{}
	for _, c := range q.Chunks {
		if c.ID == "" || ids[c.ID] {
			return fmt.Errorf("questboard: %s: empty or duplicate chunk id %q", at, c.ID)
		}
		ids[c.ID] = true
	}
	for _, c := range q.Chunks {
		cat := fmt.Sprintf("%s chunk %q", at, c.ID)
		if !endsKinds[c.Ends] {
			return fmt.Errorf("questboard: %s: unknown ends %q", cat, c.Ends)
		}
		if c.Ends == "handin" && c.Giver == nil {
			return fmt.Errorf("questboard: %s: ends handin needs a handin giver", cat)
		}
		if c.Risk != "" && !risks[c.Risk] {
			return fmt.Errorf("questboard: %s: unknown risk %q", cat, c.Risk)
		}
		if err := checkMinutes(cat, c.Minutes); err != nil {
			return err
		}
		if err := b.checkMeta(cat, c.Meta); err != nil {
			return err
		}
		if c.HandInStep != "" && c.Ends != "handin" {
			return fmt.Errorf("questboard: %s: handInStep needs ends handin", cat)
		}
		for _, n := range c.Nodes {
			if n == "" {
				return fmt.Errorf("questboard: %s: empty node name", cat)
			}
		}
		for _, a := range c.After {
			if !ids[a] || a == c.ID {
				return fmt.Errorf("questboard: %s: after names unknown chunk %q", cat, a)
			}
		}
		for _, r := range c.Requires {
			if _, ok := b.byID[r]; !ok {
				return fmt.Errorf("questboard: %s: requires unknown quest %q", cat, r)
			}
		}
	}
	return chunkCycle(q)
}

func familySet(f Families) map[string]bool {
	m := map[string]bool{}
	for _, fam := range f.Families {
		m[fam] = true
	}
	return m
}

func checkMinutes(at string, m [2]int) error {
	if m[0] < 0 || m[1] < m[0] {
		return fmt.Errorf("questboard: %s: minutes %v must be [lo, hi] with 0 <= lo <= hi", at, m)
	}
	return nil
}

func (b *Board) checkMeta(at string, m Meta) error {
	for d, n := range m {
		if _, err := DiffIndex(d); err != nil {
			return fmt.Errorf("%s: meta: %w", at, err)
		}
		if n < 0 {
			return fmt.Errorf("questboard: %s: meta %s is negative", at, d)
		}
	}
	return nil
}

// chunkCycle rejects a circular After chain.
func chunkCycle(q Quest) error {
	after := map[string][]string{}
	for _, c := range q.Chunks {
		after[c.ID] = c.After
	}
	state := map[string]int{}
	var visit func(id string) bool
	visit = func(id string) bool {
		switch state[id] {
		case 1:
			return false
		case 2:
			return true
		}
		state[id] = 1
		for _, a := range after[id] {
			if !visit(a) {
				return false
			}
		}
		state[id] = 2
		return true
	}
	for _, c := range q.Chunks {
		if !visit(c.ID) {
			return fmt.Errorf("questboard: quest %q: chunk order is circular at %q", q.ID, c.ID)
		}
	}
	return nil
}
