package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/scenario"
)

func TestShipLedgerSpoolsAndIngests(t *testing.T) {
	out := t.TempDir()
	sink := filepath.Join(t.TempDir(), "sink")
	t.Setenv("PG_LEDGER_CMD", "cat >> "+sink)
	t.Setenv("PG_JOB", "j1")
	t.Setenv("PG_BRANCH", "main")
	res := scenario.Result{Run: scenario.Run{ID: "x", Class: "Paladin", Seed: 1}, Status: scenario.Green, Tags: []string{"CHEATED"},
		Prov: &scenario.Provenance{BotCommit: "abc", D2DSHA256: "ff", Host: "builder"}}
	shipLedger(res, "fast", out, time.Now())
	for _, p := range []string{filepath.Join(out, ledgerSpool), sink} {
		b, err := os.ReadFile(p)
		if err != nil || !strings.Contains(string(b), `"setup":"CHEATED"`) || strings.Count(string(b), "\n") != 1 {
			t.Fatalf("%s: %v %q", p, err, b)
		}
	}
}

func TestShipLedgerRefusedWritesNothing(t *testing.T) {
	out := t.TempDir()
	shipLedger(scenario.Result{Run: scenario.Run{ID: "x"}, Status: scenario.Green}, "fast", out, time.Now())
	if _, err := os.Stat(filepath.Join(out, ledgerSpool)); err == nil {
		t.Fatal("a run without provenance must not reach the ledger")
	}
}

func TestLedgerCommands(t *testing.T) {
	f := filepath.Join(t.TempDir(), "l.jsonl")
	line := `{"v":1,"run":"j/x/P/1","scenario":"x","class":"P","seed":1,"stability":"candidate","verdict":"GREEN","setup":"SETUP","bot_commit":"a","d2d_sha256":"b","host":"builder","virtual_min":1,"wall_s":1,"event":"manual","branch":"main","started":"2026-11-21T00:00:00Z"}` + "\n"
	_ = os.WriteFile(f, []byte(line), 0o644)
	var sb strings.Builder
	if err := cmdLedger([]string{"check", "-in", f}, &sb); err != nil || !strings.Contains(sb.String(), "ledger ok: 1") {
		t.Fatalf("%v %s", err, sb.String())
	}
	sb.Reset()
	if err := cmdLedger([]string{"status-doc", "-in", f, "-tiers", ""}, &sb); err != nil || !strings.Contains(sb.String(), "| x | P | ? | candidate | GREEN | SETUP |") {
		t.Fatalf("%v %s", err, sb.String())
	}
	sb.Reset()
	if err := cmdLedger([]string{"page-json", "-in", f, "-tiers", "", "-commit", "a"}, &sb); err != nil ||
		!strings.Contains(sb.String(), `"version": "didle.pg-page/1"`) || !strings.Contains(sb.String(), `"release_view": [`) {
		t.Fatalf("%v %s", err, sb.String())
	}
	rel := filepath.Join(t.TempDir(), "rel.json")
	_ = os.WriteFile(rel, []byte(`[{"scenario":"x","at":"2026-11-22T00:00:00Z"}]`), 0o644)
	if err := cmdLedger([]string{"status-doc", "-in", f, "-tiers", "", "-releases", rel}, &sb); err == nil {
		t.Fatal("a release without a person or root cause is refused")
	}
	_ = os.WriteFile(f, []byte(strings.Replace(line, `"host":"builder"`, `"host":""`, 1)), 0o644)
	if err := cmdLedger([]string{"check", "-in", f}, &sb); err == nil {
		t.Fatal("a line without a host fails the check")
	}
}
