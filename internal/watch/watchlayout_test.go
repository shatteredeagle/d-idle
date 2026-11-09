package watch

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"testing"
)

func watchPage(t *testing.T) string {
	t.Helper()
	r := newRig(t, "sorceress")
	w := httptest.NewRecorder()
	r.sim.Hub.ViewHandler().ServeHTTP(w, httptest.NewRequest("GET", "/watch", nil))
	if w.Code != 200 {
		t.Fatalf("/watch: %d", w.Code)
	}
	return w.Body.String()
}

// The page is the shell (docs/ui-design-system.md section 4): the status band, the stage band (video and map, placed
// by shell.js from the manifests) and the grid band whose compatibility slot holds the pre-shell cards. The old
// header, video card and minimap markup are gone; the gear strip still finds header.top (gear.js anchors on it).
func TestWatchPageIsTheShell(t *testing.T) {
	html := watchPage(t)
	order := []string{`class="d-shell"`, `<header class="top d-band-status" data-band="status">`, `id="btn-theme"`, `data-band="stage"`, `data-band="grid"`,
		`class="d-compat"`, `<nav class="views"`, `<main id="main"`, `id="now"`, `src="/watch/shell.js"`, `src="/watch/app.js"`}
	last := -1
	for _, k := range order {
		i := strings.Index(html, k)
		if i < 0 {
			t.Fatalf("missing %s", k)
		}
		if i < last {
			t.Errorf("%s comes before the element before it", k)
		}
		last = i
	}
	for _, gone := range []string{`id="gamepane"`, `id="shot"`, `id="sensespanel"`, `id="whyline"`, `id="state"`, `id="uptime"`, `data-view="game"`, `/watch/senses.js`, `data-panel=`} {
		if strings.Contains(html, gone) {
			t.Errorf("pre-shell markup left: %s", gone)
		}
	}
	if !strings.Contains(html, `<script type="module" src="/watch/shell.js">`) {
		t.Error("shell.js must load as a module")
	}
	ids := map[string]bool{}
	for _, m := range regexp.MustCompile(` id="([^"]+)"`).FindAllStringSubmatch(html, -1) {
		if ids[m[1]] {
			t.Errorf("duplicate id %s", m[1])
		}
		ids[m[1]] = true
	}
}

func TestOldVideoPagesRedirectToWatch(t *testing.T) {
	r := newRig(t, "sorceress")
	for _, p := range []string{"/watch/video", "/watch/game"} {
		w := httptest.NewRecorder()
		r.sim.Hub.ViewHandler().ServeHTTP(w, httptest.NewRequest("GET", p+"?t=tok&live=1", nil))
		if w.Code != http.StatusFound || w.Header().Get("Location") != "/watch?t=tok&live=1" {
			t.Errorf("%s: %d %q", p, w.Code, w.Header().Get("Location"))
		}
	}
}

func runNode(t *testing.T, script, file string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	if out, err := exec.Command(node, script, file).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// The reconnect and collapse logic of the video panel, run against a stub DOM in node when it is present.
func TestVideoPanelReconnectAndCollapse(t *testing.T) {
	runNode(t, "testdata/video_panel.js", "web/panels/video/panel.js")
}

func TestMapPanelStitchedWorldAndNeighbourCache(t *testing.T) {
	runNode(t, "testdata/map_panel.js", "web/panels/map/panel.js")
}

// Unplaced cards must not fall into single columns of the 12-column grid (they were ~90 px wide). The stage band
// keeps the picture a glance: at most three columns, centred.
func TestWatchCSSPanelsAndStageSize(t *testing.T) {
	b, err := os.ReadFile("web/app.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	if !strings.Contains(css, ":where(main) > .card { grid-column: span 6; }") {
		t.Error("app.css lacks the two-per-row default")
	}
	if strings.Index(css, ":where(main) > .card") > strings.Index(css, ".rules { grid-column: 1 / 6; }") {
		t.Error("the default two-per-row rule must come before the placed cards")
	}
	for _, gone := range []string{".gamerow", ".videotop", ".sensespanel", ".whyline"} {
		if strings.Contains(css, gone) {
			t.Errorf("app.css still styles %s (moved into the shell)", gone)
		}
	}
	sh, _ := os.ReadFile("web/shell.css")
	for _, want := range []string{"--d-stage-cols: min(var(--d-cols), 3)", "max-inline-size: 72rem", "grid-column: span min(2, var(--d-cols))"} {
		if !strings.Contains(string(sh), want) {
			t.Errorf("shell.css lacks %q", want)
		}
	}
}

func TestWatchFoldableCards(t *testing.T) {
	js, err := os.ReadFile("web/unlockui.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"'rules'", "'lockedpane'", "'bestiarypane'", "'atlaspane'"} {
		if !strings.Contains(string(js), id) {
			t.Errorf("unlockui.js does not fold %s", id)
		}
	}
	css, _ := os.ReadFile("web/app.css")
	if !strings.Contains(string(css), ".card.folded >") {
		t.Error("app.css lacks the .folded rule")
	}
}
