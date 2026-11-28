package watch

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServeRefusals(t *testing.T) {
	r := newRig(t, "sorceress")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, c := range []struct {
		o   ServeOptions
		bad string
	}{
		{ServeOptions{ViewAddr: "127.0.0.1:8991"}, "both required"},
		{ServeOptions{ViewAddr: "127.0.0.1:8991", AnswerAddr: "127.0.0.1:8991"}, "own address"},
		{ServeOptions{ViewAddr: "0.0.0.0:0", AnswerAddr: "127.0.0.1:0"}, "token"},
		{ServeOptions{ViewAddr: "127.0.0.1:0", AnswerAddr: "0.0.0.0:0"}, "token"},
	} {
		if _, err := r.sim.Hub.Serve(ctx, c.o); err == nil || !strings.Contains(err.Error(), c.bad) {
			t.Errorf("%+v: %v", c.o, err)
		}
	}
}

func TestOriginsAndLoopback(t *testing.T) {
	if got := Origins("", "127.0.0.1:8891"); len(got) != 2 || got[0] != "http://localhost:8891" {
		t.Errorf("default origins: %v", got)
	}
	if got := Origins(" http://a:1 , http://b:2 ", "x:1"); len(got) != 2 || got[1] != "http://b:2" {
		t.Errorf("listed origins: %v", got)
	}
	for addr, want := range map[string]bool{"127.0.0.1:1": true, "localhost:1": true, "[::1]:1": true, "0.0.0.0:1": false, ":1": false, "192.0.2.1:1": false} {
		if Loopback(addr) != want {
			t.Errorf("Loopback(%s) != %v", addr, want)
		}
	}
}

// A page load with ?t= ends on the clean address with the cookie set; a wrong token sets no cookie.
func TestTokenGateStripsTokenFromAddress(t *testing.T) {
	h := tokenGate("tok", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	do := func(target, accept, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest("GET", target, nil)
		if accept != "" {
			req.Header.Set("Accept", accept)
		}
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: "d2dview", Value: cookie})
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w
	}
	w := do("/watch?t=tok&layout=cockpit", "text/html,*/*", "")
	if w.Code != http.StatusSeeOther || w.Header().Get("Location") != "/watch?layout=cockpit" {
		t.Fatalf("page load: %d to %q", w.Code, w.Header().Get("Location"))
	}
	if c := w.Result().Cookies(); len(c) != 1 || c[0].Value != "tok" || !c[0].HttpOnly {
		t.Fatalf("cookie after a valid token: %v", c)
	}
	if w = do("/watch?t=tok", "text/html", ""); w.Header().Get("Location") != "/watch" {
		t.Errorf("clean path: %q", w.Header().Get("Location"))
	}
	if w = do("/watch?t=bad", "text/html", ""); w.Code != 401 || len(w.Result().Cookies()) != 0 {
		t.Errorf("wrong token: %d cookies %v", w.Code, w.Result().Cookies())
	}
	if w = do("/watch?t=bad", "text/html", "tok"); w.Code != 401 {
		t.Errorf("a wrong ?t= must not fall back to the cookie: %d", w.Code)
	}
	if w = do("/api/status?t=tok", "application/json", ""); w.Code != 204 {
		t.Errorf("an API read with ?t= is served, not redirected: %d", w.Code)
	}
	w = do("/watch", "text/html", "tok")
	if w.Code != 204 || w.Header().Get("Referrer-Policy") != "no-referrer" || w.Header().Get("Cache-Control") != "no-store" {
		t.Errorf("cookie page: %d headers %v", w.Code, w.Header())
	}
}

func TestTokenGate(t *testing.T) {
	h := tokenGate("tok", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) }))
	for _, c := range []struct {
		target, auth string
		want         int
	}{{"/x", "", 401}, {"/x?t=bad", "", 401}, {"/x?t=tok", "", 204}, {"/x", "Bearer tok", 204}} {
		req := httptest.NewRequest("GET", c.target, nil)
		if c.auth != "" {
			req.Header.Set("Authorization", c.auth)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != c.want {
			t.Errorf("%s %q: %d", c.target, c.auth, w.Code)
		}
	}
}

// Source guard: the shell reads with the cookie and never builds a ?t= address itself.
func TestShellDoesNotCopyTokenIntoFetch(t *testing.T) {
	sh, err := webFS.ReadFile("web/shell.js")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(sh), "'?t='") {
		t.Error("shell.js builds a ?t= address")
	}
}
