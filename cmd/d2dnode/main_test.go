package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"d2lod-bot/internal/d2dmesh"
)

func TestFlagsAndToken(t *testing.T) {
	dir := t.TempDir()
	tok := filepath.Join(dir, "token")
	if err := os.WriteFile(tok, []byte("short\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readToken(tok); err == nil {
		t.Fatal("short token accepted")
	}
	if err := os.WriteFile(tok, []byte("a-long-enough-test-token\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if v, err := readToken(tok); err != nil || v != "a-long-enough-test-token" {
		t.Fatalf("token %q %v", v, err)
	}
	if _, err := readToken(filepath.Join(dir, "missing")); err == nil || !strings.Contains(err.Error(), "join token") {
		t.Fatalf("missing token: %v", err)
	}
	if got := splitList(" a, b,,c "); strings.Join(got, "|") != "a|b|c" {
		t.Fatalf("splitList %v", got)
	}
}

// The script runner passes only recipe, outdir and validated key=value args; the pid is recorded.
func TestScriptRunnerRecordsPid(t *testing.T) {
	dir := t.TempDir()
	script := filepath.Join(dir, "run.sh")
	if err := os.WriteFile(script, []byte("#!/bin/sh\necho \"$@\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r := &scriptRunner{script: script, outRoot: filepath.Join(dir, "out")}
	j := &d2dmesh.JobRecord{ID: "sleep-1-abcd", Recipe: "sleep", Args: map[string]string{"seconds": "0"}}
	res, err := r.Run(context.Background(), j, 2)
	if err != nil || res.Exit != 0 || res.Pid == 0 {
		t.Fatalf("res %+v err %v", res, err)
	}
	log, _ := os.ReadFile(filepath.Join(res.OutDir, "log"))
	if !strings.HasPrefix(string(log), "sleep "+res.OutDir+" seconds=0") {
		t.Fatalf("log %q", log)
	}
	if _, err := os.Stat(filepath.Join(res.OutDir, "pid")); err != nil {
		t.Fatal("pid not recorded")
	}
	if filepath.Base(filepath.Dir(res.OutDir)) != "sleep-1-abcd" || filepath.Base(res.OutDir) != "2" {
		t.Fatalf("out dir not keyed by job id and attempt: %s", res.OutDir)
	}
	// cancel stops the child by its pid
	slow := filepath.Join(dir, "slow.sh")
	if err := os.WriteFile(slow, []byte("#!/bin/sh\nsleep 30\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	r.script = slow
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := r.Run(ctx, j, 3); err == nil || time.Since(start) > 5*time.Second {
		t.Fatalf("cancel did not stop the run: err=%v after %s", err, time.Since(start))
	}
}

// Without -runner and without -dry-run a node never claims (guard says NO).
func TestDefaultNodeRefusesWork(t *testing.T) {
	cfg := d2dmesh.Config{Name: "mgr", Token: "a-long-enough-test-token", Interval: 30 * time.Millisecond, LeaseTTL: 300 * time.Millisecond,
		Runner: refuseRunner{}, Guard: func() bool { return false }, Logf: func(string, ...any) {}}
	n, err := d2dmesh.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := n.Start(); err != nil {
		t.Fatal(err)
	}
	defer n.Stop()
	id, err := n.Submit("sleep", map[string]string{"seconds": "0"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(200 * time.Millisecond)
	if j := n.Snapshot().Jobs[id]; j.State != d2dmesh.JobQueued {
		t.Fatalf("manager-only node claimed: %+v", j)
	}
}
