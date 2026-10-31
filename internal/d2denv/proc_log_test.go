package d2denv

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRotateLogKeepsPreviousStart(t *testing.T) {
	dir := t.TempDir()
	p := &Proc{LogPath: filepath.Join(dir, "d2d.out"), Bin: "x"}
	p.rotateLog()
	p.Log.WriteString("first instance\n" + strings.Repeat("x", 2100))
	p.rotateLog()
	p.Log.WriteString("second\n")
	cur, _ := os.ReadFile(p.LogPath)
	if string(cur) != "second\n" {
		t.Fatalf("current log %q", cur)
	}
	old, _ := filepath.Glob(p.LogPath + ".*")
	if len(old) != 1 {
		t.Fatalf("rotated files %v", old)
	}
	if b, _ := os.ReadFile(old[0]); !strings.HasPrefix(string(b), "first instance\n") {
		t.Fatalf("rotated content %q", b)
	}
}

func TestDiagnosisNamesExitAndMissingSocket(t *testing.T) {
	dir := t.TempDir()
	p := &Proc{LogPath: filepath.Join(dir, "d2d.out"), Bin: "x", Sock: filepath.Join(dir, "nope.sock")}
	p.rotateLog()
	p.Log.WriteString("alive: 470 frames/5s\nSIGSEGV in town.cpp\n")
	p.done = make(chan struct{})
	close(p.done)
	p.exit = "signal: segmentation fault"
	d := p.Diagnosis(errors.New("dial: connection refused"))
	for _, want := range []string{"exited: signal: segmentation fault", "socket file is gone", "SIGSEGV in town.cpp", "connection refused"} {
		if !strings.Contains(d, want) {
			t.Errorf("diagnosis lacks %q: %s", want, d)
		}
	}
}
