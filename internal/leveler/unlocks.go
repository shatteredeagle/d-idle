package leveler

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	"d2lod-bot/internal/data"
	"d2lod-bot/internal/plandata"
)

// Skill unlocks (TASK-133, ADR 0018 item 6, docs/skill-unlocks.md). A per-class table says which skills the hero may
// not spend a point on until an event happened (a quest, a waypoint, a boss, an item, a clvl). The unlock is one more
// refusal of the Rule; points are never banked: a refused row flows to the next legal row exactly as it does for a
// skills.txt prerequisite, so an unlock can reorder the build but never saves a point.

// UnlockEntry locks one skill until its event.
type UnlockEntry struct {
	Skill string `json:"skill"`
	// Event is kind:id (quest, waypoint, boss, item, clvl); alternatives are joined with "|" (any one opens the skill).
	Event string `json:"event"`
	Codex string `json:"codex,omitempty"` // id of the Codex first that lights the icon (internal/decide firsts)
	Tag   string `json:"tag,omitempty"`   // S veteran estimate, U unsure, V verified
	Note  string `json:"note,omitempty"`
}

// Unlocks is the table of one class. A skill not listed is free (its level requirements still apply).
type Unlocks struct {
	Class    string        `json:"class"`
	Doc      string        `json:"doc,omitempty"`
	Unlocks  []UnlockEntry `json:"unlocks"`
	MustFree []string      `json:"mustFree,omitempty"` // skills the bot needs to survive; never listed in Unlocks
}

var eventKinds = map[string]bool{"quest": true, "waypoint": true, "boss": true, "item": true, "clvl": true}

func checkEvent(ev string) error {
	for _, alt := range strings.Split(ev, "|") {
		k, id, ok := strings.Cut(alt, ":")
		if !ok || id == "" || !eventKinds[k] {
			return fmt.Errorf("event %q: want kind:id with kind quest, waypoint, boss, item or clvl", alt)
		}
		if k == "clvl" {
			if n, err := strconv.Atoi(id); err != nil || n < 1 {
				return fmt.Errorf("event %q: clvl needs a level", alt)
			}
		}
	}
	return nil
}

// ParseUnlocks decodes and checks one table (syntax only; Validate checks the skill names against the tables).
func ParseUnlocks(raw []byte) (*Unlocks, error) {
	var u Unlocks
	if err := json.Unmarshal(raw, &u); err != nil {
		return nil, err
	}
	if u.Class == "" {
		return nil, fmt.Errorf("unlocks: no class")
	}
	seen := map[string]bool{}
	for _, e := range u.Unlocks {
		if e.Skill == "" {
			return nil, fmt.Errorf("unlocks %s: an entry has no skill", u.Class)
		}
		if err := checkEvent(e.Event); err != nil {
			return nil, fmt.Errorf("unlocks %s, skill %q: %w", u.Class, e.Skill, err)
		}
		if seen[strings.ToLower(e.Skill)] {
			return nil, fmt.Errorf("unlocks %s: skill %q listed twice", u.Class, e.Skill)
		}
		seen[strings.ToLower(e.Skill)] = true
	}
	for _, s := range u.MustFree {
		if seen[strings.ToLower(s)] {
			return nil, fmt.Errorf("unlocks %s: skill %q is both locked and mustFree", u.Class, s)
		}
	}
	return &u, nil
}

// LoadUnlocks loads unlocks-<class>.json from dir, else from the embedded copies. An unknown class is an error that
// names it (no fallback to another class).
func LoadUnlocks(class, dir string) (*Unlocks, error) {
	class = strings.ToLower(class)
	f := "unlocks-" + class + ".json"
	var raw []byte
	if dir != "" {
		b, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil && !os.IsNotExist(err) {
			return nil, err
		}
		raw = b
	}
	if raw == nil {
		b, err := fs.ReadFile(plandata.FS, f)
		if err != nil {
			return nil, fmt.Errorf("leveler: no skill unlock table for class %q (%s)", class, f)
		}
		raw = b
	}
	u, err := ParseUnlocks(raw)
	if err != nil {
		return nil, err
	}
	if u.Class != class {
		return nil, fmt.Errorf("leveler: %s is for class %q, not %q", f, u.Class, class)
	}
	return u, nil
}

// Validate checks every skill of the table (locked and mustFree) against the game tables: it must exist and belong to
// the class. The error names the skill.
func (u *Unlocks) Validate(db *data.DB) error {
	cc := db.ClassByName(u.Class)
	if cc == nil {
		return fmt.Errorf("unlocks: class %q is not in the tables", u.Class)
	}
	names := append([]string(nil), u.MustFree...)
	for _, e := range u.Unlocks {
		names = append(names, e.Skill)
	}
	for _, n := range names {
		s := db.SkillByName(n)
		if s == nil {
			return fmt.Errorf("unlocks %s: unknown skill %q", u.Class, n)
		}
		if s.Class != cc.Code {
			return fmt.Errorf("unlocks %s: skill %q belongs to class %q", u.Class, n, s.Class)
		}
	}
	return nil
}

// Progress is the per-hero record of fired unlock events, plus the one free-choice point. The runner keeps it in
// the same store the Codex reads and calls Fire from its quest, waypoint and boss events.
type Progress struct {
	mu     sync.Mutex
	events map[string]bool
	all    bool
	choice string
}

// NewProgress is a hero that has seen nothing yet.
func NewProgress() *Progress { return &Progress{events: map[string]bool{}} }

// AllMetProgress is the table of a measured run (ADR 0016 item 18): every event counts as met.
func AllMetProgress() *Progress { return &Progress{events: map[string]bool{}, all: true} }

// Fire records an event ("boss:bishibosh"). Unknown kinds are an error.
func (p *Progress) Fire(ev string) error {
	ev = strings.ToLower(ev)
	if err := checkEvent(ev); err != nil {
		return err
	}
	p.mu.Lock()
	p.events[ev] = true
	p.mu.Unlock()
	return nil
}

// Events lists the fired events (for the progress store).
func (p *Progress) Events() []string {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []string
	for e := range p.events {
		out = append(out, e)
	}
	return out
}

func (p *Progress) met(ev string, level int) bool {
	if p == nil {
		return false
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.all {
		return true
	}
	for _, alt := range strings.Split(strings.ToLower(ev), "|") {
		if k, id, _ := strings.Cut(alt, ":"); k == "clvl" {
			if n, _ := strconv.Atoi(id); level >= n {
				return true
			}
		} else if p.events[alt] {
			return true
		}
	}
	return false
}

// Unlocked reports whether the hero may spend a point on skill. A nil table locks nothing (the fidelity default).
func (u *Unlocks) Unlocked(skill string, p *Progress, level int) bool {
	if u == nil {
		return true
	}
	for _, e := range u.Unlocks {
		if strings.EqualFold(e.Skill, skill) {
			return p.met(e.Event, level)
		}
	}
	return true
}

// WrapRule adds the unlocked(name) refusal to a skills.txt rule (nil rule: only the unlock check).
func (u *Unlocks) WrapRule(rule Rule, p *Progress) Rule {
	if u == nil {
		return rule
	}
	return func(name string, level int, lv map[string]int) bool {
		if !u.Unlocked(name, p, level) {
			return false
		}
		return rule == nil || rule(name, level, lv)
	}
}

// FreeChoice is one skill the player may take with the single free-choice point, with what it costs.
type FreeChoice struct {
	Skill     string
	Displaces string // the skill the build table would have given this point to, "" when it had no legal row
	Event     string
}

// Consequence is the line shown next to the choice.
func (c FreeChoice) Consequence() string {
	if c.Displaces == "" {
		return fmt.Sprintf("%s costs no build point right now: the table has no legal row for it", c.Skill)
	}
	return fmt.Sprintf("%s takes the point the build gives %s; the build moves one point later", c.Skill, c.Displaces)
}

func (b *Build) inTable(skill string) bool {
	for _, r := range b.Rows {
		for _, s := range r.Skills {
			if strings.EqualFold(s.Skill, skill) {
				return true
			}
		}
	}
	return false
}

// FreeChoices lists the skills that are Codex-unlocked for this hero, legal by the skills.txt rule now, and that the
// build table skips (ADR 0018 item 6).
func (u *Unlocks) FreeChoices(b *Build, h Have, rule Rule, p *Progress) []FreeChoice {
	if u == nil {
		return nil
	}
	var base string
	for _, a := range b.Plan(withPoints(h, 1), u.WrapRule(rule, p), nil) {
		if a.Kind == KindSkill {
			base = a.Skill
			break
		}
	}
	levels := map[string]int{}
	for k, v := range h.Skills {
		levels[k] = v
	}
	var out []FreeChoice
	for _, e := range u.Unlocks {
		if b.inTable(e.Skill) || !u.Unlocked(e.Skill, p, h.Level) || (rule != nil && !rule(e.Skill, h.Level, levels)) {
			continue
		}
		out = append(out, FreeChoice{Skill: e.Skill, Displaces: base, Event: e.Event})
	}
	return out
}

func withPoints(h Have, skill int) Have {
	h.SkillPts = skill
	h.StatPts = 0
	return h
}

// ChooseFree records the player's one free-choice point. It fails when the point was taken already, or when skill
// is not one of FreeChoices.
func (u *Unlocks) ChooseFree(p *Progress, b *Build, h Have, rule Rule, skill string) error {
	p.mu.Lock()
	cur := p.choice
	p.mu.Unlock()
	if cur != "" {
		return fmt.Errorf("the free-choice point is already on %q", cur)
	}
	for _, c := range u.FreeChoices(b, h, rule, p) {
		if strings.EqualFold(c.Skill, skill) {
			p.mu.Lock()
			p.choice = c.Skill
			p.mu.Unlock()
			return nil
		}
	}
	return fmt.Errorf("%q is not a free-choice candidate (locked, not legal yet, or already in the build table)", skill)
}

// Choice is the skill picked with ChooseFree, "" when none.
func (p *Progress) Choice() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.choice
}

// Plan is Build.Plan with the unlock refusal and the free-choice point: while the chosen skill has no point yet and
// a point is unspent, that point goes to it first (after the stat actions, as Plan orders them).
func (u *Unlocks) Plan(b *Build, h Have, rule Rule, skip map[string]bool, p *Progress) []Action {
	wrapped := u.WrapRule(rule, p)
	ch := ""
	if p != nil {
		ch = p.Choice()
	}
	if ch == "" || h.Skills[ch] > 0 || h.SkillPts < 1 {
		return b.Plan(h, wrapped, skip)
	}
	levels := map[string]int{}
	for k, v := range h.Skills {
		levels[k] = v
	}
	if wrapped != nil && !wrapped(ch, h.Level, levels) {
		return b.Plan(h, wrapped, skip)
	}
	h2 := h
	h2.SkillPts--
	h2.Skills = map[string]int{}
	for k, v := range h.Skills {
		h2.Skills[k] = v
	}
	h2.Skills[ch]++
	acts := b.Plan(h2, wrapped, skip)
	i := 0
	for i < len(acts) && acts[i].Kind == KindStat {
		i++
	}
	out := append([]Action{}, acts[:i]...)
	out = append(out, Action{Kind: KindSkill, Skill: ch})
	return append(out, acts[i:]...)
}
