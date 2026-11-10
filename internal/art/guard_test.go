package art

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The art package must never carry game art itself: no go:embed, no image or DC6 file next to the code
// (docs/paperdoll-art-build.md; tools/ci/check.sh forbids the same across the whole tree).
func TestPackageHoldsNoArt(t *testing.T) {
	err := filepath.WalkDir(".", func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		switch strings.ToLower(filepath.Ext(p)) {
		case ".png", ".dc6", ".dcc", ".dt1", ".mpq", ".pl2", ".dat":
			t.Errorf("art-like file in internal/art: %s", p)
		case ".go":
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if strings.Contains(string(b), "//go:"+"embed") {
				t.Errorf("%s embeds files", p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
