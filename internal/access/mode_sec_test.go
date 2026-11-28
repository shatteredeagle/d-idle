package access

import (
	"os"
	"path/filepath"
	"testing"
)

// Card sec2: the record is 0600 even when a temp file with a wider mode (or a link to another file) was left behind.
func TestSaveIgnoresLeftoverTempFile(t *testing.T) {
	dir := t.TempDir()
	p, victim := filepath.Join(dir, "access.json"), filepath.Join(dir, "victim")
	if err := os.WriteFile(victim, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, plant := range []func() error{
		func() error { return os.WriteFile(p+".tmp", []byte("old"), 0o644) },
		func() error { return os.Symlink(victim, p+".tmp") },
	} {
		if err := plant(); err != nil {
			t.Fatal(err)
		}
		r, _ := NewPasscode("secret one", t0)
		if err := NewStore(p).Save(r); err != nil {
			t.Fatal(err)
		}
		st, err := os.Lstat(p)
		if err != nil || !st.Mode().IsRegular() || st.Mode().Perm() != 0o600 {
			t.Errorf("record: %v %v; want a regular 0600 file", st.Mode(), err)
		}
		if b, _ := os.ReadFile(victim); string(b) != "keep" {
			t.Errorf("the linked file was overwritten: %q", b)
		}
	}
}
