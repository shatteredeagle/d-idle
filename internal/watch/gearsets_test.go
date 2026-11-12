package watch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"regexp"
	"strings"
	"testing"
)

// Gear sets and drag and drop (docs/gearsets.md): the node tests of the two modules, the schema 7 routes with their
// gates, and patterns that keep the new files inside the page rules.

func TestDragModuleBehaviour(t *testing.T) {
	runNode(t, "testdata/gearsets/drag_dom.js", "web/inventory-drag.js")
}

func TestSetsPanelDOMGoldens(t *testing.T) {
	runNode(t, "testdata/gearsets/sets_dom.js", "web/inventory-sets.js")
}

type fakeGear struct {
	*fakeInv
	log []string
}

func (f *fakeGear) rec(s string) { f.log = append(f.log, s) }
func (f *fakeGear) RequestUnequip(slot, target string) (any, error) {
	f.rec("unequip " + slot + "/" + target)
	return map[string]any{"kind": "unequip"}, f.err
}
func (f *fakeGear) RequestEquipFor(id uint32, slot, target string) (any, error) {
	f.rec("equipfor " + slot + "/" + target)
	return nil, f.err
}
func (f *fakeGear) SaveSet(name, target string) (any, error) {
	f.rec("save " + name + "/" + target)
	return map[string]any{"id": "gs-1"}, f.err
}
func (f *fakeGear) RenameSet(id, name string) error { f.rec("rename " + id + "/" + name); return f.err }
func (f *fakeGear) DeleteSet(id string) error       { f.rec("delete " + id); return f.err }
func (f *fakeGear) PreviewSet(id string) (any, error) {
	f.rec("preview " + id)
	return map[string]any{"setId": id, "ready": true}, f.err
}
func (f *fakeGear) ApplySet(id string) (any, error) { f.rec("apply " + id); return nil, f.err }
func (f *fakeGear) CancelApply(g string) error      { f.rec("cancel " + g); return f.err }

func TestGearSetRoutes(t *testing.T) {
	api := NewInvAPI()
	f := &fakeGear{fakeInv: &fakeInv{}}
	api.Bind(f)
	post := api.PostHandler(AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}})
	auth := map[string]string{"Authorization": "Bearer tok"}
	for _, c := range []struct{ path, body, want string }{
		{"/api/inventory/unequip", `{"slot":"rhand"}`, "unequip rhand/"},
		{"/api/inventory/unequip", `{"slot":"mhead","target":"merc"}`, "unequip mhead/merc"},
		{"/api/inventory/equip", `{"id":4,"slot":"mhead","target":"merc"}`, "equipfor mhead/merc"},
		{"/api/inventory/sets/save", `{"name":"Fire","target":"merc"}`, "save Fire/merc"},
		{"/api/inventory/sets/rename", `{"setId":"gs-1","name":"Ice"}`, "rename gs-1/Ice"},
		{"/api/inventory/sets/delete", `{"setId":"gs-1"}`, "delete gs-1"},
		{"/api/inventory/sets/apply", `{"setId":"gs-1"}`, "apply gs-1"},
		{"/api/inventory/sets/cancel", `{"group":"a1"}`, "cancel a1"},
	} {
		f.log = nil
		if w := invPost(post, c.path, c.body, auth); w.Code != 200 || len(f.log) != 1 || f.log[0] != c.want {
			t.Errorf("%s %s: %d %s log %v, want %q", c.path, c.body, w.Code, w.Body, f.log, c.want)
		}
		if w := invPost(post, c.path, c.body, nil); w.Code != 401 {
			t.Errorf("%s without a token: %d", c.path, w.Code)
		}
		if w := invPost(post, c.path, c.body, map[string]string{"Authorization": "Bearer tok", "Origin": "http://evil.test"}); w.Code != 403 {
			t.Errorf("%s from a foreign origin: %d", c.path, w.Code)
		}
	}
	// the merc is a target only where the desk answers for him; every other route still refuses it
	if w := invPost(post, "/api/inventory/trash", `{"id":1,"confirm":true,"target":"merc"}`, auth); w.Code != 409 || !strings.Contains(w.Body.String(), `"target"`) {
		t.Errorf("trash for the merc: %d %s", w.Code, w.Body)
	}
	if w := invPost(post, "/api/inventory/sets/apply", `{"setId":"gs-1","target":"merc"}`, auth); w.Code != 409 {
		t.Errorf("apply with a target: %d", w.Code)
	}
	f.err = refusal{"the bot cannot change the mercenary's gear yet"}
	w := invPost(post, "/api/inventory/equip", `{"id":4,"slot":"mhead","target":"merc"}`, auth)
	if w.Code != 409 || !strings.Contains(w.Body.String(), "mercenary") {
		t.Errorf("merc refusal: %d %s", w.Code, w.Body)
	}
	f.err = errors.New("disk full")
	if w := invPost(post, "/api/inventory/sets/save", `{"name":"x"}`, auth); w.Code != 500 || strings.Contains(w.Body.String(), "disk") {
		t.Errorf("internal error leaks: %d %s", w.Code, w.Body)
	}
	// a desk without the schema 7 part answers with code off, and an old client's equip still works
	api2 := NewInvAPI()
	api2.Bind(&fakeInv{})
	post2 := api2.PostHandler(AnswerOptions{Token: "tok", Origins: []string{"http://ok.test"}})
	if w := invPost(post2, "/api/inventory/unequip", `{"slot":"rhand"}`, auth); w.Code != 409 || !strings.Contains(w.Body.String(), `"off"`) {
		t.Errorf("no gear part: %d %s", w.Code, w.Body)
	}
	if w := invPost(post2, "/api/inventory/equip", `{"id":1,"slot":"rhand"}`, auth); w.Code != 200 {
		t.Errorf("old equip: %d", w.Code)
	}
	if w := invPost(post2, "/api/inventory/equip", `{"id":1,"slot":"mhead","target":"merc"}`, auth); w.Code != 409 {
		t.Errorf("merc target on an old desk: %d", w.Code)
	}
}

func TestSetPreviewRoute(t *testing.T) {
	api := NewInvAPI()
	f := &fakeGear{fakeInv: &fakeInv{}}
	api.Bind(f)
	get := func(a *InvAPI) *httptest.ResponseRecorder {
		w := httptest.NewRecorder()
		a.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/inventory/setpreview?id=gs-1", nil))
		return w
	}
	if w := get(api); w.Code != 200 || !strings.Contains(w.Body.String(), `"setId":"gs-1"`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("preview: %d %s", w.Code, w.Body)
	}
	f.err = refusal{"that set does not exist any more"}
	if w := get(api); w.Code != 409 {
		t.Errorf("refusal: %d", w.Code)
	}
	old := NewInvAPI()
	old.Bind(&fakeInv{})
	if w := get(old); w.Code != 409 || !strings.Contains(w.Body.String(), `"off"`) {
		t.Errorf("old desk: %d %s", w.Code, w.Body)
	}
}

func TestGearSetsPageAndFiles(t *testing.T) {
	for _, want := range []string{`href="/watch/inventory-sets.css"`, `src="/watch/inventory-drag.js"`, `src="/watch/inventory-sets.js"`, `id="gearsets"`} {
		if !strings.Contains(invPage, want) {
			t.Errorf("inventory page lacks %s", want)
		}
	}
	h := NewInvAPI().Handler()
	for path, ctype := range map[string]string{"/watch/inventory-drag.js": "text/javascript", "/watch/inventory-sets.js": "text/javascript", "/watch/inventory-sets.css": "text/css"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if w.Code != 200 || !strings.HasPrefix(w.Header().Get("Content-Type"), ctype) || w.Body.Len() < 200 {
			t.Errorf("%s: %d %q", path, w.Code, w.Header().Get("Content-Type"))
		}
	}
	bad := regexp.MustCompile(`scrollIntoView|scrollTo\(|scrollBy\(|\.scrollTop\s*=|innerHTML|outerHTML|insertAdjacentHTML|createElement\(\s*['"]style|cssText|setInterval|(?i)\bpin(ned)?\b|unpin`)
	for _, f := range []string{"inventory-drag.js", "inventory-sets.js"} {
		b, err := webFS.ReadFile("web/" + f)
		if err != nil {
			t.Fatal(err)
		}
		if m := bad.FindString(string(b)); m != "" {
			t.Errorf("web/%s matches %q", f, m)
		}
		for _, m := range regexp.MustCompile(`\.focus\([^)]*\)`).FindAllString(string(b), -1) {
			if !strings.Contains(m, "preventScroll: true") {
				t.Errorf("web/%s: focus call without preventScroll: %s", f, m)
			}
		}
	}
	css, _ := webFS.ReadFile("web/inventory-sets.css")
	if !strings.Contains(string(css), "@layer components") || regexp.MustCompile(`@media|!important|#[0-9a-fA-F]{3,6}\b`).Match(css) {
		t.Error("inventory-sets.css: needs its layer, no media query, !important or colour literal")
	}
	// the bot never reaches the sets: no source file outside the desk, the routes and the page mentions the apply route
	for _, dir := range []string{"../bot", "../town", "../leveler", "../loot"} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") || strings.HasSuffix(e.Name(), "_test.go") {
				continue
			}
			b, _ := os.ReadFile(dir + "/" + e.Name())
			if strings.Contains(string(b), "ApplySet") || strings.Contains(string(b), "SaveSet") {
				t.Errorf("%s/%s reaches the gear sets: only the player's click may", dir, e.Name())
			}
		}
	}
}
