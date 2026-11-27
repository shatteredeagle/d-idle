// Package questspec holds quest chains as checkable data (docs/questspec.md, roadmap R5): an ordered list of steps over
// named world facts, and a static checker that proves properties of a chain against a backend's capability set
// without a game. Nothing here runs in the bot: the executor still reads data/questboard; a chain is an additional
// validated view that names the same chunks and steps (CheckBoard).
package questspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Schema is the chain file schema; CapsSchema the backend capability file schema.
const (
	Schema     = "questspec/2"
	CapsSchema = "questcaps/2"
)

var (
	acts      = set("travel", "find_object", "interact", "pickup", "talk", "kill", "clear", "use_item", "transmute", "wait")
	onFails   = set("retry", "park", "abort")
	sizes     = set("small", "medium", "large")
	nameRE    = regexp.MustCompile(`^[a-z0-9_]+$`)
	itemCode  = regexp.MustCompile(`^[a-z0-9]{3,4}$`)
	spawnedRE = regexp.MustCompile(`^spawned_by\(([a-z0-9_]+)\)$`)
	// transient facts describe where the hero is, not what was achieved. A
	// restarted bot cannot use them to tell which step is done.
	transient = set("in_area")
)

func set(v ...string) map[string]bool {
	m := map[string]bool{}
	for _, s := range v {
		m[s] = true
	}
	return m
}

type Chain struct {
	Schema string `json:"schema"`
	ID     string `json:"id"`
	// Quest is the id of the data/questboard quest this chain describes (empty: a chain with no board row, e.g. a
	// test fixture); every Step.Chunk and Step.Board then has to exist there (CheckBoard).
	Quest string `json:"quest,omitempty"`
	// Latched lists the chunks whose last step has only volatile readers for its posts: the board latches them so
	// a restart does not lose them (rule 13).
	Latched []string        `json:"latched,omitempty"`
	Initial []string        `json:"initial"`
	Areas   map[string]Area `json:"areas"`
	Steps   []Step          `json:"steps"`
}

type Area struct {
	Size string `json:"size"`
}

// Target is what a step acts on. Identity is never a display name: an object or npc carries its class id (objects.txt,
// monstats hcIdx), an item its base code, a monster or area a snake_case token. Name is only a label for objects and
// npcs (rule 7 rejects a label that looks like a display name).
type Target struct {
	Kind  string `json:"kind"` // area | object | npc | monster | item
	Name  string `json:"name,omitempty"`
	Class int    `json:"class,omitempty"` // object and npc class id
	Code  string `json:"code,omitempty"`  // items only
}

type Explore struct {
	BudgetS  int    `json:"budget_s"`
	Progress string `json:"progress"` // named progress rule, e.g. coverage_grows
}

type Step struct {
	ID     string `json:"id"`
	Act    string `json:"act"`
	Target Target `json:"target"`
	// Chunk and Board link the step to data/questboard: the chunk it belongs to and the chunk's step string it
	// realises (several spec steps may realise one board step); Nodes are the task-layer node suffixes it covers.
	Chunk string   `json:"chunk,omitempty"`
	Board []string `json:"board,omitempty"`
	Nodes []string `json:"nodes,omitempty"`
	// Verbs are the backend verbs the step cannot work without (rule 8).
	Verbs []string `json:"verbs,omitempty"`
	// Tries is the lease of an on_fail "retry" step: the number of attempts before it parks (rule 10).
	Tries int `json:"tries,omitempty"`
	// Idempotent marks a pure movement step (every post is in_area): repeating it is harmless and it leaves no
	// durable fact, so resumption skips it (Derive) and the executor travels through the in_area preconditions.
	Idempotent bool     `json:"idempotent,omitempty"`
	Pre        []string `json:"pre"`
	Post       []string `json:"post"`
	Consumes   []string `json:"consumes,omitempty"` // atoms this step makes false
	Discover   string   `json:"discover"`
	BudgetS    int      `json:"budget_s"`
	OnFail     string   `json:"on_fail"`
	Explore    *Explore `json:"explore,omitempty"`
	// Exec is how the generic step executor runs the step (exec.go); absent: not run by it.
	Exec *Exec `json:"exec,omitempty"`
}

// Pred is one named fact with arguments, written name(a,b).
type Pred struct {
	Name string
	Args []string
}

func (p Pred) String() string { return p.Name + "(" + strings.Join(p.Args, ",") + ")" }

// Clause is a disjunction: any alternative being true satisfies it. As a
// post, the FIRST alternative is what the step itself produces; the others
// are later evidence that the step happened (the scroll is consumed, the
// deciphered key then proves it).
type Clause []Pred

func (c Clause) String() string {
	s := make([]string, len(c))
	for i, p := range c {
		s[i] = p.String()
	}
	return strings.Join(s, "|")
}

// Implies reports whether c being true makes d true (every alt of c is an alt of d).
func (c Clause) Implies(d Clause) bool {
	for _, p := range c {
		found := false
		for _, q := range d {
			if p.String() == q.String() {
				found = true
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func ParsePred(s string) (Pred, error) {
	s = strings.TrimSpace(s)
	i := strings.IndexByte(s, '(')
	if i <= 0 || !strings.HasSuffix(s, ")") {
		return Pred{}, fmt.Errorf("predicate %q: want name(arg,...)", s)
	}
	p := Pred{Name: s[:i]}
	if !nameRE.MatchString(p.Name) {
		return Pred{}, fmt.Errorf("predicate %q: bad name", s)
	}
	for _, a := range strings.Split(s[i+1:len(s)-1], ",") {
		a = strings.TrimSpace(a)
		if !nameRE.MatchString(a) {
			return Pred{}, fmt.Errorf("predicate %q: bad argument %q", s, a)
		}
		p.Args = append(p.Args, a)
	}
	return p, nil
}

func ParseClause(s string) (Clause, error) {
	var c Clause
	for _, part := range strings.Split(s, "|") {
		p, err := ParsePred(part)
		if err != nil {
			return nil, err
		}
		c = append(c, p)
	}
	return c, nil
}

func parseAll(in []string) ([]Clause, error) {
	var out []Clause
	for _, s := range in {
		c, err := ParseClause(s)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Clauses of a step, parsed. Load validated them, so errors cannot happen
// for a loaded chain; a hand-built chain should call Validate first.
func (s Step) PreClauses() []Clause  { c, _ := parseAll(s.Pre); return c }
func (s Step) PostClauses() []Clause { c, _ := parseAll(s.Post); return c }
func (s Step) ConsumedPreds() []Clause {
	c, _ := parseAll(s.Consumes)
	return c
}
func (c *Chain) InitialClauses() []Clause { x, _ := parseAll(c.Initial); return x }

// Load parses a chain strictly (unknown keys and trailing data are errors)
// and validates its structure.
func Load(data []byte) (*Chain, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var c Chain
	if err := dec.Decode(&c); err != nil {
		return nil, fmt.Errorf("questspec: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("questspec: trailing data")
	}
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return &c, nil
}

// Validate checks the structure only (enums, syntax, references). The
// semantic properties are Check's job.
func (c *Chain) Validate() error {
	var errs []string
	bad := func(f string, a ...any) { errs = append(errs, fmt.Sprintf(f, a...)) }
	if c.Schema != Schema {
		bad("schema %q, want %q", c.Schema, Schema)
	}
	if !nameRE.MatchString(c.ID) {
		bad("chain id %q", c.ID)
	}
	for a, v := range c.Areas {
		if !sizes[v.Size] {
			bad("area %s: size %q", a, v.Size)
		}
	}
	if _, err := parseAll(c.Initial); err != nil {
		bad("initial: %v", err)
	}
	ids := map[string]int{}
	for i, s := range c.Steps {
		if _, dup := ids[s.ID]; dup || !nameRE.MatchString(s.ID) {
			bad("step %d: id %q empty, malformed or duplicate", i, s.ID)
		}
		ids[s.ID] = i
	}
	for i, s := range c.Steps {
		w := func(f string, a ...any) { bad("step %s: "+f, append([]any{s.ID}, a...)...) }
		if !acts[s.Act] {
			w("act %q", s.Act)
		}
		if !onFails[s.OnFail] {
			w("on_fail %q", s.OnFail)
		}
		if s.Idempotent {
			for _, cl := range s.PostClauses() {
				if len(cl) != 1 || cl[0].Name != "in_area" {
					w("idempotent step may only post in_area, has %s", cl)
				}
			}
		}
		if s.BudgetS <= 0 {
			w("budget_s must be positive")
		}
		want := map[string]string{"travel": "area", "clear": "area", "find_object": "object", "interact": "object", "pickup": "item", "use_item": "item", "transmute": "item", "talk": "npc", "kill": "monster"}[s.Act]
		if want != "" && s.Target.Kind != want {
			w("act %s needs target kind %s, has %q", s.Act, want, s.Target.Kind)
		}
		if s.Target.Kind == "area" {
			if _, ok := c.Areas[s.Target.Name]; !ok {
				w("target area %q not declared", s.Target.Name)
			}
		}
		for _, f := range []struct {
			n string
			v []string
		}{{"pre", s.Pre}, {"post", s.Post}, {"consumes", s.Consumes}} {
			cl, err := parseAll(f.v)
			if err != nil {
				w("%s: %v", f.n, err)
			}
			for _, x := range cl {
				for _, p := range x {
					if p.Name == "in_area" {
						if len(p.Args) != 1 {
							w("in_area takes one area")
						} else if _, ok := c.Areas[p.Args[0]]; !ok {
							w("area %q not declared", p.Args[0])
						}
					}
				}
			}
		}
		switch d := s.Discover; {
		case d == "preset_always", d == "preset_when_near", d == "room_then_explore":
		case spawnedRE.MatchString(d):
			j, ok := ids[spawnedRE.FindStringSubmatch(d)[1]]
			if !ok || j >= i {
				w("discover %q must name an earlier step", d)
			}
		default:
			w("discover %q", d)
		}
		if s.Exec != nil {
			s.Exec.validate(s, bad)
		}
		if e := s.Explore; e != nil && (e.BudgetS <= 0 || !nameRE.MatchString(e.Progress)) {
			w("explore needs budget_s > 0 and a progress rule")
		}
	}
	if len(errs) > 0 {
		sort.Strings(errs)
		return fmt.Errorf("questspec: %s", strings.Join(errs, "; "))
	}
	return nil
}
