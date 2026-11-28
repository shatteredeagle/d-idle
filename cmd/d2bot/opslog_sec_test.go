package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Card sec2 (review 4 finding 7): `ops log` prints no credential and no control character, keeps what an operator
// reads a log for, and takes only bot.log.N as a rotation.
func TestOpsLogRedactsAndEscapes(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "bot.log")
	tok := "aB3dE6gH9jK2mN5pQ8sT" + "1vW4yZ7_-xY0"
	ip := "192.168." + "1.20"
	secrets := []string{tok, "hunter2", "correct horse", "abc123DEF456ghi789", "s3cr3t", "\x1b", "\x9b", "\x07"}
	log := strings.Join([]string{
		`{"msg":"watch","url":"http://box:8891/watch?t=` + tok + `","peer":"` + ip + `:5123"}`,
		`level=INFO msg=login passcode=hunter2 area="Blood Moor" kills=62`,
		`{"msg":"auth","passphrase":"correct horse battery","budget.token_left":2,"user_go":true}`,
		`hdr Authoriz` + `ation: Bea` + `rer abc123DEF456ghi789 path=/data/instances/main`,
		`fetch https://alice:s3cr3t@` + `example.org/x commit 0123456789abcdef0123456789abcdef01234567`,
		"item \x1b[2J\x1b]0;owned\x07 named \xc2\x9b31m by hero",
		`unlabelled ` + tok + ` in text`,
	}, "\n") + "\n"
	os.WriteFile(p, []byte(log), 0o600)
	os.WriteFile(p+".1", []byte("rotated one\n"), 0o600)
	os.WriteFile(p+".bak", []byte("not a rotation "+tok+"\n"), 0o600)
	os.WriteFile(p+".tmp", []byte("not a rotation either\n"), 0o600)
	var out bytes.Buffer
	if code, err := opsMain([]string{"log", "--file", p, "--tail", "0"}, &out); err != nil || code != 0 {
		t.Fatalf("code=%d err=%v", code, err)
	}
	got := out.String()
	for _, s := range secrets {
		if strings.Contains(got, s) {
			t.Errorf("output still contains %q:\n%s", s, got)
		}
	}
	for _, keep := range []string{"rotated one", ip + ":5123", `area="Blood Moor" kills=62`, `"budget.token_left":2,"user_go":true`, "path=/data/instances/main",
		"commit 0123456789abcdef0123456789abcdef01234567", `\x1b[2J`, "by hero", "http://box:8891/watch?t=<token>"} {
		if !strings.Contains(got, keep) {
			t.Errorf("output lost %q:\n%s", keep, got)
		}
	}
	if strings.Contains(got, "not a rotation") {
		t.Errorf("a sibling file was read as a rotation:\n%s", got)
	}
	if n := strings.Count(got, "\n"); n != 8 {
		t.Errorf("%d lines, want 8 (one per log line):\n%s", n, got)
	}
	// a glob character in --file is a plain character, not a pattern
	out.Reset()
	if code, _ := opsMain([]string{"log", "--file", filepath.Join(dir, "bot.lo?")}, &out); code != 1 || out.Len() != 0 {
		t.Errorf("pattern in --file: code %d, out %q", code, out.String())
	}
}
