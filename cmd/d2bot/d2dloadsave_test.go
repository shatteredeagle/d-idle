package main

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/env"
)

func TestClassForStartLoadsExistingSave(t *testing.T) {
	home := t.TempDir()
	log := slog.New(slog.DiscardHandler)
	if got := classForStart(3, home, "Doofer", log); got != 3 {
		t.Fatalf("no save: class %d, want 3 (create)", got)
	}
	dir := filepath.Join(home, "config", "d2d", "save")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Doofer.d2s"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, c := range []int{3, 1} { // two classes: the rule is not class specific
		if got := classForStart(c, home, "Doofer", log); got != -1 {
			t.Fatalf("saved hero, class %d: got %d, want -1 (load)", c, got)
		}
	}
	if got := classForStart(3, home, "Other", log); got != 3 {
		t.Fatalf("other name: got %d, want 3", got)
	}
	if got := classForStart(-1, home, "Doofer", log); got != -1 {
		t.Fatalf("load stays load: %d", got)
	}
}

// The restart path: the config a managed d2bot builds must load, not create, when the save exists.
func TestD2DConfigLoadsSavedHero(t *testing.T) {
	home := t.TempDir()
	dir := filepath.Join(home, "config", "d2d", "save")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "Doofer.d2s"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	var o options
	o.char, o.managed, o.d2d.class = "Doofer", true, 3
	cfg := d2dConfig(o, slog.New(slog.DiscardHandler), &env.Tuning{}, nil, home)
	if cfg.Spec.Class != -1 {
		t.Fatalf("restart over a saved hero would create a new one: class %d", cfg.Spec.Class)
	}
}
