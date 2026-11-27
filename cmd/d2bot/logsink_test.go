package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/logx"
)

func TestOpsLogReadsFileTail(t *testing.T) {
	p := filepath.Join(t.TempDir(), "bot.log")
	os.WriteFile(p, []byte("{\"msg\":\"tick a\"}\n{\"msg\":\"status\"}\n{\"msg\":\"tick b\"}\n"), 0o644)
	var out bytes.Buffer
	code, err := opsMain([]string{"log", "--file", p, "--tail", "1", "--grep", "tick"}, &out)
	if err != nil || code != 0 || strings.TrimSpace(out.String()) != `{"msg":"tick b"}` {
		t.Fatalf("code=%d err=%v out=%q", code, err, out.String())
	}
	if code, _ := opsMain([]string{"log", "--file", p + ".none"}, &out); code != 1 {
		t.Fatalf("missing file code = %d", code)
	}
	if code, _ := opsMain([]string{"log", "--grep", "("}, &out); code != 2 {
		t.Fatalf("bad regexp code = %d", code)
	}
}

func TestNewLoggerWritesFile(t *testing.T) {
	rf, err := logx.OpenRotating(filepath.Join(t.TempDir(), "logs", botLogName), 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	log, err := newLogger("info", "main", rf)
	if err != nil {
		t.Fatal(err)
	}
	log.Info("tick", "inv", 3)
	logFlush()
	lines, err := logx.Tail(filepath.Join(filepath.Dir(rf.Name()), botLogName), 5, nil)
	if err != nil || len(lines) != 1 || !strings.Contains(lines[0], `"inv":3`) {
		t.Fatalf("lines=%v err=%v", lines, err)
	}
}
