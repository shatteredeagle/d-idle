package watch

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// The item tooltip (docs/inventory-tooltip.md): DOM goldens of its states in node, plus patterns that keep it from
// moving the page or breaking the page CSP.
func TestItemTipDOMGoldens(t *testing.T) {
	runNode(t, "testdata/itemtip/tip_dom.js", "web/inventory-tip.js")
}

// Like TestPollingJSFocusKeepsScroll: the tooltip never scrolls. It takes no focus at all, uses no scroll calls, is
// placed by a transform on a fixed element, and builds its text with textContent (no HTML sink, no style element).
func TestItemTipJSNeverScrollsOrInjects(t *testing.T) {
	b, err := webFS.ReadFile("web/inventory-tip.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, re := range []string{`\.focus\(`, `scrollIntoView`, `scrollTo\(`, `scrollBy\(`, `\.scrollTop\s*=`, `innerHTML|outerHTML|insertAdjacentHTML`, `createElement\(\s*['"]style`, `cssText`, `setInterval|setTimeout|fetch\(`} {
		if regexp.MustCompile(re).MatchString(src) {
			t.Errorf("web/inventory-tip.js matches %q", re)
		}
	}
	for _, want := range []string{"passive: true", "'Escape'", "aria-describedby", "role", "tooltip", "pointerType", "d2:inventory", "style.transform"} {
		if !strings.Contains(src, want) {
			t.Errorf("web/inventory-tip.js lacks %q", want)
		}
	}
	css, _ := webFS.ReadFile("web/inventory-tip.css")
	for _, want := range []string{"@layer components", "position: fixed", "pointer-events: none", "var(--d-tip-bg)", "text-align: center", "var(--d-font-display)"} {
		if !strings.Contains(string(css), want) {
			t.Errorf("web/inventory-tip.css lacks %q", want)
		}
	}
	// the page poll hands the view over by event and does not know the tooltip
	inv, _ := webFS.ReadFile("web/inventory.js")
	if !strings.Contains(string(inv), "d2:inventory") {
		t.Error("web/inventory.js must hand the view to the tooltip with the d2:inventory event")
	}
}

// The page loads the tooltip files and the routes serve them under the page CSP (style-src 'self').
func TestItemTipRoutes(t *testing.T) {
	for _, want := range []string{`href="/watch/inventory-tip.css"`, `src="/watch/inventory-tip.js"`} {
		if !strings.Contains(invPage, want) {
			t.Errorf("inventory page lacks %s", want)
		}
	}
	h := NewInvAPI().Handler()
	for path, ctype := range map[string]string{"/watch/inventory-tip.js": "text/javascript", "/watch/inventory-tip.css": "text/css"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), ctype) || w.Body.Len() < 200 {
			t.Errorf("%s: %d %q", path, w.Code, w.Header().Get("Content-Type"))
		}
		if !strings.Contains(w.Header().Get("Content-Security-Policy"), "style-src 'self'") {
			t.Errorf("%s: CSP %q", path, w.Header().Get("Content-Security-Policy"))
		}
	}
}
