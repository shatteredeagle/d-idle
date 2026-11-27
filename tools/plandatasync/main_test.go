package main

import (
	"os"
	"path/filepath"
	"testing"
)

// The committed mirrors equal the source only right after a merge regenerated them; agents do not edit them.
func TestMirrorsMatchSource(t *testing.T) {
	if os.Getenv("PLANDATA_STRICT") == "" {
		t.Skip("mirrors are regenerated at merge; set PLANDATA_STRICT=1 to check them")
	}
	ch, err := sync(filepath.Join("..", ".."), true)
	if err != nil || len(ch) > 0 {
		t.Fatalf("docs/plan/data is stale (go run ./tools/plandatasync): %v %v", ch, err)
	}
}

func TestSyncWritesOnlyExistingMirrors(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "docs", "plan", "data")
	os.MkdirAll(filepath.Join(dir, "builds-endgame"), 0o755)
	os.WriteFile(filepath.Join(dir, "areas.json"), []byte("old"), 0o644)
	os.WriteFile(filepath.Join(dir, "builds-endgame", "sorceress_blizzard_endgame90.json"), []byte("old"), 0o644)
	ch, err := sync(root, false)
	if err != nil || len(ch) != 2 {
		t.Fatalf("changed %v err %v", ch, err)
	}
	if ch, _ = sync(root, true); len(ch) != 0 {
		t.Fatalf("second pass still differs: %v", ch)
	}
	if _, err := os.Stat(filepath.Join(dir, "runs.json")); err == nil {
		t.Fatal("created a mirror that did not exist")
	}
}
