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

// The art doll sits in a max-content grid column on wide cards: a percentage-only inline-size resolves to 0 there and
// the doll vanished (sandbox report 2026-11-10). It needs a definite width.
func TestArtDollHasDefiniteWidth(t *testing.T) {
	b, err := webFS.ReadFile("web/components.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(b), "\n") {
		if strings.Contains(line, ".d-doll[data-art] {") && !strings.Contains(line, "inline-size: 320px") {
			t.Errorf("art doll lacks a definite inline-size: %s", line)
		}
	}
}

// The art layer must never sit on the critical path of the first paint: the page loads the inventory at once and
// probes the art beside it (TASK-268: the old probe chain /api/art, panel image, then /api/inventory was three
// serial round trips before anything drew).
func TestInventoryPageDoesNotWaitOnArt(t *testing.T) {
	b, err := webFS.ReadFile("web/inventory.js")
	if err != nil {
		t.Fatal(err)
	}
	js := string(b)
	if strings.Contains(js, "artProbe().then") {
		t.Error("the first load waits for the art probe")
	}
	if !strings.Contains(js, "load(); artProbe();") {
		t.Error("load() must start before and independent of artProbe()")
	}
}
