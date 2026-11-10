package watch

import (
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeFarm struct {
	on  bool
	err error
}

func (f *fakeFarm) View() any { return map[string]any{"schema": 1, "on": f.on} }
func (f *fakeFarm) Set(on bool) error {
	if f.err == nil {
		f.on = on
	}
	return f.err
}

func TestGoldFarmRoutesGate(t *testing.T) {
	api := NewGoldFarmAPI()
	o := AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}}
	post := api.PostHandler(o)
	auth := map[string]string{"Authorization": "Bearer tok"}
	if w := invPost(post, GoldFarmRoute, `{"on":true}`, auth); w.Code != 503 {
		t.Errorf("desk not bound: %d", w.Code)
	}
	f := &fakeFarm{}
	api.Bind(f)
	if w := invPost(post, GoldFarmRoute, `{"on":true}`, auth); w.Code != 200 || !f.on || !strings.Contains(w.Body.String(), `"on":true`) {
		t.Errorf("start: %d %s", w.Code, w.Body)
	}
	if w := invPost(post, GoldFarmRoute, `{"on":false}`, auth); w.Code != 200 || f.on {
		t.Errorf("stop: %d", w.Code)
	}
	for name, c := range map[string]struct {
		body string
		hdr  map[string]string
		code int
	}{
		"no token":       {`{"on":true}`, nil, 401},
		"foreign origin": {`{"on":true}`, map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.test"}, 403},
		"unknown field":  {`{"on":true,"x":1}`, auth, 400},
		"missing on":     {`{}`, auth, 400},
	} {
		if w := invPost(post, GoldFarmRoute, c.body, c.hdr); w.Code != c.code {
			t.Errorf("%s: %d, want %d", name, w.Code, c.code)
		}
	}
	f.err = errors.New("disk full /secret/path")
	if w := invPost(post, GoldFarmRoute, `{"on":true}`, auth); w.Code != 500 || strings.Contains(w.Body.String(), "secret") {
		t.Errorf("internal error leaks detail: %d %s", w.Code, w.Body)
	}
	view := api.Handler()
	w := httptest.NewRecorder()
	view.ServeHTTP(w, httptest.NewRequest("GET", GoldFarmRoute, nil))
	if w.Code != 200 {
		t.Errorf("GET: %d", w.Code)
	}
	w = httptest.NewRecorder()
	view.ServeHTTP(w, httptest.NewRequest("POST", GoldFarmRoute, nil))
	if w.Code != 405 {
		t.Errorf("viewer accepts POST: %d", w.Code)
	}
}

// The gear panel's Farm gold part against a stub DOM (node): golden testdata/gear_farm_dom.txt, UPDATE=1 rewrites it.
func TestGearPanelFarmGoldDOMGolden(t *testing.T) {
	runNode(t, "testdata/gear_farm_dom.js", "web/panels/gear/panel.js")
}
