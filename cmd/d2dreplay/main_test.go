package main

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"d2lod-bot/internal/d2dreplay"
)

func TestInfoReadsAFile(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.replay.gz")
	w, err := d2dreplay.Create(p, d2dreplay.Header{Every: 10, Start: &d2dreplay.Recipe{Seed: 3, Class: 1, Name: "Det"}})
	if err != nil {
		t.Fatal(err)
	}
	w.Add(d2dreplay.Entry{Verb: "step 10"})
	w.Add(d2dreplay.Entry{C: 10, H: strings.Repeat("0123456789abcdef", 4), Lv: 1})
	w.Close(&d2dreplay.Entry{Ticks: 10, Verbs: 1})
	if err := info([]string{p}); err != nil {
		t.Fatal(err)
	}
	if err := info([]string{filepath.Join(os.TempDir(), "nope.replay.gz")}); err == nil {
		t.Fatal("missing file must fail")
	}
}

func craft(t *testing.T, lines ...string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "x.replay.gz")
	f, _ := os.Create(p)
	w := gzip.NewWriter(f)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	w.Close()
	f.Close()
	return p
}

func TestInfoAndPlayNeverPanicOnCraftedFiles(t *testing.T) {
	h := `{"replay":1,"hash":"detloop-journal-v1","every":10,"d2d":{"sha256":"abc"},"meta":{}}`
	h2 := `{"replay":1,"hash":"detloop-journal-v1","every":10,"d2d":{},"meta":{}}`
	for name, lines := range map[string][]string{
		"short sha":  {h},
		"blank verb": {h2, `{"v":" "}`},
		"short hash": {h2, `{"c":1,"h":"abc"}`},
	} {
		p := craft(t, lines...)
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: info panicked: %v", name, r)
				}
			}()
			if err := info([]string{p}); err == nil {
				t.Errorf("%s: accepted", name)
			}
		}()
		if err := play(context.Background(), []string{"-d2d-bin", "/nonexistent", p}, false); err == nil {
			t.Errorf("%s: play accepted", name)
		}
	}
	if pfx("abc") != "abc" || pfx(strings.Repeat("a", 20)) != strings.Repeat("a", 12) || firstToken(" ") != "?" {
		t.Fatal("helpers")
	}
}

func TestRenderDir(t *testing.T) {
	base := t.TempDir()
	for _, bad := range []string{"", base + "/../x", "a b", "x%d", "-y", base + "/a\nb", "a\x00b"} {
		if d, err := renderDir(bad); err == nil && bad != "-y" {
			t.Errorf("renderDir(%q) = %q", bad, d)
		}
	}
	// a leading dash becomes an absolute path, so ffmpeg cannot read it as an option
	d, err := renderDir(filepath.Join(base, "o", "frames"))
	if err != nil || !filepath.IsAbs(d) {
		t.Fatal(d, err)
	}
	if st, _ := os.Stat(d); st == nil || st.Mode().Perm() != 0o700 {
		t.Fatalf("mode %v", st)
	}
	f := filepath.Join(base, "file")
	os.WriteFile(f, nil, 0o600)
	if _, err := renderDir(f); err == nil {
		t.Error("a regular file is not a render dir")
	}
	ln := filepath.Join(base, "link")
	os.Symlink(base, ln)
	if _, err := renderDir(ln); err == nil {
		t.Error("a symlink is not a render dir")
	}
	t.Chdir(base)
	if d, err := renderDir("-y"); err != nil || !filepath.IsAbs(d) {
		t.Errorf("-y: %q %v", d, err)
	}
}
