package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/shadowlog"
)

func write(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "bot.log")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return p
}

const quiet = `{"time":"2026-11-27T01:00:00Z","level":"INFO","msg":"tick"}` + "\n"
const differing = quiet + `{"time":"2026-11-27T01:00:05Z","level":"INFO","msg":"admission shadow","differ":"old-only","area":3,"purpose":"level","old":"too poor","new":"","row":"","need":""}` + "\n"

func TestRunExitCodes(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run(nil, &out, &errw); code != 2 || !strings.Contains(errw.String(), "usage") {
		t.Errorf("no log: code %d, %q", code, errw.String())
	}
	if code := run([]string{"-strict", write(t, quiet)}, &out, &errw); code != 0 {
		t.Errorf("a log without shadow lines passes -strict, code %d", code)
	}
	out.Reset()
	if code := run([]string{"-strict", write(t, differing)}, &out, &errw); code != 1 {
		t.Errorf("a differing line fails -strict, code %d", code)
	}
	if !strings.Contains(out.String(), "admission (bot.admission): review") || !strings.Contains(out.String(), "old-only 1") {
		t.Errorf("text output: %s", out.String())
	}
	if code := run([]string{write(t, differing)}, &out, &errw); code != 0 {
		t.Errorf("without -strict the digest is a report, code %d", code)
	}
	if code := run([]string{filepath.Join(t.TempDir(), "absent.log")}, &out, &errw); code != 2 {
		t.Errorf("a missing file is exit 2, code %d", code)
	}
}

func TestRunJSON(t *testing.T) {
	var out, errw bytes.Buffer
	if code := run([]string{"-json", write(t, differing)}, &out, &errw); code != 0 {
		t.Fatal(code, errw.String())
	}
	var rep shadowlog.Report
	if err := json.Unmarshal(out.Bytes(), &rep); err != nil {
		t.Fatal(err)
	}
	if rep.Schema != shadowlog.Schema || len(rep.Families) != 4 || rep.Families[0].Differ != 1 {
		t.Errorf("report: %+v", rep)
	}
}
