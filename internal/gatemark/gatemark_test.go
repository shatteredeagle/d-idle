package gatemark

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func ev(action, pkg, test, out string) string {
	return fmt.Sprintf(`{"Action":%q,"Package":%q,"Test":%q,"Output":%q}`, action, pkg, test, out)
}

// stream is a gate over four packages: one passes, one is cached, one has no tests, one fails in a subtest.
func stream() string {
	l := []string{
		ev("run", "m/a", "TestA", ""), ev("pass", "m/a", "TestA", ""), ev("output", "m/a", "", "ok  \tm/a\t0.1s\n"), ev("pass", "m/a", "", ""),
		ev("output", "m/b", "", "ok  \tm/b\t(cached)\n"), ev("pass", "m/b", "", ""),
		ev("output", "m/c", "", "?   \tm/c\t[no test files]\n"), ev("skip", "m/c", "", ""),
		ev("run", "m/d", "TestD", ""), ev("run", "m/d", "TestD/two_builds", ""),
	}
	for i := 0; i < 2*TailLines; i++ {
		l = append(l, ev("output", "m/d", "TestD/two_builds", fmt.Sprintf("line %d\n", i)))
	}
	l = append(l, ev("fail", "m/d", "TestD/two_builds", ""), ev("fail", "m/d", "TestD", ""), ev("skip", "m/d", "TestSkipped", ""),
		ev("output", "m/d", "", "FAIL\tm/d\t0.2s\n"), ev("fail", "m/d", "", ""))
	return strings.Join(l, "\n") + "\n"
}

func TestParseCountsAndFailures(t *testing.T) {
	s, err := Parse(strings.NewReader(stream()), 4)
	if err != nil {
		t.Fatal(err)
	}
	if s.OK != 2 || s.NoTests != 1 || s.Failed != 1 || s.Cached != 1 || s.Tests != 4 {
		t.Fatalf("counts: %+v", s)
	}
	if len(s.Failures) != 1 || s.Failures[0] != (Failure{"m/d", "TestD/two_builds"}) {
		t.Fatalf("failures: %+v (the failing subtest is the name, not its parent)", s.Failures)
	}
	if len(s.Tail) != TailLines || s.Tail[len(s.Tail)-1] != "FAIL\tm/d\t0.2s" || s.Tail[0] != fmt.Sprintf("line %d", TailLines+1) {
		t.Fatalf("tail: %d lines, first %q last %q", len(s.Tail), s.Tail[0], s.Tail[len(s.Tail)-1])
	}
	if !s.Ran() || s.Green() || !strings.HasPrefix(s.Marker(), "gate ran: packages 4 of 4 (ok 2, no test files 1, failed 1, cached 1), tests 4") {
		t.Fatalf("marker %q ran %v green %v", s.Marker(), s.Ran(), s.Green())
	}
}

// A gate that answered from the cache in a second is a gate that ran; an empty or short one is not, whatever it took.
func TestRanIsACountNotATime(t *testing.T) {
	cached := ev("output", "m/a", "", "ok  \tm/a\t(cached)\n") + "\n" + ev("pass", "m/a", "", "") + "\n"
	for _, c := range []struct {
		name, in string
		want     int
		ran      bool
	}{
		{"all cached", cached, 1, true},
		{"empty stream (wrong directory)", "", 1, false},
		{"fewer packages than asked", cached, 2, false},
		{"more packages than asked", cached + ev("pass", "m/b", "", "") + "\n" + ev("pass", "m/c", "", "") + "\n", 2, false},
		{"only test events, no package result (killed)", ev("pass", "m/a", "TestA", "") + "\n", 1, false},
	} {
		s, err := Parse(strings.NewReader(c.in), c.want)
		if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if s.Ran() != c.ran || s.Green() != c.ran || strings.HasPrefix(s.Marker(), "gate ran:") != c.ran {
			t.Errorf("%s: ran %v, marker %q, want ran %v", c.name, s.Ran(), s.Marker(), c.ran)
		}
	}
}

func TestParseRejectsTextOutput(t *testing.T) {
	if _, err := Parse(strings.NewReader("ok  \tm/a\t0.1s\n"), 1); err == nil {
		t.Fatal("plain go test output must be an error: nothing in it can be counted as an event")
	}
}

func TestPackageFailureWithoutTestIsNamed(t *testing.T) {
	s, err := Parse(strings.NewReader(ev("output", "m/a", "", "m/a/x.go:3: undefined: y\n")+"\n"+ev("fail", "m/a", "", "")+"\n"), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Failures) != 1 || s.Failures[0] != (Failure{Package: "m/a"}) || len(s.Tail) != 1 {
		t.Fatalf("%+v", s)
	}
}

func run(t *testing.T, commit string, failing ...Failure) []Line {
	t.Helper()
	s := Summary{Want: 1, OK: 1}
	if len(failing) > 0 {
		s = Summary{Want: 1, Failed: 1, Failures: failing}
	}
	return s.Lines("2026-11-28T00:00:00Z", commit, "job")
}

func TestFlakesNeedTwoNonReproducingFailures(t *testing.T) {
	flaky := Failure{"m/d", "TestD/two_builds"}
	real := Failure{"m/e", "TestE"}
	var lines []Line
	add := func(l []Line) { lines = append(lines, l...) }

	add(run(t, "c1", flaky, real))
	add(run(t, "c1", real)) // the rerun on c1: TestD passed, the gate is still red
	if f := Flakes(lines); len(f) != 0 {
		t.Fatalf("a failure on a commit that never went green is not a flake: %+v", f)
	}
	add(run(t, "c2", flaky))
	add(run(t, "c2")) // green unchanged: one non-reproducing failure
	if f := Flakes(lines); len(f) != 0 {
		t.Fatalf("one non-reproducing failure does not quarantine: %+v", f)
	}
	add(run(t, "c2", flaky)) // the same commit again is still one commit
	if f := Flakes(lines); len(f) != 0 {
		t.Fatalf("two failures on one commit count once: %+v", f)
	}
	add(run(t, "c3", flaky, Failure{Package: "m/f"}))
	add(run(t, "c3"))
	add(run(t, "c2", Failure{Package: "m/f"}))
	f := Flakes(lines)
	if len(f) != 1 || f[0].Package != "m/d" || f[0].Test != "TestD/two_builds" || strings.Join(f[0].Commits, ",") != "c2,c3" {
		t.Fatalf("flakes: %+v (the package failure without a test and the real failure stay out)", f)
	}
	if row := f[0].QuarantineRow(); !strings.HasPrefix(row, "TestD\tm/d\tunassigned\t2 non-reproducing") {
		t.Fatalf("row %q", row)
	}

	var buf bytes.Buffer
	if err := Append(&buf, lines); err != nil {
		t.Fatal(err)
	}
	back, err := ReadLedger(&buf)
	if err != nil || len(back) != len(lines) || len(Flakes(back)) != 1 {
		t.Fatalf("round trip: %d of %d lines, err %v", len(back), len(lines), err)
	}
	if _, err := ReadLedger(strings.NewReader("{\"at\":\"x\"}\n")); err == nil {
		t.Fatal("a ledger line without a commit must be an error")
	}
}
