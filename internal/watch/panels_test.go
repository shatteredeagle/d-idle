package watch

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
)

// Every manifest parses, sits in its band in priority order, names a route some handler of this package mounts,
// and the status and stage ids are exactly the shell's fixed list (web/shell.json).
func TestPanelManifests(t *testing.T) {
	ms, err := Manifests()
	if err != nil {
		t.Fatal(err)
	}
	var shell struct{ Status, Stage []string }
	b, err := os.ReadFile("web/shell.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, &shell); err != nil {
		t.Fatal(err)
	}
	routes := map[string]bool{"/api/tally": true, AlertsRoute: true} // mounted by Fun.mountView (funwire.go)
	for _, list := range [][]string{ViewRoutes, SensesRoutes, RepairRoutes, UpdateRoutes, PolicyRoutes, ReturnRoutes, PactRoutes, InvRoutes, CampaignRoutes} {
		for _, r := range list {
			routes[r] = true
		}
	}
	got := map[string][]string{}
	last := map[string]int{}
	for _, m := range ms {
		if p, ok := last[m.Band]; ok && m.Priority < p {
			t.Errorf("%s out of priority order in %s", m.ID, m.Band)
		}
		last[m.Band] = m.Priority
		got[m.Band] = append(got[m.Band], m.ID)
		if m.Endpoint != "" && !routes[m.Endpoint] {
			t.Errorf("%s: endpoint %s is not a route of internal/watch", m.ID, m.Endpoint)
		}
		for _, f := range []string{"panel.js", "panel.css"} {
			if _, err := webFS.ReadFile("web/panels/" + m.ID + "/" + f); err != nil {
				t.Errorf("%s: %v", m.ID, err)
			}
		}
	}
	if strings.Join(got["status"], ",") != strings.Join(shell.Status, ",") || strings.Join(got["stage"], ",") != strings.Join(shell.Stage, ",") {
		t.Errorf("bands %v, shell.json %+v", got, shell)
	}
}

// The registry and the panel assets are served by the view handler with the page CSP; unknown ids are 404.
func TestPanelRegistryServed(t *testing.T) {
	r := newRig(t, "sorceress")
	var v PanelsView
	w := r.get("/watch/panels.json", &v)
	if w.Code != 200 || v.Schema != "watch.panels/1" || len(v.Panels) < 4 {
		t.Fatalf("panels.json: %d %+v", w.Code, v)
	}
	for _, m := range v.Panels {
		for _, f := range []string{"panel.js", "panel.css"} {
			res := r.get("/watch/panels/"+m.ID+"/"+f, nil)
			if res.Code != 200 || !strings.Contains(res.Header().Get("Content-Security-Policy"), "script-src 'self'") {
				t.Errorf("%s/%s: %d", m.ID, f, res.Code)
			}
		}
	}
	for _, bad := range []string{"/watch/panels/nope/panel.js", "/watch/panels/map/panel.json", "/watch/panels/../app.js", "/watch/panels/Map/panel.js"} {
		if res := r.get(bad, nil); res.Code == 200 { // 404, or the mux redirect for a path with ..
			t.Errorf("%s: %d", bad, res.Code)
		}
	}
	if res := r.get("/watch/shell.js", nil); res.Code != 200 || !strings.HasPrefix(res.Header().Get("Content-Type"), "text/javascript") {
		t.Errorf("shell.js: %d", res.Code)
	}
}

// Panel modules keep the page's text-only rule.
func TestPanelModulesTextOnly(t *testing.T) {
	ms, _ := Manifests()
	for _, m := range ms {
		b, _ := webFS.ReadFile("web/panels/" + m.ID + "/panel.js")
		for _, bad := range []string{"innerHTML", "outerHTML", "insertAdjacentHTML", "document.write", "eval(", "new Function"} {
			if strings.Contains(string(b), bad) {
				t.Errorf("%s uses %s", m.ID, bad)
			}
		}
	}
}
