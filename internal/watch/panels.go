package watch

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"path"
	"sort"
	"strings"
)

// The panel registry (docs/ui-design-system.md sections 4.3 and 6): every web/panels/<id>/panel.json is a manifest,
// the shell (web/shell.js) reads them from /watch/panels.json and places the panels; tools/uilint checks the same
// files against the contract.

// PanelManifest is one web/panels/<id>/panel.json.
type PanelManifest struct {
	Schema      string  `json:"schema"`
	Version     int     `json:"version"` // the panel's own revision; raised when its DOM or data attributes change
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Band        string  `json:"band"`
	Size        string  `json:"size"`
	MinSize     string  `json:"minSize,omitempty"`
	MaxSize     string  `json:"maxSize,omitempty"`
	Priority    int     `json:"priority"`
	Tab         string  `json:"tab"`
	DefaultOpen bool    `json:"defaultOpen"`
	Endpoint    string  `json:"endpoint"`
	Poll        float64 `json:"poll"` // seconds; 0 = on the SSE change event plus the shell's safety poll
	Unlock      string  `json:"unlock"`
	Skeleton    int     `json:"skeleton"`
	// Fit: the panel's rows under a layout follow its rendered content, not the stored rowSpan (web/layout.js fitRows).
	Fit   bool   `json:"fit,omitempty"`
	Owner string `json:"owner"`
	Doc   string `json:"doc"`
}

const panelSchema = "watch.panel/1"

// PanelsView is the /watch/panels.json reply: the manifests in placement order (band, then priority, then id).
type PanelsView struct {
	Schema string          `json:"schema"`
	Bands  []string        `json:"bands"`
	Panels []PanelManifest `json:"panels"`
	// Layout is how the ids of a layout manifest map onto the page (web/shell.json, layout.go).
	Layout LayoutConf `json:"layout"`
}

var bandOrder = map[string]int{"status": 0, "stage": 1, "side": 2, "grid": 3}

// Manifests parses every embedded panel manifest. It refuses an id that is not its directory and a band the shell
// does not have; the full rule set is tools/uilint's.
func Manifests() ([]PanelManifest, error) {
	dirs, err := fs.ReadDir(webFS, "web/panels")
	if err != nil {
		return nil, nil // no panels yet
	}
	var out []PanelManifest
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		b, err := webFS.ReadFile("web/panels/" + d.Name() + "/panel.json")
		if err != nil {
			return nil, fmt.Errorf("panel %s: %w", d.Name(), err)
		}
		m, err := parseManifest(b, d.Name())
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Band != b.Band {
			return bandOrder[a.Band] < bandOrder[b.Band]
		}
		if a.Priority != b.Priority {
			return a.Priority < b.Priority
		}
		return a.ID < b.ID
	})
	return out, nil
}

// parseManifest reads one panel.json. Unknown keys are ignored (compatibility promise, docs/ui-design-system.md
// section 0: a manifest written by a later version still loads); tools/uilint is the strict check for this repo.
func parseManifest(b []byte, dir string) (PanelManifest, error) {
	var m PanelManifest
	if err := json.Unmarshal(b, &m); err != nil {
		return m, fmt.Errorf("panel %s: %w", dir, err)
	}
	if m.Schema != panelSchema || m.ID != dir {
		return m, fmt.Errorf("panel %s: schema %q id %q", dir, m.Schema, m.ID)
	}
	if _, ok := bandOrder[m.Band]; !ok {
		return m, fmt.Errorf("panel %s: band %q", m.ID, m.Band)
	}
	return m, nil
}

// servePanels answers /watch/panels.json and /watch/panels/<id>/panel.{js,css}; it reports false for other paths.
func servePanels(w http.ResponseWriter, r *http.Request, asset func(name, ctype string) http.HandlerFunc) bool {
	p := r.URL.Path
	if p == "/watch/panels.json" {
		ms, err := Manifests()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return true
		}
		if ms == nil {
			ms = []PanelManifest{}
		}
		lc, _ := shellLayoutConf()
		writeJSON(w, 200, PanelsView{Schema: "watch.panels/1", Bands: []string{"status", "stage", "side", "grid"}, Panels: ms, Layout: lc})
		return true
	}
	rest, ok := strings.CutPrefix(p, "/watch/panels/")
	if !ok {
		return false
	}
	id, file, _ := strings.Cut(rest, "/")
	if !rePanelDir(id) {
		http.NotFound(w, r)
		return true
	}
	switch file {
	case "panel.js":
		asset(path.Join("panels", id, file), "text/javascript; charset=utf-8")(w, r)
	case "panel.css":
		asset(path.Join("panels", id, file), "text/css; charset=utf-8")(w, r)
	default:
		http.NotFound(w, r)
	}
	return true
}

func rePanelDir(id string) bool {
	if id == "" || id[0] < 'a' || id[0] > 'z' {
		return false
	}
	for _, c := range id {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}
