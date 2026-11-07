package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2"
)

func doc(t *testing.T, dir, name string, class d2.Class, level int) string {
	t.Helper()
	s := &d2.Snapshot{State: d2.StateInGame, Player: &d2.Player{Class: class, Level: level, MaxLife: 500},
		Inventory: &d2.Inventory{}}
	b, err := json.Marshal(map[string]any{"snapshot": s, "progress": &d2.PlayerProgress{}})
	if err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, name)
	os.WriteFile(p, b, 0o644)
	return p
}

func TestReplayExitCodes(t *testing.T) {
	d := t.TempDir()
	a := doc(t, d, "a.json", d2.Paladin, 1)
	same := doc(t, d, "b.json", d2.Paladin, 1)
	diff := doc(t, d, "c.json", d2.Paladin, 2)
	args := func(b string) []string {
		return []string{"-scenario", "fresh-paladin-1", "-a", "json:" + a, "-b", "json:" + b, "-reads", "1", "-out", filepath.Join(d, "out")}
	}
	var o, e bytes.Buffer
	if rc := run(args(same), &o, &e); rc != 0 || !strings.Contains(o.String(), "verdict PASS") {
		t.Fatalf("rc %d %s %s", rc, o.String(), e.String())
	}
	o.Reset()
	if rc := run(args(diff), &o, &e); rc != 3 || !strings.Contains(o.String(), "MISMATCH") {
		t.Fatalf("rc %d %s", rc, o.String())
	}
	if _, err := os.Stat(filepath.Join(d, "out", "fresh-paladin-1.json")); err != nil {
		t.Fatal(err)
	}
	// wrong class for the scenario is refused, not compared
	o.Reset()
	if rc := run([]string{"-scenario", "1", "-a", "json:" + a, "-b", "json:" + same, "-reads", "1"}, &o, &e); rc != 1 {
		t.Fatalf("rc %d", rc)
	}
}

func TestListAndErrors(t *testing.T) {
	var o, e bytes.Buffer
	if rc := run([]string{"-list"}, &o, &e); rc != 0 || strings.Count(o.String(), "\n") != 15 {
		t.Fatalf("rc %d\n%s", rc, o.String())
	}
	if rc := run([]string{"-scenario", "99"}, &o, &e); rc != 2 {
		t.Fatal(rc)
	}
	if rc := run([]string{"-scenario", "1", "-saves", t.TempDir()}, &o, &e); rc != 2 {
		t.Fatal(rc)
	}
	if rc := run([]string{"-scenario", "1", "-a", "bogus"}, &o, &e); rc != 2 {
		t.Fatal(rc)
	}
}
