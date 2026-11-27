package watch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The shipped layouts (web/layout.json, the default, and the editor's three presets) pass the strict check against
// the real page: known ids, nothing overlapping, spans inside the columns.
func TestShippedLayoutsAreValid(t *testing.T) {
	known, err := LayoutKnownIDs()
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range []string{"web/layout.json", "web/layouts/stream.json"} {
		b, err := webFS.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		l, err := ParseLayout(b)
		if err == nil {
			err = l.Check(known)
		}
		if err != nil {
			t.Errorf("%s: %v", f, err)
		}
		for _, bp := range []string{"phone", "tablet", "laptop", "wide"} {
			if len(l.Breakpoints[bp].Panels) == 0 {
				t.Errorf("%s: no panels at %s", f, bp)
			}
		}
	}
	// the default is the cockpit, byte for byte (S7); Stream is the one other preset (E11a deleted the rest)
	def, _ := webFS.ReadFile("web/layout.json")
	ck, _ := webFS.ReadFile("web/layouts/cockpit.json")
	if !bytes.Equal(def, ck) {
		t.Error("web/layout.json is not web/layouts/cockpit.json")
	}
}

// Check catches what the loader lets through.
func TestLayoutCheck(t *testing.T) {
	mk := func(panels ...LayoutPanel) LayoutManifest {
		return LayoutManifest{Schema: layoutSchema, Cols: 12, Breakpoints: map[string]LayoutBreakpoint{"wide": {Cols: 12, Panels: panels}}}
	}
	a := LayoutPanel{ID: "a", Col: 1, Row: 1, ColSpan: 6, RowSpan: 3}
	cases := []struct {
		name string
		l    LayoutManifest
		want string
	}{
		{"ok", mk(a, LayoutPanel{ID: "b", Col: 7, Row: 1, ColSpan: 6, RowSpan: 3, Order: 1}), ""},
		{"overlap", mk(a, LayoutPanel{ID: "b", Col: 6, Row: 3, ColSpan: 2, RowSpan: 1, Order: 1}), "overlaps"},
		{"collapsed takes one row", mk(LayoutPanel{ID: "a", Col: 1, Row: 1, ColSpan: 6, RowSpan: 3, Collapsed: true}, LayoutPanel{ID: "b", Col: 1, Row: 2, ColSpan: 6, RowSpan: 1, Order: 1}), ""},
		{"twice", mk(a, LayoutPanel{ID: "a", Col: 7, Row: 1, ColSpan: 6, RowSpan: 3, Order: 1}), "twice"},
		{"order", mk(a, LayoutPanel{ID: "b", Col: 7, Row: 1, ColSpan: 6, RowSpan: 3}), "order 0"},
		{"unknown id", mk(LayoutPanel{ID: "zz", Col: 1, Row: 1, ColSpan: 1, RowSpan: 1}), "unknown panel id"},
	}
	known := func(id string) bool { return id == "a" || id == "b" }
	for _, c := range cases {
		err := c.l.Check(known)
		if c.want == "" && err != nil || c.want != "" && (err == nil || !strings.Contains(err.Error(), c.want)) {
			t.Errorf("%s: %v, want %q", c.name, err, c.want)
		}
	}
	bad := mk(a)
	bad.Breakpoints["huge"] = LayoutBreakpoint{}
	if err := bad.Check(nil); err == nil {
		t.Error("an unknown breakpoint name passed")
	}
}

// GET /api/layout serves the default; POST stores a per-profile copy that survives a restart; reset restores the
// default. The route has the gates of the other answer routes and the viewer never writes.
func TestLayoutRoutes(t *testing.T) {
	r := newRig(t, "paladin")
	path := filepath.Join(t.TempDir(), "state", "layout-human.json")
	r.sim.Hub.SetLayoutPath(path)
	def, _ := webFS.ReadFile("web/layout.json")
	if w := r.get(LayoutRoute, nil); w.Code != 200 || !bytes.Equal(w.Body.Bytes(), def) {
		t.Fatalf("GET: %d, body is not web/layout.json", w.Code)
	}
	if w := r.postRaw(r.view, LayoutRoute, `{"reset":true}`, nil); w.Code != 405 {
		t.Errorf("the viewer accepted a POST: %d", w.Code)
	}
	hdr := map[string]string{"Origin": origin}
	// the editor's raw export (it has overlapping entries and names panels this page lacks) is accepted unchanged
	raw, err := os.ReadFile("testdata/compat/layout-v1-command.json")
	if err != nil {
		t.Fatal(err)
	}
	w := r.postRaw(r.ans, LayoutRoute, `{"layout":`+string(raw)+`}`, hdr)
	if w.Code != 200 {
		t.Fatalf("POST: %d %s", w.Code, w.Body.String())
	}
	var rep struct {
		OK     bool           `json:"ok"`
		Layout LayoutManifest `json:"layout"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rep); err != nil || !rep.OK || rep.Layout.Breakpoints["wide"].Panels[0].ID != "video" {
		t.Fatalf("reply %s: %v", w.Body.String(), err)
	}
	if b, err := os.ReadFile(path); err != nil || !bytes.Equal(bytes.TrimSpace(b), bytes.TrimSpace(raw)) {
		t.Errorf("the stored copy is not the posted file: %v", err)
	}
	// a fresh hub with the same profile path serves the stored layout; another profile still gets the default
	r2 := newRig(t, "paladin")
	r2.sim.Hub.SetLayoutPath(path)
	if w := r2.get(LayoutRoute, nil); !bytes.Equal(bytes.TrimSpace(w.Body.Bytes()), bytes.TrimSpace(raw)) {
		t.Error("the stored layout did not survive a restart")
	}
	r3 := newRig(t, "paladin")
	r3.sim.Hub.SetLayoutPath(filepath.Join(filepath.Dir(path), "layout-other.json"))
	if w := r3.get(LayoutRoute, nil); !bytes.Equal(w.Body.Bytes(), def) {
		t.Error("another profile got this profile's layout")
	}
	for name, c := range map[string]struct {
		body string
		hdr  map[string]string
		code int
	}{
		"foreign origin": {`{"reset":true}`, map[string]string{"Origin": "http://evil.example"}, 403},
		"later schema":   {`{"layout":{"schema":"watch.layout/2"}}`, hdr, 400},
		"outside grid":   {`{"layout":{"schema":"watch.layout/1","cols":12,"breakpoints":{"wide":{"cols":12,"panels":[{"id":"video","col":9,"row":1,"colSpan":6,"rowSpan":1}]}}}}`, hdr, 400},
		"empty":          {`{}`, hdr, 400},
		"unknown field":  {`{"layout":null,"x":1}`, hdr, 400},
		"too large":      {`{"layout":{"schema":"watch.layout/1","pad":"` + strings.Repeat("x", layoutMaxBody) + `"}}`, hdr, 400},
		"not json":       {`{"reset":true}`, map[string]string{"Origin": origin, "Content-Type": "text/plain"}, 415},
	} {
		if w := r.postRaw(r.ans, LayoutRoute, c.body, c.hdr); w.Code != c.code {
			t.Errorf("%s: %d, want %d (%s)", name, w.Code, c.code, w.Body.String())
		}
	}
	if w := r.get(LayoutRoute, nil); !bytes.Equal(bytes.TrimSpace(w.Body.Bytes()), bytes.TrimSpace(raw)) {
		t.Error("a refused POST changed the layout")
	}
	if w := r.postRaw(r.ans, LayoutRoute, `{"reset":true}`, hdr); w.Code != 200 {
		t.Fatalf("reset: %d %s", w.Code, w.Body.String())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("reset left the stored copy")
	}
	if w := r.get(LayoutRoute, nil); !bytes.Equal(w.Body.Bytes(), def) {
		t.Error("reset did not restore the default")
	}
	// a token-guarded answer route refuses a POST without it
	guarded := r.sim.Hub.AnswerHandler(AnswerOptions{Token: "tok", Origins: []string{origin}})
	if w := r.postRaw(guarded, LayoutRoute, `{"reset":true}`, hdr); w.Code != 401 {
		t.Errorf("no token: %d", w.Code)
	}
	// a stored file that no longer parses is ignored, not served
	os.MkdirAll(filepath.Dir(path), 0o755)
	os.WriteFile(path, []byte(`{"schema":"watch.layout/9"}`), 0o644)
	r.sim.Hub.SetLayoutPath(path)
	if w := r.get(LayoutRoute, nil); !bytes.Equal(w.Body.Bytes(), def) {
		t.Error("an unreadable stored layout was served")
	}
}

// /watch/panels.json carries the id mapping of the layout, and the layout assets are served.
func TestLayoutAssetsAndConf(t *testing.T) {
	r := newRig(t, "sorceress")
	var pv PanelsView
	r.get("/watch/panels.json", &pv)
	if pv.Layout.Aliases["whyline"] != "why" || !containsStr(pv.Layout.Compat, "orders-map") || !containsStr(pv.Layout.Reserved, "risk") {
		t.Errorf("layout conf: %+v", pv.Layout)
	}
	page, _ := webFS.ReadFile("web/watch.html")
	if bytes.Contains(page, []byte("data-lid=")) {
		t.Error("watch.html still has a pre-shell card (E11a deleted them; shell.json keeps the ids only so a saved layout that names one loads)")
	}
	for a, to := range pv.Layout.Aliases {
		if _, err := webFS.ReadFile("web/panels/" + to + "/panel.json"); err != nil && !containsStr(pv.Layout.Compat, to) {
			t.Errorf("alias %s points at %s, which is not on the page", a, to)
		}
	}
	for p, ctype := range map[string]string{"/watch/layout.js": "text/javascript", "/watch/layoutedit.js": "text/javascript",
		"/watch/layouts/cockpit.json": "application/json", "/watch/layouts/stream.json": "application/json"} {
		w := httptest.NewRecorder()
		r.view.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), ctype) {
			t.Errorf("%s: %d %s", p, w.Code, w.Header().Get("Content-Type"))
		}
	}
	if w := r.get("/watch/layouts/other.json", nil); w.Code != 404 {
		t.Errorf("an unknown preset: %d", w.Code)
	}
}

// The layout scripts keep the page rules: the editor is loaded only on demand, nothing injects styles (the page CSP is
// style-src 'self'), focus never scrolls the page, the editor has no timer or fetch of its own, text is textContent.
func TestLayoutJSPatterns(t *testing.T) {
	page, _ := webFS.ReadFile("web/watch.html")
	if bytes.Contains(page, []byte("layoutedit.js")) || !bytes.Contains(page, []byte(`id="btn-layout"`)) {
		t.Error("watch.html must offer Edit layout and must not load layoutedit.js itself")
	}
	shell, _ := webFS.ReadFile("web/shell.js")
	for _, want := range []string{"import('/watch/layoutedit.js')", "from '/watch/layout.js'", "'/api/layout'", "l.schema === 'watch.layout/1'"} {
		if !bytes.Contains(shell, []byte(want)) {
			t.Errorf("shell.js lacks %s", want)
		}
	}
	// which layout loads (S1 review R2 to R4): a missing X-Layout-Source header reads as stored, a new hero has ticked nothing and
	// meets the cockpit (S7: nothing pins Simple; the menu marks a choice with d2layoutpick), the overlay strip ignores the layout's status list
	for _, want := range []string{"srv.source !== 'default'", "!(o.body.steps || []).some((x) => x.done)", "effectiveChoice(pref(PREF), pref(PICK))", "selectPanels(list, sel && sel.layout, html.dataset.mode)"} {
		if !bytes.Contains(shell, []byte(want)) {
			t.Errorf("shell.js lacks %s", want)
		}
	}
	bad := regexp.MustCompile(`createElement\(\s*['"]style['"]|setAttribute\(\s*['"]style['"]|insertAdjacentHTML|innerHTML|outerHTML|cssText|<style|\sstyle\s*=`)
	own := regexp.MustCompile(`\b(setInterval|setTimeout|fetch|EventSource|XMLHttpRequest)\s*\(`)
	focus := regexp.MustCompile(`\.focus\(([^)]*)\)`)
	for _, name := range []string{"layout.js", "layoutedit.js"} {
		b, err := webFS.ReadFile("web/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if m := bad.Find(b); m != nil {
			t.Errorf("web/%s: %q is blocked by the page CSP or is an HTML sink", name, m)
		}
		if m := own.Find(b); m != nil {
			t.Errorf("web/%s: %q (the shell owns timers and requests)", name, m)
		}
		for _, m := range focus.FindAllSubmatch(b, -1) {
			if !bytes.Contains(m[1], []byte("preventScroll")) {
				t.Errorf("web/%s: %s scrolls the page", name, m[0])
			}
		}
	}
}

// The layout logic in node: the editor's raw exports (compat fixtures) and the shipped files plan without overlap at
// every width, editor ids land on the real panels.
func TestLayoutPlanJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	args := []string{"testdata/layout_plan.js", "web/layout.js", "testdata/compat/layout-v1-mockup.json", "testdata/compat/layout-v1-live.json",
		"testdata/compat/layout-v1-command.json"}
	if out, err := exec.Command(node, args...).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// Every layout fixture of v1 (files the editor exported) still loads; fixtures are only added.
func TestCompatLayoutFixturesLoad(t *testing.T) {
	files, _ := filepath.Glob("testdata/compat/layout-v1-*.json")
	if len(files) < 3 {
		t.Fatalf("layout fixtures: %v", files)
	}
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if l, err := ParseLayout(b); err != nil || len(l.Breakpoints["wide"].Panels) == 0 {
			t.Errorf("%s no longer loads: %v", f, err)
		}
	}
}

// The editor's move in node: the minimap-sized panel drops exactly into free cells (hidden panels reserve nothing),
// overlap pushes down and cascades, a preview never changes the layout it started from, a drop back is a no-op.
func TestLayoutMoveJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	if out, err := exec.Command(node, "testdata/layout_move.js", "web/layout.js", "web/layouts/stream.json").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// The live preview of the editor: it is computed from the committed cells every time (so going back is free), it is
// dropped on Escape, on a release outside the grid and on pointercancel, Enter commits a keyboard preview, the slide
// honours reduced motion, and nothing but Save reaches the stored layout. The normal page never loads any of it.
func TestLayoutEditPreviewContract(t *testing.T) {
	b, err := webFS.ReadFile("web/layoutedit.js")
	if err != nil {
		t.Fatal(err)
	}
	// the grid editor is the part before the cockpit's; the cockpit editor has its own contract (TestCockpitEditorContract)
	js, _, _ := strings.Cut(string(b), "// ---- Edit layout on a cockpit")
	for _, want := range []string{"moveItem(base, id, locked(id, rect), cols)", "prefers-reduced-motion: reduce", "k === 'Escape'", "k === 'Enter'", "'pointercancel', cancel",
		"if (keep) commit(next); else revert();", "main.dataset.preview = id", "delete main.dataset.preview"} {
		if !strings.Contains(js, want) {
			t.Errorf("layoutedit.js lacks %q", want)
		}
	}
	if n := strings.Count(js, "api.save("); n != 1 {
		t.Errorf("api.save is called from %d places; only the Save button may store the layout", n)
	}
	if strings.Contains(js, "api.preview(") {
		t.Error("a preview must not go through the shell's layout object")
	}
	css, _ := webFS.ReadFile("web/shell.css")
	for _, want := range []string{`[data-flip="go"] { transition: transform var(--d-dur-2) var(--d-ease); }`, ".d-main[data-preview]"} {
		if !strings.Contains(string(css), want) {
			t.Errorf("shell.css lacks %q", want)
		}
	}
	for _, sel := range regexp.MustCompile(`(?m)^\s*(\S[^{]*(data-flip|data-preview|data-editcell)[^{]*)\{`).FindAllStringSubmatch(string(css), -1) {
		// .d-editframe exists only while the editor runs
		if !strings.Contains(sel[1], "[data-editing]") && !strings.Contains(sel[1], "[data-preview]") && !strings.Contains(sel[1], ".d-editframe") {
			t.Errorf("an edit-mode rule applies to the normal page: %s", sel[1])
		}
	}
}

// The editor driven in a stub DOM (node): a pointer drag previews live and pushes panels down, a drop commits, Escape,
// a release outside the grid and pointercancel revert, the keyboard previews until Enter, Save posts the unchanged
// export format, Done leaves no trace.
func TestLayoutEditJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	out, err := exec.Command(node, "testdata/layout_edit.js", "web/layout.js", "web/layoutedit.js").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "ok") {
		t.Fatalf("%v\n%s", err, out)
	}
}

// Every panel root is a layout target (the pre-shell cards are gone, E11a): shell.js takes the panel roots, shell.json keeps the card ids
// valid for a saved layout, and the editor frames them without putting its handle inside a panel.
func TestEveryWatchPanelIsALayoutTarget(t *testing.T) {
	page, _ := webFS.ReadFile("web/watch.html")
	if regexp.MustCompile(`<section\b`).Match(page) {
		t.Error("watch.html has a pre-shell <section> card again")
	}
	js, _ := webFS.ReadFile("web/shell.js")
	if !strings.Contains(string(js), `querySelectorAll('.d-panel[data-panel]')`) {
		t.Error("shell.js must take every panel root as a layout target")
	}
	ed, _ := webFS.ReadFile("web/layoutedit.js")
	if regexp.MustCompile(`\bn\.(append|appendChild|prepend|insertBefore)\(`).Match(ed) {
		t.Error("the editor must not put its handle inside a panel: a card that redraws itself wipes it")
	}
}

// The game picture always fits its panel: contained and centred in a frame that is 4:3 of the width when the cell
// is as tall as its content and the whole body when the layout gives the cell a height; the card head is never
// squeezed; the panel tells the editor its shape.
func TestVideoPictureFitsItsPanel(t *testing.T) {
	css, _ := webFS.ReadFile("web/panels/video/panel.css")
	for _, want := range []string{"aspect-ratio: 4 / 3", "object-fit: contain", "object-position: center", "min-block-size: 0"} {
		if !strings.Contains(string(css), want) {
			t.Errorf("video panel.css lacks %q", want)
		}
	}
	// the card's height comes from its width: the panel declares fit, so its rows are measured, never the stored span
	sh, _ := webFS.ReadFile("web/shell.js")
	for _, want := range []string{"if (m.fit) root.dataset.fit = '1'", "function measureFit()", "contentHeight(n, getComputedStyle(n))", "new ResizeObserver(scheduleFit)", "'fullscreenchange', scheduleFit"} {
		if !strings.Contains(string(sh), want) {
			t.Errorf("shell.js lacks %q", want)
		}
	}
	js, _ := webFS.ReadFile("web/panels/video/panel.js")
	if !strings.Contains(string(js), "root.dataset.aspect = '4/3'") {
		t.Error("the video panel must declare its aspect for the editor")
	}
}

// hidden: true on an entry is additive: old files (no key) load as before, a hidden entry reserves nothing.
func TestLayoutHiddenEntry(t *testing.T) {
	l, err := ParseLayout([]byte(`{"schema":"watch.layout/1","cols":12,"breakpoints":{"wide":{"cols":12,"panels":[
		{"id":"a","col":1,"row":1,"colSpan":6,"rowSpan":3,"order":0,"hidden":true},{"id":"b","col":1,"row":1,"colSpan":6,"rowSpan":3,"order":1}]}}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !l.Breakpoints["wide"].Panels[0].Hidden || l.Breakpoints["wide"].Panels[1].Hidden {
		t.Errorf("hidden not read: %+v", l.Breakpoints["wide"].Panels)
	}
	if err := l.Check(nil); err != nil {
		t.Errorf("a hidden entry must not count as an overlap: %v", err)
	}
	b, _ := json.Marshal(l.Breakpoints["wide"].Panels[1])
	if strings.Contains(string(b), "hidden") {
		t.Errorf("a shown entry must not grow a hidden key: %s", b)
	}
}

// Fit content: every panel of the main area and every pre-shell card declares it, so no card keeps a stored span
// taller than what it shows; the manifest key is additive (a manifest without it loads, with the stored spans).
// cockpitPanels are the opt-in panels the redesign slices have shipped so far (a slice adds its id here).
var cockpitPanels = map[string]bool{"orders": true, "town-plan": true}

func TestPanelsFitTheirContent(t *testing.T) {
	ms, err := Manifests()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		if (m.Band == "side" || m.Band == "stage") && !m.Fit {
			t.Errorf("panel %s does not fit its content", m.ID)
		}
		if m.Band == "status" && m.Fit {
			t.Errorf("panel %s is in the top band; fit is for the grid", m.ID)
		}
	}
	lc, _ := shellLayoutConf()
	for _, id := range lc.Compat {
		if !containsStr(lc.Fit, id) {
			t.Errorf("the card %s does not fit its content (shell.json layout.fit)", id)
		}
	}
	old, _ := os.ReadFile("testdata/compat/panels-v1/video.json")
	if m, err := parseManifest(old, "video"); err != nil || m.Fit {
		t.Errorf("a v1 manifest must load without fit: %v %+v", err, m)
	}
	var pv PanelsView
	newRig(t, "paladin").get("/watch/panels.json", &pv)
	if len(pv.Layout.Fit) == 0 {
		t.Error("/watch/panels.json does not carry the fit list")
	}
	fit := 0
	for _, m := range pv.Panels {
		if m.Fit {
			fit++
		}
	}
	if fit == 0 {
		t.Error("/watch/panels.json does not carry the fit flag")
	}
}

// The fit in node: a video panel 600 px wide whose stored span is far too tall renders with no empty area.
func TestLayoutFitJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	if out, err := exec.Command(node, "testdata/layout_fit.js", "web/layout.js").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// S1 in node: old layouts keep their cells (the golden is what the shell before S1 planned), Standard and Simple resolve against
// today's panel list with every fallback in its cell, an opt-in panel stays unmounted unless named, a stand-in saves as its tile.
func TestLayoutS1JS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	out, err := exec.Command(node, "testdata/layout_s1.js", "check", "web/layout.js", "testdata/layout_cells_s1.json", "web", "testdata/compat").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "ok") {
		t.Fatalf("%v\n%s", err, out)
	}
}

// The compat fixtures and the old default (classic) still parse to the same placed ids and cells as before S1: the Go loader's side
// of TestLayoutS1JS (the golden holds the node plan; here the raw entries are compared with the file).
func TestOldLayoutsKeepTheirEntries(t *testing.T) {
	for _, f := range []string{"testdata/compat/layout-v1-mockup.json", "testdata/compat/layout-v1-live.json", "testdata/compat/layout-v1-command.json"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		l, err := ParseLayout(b)
		if err != nil || len(l.Status) != 0 {
			t.Errorf("%s: %v status %v", f, err, l.Status)
		}
		for bp, v := range l.Breakpoints {
			for _, p := range v.Panels {
				if p.Fallback != "" {
					t.Errorf("%s %s: %s has a fallback", f, bp, p.ID)
				}
			}
		}
	}
}

// The layout keys of S1: status entries are ids or {id, fallback}, an entry may name a fallback, both round-trip, and an
// unknown id passes only with a (known) fallback.
func TestLayoutStatusAndFallback(t *testing.T) {
	src := `{"schema":"watch.layout/1","cols":12,"status":["a",{"id":"later","fallback":"b"}],"breakpoints":{"wide":{"cols":12,"panels":[
		{"id":"later","col":1,"row":1,"colSpan":6,"rowSpan":3,"order":0,"fallback":"b"},{"id":"b","col":1,"row":1,"colSpan":6,"rowSpan":3,"order":1,"hidden":true}]}}}`
	l, err := ParseLayout([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Status) != 2 || l.Status[0] != (LayoutStatus{ID: "a"}) || l.Status[1] != (LayoutStatus{ID: "later", Fallback: "b"}) || l.Breakpoints["wide"].Panels[0].Fallback != "b" {
		t.Fatalf("read: %+v", l)
	}
	known := func(id string) bool { return id == "a" || id == "b" }
	if err := l.Check(known); err != nil {
		t.Errorf("an unbuilt id with a known fallback: %v", err)
	}
	if b, _ := json.Marshal(l.Status); string(b) != `["a",{"id":"later","fallback":"b"}]` {
		t.Errorf("status marshals as %s", b)
	}
	l.Breakpoints["wide"].Panels[0].Fallback = "zz"
	if err := l.Check(known); err == nil || !strings.Contains(err.Error(), "fallback") {
		t.Errorf("unknown fallback: %v", err)
	}
	l.Breakpoints["wide"].Panels[0].Fallback = ""
	if err := l.Check(known); err == nil {
		t.Error("an unknown id without a fallback passed")
	}
	if _, err := ParseLayout([]byte(`{"schema":"watch.layout/1","cols":12,"status":[7],"breakpoints":{}}`)); err == nil {
		t.Error("a status entry that is a number was accepted")
	}
}

// GET /api/layout says whether the layout is the hero's stored copy (the shell starts a new hero on Simple only when it is not),
// and the panel manifest key optIn reaches /watch/panels.json.
func TestLayoutSourceHeader(t *testing.T) {
	r := newRig(t, "paladin")
	path := filepath.Join(t.TempDir(), "layout-human.json")
	r.sim.Hub.SetLayoutPath(path)
	get := func() string {
		w := httptest.NewRecorder()
		r.view.ServeHTTP(w, httptest.NewRequest("GET", LayoutRoute, nil))
		return w.Header().Get("X-Layout-Source")
	}
	if got := get(); got != "default" {
		t.Errorf("no stored layout: %q", got)
	}
	hdr := map[string]string{"Origin": origin}
	raw, _ := os.ReadFile("testdata/compat/layout-v1-live.json")
	if w := r.postRaw(r.ans, LayoutRoute, `{"layout":`+string(raw)+`}`, hdr); w.Code != 200 {
		t.Fatalf("POST: %d", w.Code)
	}
	if got := get(); got != "stored" {
		t.Errorf("after a save: %q", got)
	}
	if w := r.postRaw(r.ans, LayoutRoute, `{"reset":true}`, hdr); w.Code != 200 {
		t.Fatalf("reset: %d", w.Code)
	}
	if got := get(); got != "default" {
		t.Errorf("after a reset: %q", got)
	}
	m, err := parseManifest([]byte(`{"schema":"watch.panel/1","id":"x","band":"side","optIn":true}`), "x")
	if err != nil || !m.OptIn {
		t.Errorf("optIn: %+v %v", m, err)
	}
	if ms, _ := Manifests(); len(ms) > 0 {
		for _, m := range ms {
			if m.OptIn && !m.CockpitOnly && !cockpitPanels[m.ID] {
				t.Errorf("%s: no panel of today is opt-in", m.ID)
			}
		}
	}
}

// The new tokens of the visual system (redesign E.2) exist, as aliases over the themed values, so the light theme follows.
func TestRedesignTokens(t *testing.T) {
	b, err := os.ReadFile("web/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	for _, tok := range []string{"--d-ok:", "--d-danger:", "--d-muted:", "--d-state-bg-ok:", "--d-state-bg-warn:", "--d-state-bg-danger:", "--d-state-bg-info:", "--d-hit: 44px", "--d-row: 36px"} {
		if !strings.Contains(css, tok) {
			t.Errorf("tokens.css lacks %s", tok)
		}
	}
	for _, alias := range []string{"--d-ok: var(--d-good)", "--d-danger: var(--d-bad)", "--d-muted: var(--d-text-3)"} {
		if !strings.Contains(css, alias) {
			t.Errorf("tokens.css lacks the alias %s", alias)
		}
	}
	// no hex of its own in the aliases: the light theme redefines the base colours and the aliases resolve on the same element
	if regexp.MustCompile(`--d-(ok|danger|muted|state-bg-[a-z]+):[^;]*#`).MatchString(css) {
		t.Error("a new token holds a colour literal; it must follow the themed value")
	}
}

// The older layouts render the same cells as before the cockpit existed: every file without a template plans at five
// widths, in the page and in the editor, and mounts the same panels, as hashed with the layout.js of before the cockpit
// (testdata/layout_cells.golden). A cockpit file is not in this list: its template is what changes the loader.
func TestTemplatelessLayoutsRenderTheSameCells(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	args := []string{"testdata/layout_cells.js", "web/layout.js", "testdata/layout_cells.golden", "web/layouts/stream.json",
		"testdata/compat/layout-v1-mockup.json", "testdata/compat/layout-v1-live.json", "testdata/compat/layout-v1-command.json"}
	if out, err := exec.Command(node, args...).CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, f := range args[3:] {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if l, err := ParseLayout(b); err != nil || l.Template != "" || len(l.Slots) > 0 {
			t.Errorf("%s: template %q slots %v err %v", f, l.Template, l.Slots, err)
		}
	}
}

// The cockpit's pure logic in node: plan per width, slots left out, which panels mount, swap, dividers, save and reload.
func TestCockpitPlanJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	if out, err := exec.Command(node, "testdata/layout_cockpit.js", "web/layout.js", "web/layouts/cockpit.json").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// web/layouts/cockpit.json is a valid cockpit: every id is a slot panel or an id that is not built yet, and it also
// carries old-style breakpoints for a shell that does not know the template.
func TestShippedCockpitLayout(t *testing.T) {
	b, err := webFS.ReadFile("web/layouts/cockpit.json")
	if err != nil {
		t.Fatal(err)
	}
	l, err := ParseLayout(b)
	if err != nil {
		t.Fatal(err)
	}
	known, _ := LayoutKnownIDs()
	ms, _ := Manifests()
	lc, _ := shellLayoutConf()
	slot, built := map[string]bool{}, map[string]bool{}
	for _, m := range ms {
		built[m.ID] = true
		slot[m.ID] = m.Slot || containsStr(lc.Slot, m.ID)
	}
	ok := func(id string) bool { return slot[id] || !built[id] && containsStr(lc.Reserved, id) }
	if err := l.Check(known); err != nil {
		t.Error(err)
	}
	if err := l.CheckSlots(ok); err != nil {
		t.Error(err)
	}
	sl, _, err := l.Cockpit()
	if err != nil || l.Template != "cockpit/1" || sl == nil || len(sl.Stage) != 1 || sl.RailWide == nil || len(sl.PhoneTabs) != 4 || len(sl.RailWide.Top) != 2 {
		t.Errorf("cockpit.json: %+v %v", sl, err)
	}
	if len(l.Breakpoints["wide"].Panels) == 0 || len(l.Breakpoints["phone"].Panels) == 0 {
		t.Error("cockpit.json has no old-style breakpoints for older shells")
	}
	for _, id := range []string{"video", "hero-meters"} {
		if !slot[id] {
			t.Errorf("%s must fit a slot (shell.json slot list or its manifest)", id)
		}
	}
	// the default is the cockpit since S7
	def, _ := webFS.ReadFile("web/layout.json")
	if dl, _ := ParseLayout(def); dl.Template != "cockpit/1" {
		t.Error("web/layout.json is not the cockpit")
	}
}

func TestParseCockpitKeys(t *testing.T) {
	base := `{"schema":"watch.layout/1","cols":12,"rowHeight":48,"breakpoints":{},"template":"cockpit/1","sizes":{"strip":100,"railMin":420},"slots":%s}`
	parse := func(slots string) (LayoutManifest, *LayoutSlots, LayoutSizes, error) {
		l, err := ParseLayout([]byte(fmt.Sprintf(base, slots)))
		if err != nil {
			return l, nil, LayoutSizes{}, err
		}
		sl, z, err := l.Cockpit()
		return l, sl, z, err
	}
	l, sl, z, err := parse(`{"band":["a"],"stage":"video","strip":["b","c"],"rail":["d"],"railWide":{"top":["t"],"left":["d"],"right":["e"]},"phoneTabs":{"T":["d"]}}`)
	if err != nil || l.Template != "cockpit/1" || len(sl.Stage) != 1 || sl.Stage[0] != "video" || len(sl.Strip) != 2 || z.RailMin != 420 || sl.RailWide.Right[0] != "e" || sl.RailWide.Top[0] != "t" {
		t.Fatalf("%+v %v", sl, err)
	}
	all := func(string) bool { return true }
	if err := l.CheckSlots(all); err != nil {
		t.Error(err)
	}
	// the loader ignores a slot name it does not know (a later version), the strict check refuses it
	l, sl, _, err = parse(`{"stage":"video","dock":["x"]}`)
	if err != nil || len(sl.Unknown) != 1 {
		t.Fatalf("%+v %v", sl, err)
	}
	if l.CheckSlots(all) == nil {
		t.Error("an unknown slot name passed the check")
	}
	l, _, _, _ = parse(`{"stage":"video","rail":["card"]}`)
	if err := l.CheckSlots(func(id string) bool { return id != "card" }); err == nil || !strings.Contains(err.Error(), "card") {
		t.Errorf("a non-slot panel: %v", err)
	}
	l, _, _, _ = parse(`{"stage":["video","map"]}`)
	if l.CheckSlots(all) == nil {
		t.Error("two stage panels passed")
	}
	l, _, _, _ = parse(`{"stage":"video","band":["a"],"strip":["a"]}`)
	if l.CheckSlots(all) == nil {
		t.Error("a panel twice in the page passed")
	}
	l, _, _, _ = parse(`{"stage":"video","railWide":{"top":["a"],"left":["a"]}}`)
	if l.CheckSlots(all) == nil {
		t.Error("a panel twice in the wide page passed")
	}
	l, _ = ParseLayout([]byte(`{"schema":"watch.layout/1","breakpoints":{},"template":"cockpit/1","sizes":{"strip":7},"slots":{"stage":"video"}}`))
	if l.CheckSlots(all) == nil {
		t.Error("strip 7 passed")
	}
	// a file of a later version loads, is stored and served back byte for byte; only the strict check says no
	for _, f := range []string{
		`{"schema":"watch.layout/1","breakpoints":{},"template":"cockpit/2","slots":{"stage":{"id":"video"}},"sizes":{"strip":"12%"}}`,
		`{"schema":"watch.layout/1","breakpoints":{},"template":"cockpit/1","slots":["video"],"sizes":{"strip":"12%"}}`,
		`{"schema":"watch.layout/1","breakpoints":{},"template":"cockpit/1.2","slots":{"stage":"video","rail":[{"id":"x"}]},"future":{"a":1}}`,
		`{"schema":"watch.layout/1","breakpoints":{},"template":"cockpit/1","slots":{"stage":"video","rail":[{"id":"x"}]}}`,
	} {
		l, err := ParseLayout([]byte(f))
		if err != nil {
			t.Errorf("%s: the loader refused it: %v", f, err)
			continue
		}
		if l.CheckSlots(all) == nil {
			t.Errorf("%s: the strict check passed it", f)
		}
		p := filepath.Join(t.TempDir(), "layout.json")
		os.WriteFile(p, []byte(f), 0o644)
		st := &layoutStore{path: p}
		if got, err := st.load(); err != nil || !st.stored || string(got) != f {
			t.Errorf("%s: a stored file of a later version was not served back as it is (stored %v, %v)", f, st.stored, err)
		}
	}
	// an unknown template is read as a plain layout (ignored), refused only by the strict check
	l, err = ParseLayout([]byte(`{"schema":"watch.layout/1","breakpoints":{},"template":"cockpit/9"}`))
	if err != nil || l.CheckSlots(all) == nil {
		t.Errorf("template cockpit/9: %v", err)
	}
}

// The cockpit editor stores only through Save, and every change it makes goes through one function that previews the
// layout object it holds: the shell's own layout object is changed only by api.preview (reverted by Done).
func TestCockpitEditorContract(t *testing.T) {
	b, _ := webFS.ReadFile("web/layoutedit.js")
	_, js, ok := strings.Cut(string(b), "// ---- Edit layout on a cockpit")
	if !ok {
		t.Fatal("no cockpit editor in layoutedit.js")
	}
	if n := strings.Count(js, "api.save("); n != 1 {
		t.Errorf("api.save is called from %d places in the cockpit editor; only the Save button may store the layout", n)
	}
	for _, want := range []string{"cockpitSwap(layout, id, over)", "cockpitSize(", "api.done(kept)", "'pointercancel', cancel", "role', 'separator'", "'keydown'"} {
		if !strings.Contains(js, want) {
			t.Errorf("the cockpit editor lacks %q", want)
		}
	}
}

// S7: the default is the cockpit; an auto-pinned Simple is no choice; a missing slot panel is named by the self-check.
func TestDefaultLayoutChoiceJS(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not available")
	}
	if out, err := exec.Command(node, "testdata/layout_default.js", "web/layout.js", "web/layout.json").CombinedOutput(); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}

// Every panel a cockpit slot names has a manifest that says it fits a slot (the Go side of the startup self-check).
func TestDefaultCockpitSlotsHavePanels(t *testing.T) {
	b, _ := webFS.ReadFile("web/layout.json")
	var l struct {
		Slots struct {
			Band, Strip, Rail []string
			Stage             string
		}
	}
	if err := json.Unmarshal(b, &l); err != nil || len(l.Slots.Rail) == 0 {
		t.Fatalf("default: %v", err)
	}
	ms, _ := Manifests()
	have := map[string]bool{}
	for _, m := range ms {
		have[m.ID] = true
	}
	for _, id := range append(append(append([]string{l.Slots.Stage}, l.Slots.Band...), l.Slots.Strip...), l.Slots.Rail...) {
		if !have[id] {
			t.Errorf("the default cockpit names %s, which has no panel manifest", id)
		}
	}
}
