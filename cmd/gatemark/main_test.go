package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	okA   = `{"Action":"pass","Package":"m/a"}` + "\n"
	failB = `{"Action":"output","Package":"m/b","Test":"TestB","Output":"boom\n"}` + "\n" + `{"Action":"fail","Package":"m/b","Test":"TestB"}` + "\n" + `{"Action":"fail","Package":"m/b"}` + "\n"
	okB   = `{"Action":"pass","Package":"m/b"}` + "\n"
)

func gate(t *testing.T, in string, args ...string) (int, string) {
	t.Helper()
	var out, errb bytes.Buffer
	code := run(args, strings.NewReader(in), &out, &errb, func() time.Time { return time.Unix(0, 0) })
	return code, out.String() + errb.String()
}

func TestRunExitCodesRecordAndLedger(t *testing.T) {
	dir := t.TempDir()
	rec, led := filepath.Join(dir, "rec.json"), filepath.Join(dir, "flake.jsonl")
	args := func(commit string) []string {
		return []string{"run", "-want", "2", "-commit", commit, "-job", "j1", "-load", "3.1 2.0", "-record", rec, "-ledger", led}
	}

	// Not run: no record of a commit result, no ledger line, exit 3.
	if code, out := gate(t, okA, args("c1")...); code != 3 || !strings.Contains(out, "gate NOT RUN: packages 1 of 2") {
		t.Fatalf("short gate: exit %d, %s", code, out)
	}
	if _, err := os.Stat(led); err == nil {
		t.Fatal("a gate that did not run wrote a ledger line")
	}
	if code, _ := gate(t, "", args("c1")...); code != 3 {
		t.Fatalf("empty gate: exit %d", code)
	}

	for _, c := range []string{"c1", "c2"} {
		code, out := gate(t, okA+failB, args(c)...)
		if code != 1 || !strings.Contains(out, "gate ran: packages 2 of 2") || !strings.Contains(out, "FAIL m/b TestB") {
			t.Fatalf("red gate: exit %d, %s", code, out)
		}
		var r record
		b, _ := os.ReadFile(rec)
		if err := json.Unmarshal(b, &r); err != nil || r.Commit != c || r.Job != "j1" || r.Load != "3.1 2.0" || len(r.Failures) != 1 || len(r.Tail) != 1 || r.Tail[0] != "boom" {
			t.Fatalf("record: %v %s", err, b)
		}
		os.Remove(rec)
		if code, out := gate(t, okA+okB, args(c)...); code != 0 || !strings.HasPrefix(out, "gate ran:") {
			t.Fatalf("green gate: exit %d, %s", code, out)
		}
		if _, err := os.Stat(rec); err == nil {
			t.Fatal("a green gate wrote a failure record")
		}
		code, out = gate(t, "", "quarantine", "-ledger", led)
		if want := c == "c2"; code != 0 || strings.Contains(out, "TestB\tm/b\t") != want {
			t.Fatalf("quarantine after %s: exit %d, %q", c, code, out)
		}
	}
}

func TestRunRefusesAMarkerWithoutACount(t *testing.T) {
	for _, a := range [][]string{{"run", "-commit", "c"}, {"run", "-want", "2"}, {"nope"}, {}, {"quarantine", "-ledger", "/nonexistent/x"}} {
		if code, out := gate(t, okA, a...); code != 2 {
			t.Errorf("%v: exit %d, %s", a, code, out)
		}
	}
	if code, _ := gate(t, "ok  \tm/a\t0.1s\n", "run", "-want", "1", "-commit", "c"); code != 2 {
		t.Errorf("text input: exit %d", code)
	}
}
