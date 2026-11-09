package main

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNormalizeAddress(t *testing.T) {
	ok := map[string]string{
		"game.example:8891":              "http://game.example:8891",
		"  GAME.example:8891/watch?x=1 ": "http://game.example:8891",
		"https://Game.example":           "https://game.example",
		"http://game.example:80":         "http://game.example:80", // the port is never added or dropped
		"[::1]:8891":                     "http://[::1]:8891",
	}
	for in, want := range ok {
		if got, err := NormalizeAddress(in); err != nil || got != want {
			t.Errorf("%q: got %q, %v want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"", "  ", "ftp://x:1", "http://user:pass@host:1", "host:99999", "host:abc", "http://"} {
		if got, err := NormalizeAddress(in); err == nil {
			t.Errorf("%q accepted as %q", in, got)
		}
	}
}

func TestPageURL(t *testing.T) {
	for page, path := range map[string]string{"watch": "/watch", "inventory": "/watch/inventory", "policy": "/watch/policy", "return": "/watch/return"} {
		if u, err := PageURL("http://h:1", page); err != nil || u != "http://h:1"+path {
			t.Errorf("%s: %q %v", page, u, err)
		}
	}
	if _, err := PageURL("http://h:1", "nope"); err == nil {
		t.Error("unknown page accepted")
	}
}

func TestConfigRoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cfg")
	if c, err := LoadConfig(dir); err != nil || c.Address != "" {
		t.Fatalf("first run: %+v %v", c, err)
	}
	if err := SaveConfig(dir, Config{Address: "http://h:1"}); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(dir)
	if err != nil || c.Address != "http://h:1" {
		t.Fatalf("%+v %v", c, err)
	}
	b, _ := os.ReadFile(filepath.Join(dir, "config.json"))
	var raw map[string]any
	json.Unmarshal(b, &raw)
	if len(raw) != 1 {
		t.Errorf("config holds more than the address: %s", b)
	}
	os.WriteFile(filepath.Join(dir, "config.json"), []byte("{broken"), 0o600)
	if _, err := LoadConfig(dir); err == nil {
		t.Error("corrupt config not reported")
	}
}

func TestProbe(t *testing.T) {
	var code int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if code == 302 {
			http.Redirect(w, r, "/login", 302)
			return
		}
		w.WriteHeader(code)
	}))
	defer srv.Close()
	c := NewProbeClient()
	for _, tc := range []struct {
		code int
		want State
	}{{200, Up}, {302, Up}, {403, Up}, {502, Warming}, {503, Warming}} {
		code = tc.code
		if got := Probe(context.Background(), c, srv.URL); got != tc.want {
			t.Errorf("%d: %s want %s", tc.code, got, tc.want)
		}
	}
	srv.Close()
	if got := Probe(context.Background(), c, srv.URL); got != Down {
		t.Errorf("closed server: %s", got)
	}
}

func TestFindBrowser(t *testing.T) {
	has := func(names ...string) func(string) (string, error) {
		return func(n string) (string, error) {
			for _, h := range names {
				if h == n {
					return "/usr/bin/" + n, nil
				}
			}
			return "", errors.New("no")
		}
	}
	never := func(string) bool { return false }
	if got := FindBrowser("linux", "", nil, has("chromium", "microsoft-edge"), never); got != "/usr/bin/chromium" {
		t.Errorf("linux order: %q", got)
	}
	if got := FindBrowser("linux", "", nil, has(), never); got != "" {
		t.Errorf("none: %q", got)
	}
	if got := FindBrowser("linux", "mybrowser", nil, has("mybrowser", "chromium"), never); got != "/usr/bin/mybrowser" {
		t.Errorf("override: %q", got)
	}
	if got := FindBrowser("linux", "gone", nil, has("chromium"), never); got != "" {
		t.Errorf("a bad override must not fall back silently: %q", got)
	}
	edge := filepath.Join("PF86", `Microsoft\Edge\Application\msedge.exe`)
	if got := FindBrowser("windows", "", []string{"", "PF86", "PF"}, has(), func(p string) bool { return p == edge }); got != edge {
		t.Errorf("windows edge: %q", got)
	}
}

func TestAppArgs(t *testing.T) {
	a := strings.Join(AppArgs("http://h:1/watch", "/p"), " ")
	for _, w := range []string{"--app=http://h:1/watch", "--user-data-dir=/p", "--class=d2idle-watch"} {
		if !strings.Contains(a, w) {
			t.Errorf("missing %s in %s", w, a)
		}
	}
}

func TestDesktopEntry(t *testing.T) {
	e := DesktopEntry("/opt/my app/d2idle-watch")
	for _, w := range []string{`Exec="/opt/my app/d2idle-watch" --page inventory`, `--page policy`, `--page return`, `--setup`, "StartupWMClass=d2idle-watch"} {
		if !strings.Contains(e, w) {
			t.Errorf("missing %q", w)
		}
	}
}

func newTestShell(t *testing.T, cfg Config) (*shell, *httptest.Server, string) {
	t.Helper()
	dir := t.TempDir()
	s, ln, err := newShell(dir, cfg, "inventory")
	if err != nil {
		t.Fatal(err)
	}
	srv := &httptest.Server{Listener: ln, Config: &http.Server{Handler: s.Handler()}}
	srv.Start()
	t.Cleanup(srv.Close)
	return s, srv, dir
}

func get(t *testing.T, s *shell, path string) map[string]string {
	t.Helper()
	resp, err := http.Get("http://" + s.addr + "/t/" + s.token + "/" + path)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	m := map[string]string{}
	json.NewDecoder(resp.Body).Decode(&m)
	return m
}

func TestShellFirstRunSaveAndStatus(t *testing.T) {
	game := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer game.Close()
	s, _, dir := newTestShell(t, Config{})
	if st := get(t, s, "status"); st["address"] != "" || st["state"] != "" {
		t.Fatalf("first run status: %v", st)
	}
	resp, err := http.PostForm("http://"+s.addr+"/t/"+s.token+"/save", url.Values{"address": {"http://user:pw@x:1"}})
	if err != nil || resp.StatusCode != 400 {
		t.Fatalf("credentials accepted: %v %v", resp, err)
	}
	resp, err = http.PostForm("http://"+s.addr+"/t/"+s.token+"/save", url.Values{"address": {strings.TrimPrefix(game.URL, "http://")}})
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("save: %v %v", resp, err)
	}
	if c, _ := LoadConfig(dir); c.Address != game.URL {
		t.Errorf("stored %q want %q", c.Address, game.URL)
	}
	st := get(t, s, "status")
	if st["state"] != "up" || st["target"] != game.URL+"/watch/inventory" {
		t.Errorf("status after save: %v", st)
	}
}

func TestShellOfflineAndGuards(t *testing.T) {
	dead := httptest.NewServer(http.NotFoundHandler())
	addr := dead.URL
	dead.Close()
	s, _, _ := newTestShell(t, Config{Address: addr})
	if st := get(t, s, "status"); st["state"] != "down" {
		t.Errorf("offline: %v", st)
	}
	// wrong token and rebinding Host are refused
	if resp, _ := http.Get("http://" + s.addr + "/t/wrong/status"); resp.StatusCode != 404 {
		t.Errorf("wrong token: %d", resp.StatusCode)
	}
	req, _ := http.NewRequest("GET", "http://"+s.addr+"/t/"+s.token+"/status", nil)
	req.Host = "evil.example"
	if resp, _ := http.DefaultClient.Do(req); resp.StatusCode != 403 {
		t.Errorf("rebinding host: %d", resp.StatusCode)
	}
	req, _ = http.NewRequest("POST", "http://"+s.addr+"/t/"+s.token+"/save", strings.NewReader("address=x:1"))
	req.Header.Set("Origin", "http://evil.example")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if resp, _ := http.DefaultClient.Do(req); resp.StatusCode != 403 {
		t.Errorf("foreign origin: %d", resp.StatusCode)
	}
}

func TestShellServeStopsWhenWindowGone(t *testing.T) {
	s, ln, err := newShell(t.TempDir(), Config{}, "watch")
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan struct{})
	go func() { s.Serve(context.Background(), ln, 50*time.Millisecond); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not stop after the idle time")
	}
}
