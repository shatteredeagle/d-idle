package main

import (
	"bytes"
	"strings"
	"testing"
)

const nightlyJSON = "../../tools/provingground/nightly.json"

func TestNightlyPlanOnShippedData(t *testing.T) {
	var b bytes.Buffer
	if err := cmdNightly([]string{"check", "-file", nightlyJSON, "-dir", dir, "-day", "2026-11-29"}, &b); err != nil {
		t.Fatal(err)
	}
	b.Reset()
	if err := cmdNightly([]string{"plan", "-file", nightlyJSON, "-dir", dir, "-day", "2026-11-29", "-format", "cmds"}, &b); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{"tiers -id ", "pairs determinism-run -n 2 -id ", "e2e -id "} {
		if !strings.Contains(out, want) {
			t.Errorf("no %q in:\n%s", want, out)
		}
	}
	// a quarantined scenario is left out of every command
	b.Reset()
	if err := cmdNightly([]string{"plan", "-file", nightlyJSON, "-dir", dir, "-day", "2026-11-29", "-format", "cmds", "-quarantined", "town.resupply"}, &b); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(b.String(), "town.resupply") {
		t.Fatalf("quarantined scenario planned:\n%s", b.String())
	}
}

func TestNightlyRejectsBadFormat(t *testing.T) {
	var b bytes.Buffer
	if err := cmdNightly([]string{"plan", "-file", nightlyJSON, "-dir", dir, "-format", "xml"}, &b); err == nil {
		t.Fatal("bad format accepted")
	}
}
