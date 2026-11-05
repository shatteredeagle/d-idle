package main

import (
	"strings"
	"testing"
)

// soak-gates: XP/h is advisory. A run far below the old 1000 XP/h floor passes whenever it passes without the
// advisory, and carries the advisory line.
func TestXPAdvisoryNeverFails(t *testing.T) {
	ss := synth(40, func(i int, s *Sample) { s.XP = int64(i) }) // ~120 XP/h
	base := DefaultLimits()
	want := Analyze(ss, base).Pass
	lim := base
	lim.AdvisoryXPPerHour = 1000
	r := Analyze(ss, lim)
	if r.Pass != want {
		t.Fatalf("advisory changed the verdict: %v, want %v\n%s", r.Pass, want, r.Text())
	}
	for _, f := range r.Findings {
		if f.Level == "fail" && (f.Code == "xp-rate" || f.Code == "xp-advisory") {
			t.Fatalf("XP/h produced a failure: %+v", f)
		}
	}
	if !strings.Contains(r.XPAdvisory, "advisory") || !strings.Contains(r.XPAdvisory, "below") || !strings.Contains(r.Text(), "xp advisory") {
		t.Fatalf("no advisory line: %q\n%s", r.XPAdvisory, r.Text())
	}
}
