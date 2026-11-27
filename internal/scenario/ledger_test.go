package scenario

import (
	"strings"
	"testing"
	"time"
)

func provRes() Result {
	r := Result{Run: Run{ID: "town.resupply", Class: "Paladin", Seed: 11}, Status: Green, Tags: []string{"SETUP-only"},
		AssertDigest: "d1", Digest: "d2", VirtualMin: 15, WallSec: 40,
		Prov: &Provenance{BotCommit: "abc1234", D2DSHA256: "ff00", Host: "builder", Load1: 3, Pace: 22, Parallel: 2}}
	r.State = &State{Source: "d2d-observer-reload"}
	return r
}

var ctx0 = LedgerCtx{Job: "20261121-1", Event: "nightly", Branch: "main", Started: time.Date(2026, 11, 21, 7, 0, 0, 0, time.UTC)}

func TestLedgerRecordRefusesWithoutProvenance(t *testing.T) {
	r := provRes()
	r.Prov = nil
	if _, err := NewLedgerRecord(r, ctx0); err == nil || !strings.Contains(err.Error(), "bot_commit") {
		t.Fatalf("a run without provenance must be refused, got %v", err)
	}
	r = provRes()
	r.Tags = nil
	if _, err := NewLedgerRecord(r, ctx0); err == nil || !strings.Contains(err.Error(), "setup tag") {
		t.Fatalf("a run with no declared setup must be refused, got %v", err)
	}
	r = provRes()
	r.Prov.Host = "unknown"
	if _, err := NewLedgerRecord(r, ctx0); err == nil {
		t.Fatal("host role unknown must be refused")
	}
}

func TestLedgerRecordFields(t *testing.T) {
	rec, err := NewLedgerRecord(provRes(), ctx0)
	if err != nil {
		t.Fatal(err)
	}
	if rec.Setup != SetupSetup || rec.Run != "20261121-1/town.resupply/Paladin/11" || rec.D1 != "d1" || rec.Stability != "candidate" || rec.WritesN == nil {
		t.Fatalf("%+v", rec)
	}
	for tags, want := range map[string]string{"CHEATED,SETUP-only": SetupCheated, "IMPOSSIBLE,CHEATED": SetupImpossible, "EARNED": SetupEarned, "SETUP-only": SetupSetup} {
		if got := SetupTag(strings.Split(tags, ",")); got != want {
			t.Errorf("%s: %s want %s", tags, got, want)
		}
	}
	b, err := rec.LedgerLine()
	if err != nil || strings.Contains(string(b), "\n") {
		t.Fatalf("one line: %v %q", err, b)
	}
	back, err := ReadLedger(strings.NewReader(string(b) + "\n\n"))
	if err != nil || len(back) != 1 || back[0].Run != rec.Run || back[0].Parallel != 2 || back[0].Pace != 22 || back[0].Host != "builder" {
		t.Fatalf("%v %+v", err, back)
	}
	if _, err := ReadLedger(strings.NewReader("{bad")); err == nil || !strings.Contains(err.Error(), "line 1") {
		t.Fatalf("a bad line names its number, got %v", err)
	}
}

func TestStatusDocIsDeterministicAndFiltersBranch(t *testing.T) {
	a, _ := NewLedgerRecord(provRes(), ctx0)
	b := a
	b.Branch, b.Verdict, b.Started = "agent/x", Red, "2026-11-21T10:00:00Z"
	c := a
	c.Started, c.D1 = "2026-11-21T09:00:00Z", "other" // same binaries, other D1: a flip
	doc := StatusDoc([]LedgerRecord{a, b, c}, "main")
	if doc != StatusDoc([]LedgerRecord{a, b, c}, "main") {
		t.Fatal("not deterministic")
	}
	if strings.Contains(doc, "RED") || !strings.Contains(doc, "| town.resupply | Paladin | GREEN | SETUP | 2 | 1 | other |") {
		t.Fatalf("branch filter or flip count wrong:\n%s", doc)
	}
	if !strings.Contains(StatusDoc([]LedgerRecord{a, b, c}, ""), "RED") {
		t.Fatal("empty branch shows all")
	}
}
