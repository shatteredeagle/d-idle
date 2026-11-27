package watch

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// web/toast.js dBusy in a stub DOM (node, skipped without it): busy at click, double tap swallowed, done/failed, redraw.
func TestBusyDOM(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node")
	}
	dir := t.TempDir()
	sj, _ := webFS.ReadFile("web/toast.js")
	os.WriteFile(filepath.Join(dir, "toast.cjs"), sj, 0o600)
	for _, f := range []string{"nav_dom.cjs", "busy_test.cjs"} {
		src, err := os.ReadFile(filepath.Join("testdata", f))
		if err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(dir, f), src, 0o600)
	}
	out, err := exec.Command(node, filepath.Join(dir, "busy_test.cjs"), dir).CombinedOutput()
	if err != nil {
		t.Fatalf("node: %v\n%s", err, out)
	}
	var res struct{ Fails []string }
	if err := json.Unmarshal(out[strings.Index(string(out), "{"):], &res); err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
	for _, f := range res.Fails {
		t.Error(f)
	}
}
