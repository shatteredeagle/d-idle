package watch

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// web/onboarding.js in a stub DOM (node, skipped without it; card onboardtour).
func TestOnboardingTourDOM(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("no node")
	}
	dir := t.TempDir()
	for src, dst := range map[string]string{"web/onboarding.js": "onboarding.cjs", "web/sheet.js": "sheet.cjs", "web/onboarding.json": "onboarding.json"} {
		os.WriteFile(filepath.Join(dir, dst), mustReadFS(t, src), 0o600)
	}
	for _, f := range []string{"nav_dom.cjs", "onboarding_tour_test.cjs"} {
		os.WriteFile(filepath.Join(dir, f), mustRead(t, "testdata/"+f), 0o600)
	}
	out, err := exec.Command(node, filepath.Join(dir, "onboarding_tour_test.cjs"), dir).CombinedOutput()
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

// The tour texts are plain, and the page loads the tour script and its button.
func TestOnboardingTourTexts(t *testing.T) {
	var tx struct {
		Steps []struct{ ID, Title, Body string }
		Help  struct{ Lines []string }
	}
	if err := json.Unmarshal(mustReadFS(t, "web/onboarding.json"), &tx); err != nil {
		t.Fatal(err)
	}
	if len(tx.Steps) != 5 || len(tx.Help.Lines) == 0 {
		t.Fatalf("steps %d, help lines %d", len(tx.Steps), len(tx.Help.Lines))
	}
	for _, s := range tx.Steps {
		if s.Title == "" || s.Body == "" || strings.ContainsAny(s.Title+s.Body, "<>") {
			t.Errorf("step %s: empty or markup", s.ID)
		}
	}
	page := string(mustReadFS(t, "web/watch.html"))
	for _, w := range []string{"/watch/onboarding.js", "/watch/onboarding.css", `id="btn-tour"`} {
		if !strings.Contains(page, w) {
			t.Errorf("watch.html lacks %s", w)
		}
	}
}
