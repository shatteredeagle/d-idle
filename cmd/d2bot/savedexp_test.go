package main

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestSavedExpReaderCachesByMtime(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "Soaky.d2s")
	reads, val := 0, uint32(100)
	var fail error
	f := savedExpReader(func() string { return dir }, func() string { return "Soaky" }, func(string) (int, uint32, error) {
		reads++
		return 5, val, fail
	})
	if _, ok := f(); ok {
		t.Fatal("missing file must be unknown")
	}
	os.WriteFile(p, []byte("a"), 0o644)
	if v, ok := f(); !ok || v != 100 || reads != 1 {
		t.Fatalf("first read: %d %v %d", v, ok, reads)
	}
	f()
	if reads != 1 {
		t.Fatalf("unchanged file re-read: %d", reads)
	}
	val = 250
	os.WriteFile(p, []byte("bb"), 0o644)
	if v, _ := f(); v != 250 || reads != 2 {
		t.Fatalf("changed file: %d %d", v, reads)
	}
	fail = errors.New("torn")
	os.WriteFile(p, []byte("ccc"), 0o644)
	if v, ok := f(); !ok || v != 250 {
		t.Fatalf("torn read must keep last good: %d %v", v, ok)
	}
}
