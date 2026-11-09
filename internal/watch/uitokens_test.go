package watch

import (
	"os"
	"strings"
	"testing"
)

// The design tokens default to the dark Diablo II palette and dense spacing (user, 2026-11-09); light and auto are
// options on <html data-theme>. The old app.css palette maps onto the tokens, so the old cards follow the switch.
func TestDesignTokensDarkDefaultDense(t *testing.T) {
	b, err := os.ReadFile("web/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	css := string(b)
	if strings.Count(css, "@layer reset, tokens, shell, components, panels, utilities;") != 1 {
		t.Error("tokens.css must declare the layer order once")
	}
	root := css[strings.Index(css, ":root {"):]
	root = root[:strings.Index(root, "}")]
	for _, want := range []string{"--d-bg: var(--d-stone-1)", "--d-accent: var(--d-gold-9)", "color-scheme: dark", "--d-gap: var(--d-sp-3)", "--d-pad: var(--d-sp-3)"} {
		if !strings.Contains(root, want) {
			t.Errorf("the default :root lacks %q", want)
		}
	}
	for _, want := range []string{`:root[data-theme="light"]`, `:root[data-theme="auto"]`, `:root[data-density="comfortable"]`} {
		if !strings.Contains(css, want) {
			t.Errorf("tokens.css lacks %s", want)
		}
	}
	app, _ := os.ReadFile("web/app.css")
	if !strings.Contains(string(app), "--bg: var(--d-bg)") || strings.Contains(string(app), "prefers-color-scheme") {
		t.Error("app.css must map its palette onto the tokens and leave theme switching to tokens.css")
	}
	js, _ := os.ReadFile("web/shell.js")
	if !strings.Contains(string(js), "localStorage.getItem('d2theme') || 'dark'") {
		t.Error("the theme switch must default to dark")
	}
}

// Every stylesheet of the design system is served from /watch/ with the page CSP, and the page loads them.
func TestDesignSystemAssetsServed(t *testing.T) {
	r := newRig(t, "sorceress")
	page := r.get("/watch", nil).Body.String()
	for _, f := range []string{"tokens.css", "reset.css", "app.css", "shell.css", "components.css", "utilities.css"} {
		w := r.get("/watch/"+f, nil)
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), "text/css") || !strings.Contains(w.Body.String(), "@layer") {
			t.Errorf("/watch/%s: %d %q", f, w.Code, w.Header().Get("Content-Type"))
		}
		if !strings.Contains(page, `href="/watch/`+f+`"`) {
			t.Errorf("/watch does not load %s", f)
		}
	}
	if strings.Index(page, "tokens.css") > strings.Index(page, "app.css") {
		t.Error("tokens.css must load first (it declares the layer order)")
	}
}
