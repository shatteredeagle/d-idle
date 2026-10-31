package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func classes(hits []Hit) map[string]int {
	m := map[string]int{}
	for _, h := range hits {
		m[h.Class]++
	}
	return m
}

func TestRules(t *testing.T) {
	cases := []struct {
		line  string
		class string // "" = clean
	}{
		{"ssh jesse@orion.lan.birbhub.com", "domain"},
		{"the build host Orion", "machine"},
		{"orion_box and Jade_x", "machine"},
		{"herdr, Forgejo, Unraid", "machine"},
		{"cat /mnt/Omlette/d2lod", "machine"},
		{"Venus is a PC", "machine"},
		{"the Tower is mine", "machine"},
		{"Forgotten Tower, Tower Cellar, Tamoe Highland", ""},
		{"stowerage, jadeite, Onyxia, venusian", ""},
		{"asked Desmond", "person"},
		{"http://192.168.1.20:8897/", "ip"},
		{"10.0.0.5", "ip"},
		{"172.20.1.1", "ip"},
		{"172.32.1.1 and 8.8.8.8", ""},
		{"version 1.10.0.0.1", ""},
		{"/home/alice/code", "homepath"},
		{"/home/<user>/code and $HOME", ""},
		{"/Users/bob", "homepath"},
		{"mail me@example.org", "email"},
		{"Authorization: Bearer abcdefghijklmnop", "secret"},
		{"id 123e4567-e89b-12d3-a456-426614174000", "secret"},
		{"tok Zx9Qw3Er7Ty1Ui5Op2As8Df4Gh6Jk0LzXc", "secret"},
		{"CamelCaseIdentifierWithoutAnyDigitsAtAll", ""},
	}
	for _, c := range cases {
		got := classes(scanText("f", c.line, nil))
		if c.class == "" && len(got) != 0 || c.class != "" && got[c.class] == 0 {
			t.Errorf("%q: got %v, want %q", c.line, got, c.class)
		}
	}
}

func TestSecretNotEchoed(t *testing.T) {
	for _, h := range scanText("f", "Authorization: Bearer abcdefghijklmnop", nil) {
		if h.Text != "" {
			t.Errorf("secret echoed: %+v", h)
		}
	}
}

func TestAllowStrings(t *testing.T) {
	a := parseAllow("# c\nstr noreply@anthropic.com\nstr shatteredeagle\npath docs/x/**\n")
	if n := len(scanText("f", "Co-Authored-By: C <noreply@anthropic.com>", a.Strs)); n != 0 {
		t.Errorf("allowed email flagged: %d", n)
	}
	if n := len(scanText("f", "other@anthropic.com", a.Strs)); n != 1 {
		t.Errorf("other email hits = %d", n)
	}
	if !a.skipPath("docs/x/y/z.md") || a.skipPath("docs/y.md") {
		t.Error("path allow glob")
	}
}

func TestGlob(t *testing.T) {
	for g, tc := range map[string]map[string]bool{
		"docs/**":           {"docs/a.md": true, "docs/a/b.md": true, "xdocs/a": false},
		"**/*.sh":           {"a.sh": true, "tools/a.sh": true, "tools/a.go": false},
		"tools/*.sh":        {"tools/a.sh": true, "tools/x/a.sh": false},
		"docs/boardhost.md": {"docs/boardhost.md": true, "docs/boardhostXmd": false},
	} {
		for p, want := range tc {
			if globRegexp(g).MatchString(p) != want {
				t.Errorf("glob %q on %q want %v", g, p, want)
			}
		}
	}
}

func TestRunModesAndManifest(t *testing.T) {
	root := t.TempDir()
	w := func(p, s string) {
		os.MkdirAll(filepath.Dir(filepath.Join(root, p)), 0o755)
		os.WriteFile(filepath.Join(root, p), []byte(s), 0o644)
	}
	w("a.md", "see orion.lan.birbhub.com\n")
	w("priv/b.md", "ssh 192.168.0.1\n")
	w("skip/c.md", "birbhub\n")
	w("clean.md", "nothing here\n")
	w("bin.dat", "birbhub\x00")
	man := &Manifest{Rules: []*ManifestRule{{Glob: "priv/**", Class: Priv}, {Glob: "a.md", Class: Scrub}}}
	for _, r := range man.Rules {
		r.rx = globRegexp(r.Glob)
	}
	rep := run(root, []string{"a.md", "priv/b.md", "skip/c.md", "clean.md", "bin.dat"}, parseAllow("path skip/**"), man, "warn")
	if rep.SkippedPrivate != 1 || rep.SkippedAllowed != 1 || rep.FilesScanned != 2 {
		t.Errorf("counts: %+v", rep)
	}
	if rep.Counts["domain"].Hits != 1 || rep.Counts["machine"].Hits != 1 || rep.ScrubStatus[Scrub] != 1 {
		t.Errorf("class counts: %+v", rep)
	}
	var sb strings.Builder
	rep.print(&sb, true)
	if !strings.Contains(sb.String(), "a.md:1: domain") {
		t.Errorf("output: %s", sb.String())
	}
}
