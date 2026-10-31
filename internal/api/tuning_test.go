package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	denv "d2lod-bot/internal/env"
)

type fakeTune struct {
	body    TuneBody
	edit    TuneEdit
	unlocks []string
}

func (f *fakeTune) Snapshot() TuneBody { return f.body }
func (f *fakeTune) Edit(e TuneEdit) error {
	if e.Profile != nil && *e.Profile == "bad" {
		return ErrInvalidTune
	}
	f.edit = e
	return nil
}
func (f *fakeTune) Unlock(id string) error {
	if id == "none" {
		return ErrUnknownUpgrade
	}
	if id == "nodata" {
		return ErrNoUpgrades
	}
	f.unlocks = append(f.unlocks, id)
	return nil
}

func do(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, path, strings.NewReader(body))
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestKnobsAndUpgradesRoutes(t *testing.T) {
	f := &fakeTune{body: TuneBody{Knobs: []denv.Knob{{Key: "a.b", Default: 1}}, Values: map[string]float64{"a.b": 1}, Profile: "optimal", Upgrades: []Upgrade{}}}
	kh, uh := KnobsHandler(f), UpgradesHandler(f)
	w := do(kh, "GET", "/", "")
	var got TuneBody
	if w.Code != 200 || json.Unmarshal(w.Body.Bytes(), &got) != nil || got.Values["a.b"] != 1 || got.Profile != "optimal" {
		t.Fatalf("GET %d %s", w.Code, w.Body)
	}
	if w = do(kh, "PUT", "/", `{"profile":"casual","overrides":{"a.b":2,"c.d":null}}`); w.Code != 200 || f.edit.Profile == nil || *f.edit.Profile != "casual" || *f.edit.Overrides["a.b"] != 2 || f.edit.Overrides["c.d"] != nil {
		t.Fatalf("PUT %d %s %+v", w.Code, w.Body, f.edit)
	}
	if w = do(kh, "PUT", "/", `{"profile":"bad"}`); w.Code != 400 {
		t.Fatalf("bad edit %d", w.Code)
	}
	if w = do(kh, "PUT", "/", `{"nope":1}`); w.Code != 400 {
		t.Fatalf("unknown field %d", w.Code)
	}
	if w = do(kh, "DELETE", "/", ""); w.Code != 405 {
		t.Fatalf("DELETE %d", w.Code)
	}
	if w = do(uh, "POST", "/u1/unlock", ""); w.Code != 200 || len(f.unlocks) != 1 || f.unlocks[0] != "u1" {
		t.Fatalf("unlock %d %s", w.Code, w.Body)
	}
	for path, want := range map[string]int{"/none/unlock": 404, "/nodata/unlock": 501, "/u1": 404, "/": 404} {
		if w = do(uh, "POST", path, ""); w.Code != want {
			t.Fatalf("POST %s = %d, want %d", path, w.Code, want)
		}
	}
	if w = do(uh, "GET", "/u1/unlock", ""); w.Code != 405 {
		t.Fatalf("GET unlock %d", w.Code)
	}
}
