package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/setup"
)

func TestBundle(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "config.json"), []byte(`{"ui":{"addr":":1","authToken":"SEKRET1"},"x":[{"password":"SEKRET2","ok":1}],"apiKey":"SEKRET3"}`), 0o644)
	os.WriteFile(filepath.Join(dir, "ui.token"), []byte("SEKRET4"), 0o600)
	logs := filepath.Join(dir, "instances", "i1", "logs")
	os.MkdirAll(logs, 0o755)
	var sb strings.Builder
	for i := 0; i < 700; i++ {
		fmt.Fprintf(&sb, "line %d\n", i)
	}
	os.WriteFile(filepath.Join(logs, "d2bot.log"), []byte(sb.String()), 0o644)
	out := filepath.Join(t.TempDir(), "b.tgz")
	var o, e bytes.Buffer
	run([]string{"--json", "--quick", "--pre-install", "--data", dir, "--bundle", out}, &o, &e, setup.NewEnv())

	f, err := os.Open(out)
	if err != nil {
		t.Fatal(err)
	}
	gz, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	tr := tar.NewReader(gz)
	got := map[string]string{}
	for {
		h, err := tr.Next()
		if err != nil {
			break
		}
		b, _ := io.ReadAll(tr)
		got[h.Name] = string(b)
	}
	for _, n := range []string{"doctor.json", "version.txt", "config.redacted.json", "logs/instances/i1/logs/d2bot.log"} {
		if got[n] == "" {
			t.Fatalf("missing %s in %v", n, got)
		}
	}
	for n, b := range got {
		if strings.Contains(b, "SEKRET") {
			t.Fatalf("%s leaks a secret: %s", n, b)
		}
	}
	if !strings.Contains(got["config.redacted.json"], `"ok": 1`) {
		t.Fatal("redaction dropped too much")
	}
	l := got["logs/instances/i1/logs/d2bot.log"]
	if strings.Count(l, "\n") != 500 || !strings.HasPrefix(l, "line 200\n") || !strings.HasSuffix(l, "line 699\n") {
		t.Fatalf("tail wrong: %q...", l[:20])
	}
}
