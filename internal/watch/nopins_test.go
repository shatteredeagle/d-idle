package watch

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The Pin display is gone (docs/inventory-ui.md, docs/gear-policy.md): under the player-only gear policy the bot never
// equips, so a pinned slot says nothing. The API keeps `pins` and /api/inventory/unpin as deprecated, read-only fields.
func TestNoPinDisplay(t *testing.T) {
	if strings.Contains(invPage, "PIN") || strings.Contains(invPage, "lg-pin") {
		t.Error("inventory legend still shows PIN")
	}
	for _, want := range []string{`class="lg-sell"`, `class="lg-keep"`, "durability"} {
		if !strings.Contains(invPage, want) {
			t.Errorf("legend lacks %s", want)
		}
	}
	re := regexp.MustCompile(`(?i)\bpin(ned)?\b|unpin|'pin'|data-badge="pin"`)
	for _, f := range []string{"inventory.js", "inventory-actions.js", "inventory-tip.js", "inventory-tip.css", "inventory-merc.js"} {
		b, err := webFS.ReadFile("web/" + f)
		if err != nil {
			t.Fatal(err)
		}
		if m := re.FindString(string(b)); m != "" {
			t.Errorf("web/%s still mentions %q", f, m)
		}
	}
	// the new DOM goldens (fixtures with pins in the view) show no pin anywhere
	files, _ := filepath.Glob("testdata/*dom*.nopin.txt")
	more, _ := filepath.Glob("testdata/*/*.nopin.txt")
	for _, f := range append(files, more...) {
		b, _ := os.ReadFile(f)
		if re.Match(b) {
			t.Errorf("%s shows a pin", f)
		}
	}
}
