package diffready

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Card sec2: the stored go is 0600 even when a temp file with a wider mode was left behind.
func TestConsentFileIsOwnerOnly(t *testing.T) {
	p := filepath.Join(t.TempDir(), "state", "diffgo.json")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p+".tmp", []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	var c Consent
	if err := c.Persist(p); err != nil {
		t.Fatal(err)
	}
	c.Grant("nightmare", "Hero", time.Unix(1794132000, 0))
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Mode().Perm() != 0o600 {
		t.Errorf("mode %v, want 0600", st.Mode().Perm())
	}
}
