package main

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"d2lod-bot/internal/selfupdate"
)

func TestUpdateCheckSwitch(t *testing.T) {
	for v, want := range map[string]bool{"": true, "on": true, "manual": true, "off": false, "OFF": false, "0": false, "false": false, "no": false} {
		t.Setenv("D2BOT_UPDATE_CHECK", v)
		if got := updateCheckMode() != selfupdate.CheckOff; got != want {
			t.Errorf("D2BOT_UPDATE_CHECK=%q: %v", v, got)
		}
	}
}

func TestNewUpdaterNeedsRootAndKey(t *testing.T) {
	root, keys := t.TempDir(), t.TempDir()
	t.Setenv("D2BOT_UPDATE_DIR", filepath.Join(root, "missing"))
	t.Setenv("D2BOT_UPDATE_KEYS", keys)
	if newUpdater(nil, nil) != nil {
		t.Fatal("updater without an update root")
	}
	t.Setenv("D2BOT_UPDATE_DIR", root)
	if newUpdater(nil, nil) != nil {
		t.Fatal("updater without a trusted key")
	}
	pub, _, _ := selfupdate.GenerateKey(rand.Reader)
	os.WriteFile(filepath.Join(keys, "current.pub"), selfupdate.EncodePublicKey(pub), 0o644)
	os.WriteFile(filepath.Join(keys, "junk.pub"), []byte("not a key"), 0o644)
	t.Setenv("D2BOT_UPDATE_CHECK", "off")
	e := newUpdater(nil, nil)
	if e == nil {
		t.Fatal("no updater with root and key")
	}
	if s := e.Status(); s.CheckEnabled || s.Running != version {
		t.Fatalf("status %+v", s)
	}
}

// A DEV image trusts the dev key although no key is baked; a normal image ignores the same variables.
func TestNewUpdaterDevSource(t *testing.T) {
	root, keys := t.TempDir(), t.TempDir()
	t.Setenv("D2BOT_UPDATE_DIR", root)
	t.Setenv("D2BOT_UPDATE_KEYS", keys)
	pub, _, _ := selfupdate.GenerateKey(rand.Reader)
	t.Setenv("D2BOT_UPDATE_URL", "http://lan.example:8899")
	t.Setenv("D2BOT_UPDATE_PUBKEY", string(selfupdate.EncodePublicKey(pub)))
	if newUpdater(nil, nil) != nil {
		t.Fatal("a normal image honoured the dev variables")
	}
	t.Setenv("DIDLE_DEV", "1")
	if newUpdater(nil, nil) == nil {
		t.Fatal("a dev image did not take the dev key")
	}
}

// D2BOT_UPDATE_FEED is honoured on a normal image but never adds a key: with no baked key there is no updater.
func TestNewUpdaterFeedAddsNoTrust(t *testing.T) {
	root, keys := t.TempDir(), t.TempDir()
	t.Setenv("D2BOT_UPDATE_DIR", root)
	t.Setenv("D2BOT_UPDATE_KEYS", keys)
	t.Setenv("D2BOT_UPDATE_FEED", "https://feed.example")
	pub, _, _ := selfupdate.GenerateKey(rand.Reader)
	t.Setenv("D2BOT_UPDATE_PUBKEY", string(selfupdate.EncodePublicKey(pub)))
	if newUpdater(nil, nil) != nil {
		t.Fatal("feed or dev key variable created trust")
	}
	os.WriteFile(filepath.Join(keys, "beta.pub"), selfupdate.EncodePublicKey(pub), 0o644)
	if newUpdater(nil, nil) == nil {
		t.Fatal("no updater with a baked key and a feed")
	}
	t.Setenv("D2BOT_UPDATE_FEED", "http://feed.example") // refused, falls back, still an updater
	if newUpdater(nil, nil) == nil {
		t.Fatal("bad feed killed the updater")
	}
}

func TestUpdaterGuardsAreSet(t *testing.T) {
	c := updaterCfg{}.withGuards(t.TempDir(), func() bool { return false })
	if c.IdleSafe == nil || c.SelfTest == nil || c.DiskFree == nil {
		t.Fatalf("guards missing: %+v", c)
	}
	if c.IdleSafe() {
		t.Fatal("IdleSafe not the given predicate")
	}
	if n, err := c.DiskFree(); runtime.GOOS == "linux" && (err != nil || n == 0) {
		t.Fatalf("disk free %d %v", n, err)
	}
}

func TestReleaseSelfTest(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell stub")
	}
	dir := t.TempDir()
	if err := releaseSelfTest(dir); err != nil {
		t.Fatalf("data-only release: %v", err)
	}
	bin := filepath.Join(dir, "bin")
	os.MkdirAll(bin, 0o755)
	write := func(body string) {
		os.WriteFile(filepath.Join(bin, "d2bot"), []byte(body), 0o755)
	}
	write("#!/bin/sh\necho d2bot 1.2.3\n")
	if err := releaseSelfTest(dir); err != nil {
		t.Fatalf("good binary: %v", err)
	}
	write("#!/bin/sh\nexit 3\n")
	if releaseSelfTest(dir) == nil {
		t.Fatal("a binary that fails to start passed")
	}
	write("#!/bin/sh\necho hello\n")
	if releaseSelfTest(dir) == nil {
		t.Fatal("wrong output passed")
	}
}
