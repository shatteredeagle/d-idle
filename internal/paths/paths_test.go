package paths

import (
	"d2lod-bot/internal/saves"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }
func home() (string, error)                       { return "/h/u", nil }

func TestFromEnvPrecedence(t *testing.T) {
	l, _ := fromEnv(env(nil), home)
	if l.Root != "/h/u/.local/share/d2bot" || l.Config() != "/h/u/.local/share/d2bot/config.json" || l.Run() != "/h/u/.local/share/d2bot/run" {
		t.Fatalf("defaults: %+v", l)
	}
	l, _ = fromEnv(env(map[string]string{"XDG_DATA_HOME": "/x", "XDG_RUNTIME_DIR": "/r"}), home)
	if l.Root != "/x/d2bot" || l.Run() != "/r/d2bot" {
		t.Fatalf("xdg: %+v", l)
	}
	l, _ = fromEnv(env(map[string]string{"D2BOT_DATA": "/d", "XDG_DATA_HOME": "/x", "D2BOT_CONFIG": "/c.json"}), home)
	if l.Root != "/d" || l.Config() != "/c.json" {
		t.Fatalf("override: %+v", l)
	}
}

func TestInstancePaths(t *testing.T) {
	l := Layout{Root: "/d"}
	i := l.MustInstance("i3")
	want := map[string]string{
		i.Runtime():                   "/d/instances/i3/runtime.json",
		i.State():                     "/d/instances/i3/state",
		i.BackupDir():                 "/d/instances/i3/saves-backup",
		i.Logs():                      "/d/instances/i3/logs",
		i.Fixtures():                  "/d/instances/i3/fixtures",
		i.Prefix():                    "/d/prefixes/i3",
		i.GameLock():                  "/d/run/game-i3.lock",
		l.DataTables("v2", "abc"):     "/d/cache/data/v2-abc",
		l.CheatLedger():               "/d/ledger/cheats.jsonl",
		i.WithSaveDir("/w").SaveDir(): "/w",
	}
	for got, w := range want {
		if got != w {
			t.Errorf("got %s want %s", got, w)
		}
	}
}

func TestBadInstance(t *testing.T) {
	for _, id := range []string{"", "..", "../x", "a/b", ".hidden", "a b"} {
		if _, err := (Layout{Root: "/d"}).Instance(id); !errors.Is(err, ErrBadInstance) {
			t.Errorf("%q accepted", id)
		}
	}
}

func TestMkdirAll(t *testing.T) {
	l := Layout{Root: t.TempDir()}
	i := l.MustInstance("i0")
	if err := i.MkdirAll(); err != nil {
		t.Fatal(err)
	}
	for _, d := range []string{i.State(), i.Logs(), i.Fixtures(), i.BackupDir(), l.Run()} {
		if !dirExists(d) {
			t.Errorf("missing %s", filepath.Base(d))
		}
	}
}

// Instance must satisfy the saves.Paths stub without an adapter.
var _ saves.Paths = Instance{}

func TestGameRootAndLocks(t *testing.T) {
	home := t.TempDir()
	l := Layout{Root: filepath.Join(home, "d2"), Home: home}
	if got := l.GameRoot("i2"); got != filepath.Join(home, "d2", "prefixes", "i2") {
		t.Fatalf("fresh: %s", got)
	}
	os.MkdirAll(filepath.Join(home, "Games", "d2lod"), 0o755)
	if l.GameRoot("i0") != filepath.Join(home, "Games", "d2lod") || l.GameRoot("i2") != filepath.Join(home, "Games", "d2lod-inst", "i2") {
		t.Fatalf("legacy: %s %s", l.GameRoot("i0"), l.GameRoot("i2"))
	}
	os.MkdirAll(filepath.Join(home, "d2", "prefixes", "i0"), 0o755)
	if l.GameRoot("i0") != filepath.Join(home, "d2", "prefixes", "i0") {
		t.Fatal("new layout must win")
	}
	if got := l.GameLocks("i3"); len(got) != 2 || got[0] != l.MustInstance("i3").GameLock() || got[1] != "/tmp/d2lod-game-i3.lock" {
		t.Fatalf("locks %v", got)
	}
	if LegacyGameLock("i0") != "/tmp/d2lod-game.lock" {
		t.Fatal("i0 legacy lock")
	}
}

func TestMoveIfAbsent(t *testing.T) {
	d := t.TempDir()
	from, to := filepath.Join(d, "old", "a.jsonl"), filepath.Join(d, "new", "x", "a.jsonl")
	os.MkdirAll(filepath.Dir(from), 0o755)
	os.WriteFile(from, []byte("old"), 0o644)
	if ok, err := MoveIfAbsent(from, to); !ok || err != nil {
		t.Fatalf("move: %v %v", ok, err)
	}
	if b, _ := os.ReadFile(to); string(b) != "old" {
		t.Fatal("content")
	}
	if _, err := os.Stat(from); err == nil {
		t.Fatal("source kept")
	}
	// the new file wins: a second legacy file is left alone
	os.WriteFile(from, []byte("again"), 0o644)
	if ok, _ := MoveIfAbsent(from, to); ok {
		t.Fatal("overwrote")
	}
	if ok, err := MoveIfAbsent(filepath.Join(d, "missing"), filepath.Join(d, "y")); ok || err != nil {
		t.Fatal("missing source")
	}
}
