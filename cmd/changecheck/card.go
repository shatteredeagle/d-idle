package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Card is one change folder: the header block of tasks.md plus what proposal.md says.
type Card struct {
	ID       string // folder name without the archive date prefix
	Dir      string
	Archived bool
	Fields   map[string][]string
	Why      string
	HasWhat  bool
}

func (c *Card) one(k string) string {
	if v := c.Fields[k]; len(v) > 0 {
		return v[0]
	}
	return ""
}

func (c *Card) Status() string      { return c.one("status") }
func (c *Card) Agent() string       { return c.one("agent") }
func (c *Card) Branch() string      { return c.one("branch") }
func (c *Card) Owns() []string      { return c.Fields["owns"] }
func (c *Card) Exclusive() []string { return c.Fields["exclusive"] }
func (c *Card) Gates() []string     { return c.Fields["gates"] }

// Active cards are the ones whose ownership and exclusive claims count.
func (c *Card) Active() bool {
	return !c.Archived && (c.Status() == "proposed" || c.Status() == "in-progress")
}

var (
	keyRe  = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*):\s*(.*)$`)
	itemRe = regexp.MustCompile(`^\s+-\s+(.*)$`)
)

// parseHeader reads the first ```yaml block: scalar keys, inline [a, b] lists and block lists.
// It is deliberately not a YAML parser; the format is documented in openspec/README.md.
func parseHeader(text string) (map[string][]string, error) {
	lines := strings.Split(text, "\n")
	start := -1
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "```yaml") {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("no yaml header block")
	}
	f := map[string][]string{}
	cur := ""
	for _, l := range lines[start:] {
		if strings.HasPrefix(strings.TrimSpace(l), "```") {
			return f, nil
		}
		if strings.TrimSpace(l) == "" || strings.HasPrefix(strings.TrimSpace(l), "#") {
			continue
		}
		if m := itemRe.FindStringSubmatch(l); m != nil && cur != "" {
			f[cur] = append(f[cur], unquote(m[1]))
			continue
		}
		if m := keyRe.FindStringSubmatch(l); m != nil {
			cur = m[1]
			f[cur] = nil
			v := stripComment(m[2])
			switch {
			case v == "":
			case strings.HasPrefix(v, "[") && strings.HasSuffix(v, "]"):
				for _, p := range strings.Split(v[1:len(v)-1], ",") {
					if p = strings.TrimSpace(p); p != "" {
						f[cur] = append(f[cur], unquote(p))
					}
				}
			default:
				f[cur] = []string{unquote(v)}
			}
		}
	}
	return nil, fmt.Errorf("unterminated yaml header block")
}

func stripComment(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, `"`) || strings.HasPrefix(s, "'") {
		if j := strings.LastIndex(s, s[:1]); j > 0 {
			return s[:j+1]
		}
		return s
	}
	if i := strings.Index(s, " #"); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

func unquote(s string) string {
	s = strings.TrimSpace(s)
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

var archiveRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}-(.+)$`)

// section returns the body of the first "## name" heading, trimmed.
func section(md, name string) (string, bool) {
	lines := strings.Split(md, "\n")
	in := false
	var b []string
	for _, l := range lines {
		if strings.HasPrefix(l, "## ") {
			if in {
				break
			}
			in = strings.TrimSpace(l[3:]) == name
			continue
		}
		if in {
			b = append(b, l)
		}
	}
	return strings.TrimSpace(strings.Join(b, "\n")), in || len(b) > 0
}

// loadCards reads every change folder; problems reading a folder become findings, not errors.
func loadCards(root string) (cards []*Card, fs []Finding) {
	base := filepath.Join(root, "openspec", "changes")
	for _, arch := range []bool{false, true} {
		dir := base
		if arch {
			dir = filepath.Join(base, "archive")
		}
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() || e.Name() == "archive" {
				continue
			}
			c := &Card{ID: e.Name(), Dir: filepath.Join(dir, e.Name()), Archived: arch}
			if m := archiveRe.FindStringSubmatch(e.Name()); arch && m != nil {
				c.ID = m[1]
			}
			rel := filepath.ToSlash(strings.TrimPrefix(c.Dir, root+string(filepath.Separator)))
			tasks, err := os.ReadFile(filepath.Join(c.Dir, "tasks.md"))
			if err != nil {
				fs = append(fs, Finding{"card", rel + ": tasks.md missing"})
			} else if c.Fields, err = parseHeader(string(tasks)); err != nil {
				fs = append(fs, Finding{"card", rel + "/tasks.md: " + err.Error()})
			}
			prop, err := os.ReadFile(filepath.Join(c.Dir, "proposal.md"))
			if err != nil {
				fs = append(fs, Finding{"card", rel + ": proposal.md missing"})
			} else {
				c.Why, _ = section(string(prop), "Why")
				w, _ := section(string(prop), "What Changes")
				c.HasWhat = w != ""
			}
			if c.Fields == nil {
				c.Fields = map[string][]string{}
			}
			cards = append(cards, c)
		}
	}
	sort.Slice(cards, func(i, j int) bool { return cards[i].ID < cards[j].ID })
	return cards, fs
}

var validStatus = map[string]bool{"proposed": true, "in-progress": true, "done": true, "deferred": true, "archived": true}
var idRe = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func checkCards(cards []*Card) (fs []Finding) {
	add := func(c *Card, f string, a ...any) {
		fs = append(fs, Finding{"card", c.ID + ": " + fmt.Sprintf(f, a...)})
	}
	for _, c := range cards {
		if !idRe.MatchString(c.ID) {
			add(c, "id is not lowercase kebab-case")
		}
		if len(c.Fields) == 0 {
			continue // already reported by loadCards
		}
		if got := c.one("change"); got != c.ID {
			add(c, "header change %q does not match the folder", got)
		}
		st := c.Status()
		if !validStatus[st] {
			add(c, "status %q is not proposed|in-progress|done|deferred|archived", st)
		}
		if c.Archived && st != "archived" {
			add(c, "in archive/ but status is %q", st)
		}
		if !c.Archived && st == "archived" {
			add(c, "status archived but the folder is not under archive/")
		}
		if !c.Archived && st == "done" {
			add(c, "status done: archive it (git mv to changes/archive/DATE-%s)", c.ID)
		}
		for _, k := range []string{"agent", "branch"} {
			if c.one(k) == "" {
				add(c, "header field %s is empty", k)
			}
		}
		if c.Active() && len(c.Owns()) == 0 {
			add(c, "owns: is empty")
		}
		if (st == "in-progress" || st == "done") && len(c.Gates()) == 0 {
			add(c, "gates: is empty (acceptance needs a number or a named test)")
		}
		if c.Archived && (c.one("merge") == "" || c.one("result") == "") {
			add(c, "archived card needs merge: and result:")
		}
		if n := len(c.Why); n < 50 || n > 1000 {
			add(c, "proposal ## Why has %d characters (needs 50 to 1000)", n)
		}
		if !c.HasWhat {
			add(c, "proposal has no ## What Changes")
		}
	}
	return fs
}
