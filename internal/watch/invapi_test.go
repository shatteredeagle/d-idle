package watch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeInv struct {
	marks    []string
	err      error
	trashed  []bool
	canceled []string
}
type refusal struct{ msg string }

func (r refusal) Error() string { return r.msg }
func (r refusal) Kind() string  { return "town" }

func (f *fakeInv) View() any { return map[string]any{"schema": 1} }
func (f *fakeInv) SetMark(id uint32, m string) error {
	f.marks = append(f.marks, m)
	return f.err
}
func (f *fakeInv) RequestEquip(id uint32, slot string) (any, error) {
	return map[string]any{"slot": slot}, f.err
}
func (f *fakeInv) Unpin(slot string) error { return f.err }
func (f *fakeInv) RequestTrash(id uint32, confirm bool) (any, error) {
	f.trashed = append(f.trashed, confirm)
	return map[string]any{"kind": "trash"}, f.err
}
func (f *fakeInv) Cancel(id uint32, kind string) error {
	f.canceled = append(f.canceled, kind)
	return f.err
}

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

type countInv struct {
	fakeInv
	views int
}

func (c *countInv) View() any { c.views++; return map[string]any{"schema": 1, "n": c.views} }

// One built view is shared by polls inside the TTL; a POST drops it so the page sees its own change at once.
func TestInventoryViewIsSharedAndDroppedByPost(t *testing.T) {
	api := NewInvAPI()
	c := &countInv{}
	api.Bind(c)
	get := func() string {
		w := httptest.NewRecorder()
		api.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/inventory", nil))
		if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || !strings.Contains(w.Header().Get("Content-Type"), "json") {
			t.Fatalf("%d %v", w.Code, w.Header())
		}
		return w.Body.String()
	}
	a, b := get(), get()
	if a != b || c.views != 1 {
		t.Errorf("polls inside the TTL rebuilt the view: %d builds", c.views)
	}
	post := api.PostHandler(AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}})
	if w := invPost(post, "/api/inventory/mark", `{"id":1,"mark":"sell"}`, map[string]string{"Authorization": "Bearer tok"}); w.Code != 200 {
		t.Fatal(w.Code)
	}
	if get() == a || c.views != 2 {
		t.Errorf("a POST must drop the cached view (%d builds)", c.views)
	}
}

// Schema 3 routes: trash needs the same gates, carries the confirm flag, cancel names the kind; an old client that only
// knows mark, equip and unpin is unaffected.
func TestInventoryTrashAndCancelRoutes(t *testing.T) {
	api := NewInvAPI()
	f := &fakeInv{}
	api.Bind(f)
	post := api.PostHandler(AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}})
	auth := map[string]string{"Authorization": "Bearer tok"}
	if w := invPost(post, "/api/inventory/trash", `{"id":1,"confirm":true}`, auth); w.Code != 200 || len(f.trashed) != 1 || !f.trashed[0] {
		t.Errorf("trash: %d %s", w.Code, w.Body)
	}
	if w := invPost(post, "/api/inventory/trash", `{"id":1}`, auth); w.Code != 200 || f.trashed[1] {
		t.Errorf("trash without confirm must reach the desk as false: %d", w.Code)
	}
	if w := invPost(post, "/api/inventory/cancel", `{"id":1,"kind":"trash"}`, auth); w.Code != 200 || f.canceled[0] != "trash" {
		t.Errorf("cancel: %d %v", w.Code, f.canceled)
	}
	if w := invPost(post, "/api/inventory/trash", `{"id":1,"confirm":true}`, nil); w.Code != 401 || len(f.trashed) != 2 {
		t.Errorf("trash without a token: %d", w.Code)
	}
	bad := map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.test"}
	if w := invPost(post, "/api/inventory/cancel", `{"id":1}`, bad); w.Code != 403 || len(f.canceled) != 1 {
		t.Errorf("cancel from a foreign origin: %d", w.Code)
	}
	f.err = refusal{"it is marked keep"}
	if w := invPost(post, "/api/inventory/trash", `{"id":1,"confirm":true}`, auth); w.Code != 409 || !strings.Contains(w.Body.String(), "marked keep") {
		t.Errorf("refusal: %d %s", w.Code, w.Body)
	}
	if !containsAll(InvPostRoutes, "/api/inventory/mark", "/api/inventory/equip", "/api/inventory/unpin") {
		t.Error("a v1 route was removed from InvPostRoutes")
	}
}

func containsAll(list []string, want ...string) bool {
	for _, w := range want {
		found := false
		for _, x := range list {
			found = found || x == w
		}
		if !found {
			return false
		}
	}
	return true
}
