package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	rePanelTag = regexp.MustCompile(`<[a-z]+\b[^>]*\bdata-panel="([a-z0-9-]+)"[^>]*>`)
	reState    = regexp.MustCompile(`\bdata-state="([a-z]+)"`)
)

// pagePanels are the panels a page dump must hold, ready, by file name prefix. The fixture viewer offers data for
// every one of them, so a missing or still-loading panel is a regression.
var pagePanels = map[string][]string{
	"watch-":     {"hero-status", "video", "map", "hero-vitals", "activity", "why", "gear", "tally"},
	"overlay-":   {"hero-status"},
	"inventory-": {"hero-status"},
}

// checkDir reads the --dump-dom files of a run and checks the contract on the rendered DOM: the shell is there, each
// expected panel exists and left the loading state, and no panel failed.
func checkDir(dir string) ([]string, error) {
	names, err := filepath.Glob(filepath.Join(dir, "*.html"))
	if err != nil || len(names) == 0 {
		return nil, fmt.Errorf("no DOM dumps in %q", dir)
	}
	sort.Strings(names)
	var bad []string
	for _, n := range names {
		b, err := os.ReadFile(n)
		if err != nil {
			return bad, err
		}
		bad = append(bad, checkDump(filepath.Base(n), string(b))...)
	}
	return bad, nil
}

func checkDump(name, html string) []string {
	var bad []string
	state := map[string]string{}
	for _, m := range rePanelTag.FindAllStringSubmatch(html, -1) {
		s := ""
		if st := reState.FindStringSubmatch(m[0]); st != nil {
			s = st[1]
		}
		state[m[1]] = s
		if s == "error" {
			bad = append(bad, fmt.Sprintf("%s: panel %s is in the error state", name, m[1]))
		}
	}
	for prefix, want := range pagePanels {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		if !strings.Contains(html, `class="d-shell"`) {
			bad = append(bad, name+": no d-shell")
		}
		for _, id := range want {
			switch s, ok := state[id]; {
			case !ok:
				bad = append(bad, fmt.Sprintf("%s: panel %s missing", name, id))
			case s == "loading":
				bad = append(bad, fmt.Sprintf("%s: panel %s still loading", name, id))
			}
		}
	}
	return bad
}
