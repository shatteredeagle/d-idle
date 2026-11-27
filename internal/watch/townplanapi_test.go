package watch

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/supply"
)

func TestTownPlanAPI(t *testing.T) {
	a := NewTownPlanAPI()
	h := a.Handler()
	get := func(method string) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(method, TownPlanRoute, nil))
		return w
	}
	if w := get("GET"); w.Code != http.StatusServiceUnavailable {
		t.Fatalf("unbound: %d", w.Code)
	}
	// recorded plans of two builds (mana need 2 and 0)
	for name, in := range map[string]supply.Inputs{
		"caster": {ManaNeed: 2, Capacity: map[supply.Line]int{supply.Healing: 4, supply.Mana: 4}},
		"melee":  {ManaNeed: 0, Capacity: map[supply.Line]int{supply.Healing: 4}},
	} {
		n, err := supply.NeedFor("quest", "monastery_road", in)
		if err != nil {
			t.Fatal(err)
		}
		c := supply.Count{Purse: 40, StashKnown: true}
		v := supply.ViewOf(supply.Solve(n, c, supply.Prices{supply.Healing: 30, supply.Mana: 30, supply.Portal: 100}, supply.Policy{}, 0), c, []string{"akara"}, 0)
		a.Bind(func() supply.View { return v })
		w := get("GET")
		var got supply.View
		if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &got) != nil || got.Schema != "watch.townplan/1" || !got.Known {
			t.Fatalf("%s: %d %s", name, w.Code, w.Body.String())
		}
		if got.Chip != "Short of gold" || got.Waiting == "" || len(got.Lines) != len(v.Lines) {
			t.Errorf("%s: %+v", name, got)
		}
		if w.Header().Get("Cache-Control") != "no-store" {
			t.Error("cache header")
		}
	}
	a.Bind(supply.NoView)
	var nv supply.View
	if w := get("GET"); json.Unmarshal(w.Body.Bytes(), &nv) != nil || nv.Known || nv.Waiting == "" {
		t.Errorf("no view: %s", w.Body.String())
	}
	if w := get("POST"); w.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST must be refused: %d", w.Code)
	}
}

// The Town plan card's text helpers (web/panels/town-plan/panel.js, an ES module) on recorded plans of two builds (node, skipped without it).
func TestTownPlanCardText(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node")
	}
	dir := t.TempDir()
	src, _ := webFS.ReadFile("web/panels/town-plan/panel.js")
	os.WriteFile(filepath.Join(dir, "panel.mjs"), src, 0o600)
	for name, in := range map[string]supply.Inputs{
		"caster": {ManaNeed: 2, Capacity: map[supply.Line]int{supply.Healing: 4, supply.Mana: 4}},
		"melee":  {ManaNeed: 0, Capacity: map[supply.Line]int{supply.Healing: 4}},
	} {
		n, _ := supply.NeedFor("quest", "monastery_road", in)
		c := supply.Count{Purse: 5000, StashKnown: true}
		v := supply.ViewOf(supply.Solve(n, c, supply.Prices{supply.Healing: 30, supply.Mana: 30, supply.Portal: 100}, supply.Policy{}, 0), c, []string{"akara", "stash"}, 1)
		js, _ := json.Marshal(v)
		os.WriteFile(filepath.Join(dir, name+".json"), js, 0o600)
	}
	script := `import {lineText, stopsText, budgetText} from './panel.mjs';
import fs from 'fs';
const out = {};
for (const n of ['caster','melee']) { const v = JSON.parse(fs.readFileSync(n + '.json')); out[n] = {lines: v.lines.map(lineText), stops: stopsText(v.stops, v.stopsDone), budget: budgetText(v.budget)}; }
console.log(JSON.stringify(out));`
	os.WriteFile(filepath.Join(dir, "run.mjs"), []byte(script), 0o600)
	cmd := exec.Command(node, "run.mjs")
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}
	var res map[string]struct {
		Lines  []string
		Stops  string
		Budget string
	}
	if err := json.Unmarshal(out, &res); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	if len(res["caster"].Lines) <= len(res["melee"].Lines) {
		t.Errorf("the caster plan has a mana row the melee plan lacks: %v / %v", res["caster"].Lines, res["melee"].Lines)
	}
	for name, r := range res {
		if r.Stops != "akara (done), stash" || !strings.HasPrefix(r.Budget, "Purse 5000, stash 0, spending ") {
			t.Errorf("%s: %q %q", name, r.Stops, r.Budget)
		}
		if len(r.Lines) == 0 || !strings.HasPrefix(r.Lines[0], "healing potions: have 0, need 4") || !strings.Contains(r.Lines[0], "buy ") {
			t.Errorf("%s: lines %v", name, r.Lines)
		}
	}
}
