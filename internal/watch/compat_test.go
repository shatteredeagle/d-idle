package watch

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The compatibility promise (docs/ui-design-system.md section 0): the golden files under testdata/compat are today's
// public contracts. A failure here is a breaking change: add an alias, a stub or a schema version instead of editing
// a fixture. Fixtures are only added to.

func compatList(t *testing.T, name string) []string {
	t.Helper()
	f, err := os.Open(filepath.Join("testdata/compat", name))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	var out []string
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if l := strings.TrimSpace(sc.Text()); l != "" && !strings.HasPrefix(l, "#") {
			out = append(out, l)
		}
	}
	return out
}

// Every panel manifest of v1 still loads, and its panel still exists (a removed panel leaves a stub directory).
func TestCompatPanelManifestsLoad(t *testing.T) {
	files, _ := filepath.Glob("testdata/compat/panels-v1/*.json")
	if len(files) == 0 {
		t.Fatal("no v1 panel fixtures")
	}
	live := map[string]PanelManifest{}
	ms, err := Manifests()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range ms {
		live[m.ID] = m
	}
	for _, f := range files {
		id := strings.TrimSuffix(filepath.Base(f), ".json")
		b, _ := os.ReadFile(f)
		m, err := parseManifest(b, id)
		if err != nil {
			t.Errorf("v1 manifest %s no longer loads: %v", id, err)
			continue
		}
		cur, ok := live[id]
		switch {
		case !ok:
			t.Errorf("panel %s was removed; leave a stub panel (compatibility rule 4)", id)
		case cur.Endpoint != m.Endpoint && cur.Endpoint != "":
			t.Errorf("panel %s moved from %s to %s; keep the old endpoint or bump the panel's schema", id, m.Endpoint, cur.Endpoint)
		case cur.Version < m.Version:
			t.Errorf("panel %s version went down (%d < %d)", id, cur.Version, m.Version)
		}
	}
	// a manifest from a later version with a field this version does not know still loads
	b, _ := os.ReadFile(files[0])
	var raw map[string]any
	json.Unmarshal(b, &raw)
	raw["someFutureField"] = map[string]any{"x": 1}
	nb, _ := json.Marshal(raw)
	if _, err := parseManifest(nb, strings.TrimSuffix(filepath.Base(files[0]), ".json")); err != nil {
		t.Errorf("an unknown field broke the loader: %v", err)
	}
}

// A layout exported by the editor today (docs/mockups/ui/editor.html, the mockup preset) loads, also with extra keys.
func TestCompatLayoutManifestLoads(t *testing.T) {
	b, err := os.ReadFile("testdata/compat/layout-v1-mockup.json")
	if err != nil {
		t.Fatal(err)
	}
	l, err := ParseLayout(b)
	if err != nil {
		t.Fatal(err)
	}
	if len(l.Breakpoints["wide"].Panels) == 0 || l.Cols != 12 {
		t.Fatalf("layout read empty: %+v", l)
	}
	var raw map[string]any
	json.Unmarshal(b, &raw)
	raw["editorVersion"] = 9
	nb, _ := json.Marshal(raw)
	if _, err := ParseLayout(nb); err != nil {
		t.Errorf("an unknown key broke the layout loader: %v", err)
	}
	if _, err := ParseLayout([]byte(`{"schema":"watch.layout/2"}`)); err == nil {
		t.Error("a later major version must be refused, not misread")
	}
}

// Every v1 token name is still defined in tokens.css (a renamed token keeps an alias).
func TestCompatTokenNames(t *testing.T) {
	b, err := os.ReadFile("web/tokens.css")
	if err != nil {
		t.Fatal(err)
	}
	defined := map[string]bool{}
	for _, m := range regexp.MustCompile(`(--d-[a-z0-9-]+)\s*:`).FindAllStringSubmatch(string(b), -1) {
		defined[m[1]] = true
	}
	for _, tok := range compatList(t, "tokens-v1.txt") {
		if !defined[tok] {
			t.Errorf("token %s is gone; keep it as an alias (compatibility rule 3)", tok)
		}
	}
}

// Every v1 field of /api/hero is still in the reply (watch.hero/1 only grows).
func TestCompatHeroFields(t *testing.T) {
	r := newRig(t, "paladin")
	var v map[string]any
	r.get(HeroRoute, &v)
	if v["schema"] != heroSchema {
		t.Fatalf("schema %v", v["schema"])
	}
	for _, k := range compatList(t, "hero-v1.txt") {
		if _, ok := v[k]; !ok {
			t.Errorf("/api/hero lost the field %s; bump the schema instead", k)
		}
	}
}
