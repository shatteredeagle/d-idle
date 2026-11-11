package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

const goodTokens = `@layer reset, tokens, shell, components, panels, utilities;
@layer tokens { :root { --d-text: #eee; --d-bg: #000; --d-sp-2: 8px; } }`

const goodManifest = `{"schema":"watch.panel/1","version":1,"id":"probe","title":"Probe","band":"grid","size":"s","minSize":"s","maxSize":"m",
"priority":10,"tab":"hero","defaultOpen":true,"endpoint":"/api/probe","poll":5,"unlock":"","skeleton":2,"owner":"uishell","doc":"docs/x.md"}`

// good is a tree with one valid panel; each case below changes or adds one file.
func good() map[string]string {
	return map[string]string{
		"tokens.css":                  goodTokens,
		"reset.css":                   `@layer reset { body { margin: 0; } }`,
		"components.css":              `@layer components { .d-card { color: var(--d-text); padding: var(--d-sp-2); } }`,
		"shell.json":                  `{"status":["hero-status"],"stage":["video"]}`,
		"watch.html":                  `<!DOCTYPE html><html><head><link rel="stylesheet" href="/watch/tokens.css"></head><body><script src="/watch/shell.js"></script></body></html>`,
		"shell.js":                    `el.style.gridColumn = 'span 2'; el.style.setProperty('--fill', '50%'); el.textContent = 'x';`,
		"panels/probe/panel.json":     goodManifest,
		"panels/probe/panel.css":      `@layer panels { [data-panel="probe"] .row { color: var(--d-text); --p-probe-w: 3ch; inline-size: var(--p-probe-w); } @container (width > 20rem) { [data-panel="probe"] .row { gap: 1cqi; } } }`,
		"panels/probe/panel.js":       "export function mount(root, ctx) {\n  const b = root.querySelector('.row');\n  return { update(d) { b.textContent = d.x; } };\n}\n",
		"../docs/x.md":                "doc",
		"../data/unlocks/ladder.json": `{"entries":[{"id":"risk"}]}`,
	}
}

func run(t *testing.T, files map[string]string) []Finding {
	t.Helper()
	repo := t.TempDir()
	web := filepath.Join(repo, "web")
	for name, body := range files {
		p := filepath.Join(web, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	l := &Linter{Root: web, Repo: repo, Ladder: filepath.Join(repo, "data/unlocks/ladder.json")}
	fs, err := l.Run()
	if err != nil {
		t.Fatal(err)
	}
	return fs
}

func TestGoodTreePasses(t *testing.T) {
	if fs := run(t, good()); len(fs) != 0 {
		t.Fatalf("findings on the good tree: %v", fs)
	}
}

func TestEveryRuleFires(t *testing.T) {
	cases := []struct {
		rule, file, body string
	}{
		{"css-parse", "components.css", `@layer components { .d-card { color: red; }`},
		{"css-layered", "components.css", `.d-card { padding: 0; }`},
		{"css-layered", "tokens.css", `@layer tokens, reset; @layer tokens { :root { --d-text: #eee; --d-bg: #000; --d-sp-2: 8px; } }`},
		{"css-layered", "components.css", `@layer reset, tokens, shell, components, panels, utilities; @layer components { .d-card { padding: 0; } }`},
		{"css-layer-file", "components.css", `@layer shell { .d-card { padding: 0; } }`},
		{"css-layer-file", "extra.css", `@layer legacy { .x { padding: 0; } }`},
		{"css-no-important", "components.css", `@layer components { .d-card { padding: 0 !important; } }`},
		{"css-no-id", "components.css", `@layer components { .d-card #x { padding: 0; } }`},
		{"css-element-only-in-reset", "components.css", `@layer components { h2 { padding: 0; } }`},
		{"css-panel-scope", "panels/probe/panel.css", `@layer panels { .row { padding: 0; } }`},
		{"css-panel-scope", "panels/probe/panel.css", `@layer panels { [data-panel="other"] .row { padding: 0; } }`},
		{"css-no-media-in-panel", "panels/probe/panel.css", `@layer panels { @media (width > 40rem) { [data-panel="probe"] .row { padding: 0; } } }`},
		{"css-no-media-in-panel", "components.css", `@layer components { @media (width > 40rem) { .d-card { padding: 0; } } }`},
		{"css-no-fixed-in-panel", "panels/probe/panel.css", `@layer panels { [data-panel="probe"] .row { position: fixed; } }`},
		{"css-no-fixed-in-panel", "panels/probe/panel.css", `@layer panels { [data-panel="probe"] .row { z-index: 3; } }`},
		{"css-no-root-size", "panels/probe/panel.css", `@layer panels { [data-panel="probe"] { inline-size: 20rem; } }`},
		{"css-no-root-size", "panels/probe/panel.css", `@layer panels { [data-panel="probe"] { grid-column: span 2; } }`},
		{"css-no-colour-literal", "components.css", `@layer components { .d-card { color: #fff; } }`},
		{"css-no-colour-literal", "components.css", `@layer components { .d-card { box-shadow: 0 0 2px rgb(0 0 0 / .4); } }`},
		{"css-no-colour-literal", "components.css", `@layer components { .d-card { border-color: gold; } }`},
		{"css-token-exists", "components.css", `@layer components { .d-card { color: var(--d-nope); } }`},
		{"css-token-exists", "panels/probe/panel.css", `@layer panels { [data-panel="probe"] .row { inline-size: var(--p-probe-undefined); } }`},
		{"css-no-vw-in-panel", "panels/probe/panel.css", `@layer panels { [data-panel="probe"] .row { font-size: 2vw; } }`},
		{"html-no-style-attr", "watch.html", `<div style="color: red"></div>`},
		{"html-no-style-element", "watch.html", `<style>.x{}</style>`},
		{"html-no-inline-script", "watch.html", `<script>alert(1)</script>`},
		{"html-no-inline-script", "watch.html", `<button onclick="x()">x</button>`},
		{"html-panel-root", "watch.html", `<section data-panel="probe"></section>`},
		{"js-text-only", "shell.js", `el.innerHTML = s;`},
		{"js-no-inline-style", "shell.js", `el.style.cssText = 'color: red';`},
		{"js-no-inline-style", "shell.js", `document.createElement('style');`},
		{"js-no-inline-style", "shell.js", `el.style.width = '3px';`},
		{"js-no-inline-style", "shell.js", `el.style.setProperty('color', 'red');`},
		{"js-no-globals", "panels/probe/panel.js", "export function mount(root) {\n  window.probe = 1;\n}\n"},
		{"js-no-globals", "panels/probe/panel.js", "var x = 1;\nexport function mount(root) {}\n"},
		{"js-no-globals", "panels/probe/panel.js", "function mount(root) {}\n"},
		{"js-no-timer", "panels/probe/panel.js", "export function mount(root) {\n  setInterval(f, 10);\n}\n"},
		{"js-no-timer", "panels/probe/panel.js", "export function mount(root) {\n  fetch('/api/x');\n}\n"},
		{"js-no-query-outside-root", "panels/probe/panel.js", "export function mount(root) {\n  document.getElementById('x');\n}\n"},
		{"manifest-schema", "panels/probe/panel.json", strings.Replace(goodManifest, `"doc"`, `"extra":1,"doc"`, 1)},
		{"manifest-schema", "panels/probe/panel.json", strings.Replace(goodManifest, `watch.panel/1`, `watch.panel/2`, 1)},
		{"manifest-version", "panels/probe/panel.json", strings.Replace(goodManifest, `"version":1,`, ``, 1)},
		{"manifest-version", "panels/probe/panel.json", strings.Replace(goodManifest, `"version":1,`, `"version":0,`, 1)},
		{"manifest-id", "panels/probe/panel.json", strings.Replace(goodManifest, `"id":"probe"`, `"id":"Probe"`, 1)},
		{"manifest-size", "panels/probe/panel.json", strings.Replace(goodManifest, `"maxSize":"m"`, `"maxSize":"x"`, 1)},
		{"manifest-size", "panels/probe/panel.json", strings.Replace(goodManifest, `"minSize":"s"`, `"minSize":"l"`, 1)},
		{"manifest-band", "panels/probe/panel.json", strings.Replace(goodManifest, `"band":"grid"`, `"band":"status"`, 1)},
		{"manifest-band", "panels/probe/panel.json", strings.Replace(goodManifest, `"band":"grid"`, `"band":"aside"`, 1)},
		{"manifest-priority", "panels/probe/panel.json", strings.Replace(goodManifest, `"priority":10`, `"priority":1000`, 1)},
		{"manifest-tab", "panels/probe/panel.json", strings.Replace(goodManifest, `"tab":"hero"`, `"tab":"misc"`, 1)},
		{"manifest-endpoint", "panels/probe/panel.json", strings.Replace(goodManifest, `/api/probe`, `/watch/probe`, 1)},
		{"manifest-endpoint", "panels/probe/panel.json", strings.Replace(goodManifest, `"/api/probe"`, `""`, 1)},
		{"manifest-unlock", "panels/probe/panel.json", strings.Replace(goodManifest, `"unlock":""`, `"unlock":"nope"`, 1)},
		{"manifest-doc", "panels/probe/panel.json", strings.Replace(goodManifest, `docs/x.md`, `docs/missing.md`, 1)},
		{"manifest-orphan", "panels/empty/panel.css", `@layer panels { [data-panel="empty"] .x { padding: 0; } }`},
	}
	for _, c := range cases {
		files := good()
		files[c.file] = c.body
		fs := run(t, files)
		var rules []string
		hit := false
		for _, f := range fs {
			rules = append(rules, f.Rule)
			if f.Rule == c.rule {
				hit = true
			}
		}
		sort.Strings(rules)
		if !hit {
			t.Errorf("%s in %s: want rule %s, got %v", c.body, c.file, c.rule, rules)
		}
	}
}

func TestDuplicatePriorityAndID(t *testing.T) {
	files := good()
	files["panels/probe2/panel.json"] = strings.Replace(goodManifest, `"id":"probe"`, `"id":"probe2"`, 1)
	files["panels/probe2/panel.css"] = `@layer panels { [data-panel="probe2"] .x { padding: 0; } }`
	files["panels/probe2/panel.js"] = "export function mount(root) {}\n"
	got := map[string]bool{}
	for _, f := range run(t, files) {
		got[f.Rule] = true
	}
	if !got["manifest-priority"] {
		t.Error("two panels with priority 10 in one band passed")
	}
}

func TestStringsAndCommentsDoNotCount(t *testing.T) {
	files := good()
	files["components.css"] = `/* .d-card { color: #fff } */ @layer components { .d-card::before { content: "{ #fff red }"; } }`
	if fs := run(t, files); len(fs) != 0 {
		t.Fatalf("findings: %v", fs)
	}
}

func TestGoTemplateStrings(t *testing.T) {
	dir := t.TempDir()
	src := "package x\n\nconst page = `<html><body><p style=\"color:red\">x</p></body></html>`\n"
	if err := os.WriteFile(filepath.Join(dir, "page.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	web := t.TempDir()
	os.WriteFile(filepath.Join(web, "tokens.css"), []byte(goodTokens), 0o644)
	l := &Linter{Root: web, GoDir: dir}
	fs, err := l.Run()
	if err != nil {
		t.Fatal(err)
	}
	if len(fs) != 1 || fs[0].Rule != "html-no-style-attr" || fs[0].File != "go:page.go" || fs[0].Line != 3 {
		t.Fatalf("got %v", fs)
	}
}

// The real tree passes with the committed allow-list (the same check tools/ci/check.sh runs).
func TestRepoTreeWithAllowList(t *testing.T) {
	repo := filepath.Join("..", "..")
	l := &Linter{Root: filepath.Join(repo, "internal/watch/web"), GoDir: filepath.Join(repo, "internal/watch"), Repo: repo,
		Ladder: filepath.Join(repo, "data/unlocks/ladder.json")}
	fs, err := l.Run()
	if err != nil {
		t.Fatal(err)
	}
	allow, err := readAllow("allow.txt")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range fs {
		if !allow[f.File+" "+f.Rule] {
			t.Errorf("not allow-listed: %v", f)
		}
	}
}

// layout.json and the presets are checked against the panels and the layout block of shell.json.
func TestLayoutManifestRules(t *testing.T) {
	lay := func(panels string) string {
		return `{"schema":"watch.layout/1","cols":12,"rowHeight":48,"breakpoints":{"wide":{"width":1920,"cols":12,"panels":[` + panels + `]}}}`
	}
	p := func(id string, col, row, w, h, order int) string {
		return fmt.Sprintf(`{"id":%q,"band":"grid","col":%d,"row":%d,"colSpan":%d,"rowSpan":%d,"collapsed":false,"order":%d}`, id, col, row, w, h, order)
	}
	shell := `{"status":[],"stage":[],"layout":{"aliases":{"old":"probe"},"compat":["now"],"reserved":["later"]}}`
	cases := []struct {
		name, file, body, rule string
	}{
		{"good", "layout.json", lay(p("probe", 1, 1, 6, 3, 0) + "," + p("now", 7, 1, 6, 3, 1) + "," + p("later", 1, 4, 3, 3, 2)), ""},
		{"alias", "layouts/x.json", lay(p("old", 1, 1, 6, 3, 0)), ""},
		{"unknown id", "layout.json", lay(p("nope", 1, 1, 6, 3, 0)), "layout-manifest"},
		{"overlap", "layout.json", lay(p("probe", 1, 1, 6, 3, 0) + "," + p("now", 6, 2, 6, 3, 1)), "layout-manifest"},
		{"outside the columns", "layouts/x.json", lay(p("probe", 8, 1, 6, 3, 0)), "layout-manifest"},
		{"order twice", "layout.json", lay(p("probe", 1, 1, 6, 3, 0) + "," + p("now", 7, 1, 6, 3, 0)), "layout-manifest"},
		{"schema", "layout.json", `{"schema":"watch.layout/2"}`, "layout-manifest"},
	}
	for _, c := range cases {
		files := good()
		files["shell.json"] = shell
		files[c.file] = c.body
		var got []string
		for _, f := range run(t, files) {
			got = append(got, f.Rule)
		}
		if c.rule == "" && len(got) != 0 || c.rule != "" && (len(got) != 1 || got[0] != c.rule) {
			t.Errorf("%s: findings %v, want %q", c.name, got, c.rule)
		}
	}
	files := good()
	files["shell.json"] = `{"status":[],"stage":[],"layout":{"aliases":{"old":"gone"}}}`
	if fs := run(t, files); len(fs) != 1 || fs[0].Rule != "layout-alias" {
		t.Errorf("an alias to nothing: %v", fs)
	}
}
