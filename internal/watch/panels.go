package watch

import (
	"bytes"
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
	Owner       string  `json:"owner"`
	Doc         string  `json:"doc"`
}

const panelSchema = "watch.panel/1"

// PanelsView is the /watch/panels.json reply: the manifests in placement order (band, then priority, then id).
type PanelsView struct {
	Schema string          `json:"schema"`
	Bands  []string        `json:"bands"`
	Panels []PanelManifest `json:"panels"`
}

var bandOrder = map[string]int{"status": 0, "stage": 1, "side": 2, "grid": 3}

// Manifests parses every embedded panel manifest. It refuses unknown keys, an id that is not its directory, and a
// band the shell does not have; the full rule set is tools/uilint's.
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
		var m PanelManifest
		dec := json.NewDecoder(bytes.NewReader(b))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&m); err != nil {
			return nil, fmt.Errorf("panel %s: %w", d.Name(), err)
		}
		if m.Schema != panelSchema || m.ID != d.Name() {
			return nil, fmt.Errorf("panel %s: schema %q id %q", d.Name(), m.Schema, m.ID)
		}
		if _, ok := bandOrder[m.Band]; !ok {
			return nil, fmt.Errorf("panel %s: band %q", m.ID, m.Band)
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
		writeJSON(w, 200, PanelsView{Schema: "watch.panels/1", Bands: []string{"status", "stage", "side", "grid"}, Panels: ms})
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
