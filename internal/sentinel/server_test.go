package sentinel

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"d2lod-bot/internal/status"
)

func get(t *testing.T, h http.Handler, path string, hdr ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Host = "127.0.0.1:8895"
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestServerRoutes(t *testing.T) {
	r, _, _ := newRunner(Mode{})
	srv, h, err := r.NewServer(status.NewHub(), "127.0.0.1:8895", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	v1 := get(t, h, "/api/v1/sentinel")
	if v1.Code != 200 || !strings.Contains(v1.Body.String(), `"summary"`) {
		t.Fatalf("v1: %d %s", v1.Code, v1.Body)
	}
	if v1.Header().Get("Deprecation") != "" {
		t.Error("v1 must not be marked deprecated")
	}
	old := get(t, h, "/api/sentinel")
	if old.Code != 200 || old.Header().Get("Deprecation") != "true" || !strings.Contains(old.Header().Get("Link"), "/api/v1/sentinel") {
		t.Fatalf("alias: %d %v", old.Code, old.Header())
	}
	if get(t, h, "/api/sentinelx").Code != 404 {
		t.Error("alias must match the exact path only")
	}
	if c := get(t, h, "/live/").Code; c != 200 {
		t.Errorf("/live/ = %d", c)
	}
	if c := get(t, h, "/healthz").Code; c != 200 {
		t.Errorf("/healthz = %d", c)
	}
}

func TestServerToken(t *testing.T) {
	r, _, _ := newRunner(Mode{})
	if _, _, err := r.NewServer(status.NewHub(), "0.0.0.0:8895", "", nil); err == nil {
		t.Fatal("non-loopback bind without a token must be refused")
	}
	srv, h, err := r.NewServer(status.NewHub(), "0.0.0.0:8895", "s3cret", nil)
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	host := func(path string, hdr ...string) int {
		req := httptest.NewRequest("GET", path, nil)
		req.Host = "worker.example:8895"
		for i := 0; i+1 < len(hdr); i += 2 {
			req.Header.Set(hdr[i], hdr[i+1])
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		return w.Code
	}
	if c := host("/api/v1/sentinel"); c != 401 {
		t.Errorf("no token = %d, want 401", c)
	}
	if c := host("/api/sentinel"); c != 401 {
		t.Errorf("alias without token = %d, want 401", c)
	}
	if c := host("/api/v1/sentinel", "Authorization", "Bearer s3cret"); c != 200 {
		t.Errorf("bearer = %d", c)
	}
	if c := host("/api/sentinel", "X-Token", "s3cret"); c != 200 {
		t.Errorf("alias with token = %d", c)
	}
}
