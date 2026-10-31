package main

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

const (
	beginMark = "<!-- knobdump:begin -->"
	endMark   = "<!-- knobdump:end -->"
)

// noReaders drops the readers column (a grep heuristic that moves with every unrelated edit) so the test only
// fails on real drift: a knob added, removed or changed in the code but not in the doc.
func noReaders(s string) string {
	var out []string
	for _, ln := range strings.Split(strings.TrimSpace(s), "\n") {
		if c := strings.Split(ln, "|"); len(c) > 7 && strings.HasPrefix(ln, "|") {
			c = append(c[:6], c[7:]...)
			ln = strings.Join(c, "|")
		}
		out = append(out, strings.TrimRight(ln, " "))
	}
	return strings.Join(out, "\n")
}

func TestKnobsDocMatchesTables(t *testing.T) {
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	doc, err := os.ReadFile("docs/knobs.md")
	if err != nil {
		t.Fatal(err)
	}
	s := string(doc)
	i, j := strings.Index(s, beginMark), strings.Index(s, endMark)
	if i < 0 || j < i {
		t.Fatalf("docs/knobs.md lacks the %s ... %s section", beginMark, endMark)
	}
	var buf bytes.Buffer
	missing := render(&buf)
	if len(missing) > 0 {
		t.Errorf("Tuning fields without a knob entry (add a Knob in internal/env): %v", missing)
	}
	got, want := noReaders(buf.String()), noReaders(s[i+len(beginMark):j])
	if got != want {
		gl, wl := strings.Split(got, "\n"), strings.Split(want, "\n")
		for n := 0; n < len(gl) && n < len(wl); n++ {
			if gl[n] != wl[n] {
				t.Fatalf("docs/knobs.md is stale at line %d of the section:\n code: %s\n doc:  %s\nregenerate: go run ./tools/knobdump", n+1, gl[n], wl[n])
			}
		}
		t.Fatalf("docs/knobs.md section has %d lines, tables have %d; regenerate: go run ./tools/knobdump", len(wl), len(gl))
	}
}
