package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"d2lod-bot/internal/watch"
)

func watchServer(t *testing.T, token string) (*server, *watch.Sim) {
	t.Helper()
	sim, err := watch.NewSim("paladin", 1)
	if err != nil {
		t.Fatal(err)
	}
	return &server{token: token, watch: sim.Hub.ViewHandler()}, sim
}

func do(h http.Handler, method, path, auth string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
	if auth != "" {
		req.Header.Set("Authorization", "Bearer "+auth)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestWatchIsOffByDefault(t *testing.T) {
	s := &server{} // no -watch
	h := s.routes()
	for _, p := range watch.ViewRoutes {
		if w := do(h, "GET", p, ""); w.Code != 404 && !(p == "/watch" || p == "/watch/") {
			t.Errorf("GET %s without -watch: %d", p, w.Code)
		}
	}
	if w := do(h, "GET", "/api/thoughts", ""); w.Code != 404 {
		t.Errorf("/api/thoughts without -watch: %d", w.Code)
	}
}

func TestWatchRoutesOnTheViewerAreReadOnly(t *testing.T) {
	s, sim := watchServer(t, "")
	for name, h := range map[string]http.Handler{"with a game": s.routes(), "watch only": s.watchOnlyRoutes()} {
		if w := do(h, "GET", "/api/thoughts", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "watch.thoughts/1") {
			t.Errorf("%s: GET /api/thoughts: %d", name, w.Code)
		}
		if w := do(h, "GET", "/watch", ""); w.Code != 200 || !strings.Contains(w.Body.String(), "/watch/app.js") {
			t.Errorf("%s: GET /watch: %d", name, w.Code)
		}
		for _, p := range append(append([]string{}, watch.ViewRoutes...), "/api/answer", "/api/confirm", "/api/revoke", "/api/revoke-all") {
			for _, m := range []string{"POST", "PUT", "DELETE", "PATCH"} {
				w := do(h, m, p, "")
				// /api/answer and the other write names do not exist on the viewer at all; the watch routes say 405.
				if w.Code == 200 || w.Code == 204 {
					t.Errorf("%s: %s %s accepted: %d", name, m, p, w.Code)
				}
				if strings.HasPrefix(p, "/api/") && p != "/api/answer" && p != "/api/confirm" && p != "/api/revoke" && p != "/api/revoke-all" &&
					(p == "/api/thoughts" || p == "/api/journal" || p == "/api/rules" || p == "/api/uptime" || p == "/api/orders") && w.Code != 405 {
					t.Errorf("%s: %s %s: %d, want 405", name, m, p, w.Code)
				}
			}
		}
	}
	if len(sim.Store.Log()) != 0 || len(sim.Store.OpenThoughts()) != 0 {
		t.Error("a viewer request reached the store")
	}
}

func TestWatchFollowsTheViewerToken(t *testing.T) {
	s, _ := watchServer(t, "tok")
	h := s.routes()
	if w := do(h, "GET", "/api/thoughts", ""); w.Code != 401 {
		t.Errorf("no token: %d", w.Code)
	}
	if w := do(h, "GET", "/api/thoughts", "tok"); w.Code != 200 {
		t.Errorf("token: %d", w.Code)
	}
}

func TestStartWatchServesAnswersOnItsOwnPort(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	h, err := startWatch(ctx, watchOpts{on: true, answerAddr: "127.0.0.1:0", class: "sorceress", speed: 1}, "127.0.0.1:8891", "", false, false)
	if err != nil {
		t.Fatal(err)
	}
	w := do(h, "GET", "/api/config", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"answerPort":`) {
		t.Errorf("config must point the page at the answer port: %d %s", w.Code, w.Body.String())
	}
	if got := watchOrigins("", "127.0.0.1:8891"); len(got) != 2 || got[0] != "http://localhost:8891" {
		t.Errorf("default origins: %v", got)
	}
	if got := watchOrigins(" http://a:1 , http://b:2 ", "x:1"); len(got) != 2 || got[1] != "http://b:2" {
		t.Errorf("origin flag: %v", got)
	}
	if _, err := startWatch(ctx, watchOpts{on: true, answerAddr: "0.0.0.0:0", class: "sorceress", speed: 1}, ":8891", "", false, false); err == nil {
		t.Error("a non-loopback answer route without a token must be refused")
	}
	if _, err := startWatch(ctx, watchOpts{on: true, answerAddr: "127.0.0.1:0", class: "monk", speed: 1}, ":8891", "", false, false); err == nil {
		t.Error("an unknown class must be an error, not a fallback to the Sorceress")
	}
}
