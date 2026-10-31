package main

import (
	"net/http/httptest"
	"testing"
	"time"

	"d2lod-bot/internal/server"
)

func TestInstancesServeRoutes(t *testing.T) {
	reg := &server.StaticRegistry{}
	for _, id := range []string{"i1", "i2"} {
		_, m := newInstance(1, "", time.Minute, false)
		if len(m.Goals()) == 0 {
			t.Fatal("seeded instance has no goals")
		}
		if err := reg.Add(&server.Instance{ID: id, Manager: m}); err != nil {
			t.Fatal(err)
		}
	}
	srv, err := server.New(server.Config{Instances: reg})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	for _, p := range []string{"/api/v1/instances", "/api/v1/instances/i2/status", "/api/v1/instances/i1/goals", "/api/v1/fleet"} {
		if c := serveGet(srv, p); c != 200 {
			t.Errorf("%s = %d", p, c)
		}
	}
	if c := serveGet(srv, "/api/v1/instances/i3/status"); c != 404 {
		t.Errorf("unknown instance = %d", c)
	}
}

func serveGet(s *server.Server, path string) int {
	req := httptest.NewRequest("GET", path, nil)
	req.Host = "127.0.0.1:8420"
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	return w.Code
}
