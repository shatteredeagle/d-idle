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
