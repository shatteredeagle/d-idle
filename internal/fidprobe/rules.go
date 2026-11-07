package fidprobe

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"
)

//go:embed rules.json
var rulesJSON []byte

//go:embed known.json
var knownJSON []byte

//go:embed scenarios.json
var scenariosJSON []byte

// Rule decides how two readable values of a key compare. Sets always compare by equality.
type Rule struct {
	Match string  `json:"match"` // path.Match pattern over the key
	Rule  string  `json:"rule"`  // exact | rel
	Tol   float64 `json:"tol"`   // relative tolerance for rel
}

// Known is a registered, explained deviation. It only turns a MISMATCH into KNOWN; it never hides an
// UNREADABLE, and a new entry needs the matrix row and a reason (design section 6).
type Known struct {
	Match  string `json:"match"`
	Row    string `json:"row"`
	Reason string `json:"reason"`
}

// Rules is the data the engine runs on.
type Rules struct {
	Tolerance     []Rule            `json:"tolerance"`
	UnreadableMap map[string]string `json:"unreadable_map"`
	// Reader scope (run 1, fidprobe.md): skills.base lists only the class's own skills with points
	// (game.exe also lists Attack, Kick and the like); stats.vector skips zero entries and the ids in
	// StatsScopeOnly, which only one reader lists (S: reader scope, not a game difference). Those ids go
	// to the group statsx, which no scenario compares.
	ClassSkillFirst []int   `json:"class_skill_first"`
	ClassSkillCount int     `json:"class_skill_count"`
	StatsScopeOnly  []int   `json:"stats_scope_only"`
	Known           []Known `json:"-"`
}

// DefaultRules loads the embedded tables.
func DefaultRules() (*Rules, error) {
	var r Rules
	if err := json.Unmarshal(rulesJSON, &r); err != nil {
		return nil, fmt.Errorf("fidprobe: rules.json: %w", err)
	}
	if err := json.Unmarshal(knownJSON, &r.Known); err != nil {
		return nil, fmt.Errorf("fidprobe: known.json: %w", err)
	}
	return &r, r.validate()
}

func (r *Rules) validate() error {
	for _, t := range r.Tolerance {
		if _, err := path.Match(t.Match, "x"); err != nil {
			return fmt.Errorf("fidprobe: bad pattern %q: %w", t.Match, err)
		}
		if t.Rule != "exact" && t.Rule != "rel" {
			return fmt.Errorf("fidprobe: rule %q for %q: want exact or rel", t.Rule, t.Match)
		}
	}
	for _, k := range r.Known {
		if k.Row == "" || k.Reason == "" {
			return fmt.Errorf("fidprobe: known %q needs a matrix row and a reason", k.Match)
		}
	}
	return nil
}

func (r *Rules) ruleFor(key string) Rule {
	for _, t := range r.Tolerance {
		if ok, _ := path.Match(t.Match, key); ok {
			return t
		}
	}
	return Rule{Match: "*", Rule: "exact"}
}

func (r *Rules) knownFor(key string) (Known, bool) {
	for _, k := range r.Known {
		if ok, _ := path.Match(k.Match, key); ok {
			return k, true
		}
	}
	return Known{}, false
}

// applyUnreadable voids the sheet fields that a producer reported as unknown. A producer name matches a
// map entry when it equals it or starts with it plus a dot.
func applyUnreadable(s *Sheet, unknown map[string]string) {
	if len(unknown) == 0 {
		return
	}
	rules, err := DefaultRules()
	if err != nil {
		return
	}
	names := make([]string, 0, len(unknown))
	for n := range unknown {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		for ent, prefix := range rules.UnreadableMap {
			if n == ent || strings.HasPrefix(n, ent+".") {
				for k := range s.Fields {
					if strings.HasPrefix(k, prefix) {
						s.Unreadable(k, n+": "+unknown[n])
					}
				}
			}
		}
	}
}

// Scenario is one catalogue entry. Saves are referenced by Role: the tool takes a directory holding
// <role>.d2s; no save or game data is in the repo.
type Scenario struct {
	ID         int      `json:"id"`
	Name       string   `json:"name"`
	Role       string   `json:"role"`
	Class      string   `json:"class"`
	Difficulty int      `json:"difficulty"`
	Slice1     bool     `json:"slice1"`
	Groups     []string `json:"groups"`
	Purpose    string   `json:"purpose"`
}

// Scenarios returns the 14-entry catalogue.
func Scenarios() ([]Scenario, error) {
	var s []Scenario
	if err := json.Unmarshal(scenariosJSON, &s); err != nil {
		return nil, fmt.Errorf("fidprobe: scenarios.json: %w", err)
	}
	return s, nil
}

// FindScenario resolves a numeric id or a name.
func FindScenario(ref string) (Scenario, error) {
	all, err := Scenarios()
	if err != nil {
		return Scenario{}, err
	}
	for _, s := range all {
		if ref == s.Name || ref == fmt.Sprint(s.ID) {
			return s, nil
		}
	}
	return Scenario{}, fmt.Errorf("fidprobe: unknown scenario %q", ref)
}

// InClassSkills reports whether skill id belongs to the 30 skills of class.
func (r *Rules) InClassSkills(class, id int) bool {
	if class < 0 || class >= len(r.ClassSkillFirst) {
		return false
	}
	return id >= r.ClassSkillFirst[class] && id < r.ClassSkillFirst[class]+r.ClassSkillCount
}

func (r *Rules) scopeOnly(id int) bool {
	for _, x := range r.StatsScopeOnly {
		if x == id {
			return true
		}
	}
	return false
}
