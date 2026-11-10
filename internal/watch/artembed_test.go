package watch

import (
	"io/fs"
	"strings"
	"testing"
)

// Game art is read from the user's install at runtime and never embedded, so no release binary can contain it:
// the embedded web tree has no art directory and no image or DC6 file at all (docs/paperdoll-art-build.md).
func TestEmbeddedWebHasNoArt(t *testing.T) {
	n := 0
	err := fs.WalkDir(webFS, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		n++
		lp := strings.ToLower(p)
		if d.IsDir() && (strings.HasSuffix(lp, "/art") || lp == "art") {
			t.Errorf("embedded art directory: %s", p)
		}
		for _, ext := range []string{".png", ".dc6", ".dcc", ".dt1", ".mpq", ".jpg", ".gif", ".webp"} {
			if strings.HasSuffix(lp, ext) {
				t.Errorf("embedded image or game file: %s", p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n < 10 {
		t.Fatalf("walked only %d entries: is the embed empty?", n)
	}
}
