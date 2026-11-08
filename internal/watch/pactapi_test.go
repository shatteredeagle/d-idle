package watch

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeDesk struct{ calls []string }

func (f *fakeDesk) Offer() any    { return map[string]string{"x": "offer"} }
func (f *fakeDesk) Hardcore() any { return map[string]bool{"active": false} }
func (f *fakeDesk) Consent(p string, r, h int, t string) error {
	f.calls = append(f.calls, "consent "+p+" "+t)
	if t != "ok" {
		return errors.New("type the name")
	}
	return nil
}
func (f *fakeDesk) Revoke(p string) error { f.calls = append(f.calls, "revoke "+p); return nil }
func (f *fakeDesk) HardcoreConfirm(t string) error {
	f.calls = append(f.calls, "hc "+t)
	return nil
}
func (f *fakeDesk) HardcoreRevoke() error   { return nil }
func (f *fakeDesk) HardcoreActivate() error { return ErrPactInactive }

func pactPost(h http.Handler, path, tok, origin, body string) *httptest.ResponseRecorder {
	r := httptest.NewRequest("POST", path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	if tok != "" {
		r.Header.Set("Authorization", "Bearer "+tok)
	}
	if origin != "" {
		r.Header.Set("Origin", origin)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

func TestPactRoutesGatesAndErrors(t *testing.T) {
	a := NewPactAPI()
	o := AnswerOptions{Token: "tok", Origins: []string{"http://ok"}}
	post := a.PostHandler(o)
	// no desk yet
	if w := pactPost(post, "/api/pacts/consent", "tok", "", `{"pact":"p"}`); w.Code != 503 {
		t.Fatalf("no desk: %d", w.Code)
	}
	if w := httptest.NewRecorder(); true {
		a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/pacts/offer", nil))
		if w.Code != 503 {
			t.Fatalf("offer without desk: %d", w.Code)
		}
	}
	d := &fakeDesk{}
	a.Bind(d)
	for _, c := range []struct {
		name, tok, origin, body string
		want                    int
	}{
		{"no token", "", "", `{"pact":"p"}`, 401},
		{"bad token", "x", "", `{"pact":"p"}`, 401},
		{"bad origin", "tok", "http://evil", `{"pact":"p"}`, 403},
		{"unknown field", "tok", "", `{"nope":1}`, 400},
		{"typed wrong", "tok", "http://ok", `{"pact":"p","typed":"no"}`, 400},
		{"typed right", "tok", "http://ok", `{"pact":"p","rank":1,"awayHours":4,"typed":"ok"}`, 200},
	} {
		if w := pactPost(post, "/api/pacts/consent", c.tok, c.origin, c.body); w.Code != c.want {
			t.Errorf("%s: %d %s", c.name, w.Code, w.Body)
		}
	}
	if len(d.calls) != 2 { // the two that reached the desk
		t.Fatalf("calls %v", d.calls)
	}
	// Hardcore: own action, activation refused with 501, unknown action 400
	if w := pactPost(post, "/api/pacts/hardcore", "tok", "", `{"action":"activate"}`); w.Code != 501 || !strings.Contains(w.Body.String(), "not active") {
		t.Errorf("activate: %d %s", w.Code, w.Body)
	}
	if w := pactPost(post, "/api/pacts/hardcore", "tok", "", `{"action":"confirm","typed":"Hero"}`); w.Code != 200 {
		t.Errorf("confirm: %d", w.Code)
	}
	if w := pactPost(post, "/api/pacts/hardcore", "tok", "", `{"action":"x"}`); w.Code != 400 {
		t.Errorf("action: %d", w.Code)
	}
	// the consent route does not confirm hardcore and the reverse
	d.calls = nil
	pactPost(post, "/api/pacts/hardcore", "tok", "", `{"action":"confirm","typed":"ok","pact":"p"}`)
	if len(d.calls) != 1 || !strings.HasPrefix(d.calls[0], "hc ") {
		t.Errorf("hardcore reached the consent path: %v", d.calls)
	}
	if w := httptest.NewRecorder(); true {
		a.Handler().ServeHTTP(w, httptest.NewRequest("GET", "/api/pacts/offer", nil))
		if w.Code != 200 || !strings.Contains(w.Body.String(), "offer") {
			t.Errorf("offer: %d %s", w.Code, w.Body)
		}
	}
	if w := httptest.NewRecorder(); true {
		post.ServeHTTP(w, httptest.NewRequest("GET", "/api/pacts/consent", nil))
		if w.Code != 405 {
			t.Errorf("GET on a post route: %d", w.Code)
		}
	}
}
