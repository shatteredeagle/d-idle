package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogSummary(t *testing.T) {
	if logSummary("") != "" {
		t.Fatal("empty path must print nothing")
	}
	if !strings.Contains(logSummary(filepath.Join(t.TempDir(), "none.log")), "bot log:") {
		t.Fatal("missing file must be one line")
	}
	p := filepath.Join(t.TempDir(), "bot.log")
	os.WriteFile(p, []byte("time=2026-10-29T10:00:00Z level=INFO msg=\"death diff=hell area=108 clvl=70 mlvl=85 gap=15 killer=Doom class=x boss=false affixes=none dist=3 build=b\"\n"), 0o644)
	if got := logSummary(p); got == "" {
		t.Fatal("no summary for a log with a death line")
	}
}
