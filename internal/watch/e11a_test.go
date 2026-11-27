package watch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// E11a (docs/design/fable-review-2026-11-22/e11-deletion-plan.md): the pre-shell cards, their scripts and the old layout presets are gone.
// These tests fail while one of them is still in the tree or still named by shipped code, so a deleted file cannot creep back in.
var e11aDeleted = []string{
	"web/app.js", "web/unlockui.js", "web/alerts.js",
	"web/layouts/classic.json", "web/layouts/simple.json", "web/layouts/standard.json", "web/layouts/debug.json",
	"web/layouts/mockup.json", "web/layouts/live.json", "web/layouts/command.json",
}

func TestE11aDeletedFilesStayGone(t *testing.T) {
	for _, f := range e11aDeleted {
		if _, err := webFS.ReadFile(f); err == nil {
			t.Errorf("%s is still embedded: E11a deletes it", f)
		}
	}
}

// Nothing the viewer ships (HTML, JS, CSS, JSON, non-test Go) may still name a deleted file or route.
func TestE11aNoReferenceToDeleted(t *testing.T) {
	ban := regexp.MustCompile(`/watch/app\.js|web/app\.js|unlockui|UnlockUIRoute|AlertsUIRoute|/watch/alerts\.js|web/alerts\.js|layouts/(classic|simple|standard|debug|mockup|live|command)\b|id="(sheet|scrim)"|class="d-compat"|data-lid=`)
	check := func(path string) {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Error(err)
			return
		}
		if m := ban.Find(b); m != nil {
			t.Errorf("%s still names %q (deleted in E11a)", path, m)
		}
	}
	for _, g := range []string{"web/*.html", "web/*.js", "web/*.css", "web/panels/*/*", "*.go"} {
		files, _ := filepath.Glob(g)
		for _, f := range files {
			if strings.HasSuffix(f, "_test.go") {
				continue
			}
			check(f)
		}
	}
}

// E11a2: the alerts badge opens the Alerts section of the Settings sheet; nothing links to the deleted #alertspane.
func TestAlertsBadgeOpensSettingsAlerts(t *testing.T) {
	b, _ := webFS.ReadFile("web/panels/alerts-count/panel.js")
	if strings.Contains(string(b), "#alertspane") || !strings.Contains(string(b), "settings=alerts'") {
		t.Error("the alerts badge must link to ?settings=alerts")
	}
	nav, _ := webFS.ReadFile("web/nav.js")
	set, _ := webFS.ReadFile("web/settings.js")
	if !strings.Contains(string(nav), "get('settings')") || !strings.Contains(string(set), "env.section === 'alerts'") {
		t.Error("?settings=alerts no longer opens the Alerts section")
	}
}

// The stream layout serves the OBS overlay and the startup fallback (shell.js names it for both): it must exist, parse, and cover an overlay breakpoint or the grid.
func TestStreamLayoutServesOverlayAndFallback(t *testing.T) {
	sh, _ := webFS.ReadFile("web/shell.js")
	for _, want := range []string{"html.dataset.mode === 'overlay' ? 'stream' : 'cockpit'", "presetFile('stream')"} {
		if !strings.Contains(string(sh), want) {
			t.Errorf("shell.js lost %q", want)
		}
	}
	b, err := webFS.ReadFile("web/layouts/stream.json")
	if err != nil {
		t.Fatal(err)
	}
	l, err := ParseLayout(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, bp := range []string{"phone", "tablet", "laptop", "wide"} {
		if len(l.Breakpoints[bp].Panels) == 0 {
			t.Errorf("stream.json has no panels at %s (the overlay and the fallback use them)", bp)
		}
	}
	r := newRig(t, "sorceress")
	if w := r.get("/watch/layouts/stream.json", nil); w.Code != 200 {
		t.Errorf("stream.json is not served: %d", w.Code)
	}
}

// senses_units.js is not an orphan: the area-map panel draws units through window.sensesUnitsDraw, watch.html loads it and senses.go serves it.
func TestSensesUnitsScriptIsReferenced(t *testing.T) {
	am, _ := webFS.ReadFile("web/panels/area-map/panel.js")
	pg, _ := webFS.ReadFile("web/watch.html")
	if !strings.Contains(string(am), "sensesUnitsDraw") || !strings.Contains(string(pg), "senses_units.js") {
		t.Error("senses_units.js lost its caller: delete the file and its route together, not one of them")
	}
}

// The phone needs-you list grows (user decision 2026-11-23): no height cap, so no inner scroll box on the strip.
func TestNeedsYouListHasNoPhoneHeightCap(t *testing.T) {
	b, _ := webFS.ReadFile("web/utilities.css")
	if regexp.MustCompile(`needs-you"\][^{]*\.n-list[^{]*\{[^}]*max-block-size`).Match(b) {
		t.Error("the phone needs-you list is capped again: it must grow")
	}
}

// TASK-424 AC 5: the Layout menu switches presets without a request (the choice lives in this browser's storage and the
// page reloads); only Edit layout saves a copy. A source guard, because shell.js has no stub-DOM lane.
func TestChoosePresetMakesNoRequest(t *testing.T) {
	sh, _ := webFS.ReadFile("web/shell.js")
	src := string(sh)
	i := strings.Index(src, "function choosePreset(")
	if i < 0 {
		t.Fatal("shell.js lost choosePreset")
	}
	body := src[i:]
	if j := strings.Index(body, "\n}\n"); j > 0 {
		body = body[:j]
	}
	for _, bad := range []string{"fetch(", "postJSON", "XMLHttpRequest", "sendBeacon", "method:"} {
		if strings.Contains(body, bad) {
			t.Errorf("choosePreset makes a request (%s)", bad)
		}
	}
	if !strings.Contains(body, "pref(PREF, name)") || !strings.Contains(body, "location.reload()") {
		t.Error("choosePreset no longer stores the choice and reloads")
	}
}
