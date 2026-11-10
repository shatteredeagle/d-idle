package main

import (
	"io"
	"strings"
	"testing"
)

func TestArtMainErrors(t *testing.T) {
	t.Setenv("D2BOT_GAMEDIR", "")
	if err := artMain(nil, io.Discard); err == nil || !strings.Contains(err.Error(), "usage") {
		t.Errorf("no args: %v", err)
	}
	if err := artMain([]string{"extract", "--data", t.TempDir()}, io.Discard); err == nil || !strings.Contains(err.Error(), "drawn cells") {
		t.Errorf("no install: %v", err)
	}
	// an install dir without MPQs is reported clearly, nothing is written to it
	if err := artMain([]string{"extract", "--data", t.TempDir(), "--gamedir", t.TempDir()}, io.Discard); err == nil || !strings.Contains(err.Error(), "drawn cells") {
		t.Errorf("empty install: %v", err)
	}
}
