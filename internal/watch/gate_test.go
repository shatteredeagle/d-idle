package watch

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/access"
)

func init() { access.Iterations = 1000 }

func newAccess(t *testing.T, rec *access.Record) *Access {
	t.Helper()
	st := access.NewStore(t.TempDir() + "/access.json")
	if rec != nil {
		if err := st.Save(*rec); err != nil {
			t.Fatal(err)
		}
	}
	return &Access{Store: st, Limiter: access.NewLimiter()}
}

func passRec(t *testing.T, p string) access.Record {
	r, err := access.NewPasscode(p, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

var ok204 = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })

func do(h http.Handler, method, target, body string, hdr map[string]string, cookies ...*http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.7:5555"
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	for _, c := range cookies {
		req.AddCookie(c)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func form(p string) string { return url.Values{"passcode": {p}}.Encode() }

func TestViewGatePasscodeLogin(t *testing.T) {
	rec := passRec(t, "four plain words")
	a := newAccess(t, &rec)
	h := a.viewGate("tokentokentokentoken", ok204)

	if w := do(h, "GET", "/watch", "", nil); w.Code != 303 || w.Header().Get("Location") != "/login" {
		t.Errorf("anonymous page: %d %s", w.Code, w.Header().Get("Location"))
	}
	for _, p := range []string{"/api/thoughts", "/stream"} {
		if w := do(h, "GET", p, "", nil); w.Code != 401 {
			t.Errorf("anonymous %s: %d", p, w.Code)
		}
	}
	// the token does not open a passcode install, as bearer or as ?t=
	if w := do(h, "GET", "/watch?t=tokentokentokentoken", "", nil); w.Code == 204 {
		t.Error("?t= token accepted in passcode mode")
	}
	if w := do(h, "GET", "/api/rules", "", map[string]string{"Authorization": "Bearer tokentokentokentoken"}); w.Code != 401 {
		t.Errorf("bearer token: %d", w.Code)
	}
	if w := do(h, "GET", "/login", "", nil); w.Code != 200 || !strings.Contains(w.Body.String(), `type="password"`) || w.Header().Get("Content-Security-Policy") == "" {
		t.Errorf("login page: %d", w.Code)
	}
	w := do(h, "POST", "/login", form("four plain words"), nil)
	if w.Code != 303 || w.Header().Get("Location") != "/watch" {
		t.Fatalf("good login: %d", w.Code)
	}
	cs := w.Result().Cookies()
	if len(cs) != 1 {
		t.Fatalf("cookies %v", cs)
	}
	c := cs[0]
	if c.Name != access.SessionCookie || !c.HttpOnly || c.SameSite != http.SameSiteStrictMode || c.Path != "/" || c.MaxAge <= 0 || c.Secure {
		t.Errorf("cookie flags: %+v", c)
	}
	if strings.Contains(c.Value, "four") || c.Value == rec.Hash {
		t.Error("cookie leaks the passcode or the hash")
	}
	if w := do(h, "GET", "/api/thoughts", "", nil, c); w.Code != 204 {
		t.Errorf("with cookie: %d", w.Code)
	}
	if w := do(h, "GET", "/api/thoughts", "", nil, &http.Cookie{Name: access.SessionCookie, Value: "x"}); w.Code != 401 {
		t.Errorf("forged cookie: %d", w.Code)
	}
	// a new passcode (reset) invalidates the old cookie without a restart
	if err := a.Store.Save(passRec(t, "another phrase")); err != nil {
		t.Fatal(err)
	}
	if w := do(h, "GET", "/api/thoughts", "", nil, c); w.Code != 401 {
		t.Errorf("old cookie after reset: %d", w.Code)
	}
}

func TestLoginLockoutAndNoLeak(t *testing.T) {
	rec := passRec(t, "four plain words")
	a := newAccess(t, &rec)
	h := a.viewGate("", ok204)
	for i := 0; i < 5; i++ {
		w := do(h, "POST", "/login", form("wrong-"+string(rune('a'+i))+"zzz"), nil)
		if w.Code != 401 || strings.Contains(w.Body.String(), "wrong-") || w.Result().Cookies() != nil && len(w.Result().Cookies()) != 0 {
			t.Fatalf("attempt %d: %d", i, w.Code)
		}
	}
	// locked: even the right passcode is refused, with Retry-After
	w := do(h, "POST", "/login", form("four plain words"), nil)
	if w.Code != 429 || w.Header().Get("Retry-After") == "" || len(w.Result().Cookies()) != 0 {
		t.Errorf("locked client: %d", w.Code)
	}
	// another client is not affected
	req := httptest.NewRequest("POST", "/login", strings.NewReader(form("four plain words")))
	req.RemoteAddr = "192.0.2.99:1"
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w2 := httptest.NewRecorder()
	h.ServeHTTP(w2, req)
	if w2.Code != 303 {
		t.Errorf("other client: %d", w2.Code)
	}
	// cross-origin login post is refused
	if w := do(h, "POST", "/login", form("four plain words"), map[string]string{"Origin": "http://evil.example"}); w.Code != 403 {
		t.Errorf("cross-origin login: %d", w.Code)
	}
}

func TestViewGateTokenKindAndNoFile(t *testing.T) {
	for name, rec := range map[string]*access.Record{"no file": nil, "token kind": func() *access.Record { r := access.TokenRecord(time.Now()); return &r }()} {
		h := newAccess(t, rec).viewGate("tok", ok204)
		for _, c := range []struct {
			target, auth string
			want         int
		}{{"/x", "", 401}, {"/x?t=bad", "", 401}, {"/x?t=tok", "", 204}, {"/x", "Bearer tok", 204}} {
			hdr := map[string]string{}
			if c.auth != "" {
				hdr["Authorization"] = c.auth
			}
			if w := do(h, "GET", c.target, "", hdr); w.Code != c.want {
				t.Errorf("%s %s %q: %d", name, c.target, c.auth, w.Code)
			}
		}
		if w := do(h, "GET", "/x?t=tok", "", nil); !strings.Contains(w.Header().Get("Set-Cookie"), "d2dview=tok") {
			t.Errorf("%s: token cookie not set", name)
		}
	}
	// no token and nothing chosen: closed, never open
	if w := do(newAccess(t, nil).viewGate("", ok204), "GET", "/watch", "", nil); w.Code != 401 {
		t.Errorf("unset access with no token: %d", w.Code)
	}
}

func TestAnswerGate(t *testing.T) {
	rec := passRec(t, "four plain words")
	a := newAccess(t, &rec)
	h := a.answerGate("tok", ok204)
	if w := do(h, "POST", "/api/answer", "{}", nil); w.Code != 401 || !strings.Contains(w.Body.String(), "unauthorized") {
		t.Errorf("no cookie: %d", w.Code)
	}
	c := rec.Cookie()
	if w := do(h, "POST", "/api/answer", "{}", nil, c); w.Code != 204 {
		t.Errorf("cookie: %d", w.Code)
	}
	if w := do(h, "POST", "/api/answer", "{}", map[string]string{"Authorization": "Bearer tok"}); w.Code != 401 {
		t.Errorf("token in passcode mode: %d", w.Code)
	}
	if w := do(h, "OPTIONS", "/api/answer", "", nil); w.Code != 204 {
		t.Errorf("preflight must reach the origin check: %d", w.Code)
	}
	// token kind
	st := access.NewStore(t.TempDir() + "/a.json")
	st.Save(access.TokenRecord(time.Now()))
	ht := (&Access{Store: st, Limiter: access.NewLimiter()}).answerGate("tok", ok204)
	if w := do(ht, "POST", "/api/answer", "{}", map[string]string{"Authorization": "Bearer tok"}); w.Code != 204 {
		t.Errorf("token kind bearer: %d", w.Code)
	}
	if w := do(ht, "POST", "/api/answer", "{}", nil, &http.Cookie{Name: "d2dview", Value: "tok"}); w.Code != 204 {
		t.Errorf("token kind cookie: %d", w.Code)
	}
	if w := do(ht, "POST", "/api/answer", "{}", nil); w.Code != 401 {
		t.Errorf("token kind anonymous: %d", w.Code)
	}
}

func TestLogout(t *testing.T) {
	rec := passRec(t, "four plain words")
	h := newAccess(t, &rec).viewGate("", ok204)
	w := do(h, "GET", "/logout", "", nil)
	if w.Code != 303 || len(w.Result().Cookies()) != 1 || w.Result().Cookies()[0].MaxAge >= 0 {
		t.Errorf("logout: %d %v", w.Code, w.Result().Cookies())
	}
}
