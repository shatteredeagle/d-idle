package d2dcontainer_test

import (
	"os"
	"path/filepath"
	"testing"

	"d2lod-bot/internal/selfupdate"
)

// Both release keys are pinned: current.pub (GitHub releases) and beta.pub (the friends' feed). Each must parse and
// they must differ, so a copy mistake cannot leave the image with one key.
func TestPinnedKeysBothPresent(t *testing.T) {
	seen := map[string]string{}
	for _, n := range []string{"current.pub", "beta.pub"} {
		b, err := os.ReadFile(filepath.Join("pubkeys", n))
		if err != nil {
			t.Fatal(err)
		}
		k, err := selfupdate.DecodePublicKey(b)
		if err != nil {
			t.Fatalf("%s: %v", n, err)
		}
		if prev, dup := seen[string(k)]; dup {
			t.Fatalf("%s duplicates %s", n, prev)
		}
		seen[string(k)] = n
	}
}
