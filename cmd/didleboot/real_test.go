package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/selfupdate"
)

// Real processes, real clock, no network: the shim runs shell launchers.
func launcher(t *testing.T, dir, body string) string {
	t.Helper()
	os.MkdirAll(dir, 0o755)
	p := filepath.Join(dir, "entrypoint.sh")
	if err := os.WriteFile(p, []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return p
}

func realEnv(t *testing.T) (root, baseBin string) {
	root = filepath.Join(t.TempDir(), "didle")
	baseBin = filepath.Join(t.TempDir(), "base")
	t.Setenv("DIDLE_ROOT", root)
	t.Setenv("DIDLE_BASE_BIN", baseBin)
	t.Setenv("DIDLE_BASE_D2D", filepath.Join(baseBin, "d2d"))
	t.Setenv("DIDLE_BASE_VERSION", "0.3.0")
	t.Setenv("DIDLE_KEYS", t.TempDir())
	t.Setenv("D2BOT_IMAGE_ABI", "2")
	t.Setenv("D2BOT_ADDR", "127.0.0.1:1")
	t.Setenv("TESTER", "")
	return
}

func TestRealBaseLauncherExitCodeAndArgs(t *testing.T) {
	_, baseBin := realEnv(t)
	out := filepath.Join(t.TempDir(), "out")
	launcher(t, baseBin, `echo "$DIDLE_RELEASE $@" > `+out+`; exit 7`)
	if code := realMain([]string{"tester", "x"}); code != 7 {
		t.Fatalf("exit code %d, want the launcher's 7", code)
	}
	if b, _ := os.ReadFile(out); strings.TrimSpace(string(b)) != "0.3.0 tester x" {
		t.Fatalf("launcher saw %q", b)
	}
}

func TestRealLauncherFlag(t *testing.T) {
	_, baseBin := realEnv(t)
	custom := launcher(t, filepath.Join(baseBin, "elsewhere"), `exit 4`)
	if code := realMain([]string{"--launcher", custom, "bot"}); code != 4 {
		t.Fatalf("exit code %d, want 4 from --launcher", code)
	}
}

func TestRealPendingReleaseThatDiesIsRolledBack(t *testing.T) {
	root, baseBin := realEnv(t)
	pub, priv, _ := selfupdate.GenerateKey(strings.NewReader(strings.Repeat("seed", 64)))
	keys := os.Getenv("DIDLE_KEYS")
	os.WriteFile(filepath.Join(keys, "current.pub"), selfupdate.EncodePublicKey(pub), 0o644)
	out := filepath.Join(t.TempDir(), "out")
	launcher(t, baseBin, `echo "base $DIDLE_RELEASE" >> `+out+`; exit 0`)

	l := selfupdate.Layout{Root: root}
	rdir := l.Release("0.4.0")
	launcher(t, filepath.Join(rdir, "bin"), `echo "new $DIDLE_RELEASE" >> `+out+`; exit 1`)
	os.WriteFile(filepath.Join(rdir, "bin", "d2bot"), []byte("x"), 0o755)
	m := selfupdate.Manifest{Schema: 1, Version: "0.4.0", Seq: 2, Issued: "2026-11-20T00:00:00Z", Channel: "stable", MinImage: 1,
		Compat: selfupdate.BuildCompat, Files: []selfupdate.File{{Name: "a", SHA256: strings.Repeat("0", 64), Size: 1, Dest: "bin"}}}
	raw, err := m.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(rdir, selfupdate.ManifestName), raw, 0o644)
	os.WriteFile(filepath.Join(rdir, selfupdate.ManifestName+".sig"), selfupdate.Sign(priv, raw), 0o644)
	l.SwapCurrent("0.4.0")
	l.SaveState(selfupdate.State{Current: "0.4.0", Pending: "0.4.0"})

	if code := realMain(nil); code != 0 {
		t.Fatalf("exit code %d", code)
	}
	if b, _ := os.ReadFile(out); strings.TrimSpace(string(b)) != "new 0.4.0\nbase 0.3.0" {
		t.Fatalf("run order %q", b)
	}
	st, _ := l.LoadState()
	if !st.IsBad("0.4.0") || st.Pending != "" || l.CurrentVersion() != "" {
		t.Fatalf("state %+v current %q", st, l.CurrentVersion())
	}
	if mk, _ := l.ReadMarker(); mk == nil || !strings.Contains(mk.Reason, "exited with code 1") {
		t.Fatalf("marker %+v", mk)
	}
}
