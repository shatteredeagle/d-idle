package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestLoopback(t *testing.T) {
	for addr, want := range map[string]bool{"127.0.0.1:1": true, "[::1]:1": true, "localhost:1": true, ":8890": false, "0.0.0.0:1": false, "192.0.2.2:1": false} {
		if loopback(addr) != want {
			t.Errorf("loopback(%q) != %v", addr, want)
		}
	}
}

func TestGuard(t *testing.T) {
	h := guard("sek", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) }))
	do := func(method, url string, hdr map[string]string) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, url, nil)
		for k, v := range hdr {
			r.Header.Set(k, v)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		return w
	}
	if c := do("GET", "/", nil).Code; c != 401 {
		t.Errorf("no token: %d", c)
	}
	if c := do("GET", "/?token=bad", nil).Code; c != 401 {
		t.Errorf("bad token: %d", c)
	}
	w := do("GET", "/?token=sek", nil)
	if w.Code != 200 || len(w.Result().Cookies()) != 1 {
		t.Errorf("good token: %d cookies %d", w.Code, len(w.Result().Cookies()))
	}
	if c := do("GET", "/", map[string]string{"X-Token": "sek"}).Code; c != 200 {
		t.Errorf("header token: %d", c)
	}
	if c := do("POST", "/?token=sek", nil).Code; c != 405 {
		t.Errorf("POST must be refused: %d", c)
	}
}

func TestAreaName(t *testing.T) {
	if areaName(1) != "Rogue Encampment" || areaName(68) != "Tal Rasha's Tomb" || areaName(0) != "" || areaName(999) != "Area 999" {
		t.Error("areaName")
	}
}

func TestResolveTokenFileMovesLegacy(t *testing.T) {
	home, data := t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("D2BOT_DATA", data)
	old := filepath.Join(home, ".config", "d2watch", "token")
	os.MkdirAll(filepath.Dir(old), 0o700)
	os.WriteFile(old, []byte("secret\n"), 0o600)
	got := resolveTokenFile("")
	if got != filepath.Join(data, "tokens", "d2watch") {
		t.Fatalf("path %s", got)
	}
	if b, _ := os.ReadFile(got); string(b) != "secret\n" {
		t.Error("token not moved")
	}
	if st, _ := os.Stat(got); st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v", st.Mode())
	}
	if resolveTokenFile("/x/t") != "/x/t" {
		t.Error("flag wins")
	}
}
