package watch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"d2lod-bot/internal/invdesk"
)

// The merc doll (docs/research/merc-gear.md, TASK-287): DOM goldens of its states in node.
func TestMercDollDOMGoldens(t *testing.T) {
	runNode(t, "testdata/mercdoll/merc_dom.js", "web/inventory-merc.js")
}

// The golden views are the schema 3 reply of /api/inventory: they decode into invdesk.View with no unknown field, so
// the fixtures cannot drift from the Go type, and the absent one is a schema 2 style reply that must still decode.
func TestMercFixturesDecodeIntoView(t *testing.T) {
	files, _ := filepath.Glob("testdata/inventory_merc_*.json")
	if len(files) != 6 {
		t.Fatalf("fixtures %v", files)
	}
	want := map[string]string{"none": invdesk.MercNone, "desert": invdesk.MercReady, "ironwolf": invdesk.MercReady, "dead": invdesk.MercDead, "unknown": invdesk.MercUnknown}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		var v invdesk.View
		if err := dec.Decode(&v); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		name := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "inventory_merc_"), ".json")
		if name == "absent" {
			if v.Merc != nil {
				t.Errorf("absent has a merc block")
			}
			continue
		}
		if v.Merc == nil || v.Merc.State != want[name] || v.Schema != invdesk.ViewSchema {
			t.Errorf("%s: %+v schema %d", name, v.Merc, v.Schema)
		}
	}
}

// The page loads the merc files, the routes serve them under the page CSP, and the script stays inside the page rules
// (text through textContent, no style writes, no HTML sink, no scrolling, no polling of its own).
func TestMercDollRoutesAndRules(t *testing.T) {
	for _, want := range []string{`href="/watch/inventory-merc.css"`, `src="/watch/inventory-merc.js"`, `id="merc-doll"`} {
		if !strings.Contains(invPage, want) {
			t.Errorf("inventory page lacks %s", want)
		}
	}
	h := NewInvAPI().Handler()
	for path, ctype := range map[string]string{"/watch/inventory-merc.js": "text/javascript", "/watch/inventory-merc.css": "text/css"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), ctype) || w.Body.Len() < 200 {
			t.Errorf("%s: %d %q", path, w.Code, w.Header().Get("Content-Type"))
		}
	}
	b, err := webFS.ReadFile("web/inventory-merc.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, re := range []string{`innerHTML|outerHTML|insertAdjacentHTML`, `createElement\(\s*['"]style`, `cssText`, `\.style\b`, `scrollIntoView|scrollTo\(|scrollBy\(`, `setInterval|setTimeout|fetch\(`, `\.focus\(`} {
		if regexp.MustCompile(re).MatchString(string(b)) {
			t.Errorf("web/inventory-merc.js matches %q", re)
		}
	}
	if !strings.Contains(string(b), "d2:inventory") {
		t.Error("the merc doll must follow the d2:inventory event")
	}
	css, _ := webFS.ReadFile("web/inventory-merc.css")
	if !strings.Contains(string(css), "@layer components") || !strings.Contains(string(css), `data-owner="merc"`) {
		t.Error("web/inventory-merc.css lacks its layer or selector")
	}
}
