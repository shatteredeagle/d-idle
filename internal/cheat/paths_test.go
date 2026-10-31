package cheat

import (
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/paths"
)

func TestLedgerMigratesFromLegacyState(t *testing.T) {
	home := t.TempDir()
	l := paths.Layout{Root: filepath.Join(home, "data"), Home: home}
	os.MkdirAll(l.LegacyStateDir(), 0o755)
	os.WriteFile(l.LegacyLedger(), []byte(`{"kind":"x"}`+"\n"), 0o644)
	if err := MigrateLegacy(l); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(l.CheatLedger()); string(b) != `{"kind":"x"}`+"\n" {
		t.Fatalf("ledger not moved: %q", b)
	}
	if _, err := os.Stat(l.LegacyLedger()); err == nil {
		t.Error("legacy ledger left behind")
	}
	// A ledger at the new place is never overwritten.
	os.WriteFile(l.LegacyLedger(), []byte("stale\n"), 0o644)
	MigrateLegacy(l)
	if b, _ := os.ReadFile(l.CheatLedger()); string(b) != `{"kind":"x"}`+"\n" {
		t.Error("new ledger overwritten")
	}
}

func TestDefaultPathsFollowDataDir(t *testing.T) {
	d := t.TempDir()
	t.Setenv("D2BOT_DATA", d)
	t.Setenv("D2CHEAT_LEDGER", "")
	t.Setenv("D2CHEAT_BACKUPS", "")
	t.Setenv("HOME", t.TempDir()) // no legacy ~/.local/state, no ~/Games
	if got := DefaultLedgerPath(); got != filepath.Join(d, "ledger", "cheats.jsonl") {
		t.Errorf("ledger %s", got)
	}
	if got := DefaultBackupDir(); got != filepath.Join(d, "ledger", "backups") {
		t.Errorf("backups %s", got)
	}
	if got := DefaultSaveDir("i2"); got != filepath.Join(d, "prefixes", "i2", "proton", "pfx", "drive_c", "users", "steamuser", "Saved Games", "Diablo II") {
		t.Errorf("save dir %s", got)
	}
	t.Setenv("D2CHEAT_LEDGER", "/x/l.jsonl")
	if DefaultLedgerPath() != "/x/l.jsonl" {
		t.Error("env override")
	}
}
