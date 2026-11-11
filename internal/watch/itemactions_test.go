package watch

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

// The item action menu (docs/inventory-actions.md): DOM goldens of its states in node for a Paladin and a Sorceress
// view, plus patterns that keep it from moving the page or breaking the page CSP.
func TestItemActionsDOMGoldens(t *testing.T) {
	runNode(t, "testdata/itemactions/act_dom.js", "web/inventory-actions.js")
}

func TestItemActionsJSNeverScrollsOrInjects(t *testing.T) {
	b, err := webFS.ReadFile("web/inventory-actions.js")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	for _, re := range []string{`scrollIntoView`, `scrollTo\(`, `scrollBy\(`, `\.scrollTop\s*=`, `innerHTML|outerHTML|insertAdjacentHTML`, `createElement\(\s*['"]style`, `cssText`, `setInterval`, `\.focus\(\)`} {
		if regexp.MustCompile(re).MatchString(src) {
			t.Errorf("web/inventory-actions.js matches %q", re)
		}
	}
	// every focus() call passes preventScroll
	for _, m := range regexp.MustCompile(`\.focus\([^)]*\)`).FindAllString(src, -1) {
		if !strings.Contains(m, "preventScroll: true") {
			t.Errorf("focus call without preventScroll: %s", m)
		}
	}
	for _, want := range []string{"'Escape'", "ArrowDown", `role', 'menu'`, "menuitem", "aria-disabled", "d2:inventory", "style.transform", "/api/inventory/trash", "/api/inventory/cancel", "confirm: true", "EXTENSION POINT", "data-slot", "/api/inventory/identify", "/api/inventory/buyid", "/api/inventory/cain", "opens when Cain is rescued", "d-locked", "aria-label"} {
		if !strings.Contains(src, want) {
			t.Errorf("web/inventory-actions.js lacks %q", want)
		}
	}
	// the errand strip never rebuilds on a poll (a typed count must survive) and the page has its mount point
	if !strings.Contains(invPage, `id="errands"`) {
		t.Error("inventory page lacks the errands mount point")
	}
	css, _ := webFS.ReadFile("web/inventory-actions.css")
	for _, want := range []string{"@layer components", "position: fixed", `data-mode="sheet"`, "var(--d-focus)", "var(--d-surface-2)"} {
		if !strings.Contains(string(css), want) {
			t.Errorf("web/inventory-actions.css lacks %q", want)
		}
	}
	if regexp.MustCompile(`@media|!important|#[0-9a-fA-F]{3,6}\b`).Match(css) {
		t.Error("inventory-actions.css: no media query, !important or colour literal")
	}
}

func TestItemActionsRoutesAndPage(t *testing.T) {
	for _, want := range []string{`href="/watch/inventory-actions.css"`, `src="/watch/inventory-actions.js"`} {
		if !strings.Contains(invPage, want) {
			t.Errorf("inventory page lacks %s", want)
		}
	}
	h := NewInvAPI().Handler()
	for path, ctype := range map[string]string{"/watch/inventory-actions.js": "text/javascript", "/watch/inventory-actions.css": "text/css"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), ctype) || w.Body.Len() < 200 {
			t.Errorf("%s: %d %q", path, w.Code, w.Header().Get("Content-Type"))
		}
	}
}
