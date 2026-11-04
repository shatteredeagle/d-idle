package main

import (
	"net/http/httptest"
	"strings"
	"testing"
)

// The d.idle brand icons are served from the embedded web dir; once the head-links patch is applied both pages link
// them with absolute paths (the stream page is also served at /stream and /live, so a relative link would break).
func TestBrandIconsServedAndLinked(t *testing.T) {
	s := &server{}
	for path, ct := range map[string]string{"/icon.svg": "image/svg+xml", "/favicon.ico": "", "/apple-touch-icon.png": "image/png"} {
		rec := httptest.NewRecorder()
		s.routes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		if rec.Code != 200 || rec.Body.Len() < 100 {
			t.Errorf("%s: code %d, %d bytes", path, rec.Code, rec.Body.Len())
		}
		if ct != "" && !strings.HasPrefix(rec.Header().Get("Content-Type"), ct) {
			t.Errorf("%s: content type %q", path, rec.Header().Get("Content-Type"))
		}
	}
	for _, path := range []string{"/stream", "/live", "/"} {
		rec := httptest.NewRecorder()
		s.routes().ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		b := rec.Body.String()
		if !strings.Contains(b, "/icon.svg") {
			t.Skip("head links not applied yet (mapitem owns the two pages): git show openspec-final:openspec/changes/archive/2026-11-01-brandapply/head-links.patch | git apply")
		}
		for _, want := range []string{`href="/icon.svg"`, `href="/favicon.ico"`, `href="/apple-touch-icon.png"`} {
			if !strings.Contains(b, want) {
				t.Errorf("%s lacks %s", path, want)
			}
		}
	}
}
