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
	"watch-":     {"hero-status", "hero-meters", "alerts-count", "hero-detail", "video", "map", "hero-vitals", "activity", "why", "gear", "tally"},
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
		// the browser's console next to the dump (uishots.sh): an uncaught exception in a page script is a failure
		if lg, err := os.ReadFile(strings.TrimSuffix(n, ".html") + ".log"); err == nil {
			bad = append(bad, checkConsole(filepath.Base(n), string(lg))...)
		}
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
	if strings.Contains(name, "-edit") {
		// every pre-shell card and the game picture have their own frame with a handle bar
		for _, id := range []string{"now", "queue", "orders-map", "thoughts", "journal", "rules", "video"} {
			if !strings.Contains(html, `class="d-editframe" data-for="`+id+`"`) {
				bad = append(bad, name+": no edit frame for "+id)
			}
		}
		if !strings.Contains(html, `data-aspect="4/3"`) {
			bad = append(bad, name+": the game picture is not aspect-locked")
		}
	}
	if strings.Contains(name, "-edit") && (!strings.Contains(html, `class="d-editbar"`) || !strings.Contains(html, `data-editcell="1"`) || !strings.Contains(html, `class="d-edithandle"`)) {
		bad = append(bad, name+": Edit layout did not start (no edit bar or no editable cells)")
	}
	if strings.Contains(name, "-editdrag") && !strings.Contains(html, `data-preview="map"`) {
		bad = append(bad, name+": no drag preview on the page")
	}
	// fit content: the game picture's rows are measured from its width (six columns in the default layout), whatever
	// span the layout file stores; a card taller than its picture by a row or more is the dead-area regression
	if want, ok := videoRows[strings.TrimSuffix(name, ".html")]; ok {
		m := regexp.MustCompile(`data-panel="video"[^>]*\bdata-rows="(\d+)"|\bdata-rows="(\d+)"[^>]*data-panel="video"`).FindStringSubmatch(html)
		got := 0
		if m != nil {
			fmt.Sscan(m[1]+m[2], &got)
		}
		if got < want-1 || got > want {
			bad = append(bad, fmt.Sprintf("%s: the video panel spans %d rows, its picture needs %d", name, got, want))
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

var reConsole = regexp.MustCompile(`CONSOLE[:(]\d+\)?\] "([^"]*)`) // Chrome writes CONSOLE:12] or CONSOLE(12)] by version

// checkConsole reports the uncaught errors a page logged (Chrome's --enable-logging lines).
func checkConsole(name, log string) []string {
	var bad []string
	for _, m := range reConsole.FindAllStringSubmatch(log, -1) {
		if strings.Contains(m[1], "Uncaught") {
			bad = append(bad, fmt.Sprintf("%s: console: %s", name, m[1]))
		}
	}
	return bad
}

// videoRows are the rows a six-column game picture needs (4:3 plus the card head) at the width of each default shot.
var videoRows = map[string]int{"watch-1280-dark": 9, "watch-1440-dark": 10, "watch-1920-dark": 13, "watch-1920full-dark": 13}
