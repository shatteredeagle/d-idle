// Package gatemark reads the `go test -json` stream of a merge gate and answers three things the gate could not say
// (design docs/design/fable-review-2026-11-21/proving-ground-gate.md 4.2 and 4.7, card mgonejob): did the tests run
// (a positive marker instead of a timing guess), which tests failed with the tail of their output, and which tests
// failed twice on commits that later passed unchanged (the flake ledger).
package gatemark

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
)

// TailLines is how many output lines of the failing packages a record keeps (design 4.7).
const TailLines = 60

// QuarantineAfter is how many non-reproducing failures put a test on the quarantine list (design 4.7).
const QuarantineAfter = 2

type event struct {
	Action  string
	Package string
	Test    string
	Output  string
}

// Failure names one failing test; Test is empty when the package failed without a failing test (build, init, timeout).
type Failure struct {
	Package string `json:"pkg"`
	Test    string `json:"test,omitempty"`
}

// Summary is what one gate run did.
type Summary struct {
	Want     int       `json:"want"` // packages the gate was asked to run; 0 when not given
	OK       int       `json:"ok"`
	NoTests  int       `json:"no_tests"`
	Failed   int       `json:"failed"`
	Cached   int       `json:"cached"`
	Tests    int       `json:"tests"` // test and subtest results (pass, fail or skip)
	Failures []Failure `json:"failures,omitempty"`
	Tail     []string  `json:"tail,omitempty"`
}

// Parse reads a `go test -json` stream. want is the package count of `go list` for the same pattern (0: unknown).
// A line that is not a test event is an error: text output means the gate ran without -json and nothing can be counted.
func Parse(r io.Reader, want int) (Summary, error) {
	s := Summary{Want: want}
	out := map[string][]string{}
	failedTests := map[string][]string{}
	var failedPkgs []string
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for n := 1; sc.Scan(); n++ {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		var e event
		if err := json.Unmarshal([]byte(line), &e); err != nil || e.Action == "" {
			return s, fmt.Errorf("gatemark: line %d is not a go test -json event: %.80q", n, line)
		}
		if e.Package == "" {
			continue
		}
		switch e.Action {
		case "output":
			if e.Test == "" && strings.Contains(e.Output, "(cached)") && strings.HasPrefix(e.Output, "ok") {
				s.Cached++
			}
			out[e.Package] = append(out[e.Package], strings.TrimRight(e.Output, "\n"))
			if k := len(out[e.Package]); k > 2*TailLines {
				out[e.Package] = out[e.Package][k-TailLines:]
			}
		case "pass", "fail", "skip":
			if e.Test != "" {
				s.Tests++
				if e.Action == "fail" {
					failedTests[e.Package] = append(failedTests[e.Package], e.Test)
				}
				continue
			}
			switch e.Action {
			case "pass":
				s.OK++
			case "skip":
				s.NoTests++
			default:
				s.Failed++
				failedPkgs = append(failedPkgs, e.Package)
			}
		}
	}
	if err := sc.Err(); err != nil {
		return s, fmt.Errorf("gatemark: %w", err)
	}
	sort.Strings(failedPkgs)
	for _, p := range failedPkgs {
		leaves := leafTests(failedTests[p])
		if len(leaves) == 0 {
			s.Failures = append(s.Failures, Failure{Package: p})
		}
		for _, t := range leaves {
			s.Failures = append(s.Failures, Failure{Package: p, Test: t})
		}
		s.Tail = append(s.Tail, out[p]...)
	}
	if len(s.Tail) > TailLines {
		s.Tail = s.Tail[len(s.Tail)-TailLines:]
	}
	return s, nil
}

// leafTests drops a failing parent whose failing subtest is listed: the subtest is the name to look at.
func leafTests(names []string) []string {
	sort.Strings(names)
	var out []string
	for i, n := range names {
		if i+1 < len(names) && strings.HasPrefix(names[i+1], n+"/") {
			continue
		}
		out = append(out, n)
	}
	return out
}

// Packages is the number of packages that reported a result.
func (s Summary) Packages() int { return s.OK + s.NoTests + s.Failed }

// Ran says whether the gate provably ran: at least one package reported, and every package asked for did.
func (s Summary) Ran() bool {
	if s.Packages() == 0 {
		return false
	}
	return s.Want == 0 || s.Packages() == s.Want
}

// Green is a gate that ran and failed nothing.
func (s Summary) Green() bool { return s.Ran() && s.Failed == 0 }

// Marker is the one line the merge worker requires. It starts with "gate ran:" only when Ran is true.
func (s Summary) Marker() string {
	head := "gate ran:"
	if !s.Ran() {
		head = "gate NOT RUN:"
	}
	want := "?"
	if s.Want > 0 {
		want = fmt.Sprint(s.Want)
	}
	return fmt.Sprintf("%s packages %d of %s (ok %d, no test files %d, failed %d, cached %d), tests %d",
		head, s.Packages(), want, s.OK, s.NoTests, s.Failed, s.Cached, s.Tests)
}
