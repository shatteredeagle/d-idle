package main

import (
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/selfupdate"
)

func TestUpdateCheckSwitch(t *testing.T) {
	for v, want := range map[string]bool{"": true, "on": true, "off": false, "OFF": false, "0": false, "false": false, "no": false} {
		t.Setenv("D2BOT_UPDATE_CHECK", v)
		if got := updateCheckEnabled(); got != want {
			t.Errorf("D2BOT_UPDATE_CHECK=%q: %v", v, got)
		}
	}
}

func TestNewUpdaterNeedsRootAndKey(t *testing.T) {
	root, keys := t.TempDir(), t.TempDir()
	t.Setenv("D2BOT_UPDATE_DIR", filepath.Join(root, "missing"))
	t.Setenv("D2BOT_UPDATE_KEYS", keys)
	if newUpdater(nil) != nil {
		t.Fatal("updater without an update root")
	}
	t.Setenv("D2BOT_UPDATE_DIR", root)
	if newUpdater(nil) != nil {
		t.Fatal("updater without a trusted key")
	}
	pub, _, _ := selfupdate.GenerateKey(rand.Reader)
	os.WriteFile(filepath.Join(keys, "current.pub"), selfupdate.EncodePublicKey(pub), 0o644)
	os.WriteFile(filepath.Join(keys, "junk.pub"), []byte("not a key"), 0o644)
	t.Setenv("D2BOT_UPDATE_CHECK", "off")
	e := newUpdater(nil)
	if e == nil {
		t.Fatal("no updater with root and key")
	}
	if s := e.Status(); s.CheckEnabled || s.Running != version {
		t.Fatalf("status %+v", s)
	}
}
