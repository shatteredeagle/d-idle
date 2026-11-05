package decide

import (
	"bytes"
	"errors"
	"math/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPersistRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cat := mustCat(t)
	clk := &ManualClock{}
	s, err := Open(dir, "alice", cat, clk)
	if err != nil {
		t.Fatal(err)
	}
	o := baseObs(cat, "sorceress")
	ctx, _ := cat.Build("t1.restock", o, 1, nil)
	for i := 0; i < 2; i++ {
		d, _ := s.Ask(ctx)
		if err := s.Answer(d.Thought.ID, "yes"); err != nil {
			t.Fatal(err)
		}
	}
	open, _ := s.Ask(mustBuild(t, cat, "t0.retreat.lethal", o))
	s.SetQuality("altar", 4)
	if err := s.Rebirth("hero2"); err != nil {
		t.Fatal(err)
	}
	_ = open
	want, _ := s.RulesJSON()

	s2, err := Open(dir, "alice", cat, &ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	got, _ := s2.RulesJSON()
	if !bytes.Equal(want, got) || s2.Quality("altar") != 4 || len(s2.Log()) != len(s.Log()) {
		t.Fatal("reload differs")
	}
	if d, _ := s2.Ask(ctx); d.Source != "rule" {
		t.Fatalf("rule lost across reload: %+v", d)
	}
	// profiles are separate files
	s3, _ := Open(dir, "bob", cat, &ManualClock{})
	if len(s3.Rules()) != 0 {
		t.Fatal("profiles share state")
	}
	if _, err := os.Stat(Path(dir, "alice") + ".bak"); err != nil {
		t.Fatalf("no .bak after a second save: %v", err)
	}
	if _, err := os.Stat(Path(dir, "alice") + ".tmp"); err == nil {
		t.Fatal("tmp file left behind")
	}
}

func mustBuild(t *testing.T, cat *Catalogue, point string, o Obs) Context {
	t.Helper()
	c, err := cat.Build(point, o, 1, nil)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestPersistRefusesBadFiles(t *testing.T) {
	cat := mustCat(t)
	good := `{"version":1,"profile":"p","hero":"h","log":[]}`
	for name, tc := range map[string]struct {
		body string
		want error
	}{
		"future":   {`{"version":3,"profile":"p","log":[]}`, ErrFutureVersion},
		"garbage":  {`not json`, ErrCorrupt},
		"no ver":   {`{"profile":"p","log":[]}`, ErrCorrupt},
		"truncate": {good[:20], ErrCorrupt},
		"other":    {`{"version":1,"profile":"q","log":[]}`, ErrCorrupt},
		"bad log":  {`{"version":1,"profile":"p","log":[{"seq":1,"op":"answer","thought":4}]}`, ErrCorrupt},
		"bad op":   {`{"version":1,"profile":"p","log":[{"seq":1,"op":"explode"}]}`, ErrCorrupt},
	} {
		dir := t.TempDir()
		if err := os.WriteFile(Path(dir, "p"), []byte(tc.body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(dir, "p", cat, &ManualClock{}); !errors.Is(err, tc.want) {
			t.Errorf("%s: err = %v, want %v", name, err, tc.want)
		}
		if b, _ := os.ReadFile(Path(dir, "p")); string(b) != tc.body {
			t.Errorf("%s: the refused file was modified", name)
		}
	}
	if _, err := os.ReadFile(filepath.Join(t.TempDir(), "none")); err == nil {
		t.Fatal("sanity")
	}
	for _, bad := range []string{"", "../x", "A b", strings.Repeat("a", 65)} {
		if _, err := Open(t.TempDir(), bad, cat, &ManualClock{}); !errors.Is(err, ErrBadProfile) {
			t.Errorf("profile %q: %v", bad, err)
		}
	}
	if _, err := Open(t.TempDir(), "p", nil, nil); err == nil {
		t.Error("nil catalogue")
	}
}

func TestPersistedProfileSurvivesScript(t *testing.T) {
	dir := t.TempDir()
	cat := mustCat(t)
	clk := &ManualClock{}
	s, _ := Open(dir, "run", cat, clk)
	prof := holdout(77)
	steps := genScript(rand.New(rand.NewSource(3)), cat, "paladin", prof, 300)
	if _, err := RunScript(s, clk, 1, nil, steps); err != nil {
		t.Fatal(err)
	}
	want, _ := s.RulesJSON()
	s2, err := Open(dir, "run", cat, &ManualClock{})
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := s2.RulesJSON(); !bytes.Equal(got, want) {
		t.Fatal("reload after a long script differs")
	}
}
