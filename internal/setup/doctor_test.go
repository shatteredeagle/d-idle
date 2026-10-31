package setup

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"d2lod-bot/internal/d2"
)

// realEnv: real file access over a temp dir, faked clock, network and game.
func realEnv() *Env {
	e := NewEnv()
	e.Dial = func(string) error { return errors.New("refused") }
	e.Player = func(string) (*d2.Player, error) { return nil, errors.New("no game") }
	e.Now = func() time.Time { return time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC) }
	return e
}

func runOne(t *testing.T, id string, e *Env, o Options) Result {
	t.Helper()
	o.DataDir = t.TempDir()
	return oneIn(t, id, e, o)
}

func oneIn(t *testing.T, id string, e *Env, o Options) Result {
	t.Helper()
	for _, r := range RunChecks(t.Context(), e, o, Checks()) {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no check %s", id)
	return Result{}
}

func write(t *testing.T, p, s string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(s), mode); err != nil {
		t.Fatal(err)
	}
}

func TestConfigParse(t *testing.T) {
	d := t.TempDir()
	o := Options{DataDir: d}
	if r := oneIn(t, "config.parse", realEnv(), o); r.Status != Pass {
		t.Errorf("absent: %+v", r)
	}
	write(t, filepath.Join(d, "config.json"), `{"a":1}`, 0o644)
	if r := oneIn(t, "config.parse", realEnv(), o); r.Status != Pass {
		t.Errorf("valid: %+v", r)
	}
	write(t, filepath.Join(d, "instances", "i0", "goals.json"), `{oops`, 0o644)
	if r := oneIn(t, "config.parse", realEnv(), o); r.Status != Fail || r.Fix == "" {
		t.Errorf("corrupt: %+v", r)
	}
}

func TestInstancePaths(t *testing.T) {
	d := t.TempDir()
	o := Options{DataDir: d, Slots: 1}
	if r := oneIn(t, "instance.paths", realEnv(), o); r.Status != Warn {
		t.Errorf("empty: %+v", r)
	}
	for _, p := range []string{"prefixes/i0", "instances/i0/saves", "instances/i0"} {
		if err := os.MkdirAll(filepath.Join(d, p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if r := oneIn(t, "instance.paths", realEnv(), o); r.Status != Pass {
		t.Errorf("laid out: %+v", r)
	}
	os.RemoveAll(filepath.Join(d, "instances", "i0", "saves"))
	write(t, filepath.Join(d, "instances", "i0", "saves"), "x", 0o644)
	if r := oneIn(t, "instance.paths", realEnv(), o); r.Status != Fail {
		t.Errorf("file for dir: %+v", r)
	}
}

func TestTablesAndTokens(t *testing.T) {
	d := t.TempDir()
	if r := oneIn(t, "data.tables", realEnv(), Options{DataDir: d, GameExpected: true}); r.Status != Fail {
		t.Errorf("tables: %+v", r)
	}
	if r := oneIn(t, "data.tables", realEnv(), Options{DataDir: d}); r.Status != Warn {
		t.Errorf("pre-install: %+v", r)
	}
	write(t, filepath.Join(d, "cache", "data", "x", "a.tsv"), "x", 0o644)
	if r := oneIn(t, "data.tables", realEnv(), Options{DataDir: d}); r.Status != Pass {
		t.Errorf("tables ok: %+v", r)
	}
	write(t, filepath.Join(d, "tokens", "d2watch"), "secret", 0o600)
	if r := oneIn(t, "tokens.perms", realEnv(), Options{DataDir: d}); r.Status != Pass {
		t.Errorf("0600: %+v", r)
	}
	write(t, filepath.Join(d, "tokens", "d2fleet"), "secret", 0o644)
	if err := os.Chmod(filepath.Join(d, "tokens", "d2fleet"), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := oneIn(t, "tokens.perms", realEnv(), Options{DataDir: d}); r.Status != Fail || r.Fix == "" {
		t.Errorf("0644: %+v", r)
	}
}

func TestHelperReach(t *testing.T) {
	e := realEnv()
	if r := runOne(t, "helper.reach", e, Options{Slots: 2}); r.Status != Warn {
		t.Errorf("down: %+v", r)
	}
	e.Dial = func(string) error { return nil }
	if r := runOne(t, "helper.reach", e, Options{Slots: 2}); r.Status != Pass {
		t.Errorf("up: %+v", r)
	}
}

func TestLoad(t *testing.T) {
	for load, want := range map[string]Status{"0.5 0.5 0.5 1/100 1\n": Pass, "7.5 1 1 1/1 1\n": Warn, "20 1 1 1/1 1\n": Fail} {
		e := fakeEnv(map[string]string{"/proc/loadavg": load})
		e.NumCPU = 8
		if r := one(t, "resources.load", e, Options{Slots: 2}); r.Status != want {
			t.Errorf("%q: %+v", load, r)
		}
	}
}

func TestLoadWarnInContainerOrGateOff(t *testing.T) {
	busy := "20 1 1 1/1 1\n"
	e := fakeEnv(map[string]string{"/proc/loadavg": busy, "/.dockerenv": ""})
	if r := one(t, "resources.load", e, Options{Slots: 2}); r.Status != Warn {
		t.Errorf("container: %+v", r)
	}
	e = fakeEnv(map[string]string{"/proc/loadavg": busy})
	e.Getenv = func(k string) string {
		if k == "D2FLEET_MAX_LOAD_FACTOR" {
			return "-1"
		}
		return ""
	}
	if r := one(t, "resources.load", e, Options{Slots: 2}); r.Status != Warn {
		t.Errorf("gate off: %+v", r)
	}
}

func TestClock(t *testing.T) {
	e := realEnv()
	e.Now = time.Now
	if r := runOne(t, "clock.sanity", e, Options{}); r.Status != Pass {
		t.Errorf("sane: %+v", r)
	}
	e.Now = func() time.Time { return time.Date(1970, 1, 1, 0, 0, 0, 0, time.UTC) }
	if r := runOne(t, "clock.sanity", e, Options{}); r.Status != Fail {
		t.Errorf("epoch: %+v", r)
	}
	e.Now = func() time.Time { return time.Now().Add(-time.Hour) }
	if r := runOne(t, "clock.sanity", e, Options{}); r.Status != Warn {
		t.Errorf("behind: %+v", r)
	}
}

func TestCharacter(t *testing.T) {
	e := realEnv()
	if r := runOne(t, "char.difficulty", e, Options{}); r.Status != Skip {
		t.Errorf("detached: %+v", r)
	}
	e.Player = func(string) (*d2.Player, error) {
		return &d2.Player{Difficulty: d2.Nightmare, Level: 40}, nil
	}
	r := runOne(t, "char.difficulty", e, Options{})
	if r.Status != Pass || r.Detail == "" {
		t.Errorf("attached: %+v", r)
	}
	t.Log(r.Detail)
}
