package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dctl/fake"
)

func TestPing(t *testing.T) {
	path, cleanup, err := fake.TempSocket()
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	srv, err := fake.New(path, fake.Sample(), fake.Options{})
	if err != nil {
		t.Fatal(err)
	}
	defer srv.Close()
	var out, errw bytes.Buffer
	if rc := run([]string{"-sock", path}, &out, &errw); rc != 0 {
		t.Fatalf("rc %d: %s", rc, errw.String())
	}
	if out.Len() == 0 {
		t.Fatal("empty reply")
	}
}

func TestNoSocket(t *testing.T) {
	var out, errw bytes.Buffer
	rc := run([]string{"-sock", filepath.Join(t.TempDir(), "none.sock"), "ping"}, &out, &errw)
	if rc != 1 || !strings.Contains(errw.String(), "d2dping:") {
		t.Fatalf("rc %d err %q", rc, errw.String())
	}
}
