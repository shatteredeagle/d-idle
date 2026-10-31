package main

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
)

// Manifest classes.
const (
	Public = "public"
	Priv   = "private"
	Scrub  = "scrub-needed"
)

type ManifestRule struct {
	Glob    string `json:"glob"`
	Class   string `json:"class"`
	Replace string `json:"replace,omitempty"` // for scrub-needed: what to do (see docs/process/public-split.md)
	Note    string `json:"note,omitempty"`
	rx      *regexp.Regexp
}

type Manifest struct {
	Rules []*ManifestRule `json:"rules"`
}

func loadManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	for _, r := range m.Rules {
		r.rx = globRegexp(r.Glob)
	}
	return &m, nil
}

// Lookup: first matching rule wins; no match means public.
func (m *Manifest) Lookup(path string) *ManifestRule {
	if m == nil {
		return nil
	}
	for _, r := range m.Rules {
		if r.rx.MatchString(path) {
			return r
		}
	}
	return nil
}

// globRegexp: `**/` any directories, `**` anything, `*` within one segment, `?` one char.
func globRegexp(g string) *regexp.Regexp {
	var sb strings.Builder
	sb.WriteString("^")
	for i := 0; i < len(g); i++ {
		switch c := g[i]; {
		case c == '*' && strings.HasPrefix(g[i:], "**/"):
			sb.WriteString("(?:.*/)?")
			i += 2
		case c == '*' && strings.HasPrefix(g[i:], "**"):
			sb.WriteString(".*")
			i++
		case c == '*':
			sb.WriteString("[^/]*")
		case c == '?':
			sb.WriteString("[^/]")
		default:
			sb.WriteString(regexp.QuoteMeta(string(c)))
		}
	}
	sb.WriteString("$")
	return regexp.MustCompile(sb.String())
}

// Allow is the parsed leakcheck-allow.txt: `path <glob>` skips a file, `str <text>` blanks that exact text.
type Allow struct {
	Paths []*regexp.Regexp
	Strs  []string
}

func parseAllow(text string) *Allow {
	a := &Allow{}
	for _, l := range strings.Split(text, "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "#") {
			continue
		}
		kind, val, ok := strings.Cut(l, " ")
		val = strings.TrimSpace(val)
		if !ok || val == "" {
			continue
		}
		switch kind {
		case "path":
			a.Paths = append(a.Paths, globRegexp(val))
		case "str":
			a.Strs = append(a.Strs, val)
		}
	}
	return a
}

func (a *Allow) skipPath(p string) bool {
	for _, rx := range a.Paths {
		if rx.MatchString(p) {
			return true
		}
	}
	return false
}
