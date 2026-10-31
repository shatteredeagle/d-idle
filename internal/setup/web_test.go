package setup

import (
	"bytes"
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

	"d2lod-bot/internal/launch"
	"d2lod-bot/internal/server"
	"d2lod-bot/web"
)

// fakeDeps is the game-dir plan with every host effect faked (same shape as TestDefaultStepsGameDirMode).
func fakeDeps(t *testing.T, data string) *Deps {
	env := NewEnv()
	env.GameVersion = func(string) (string, error) { return "1.14.3.71", nil }
	env.SHA256 = func(string) (string, error) { return launch.PatchSHA256, nil }
	d := NewDeps(env)
	d.Config = func(s *State) (launch.Config, error) {
		return launch.Config{Root: s.DataDir, PrefixDir: BasePrefix(s.DataDir), GameDir: GameDirIn(BasePrefix(s.DataDir))}, nil
	}
	d.Prefix = func(ctx context.Context, c launch.Config) error {
		p := filepath.Join(c.PrefixDir, "pfx")
		os.MkdirAll(p, 0o755)
		return os.WriteFile(filepath.Join(p, "system.reg"), []byte("r"), 0o644)
	}
	var extracted bool
	d.TablesDir = func(dd, g string) (string, error) { return filepath.Join(dd, "cache", "t"), nil }
	d.CacheCurrent = func(string, string) bool { return extracted }
	d.Extract = func(string, string) error { extracted = true; return nil }
	d.CreateCharacter = func(ctx context.Context, c launch.Config, n, cl string) error {
		os.MkdirAll(c.SaveDir(), 0o755)
		return os.WriteFile(filepath.Join(c.SaveDir(), n+".d2s"), []byte("x"), 0o644)
	}
	d.RunWindowed = func(context.Context, launch.Config, string, string, func() error) error {
		t.Error("no window expected")
		return nil
	}
	helper := filepath.Join(data, "helper", "d2input.exe")
	os.MkdirAll(filepath.Dir(helper), 0o755)
	os.WriteFile(helper, []byte("h"), 0o644)
	return d
}

func TestWebEndToEnd(t *testing.T) {
	root := t.TempDir()
	data := filepath.Join(root, "fresh-data") // does not exist yet
	src := filepath.Join(root, "myd2")
	installTree(t, src)
	patch := filepath.Join(root, "patch.exe")
	os.WriteFile(patch, []byte("p"), 0o644)

	wz, err := NewWeb(WebConfig{DataDir: data, Deps: fakeDeps(t, data), BaseCtx: t.Context()})
	if err != nil {
		t.Fatal(err)
	}
	srv, err := server.New(server.Config{Token: "tok", Assets: web.FS, Setup: wz.Handler()})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

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
	state := func() View {
		_, b := do("GET", "/api/setup/state", "")
		var v View
		if err := json.Unmarshal([]byte(b), &v); err != nil {
			t.Fatal(err, b)
		}
		return v
	}
	wait := func(cond func(View) bool) View {
		for i := 0; i < 500; i++ {
			if v := state(); !v.Running && cond(v) {
				return v
			}
			time.Sleep(10 * time.Millisecond)
		}
		t.Fatalf("timeout: %+v", state())
		return View{}
	}

	if c, b := do("GET", "/setup.html", ""); c != 200 || !strings.Contains(b, "<") {
		t.Fatalf("setup.html %d", c)
	}
	if wz.Ready() {
		t.Fatal("fresh wizard is ready")
	}
	in, _ := json.Marshal(Inputs{Mode: ModeGameDir, GameDirSrc: src, PatchFile: patch})
	if c, b := do("PUT", "/api/setup/inputs", string(in)); c != 200 {
		t.Fatal(c, b)
	}
	// run until the character step wants a name, then hold
	if c, b := do("POST", "/api/setup/start", ""); c != 202 {
		t.Fatal(c, b)
	}
	v := wait(func(v View) bool { return v.Current == StepCharacter })
	if v.Done || wz.Ready() {
		t.Fatalf("done without a character: %+v", v.Steps)
	}
	if c, b := do("PUT", "/api/setup/inputs", `{"charName":"Wizbot","cdKey":"AAAA-BBBB-CCCC"}`); c != 400 {
		t.Fatalf("cdKey accepted: %d %s", c, b)
	}
	if c, b := do("PUT", "/api/setup/inputs", `{"charName":"Wizbot"}`); c != 200 {
		t.Fatal(c, b)
	}
	if c, b := do("POST", "/api/setup/start", ""); c != 202 {
		t.Fatal(c, b)
	}
	v = wait(func(v View) bool { return v.Done })
	if !wz.Ready() {
		t.Fatal("not ready after done")
	}
	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()
	if err := wz.WaitReady(ctx); err != nil {
		t.Fatal(err)
	}
	// nothing secret on disk
	b, _ := os.ReadFile(filepath.Join(data, "setup.json"))
	if bytes.Contains(b, []byte("AAAA")) || bytes.Contains(bytes.ToLower(b), []byte("cdkey")) {
		t.Fatal("setup.json holds key material")
	}
	if _, err := os.Stat(filepath.Join(data, "setup.json")); err != nil {
		t.Fatal(err)
	}
}
