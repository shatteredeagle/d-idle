package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// run reconciles results with ledger lines. Both runs of cain.scroll.sandbox are blocked before the start (no sandbox
// d2d), so no slot and no game is needed: with a host role they are recorded as BLOCKED lines and the run is only "not
// green" (exit 3); without one the ledger refuses them, the refusals are listed in a file and the run exits 4.
func TestRunExitsNonZeroWhenARunIsNotRecorded(t *testing.T) {
	t.Setenv("D2D_BIN_SANDBOX", "")
	t.Setenv("D2D_BIN", filepath.Join(t.TempDir(), "no-d2d"))
	t.Setenv("PG_LEDGER_CMD", "")
	t.Setenv("D2LOD_BOT_COMMIT", "")
	run := func(role string) (string, string, error) {
		t.Setenv("D2D_HOST_ROLE", role)
		out := t.TempDir()
		var b bytes.Buffer
		err := cmdRun([]string{"-dir", dir, "-id", "cain.scroll.sandbox", "-out", out, "-bot", filepath.Join(out, "no-bot")}, &b)
		return out, b.String(), err
	}
	out, text, err := run("builder")
	if e, ok := err.(exitErr); !ok || e.code != 3 {
		t.Fatalf("recorded blocked runs: %v\n%s", err, text)
	}
	spool, _ := os.ReadFile(filepath.Join(out, ledgerSpool))
	n := strings.Count(string(spool), "\n")
	if n == 0 || !strings.Contains(text, "results "+strconv.Itoa(n)+", ledger "+strconv.Itoa(n)+"\n") {
		t.Fatalf("%d spool lines, output:\n%s", n, text)
	}
	if !strings.Contains(string(spool), `"verdict":"BLOCKED"`) || !strings.Contains(string(spool), `"d2d_sha256":"missing"`) {
		t.Fatalf("spool: %s", spool)
	}
	if _, err := os.Stat(filepath.Join(out, ledgerRefused)); err == nil {
		t.Fatal("a refusal file without a refusal")
	}

	out, text, err = run("") // host role unknown: every line is refused
	if e, ok := err.(exitErr); !ok || e.code != exitUnrecorded || !strings.Contains(e.msg, "NOT RECORDED") {
		t.Fatalf("unrecorded runs: %v\n%s", err, text)
	}
	if !strings.Contains(text, "results "+strconv.Itoa(n)+", ledger 0\n") {
		t.Fatal(text)
	}
	refused, _ := os.ReadFile(filepath.Join(out, ledgerRefused))
	if strings.Count(string(refused), "\n") != n || !strings.Contains(string(refused), "host role unknown") || !strings.Contains(string(refused), "cain.scroll.sandbox") {
		t.Fatalf("refusals: %s", refused)
	}
}
