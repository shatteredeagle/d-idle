package questspec

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
)

// Finding is one violated rule. Rule is 1..6 as numbered in the README.
type Finding struct {
	Rule int
	Step string
	Fact string
	Msg  string
	RC   string // documented root cause the finding matches, if the caps note names one
}

func (f Finding) String() string {
	rc := ""
	if f.RC != "" {
		rc = " [" + f.RC + "]"
	}
	return fmt.Sprintf("rule %d step %s%s: %s", f.Rule, f.Step, rc, f.Msg)
}

var rcRE = regexp.MustCompile(`^(RC\d+[a-z]?):`)

func rcOf(note string) string {
	if m := rcRE.FindStringSubmatch(note); m != nil {
		return m[1]
	}
	return ""
}

func isPersistent(p Pred) bool { return !transient[p.Name] }

// Check runs the ten static rules and returns findings in a stable order.
func Check(c *Chain, caps *Caps) []Finding {
	var out []Finding
	add := func(rule int, s, fact, note string, f string, a ...any) {
		out = append(out, Finding{Rule: rule, Step: s, Fact: fact, Msg: fmt.Sprintf(f, a...), RC: rcOf(note)})
	}

	// Rule 1: every fact has a reader on the backend. A disjunction needs one readable alternative.
	for _, s := range c.Steps {
		for _, grp := range []struct {
			role string
			cl   []Clause
		}{{"pre", s.PreClauses()}, {"post", s.PostClauses()}} {
			for _, cl := range grp.cl {
				if clauseReadable(caps, cl) || (len(cl) == 1 && cl[0].Name == "in_area") {
					continue
				}
				note := ""
				if r, ok := caps.reader(cl[0]); ok {
					note = r.Note
				}
				if v := caps.missingVerb(cl[0]); v != "" {
					add(1, s.ID, cl.String(), note, "%s %s needs the verb %q, which %s lacks (%s)", grp.role, cl, v, caps.Backend, noteOr(note, "no note"))
					continue
				}
				add(1, s.ID, cl.String(), note, "%s %s has no reader on %s (%s)", grp.role, cl, caps.Backend, noteOr(note, "no entry"))
			}
		}
	}

	// Rule 2: every pre is established by an earlier post or is initial.
	var have []Clause
	have = append(have, c.InitialClauses()...)
	for _, s := range c.Steps {
		for _, pre := range s.PreClauses() {
			if len(pre) == 1 && pre[0].Name == "in_area" {
				continue // the executor travels; not an achieved fact
			}
			ok := false
			for _, h := range have {
				if (Clause{h[0]}).Implies(pre) { // the step guarantees its first alternative
					ok = true
				}
			}
			if !ok {
				add(2, s.ID, pre.String(), "", "pre %s is neither initial nor established by an earlier post", pre)
			}
		}
		have = append(have, s.PostClauses()...)
	}

	// Rule 3: completion is observable, so some post must be readable.
	for _, s := range c.Steps {
		if s.Idempotent {
			continue // the hero's position is read from the area, not from a fact
		}
		posts := s.PostClauses()
		if len(posts) == 0 {
			add(3, s.ID, "", "", "step has no post: its completion cannot be seen")
			continue
		}
		seen := false
		note := ""
		for _, cl := range posts {
			if clauseReadable(caps, cl) {
				seen = true
			} else if r, ok := caps.reader(cl[0]); ok && note == "" {
				note = r.Note
			}
		}
		if !seen {
			add(3, s.ID, "", note, "no post of this step is readable on %s: completion cannot be seen", caps.Backend)
		}
	}

	// Rule 4: a find_object that depends on presets appearing near the hero in a large area needs an
	// explore budget and a progress rule that the executor supports.
	for _, s := range c.Steps {
		if s.Act != "find_object" || s.Discover == "preset_always" {
			continue
		}
		area := stepArea(s)
		if c.Areas[area].Size != "large" {
			continue
		}
		switch {
		case s.Explore == nil:
			add(4, s.ID, "", "", "find_object in large area %s (%s) has no explore budget and progress rule", area, s.Discover)
		case !contains(caps.ProgressRules, s.Explore.Progress):
			add(4, s.ID, s.Explore.Progress, caps.ProgressNote, "progress rule %s is not supported by %s: exploration progress is lost to the failure counter (%s)", s.Explore.Progress, caps.Backend, noteOr(caps.ProgressNote, "no note"))
		}
	}

	// Rule 5: items by code, never by display name; and the backend must match by code.
	usesItem := false
	for _, s := range c.Steps {
		if s.Target.Kind == "item" {
			usesItem = true
			if s.Target.Name != "" || !itemCode.MatchString(s.Target.Code) {
				add(5, s.ID, s.Target.Name, "", "item must be referenced by a 3-4 char code, got name %q code %q", s.Target.Name, s.Target.Code)
			}
		}
		for _, grp := range [][]Clause{s.PreClauses(), s.PostClauses()} {
			for _, cl := range grp {
				for _, p := range cl {
					if p.Name == "has_item" || p.Name == "lacks_item" || p.Name == "ground_item" {
						usesItem = true
						if len(p.Args) != 1 || !itemCode.MatchString(p.Args[0]) {
							add(5, s.ID, p.String(), "", "%s: item argument is not a code", p)
						}
					}
				}
			}
		}
	}
	if usesItem && caps.ItemMatch != "code" {
		add(5, "*", "has_item", caps.ItemMatchNote, "%s matches items by display name, so a code cannot be seen (%s)", caps.Backend, noteOr(caps.ItemMatchNote, "no note"))
	}

	// Rule 6: resumable. After any prefix of steps the next step is derivable from facts alone.
	for _, s := range c.Steps {
		if s.Idempotent {
			continue
		}
		hasP := false
		for _, cl := range s.PostClauses() {
			if isPersistent(cl[0]) {
				hasP = true
			}
		}
		if !hasP {
			add(6, s.ID, "", "", "no persistent post: after a restart nothing says this step is done")
		}
		durable := false
		for _, cl := range s.PostClauses() {
			for _, p := range cl {
				if isPersistent(p) && caps.readable(p) && !caps.volatile(p) {
					durable = true
				}
			}
		}
		if hasP && !durable {
			add(6, s.ID, "", "", "every readable post of this step is a per-run observation on %s: after a restart nothing says it is done", caps.Backend)
		}
	}
	truth := map[string]bool{}
	for _, cl := range c.InitialClauses() {
		for _, p := range cl {
			truth[p.String()] = true
		}
	}
	for k := 0; k <= len(c.Steps); k++ {
		want := -1
		for j := k; j < len(c.Steps); j++ {
			if !c.Steps[j].Idempotent {
				want = j
				break
			}
		}
		if got := Derive(c, func(cl Clause) bool { return holdsAny(truth, cl) }); got != want {
			who := "done"
			if got >= 0 {
				who = c.Steps[got].ID
			}
			at := "start"
			if k > 0 {
				at = "after " + c.Steps[k-1].ID
			}
			add(6, stepID(c, k), "", "", "%s the next step derives as %s (index %d), want %d: hidden handler memory or a post that is not distinguishing", at, who, got, want)
		}
		if k < len(c.Steps) {
			s := c.Steps[k]
			for _, cl := range s.PostClauses() {
				truth[cl[0].String()] = true
			}
			for _, cl := range s.ConsumedPreds() {
				for _, p := range cl {
					delete(truth, p.String())
				}
			}
		}
	}
	// Rule 7: a target is named by class id or item code, never by display name.
	for _, s := range c.Steps {
		t := s.Target
		switch t.Kind {
		case "object", "npc":
			if t.Class <= 0 {
				add(7, s.ID, t.Name, "", "%s %q has no class id: a display name is matched by text, which differs by locale and table build", t.Kind, t.Name)
			}
			if t.Name != "" && !nameRE.MatchString(t.Name) {
				add(7, s.ID, t.Name, "", "label %q looks like a display name (want snake_case; the class id is the identity)", t.Name)
			}
		case "monster", "area":
			if !nameRE.MatchString(t.Name) {
				add(7, s.ID, t.Name, "", "%s %q is not a token", t.Kind, t.Name)
			}
		}
	}

	// Rule 8: every verb a step needs is answered by the backend.
	for _, s := range c.Steps {
		for _, v := range s.Verbs {
			if !contains(caps.Verbs, v) {
				add(8, s.ID, v, "", "step needs the verb %q, which %s does not answer", v, caps.Backend)
			}
		}
	}

	// Rule 9: the target can become known. Each discovery mode needs a knowledge source on the backend.
	for _, s := range c.Steps {
		need := map[string]string{"preset_always": "preset_list", "preset_when_near": "preset_near", "room_then_explore": "room_rows"}[s.Discover]
		if need != "" && s.Act != "travel" && s.Act != "wait" && !contains(caps.Knowledge, need) {
			add(9, s.ID, s.Discover, caps.KnowledgeNote, "discover %s needs the knowledge source %q, which %s does not offer (%s)", s.Discover, need, caps.Backend, noteOr(caps.KnowledgeNote, "no note"))
		}
	}

	// Rule 10: a loop has a lease. A retrying step declares how often it tries; an exploring step has its budget.
	for _, s := range c.Steps {
		if s.OnFail == "retry" && s.Tries <= 0 {
			add(10, s.ID, "", "", "on_fail retry without tries: the step may loop for ever inside budget_s")
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := stepIdx(c, out[i].Step), stepIdx(c, out[j].Step)
		if a != b {
			return a < b
		}
		return out[i].Rule < out[j].Rule
	})
	return out
}

// Derive returns the index of the first step whose persistent posts do not
// all hold, or -1 when the chain is complete. It reads facts only.
func Derive(c *Chain, holds func(Clause) bool) int {
	for i, s := range c.Steps {
		if s.Idempotent {
			continue
		}
		for _, cl := range s.PostClauses() {
			if isPersistent(cl[0]) && !holds(cl) {
				return i
			}
		}
	}
	return -1
}

func holdsAny(truth map[string]bool, cl Clause) bool {
	for _, p := range cl {
		if truth[p.String()] {
			return true
		}
	}
	return false
}

func clauseReadable(caps *Caps, cl Clause) bool {
	for _, p := range cl {
		if caps.readable(p) {
			return true
		}
	}
	return false
}

func stepArea(s Step) string {
	for _, cl := range s.PreClauses() {
		if len(cl) == 1 && cl[0].Name == "in_area" {
			return cl[0].Args[0]
		}
	}
	if s.Target.Kind == "area" {
		return s.Target.Name
	}
	return ""
}

func stepID(c *Chain, k int) string {
	if k < len(c.Steps) {
		return c.Steps[k].ID
	}
	return "*"
}

func stepIdx(c *Chain, id string) int {
	for i, s := range c.Steps {
		if s.ID == id {
			return i
		}
	}
	return len(c.Steps)
}

func contains(l []string, s string) bool {
	for _, x := range l {
		if x == s {
			return true
		}
	}
	return false
}

func noteOr(n, d string) string {
	if n == "" {
		return d
	}
	return strings.TrimSpace(n)
}
