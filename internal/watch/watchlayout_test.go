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

// The live video sits at the top of /watch: before the page tabs and before every card of <main>, with the minimap
// beside it (same row), and there is no separate Game tab any more.
func TestWatchPageVideoIsInlineAtTheTop(t *testing.T) {
	html := watchPage(t)
	order := []string{`id="gamepane"`, `id="video-full"`, `id="video-collapse"`, `id="shot"`, `id="sensespanel"`, `<nav class="views"`, `<main id="main"`, `id="now"`}
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
	if strings.Index(html, `id="gamerow"`) > strings.Index(html, `id="sensespanel"`) {
		t.Error("the minimap must sit inside the video row")
	}
	for _, bad := range []string{`data-view="game"`, `id="tab-game"`} {
		if strings.Contains(html, bad) {
			t.Errorf("old Game tab left: %s", bad)
		}
	}
	// every id is unique and a single stream image exists
	ids := map[string]bool{}
	for _, m := range regexp.MustCompile(` id="([^"]+)"`).FindAllStringSubmatch(html, -1) {
		if ids[m[1]] {
			t.Errorf("duplicate id %s", m[1])
		}
		ids[m[1]] = true
	}
	if strings.Count(html, `<img id="shot"`) != 1 || strings.Contains(html, `src="/stream`) {
		t.Error("expected one <img id=shot> whose src is set by app.js")
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

// The reconnect and collapse logic lives in app.js; run the real file against a stub DOM in node when it is present.
func TestAppJSVideoReconnectAndCollapse(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	src, err := os.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(src), "function video(url)") {
		t.Fatal("video() missing")
	}
	cmd := exec.Command(node, "testdata/video_logic.js", "web/app.js")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

func TestSensesJSStitchedWorldAndNeighbourCache(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	if out, err := exec.Command(node, "testdata/senses_logic.js", "web/senses.js").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// Unplaced cards must not fall into single columns of the 12-column grid (they were ~90 px wide), and the video
// is about half size with the minimap beside it.
func TestWatchCSSPanelsAndVideoSize(t *testing.T) {
	b, err := os.ReadFile("web/app.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	for _, want := range []string{":where(main) > .card { grid-column: span 6; }", ".gamerow #frame { flex: 0 1 440px;", "max-width: 699px) { .gamerow { flex-direction", "flex-wrap: nowrap; }", ".sensespanel { flex: 0 0 auto; width: 338px;", ".videotop { width: calc(100% - 32px); max-width: 840px;"} {
		if !strings.Contains(css, want) {
			t.Errorf("app.css lacks %q", want)
		}
	}
	if strings.Index(css, ":where(main) > .card") > strings.Index(css, ".rules { grid-column: 1 / 6; }") {
		t.Error("the default two-per-row rule must come before the placed cards")
	}
}
