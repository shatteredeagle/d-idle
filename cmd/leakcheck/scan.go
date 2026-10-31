package main

import (
	"regexp"
	"strings"
)

// Hit is one private-looking string in one line. Text is empty for the secret classes (never echo a secret).
type Hit struct {
	Path  string `json:"path"`
	Line  int    `json:"line"`
	Class string `json:"class"`
	Text  string `json:"text,omitempty"`
}

type rule struct {
	class string
	rx    *regexp.Regexp
	quiet bool // do not echo the match
}

// RE2 has no lookbehind, so the letter-boundary rule (PUBFIX: letters only, so orion_box and Jade_x match) is
// written with consumed boundary groups; the name itself is capture group 1.
var rules = []rule{
	{"domain", regexp.MustCompile(`(?i)birbhub`), false},
	{"machine", regexp.MustCompile(`(?i)(orion|herdr|forgejo|unraid|omlette|macmini)|(?:^|[^a-z])(venus|jade|onyx|tower)(?:[^a-z]|$)`), false},
	{"person", regexp.MustCompile(`(?i)jesse|desmond`), false},
	{"ip", regexp.MustCompile(`(?:^|[^0-9.])((?:10(?:\.\d{1,3}){3}|192\.168(?:\.\d{1,3}){2}|172\.(?:1[6-9]|2\d|3[01])(?:\.\d{1,3}){2}))(?:[^0-9]|\.[^0-9]|\.?$)`), false},
	{"homepath", regexp.MustCompile(`/home/[A-Za-z_][\w.-]*|/Users/`), false},
	{"email", regexp.MustCompile(`[A-Za-z0-9._%+-]+@[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+`), false},
	{"secret", regexp.MustCompile(`(?i)authorization|bearer\s+\S{8,}|BEGIN [A-Z ]*PRIVATE KEY|\bgh[pousr]_[A-Za-z0-9]{20,}|\b[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\b`), true},
}

var (
	// Game areas, quests and items that contain a blocked word (the Forgotten Tower, the Jade Figurine).
	gamePhrases = regexp.MustCompile(`(?i)forgotten[ _]tower|tower cellar|tamoe|jade figurine|"jade",|` +
		`road and the tower|in the tower|blocked tower|a closed tower|the tower (?:verdict|below|skipped|guard|needs|is closed|\()|` +
		`tower(?: guard| verdict| corpse| should| not skipped| closed| kit| %| \+| \d| \(|: a merc|,| &&| :=)|` +
		`(?:and|with) tower (?:open|done)|tower to ex |tower cellar|with a closed tower`)
	tokenRx = regexp.MustCompile(`[A-Za-z0-9+/_-]{32,}={0,2}`)
)

// randomToken: long, with digits and both cases, so identifiers and plain paths do not count.
func randomToken(s string) bool {
	var lo, up, dg int
	for _, c := range s {
		switch {
		case c >= 'a' && c <= 'z':
			lo++
		case c >= 'A' && c <= 'Z':
			up++
		case c >= '0' && c <= '9':
			dg++
		}
	}
	return lo > 0 && up > 2 && dg > 2 && !strings.Contains(s, "/")
}

// blank replaces every allow-listed string and game phrase with spaces of the same length.
func blank(line string, allowStr []string) string {
	for _, a := range allowStr {
		if strings.Contains(line, a) {
			line = strings.ReplaceAll(line, a, strings.Repeat(" ", len(a)))
		}
	}
	return gamePhrases.ReplaceAllStringFunc(line, func(m string) string { return strings.Repeat(" ", len(m)) })
}

func scanText(path, text string, allowStr []string) []Hit {
	var out []Hit
	for i, line := range strings.Split(text, "\n") {
		work := blank(line, allowStr)
		for _, r := range rules {
			for _, m := range r.rx.FindAllStringSubmatch(work, -1) {
				h := Hit{Path: path, Line: i + 1, Class: r.class}
				if !r.quiet {
					h.Text = firstGroup(m)
				}
				out = append(out, h)
			}
		}
		for _, m := range tokenRx.FindAllString(work, -1) {
			if randomToken(m) {
				out = append(out, Hit{Path: path, Line: i + 1, Class: "secret"})
			}
		}
	}
	return out
}

func firstGroup(m []string) string {
	for _, g := range m[1:] {
		if g != "" {
			return g
		}
	}
	return m[0]
}
