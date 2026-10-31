package main

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/fleet"
	"d2lod-bot/internal/server"
	"d2lod-bot/internal/setup"
	"d2lod-bot/internal/setup/setuptest"
	"d2lod-bot/web"
)

// Wizard end to end, offline: detect game dir, pick class and goals, write config, then the managed
// bot slot appears, and only then. Host effects are fakes (setuptest), the runtime starts no game.
func TestWizardToManagedBot(t *testing.T) {
	root, _ := os.MkdirTemp("", "wiz")
	defer func() { time.Sleep(50 * time.Millisecond); os.RemoveAll(root) }()
	data := filepath.Join(root, "data")
	src := filepath.Join(root, "install")
	if err := setuptest.InstallTree(src); err != nil {
		t.Fatal(err)
	}
	patch := filepath.Join(root, "LODPatch_114d.exe")
	os.WriteFile(patch, []byte("p"), 0o644)

	f, err := fleet.New(fleet.Config{Runtime: fakeRuntime{}, DataDir: data, StateFile: filepath.Join(data, "fleet.json"), Limits: fleet.Limits{MaxLoadFactor: -1}})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close(context.Background())
	wz, err := setup.NewWeb(setup.WebConfig{DataDir: data, Deps: setuptest.Deps(data), BaseCtx: t.Context()})
	if err != nil {
		t.Fatal(err)
	}
	srv, err := server.New(server.Config{Token: "tok", Assets: web.FS, Setup: wz.Handler(), FleetAdmin: fleet.Handler(f, "")})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	done := make(chan error, 1)
	go func() { done <- afterWizard(t.Context(), wz, f) }()

	do := func(method, path, body string) (int, string) {
		req, _ := http.NewRequest(method, ts.URL+path, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer tok")
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		b, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, string(b)
	}
	wait := func(cond func(setup.View) bool) setup.View {
		for range 500 {
			_, b := do("GET", "/api/setup/state", "")
			var v setup.View
			if json.Unmarshal([]byte(b), &v) == nil && !v.Running && cond(v) {
				return v
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatal("timeout")
		return setup.View{}
	}

	in, _ := json.Marshal(setup.Inputs{Mode: setup.ModeGameDir, GameDirSrc: src, PatchFile: patch})
	if c, b := do("PUT", "/api/setup/inputs", string(in)); c != 200 {
		t.Fatal(c, b)
	}
	do("POST", "/api/setup/start", "")
	wait(func(v setup.View) bool { return v.Current == setup.StepCharacter })
	if n := len(f.List()); n != 0 {
		t.Fatalf("%d slots before the character exists", n)
	}
	if c, _ := do("PUT", "/api/setup/inputs", `{"charName":"Wizbot","charClass":"necromancer","profile":"rookie","goals":"levelto:8"}`); c != 200 {
		t.Fatal(c)
	}
	if c, _ := do("PUT", "/api/setup/inputs", `{"goals":"bad goals; rm"}`); c != 400 {
		t.Fatalf("bad profile accepted: %d", c)
	}
	do("POST", "/api/setup/start", "")
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(data, "setup.json")); err != nil {
		t.Fatal(err)
	}
	l := f.List()
	if len(l) != 1 || l[0].Spec.Character != "Wizbot" || l[0].Spec.Class != "necromancer" || l[0].Spec.Profile != "rookie" || l[0].Spec.Goals != "levelto:8" {
		t.Fatalf("slots %+v", l)
	}

	// restart: a new wizard on the same data is done at once, the saved slot is restored, no duplicate.
	f2, err := fleet.New(fleet.Config{Runtime: fakeRuntime{}, DataDir: data, StateFile: filepath.Join(data, "fleet.json"), Limits: fleet.Limits{MaxLoadFactor: -1}})
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close(context.Background())
	wz2, err := setup.NewWeb(setup.WebConfig{DataDir: data, Deps: setuptest.Deps(data), BaseCtx: t.Context()})
	if err != nil {
		t.Fatal(err)
	}
	if err := wz2.W.Start(t.Context()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := afterWizard(ctx, wz2, f2); err != nil {
		t.Fatal(err)
	}
	if len(f2.List()) != 1 {
		t.Fatalf("after restart: %+v", f2.List())
	}
}
