package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Finding is one rule violation; String gives the `file:line: rule: message` form the tool prints.
type Finding struct {
	File string
	Line int
	Rule string
	Msg  string
}

func (f Finding) String() string { return fmt.Sprintf("%s:%d: %s: %s", f.File, f.Line, f.Rule, f.Msg) }

// Layers is the one layer order of the watch UI (docs/ui-design-system.md section 8).
var Layers = []string{"reset", "tokens", "shell", "components", "panels", "utilities"}

var fileLayer = map[string]string{"tokens.css": "tokens", "reset.css": "reset", "shell.css": "shell", "components.css": "components", "utilities.css": "utilities"}

var (
	reHexColour  = regexp.MustCompile(`#[0-9a-fA-F]{3,8}\b`)
	reColourFn   = regexp.MustCompile(`\b(rgba?|hsla?|oklch|oklab|lab|lch|hwb|color-mix)\(`)
	reIdent      = regexp.MustCompile(`-?[A-Za-z_][A-Za-z0-9_-]*`)
	reVar        = regexp.MustCompile(`var\(\s*(--[A-Za-z0-9_-]+)`)
	reViewport   = regexp.MustCompile(`\d(vw|vh|dvh|svh|lvh|vmin|vmax)\b`)
	reIDSel      = regexp.MustCompile(`#[A-Za-z_-]`)
	rePanelID    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	reStyleAttr  = regexp.MustCompile(`(?i)\sstyle\s*=`)
	reStyleEl    = regexp.MustCompile(`(?i)<style[\s>]`)
	reScriptTag  = regexp.MustCompile(`(?i)<script\b[^>]*>`)
	reOnAttr     = regexp.MustCompile(`(?i)\son[a-z]+\s*=\s*["']`)
	reDataPanel  = regexp.MustCompile(`data-panel\s*=`)
	reHTMLSink   = regexp.MustCompile(`\.(innerHTML|outerHTML)\b|insertAdjacentHTML`)
	reStyleWrite = regexp.MustCompile(`\.style\.([A-Za-z]+)\s*=[^=]`)
	reStyleBad   = regexp.MustCompile(`\.style\.cssText|setAttribute\(\s*['"]style['"]|createElement\(\s*['"]style['"]`)
	reSetProp    = regexp.MustCompile(`\.style\.setProperty\(\s*(['"])([^'"]*)`)
	reGlobal     = regexp.MustCompile(`\bwindow\.[A-Za-z_$][\w$]*\s*=[^=]`)
	reTimer      = regexp.MustCompile(`\b(setInterval|setTimeout|fetch|EventSource)\s*\(`)
	reDocQuery   = regexp.MustCompile(`\bdocument\.(querySelector|querySelectorAll|getElementById|getElementsBy\w+|body)\b`)
	reBacktick   = regexp.MustCompile("(?s)`[^`]*`")
)

// The CSS named colours a hand-written sheet is likely to use; transparent and currentColor stay allowed.
var namedColours = map[string]bool{}

func init() {
	for _, n := range strings.Fields(`black white red green blue yellow orange purple pink gray grey silver gold navy teal
		maroon olive lime aqua fuchsia cyan magenta brown crimson indigo violet beige ivory khaki coral salmon tomato orchid
		plum tan wheat darkred darkgreen darkblue lightgray lightgrey darkgray darkgrey whitesmoke`) {
		namedColours[n] = true
	}
}

var styleWriteOK = map[string]bool{"gridArea": true, "gridColumn": true, "gridRow": true, "transform": true}

var rootSizeProps = map[string]bool{"width": true, "height": true, "min-width": true, "max-width": true, "min-height": true,
	"max-height": true, "inline-size": true, "block-size": true, "min-inline-size": true, "max-inline-size": true,
	"grid-column": true, "grid-row": true, "grid-area": true, "order": true, "margin": true}

// Linter holds the inputs of one run. Root is the web directory (internal/watch/web); GoDir, when set, is scanned for
// Go template strings (backtick strings holding HTML); Ladder is the unlock ladder file for manifest unlock ids; Repo,
// when set, is the directory manifest doc paths are relative to.
type Linter struct {
	Root, GoDir, Ladder, Repo string
	findings                  []Finding
	tokens                    map[string]bool
	layerStmts                []Finding // where `@layer a, b, ...;` order statements were found
}

func (l *Linter) add(file string, line int, rule, format string, a ...any) {
	l.findings = append(l.findings, Finding{File: file, Line: line, Rule: rule, Msg: fmt.Sprintf(format, a...)})
}

// Run lints every file under Root (and GoDir) and returns the findings sorted by file and line.
func (l *Linter) Run() ([]Finding, error) {
	var css, html, js, manifests []string
	err := filepath.WalkDir(l.Root, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(l.Root, p)
		rel = filepath.ToSlash(rel)
		switch {
		case strings.HasSuffix(rel, ".css"):
			css = append(css, rel)
		case strings.HasSuffix(rel, ".html"):
			html = append(html, rel)
		case strings.HasSuffix(rel, ".js"):
			js = append(js, rel)
		case strings.HasSuffix(rel, "/panel.json"):
			manifests = append(manifests, rel)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	l.tokens = map[string]bool{}
	parsed := map[string][]*block{}
	for _, f := range css {
		src, err := os.ReadFile(filepath.Join(l.Root, f))
		if err != nil {
			return nil, err
		}
		bs, err := parseCSS(string(src))
		if err != nil {
			l.add(f, 0, "css-parse", "%v", err)
			continue
		}
		parsed[f] = bs
		if f == "tokens.css" || f == "shell.css" {
			for _, b := range bs {
				for _, d := range b.decls {
					if strings.HasPrefix(d.prop, "--d-") {
						l.tokens[d.prop] = true
					}
				}
			}
		}
	}
	for _, f := range css {
		if bs, ok := parsed[f]; ok {
			l.lintCSS(f, bs)
		}
	}
	l.lintLayerOrder()
	for _, f := range html {
		b, err := os.ReadFile(filepath.Join(l.Root, f))
		if err != nil {
			return nil, err
		}
		l.lintHTML(f, string(b), 1, true)
	}
	for _, f := range js {
		b, err := os.ReadFile(filepath.Join(l.Root, f))
		if err != nil {
			return nil, err
		}
		l.lintJS(f, string(b))
	}
	l.lintManifests(manifests)
	if l.GoDir != "" {
		if err := l.lintGo(); err != nil {
			return nil, err
		}
	}
	sort.SliceStable(l.findings, func(i, j int) bool {
		a, b := l.findings[i], l.findings[j]
		if a.File != b.File {
			return a.File < b.File
		}
		return a.Line < b.Line
	})
	return l.findings, nil
}

// panelOf returns the panel id of panels/<id>/..., or "".
func panelOf(f string) string {
	p := strings.Split(f, "/")
	if len(p) >= 3 && p[0] == "panels" {
		return p[1]
	}
	return ""
}

func (l *Linter) lintCSS(f string, bs []*block) {
	panel := panelOf(f)
	want, known := fileLayer[f]
	if panel != "" {
		want, known = "panels", true
	}
	panelVars := map[string]bool{}
	if panel != "" {
		for _, b := range bs {
			for _, d := range b.decls {
				if strings.HasPrefix(d.prop, "--p-") {
					panelVars[d.prop] = true
				}
			}
		}
	}
	for _, b := range bs {
		if b.at == "layer" {
			if b.stmt || strings.Contains(b.prelude, ",") {
				l.layerStmts = append(l.layerStmts, Finding{File: f, Line: b.line, Msg: b.prelude})
				continue
			}
			name := strings.TrimSpace(b.prelude)
			switch {
			case known && name != want:
				l.add(f, b.line, "css-layer-file", "@layer %s in %s (only @layer %s here)", name, f, want)
			case !known && !contains(Layers, name):
				l.add(f, b.line, "css-layer-file", "@layer %s is not one of %s", name, strings.Join(Layers, ", "))
			}
			continue
		}
		if b.at == "media" && (panel != "" || f == "components.css") {
			l.add(f, b.line, "css-no-media-in-panel", "@media in %s (container queries only; media queries live in shell.css)", f)
		}
		if b.at != "" || b.inside("keyframes") != nil {
			continue
		}
		if b.inside("layer") == nil {
			l.add(f, b.line, "css-layered", "rule %q is outside any @layer", short(b.prelude))
		}
		for _, sel := range selectors(b.prelude) {
			clean := stripAttrValues(sel)
			if reIDSel.MatchString(clean) {
				l.add(f, b.line, "css-no-id", "id selector %q (ids are never styled)", sel)
			}
			if c := firstRune(sel); f != "reset.css" && (isLetter(c) || c == '*') {
				l.add(f, b.line, "css-element-only-in-reset", "selector %q starts with an element; element selectors live in reset.css", sel)
			}
			if panel != "" {
				root := `[data-panel="` + panel + `"]`
				if !strings.HasPrefix(sel, root) {
					l.add(f, b.line, "css-panel-scope", "selector %q does not start with %s", sel, root)
				}
			}
		}
		isRoot := panel != "" && strings.TrimSpace(b.prelude) == `[data-panel="`+panel+`"]`
		for _, d := range b.decls {
			if strings.Contains(d.bare, "!important") {
				l.add(f, d.line, "css-no-important", "%s: !important", d.prop)
			}
			if panel != "" {
				if d.prop == "z-index" || d.prop == "position" && (strings.Contains(d.value, "fixed") || strings.Contains(d.value, "sticky")) {
					l.add(f, d.line, "css-no-fixed-in-panel", "%s: %s in a panel (overlays are the shell's)", d.prop, d.value)
				}
				if isRoot && rootSizeProps[d.prop] {
					l.add(f, d.line, "css-no-root-size", "%s on the panel root (the shell sizes and places panels)", d.prop)
				}
				if reViewport.MatchString(d.bare) {
					l.add(f, d.line, "css-no-vw-in-panel", "%s: %s uses a viewport unit (use cqi, cqb or rem)", d.prop, d.value)
				}
			}
			if f != "tokens.css" && hasColourLiteral(d.bare) {
				l.add(f, d.line, "css-no-colour-literal", "%s: %s (colours come from tokens.css)", d.prop, d.value)
			}
			for _, m := range reVar.FindAllStringSubmatch(d.value, -1) {
				v := m[1]
				switch {
				case strings.HasPrefix(v, "--d-") && !l.tokens[v]:
					l.add(f, d.line, "css-token-exists", "%s is not defined in tokens.css or shell.css", v)
				case panel != "" && strings.HasPrefix(v, "--p-") && (!strings.HasPrefix(v, "--p-"+panel+"-") || !panelVars[v]):
					l.add(f, d.line, "css-token-exists", "%s is not a property of panel %s defined in this file", v, panel)
				}
			}
		}
	}
}

func (l *Linter) lintLayerOrder() {
	want := strings.Join(Layers, ", ")
	if len(l.layerStmts) == 0 {
		l.add("tokens.css", 1, "css-layered", "the layer order statement `@layer %s;` is missing", want)
		return
	}
	for i, s := range l.layerStmts {
		got := strings.Join(strings.Fields(strings.ReplaceAll(s.Msg, ",", " ")), ", ")
		switch {
		case s.File != "tokens.css":
			l.add(s.File, s.Line, "css-layered", "a layer order statement outside tokens.css")
		case i > 0:
			l.add(s.File, s.Line, "css-layered", "the layer order is declared more than once")
		case got != want:
			l.add(s.File, s.Line, "css-layered", "layer order %q, want %q", got, want)
		}
	}
}

func hasColourLiteral(v string) bool {
	if reHexColour.MatchString(v) || reColourFn.MatchString(v) {
		return true
	}
	for _, w := range reIdent.FindAllString(v, -1) {
		if namedColours[strings.ToLower(w)] {
			return true
		}
	}
	return false
}

// lintHTML checks markup; page is false for Go template strings (those may hold fragments, so data-panel is fine).
func (l *Linter) lintHTML(f, s string, firstLine int, page bool) {
	at := func(i int) int { return firstLine + strings.Count(s[:i], "\n") }
	for _, m := range reStyleAttr.FindAllStringIndex(s, -1) {
		l.add(f, at(m[0]), "html-no-style-attr", "style attribute (CSP style-src 'self' blocks it; use a class)")
	}
	for _, m := range reStyleEl.FindAllStringIndex(s, -1) {
		l.add(f, at(m[0]), "html-no-style-element", "<style> element (stylesheets are files under web/)")
	}
	for _, m := range reScriptTag.FindAllStringIndex(s, -1) {
		if !strings.Contains(strings.ToLower(s[m[0]:m[1]]), "src=") {
			l.add(f, at(m[0]), "html-no-inline-script", "inline <script> (scripts are files under web/)")
		}
	}
	for _, m := range reOnAttr.FindAllStringIndex(s, -1) {
		l.add(f, at(m[0]), "html-no-inline-script", "inline event handler attribute")
	}
	if page {
		for _, m := range reDataPanel.FindAllStringIndex(s, -1) {
			l.add(f, at(m[0]), "html-panel-root", "data-panel in static HTML (shell.js creates panel roots)")
		}
	}
}

func (l *Linter) lintJS(f, s string) {
	lines := strings.Split(s, "\n")
	panel := panelOf(f)
	for i, ln := range lines {
		n := i + 1
		code := stripJSComment(ln)
		if reHTMLSink.MatchString(code) {
			l.add(f, n, "js-text-only", "HTML sink (build DOM with textContent)")
		}
		if reStyleBad.MatchString(code) {
			l.add(f, n, "js-no-inline-style", "inline style or injected <style>")
		}
		for _, m := range reStyleWrite.FindAllStringSubmatch(code, -1) {
			if !styleWriteOK[m[1]] && m[1] != "cssText" {
				l.add(f, n, "js-no-inline-style", ".style.%s write (allowed: gridArea, gridColumn, gridRow, transform, setProperty('--...'))", m[1])
			}
		}
		for _, m := range reSetProp.FindAllStringSubmatch(code, -1) {
			if !strings.HasPrefix(m[2], "--") {
				l.add(f, n, "js-no-inline-style", "style.setProperty(%q) (custom properties only)", m[2])
			}
		}
		if panel == "" {
			continue
		}
		if reGlobal.MatchString(code) {
			l.add(f, n, "js-no-globals", "window global in a panel module")
		}
		if strings.HasPrefix(strings.TrimSpace(code), "var ") && !strings.HasPrefix(ln, " ") && !strings.HasPrefix(ln, "\t") {
			l.add(f, n, "js-no-globals", "top-level var in a panel module (use const or let)")
		}
		if m := reTimer.FindStringSubmatch(code); m != nil {
			l.add(f, n, "js-no-timer", "%s in a panel (the shell schedules; use ctx.later or the manifest poll)", m[1])
		}
		if m := reDocQuery.FindStringSubmatch(code); m != nil {
			l.add(f, n, "js-no-query-outside-root", "document.%s in a panel (query inside root)", m[1])
		}
	}
	if panel != "" && strings.HasSuffix(f, "/panel.js") && !strings.Contains(s, "export function mount") {
		l.add(f, 1, "js-no-globals", "panel.js must be an ES module exporting mount(root, ctx)")
	}
}

func stripJSComment(s string) string {
	if t := strings.TrimSpace(s); strings.HasPrefix(t, "//") || strings.HasPrefix(t, "*") || strings.HasPrefix(t, "/*") {
		return ""
	}
	return s
}

// Manifest is a panel's panel.json (docs/ui-design-system.md section 6.1).
type Manifest struct {
	Schema      string  `json:"schema"`
	Version     *int    `json:"version"`
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Band        string  `json:"band"`
	Size        string  `json:"size"`
	MinSize     string  `json:"minSize"`
	MaxSize     string  `json:"maxSize"`
	Priority    *int    `json:"priority"`
	Tab         string  `json:"tab"`
	DefaultOpen *bool   `json:"defaultOpen"`
	Endpoint    *string `json:"endpoint"`
	Poll        float64 `json:"poll"`
	Unlock      string  `json:"unlock"`
	Skeleton    int     `json:"skeleton"`
	Owner       string  `json:"owner"`
	Doc         string  `json:"doc"`
}

// ShellConf is web/shell.json: the fixed panel ids of the status and stage bands.
type ShellConf struct {
	Status []string `json:"status"`
	Stage  []string `json:"stage"`
}

var (
	sizes = map[string]int{"s": 1, "m": 2, "l": 3}
	tabs  = map[string]bool{"hero": true, "orders": true, "story": true, "discover": true, "settings": true}
)

func (l *Linter) lintManifests(files []string) {
	var shell ShellConf
	if b, err := os.ReadFile(filepath.Join(l.Root, "shell.json")); err == nil {
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&shell); err != nil {
			l.add("shell.json", 1, "manifest-schema", "%v", err)
		}
	}
	ladder := l.ladderIDs()
	seen := map[string]string{}
	prio := map[string]map[int]string{}
	dirs := map[string]bool{}
	for _, f := range files {
		id := panelOf(f)
		dirs[id] = true
		b, err := os.ReadFile(filepath.Join(l.Root, f))
		if err != nil {
			l.add(f, 1, "manifest-schema", "%v", err)
			continue
		}
		var m Manifest
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			l.add(f, 1, "manifest-schema", "%v", err)
			continue
		}
		if m.Schema != "watch.panel/1" {
			l.add(f, 1, "manifest-schema", "schema %q, want watch.panel/1", m.Schema)
		}
		for k, v := range map[string]string{"title": m.Title, "band": m.Band, "size": m.Size, "tab": m.Tab, "owner": m.Owner, "doc": m.Doc} {
			if v == "" {
				l.add(f, 1, "manifest-schema", "missing %s", k)
			}
		}
		if m.Priority == nil || m.DefaultOpen == nil || m.Endpoint == nil {
			l.add(f, 1, "manifest-schema", "missing priority, defaultOpen or endpoint")
			continue
		}
		if m.Version == nil || *m.Version < 1 {
			l.add(f, 1, "manifest-version", "version must be an integer from 1 (the panel's revision; compatibility promise, docs/ui-design-system.md section 0)")
		}
		if strings.ContainsAny(m.Title, "<>&") {
			l.add(f, 1, "manifest-schema", "title holds markup")
		}
		if !rePanelID.MatchString(m.ID) || m.ID != id {
			l.add(f, 1, "manifest-id", "id %q must match ^[a-z][a-z0-9-]*$ and equal the directory %q", m.ID, id)
		}
		if other, dup := seen[m.ID]; dup {
			l.add(f, 1, "manifest-id", "id %q also in %s", m.ID, other)
		}
		seen[m.ID] = f
		lo, sz, hi := sizes[m.MinSize], sizes[m.Size], sizes[m.MaxSize]
		if m.MinSize == "" {
			lo = sz
		}
		if m.MaxSize == "" {
			hi = sz
		}
		if sz == 0 || lo == 0 || hi == 0 || lo > sz || sz > hi {
			l.add(f, 1, "manifest-size", "size %q min %q max %q (s, m or l, min <= size <= max)", m.Size, m.MinSize, m.MaxSize)
		}
		switch m.Band {
		case "status", "stage":
			list := shell.Status
			if m.Band == "stage" {
				list = shell.Stage
			}
			if !contains(list, m.ID) {
				l.add(f, 1, "manifest-band", "%s is not a %s panel in shell.json", m.ID, m.Band)
			}
		case "grid", "side":
		default:
			l.add(f, 1, "manifest-band", "band %q (status, stage, side or grid)", m.Band)
		}
		if p := *m.Priority; p < 0 || p > 999 {
			l.add(f, 1, "manifest-priority", "priority %d outside 0 to 999", p)
		} else {
			if prio[m.Band] == nil {
				prio[m.Band] = map[int]string{}
			}
			if other, dup := prio[m.Band][p]; dup {
				l.add(f, 1, "manifest-priority", "priority %d also used by %s in band %s", p, other, m.Band)
			}
			prio[m.Band][p] = m.ID
		}
		if !tabs[m.Tab] {
			l.add(f, 1, "manifest-tab", "tab %q (hero, orders, story, discover, settings)", m.Tab)
		}
		if e := *m.Endpoint; e != "" && !strings.HasPrefix(e, "/api/") || e == "" && m.Poll != 0 || m.Poll < 0 {
			l.add(f, 1, "manifest-endpoint", "endpoint %q poll %v (an /api/ path, or empty with poll 0)", e, m.Poll)
		}
		if m.Unlock != "" && ladder != nil && !ladder[m.Unlock] {
			l.add(f, 1, "manifest-unlock", "unlock %q is not in the ladder", m.Unlock)
		}
		if m.Skeleton < 0 || m.Skeleton > 8 {
			l.add(f, 1, "manifest-schema", "skeleton %d outside 0 to 8", m.Skeleton)
		}
		if m.Doc != "" && l.Repo != "" {
			if _, err := os.Stat(filepath.Join(l.Repo, m.Doc)); err != nil {
				l.add(f, 1, "manifest-doc", "doc %s does not exist", m.Doc)
			}
		}
		for _, need := range []string{"panel.css", "panel.js"} {
			if _, err := os.Stat(filepath.Join(l.Root, "panels", id, need)); err != nil {
				l.add(f, 1, "manifest-orphan", "panels/%s has no %s", id, need)
			}
		}
	}
	ents, _ := os.ReadDir(filepath.Join(l.Root, "panels"))
	for _, e := range ents {
		if e.IsDir() && !dirs[e.Name()] {
			l.add("panels/"+e.Name(), 0, "manifest-orphan", "panel directory without panel.json")
		}
	}
}

func (l *Linter) ladderIDs() map[string]bool {
	if l.Ladder == "" {
		return nil
	}
	b, err := os.ReadFile(l.Ladder)
	if err != nil {
		return nil
	}
	var v struct {
		Entries []struct {
			ID string `json:"id"`
		} `json:"entries"`
	}
	if json.Unmarshal(b, &v) != nil {
		return nil
	}
	ids := map[string]bool{}
	for _, e := range v.Entries {
		ids[e.ID] = true
	}
	return ids
}

// lintGo applies the HTML rules to backtick strings that hold markup in the non-test Go files of GoDir.
func (l *Linter) lintGo() error {
	ents, err := os.ReadDir(l.GoDir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		n := e.Name()
		if e.IsDir() || !strings.HasSuffix(n, ".go") || strings.HasSuffix(n, "_test.go") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(l.GoDir, n))
		if err != nil {
			return err
		}
		s := string(b)
		for _, m := range reBacktick.FindAllStringIndex(s, -1) {
			lit := s[m[0]:m[1]]
			if !strings.Contains(lit, "<") || !strings.Contains(lit, ">") {
				continue
			}
			l.lintHTML("go:"+n, lit, 1+strings.Count(s[:m[0]], "\n"), false)
		}
	}
	return nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func short(s string) string {
	if len(s) > 60 {
		return s[:57] + "..."
	}
	return s
}

func firstRune(s string) rune {
	for _, c := range s {
		return c
	}
	return 0
}

func isLetter(c rune) bool { return c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' }

// stripAttrValues blanks [attr=value] contents so a '#' inside an attribute value is not an id selector.
func stripAttrValues(s string) string {
	var b strings.Builder
	depth := 0
	for _, c := range s {
		switch {
		case c == '[':
			depth++
			b.WriteRune(c)
		case c == ']':
			depth--
			b.WriteRune(c)
		case depth == 0:
			b.WriteRune(c)
		}
	}
	return b.String()
}
