package watch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeInv struct {
	marks []string
	err   error
}
type refusal struct{ msg string }

func (r refusal) Error() string { return r.msg }
func (r refusal) Kind() string  { return "town" }

func (f *fakeInv) View() any { return map[string]any{"schema": 1} }
func (f *fakeInv) SetMark(id uint32, m string) error {
	f.marks = append(f.marks, m)
	return f.err
}
func (f *fakeInv) RequestEquip(id uint32, slot string) (any, error) { return map[string]any{"slot": slot}, f.err }
func (f *fakeInv) Unpin(slot string) error                          { return f.err }

func invPost(h http.Handler, path, body string, hdr map[string]string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		r.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestInventoryRoutesGateAndReplyWithReasons(t *testing.T) {
	api := NewInvAPI()
	o := AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}}
	post := api.PostHandler(o)
	auth := map[string]string{"Authorization": "Bearer tok"}

	if w := invPost(post, "/api/inventory/mark", `{"id":1,"mark":"sell"}`, auth); w.Code != 503 {
		t.Errorf("desk not bound: %d", w.Code)
	}
	f := &fakeInv{}
	api.Bind(f)
	if w := invPost(post, "/api/inventory/mark", `{"id":1,"mark":"sell"}`, auth); w.Code != 200 || len(f.marks) != 1 {
		t.Errorf("mark: %d %s", w.Code, w.Body)
	}
	if w := invPost(post, "/api/inventory/mark", `{"id":1,"mark":"sell"}`, nil); w.Code != 401 {
		t.Errorf("no token: %d", w.Code)
	}
	bad := map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.test"}
	if w := invPost(post, "/api/inventory/equip", `{"id":1,"slot":"rhand"}`, bad); w.Code != 403 || len(f.marks) != 1 {
		t.Errorf("foreign origin: %d", w.Code)
	}
	if w := invPost(post, "/api/inventory/equip", `{"id":1,"slot":"rhand","x":1}`, auth); w.Code != 400 {
		t.Errorf("unknown field: %d", w.Code)
	}
	f.err = refusal{"the hero is not in town"}
	w := invPost(post, "/api/inventory/equip", `{"id":1,"slot":"rhand"}`, auth)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "not in town") || !strings.Contains(w.Body.String(), `"town"`) {
		t.Errorf("refusal: %d %s", w.Code, w.Body)
	}
	f.err = errors.New("disk full")
	if w := invPost(post, "/api/inventory/unpin", `{"slot":"rhand"}`, auth); w.Code != 500 || strings.Contains(w.Body.String(), "disk") {
		t.Errorf("internal error leaks detail: %d %s", w.Code, w.Body)
	}

	view := api.Handler()
	for _, p := range []string{"/api/inventory", "/watch/inventory", "/watch/inventory.js"} {
		w := httptest.NewRecorder()
		view.ServeHTTP(w, httptest.NewRequest("GET", p, nil))
		if w.Code != 200 {
			t.Errorf("GET %s: %d", p, w.Code)
		}
	}
	w = httptest.NewRecorder()
	view.ServeHTTP(w, httptest.NewRequest("POST", "/api/inventory", nil))
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("viewer accepts POST: %d", w.Code)
	}
}
